package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"stone-relic-monitor/internal/algorithms"
	"stone-relic-monitor/internal/config"
	"stone-relic-monitor/internal/db"
	"stone-relic-monitor/internal/models"
)

type SensorHandler struct {
	cfg *config.Config
	db  *db.ClickHouse
}

func NewSensorHandler(cfg *config.Config, db *db.ClickHouse) *SensorHandler {
	return &SensorHandler{cfg: cfg, db: db}
}

func (h *SensorHandler) GetLatestByRelic(c *gin.Context) {
	relicID, err := strconv.ParseUint(c.Param("relic_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid relic_id"})
		return
	}

	ctx := context.Background()
	query := `SELECT relic_id, sensor_id, latest_time, latest_value, latest_unit, latest_so2, latest_humidity, latest_temperature
		FROM v_latest_sensor_data WHERE relic_id = ?`

	rows, err := h.db.Query(ctx, query, relicID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []models.LatestSensorData
	for rows.Next() {
		var d models.LatestSensorData
		if err := rows.Scan(&d.RelicID, &d.SensorID, &d.LatestTime, &d.LatestValue,
			&d.LatestUnit, &d.LatestSO2, &d.LatestHumidity, &d.LatestTemperature); err == nil {
			results = append(results, d)
		}
	}
	c.JSON(http.StatusOK, results)
}

func (h *SensorHandler) GetHistory(c *gin.Context) {
	sensorID, err := strconv.ParseUint(c.Param("sensor_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sensor_id"})
		return
	}

	hours := 24
	if h := c.Query("hours"); h != "" {
		if v, err := strconv.Atoi(h); err == nil && v > 0 {
			hours = v
		}
	}

	ctx := context.Background()
	query := `SELECT id, sensor_id, relic_id, timestamp, value, unit, so2_concentration, humidity, temperature
		FROM sensor_data WHERE sensor_id = ? AND timestamp > now() - INTERVAL ? HOUR ORDER BY timestamp`

	rows, err := h.db.Query(ctx, query, sensorID, hours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []models.SensorData
	for rows.Next() {
		var d models.SensorData
		if err := rows.Scan(&d.ID, &d.SensorID, &d.RelicID, &d.Timestamp, &d.Value,
			&d.Unit, &d.SO2Concentration, &d.Humidity, &d.Temperature); err == nil {
			results = append(results, d)
		}
	}
	c.JSON(http.StatusOK, results)
}

