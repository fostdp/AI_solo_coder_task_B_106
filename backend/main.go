package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"stone-relic-monitor/internal/alert_ws"
	"stone-relic-monitor/internal/algorithms"
	"stone-relic-monitor/internal/config"
	"stone-relic-monitor/internal/db"
	"stone-relic-monitor/internal/ethercat_ingest"
	"stone-relic-monitor/internal/handlers"
	"stone-relic-monitor/internal/laser_threshold"
	"stone-relic-monitor/internal/metrics"
	"stone-relic-monitor/internal/models"
	"stone-relic-monitor/internal/scaling_model"
	"stone-relic-monitor/internal/services"
)

var Version = "2.0.0"

const (
	chanBufferSize = 1000
)

type App struct {
	cfg                *config.Config
	db                 *db.ClickHouse
	stopChan           chan struct{}

	sensorDataChan     chan *models.SensorData
	predictionChan     chan *scaling_model.PredictionResult
	cleaningResultChan chan *laser_threshold.CleaningResult

	ingestSvc          *ethercat_ingest.IngestService
	scalingSvc         *scaling_model.ScalingModelService
	laserSvc           *laser_threshold.LaserThresholdService
	alertSvc           *alert_ws.AlertService
	commonHdlr         *handlers.CommonHandler
	advancedHdlr       *handlers.AdvancedCleaningHandler
	monitorSvc         *services.MonitorService
	metricsSvc         *metrics.MetricsServer
}

func NewApp() *App {
	cfg := config.Load()

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

	algorithms.SetLaserConfig(cfg.Laser)

	chDB := db.NewClickHouse(cfg)
	if err := chDB.Connect(); err != nil {
		zap.L().Fatal("Failed to connect to ClickHouse", zap.Error(err))
	}

	stopChan := make(chan struct{})

	sensorDataChan := make(chan *models.SensorData, chanBufferSize)
	predictionChan := make(chan *scaling_model.PredictionResult, chanBufferSize)
	cleaningResultChan := make(chan *laser_threshold.CleaningResult, chanBufferSize)

	ingestSvc := ethercat_ingest.NewIngestService(cfg, chDB, sensorDataChan)
	scalingSvc := scaling_model.NewScalingModelService(cfg, sensorDataChan, predictionChan)
	laserSvc := laser_threshold.NewLaserThresholdService(cfg, chDB, predictionChan, cleaningResultChan)
	alertSvc := alert_ws.NewAlertService(cfg, chDB, sensorDataChan)
	commonHdlr := handlers.NewCommonHandler(cfg, chDB)
	advancedHdlr := handlers.NewAdvancedCleaningHandler(cfg, chDB)
	monitorSvc := services.NewMonitorService(cfg, chDB)
	metricsSvc := metrics.NewMetricsServer(":6060")

	return &App{
		cfg:                cfg,
		db:                 chDB,
		stopChan:           stopChan,
		sensorDataChan:     sensorDataChan,
		predictionChan:     predictionChan,
		cleaningResultChan: cleaningResultChan,
		ingestSvc:          ingestSvc,
		scalingSvc:         scalingSvc,
		laserSvc:           laserSvc,
		alertSvc:           alertSvc,
		commonHdlr:         commonHdlr,
		advancedHdlr:       advancedHdlr,
		monitorSvc:         monitorSvc,
		metricsSvc:         metricsSvc,
	}
}

func (a *App) setupRouter() *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.Use(metrics.PrometheusMiddleware())

	r.GET("/health", a.commonHdlr.Health)

	api := r.Group("/api/v1")
	{
		a.commonHdlr.RegisterRoutes(api)
		a.ingestSvc.RegisterRoutes(api)
		a.scalingSvc.RegisterRoutes(api)
		a.laserSvc.RegisterRoutes(api)
		a.alertSvc.RegisterRoutes(api)
		a.advancedHdlr.RegisterRoutes(api)
	}

	return r
}

func (a *App) Start() {
	zap.L().Info("Starting Stone Relic Monitor",
		zap.String("version", Version),
		zap.Strings("modules", []string{"ethercat_ingest", "scaling_model", "laser_threshold", "alert_ws", "metrics", "advanced_cleaning"}))

	go a.scalingSvc.Run(a.stopChan)
	go a.laserSvc.Run(a.stopChan)
	go a.alertSvc.Run(a.stopChan)
	go a.monitorSvc.Start()
	a.metricsSvc.Start()

	r := a.setupRouter()

	addr := fmt.Sprintf(":%d", a.cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		zap.L().Info("HTTP server starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Fatal("Failed to start server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zap.L().Info("Shutdown signal received")
	close(a.stopChan)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		zap.L().Error("Server forced shutdown", zap.Error(err))
	}

	a.metricsSvc.Stop()
	a.monitorSvc.Stop()
	if err := a.db.Close(); err != nil {
		zap.L().Error("DB close error", zap.Error(err))
	}

	zap.L().Info("Server exited properly")
}

func main() {
	app := NewApp()
	app.Start()
}
