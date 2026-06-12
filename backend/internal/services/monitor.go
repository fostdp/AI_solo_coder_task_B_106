package services

import (
	"context"
	"database/sql"
	"go.uber.org/zap"
	"stone-relic-monitor/internal/config"
	"stone-relic-monitor/internal/db"
	"stone-relic-monitor/internal/models"
	"sync"
	"time"
)

type _context_key string

type MonitorService struct {
	cfg          *config.Config
	db           *db.ClickHouse
	stopChan     chan struct{}
	mu           sync.Mutex
}

func NewMonitorService(cfg *config.Config, db *db.ClickHouse) *MonitorService {
	return &MonitorService{
		cfg:          cfg,
		db:           db,
		stopChan:     make(chan struct{}),
	}
}

func (s *MonitorService) Start() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	zap.L().Info("Monitor service started")

	for {
		select {
		case <-ticker.C:
			s.checkLatestData()
		case <-s.stopChan:
			return
		}
	}
}

func (s *MonitorService) Stop() {
	close(s.stopChan)
}

func (s *MonitorService) checkLatestData() {
	ctx := context.Background()

	query := `SELECT s.relic_id, s.id, s.type, vl.latest_value, vl.latest_time
		FROM sensor s
		LEFT JOIN v_latest_sensor_data vl ON s.relic_id = vl.relic_id AND s.id = vl.sensor_id
		WHERE vl.latest_time > now() - INTERVAL 3 HOUR`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		zap.L().Error("Failed to query latest sensor data", zap.Error(err))
		return
	}
	defer rows.Close()

	for rows.Next() {
		var (
			relicID     uint64
			sensorID    uint64
			sensorType  string
			latestValue float32
			latestTime  time.Time
		)
		if err := rows.Scan(&relicID, &sensorID, &sensorType, &latestValue, &latestTime); err != nil {
			continue
		}

		unit := "mm"
		if sensorType == "roughness" {
			unit = "μm"
		}

		_ = &models.SensorData{
			SensorID:  sensorID,
			RelicID:   relicID,
			Value:     latestValue,
			Timestamp: latestTime,
			Unit:      unit,
		}
	}
}

func (s *MonitorService) ProcessIncomingData(data *models.SensorData, sensorType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()

	var unit string
	if sensorType == "ultrasonic" {
		unit = "mm"
	} else {
		unit = "μm"
	}

	query := `INSERT INTO sensor_data (id, sensor_id, relic_id, timestamp, value, unit, so2_concentration, humidity, temperature)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	id := uint64(time.Now().UnixNano() / 1e6)
	if err := s.db.Exec(ctx, query, id, data.SensorID, data.RelicID, data.Timestamp, data.Value,
		unit, data.SO2Concentration, data.Humidity, data.Temperature); err != nil {
		zap.L().Error("Failed to insert sensor data", zap.Error(err), zap.Uint64("sensor_id", data.SensorID))
		return err
	}

	data.ID = id
	data.Unit = unit

	return nil
}

func (s *MonitorService) GetRelicDetail(relicID uint64) (*models.RelicDetail, error) {
	ctx := context.Background()
	detail := &models.RelicDetail{}

	relicQuery := `SELECT id, name, location, model_path, created_at FROM stone_relic WHERE id = ?`
	row := s.db.QueryRow(ctx, relicQuery, relicID)
	if err := row.Scan(&detail.ID, &detail.Name, &detail.Location, &detail.ModelPath, &detail.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	sensorQuery := `SELECT id, relic_id, type, model, position_x, position_y, created_at FROM sensor WHERE relic_id = ?`
	rows, err := s.db.Query(ctx, sensorQuery, relicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var sensor models.Sensor
		if err := rows.Scan(&sensor.ID, &sensor.RelicID, &sensor.Type, &sensor.Model,
			&sensor.PositionX, &sensor.PositionY, &sensor.CreatedAt); err == nil {
			detail.Sensors = append(detail.Sensors, sensor)
		}
	}

	latestQuery := `SELECT relic_id, sensor_id, latest_time, latest_value, latest_unit, latest_so2, latest_humidity, latest_temperature
		FROM v_latest_sensor_data WHERE relic_id = ?`
	rows2, err := s.db.Query(ctx, latestQuery, relicID)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var ld models.LatestSensorData
			if err := rows2.Scan(&ld.RelicID, &ld.SensorID, &ld.LatestTime, &ld.LatestValue,
				&ld.LatestUnit, &ld.LatestSO2, &ld.LatestHumidity, &ld.LatestTemperature); err == nil {
				detail.LatestData = append(detail.LatestData, ld)
				if ld.LatestValue > detail.MaxThickness && ld.LatestUnit == "mm" {
					detail.MaxThickness = ld.LatestValue
				}
				if ld.LatestUnit == "μm" {
					detail.AvgRoughness += ld.LatestValue
				}
			}
		}
	}

	roughnessCount := 0
	for _, ld := range detail.LatestData {
		if ld.LatestUnit == "μm" {
			roughnessCount++
		}
	}
	if roughnessCount > 0 {
		detail.AvgRoughness /= float32(roughnessCount)
	}

	alertQuery := `SELECT count() FROM alert_record WHERE relic_id = ? AND created_at > now() - INTERVAL 7 DAY`
	var alertCount uint64
	row = s.db.QueryRow(ctx, alertQuery, relicID)
	row.Scan(&alertCount)
	detail.AlertCount = int(alertCount)

	return detail, nil
}
