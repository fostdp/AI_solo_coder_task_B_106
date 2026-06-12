package ethercat_ingest

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"stone-relic-monitor/internal/config"
	"stone-relic-monitor/internal/db"
	"stone-relic-monitor/internal/models"
)

type IngestService struct {
	cfg           *config.Config
	db            *db.ClickHouse
	SensorDataChan chan<- *models.SensorData
}

func NewIngestService(cfg *config.Config, db *db.ClickHouse, dataChan chan<- *models.SensorData) *IngestService {
	return &IngestService{
		cfg:           cfg,
		db:            db,
		SensorDataChan: dataChan,
	}
}

func (s *IngestService) GetLatestByRelic(c *gin.Context) {
	relicID, err := strconv.ParseUint(c.Param("relic_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid relic_id"})
		return
	}

	ctx := context.Background()
	query := `SELECT relic_id, sensor_id, latest_time, latest_value, latest_unit, latest_so2, latest_humidity, latest_temperature
		FROM v_latest_sensor_data WHERE relic_id = ?`

	rows, err := s.db.Query(ctx, query, relicID)
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

func (s *IngestService) GetHistory(c *gin.Context) {
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

	rows, err := s.db.Query(ctx, query, sensorID, hours)
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

func (s *IngestService) UploadBatch(c *gin.Context) {
	var batch models.SensorDataBatch
	if err := c.ShouldBindJSON(&batch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()

	var rows [][]interface{}
	for i, data := range batch.Data {
		if data.Timestamp.IsZero() {
			data.Timestamp = time.Now()
		}
		unit := "mm"
		if data.Unit == "μm" || data.Unit == "um" {
			unit = "μm"
		}
		id := uint64(time.Now().UnixNano()/1e6) + uint64(i)
		rows = append(rows, []interface{}{
			id, data.SensorID, data.RelicID, data.Timestamp, data.Value, unit,
			data.SO2Concentration, data.Humidity, data.Temperature,
		})

		sd := &models.SensorData{
			ID:               id,
			SensorID:         data.SensorID,
			RelicID:          data.RelicID,
			Timestamp:        data.Timestamp,
			Value:            data.Value,
			Unit:             unit,
			SO2Concentration: data.SO2Concentration,
			Humidity:         data.Humidity,
			Temperature:      data.Temperature,
		}

		select {
		case s.SensorDataChan <- sd:
		default:
			zap.L().Warn("SensorDataChan full, dropping broadcast", zap.Uint64("sensor_id", data.SensorID))
		}
	}

	query := `INSERT INTO sensor_data
		(id, sensor_id, relic_id, timestamp, value, unit, so2_concentration, humidity, temperature)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	inserted, err := s.db.BatchInsertSync(ctx, query, rows)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"inserted": inserted, "total": len(batch.Data)})
}

func (s *IngestService) RegisterRoutes(r *gin.RouterGroup) {
	sensors := r.Group("/sensors")
	{
		sensors.GET("/relic/:relic_id/latest", s.GetLatestByRelic)
		sensors.GET("/:sensor_id/history", s.GetHistory)
		sensors.POST("/upload", s.UploadBatch)
	}
}
