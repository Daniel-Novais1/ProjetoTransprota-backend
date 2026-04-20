package infrastructure

import (
	"context"
	"net/http"
	"sync"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// WebSocketHub gerencia clientes conectados
type WebSocketHub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.Mutex
	redis      *redis.Client
}

// NewWebSocketHub cria um novo hub
func NewWebSocketHub(rdb *redis.Client) *WebSocketHub {
	hub := &WebSocketHub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		redis:      rdb,
	}
	go hub.run()
	// Só iniciar subscribeToRedis se Redis estiver conectado
	if rdb != nil {
		go hub.subscribeToRedis()
	}
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
				client.Close()
			}
			h.mu.Unlock()
			logger.Info("WebSocket", "Cliente desconectado. Total: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				err := client.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					logger.Error("WebSocket", "Erro ao enviar: %v", err)
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

// subscribeToRedis ouve canal Redis e faz broadcast
func (h *WebSocketHub) subscribeToRedis() {
	if h.redis == nil {
		logger.Warn("WebSocket", "Redis não está conectado, subscribeToRedis desabilitado")
		return
	}
	ctx := context.Background()
	pubsub := h.redis.Subscribe(ctx, "bus_updates")
	ch := pubsub.Channel()

	for msg := range ch {
		logger.Info("WebSocket", "Mensagem do Redis: %s", msg.Payload)
		h.broadcast <- []byte(msg.Payload)
	}
}

// HandleWebSocket trata conexões WebSocket
func (h *WebSocketHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("WebSocket", "Erro ao fazer upgrade: %v", err)
		return
	}

	h.register <- conn

	go func() {
		defer func() {
			h.unregister <- conn
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}
