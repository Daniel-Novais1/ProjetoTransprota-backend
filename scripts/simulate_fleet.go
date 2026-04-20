package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"time"
)

// Bus representa um ônibus na frota
type Bus struct {
	ID         string
	CurrentLat float64
	CurrentLng float64
	TargetLat  float64
	TargetLng  float64
	RouteID    string
	Occupancy  int
	StepLat    float64
	StepLng    float64
}

// BusUpdatePayload representa o payload para o endpoint /api/v1/bus-update
type BusUpdatePayload struct {
	UserID     string  `json:"user_id"`
	DeviceHash string  `json:"device_hash"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	RouteID    string  `json:"route_id"`
	Speed      float64 `json:"speed"`
	Heading    float64 `json:"heading"`
	IsOnBus    bool    `json:"is_on_bus"`
	Occupancy  string  `json:"occupancy"`
	TerminalID string  `json:"terminal_id"`
	Timestamp  string  `json:"timestamp"`
}

// Coordenadas aproximadas de Goiânia (retângulo: Lat -16.6, Lon -49.2)
const (
	// Terminal Padre Pelágio (ponto de partida)
	startLat = -16.6869
	startLng = -49.2648

	// Terminal Praça da Bíblia (destino)
	endLat = -16.7000
	endLng = -49.2500

	// Limites de Goiânia para validação
	goianiaLatMin = -16.8
	goianiaLatMax = -16.5
	goianiaLngMin = -49.4
	goianiaLngMax = -49.1

	// Configurações
	apiURL     = "http://localhost:8080/api/v1/bus-locations"
	steps      = 50 // Número de passos para completar a rota
	interval   = 500 * time.Millisecond
	totalBuses = 5
)

func main() {
	log.Println("🚌 Iniciando simulação de frota em Goiânia...")
	log.Printf("📍 Rota: Terminal Padre Pelágio → Terminal Praça da Bíblia")
	log.Printf("🔄 Intervalo: %v", interval)
	log.Printf("📡 API: %s", apiURL)

	// Inicializar seed aleatório
	rand.Seed(time.Now().UnixNano())

	// Criar frota de ônibus
	fleet := make([]Bus, totalBuses)
	for i := 0; i < totalBuses; i++ {
		busID := fmt.Sprintf("EIXO-%03d", i+1)

		// Calcular passo incremental
		stepLat := (endLat - startLat) / float64(steps)
		stepLng := (endLng - startLng) / float64(steps)

		fleet[i] = Bus{
			ID:         busID,
			CurrentLat: startLat,
			CurrentLng: startLng,
			TargetLat:  endLat,
			TargetLng:  endLng,
			RouteID:    "006",
			Occupancy:  rand.Intn(5) + 1,
			StepLat:    stepLat,
			StepLng:    stepLng,
		}

		log.Printf("✅ Ônibus %s inicializado em (%.6f, %.6f)", busID, startLat, startLng)
	}

	// Loop infinito de simulação
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		<-ticker.C

		for i := range fleet {
			bus := &fleet[i]

			// Atualizar posição
			bus.CurrentLat += bus.StepLat
			bus.CurrentLng += bus.StepLng

			// Validar se coordenadas estão dentro do retângulo de Goiânia
			if !isWithinGoiania(bus.CurrentLat, bus.CurrentLng) {
				log.Printf("⚠️ Coordenadas fora de Goiânia: (%.6f, %.6f) - reiniciando", bus.CurrentLat, bus.CurrentLng)
				bus.CurrentLat = startLat
				bus.CurrentLng = startLng
			}

			// Alternar ocupação aleatoriamente
			if rand.Float32() < 0.3 {
				bus.Occupancy = rand.Intn(5) + 1
			}

			// Calcular heading baseado na direção
			heading := calculateHeading(bus.CurrentLat, bus.CurrentLng, bus.CurrentLat+bus.StepLat, bus.CurrentLng+bus.StepLng)
			speed := 30.0 + float64(rand.Intn(20)) // 30-50 km/h

			// Se chegou ao destino, reiniciar
			if bus.CurrentLat <= endLat || bus.CurrentLng >= endLng {
				bus.CurrentLat = startLat
				bus.CurrentLng = startLng
				log.Printf("🔄 Ônibus %s reiniciou a rota", bus.ID)
			}

			// Enviar telemetria
			occupancyLevels := []string{"empty", "low", "medium", "high", "full"}
			payload := BusUpdatePayload{
				UserID:     "sim-user-" + bus.ID,
				DeviceHash: bus.ID,
				Lat:        bus.CurrentLat,
				Lng:        bus.CurrentLng,
				RouteID:    bus.RouteID,
				Speed:      speed,
				Heading:    heading,
				IsOnBus:    true,
				Occupancy:  occupancyLevels[bus.Occupancy-1],
				TerminalID: "terminal-padre-pelagio",
				Timestamp:  time.Now().Format(time.RFC3339),
			}

			if err := sendTelemetry(payload); err != nil {
				log.Printf("❌ Erro ao enviar telemetria do ônibus %s: %v", bus.ID, err)
			} else {
				log.Printf("📤 %s → (%.6f, %.6f) | Ocupação: %d | Velocidade: %.1f km/h",
					bus.ID, bus.CurrentLat, bus.CurrentLng, bus.Occupancy, speed)
			}
		}
	}
}

// sendTelemetry envia dados de telemetria para o backend
func sendTelemetry(payload BusUpdatePayload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erro ao marshal JSON: %w", err)
	}

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("erro ao fazer POST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status inesperado: %d", resp.StatusCode)
	}

	return nil
}

// isWithinGoiania verifica se coordenadas estão dentro do retângulo de Goiânia
func isWithinGoiania(lat, lng float64) bool {
	return lat >= goianiaLatMin && lat <= goianiaLatMax &&
		lng >= goianiaLngMin && lng <= goianiaLngMax
}

// calculateHeading calcula o heading (direção) entre dois pontos
func calculateHeading(lat1, lng1, lat2, lng2 float64) float64 {
	dLon := (lng2 - lng1) * 3.14159265359 / 180.0
	lat1Rad := lat1 * 3.14159265359 / 180.0
	lat2Rad := lat2 * 3.14159265359 / 180.0

	y := math.Sin(dLon) * math.Cos(lat2Rad)
	x := math.Cos(lat1Rad)*math.Sin(lat2Rad) - math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(dLon)

	heading := math.Atan2(y, x) * 180.0 / 3.14159265359
	heading = math.Mod(heading+360, 360)

	return heading
}
