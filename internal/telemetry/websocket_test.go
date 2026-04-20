package telemetry

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestWebSocketLatency valida se mensagens chegam via WebSocket em menos de 200ms
func TestWebSocketLatency(t *testing.T) {
	// URL do WebSocket
	wsURL := "ws://localhost:8081/api/v1/telemetry/ws"

	// Conectar ao WebSocket
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Skipf("Servidor não está rodando (teste requer servidor ativo): %v", err)
		return
	}
	defer conn.Close()

	log.Printf("[WebSocket] Conectado a %s", wsURL)

	// Aguardar mensagens do WebSocket
	done := make(chan bool)
	var latency time.Duration

	go func() {
		defer close(done)

		startTime := time.Now()

		// Ler mensagem do WebSocket
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[WebSocket] Erro ao ler mensagem: %v", err)
			return
		}

		latency = time.Since(startTime)

		if messageType != websocket.TextMessage {
			t.Errorf("Esperava TextMessage, mas recebeu: %d", messageType)
			return
		}

		// Validar payload JSON
		var payload map[string]interface{}
		if err := json.Unmarshal(message, &payload); err != nil {
			t.Errorf("Erro ao deserializar JSON: %v", err)
			return
		}

		// Validar campos obrigatórios
		requiredFields := []string{"id", "lat", "lng", "speed", "traffic_status"}
		for _, field := range requiredFields {
			if _, ok := payload[field]; !ok {
				t.Errorf("Campo obrigatório '%s' não encontrado no payload", field)
			}
		}

		log.Printf("[WebSocket] Mensagem recebida: %s", string(message))
		log.Printf("[WebSocket] Latência: %v", latency)
	}()

	// Timeout de 5 segundos para receber mensagem
	select {
	case <-done:
		// Validar latência
		if latency > 200*time.Millisecond {
			t.Errorf("Latência excedeu 200ms: %v", latency)
		} else {
			log.Printf("[WebSocket] ✅ Latência dentro do limite: %v", latency)
		}
	case <-time.After(5 * time.Second):
		t.Error("Timeout: nenhuma mensagem recebida em 5 segundos")
	}
}

// TestWebSocketConnection testa apenas a conexão WebSocket
func TestWebSocketConnection(t *testing.T) {
	wsURL := "ws://localhost:8081/api/v1/telemetry/ws"

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Skipf("Servidor não está rodando (teste requer servidor ativo): %v", err)
		return
	}
	defer conn.Close()

	log.Printf("[WebSocket] ✅ Conexão estabelecida com sucesso")

	// Enviar mensagem de teste (opcional)
	testMsg := map[string]string{"type": "ping"}
	jsonMsg, _ := json.Marshal(testMsg)
	err = conn.WriteMessage(websocket.TextMessage, jsonMsg)
	if err != nil {
		t.Errorf("Erro ao enviar mensagem de teste: %v", err)
	}
}

// TestWebSocketPayloadStructure valida a estrutura do payload
func TestWebSocketPayloadStructure(t *testing.T) {
	// Payload de exemplo
	payload := map[string]interface{}{
		"id":             "device123",
		"lat":            -16.6869,
		"lng":            -49.2648,
		"speed":          45.5,
		"traffic_status": "moderado",
		"timestamp":      time.Now().Unix(),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Erro ao serializar payload: %v", err)
	}

	// Validar tamanho do payload (deve ser leve, < 200 bytes)
	if len(jsonData) > 200 {
		t.Errorf("Payload muito pesado: %d bytes (esperado < 200 bytes)", len(jsonData))
	}

	log.Printf("[WebSocket] Payload size: %d bytes", len(jsonData))
	log.Printf("[WebSocket] Payload: %s", string(jsonData))
}

// BenchmarkWebSocketLatency benchmark de latência WebSocket
func BenchmarkWebSocketLatency(b *testing.B) {
	wsURL := "ws://localhost:8081/api/v1/telemetry/ws"

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		b.Skipf("Servidor não está rodando: %v", err)
		return
	}
	defer conn.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, _, err := conn.ReadMessage()
		if err != nil {
			b.Fatalf("Erro ao ler mensagem: %v", err)
		}
		latency := time.Since(start)
		b.ReportMetric(float64(latency.Milliseconds()), "ms/op")
	}
}

// Helper function to trigger a GPS update for testing
func sendGPSUpdate() error {
	payload := map[string]interface{}{
		"device_id":      "test-device-ws",
		"recorded_at":    time.Now(),
		"latitude":       -16.6869,
		"longitude":      -49.2648,
		"speed":          45.5,
		"heading":        180.0,
		"accuracy":       15.0,
		"transport_mode": "bus",
		"route_id":       "801",
		"battery_level":  75,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post("http://localhost:8081/api/v1/telemetry/gps", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