func (h *SensorHandler) UploadBatch(c *gin.Context) {
	var batch models.SensorDataBatch
	if err := c.ShouldBindJSON(&batch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	tx, err := h.db.DB.BeginTx(ctx, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO sensor_data
		(id, sensor_id, relic_id, timestamp, value, unit, so2_concentration, humidity, temperature)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer stmt.Close()

	inserted := 0
	for i, data := range batch.Data {
		if data.Timestamp.IsZero() {
			data.Timestamp = time.Now()
		}
		unit := "mm"
		if data.Unit == "μm" || data.Unit == "um" {
			unit = "μm"
		}
		id := uint64(time.Now().UnixNano()/1e6) + uint64(i)
		if _, err := stmt.ExecContext(ctx, id, data.SensorID, data.RelicID, data.Timestamp,
			data.Value, unit, data.SO2Concentration, data.Humidity, data.Temperature); err != nil {
			zap.L().Warn("Insert sensor data failed", zap.Error(err), zap.Uint64("sensor_id", data.SensorID))
			continue
		}
		inserted++
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"inserted": inserted, "total": len(batch.Data)})
}

type RelicHandler struct {
	cfg *config.Config
	db  *db.ClickHouse
}

func NewRelicHandler(cfg *config.Config, db *db.ClickHouse) *RelicHandler {
	return &RelicHandler{cfg: cfg, db: db}
}

func (h *RelicHandler) List(c *gin.Context) {
	ctx := context.Background()
	query := `SELECT id, name, location, model_path, created_at FROM stone_relic ORDER BY id`
	rows, err := h.db.Query(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []models.StoneRelic
	for rows.Next() {
		var r models.StoneRelic
		if err := rows.Scan(&r.ID, &r.Name, &r.Location, &r.ModelPath, &r.CreatedAt); err == nil {
			results = append(results, r)
		}
	}

	if len(results) == 0 {
		results = make([]models.StoneRelic, 0)
	}

	c.JSON(http.StatusOK, results)
}

func (h *RelicHandler) Get(c *gin.Context) {
	relicID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	ctx := context.Background()
	detail := &models.RelicDetail{}

	query := `SELECT id, name, location, model_path, created_at FROM stone_relic WHERE id = ?`
	row := h.db.QueryRow(ctx, query, relicID)
	if err := row.Scan(&detail.ID, &detail.Name, &detail.Location, &detail.ModelPath, &detail.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "relic not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	sensorQuery := `SELECT id, relic_id, type, model, position_x, position_y, created_at FROM sensor WHERE relic_id = ?`
	rows, err := h.db.Query(ctx, sensorQuery, relicID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var sensor models.Sensor
			if err := rows.Scan(&sensor.ID, &sensor.RelicID, &sensor.Type, &sensor.Model,
				&sensor.PositionX, &sensor.PositionY, &sensor.CreatedAt); err == nil {
				detail.Sensors = append(detail.Sensors, sensor)
			}
		}
	}

	latestQuery := `SELECT relic_id, sensor_id, latest_time, latest_value, latest_unit, latest_so2, latest_humidity, latest_temperature
		FROM v_latest_sensor_data WHERE relic_id = ?`
	rows2, err := h.db.Query(ctx, latestQuery, relicID)
	if err == nil {
		defer rows2.Close()
		roughnessSum := float32(0)
		roughnessCount := 0
		for rows2.Next() {
			var ld models.LatestSensorData
			if err := rows2.Scan(&ld.RelicID, &ld.SensorID, &ld.LatestTime, &ld.LatestValue,
				&ld.LatestUnit, &ld.LatestSO2, &ld.LatestHumidity, &ld.LatestTemperature); err == nil {
				detail.LatestData = append(detail.LatestData, ld)
				if ld.LatestUnit == "mm" && ld.LatestValue > detail.MaxThickness {
					detail.MaxThickness = ld.LatestValue
				}
				if ld.LatestUnit == "μm" {
					roughnessSum += ld.LatestValue
					roughnessCount++
				}
			}
		}
		if roughnessCount > 0 {
			detail.AvgRoughness = roughnessSum / float32(roughnessCount)
		}
	}

	alertQuery := `SELECT count() FROM alert_record WHERE relic_id = ? AND created_at > now() - INTERVAL 7 DAY`
	var ac uint64
	h.db.QueryRow(ctx, alertQuery, relicID).Scan(&ac)
	detail.AlertCount = int(ac)

	c.JSON(http.StatusOK, detail)
}

func (h *RelicHandler) GetDailyStats(c *gin.Context) {
	relicID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	days := 7
	if d := c.Query("days"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 {
			days = v
		}
	}

	ctx := context.Background()
	query := `SELECT relic_id, date, avg_thickness, max_thickness, avg_roughness, max_roughness,
		avg_so2, avg_humidity, avg_temperature, data_count
		FROM v_daily_statistics WHERE relic_id = ? AND date >= today() - INTERVAL ? DAY ORDER BY date`

	rows, err := h.db.Query(ctx, query, relicID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []models.DailyStatistics
	for rows.Next() {
		var d models.DailyStatistics
		if err := rows.Scan(&d.RelicID, &d.Date, &d.AvgThickness, &d.MaxThickness,
			&d.AvgRoughness, &d.MaxRoughness, &d.AvgSO2, &d.AvgHumidity, &d.AvgTemperature, &d.DataCount); err == nil {
			results = append(results, d)
		}
	}
	c.JSON(http.StatusOK, results)
}

type AlgorithmHandler struct {
	cfg *config.Config
	db  *db.ClickHouse
}

func NewAlgorithmHandler(cfg *config.Config, db *db.ClickHouse) *AlgorithmHandler {
	return &AlgorithmHandler{cfg: cfg, db: db}
}

func (h *AlgorithmHandler) PredictScaleGrowth(c *gin.Context) {
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

func (h *AlgorithmHandler) PredictLaserCleaning(c *gin.Context) {
	var req models.LaserCleaningRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.TargetThickness <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_thickness must be positive"})
		return
	}
	if req.MaterialType == "" {
		req.MaterialType = "calcium_sulfate"
	}

	result := algorithms.PredictLaserCleaning(&req)

	ctx := context.Background()
	id := uint64(time.Now().UnixNano() / 1e6)
	logQuery := `INSERT INTO cleaning_parameter_opt_log
		(id, relic_id, area_id, target_thickness, material_type, optimal_power, optimal_pulse, optimal_speed,
		 predicted_energy_density, ablation_threshold, confidence, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	h.db.Exec(ctx, logQuery, id, req.RelicID, req.AreaID, req.TargetThickness, req.MaterialType,
		result.OptimalPower, result.OptimalPulse, result.OptimalSpeed,
		result.PredictedEnergyDensity, result.AblationThreshold, result.Confidence, time.Now())

	c.JSON(http.StatusOK, result)
}
