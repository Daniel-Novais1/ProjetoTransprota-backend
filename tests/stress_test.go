package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// StressTest simula 50 ônibus enviando coordenadas simultaneamente
type StressTest struct {
	baseURL    string
	authToken  string
	busCount   int
	duration   time.Duration
	results    *StressResults
	mu         sync.Mutex
}

// StressResults armazena resultados do teste
type StressResults struct {
	TotalRequests     int
	SuccessRequests   int
	FailedRequests    int
	AverageLatency    time.Duration
	MaxLatency        time.Duration
	MinLatency        time.Duration
	Errors            []string
	StartTime         time.Time
	EndTime           time.Time
}

// GPSPayload representa o payload de GPS
type GPSPayload struct {
	DeviceID   string  `json:"device_id"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	Speed      float64 `json:"speed"`
	Heading    float64 `json:"heading"`
	Accuracy   float64 `json:"accuracy"`
	RecordedAt string  `json:"recorded_at"`
}

// LoginResponse representa resposta de login
type LoginResponse struct {
	Token string `json:"token"`
}

// NewStressTest cria um novo teste de stress
func NewStressTest(baseURL string) *StressTest {
	return &StressTest{
		baseURL:  baseURL,
		busCount: 50,
		duration: 30 * time.Second,
		results:  &StressResults{},
	}
}

// Run executa o teste de stress
func (s *StressTest) Run() error {
	fmt.Println("🚀 Iniciando Stress Test - 50 ônibus simultâneos")

	// 1. Obter token JWT
	if err := s.login(); err != nil {
		return fmt.Errorf("erro ao fazer login: %w", err)
	}

	s.results.StartTime = time.Now()

	// 2. Criar wait group para goroutines
	var wg sync.WaitGroup

	// 3. Simular 50 ônibus
	for i := 0; i < s.busCount; i++ {
		wg.Add(1)
		go func(busID int) {
			defer wg.Done()
			s.simulateBus(busID)
		}(i)
	}

	// 4. Aguardar todas as goroutines
	wg.Wait()

	s.results.EndTime = time.Now()

	// 5. Calcular métricas
	s.calculateMetrics()

	// 6. Imprimir resultados
	s.printResults()

	return nil
}

// login obtém token JWT
func (s *StressTest) login() error {
	loginPayload := map[string]string{
		"username": "admin",
		"password": "admin123",
	}

	jsonPayload, _ := json.Marshal(loginPayload)
	resp, err := http.Post(s.baseURL+"/api/v1/auth/login", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login falhou com status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var loginResp LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return err
	}

	s.authToken = loginResp.Token
	fmt.Printf("✅ Login realizado com sucesso. Token: %s...\n", s.authToken[:20])
	return nil
}

// simulateBus simula um ônibus enviando coordenadas
func (s *StressTest) simulateBus(busID int) {
	busName := fmt.Sprintf("bus-%03d", busID)

	for i := 0; i < 10; i++ { // 10 pings por ônibus
		lat := -16.6869 + float64(busID)*0.001
		lng := -49.2648 + float64(busID)*0.001

		payload := GPSPayload{
			DeviceID:   busName,
			Lat:        lat,
			Lng:        lng,
			Speed:      40.0 + float64(i),
			Heading:    180.0,
			Accuracy:   10.0,
			RecordedAt: time.Now().Format(time.RFC3339),
		}

		jsonPayload, _ := json.Marshal(payload)

		start := time.Now()
		resp, err := s.sendGPS(jsonPayload)
		latency := time.Since(start)

		s.mu.Lock()
		s.results.TotalRequests++
		if err != nil {
			s.results.FailedRequests++
			s.results.Errors = append(s.results.Errors, fmt.Sprintf("%s: %v", busName, err))
		} else {
			s.results.SuccessRequests++
			if s.results.MinLatency == 0 || latency < s.results.MinLatency {
				s.results.MinLatency = latency
			}
			if latency > s.results.MaxLatency {
				s.results.MaxLatency = latency
			}
		}
		s.mu.Unlock()

		if resp != nil {
			resp.Body.Close()
		}

		time.Sleep(100 * time.Millisecond) // 100ms entre pings
	}
}

// sendGPS envia dados GPS para o servidor
func (s *StressTest) sendGPS(payload []byte) (*http.Response, error) {
	req, _ := http.NewRequest("POST", s.baseURL+"/api/v1/telemetry/gps", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.authToken)

	client := &http.Client{Timeout: 10 * time.Second}
	return client.Do(req)
}

// calculateMetrics calcula métricas do teste
func (s *StressTest) calculateMetrics() {
	if s.results.TotalRequests > 0 {
		s.results.AverageLatency = s.results.EndTime.Sub(s.results.StartTime) / time.Duration(s.results.TotalRequests)
	}
}

// printResults imprime resultados do teste
func (s *StressTest) printResults() {
	fmt.Println("\n📊 RESULTADOS DO STRESS TEST")
	fmt.Println("================================")
	fmt.Printf("Duração: %v\n", s.results.EndTime.Sub(s.results.StartTime))
	fmt.Printf("Total de Requisições: %d\n", s.results.TotalRequests)
	fmt.Printf("Sucesso: %d (%.1f%%)\n", s.results.SuccessRequests, float64(s.results.SuccessRequests)/float64(s.results.TotalRequests)*100)
	fmt.Printf("Falhas: %d (%.1f%%)\n", s.results.FailedRequests, float64(s.results.FailedRequests)/float64(s.results.TotalRequests)*100)
	fmt.Printf("Latência Média: %v\n", s.results.AverageLatency)
	fmt.Printf("Latência Mínima: %v\n", s.results.MinLatency)
	fmt.Printf("Latência Máxima: %v\n", s.results.MaxLatency)

	if len(s.results.Errors) > 0 {
		fmt.Println("\n❌ ERROS:")
		for _, err := range s.results.Errors {
			fmt.Printf("  - %s\n", err)
		}
	}

	fmt.Println("================================")
}
