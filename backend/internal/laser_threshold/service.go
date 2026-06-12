package laser_threshold

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"stone-relic-monitor/internal/algorithms"
	"stone-relic-monitor/internal/config"
	"stone-relic-monitor/internal/db"
	"stone-relic-monitor/internal/models"
	"stone-relic-monitor/internal/scaling_model"
)

type CleaningRequest struct {
	*models.LaserCleaningRequest
}

type CleaningResult struct {
	*models.LaserCleaningResult
}

type LaserThresholdService struct {
	cfg                *config.Config
	db                 *db.ClickHouse
	PredictionInChan   <-chan *scaling_model.PredictionResult
	CleaningOutChan    chan<- *CleaningResult
	mu                 sync.Mutex
	latestResults      map[uint64]*CleaningResult
}

func NewLaserThresholdService(
	cfg *config.Config,
	db *db.ClickHouse,
	predInChan <-chan *scaling_model.PredictionResult,
	cleaningOutChan chan<- *CleaningResult,
) *LaserThresholdService {
	svc := &LaserThresholdService{
		cfg:              cfg,
		db:               db,
		PredictionInChan: predInChan,
		CleaningOutChan:  cleaningOutChan,
		latestResults:    make(map[uint64]*CleaningResult),
	}
	return svc
}

func (s *LaserThresholdService) Run(stopChan <-chan struct{}) {
	zap.L().Info("LaserThresholdService started")

	for {
		select {
		case <-stopChan:
			zap.L().Info("LaserThresholdService stopped")
			return
		case pred, ok := <-s.PredictionInChan:
			if !ok {
				return
			}
			s.processPrediction(pred)
		}
	}
}

func (s *LaserThresholdService) processPrediction(pred *scaling_model.PredictionResult) {
	req := &models.LaserCleaningRequest{
		MaterialType:    "calcium_sulfate",
		TargetThickness: pred.FinalThickness,
		SurfaceRoughness: 15.0,
	}

	result := algorithms.PredictLaserCleaning(req)

	cleaning := &CleaningResult{LaserCleaningResult: result}
	cleaning.RelicID = 1

	s.mu.Lock()
	s.latestResults[1] = cleaning
	s.mu.Unlock()

	select {
	case s.CleaningOutChan <- cleaning:
	default:
		zap.L().Warn("CleaningOutChan full, dropping result")
	}
}

func (s *LaserThresholdService) PredictLaserCleaning(c *gin.Context) {
	var req models.LaserCleaningRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.TargetThickness <= 0 {
		req.TargetThickness = 1.0
	}
	if req.MaterialType == "" {
		req.MaterialType = "calcium_sulfate"
	}

	result := algorithms.PredictLaserCleaning(&req)
	c.JSON(http.StatusOK, result)
}

