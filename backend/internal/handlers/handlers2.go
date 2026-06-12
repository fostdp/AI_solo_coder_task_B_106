package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"stone-relic-monitor/internal/alert"
	"stone-relic-monitor/internal/config"
	"stone-relic-monitor/internal/db"
	"stone-relic-monitor/internal/models"
)

type AlertHandler struct {
	cfg   *config.Config
	db    *db.ClickHouse
	wsHub *alert.Hub
}

func NewAlertHandler(cfg *config.Config, db *db.ClickHouse, wsHub *alert.Hub) *AlertHandler {
	return &AlertHandler{cfg: cfg, db: db, wsHub: wsHub}
}

func (h *AlertHandler) List(c *gin.Context) {
	limit := 100
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 1000 {
			limit = v
		}
	}

	days := 7
	if d := c.Query("days"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 {
			days = v
		}
	}

	ctx := context.Background()
	query := `SELECT id, relic_id, sensor_id, type, level, value, threshold, message, created_at
		FROM alert_record WHERE created_at > now() - INTERVAL ? DAY ORDER BY created_at DESC LIMIT ?`

	rows, err := h.db.Query(ctx, query, days, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []models.AlertRecord
	for rows.Next() {
		var a models.AlertRecord
		var at, lv string
		if err := rows.Scan(&a.ID, &a.RelicID, &a.SensorID, &at, &lv,
			&a.Value, &a.Threshold, &a.Message, &a.CreatedAt); err == nil {
			a.Type = at
			a.Level = lv
			results = append(results, a)
		}
	}
	if len(results) == 0 {
		results = make([]models.AlertRecord, 0)
	}
	c.JSON(http.StatusOK, results)
}

func (h *AlertHandler) GetByRelic(c *gin.Context) {
	relicID, err := strconv.ParseUint(c.Param("relic_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid relic_id"})
		return
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}

	ctx := context.Background()
	query := `SELECT id, relic_id, sensor_id, type, level, value, threshold, message, created_at
		FROM alert_record WHERE relic_id = ? ORDER BY created_at DESC LIMIT ?`

	rows, err := h.db.Query(ctx, query, relicID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []models.AlertRecord
	for rows.Next() {
		var a models.AlertRecord
		var at, lv string
		if err := rows.Scan(&a.ID, &a.RelicID, &a.SensorID, &at, &lv,
			&a.Value, &a.Threshold, &a.Message, &a.CreatedAt); err == nil {
			a.Type = at
			a.Level = lv
			results = append(results, a)
		}
	}
	c.JSON(http.StatusOK, results)
}

func (h *AlertHandler) GetStats(c *gin.Context) {
	days := 30
	if d := c.Query("days"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 {
			days = v
		}
	}

	ctx := context.Background()
	query := `SELECT
		count() as total,
		countIf(level = 'critical') as critical,
		countIf(level = 'warning') as warning,
		countIf(type = 'thickness') as thickness_alerts,
		countIf(type = 'roughness') as roughness_alerts
		FROM alert_record WHERE created_at > now() - INTERVAL ? DAY`

	var total, critical, warning, thickness, roughness uint64
	row := h.db.QueryRow(ctx, query, days)
	row.Scan(&total, &critical, &warning, &thickness, &roughness)

	c.JSON(http.StatusOK, gin.H{
		"total":            total,
		"critical":         critical,
		"warning":          warning,
		"thickness_alerts": thickness,
		"roughness_alerts": roughness,
		"days":             days,
	})
}

type CleaningHandler struct {
	cfg *config.Config
	db  *db.ClickHouse
}

func NewCleaningHandler(cfg *config.Config, db *db.ClickHouse) *CleaningHandler {
	return &CleaningHandler{cfg: cfg, db: db}
}

func (h *CleaningHandler) CreateRecord(c *gin.Context) {
	var record models.CleaningRecord
	if err := c.ShouldBindJSON(&record); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	record.ID = uint64(time.Now().UnixNano() / 1e6)
	record.CreatedAt = time.Now()

	query := `INSERT INTO cleaning_record
		(id, relic_id, area_id, laser_power, pulse_duration, scan_speed, predicted_depth, actual_depth, operator, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	if err := h.db.Exec(ctx, query, record.ID, record.RelicID, record.AreaID, record.LaserPower,
		record.PulseDuration, record.ScanSpeed, record.PredictedDepth, record.ActualDepth,
		record.Operator, record.CreatedAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, record)
}

func (h *CleaningHandler) List(c *gin.Context) {
	limit := 100
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 1000 {
			limit = v
		}
	}

	relicID := uint64(0)
	if rid := c.Query("relic_id"); rid != "" {
		if v, err := strconv.ParseUint(rid, 10, 64); err == nil {
			relicID = v
		}
	}

	ctx := context.Background()
	var (
		rows *sql.Rows
		err  error
	)
	if relicID > 0 {
		query := `SELECT id, relic_id, area_id, laser_power, pulse_duration, scan_speed, predicted_depth, actual_depth, operator, created_at
			FROM cleaning_record WHERE relic_id = ? ORDER BY created_at DESC LIMIT ?`
		rows, err = h.db.Query(ctx, query, relicID, limit)
	} else {
		query := `SELECT id, relic_id, area_id, laser_power, pulse_duration, scan_speed, predicted_depth, actual_depth, operator, created_at
			FROM cleaning_record ORDER BY created_at DESC LIMIT ?`
		rows, err = h.db.Query(ctx, query, limit)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []models.CleaningRecord
	for rows.Next() {
		var r models.CleaningRecord
		if err := rows.Scan(&r.ID, &r.RelicID, &r.AreaID, &r.LaserPower, &r.PulseDuration,
			&r.ScanSpeed, &r.PredictedDepth, &r.ActualDepth, &r.Operator, &r.CreatedAt); err == nil {
			results = append(results, r)
		}
	}
	c.JSON(http.StatusOK, results)
}

func (h *CleaningHandler) GetOptLog(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}

	ctx := context.Background()
	query := `SELECT id, relic_id, area_id, target_thickness, material_type, optimal_power, optimal_pulse,
		optimal_speed, predicted_energy_density, ablation_threshold, confidence, created_at
		FROM cleaning_parameter_opt_log ORDER BY created_at DESC LIMIT ?`

	rows, err := h.db.Query(ctx, query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []models.CleaningParameterOpt
	for rows.Next() {
		var r models.CleaningParameterOpt
		if err := rows.Scan(&r.ID, &r.RelicID, &r.AreaID, &r.TargetThickness, &r.MaterialType,
			&r.OptimalPower, &r.OptimalPulse, &r.OptimalSpeed, &r.PredictedEnergyDensity,
			&r.AblationThreshold, &r.Confidence, &r.CreatedAt); err == nil {
			results = append(results, r)
		}
	}
	c.JSON(http.StatusOK, results)
}
