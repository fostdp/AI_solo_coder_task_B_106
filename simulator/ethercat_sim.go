package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type SensorConfig struct {
	ID           uint64
	RelicID      uint64
	Type         string
	BaseValue    float64
	DriftRate    float64
	CurrentValue float64
	LastUpdate   time.Time
}

type SensorData struct {
	ID               uint64    `json:"id"`
	SensorID         uint64    `json:"sensor_id"`
	RelicID          uint64    `json:"relic_id"`
	Timestamp        time.Time `json:"timestamp"`
	Value            float32   `json:"value"`
	Unit             string    `json:"unit"`
	SO2Concentration float32   `json:"so2_concentration"`
	Humidity         float32   `json:"humidity"`
	Temperature      float32   `json:"temperature"`
}

type HighScaleEvent struct {
	StartHour  int
	EndHour    int
	Multiplier float64
	SO2Boost   float32
	HumidityBoost float32
}

type EtherCATSimulator struct {
	apiEndpoint     string
	interval        time.Duration
	sensors         []*SensorConfig
	stopChan        chan struct{}
	mu              sync.Mutex
	totalSent       uint64
	alertCounter    uint64
	historicStart   time.Time
	rng             *rand.Rand
	httpClient      *http.Client
	sendSem         chan struct{}
	deviceCount     int
	highScaleMode   bool
	highScaleEvents []HighScaleEvent
	backfillDays    int
}

