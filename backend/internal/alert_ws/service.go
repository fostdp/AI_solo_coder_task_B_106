package alert_ws

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"stone-relic-monitor/internal/config"
	"stone-relic-monitor/internal/db"
	"stone-relic-monitor/internal/models"
)

const (
	alertDedupWindow = time.Hour
)

type AlertService struct {
	cfg               *config.Config
	db                *db.ClickHouse
	SensorDataInChan  <-chan *models.SensorData
	hub               *Hub
	alertCache        map[string]time.Time
	alertCacheMu      sync.Mutex
}

func NewAlertService(
	cfg *config.Config,
	db *db.ClickHouse,
	dataInChan <-chan *models.SensorData,
) *AlertService {
	svc := &AlertService{
		cfg:              cfg,
		db:               db,
		SensorDataInChan: dataInChan,
		hub:              NewHub(),
		alertCache:       make(map[string]time.Time),
	}
	go svc.hub.Run()
	return svc
}

func (s *AlertService) Run(stopChan <-chan struct{}) {
	zap.L().Info("AlertService started")

	for {
		select {
		case <-stopChan:
			zap.L().Info("AlertService stopped")
			return
		case data, ok := <-s.SensorDataInChan:
			if !ok {
				return
			}
			s.CheckAndAlert(data)
		}
	}
}

func (s *AlertService) CheckAndAlert(data *models.SensorData) {
	threshold := s.cfg.Threshold
	var alertType, message string
	var value float32

	switch data.Unit {
	case "mm":
		thresholdVal := float32(threshold.ScaleThicknessMM)
		if data.Value > thresholdVal {
			alertType = "SCALE_THICKNESS"
			message = "结垢厚度超标"
			value = data.Value
		}
	case "μm", "um":
		thresholdVal := float32(threshold.RoughnessUM)
		if data.Value > thresholdVal {
			alertType = "SURFACE_ROUGHNESS"
			message = "表面粗糙度超标"
			value = data.Value
		}
	}

	if alertType == "" {
		return
	}

	cacheKey := strconv.FormatUint(data.SensorID, 10) + ":" + alertType
	s.alertCacheMu.Lock()
	if lastAlert, exists := s.alertCache[cacheKey]; exists && time.Since(lastAlert) < alertDedupWindow {
		s.alertCacheMu.Unlock()
		return
	}
	s.alertCache[cacheKey] = time.Now()
	s.alertCacheMu.Unlock()

	alert := &models.AlertRecord{
		ID:         uint64(time.Now().UnixNano() / 1e6),
		RelicID:    data.RelicID,
		SensorID:   data.SensorID,
		Timestamp:  time.Now(),
		AlertType:  alertType,
		Severity:   "WARNING",
		Message:    message,
		Value:      value,
		Threshold:  float32(s.getThreshold(alertType)),
		Resolved:   false,
	}

	if err := s.saveAlert(alert); err != nil {
		zap.L().Error("Save alert failed", zap.Error(err))
	}

	go func() {
		if err := s.sendDingTalk(alert); err != nil {
			zap.L().Error("DingTalk alert failed", zap.Error(err))
		}
	}()

	s.hub.BroadcastAlert(alert)
}

func (s *AlertService) getThreshold(alertType string) float64 {
	switch alertType {
	case "SCALE_THICKNESS":
		return s.cfg.Threshold.ScaleThicknessMM
	case "SURFACE_ROUGHNESS":
		return s.cfg.Threshold.RoughnessUM
	}
	return 0
}

