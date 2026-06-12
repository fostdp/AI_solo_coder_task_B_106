package scaling_model

import (
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"stone-relic-monitor/internal/algorithms"
	"stone-relic-monitor/internal/config"
	"stone-relic-monitor/internal/models"
	"stone-relic-monitor/internal/scaling"
)

type PredictionRequest struct {
	*models.ScaleGrowthPrediction
}

type PredictionResult struct {
	*models.ScaleGrowthPrediction
}

type ScalingModelService struct {
	cfg                *config.Config
	SensorDataInChan   <-chan *models.SensorData
	PredictionOutChan  chan<- *PredictionResult
	cfdField           *scaling.CFDWindField
	mu                 sync.Mutex
	latestPredictions  map[uint64]*PredictionResult
}

func NewScalingModelService(
	cfg *config.Config,
	dataInChan <-chan *models.SensorData,
	predOutChan chan<- *PredictionResult,
) *ScalingModelService {
	svc := &ScalingModelService{
		cfg:               cfg,
		SensorDataInChan:  dataInChan,
		PredictionOutChan: predOutChan,
		latestPredictions: make(map[uint64]*PredictionResult),
		cfdField:          scaling.GenerateSyntheticBuddhaCFDField(),
	}
	return svc
}

func (s *ScalingModelService) Run(stopChan <-chan struct{}) {
	zap.L().Info("ScalingModelService started")

	for {
		select {
		case <-stopChan:
			zap.L().Info("ScalingModelService stopped")
			return
		case data, ok := <-s.SensorDataInChan:
			if !ok {
				return
			}
			s.processRealtimeData(data)
		}
	}
}

func (s *ScalingModelService) processRealtimeData(data *models.SensorData) {
	if data.Unit != "mm" {
		return
	}

	pred := &models.ScaleGrowthPrediction{
		Hours:            168,
		InitialThickness: data.Value,
		SO2Concentration: data.SO2Concentration,
		Humidity:         data.Humidity,
		Temperature:      data.Temperature,
	}

	algorithms.PredictScaleGrowth(pred)

	result := &PredictionResult{ScaleGrowthPrediction: pred}

	s.mu.Lock()
	s.latestPredictions[data.SensorID] = result
	s.mu.Unlock()

	select {
	case s.PredictionOutChan <- result:
	default:
		zap.L().Warn("PredictionOutChan full, dropping prediction",
			zap.Uint64("sensor_id", data.SensorID))
	}
}

func (s *ScalingModelService) PredictScaleGrowth(c *gin.Context) {
	var req models.ScaleGrowthPrediction
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Hours <= 0 {
		req.Hours = 168
	}
	if req.Hours > 24*30*12 {
		req.Hours = 24 * 30 * 12
	}

	algorithms.PredictScaleGrowth(&req)
	c.JSON(http.StatusOK, req)
}

func (s *ScalingModelService) PredictWithOrientation(c *gin.Context) {
	var req struct {
		Hours             int                             `json:"hours"`
		InitialThickness  float32                         `json:"initial_thickness"`
		SO2Concentration  float32                         `json:"so2_concentration"`
		Humidity          float32                         `json:"humidity"`
		Temperature       float32                         `json:"temperature"`
		PointX            float64                         `json:"point_x"`
		PointY            float64                         `json:"point_y"`
		PointZ            float64                         `json:"point_z"`
		Surface           scaling.SurfaceOrientation      `json:"surface"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Hours <= 0 {
		req.Hours = 168
	}

	model := scaling.NewScaleGrowthWithOrientation(s.cfdField)
	pred := model.PredictPointGrowth(
		req.PointX, req.PointY, req.PointZ,
		float64(req.InitialThickness),
		float64(req.SO2Concentration),
		float64(req.Humidity),
		float64(req.Temperature),
		req.Hours,
		req.Surface,
	)

	c.JSON(http.StatusOK, pred)
}

func (s *ScalingModelService) GetLatestPrediction(c *gin.Context) {
	sensorID, err := strconv.ParseUint(c.Param("sensor_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sensor_id"})
		return
	}

	s.mu.Lock()
	pred, exists := s.latestPredictions[sensorID]
	s.mu.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "no prediction available"})
		return
	}

	c.JSON(http.StatusOK, pred)
}

func (s *ScalingModelService) RegisterRoutes(r *gin.RouterGroup) {
	alg := r.Group("/algorithms")
	{
		alg.POST("/predict-scale-growth", s.PredictScaleGrowth)
		alg.POST("/predict-scale-orientation", s.PredictWithOrientation)
		alg.GET("/scale-prediction/:sensor_id", s.GetLatestPrediction)
	}
}