var defaultRelicLayout = []struct {
	RelicID    uint64
	Ultrasonic int
	Roughness  int
}{
	{1, 3, 2}, {2, 4, 3}, {3, 4, 2}, {4, 3, 2}, {5, 2, 2},
	{6, 3, 2}, {7, 2, 2}, {8, 3, 2}, {9, 3, 1}, {10, 3, 2},
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if iv, err := strconv.Atoi(v); err == nil {
			return iv
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if v := os.Getenv(key); v != "" {
		if bv, err := strconv.ParseBool(v); err == nil {
			return bv
		}
	}
	return defaultVal
}

func getEnvString(key string, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func NewEtherCATSimulator() *EtherCATSimulator {
	apiEndpoint := getEnvString("API_ENDPOINT", "http://127.0.0.1:8080")
	intervalSeconds := getEnvInt("INTERVAL_SECONDS", 7200)
	deviceCount := getEnvInt("DEVICE_COUNT", 50)
	highScaleMode := getEnvBool("HIGH_SCALE_MODE", false)
	backfillDays := getEnvInt("BACKFILL_DAYS", 7)

	sim := &EtherCATSimulator{
		apiEndpoint:   apiEndpoint,
		interval:      time.Duration(intervalSeconds) * time.Second,
		stopChan:      make(chan struct{}),
		historicStart: time.Now().AddDate(0, 0, -backfillDays),
		rng:           rand.New(rand.NewSource(time.Now().UnixNano())),
		httpClient:    &http.Client{Timeout: 60 * time.Second},
		sendSem:       make(chan struct{}, 4),
		deviceCount:   deviceCount,
		highScaleMode: highScaleMode,
		backfillDays:  backfillDays,
		highScaleEvents: []HighScaleEvent{
			{StartHour: 0, EndHour: 24, Multiplier: 5.0, SO2Boost: 40.0, HumidityBoost: 20.0},
		},
	}
	sim.initSensors()
	return sim
}

func (s *EtherCATSimulator) initSensors() {
	layout := s.buildDeviceLayout()
	nextUS := uint64(1)
	nextRT := uint64(10000)

	for _, layoutItem := range layout {
		for i := 0; i < layoutItem.Ultrasonic; i++ {
			baseV := 0.3 + s.rng.Float64()*1.5
			driftRate := 0.0005 + s.rng.Float64()*0.0015
			if s.highScaleMode {
				driftRate *= 5.0
				baseV = 1.5 + s.rng.Float64()*2.0
			}
			s.sensors = append(s.sensors, &SensorConfig{
				ID:           nextUS,
				RelicID:      layoutItem.RelicID,
				Type:         "ultrasonic",
				BaseValue:    baseV,
				DriftRate:    driftRate,
				CurrentValue: baseV,
				LastUpdate:   s.historicStart,
			})
			nextUS++
		}
		for i := 0; i < layoutItem.Roughness; i++ {
			baseV := 5.0 + s.rng.Float64()*20.0
			driftRate := 0.01 + s.rng.Float64()*0.05
			if s.highScaleMode {
				driftRate *= 3.0
				baseV = 25.0 + s.rng.Float64()*30.0
			}
			s.sensors = append(s.sensors, &SensorConfig{
				ID:           nextRT,
				RelicID:      layoutItem.RelicID,
				Type:         "roughness",
				BaseValue:    baseV,
				DriftRate:    driftRate,
				CurrentValue: baseV,
				LastUpdate:   s.historicStart,
			})
			nextRT++
		}
	}
	zap.L().Info(fmt.Sprintf("Initialized %d sensors across %d relics (target %d devices)",
		len(s.sensors), len(layout), s.deviceCount))
}

func (s *EtherCATSimulator) buildDeviceLayout() []struct {
	RelicID    uint64
	Ultrasonic int
	Roughness  int
} {
	relicCount := 10
	targetPerRelic := s.deviceCount / relicCount
	if targetPerRelic < 5 {
		return defaultRelicLayout
	}

	layout := make([]struct {
		RelicID    uint64
		Ultrasonic int
		Roughness  int
	}, relicCount)

	for i := 0; i < relicCount; i++ {
		usCount := targetPerRelic * 3 / 5
		if usCount < 2 {
			usCount = 2
		}
		rtCount := targetPerRelic - usCount
		if rtCount < 1 {
			rtCount = 1
		}

		if i == 1 {
			usCount += 2
			rtCount += 1
		}
		if i == 2 {
			usCount += 1
		}

		layout[i] = struct {
			RelicID    uint64
			Ultrasonic int
			Roughness  int
		}{
			RelicID:    uint64(i + 1),
			Ultrasonic: usCount,
			Roughness:  rtCount,
		}
	}

	return layout
}

func (s *EtherCATSimulator) getHighScaleMultiplier(ts time.Time) (float64, float32, float32) {
	if !s.highScaleMode {
		return 1.0, 0, 0
	}
	hour := ts.Hour()
	for _, evt := range s.highScaleEvents {
		if hour >= evt.StartHour && hour < evt.EndHour {
			return evt.Multiplier, evt.SO2Boost, evt.HumidityBoost
		}
	}
	return 1.0, 0, 0
}

func (s *EtherCATSimulator) simulateValue(sensor *SensorConfig, ts time.Time) float64 {
	hoursElapsed := ts.Sub(s.historicStart).Hours()
	diurnalCycle := math.Sin(2*math.Pi*float64(ts.Hour())/24) * 0.08
	seasonalCycle := math.Sin(2*math.Pi*float64(ts.YearDay())/365) * 0.12

	mult, _, _ := s.getHighScaleMultiplier(ts)
	growthTrend := sensor.DriftRate * hoursElapsed * mult
	randomNoise := (s.rng.Float64() - 0.5) * 0.15

	anomaly := 0.0
	if s.rng.Float64() < 0.01 {
		anomaly = s.rng.Float64() * 2.0 * mult
		zap.L().Warn(fmt.Sprintf("Sensor %d anomaly spike generated", sensor.ID))
	}

	value := sensor.BaseValue * (1 + growthTrend + diurnalCycle + seasonalCycle + randomNoise)
	value += anomaly

	if value < 0 {
		value = 0
	}
	if sensor.Type == "ultrasonic" && value > 10 {
		value = 10
	}
	if sensor.Type == "roughness" && value > 300 {
		value = 300
	}
	return value
}

func (s *EtherCATSimulator) generateSO2(relicID uint64, ts time.Time) float32 {
	base := 10.0 + float64(relicID)*2.5
	seasonal := 8.0 * math.Sin(2*math.Pi*float64(ts.YearDay())/365+1.5)
	random := (s.rng.Float64() - 0.3) * 5.0
	_, so2Boost, _ := s.getHighScaleMultiplier(ts)
	return float32(math.Max(0, base+seasonal+random)) + so2Boost
}

func (s *EtherCATSimulator) generateHumidity(relicID uint64, ts time.Time) float32 {
	baseHumidity := map[uint64]float64{
		1: 45, 2: 75, 3: 55, 4: 35, 5: 60,
		6: 70, 7: 50, 8: 48, 9: 52, 10: 40,
	}
	base := baseHumidity[relicID]
	diurnal := -10.0 * math.Sin(2*math.Pi*float64(ts.Hour()-6)/24)
	random := (s.rng.Float64() - 0.5) * 8.0
	_, _, humBoost := s.getHighScaleMultiplier(ts)
	h := base + diurnal + random + float64(humBoost)
	return float32(math.Max(10, math.Min(99, h)))
}

func (s *EtherCATSimulator) generateTemperature(relicID uint64, ts time.Time) float32 {
	baseTemp := map[uint64]float64{
		1: 8, 2: 18, 3: 14, 4: 11, 5: 10,
		6: 17, 7: 12, 8: 9, 9: 13, 10: 7,
	}
	base := baseTemp[relicID]
	seasonal := 15.0 * math.Sin(2*math.Pi*(float64(ts.YearDay())-80)/365)
	diurnal := 6.0 * math.Sin(2*math.Pi*float64(ts.Hour()-14)/24)
	random := (s.rng.Float64() - 0.5) * 2.0
	return float32(base + seasonal + diurnal + random)
}

func (s *EtherCATSimulator) generateBatch(ts time.Time) []SensorData {
	var batch []SensorData
	baseID := uint64(ts.UnixNano() / 1e6)

	for i, sensor := range s.sensors {
		value := s.simulateValue(sensor, ts)
		unit := "mm"
		if sensor.Type == "roughness" {
			unit = "μm"
		}

		batch = append(batch, SensorData{
			ID:               baseID + uint64(i),
			SensorID:         sensor.ID,
			RelicID:          sensor.RelicID,
			Timestamp:        ts,
			Value:            float32(value),
			Unit:             unit,
			SO2Concentration: s.generateSO2(sensor.RelicID, ts),
			Humidity:         s.generateHumidity(sensor.RelicID, ts),
			Temperature:      s.generateTemperature(sensor.RelicID, ts),
		})

		sensor.CurrentValue = value
		sensor.LastUpdate = ts
	}
	return batch
}

func (s *EtherCATSimulator) sendBatch(batch []SensorData) error {
	payload := map[string]interface{}{
		"data": batch,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := s.apiEndpoint + "/api/v1/sensors/upload"
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-EtherCAT-Node", fmt.Sprintf("sim-node-%d", time.Now().Unix()%100))

	client := s.httpClient
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	s.mu.Lock()
	s.totalSent += uint64(len(batch))
	for _, d := range batch {
		if (d.Unit == "mm" && d.Value > 3.0) || (d.Unit == "μm" && d.Value > 50.0) {
			s.alertCounter++
		}
	}
	s.mu.Unlock()

	return nil
}

func (s *EtherCATSimulator) BackfillHistory() {
	zap.L().Info(fmt.Sprintf("Starting historical data backfill for %d days...", s.backfillDays))
	step := 2 * time.Hour
	current := s.historicStart
	end := time.Now().Add(-2 * time.Hour)
	batchSize := 6

	var pending []SensorData
	count := 0
	interval := s.interval
	if interval < time.Hour {
		interval = 2 * time.Hour
	}

	for current.Before(end) {
		batch := s.generateBatch(current)
		pending = append(pending, batch...)

		if len(pending) >= batchSize*len(s.sensors) {
			if err := s.sendBatch(pending); err != nil {
				zap.L().Error("Backfill batch failed", zap.Error(err))
			} else {
				count += len(pending)
			}
			pending = nil
			time.Sleep(50 * time.Millisecond)
		}
		current = current.Add(interval)
	}
	if len(pending) > 0 {
		if err := s.sendBatch(pending); err != nil {
			zap.L().Error("Final backfill batch failed", zap.Error(err))
		} else {
			count += len(pending)
		}
	}
	zap.L().Info(fmt.Sprintf("Historical backfill complete: %d records inserted across %d sensors",
		count, len(s.sensors)))
}

func (s *EtherCATSimulator) Start() {
	zap.L().Info(fmt.Sprintf("EtherCAT simulator started: interval=%v, endpoint=%s, devices=%d, high_scale=%v",
		s.interval, s.apiEndpoint, len(s.sensors), s.highScaleMode))

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	statsTicker := time.NewTicker(5 * time.Minute)
	defer statsTicker.Stop()

	var prevSendTime time.Time

	for {
		select {
		case ts := <-ticker.C:
			if !prevSendTime.IsZero() && ts.Sub(prevSendTime) < s.interval-time.Second {
				continue
			}

			batch := s.generateBatch(ts)
			prevSendTime = ts

			select {
			case s.sendSem <- struct{}{}:
				go func(b []SensorData, sem chan struct{}) {
					defer func() {
						<-sem
						if r := recover(); r != nil {
							zap.L().Error("EtherCAT send goroutine panic recovered",
								zap.Any("recover", r),
								zap.Int("batch_size", len(b)))
						}
					}()
					if err := s.sendBatch(b); err != nil {
						zap.L().Error("Send batch failed", zap.Error(err), zap.Time("timestamp", b[0].Timestamp))
					} else {
						zap.L().Debug(fmt.Sprintf("Sent EtherCAT batch: %d samples at %s",
							len(b), b[0].Timestamp.Format("2006-01-02 15:04:05")))
					}
				}(batch, s.sendSem)
			default:
				zap.L().Warn("Send semaphore full, skipping batch to prevent jitter",
					zap.Int("batch_size", len(batch)))
			}

		case <-statsTicker.C:
			s.mu.Lock()
			zap.L().Info(fmt.Sprintf("Simulator stats: total=%d, alerts=%d, sensors=%d, interval=%v",
				s.totalSent, s.alertCounter, len(s.sensors), s.interval))
			s.mu.Unlock()

		case <-s.stopChan:
			zap.L().Info("EtherCAT simulator stopped")
			return
		}
	}
}

func (s *EtherCATSimulator) Stop() {
	close(s.stopChan)
}

func (s *EtherCATSimulator) GetStats() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]interface{}{
		"total_sent":     s.totalSent,
		"alerts":         s.alertCounter,
		"sensors":        len(s.sensors),
		"interval_s":     s.interval.Seconds(),
		"api_endpoint":   s.apiEndpoint,
		"device_count":   s.deviceCount,
		"high_scale_mode": s.highScaleMode,
	}
}

func waitForBackend(endpoint string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(endpoint + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return true
		}
		if resp != nil {
			resp.Body.Close()
		}
		zap.L().Info(fmt.Sprintf("Waiting for backend... %v until ready", time.Until(deadline).Round(time.Second)))
		time.Sleep(5 * time.Second)
	}
	return false
}

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

	sim := NewEtherCATSimulator()

	zap.L().Info("Waiting for backend to be ready...")
	if !waitForBackend(sim.apiEndpoint, 5*time.Minute) {
		zap.L().Fatal("Backend not ready after 5 minutes")
	}
	zap.L().Info("Backend is ready")

	sim.BackfillHistory()
	sim.Start()
}
