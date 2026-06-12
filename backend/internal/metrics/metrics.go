package metrics

import (
	"net/http"
	"net/http/pprof"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var (
	RequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	SensorDataReceived = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "sensor_data_received_total",
			Help: "Total number of sensor data points received",
		},
	)

	SensorDataProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sensor_data_processed_total",
			Help: "Total number of sensor data points processed by module",
		},
		[]string{"module"},
	)

	AlertsTriggered = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerts_triggered_total",
			Help: "Total number of alerts triggered by type",
		},
		[]string{"alert_type"},
	)

	AlertsDelivered = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerts_delivered_total",
			Help: "Total number of alerts delivered by channel",
		},
		[]string{"channel"},
	)

	LaserCleaningPredictions = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "laser_cleaning_predictions_total",
			Help: "Total number of laser cleaning parameter predictions",
		},
	)

	ScaleGrowthPredictions = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "scale_growth_predictions_total",
			Help: "Total number of scale growth predictions",
		},
	)

	ChannelDroppedMessages = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "channel_dropped_messages_total",
			Help: "Total number of dropped messages due to full channel buffer",
		},
		[]string{"channel_name"},
	)

	DBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		},
		[]string{"query_type"},
	)

	ActiveWebSocketClients = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_websocket_clients",
			Help: "Number of active WebSocket clients",
		},
	)

	uptimeStart = time.Now()

	UptimeSeconds = promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "uptime_seconds",
			Help: "Application uptime in seconds",
		},
		func() float64 {
			return time.Since(uptimeStart).Seconds()
		},
	)
)

type MetricsServer struct {
	server     *http.Server
	pprofMux   *http.ServeMux
	httpTotal  uint64
	httpErrors uint64
}

func NewMetricsServer(addr string) *MetricsServer {
	mux := http.NewServeMux()

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.HandleFunc("/debug/pprof/heap", pprof.Handler("heap").ServeHTTP)
	mux.HandleFunc("/debug/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
	mux.HandleFunc("/debug/pprof/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
	mux.HandleFunc("/debug/pprof/block", pprof.Handler("block").ServeHTTP)
	mux.HandleFunc("/debug/pprof/mutex", pprof.Handler("mutex").ServeHTTP)
	mux.HandleFunc("/debug/pprof/allocs", pprof.Handler("allocs").ServeHTTP)

	mux.Handle("/metrics", promhttp.Handler())

	return &MetricsServer{
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
		pprofMux: mux,
	}
}

func (m *MetricsServer) Start() {
	zap.L().Info("Metrics/pprof server starting", zap.String("addr", m.server.Addr))
	go func() {
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Error("Metrics server error", zap.Error(err))
		}
	}()
}

func (m *MetricsServer) Stop() {
	if m.server != nil {
		zap.L().Info("Stopping metrics server")
		_ = m.server.Close()
	}
}

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		method := c.Request.Method

		c.Next()

		duration := time.Since(start).Seconds()
		status := c.Writer.Status()

		RequestDuration.WithLabelValues(method, path).Observe(duration)
		RequestTotal.WithLabelValues(method, path, http.StatusText(status)).Inc()
	}
}

func IncrSensorDataReceived() {
	SensorDataReceived.Inc()
	atomic.AddUint64(&globalSensorDataCount, 1)
}

func IncrSensorDataProcessed(module string) {
	SensorDataProcessed.WithLabelValues(module).Inc()
}

func IncrAlertsTriggered(alertType string) {
	AlertsTriggered.WithLabelValues(alertType).Inc()
}

func IncrAlertsDelivered(channel string) {
	AlertsDelivered.WithLabelValues(channel).Inc()
}

func IncrChannelDropped(channelName string) {
	ChannelDroppedMessages.WithLabelValues(channelName).Inc()
}

func IncrLaserCleaningPredictions() {
	LaserCleaningPredictions.Inc()
}

func IncrScaleGrowthPredictions() {
	ScaleGrowthPredictions.Inc()
}

func ObserveDBQuery(queryType string, duration time.Duration) {
	DBQueryDuration.WithLabelValues(queryType).Observe(duration.Seconds())
}

func SetActiveWebSocketClients(n int) {
	ActiveWebSocketClients.Set(float64(n))
}

func AddActiveWebSocketClients(delta int) {
	ActiveWebSocketClients.Add(float64(delta))
}

var (
	globalSensorDataCount uint64
)

func GetGlobalSensorDataCount() uint64 {
	return atomic.LoadUint64(&globalSensorDataCount)
}
