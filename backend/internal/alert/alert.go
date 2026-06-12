package alert

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"stone-relic-monitor/internal/config"
	"stone-relic-monitor/internal/db"
	"stone-relic-monitor/internal/models"
	"time"
)

type AlertService struct {
	cfg      *config.Config
	db       *db.ClickHouse
	wsHub    *Hub
	lastAlert map[string]time.Time
}

func NewAlertService(cfg *config.Config, db *db.ClickHouse, wsHub *Hub) *AlertService {
	return &AlertService{
		cfg:       cfg,
		db:        db,
		wsHub:     wsHub,
		lastAlert: make(map[string]time.Time),
	}
}

func (s *AlertService) CheckAndAlert(data *models.SensorData, sensorType string) error {
	var threshold float64
	var alertType string
	var unit string

	if sensorType == "ultrasonic" {
		threshold = s.cfg.Threshold.ScaleThicknessMM
		alertType = "thickness"
		unit = "mm"
	} else if sensorType == "roughness" {
		threshold = s.cfg.Threshold.RoughnessUM
		alertType = "roughness"
		unit = "μm"
	} else {
		return nil
	}

	if float64(data.Value) <= threshold {
		return nil
	}

	key := fmt.Sprintf("%d-%s", data.SensorID, alertType)
	if lastTime, exists := s.lastAlert[key]; exists {
		if time.Since(lastTime) < 1*time.Hour {
			return nil
		}
	}
	s.lastAlert[key] = time.Now()

	level := "warning"
	ratio := float64(data.Value) / threshold
	if ratio > 1.5 {
		level = "critical"
	}

	alert := &models.AlertRecord{
		RelicID:   data.RelicID,
		SensorID:  data.SensorID,
		Type:      alertType,
		Level:     level,
		Value:     data.Value,
		Threshold: float32(threshold),
		Message:   fmt.Sprintf("%s超标: %.2f%s (阈值: %.1f%s)", alertTypeName(alertType), data.Value, unit, threshold, unit),
		CreatedAt: time.Now(),
	}

	if err := s.saveAlert(alert); err != nil {
		zap.L().Error("Failed to save alert", zap.Error(err))
	}

	if err := s.sendDingTalk(alert); err != nil {
		zap.L().Warn("Failed to send DingTalk alert", zap.Error(err))
	}

	if s.cfg.Alert.EnableWS {
		s.wsHub.BroadcastAlert(alert)
	}

	zap.L().Info("Alert triggered",
		zap.String("type", alertType),
		zap.String("level", level),
		zap.Float32("value", data.Value),
		zap.Uint64("relic_id", data.RelicID))

	return nil
}

func alertTypeName(t string) string {
	switch t {
	case "thickness":
		return "结垢厚度"
	case "roughness":
		return "表面粗糙度"
	default:
		return t
	}
}

func (s *AlertService) saveAlert(alert *models.AlertRecord) error {
	ctx := context.Background()
	query := `INSERT INTO alert_record (relic_id, sensor_id, type, level, value, threshold, message, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	return s.db.Exec(ctx, query,
		alert.RelicID, alert.SensorID, alert.Type, alert.Level,
		alert.Value, alert.Threshold, alert.Message, alert.CreatedAt)
}

func (s *AlertService) sendDingTalk(alert *models.AlertRecord) error {
	if s.cfg.Alert.DingtalkWebhook == "" || s.cfg.Alert.DingtalkWebhook == "https://oapi.dingtalk.com/robot/send?access_token=YOUR_TOKEN_HERE" {
		return nil
	}

	webhookURL := s.cfg.Alert.DingtalkWebhook
	secret := s.cfg.Alert.DingtalkSecret

	if secret != "" && secret != "YOUR_SECRET_HERE" {
		timestamp := time.Now().UnixMilli()
		stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
		hmac256 := hmac.New(sha256.New, []byte(secret))
		hmac256.Write([]byte(stringToSign))
		sign := url.QueryEscape(base64.StdEncoding.EncodeToString(hmac256.Sum(nil)))
		webhookURL = fmt.Sprintf("%s&timestamp=%d&sign=%s", webhookURL, timestamp, sign)
	}

	emoji := "⚠️"
	if alert.Level == "critical" {
		emoji = "🚨"
	}

	message := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": "石质文物监测告警",
			"text": fmt.Sprintf(
				"## %s 石质文物告警\n\n**告警级别:** %s%s\n\n**文物ID:** %d\n\n**传感器ID:** %d\n\n**告警类型:** %s\n\n**当前值:** %.2f\n\n**阈值:** %.2f\n\n**时间:** %s\n\n> 请及时处理！",
				emoji, emoji, alert.Level,
				alert.RelicID, alert.SensorID,
				alertTypeName(alert.Type),
				alert.Value, alert.Threshold,
				alert.CreatedAt.Format("2006-01-02 15:04:05"),
			),
		},
		"at": map[string]interface{}{
			"isAtAll": alert.Level == "critical",
		},
	}

	body, _ := json.Marshal(message)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dingtalk returned status: %d", resp.StatusCode)
	}

	return nil
}