func (s *LaserThresholdService) CreateCleaningRecord(c *gin.Context) {
	var record models.CleaningRecord
	if err := c.ShouldBindJSON(&record); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}

	actualDepth := algorithms.CalculateAblationDepth(
		record.LaserPower,
		record.PulseDuration,
		record.EnergyDensity,
	)
	record.ActualDepth = actualDepth
	record.Effectiveness = float32(float64(record.TargetDepth) /
		float64(record.TargetDepth+record.ActualDepth) * 2)
	if record.Effectiveness > 1.0 {
		record.Effectiveness = 1.0
	}

	query := `INSERT INTO cleaning_record
		(id, relic_id, timestamp, laser_power, pulse_duration, scan_speed,
		 target_depth, actual_depth, energy_density, effectiveness, operator_notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	id := uint64(time.Now().UnixNano() / 1e6)
	_, err := s.db.Exec(context.Background(), query,
		id, record.RelicID, record.Timestamp,
		record.LaserPower, record.PulseDuration, record.ScanSpeed,
		record.TargetDepth, record.ActualDepth, record.EnergyDensity,
		record.Effectiveness, record.OperatorNotes,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	optLog := &models.CleaningParameterOptLog{
		ID:              id + 1,
		RelicID:         record.RelicID,
		Timestamp:       record.Timestamp,
		RequestedPower:  record.LaserPower,
		RequestedPulse:  record.PulseDuration,
		RequestedSpeed:  record.ScanSpeed,
		OptimalPower:    record.LaserPower,
		OptimalPulse:    record.PulseDuration,
		OptimalSpeed:    record.ScanSpeed,
		TargetDepth:     record.TargetDepth,
		PredictedDepth:  record.ActualDepth,
		OptimizationGain: record.Effectiveness,
	}

	optQuery := `INSERT INTO cleaning_parameter_opt_log
		(id, relic_id, timestamp, requested_power, requested_pulse, requested_speed,
		 optimal_power, optimal_pulse, optimal_speed, target_depth, predicted_depth, optimization_gain)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, _ = s.db.Exec(context.Background(), optQuery,
		optLog.ID, optLog.RelicID, optLog.Timestamp,
		optLog.RequestedPower, optLog.RequestedPulse, optLog.RequestedSpeed,
		optLog.OptimalPower, optLog.OptimalPulse, optLog.OptimalSpeed,
		optLog.TargetDepth, optLog.PredictedDepth, optLog.OptimizationGain,
	)

	record.ID = id
	c.JSON(http.StatusOK, record)
}

func (s *LaserThresholdService) GetCleaningRecords(c *gin.Context) {
	relicID, err := strconv.ParseUint(c.Param("relic_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid relic_id"})
		return
	}

	days := 30
	if d := c.Query("days"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 {
			days = v
		}
	}

	ctx := context.Background()
	query := `SELECT id, relic_id, timestamp, laser_power, pulse_duration, scan_speed,
		target_depth, actual_depth, energy_density, effectiveness, operator_notes
		FROM cleaning_record WHERE relic_id = ? AND timestamp > now() - INTERVAL ? DAY
		ORDER BY timestamp DESC LIMIT 1000`

	rows, err := s.db.Query(ctx, query, relicID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var records []models.CleaningRecord
	for rows.Next() {
		var r models.CleaningRecord
		if err := rows.Scan(&r.ID, &r.RelicID, &r.Timestamp,
			&r.LaserPower, &r.PulseDuration, &r.ScanSpeed,
			&r.TargetDepth, &r.ActualDepth, &r.EnergyDensity,
			&r.Effectiveness, &r.OperatorNotes); err == nil {
			records = append(records, r)
		}
	}

	c.JSON(http.StatusOK, gin.H{"count": len(records), "data": records})
}

func (s *LaserThresholdService) GetCleaningStats(c *gin.Context) {
	ctx := context.Background()
	query := `SELECT count(), avg(effectiveness), sum(actual_depth)
		FROM cleaning_record WHERE timestamp > now() - INTERVAL 30 DAY`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	if rows.Next() {
		var count int64
		var avgEff, totalDepth float64
		if err := rows.Scan(&count, &avgEff, &totalDepth); err == nil {
			c.JSON(http.StatusOK, gin.H{
				"total_cleanings": count,
				"avg_effectiveness": avgEff,
				"total_depth_removed_um": totalDepth,
			})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"total_cleanings": 0,
		"avg_effectiveness": 0,
		"total_depth_removed_um": 0,
	})
}

func (s *LaserThresholdService) RegisterRoutes(r *gin.RouterGroup) {
	alg := r.Group("/algorithms")
	{
		alg.POST("/predict-laser-cleaning", s.PredictLaserCleaning)
	}

	cleaning := r.Group("/cleaning")
	{
		cleaning.POST("/records", s.CreateCleaningRecord)
		cleaning.GET("/records/relic/:relic_id", s.GetCleaningRecords)
		cleaning.GET("/stats", s.GetCleaningStats)
	}
}