func (s *AlertService) saveAlert(alert *models.AlertRecord) error {
	query := `INSERT INTO alert_record
		(id, relic_id, sensor_id, timestamp, alert_type, severity, message, value, threshold, resolved)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(context.Background(), query,
		alert.ID, alert.RelicID, alert.SensorID, alert.Timestamp,
		alert.AlertType, alert.Severity, alert.Message,
		alert.Value, alert.Threshold, alert.Resolved,
	)
	return err
}

func (s *AlertService) sendDingTalk(alert *models.AlertRecord) error {
	if s.cfg.Alert.WebhookURL == "" {
		return nil
	}

	timestamp := time.Now().UnixMilli()
	sign := s.computeDingTalkSign(timestamp, s.cfg.Alert.Secret)

	url := s.cfg.Alert.WebhookURL + "&timestamp=" + strconv.FormatInt(timestamp, 10) + "&sign=" + sign

	title := "【石质文物监测告警】"
	content := "## " + title + "\n\n" +
		"**告警时间**: " + alert.Timestamp.Format("2006-01-02 15:04:05") + "\n\n" +
		"**文物ID**: " + strconv.FormatUint(alert.RelicID, 10) + "\n\n" +
		"**传感器ID**: " + strconv.FormatUint(alert.SensorID, 10) + "\n\n" +
		"**告警类型**: " + alert.AlertType + "\n\n" +
		"**当前值**: " + strconv.FormatFloat(float64(alert.Value), 'f', 3, 32) + "\n\n" +
		"**阈值**: " + strconv.FormatFloat(float64(alert.Threshold), 'f', 3, 32) + "\n\n" +
		"**严重级别**: " + alert.Severity + "\n\n" +
		"**消息**: " + alert.Message

	msg := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  content,
		},
		"at": map[string]interface{}{
			"isAtAll": true,
		},
	}

	body, _ := json.Marshal(msg)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (s *AlertService) computeDingTalkSign(timestamp int64, secret string) string {
	stringToSign := strconv.FormatInt(timestamp, 10) + "\n" + secret
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (s *AlertService) GetAlerts(c *gin.Context) {
	relicID, err := strconv.ParseUint(c.Param("relic_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid relic_id"})
		return
	}

	hours := 24
	if h := c.Query("hours"); h != "" {
		if v, err := strconv.Atoi(h); err == nil && v > 0 {
			hours = v
		}
	}

	unresolvedOnly := false
	if c.Query("unresolved") == "true" {
		unresolvedOnly = true
	}

	ctx := context.Background()
	var query string
	var args []interface{}
	if unresolvedOnly {
		query = `SELECT id, relic_id, sensor_id, timestamp, alert_type, severity, message, value, threshold, resolved, resolved_at, resolution_notes
			FROM alert_record WHERE relic_id = ? AND timestamp > now() - INTERVAL ? HOUR AND resolved = 0
			ORDER BY timestamp DESC LIMIT 500`
		args = []interface{}{relicID, hours}
	} else {
		query = `SELECT id, relic_id, sensor_id, timestamp, alert_type, severity, message, value, threshold, resolved, resolved_at, resolution_notes
			FROM alert_record WHERE relic_id = ? AND timestamp > now() - INTERVAL ? HOUR
			ORDER BY timestamp DESC LIMIT 500`
		args = []interface{}{relicID, hours}
	}

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var alerts []models.AlertRecord
	for rows.Next() {
		var a models.AlertRecord
		if err := rows.Scan(&a.ID, &a.RelicID, &a.SensorID, &a.Timestamp,
			&a.AlertType, &a.Severity, &a.Message, &a.Value, &a.Threshold,
			&a.Resolved, &a.ResolvedAt, &a.ResolutionNotes); err == nil {
			alerts = append(alerts, a)
		}
	}

	c.JSON(http.StatusOK, gin.H{"count": len(alerts), "data": alerts})
}

func (s *AlertService) ResolveAlert(c *gin.Context) {
	alertID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert_id"})
		return
	}

	var req struct {
		ResolutionNotes string `json:"resolution_notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `ALTER TABLE alert_record UPDATE resolved = 1, resolved_at = now(), resolution_notes = ? WHERE id = ?`
	_, err = s.db.Exec(context.Background(), query, req.ResolutionNotes, alertID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "resolved", "alert_id": alertID})
}

func (s *AlertService) GetAlertStats(c *gin.Context) {
	ctx := context.Background()
	query := `SELECT count(), resolved FROM alert_record WHERE timestamp > now() - INTERVAL 24 HOUR GROUP BY resolved`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	total := int64(0)
	unresolved := int64(0)
	for rows.Next() {
		var cnt int64
		var res bool
		if err := rows.Scan(&cnt, &res); err == nil {
			total += cnt
			if !res {
				unresolved += cnt
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_alerts_24h":   total,
		"unresolved_alerts":  unresolved,
		"resolved_alerts":    total - unresolved,
	})
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *AlertService) ServeWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		zap.L().Error("WebSocket upgrade failed", zap.Error(err))
		return
	}

	client := NewClient(conn, s.hub)
	s.hub.Register(client)
	go client.WritePump()
	client.ReadPump()
}

func (s *AlertService) RegisterRoutes(r *gin.RouterGroup) {
	alerts := r.Group("/alerts")
	{
		alerts.GET("/relic/:relic_id", s.GetAlerts)
		alerts.POST("/:id/resolve", s.ResolveAlert)
		alerts.GET("/stats", s.GetAlertStats)
	}

	ws := r.Group("/ws")
	{
		ws.GET("", s.ServeWS)
	}
}
