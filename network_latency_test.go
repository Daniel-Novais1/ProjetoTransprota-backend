package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// NetworkCondition representa condições de rede simuladas
type NetworkCondition struct {
	Latency     time.Duration `json:"latency"`      // Latência em ms
	Jitter      time.Duration `json:"jitter"`       // Variação da latência
	PacketLoss  float64       `json:"packet_loss"`  // % de perda de pacotes (0-1)
	Bandwidth   int           `json:"bandwidth"`    // KB/s
	Unreliable  bool          `json:"unreliable"`   // Conexão instável
	NetworkType string        `json:"network_type"` // 4G, 5G, WiFi, etc.
}

// LatencyTestResult representa resultado do teste de latência
type LatencyTestResult struct {
	TestID           string           `json:"test_id"`
	NetworkCondition NetworkCondition `json:"network_condition"`
	TotalRequests    int              `json:"total_requests"`
	SuccessfulReqs   int              `json:"successful_requests"`
	FailedReqs       int              `json:"failed_requests"`
	AvgLatency       time.Duration    `json:"avg_latency"`
	MinLatency       time.Duration    `json:"min_latency"`
	MaxLatency       time.Duration    `json:"max_latency"`
	P95Latency       time.Duration    `json:"p95_latency"`
	P99Latency       time.Duration    `json:"p99_latency"`
	ErrorRate        float64          `json:"error_rate"`
	Throughput       float64          `json:"throughput"`       // req/s
	DataTransferred  int64            `json:"data_transferred"` // bytes
	TestDuration     time.Duration    `json:"test_duration"`
	Timestamp        time.Time        `json:"timestamp"`
	Errors           []string         `json:"errors"`
}

// NetworkSimulator simula condições de rede
type NetworkSimulator struct {
	condition NetworkCondition
	rand      *rand.Rand
	mutex     sync.Mutex
}

