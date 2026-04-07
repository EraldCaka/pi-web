package ws

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
)

type MessageType string

const (
	TypeSensorData    MessageType = "sensor_data"
	TypeGPIOEvent     MessageType = "gpio_event"
	TypeMQTTMessage   MessageType = "mqtt_message"
	TypeDeviceStatus  MessageType = "device_status"
	TypeDeviceMetrics MessageType = "device_metrics"
	TypeError         MessageType = "error"
)

type Message struct {
	Type      MessageType `json:"type"`
	Timestamp int64       `json:"timestamp"`
	DeviceID  string      `json:"device_id,omitempty"`
	Payload   any         `json:"payload"`
}

func NewMessage(t MessageType, deviceID string, payload any) []byte {
	m := Message{
		Type:      t,
		Timestamp: time.Now().UnixMilli(),
		DeviceID:  deviceID,
		Payload:   payload,
	}
	b, _ := json.Marshal(m)
	return b
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

const sendBuf = 64

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	logger     *slog.Logger
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
	}
}

func (h *Hub) Broadcast(msg []byte) {
	select {
	case h.broadcast <- msg:
	default:
		h.logger.Warn("hub broadcast channel full, dropping message")
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = true
			h.mu.Unlock()
			h.logger.Info("ws client connected", "total", len(h.clients))

		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()
			h.logger.Info("ws client disconnected", "total", len(h.clients))

		case msg := <-h.broadcast:
			h.mu.RLock()
			for c := range h.clients {
				select {
				case c.send <- msg:
				default:
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) ServeWS(conn *websocket.Conn) {
	c := &Client{hub: h, conn: conn, send: make(chan []byte, sendBuf)}
	h.register <- c

	go c.writePump()
	c.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(512)
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}
