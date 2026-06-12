package alert

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"net/http"
	"stone-relic-monitor/internal/models"
	"sync"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	Hub  *Hub
	Conn *websocket.Conn
	Send chan []byte
	mu   sync.Mutex
}

type Hub struct {
	Clients    map[*Client]bool
	Broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		Broadcast:  make(chan []byte, 256),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client] = true
			h.mu.Unlock()
			zap.L().Info("New WS client connected", zap.Int("total", len(h.Clients)))
		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
			zap.L().Info("WS client disconnected", zap.Int("total", len(h.Clients)))
		case message := <-h.Broadcast:
			h.mu.RLock()
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) BroadcastAlert(alert *models.AlertRecord) {
	msg := map[string]interface{}{
		"type": "alert",
		"data": alert,
	}
	if data, err := json.Marshal(msg); err == nil {
		h.Broadcast <- data
	}
}

func (h *Hub) BroadcastSensorData(data *models.SensorData) {
	msg := map[string]interface{}{
		"type": "sensor_data",
		"data": data,
	}
	if jsonData, err := json.Marshal(msg); err == nil {
		h.Broadcast <- jsonData
	}
}

func (h *Hub) BroadcastStats(stats map[string]interface{}) {
	msg := map[string]interface{}{
		"type": "stats",
		"data": stats,
	}
	if data, err := json.Marshal(msg); err == nil {
		h.Broadcast <- data
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()
	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				zap.L().Warn("WS read error", zap.Error(err))
			}
			break
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.mu.Lock()
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				c.mu.Unlock()
				return
			}
			c.Conn.WriteMessage(websocket.TextMessage, message)
			c.mu.Unlock()
		case <-ticker.C:
			c.mu.Lock()
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			c.Conn.WriteMessage(websocket.PingMessage, nil)
			c.mu.Unlock()
		}
	}
}

func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		zap.L().Error("Failed to upgrade WS", zap.Error(err))
		return
	}
	client := &Client{Hub: hub, Conn: conn, Send: make(chan []byte, 256)}
	client.Hub.Register <- client
	go client.WritePump()
	go client.ReadPump()
}
