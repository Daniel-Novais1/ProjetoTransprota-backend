package telemetry

import (
	"context"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	HandshakeTimeout: 10 * time.Second,
}

const (
	MaxReconnectAttempts  = 10
	InitialReconnectDelay = 1 * time.Second
	MaxReconnectDelay     = 30 * time.Second
	PingInterval          = 30 * time.Second
	PongTimeout           = 60 * time.Second
)

type WebSocketClient struct {
	conn              *websocket.Conn
	hub               *WebSocketHub
	reconnectAttempts int
	lastPing          time.Time
	mu                sync.Mutex
}

// WebSocketHub gerencia clientes conectados
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan []byte
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mu         sync.Mutex
	redis      *redis.Client
	upgrader   *websocket.Upgrader
}

// NewWebSocketHub cria um novo hub
func NewWebSocketHub(rdb *redis.Client) *WebSocketHub {
	hub := &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		redis:      rdb,
		upgrader:   &upgrader,
	}
	go hub.run()
	go hub.subscribeToRedis()
	return hub
}

func (h *WebSocketHub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			logger.Info("WebSocket", "Cliente conectado. Total: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.conn.Close()
			}
			h.mu.Unlock()
			logger.Info("WebSocket", "Cliente desconectado. Total: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				client.mu.Lock()
				err := client.conn.WriteMessage(websocket.TextMessage, message)
				client.mu.Unlock()
				if err != nil {
					logger.Error("WebSocket", "Erro ao enviar: %v", err)
					go h.handleClientDisconnect(client)
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *WebSocketHub) handleClientDisconnect(client *WebSocketClient) {
	h.unregister <- client
	go h.attemptReconnect(client)
}

func (h *WebSocketHub) attemptReconnect(client *WebSocketClient) {
	delay := InitialReconnectDelay
	maxAttempts := MaxReconnectAttempts

	for i := 0; i < maxAttempts; i++ {
		time.Sleep(delay)

		logger.Info("WebSocket", "Tentativa de reconexão %d/%d (delay: %v)", i+1, maxAttempts, delay)

		err := h.tryReconnect(client)
		if err == nil {
			logger.Info("WebSocket", "Reconexão bem-sucedida após %d tentativas", i+1)
			client.reconnectAttempts = 0
			return
		}

		delay = time.Duration(math.Min(float64(delay)*1.5, float64(MaxReconnectDelay)))
	}

	logger.Warn("WebSocket", "Falha na reconexão após %d tentativas", maxAttempts)
}

func (h *WebSocketHub) tryReconnect(client *WebSocketClient) error {
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.conn != nil {
		client.conn.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, "ws://localhost:8081/ws", nil)
	if err != nil {
		return err
	}

	client.conn = conn
	client.lastPing = time.Now()

	conn.SetPingHandler(func(appData string) error {
		client.lastPing = time.Now()
		return conn.WriteMessage(websocket.PongMessage, []byte(appData))
	})

	conn.SetPongHandler(func(appData string) error {
		client.lastPing = time.Now()
		return nil
	})

	go h.startPingLoop(client)

	h.register <- client
	return nil
}

func (h *WebSocketHub) startPingLoop(client *WebSocketClient) {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			client.mu.Lock()

			if time.Since(client.lastPing) > PongTimeout {
				logger.Warn("WebSocket", "Cliente não respondeu ao ping - desconectando")
				client.mu.Unlock()
				h.handleClientDisconnect(client)
				return
			}

			err := client.conn.WriteMessage(websocket.PingMessage, nil)
			client.mu.Unlock()

			if err != nil {
				logger.Warn("WebSocket", "Erro ao enviar ping: %v", err)
				h.handleClientDisconnect(client)
				return
			}
		}
	}
}

// subscribeToRedis ouve canal Redis e faz broadcast
func (h *WebSocketHub) subscribeToRedis() {
	ctx := context.Background()
	pubsub := h.redis.Subscribe(ctx, "bus_updates")
	ch := pubsub.Channel()

	for msg := range ch {
		logger.Info("WebSocket", "Mensagem do Redis: %s", msg.Payload)
		h.broadcast <- []byte(msg.Payload)
	}
}

// HandleWebSocket endpoint
func (h *WebSocketHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("WebSocket", "Erro no upgrade: %v", err)
		return
	}

	client := &WebSocketClient{
		conn:              conn,
		hub:               h,
		reconnectAttempts: 0,
		lastPing:          time.Now(),
	}

	conn.SetPingHandler(func(appData string) error {
		client.lastPing = time.Now()
		return conn.WriteMessage(websocket.PongMessage, []byte(appData))
	})

	conn.SetPongHandler(func(appData string) error {
		client.lastPing = time.Now()
		return nil
	})

	go h.startPingLoop(client)
	h.register <- client

	go func() {
		defer func() {
			h.unregister <- client
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}