// NewNetworkSimulator cria um novo simulador de rede
func NewNetworkSimulator(condition NetworkCondition) *NetworkSimulator {
	return &NetworkSimulator{
		condition: condition,
		rand:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SimulateRequest simula uma requisição com condições de rede
func (ns *NetworkSimulator) SimulateRequest(reqFunc func() (*http.Response, error)) (*http.Response, error) {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()

	// Simular perda de pacotes
	if ns.condition.PacketLoss > 0 && ns.rand.Float64() < ns.condition.PacketLoss {
		return nil, fmt.Errorf("simulated packet loss")
	}

	// Simular latência com jitter
	baseLatency := ns.condition.Latency
	if ns.condition.Jitter > 0 {
		jitterRange := int(ns.condition.Jitter.Milliseconds())
		jitter := time.Duration(ns.rand.Intn(jitterRange)) * time.Millisecond
		baseLatency += jitter
	}

	// Simular conexão instável
	if ns.condition.Unreliable && ns.rand.Float64() < 0.1 {
		return nil, fmt.Errorf("simulated network instability")
	}

	// Aplicar latência antes da requisição
	time.Sleep(baseLatency)

	// Executar requisição
	resp, err := reqFunc()
	if err != nil {
		return nil, err
	}

	// Simular bandwidth limit (simplificado)
	if ns.condition.Bandwidth > 0 && resp.ContentLength > 0 {
		// Calcular tempo de transferência baseado no bandwidth
		transferTime := time.Duration(float64(resp.ContentLength)/float64(ns.condition.Bandwidth*1024)) * time.Second
		time.Sleep(transferTime)
	}

	return resp, nil
}

// LatencyTester realiza testes de latência
type LatencyTester struct {
	simulator *NetworkSimulator
	baseURL   string
}

// NewLatencyTester cria um novo testador de latência
func NewLatencyTester(baseURL string, condition NetworkCondition) *LatencyTester {
	return &LatencyTester{
		simulator: NewNetworkSimulator(condition),
		baseURL:   baseURL,
	}
}

// RunLatencyTest executa teste completo de latência
func (lt *LatencyTester) RunLatencyTest(concurrentUsers, requestsPerUser int) *LatencyTestResult {
	testID := fmt.Sprintf("test_%d", time.Now().UnixNano())
	startTime := time.Now()

	var wg sync.WaitGroup
	var mutex sync.Mutex

	result := &LatencyTestResult{
		TestID:           testID,
		NetworkCondition: lt.simulator.condition,
		TotalRequests:    concurrentUsers * requestsPerUser,
		SuccessfulReqs:   0,
		FailedReqs:       0,
		MinLatency:       time.Hour, // Valor inicial alto
		MaxLatency:       0,
		Timestamp:        startTime,
		Errors:           []string{},
	}

	var latencies []time.Duration
	var totalBytes int64

	// Simular usuários concorrentes
	for i := 0; i < concurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			for j := 0; j < requestsPerUser; j++ {
				requestStart := time.Now()

				// Simular requisição com condições de rede
				resp, err := lt.simulator.SimulateRequest(func() (*http.Response, error) {
					return lt.makeTestRequest(userID, j)
				})

				requestLatency := time.Since(requestStart)

				mutex.Lock()
				if err != nil {
					result.FailedReqs++
					result.Errors = append(result.Errors, fmt.Sprintf("User %d, Req %d: %v", userID, j, err))
				} else {
					result.SuccessfulReqs++
					latencies = append(latencies, requestLatency)

					// Atualizar estatísticas
					if requestLatency < result.MinLatency {
						result.MinLatency = requestLatency
					}
					if requestLatency > result.MaxLatency {
						result.MaxLatency = requestLatency
					}

					// Contar bytes transferidos
					if resp != nil {
						totalBytes += resp.ContentLength
						resp.Body.Close()
					}
				}
				mutex.Unlock()

				// Pequeno delay entre requisições do mesmo usuário
				time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	result.TestDuration = time.Since(startTime)
	result.DataTransferred = totalBytes

	// Calcular estatísticas finais
	if len(latencies) > 0 {
		// Média
		var totalLatency time.Duration
		for _, latency := range latencies {
			totalLatency += latency
		}
		result.AvgLatency = totalLatency / time.Duration(len(latencies))

		// Percentis
		sortDurations(latencies)
		result.P95Latency = latencies[int(float64(len(latencies))*0.95)]
		result.P99Latency = latencies[int(float64(len(latencies))*0.99)]
	}

	// Taxa de erro
	if result.TotalRequests > 0 {
		result.ErrorRate = float64(result.FailedReqs) / float64(result.TotalRequests)
	}

	// Throughput
	if result.TestDuration > 0 {
		result.Throughput = float64(result.SuccessfulReqs) / result.TestDuration.Seconds()
	}

	return result
}

// makeTestRequest faz uma requisição de teste
func (lt *LatencyTester) makeTestRequest(userID, requestID int) (*http.Response, error) {
	// Criar requisição para endpoint de teste
	url := fmt.Sprintf("%s/api/v1/walkability?distance=1.5&user_id=%d&req_id=%d",
		lt.baseURL, userID, requestID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "TranspRota-Latency-Test/1.0")
	req.Header.Set("X-Test-ID", fmt.Sprintf("user_%d_req_%d", userID, requestID))

	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}

// sortDurations ordena slice de durações
func sortDurations(durations []time.Duration) {
	for i := 0; i < len(durations); i++ {
		for j := i + 1; j < len(durations); j++ {
			if durations[i] > durations[j] {
				durations[i], durations[j] = durations[j], durations[i]
			}
		}
	}
}

// GetNetworkPreset retorna configurações predefinidas de rede
func GetNetworkPreset(networkType string) NetworkCondition {
	switch networkType {
	case "4g_poor":
		return NetworkCondition{
			Latency:     300 * time.Millisecond,
			Jitter:      100 * time.Millisecond,
			PacketLoss:  0.05, // 5%
			Bandwidth:   1000, // 1 MB/s
			Unreliable:  true,
			NetworkType: "4G (Poor)",
		}
	case "4g_good":
		return NetworkCondition{
			Latency:     50 * time.Millisecond,
			Jitter:      20 * time.Millisecond,
			PacketLoss:  0.01, // 1%
			Bandwidth:   5000, // 5 MB/s
			Unreliable:  false,
			NetworkType: "4G (Good)",
		}
	case "5g":
		return NetworkCondition{
			Latency:     10 * time.Millisecond,
			Jitter:      5 * time.Millisecond,
			PacketLoss:  0.001, // 0.1%
			Bandwidth:   20000, // 20 MB/s
			Unreliable:  false,
			NetworkType: "5G",
		}
	case "wifi_slow":
		return NetworkCondition{
			Latency:     150 * time.Millisecond,
			Jitter:      50 * time.Millisecond,
			PacketLoss:  0.02, // 2%
			Bandwidth:   2000, // 2 MB/s
			Unreliable:  false,
			NetworkType: "WiFi (Slow)",
		}
	case "edge_cases":
		return NetworkCondition{
			Latency:     1000 * time.Millisecond, // 1 segundo
			Jitter:      500 * time.Millisecond,
			PacketLoss:  0.15, // 15%
			Bandwidth:   500,  // 500 KB/s
			Unreliable:  true,
			NetworkType: "Edge Cases",
		}
	default:
		return NetworkCondition{
			Latency:     25 * time.Millisecond,
			Jitter:      10 * time.Millisecond,
			PacketLoss:  0.0,
			Bandwidth:   10000, // 10 MB/s
			Unreliable:  false,
			NetworkType: "Default",
		}
	}
}

// RunComprehensiveTest executa testes em múltiplas condições de rede
func RunComprehensiveTest(baseURL string) map[string]*LatencyTestResult {
	presets := []string{"4g_poor", "4g_good", "5g", "wifi_slow", "edge_cases"}
	results := make(map[string]*LatencyTestResult)

	for _, preset := range presets {
		condition := GetNetworkPreset(preset)
		tester := NewLatencyTester(baseURL, condition)

		fmt.Printf("Executando teste de latência para %s...\n", condition.NetworkType)

		// Teste com 10 usuários concorrentes, 5 requisições cada
		result := tester.RunLatencyTest(10, 5)
		results[preset] = result

		fmt.Printf("Resultado %s: %.1f%% sucesso, latência média: %v\n",
			condition.NetworkType,
			(1-result.ErrorRate)*100,
			result.AvgLatency)
	}

	return results
}

// GenerateLatencyReport gera relatório completo dos testes
func GenerateLatencyReport(results map[string]*LatencyTestResult) map[string]interface{} {
	report := map[string]interface{}{
		"test_summary": map[string]interface{}{
			"total_tests":    len(results),
			"test_timestamp": time.Now(),
			"base_url":       "http://localhost:8080",
		},
		"results":  results,
		"analysis": map[string]interface{}{},
	}

	// Análise comparativa
	var avgSuccessRate float64
	var avgLatency time.Duration
	var worstNetwork string
	var worstLatency time.Duration
	var bestNetwork string
	var bestLatency time.Duration = time.Hour

	for preset, result := range results {
		successRate := (1 - result.ErrorRate) * 100
		avgSuccessRate += successRate
		avgLatency += result.AvgLatency

		if result.AvgLatency > worstLatency {
			worstLatency = result.AvgLatency
			worstNetwork = preset
		}

		if result.AvgLatency < bestLatency {
			bestLatency = result.AvgLatency
			bestNetwork = preset
		}
	}

	if len(results) > 0 {
		avgSuccessRate /= float64(len(results))
		avgLatency /= time.Duration(len(results))
	}

	report["analysis"] = map[string]interface{}{
		"avg_success_rate": avgSuccessRate,
		"avg_latency":      avgLatency,
		"worst_performer":  worstNetwork,
		"worst_latency":    worstLatency,
		"best_performer":   bestNetwork,
		"best_latency":     bestLatency,
		"recommendations":  generateRecommendations(results),
	}

	return report
}

// generateRecommendations gera recomendações baseadas nos resultados
func generateRecommendations(results map[string]*LatencyTestResult) []string {
	recommendations := []string{}

	// Verificar performance em diferentes condições
	for preset, result := range results {
		if result.ErrorRate > 0.1 { // > 10% de erro
			recommendations = append(recommendations,
				fmt.Sprintf("Otimizar para %s: Taxa de erro muito alta (%.1f%%)",
					preset, result.ErrorRate*100))
		}

		if result.AvgLatency > 500*time.Millisecond {
			recommendations = append(recommendations,
				fmt.Sprintf("Implementar cache para %s: Latência muito alta (%v)",
					preset, result.AvgLatency))
		}
	}

	// Recomendações gerais
	if len(recommendations) == 0 {
		recommendations = append(recommendations,
			"Performance excelente em todas as condições de rede testadas")
		recommendations = append(recommendations,
			"Sistema pronto para uso em redes móveis 4G/5G")
	} else {
		recommendations = append(recommendations,
			"Implementar retry automático para requisições falhadas")
		recommendations = append(recommendations,
			"Adicionar indicadores de carregamento para longas esperas")
	}

	return recommendations
}

// setupLatencyTestRoutes configura rotas para testes de latência
func setupLatencyTestRoutes(r *gin.Engine) {
	// GET /api/v1/test/latency - Executar teste de latência
	r.GET("/api/v1/test/latency", func(c *gin.Context) {
		networkType := c.DefaultQuery("network", "4g_good")
		concurrentUsers := 10
		requestsPerUser := 5

		// Permitir override dos parâmetros
		if cu := c.Query("concurrent"); cu != "" {
			if parsed, err := parseInt(cu); err == nil && parsed > 0 && parsed <= 50 {
				concurrentUsers = parsed
			}
		}
		if rp := c.Query("requests"); rp != "" {
			if parsed, err := parseInt(rp); err == nil && parsed > 0 && parsed <= 20 {
				requestsPerUser = parsed
			}
		}

		condition := GetNetworkPreset(networkType)
		tester := NewLatencyTester("http://localhost:8080", condition)

		result := tester.RunLatencyTest(concurrentUsers, requestsPerUser)

		c.JSON(http.StatusOK, gin.H{
			"test_result":       result,
			"network_condition": condition,
		})
	})

	// GET /api/v1/test/latency/comprehensive - Teste completo em múltiplas condições
	r.GET("/api/v1/test/latency/comprehensive", func(c *gin.Context) {
		baseURL := "http://localhost:8080"
		results := RunComprehensiveTest(baseURL)
		report := GenerateLatencyReport(results)

		c.JSON(http.StatusOK, report)
	})

	// GET /api/v1/test/latency/simulate - Simular condição de rede específica
	r.POST("/api/v1/test/latency/simulate", func(c *gin.Context) {
		var condition NetworkCondition
		if err := c.ShouldBindJSON(&condition); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
			return
		}

		concurrentUsers := 10
		requestsPerUser := 5

		tester := NewLatencyTester("http://localhost:8080", condition)
		result := tester.RunLatencyTest(concurrentUsers, requestsPerUser)

		c.JSON(http.StatusOK, gin.H{
			"test_result":         result,
			"simulated_condition": condition,
		})
	})

	// GET /api/v1/test/network-presets - Listar presets de rede
	r.GET("/api/v1/test/network-presets", func(c *gin.Context) {
		presets := map[string]NetworkCondition{
			"4g_poor":    GetNetworkPreset("4g_poor"),
			"4g_good":    GetNetworkPreset("4g_good"),
			"5g":         GetNetworkPreset("5g"),
			"wifi_slow":  GetNetworkPreset("wifi_slow"),
			"edge_cases": GetNetworkPreset("edge_cases"),
		}

		c.JSON(http.StatusOK, gin.H{
			"presets": presets,
			"count":   len(presets),
		})
	})
}
