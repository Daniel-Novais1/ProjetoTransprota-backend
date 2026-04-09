package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// FinalStressTest representa teste de stress final
type FinalStressTest struct {
	TargetURL       string
	Duration        time.Duration
	TargetRPS       int
	Workers         int
	Results         *StressTestResults
	ActiveRequests  int64
	TotalRequests   int64
	SuccessCount    int64
	ErrorCount      int64
	ResponseTimeSum int64
	StartTime       time.Time
	EndTime         time.Time
	Running         bool
	StopChan        chan struct{}
}

// StressTestResults representa resultados do teste
type StressTestResults struct {
	TestDuration      time.Duration   `json:"test_duration"`
	TotalRequests     int64           `json:"total_requests"`
	RequestsPerSecond float64         `json:"requests_per_second"`
	SuccessCount      int64           `json:"success_count"`
	ErrorCount        int64           `json:"error_count"`
	SuccessRate       float64         `json:"success_rate"`
	AvgResponseTime   float64         `json:"avg_response_time_ms"`
	P95ResponseTime   float64         `json:"p95_response_time_ms"`
	P99ResponseTime   float64         `json:"p99_response_time_ms"`
	MinResponseTime   float64         `json:"min_response_time_ms"`
	MaxResponseTime   float64         `json:"max_response_time_ms"`
	ThroughputMBps    float64         `json:"throughput_mbps"`
	ZeroDowntime      bool            `json:"zero_downtime"`
	FailoverEvents    []FailoverEvent `json:"failover_events"`
}

// FailoverEvent representa evento de failover
type FailoverEvent struct {
	Timestamp    time.Time `json:"timestamp"`
	EventType    string    `json:"event_type"`
	Instance     string    `json:"instance"`
	ResponseTime float64   `json:"response_time_ms"`
	Success      bool      `json:"success"`
}

// NewFinalStressTest cria novo teste de stress
func NewFinalStressTest(targetURL string, duration time.Duration, targetRPS int) *FinalStressTest {
	workers := targetRPS / 100 // 100 requests por worker
	if workers < 10 {
		workers = 10
	}
	if workers > 1000 {
		workers = 1000
	}

	return &FinalStressTest{
		TargetURL: targetURL,
		Duration:  duration,
		TargetRPS: targetRPS,
		Workers:   workers,
		Results:   &StressTestResults{},
		StopChan:  make(chan struct{}),
	}
}

// RunStressTest executa teste de stress completo
func (fst *FinalStressTest) RunStressTest() *StressTestResults {
	log.Printf("=== INICIANDO BATERIA DE STRESS FINAL ===")
	log.Printf("Target: %s", fst.TargetURL)
	log.Printf("Duração: %v", fst.Duration)
	log.Printf("Target RPS: %d", fst.TargetRPS)
	log.Printf("Workers: %d", fst.Workers)

	fst.StartTime = time.Now()
	fst.Running = true

	// Canal para resultados
	resultChan := make(chan StressResult, 10000)

	// Iniciar workers
	var wg sync.WaitGroup
	for i := 0; i < fst.Workers; i++ {
		wg.Add(1)
		go fst.worker(i, resultChan, &wg)
	}

	// Coletor de resultados
	go fst.collectResults(resultChan)

	// Simular failover durante o teste
	go fst.simulateFailover()

	// Aguardar duração do teste
	time.Sleep(fst.Duration)

	// Parar teste
	fst.Running = false
	close(fst.StopChan)
	wg.Wait()
	close(resultChan)

	fst.EndTime = time.Now()
	fst.calculateFinalResults()

	log.Printf("=== STRESS TEST CONCLUÍDO ===")
	log.Printf("Total Requests: %d", fst.Results.TotalRequests)
	log.Printf("Success Rate: %.2f%%", fst.Results.SuccessRate)
	log.Printf("Avg Response Time: %.2fms", fst.Results.AvgResponseTime)
	log.Printf("Zero Downtime: %t", fst.Results.ZeroDowntime)

	return fst.Results
}

// StressResult representa resultado de requisição
type StressResult struct {
	Success      bool
	ResponseTime time.Duration
	StatusCode   int
	Error        string
	Timestamp    time.Time
}

// worker executa requisições
func (fst *FinalStressTest) worker(id int, resultChan chan<- StressResult, wg *sync.WaitGroup) {
	defer wg.Done()

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	// Calcular intervalo para atingir RPS target
	interval := time.Duration(float64(time.Second) / float64(fst.TargetRPS/fst.Workers))
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for fst.Running {
		select {
		case <-fst.StopChan:
			return
		case <-ticker.C:
			atomic.AddInt64(&fst.ActiveRequests, 1)
			start := time.Now()

			// Fazer requisição
			resp, err := client.Get(fst.TargetURL)
			responseTime := time.Since(start)
			atomic.AddInt64(&fst.ActiveRequests, -1)

			result := StressResult{
				ResponseTime: responseTime,
				Timestamp:    start,
			}

			if err != nil {
				result.Success = false
				result.Error = err.Error()
				atomic.AddInt64(&fst.ErrorCount, 1)
			} else {
				result.Success = resp.StatusCode == 200
				result.StatusCode = resp.StatusCode
				resp.Body.Close()

				if result.Success {
					atomic.AddInt64(&fst.SuccessCount, 1)
				} else {
					atomic.AddInt64(&fst.ErrorCount, 1)
				}
			}

			atomic.AddInt64(&fst.TotalRequests, 1)
			atomic.AddInt64(&fst.ResponseTimeSum, int64(responseTime.Milliseconds()))

			select {
			case resultChan <- result:
			default:
				// Channel cheio, ignorar
			}
		}
	}
}

// collectResults coleta e processa resultados
func (fst *FinalStressTest) collectResults(resultChan <-chan StressResult) {
	var responseTimes []float64
	var minTime, maxTime float64 = 999999, 0

	for result := range resultChan {
		rt := float64(result.ResponseTime.Milliseconds())
		responseTimes = append(responseTimes, rt)

		if rt < minTime {
			minTime = rt
		}
		if rt > maxTime {
			maxTime = rt
		}

		// Registrar eventos de failover
		if !result.Success || rt > 1000 {
			event := FailoverEvent{
				Timestamp:    result.Timestamp,
				EventType:    "slow_response",
				ResponseTime: rt,
				Success:      result.Success,
			}
			fst.Results.FailoverEvents = append(fst.Results.FailoverEvents, event)
		}
	}

	// Calcular percentis
	if len(responseTimes) > 0 {
		fst.Results.MinResponseTime = minTime
		fst.Results.MaxResponseTime = maxTime
		fst.Results.P95ResponseTime = calculatePercentile(responseTimes, 0.95)
		fst.Results.P99ResponseTime = calculatePercentile(responseTimes, 0.99)
	}
}

// simulateFailover simula eventos de failover
func (fst *FinalStressTest) simulateFailover() {
	// Simular failover a cada 90 segundos (para teste de 5 minutos)
	failoverInterval := 90 * time.Second
	ticker := time.NewTicker(failoverInterval)
	defer ticker.Stop()

	failoverCount := 0

	for fst.Running {
		select {
		case <-fst.StopChan:
			return
		case <-ticker.C:
			failoverCount++
			log.Printf("Simulando failover #%d", failoverCount)

			// Registrar evento de failover
			event := FailoverEvent{
				Timestamp: time.Now(),
				EventType: "simulated_failover",
				Instance:  fmt.Sprintf("api-%d", (failoverCount%2)+1),
				Success:   true,
			}
			fst.Results.FailoverEvents = append(fst.Results.FailoverEvents, event)

			// Aguardar recuperação
			time.Sleep(5 * time.Second)
		}
	}
}

// calculateFinalResults calcula resultados finais
func (fst *FinalStressTest) calculateFinalResults() {
	fst.Results.TestDuration = fst.EndTime.Sub(fst.StartTime)
	fst.Results.TotalRequests = fst.TotalRequests
	fst.Results.SuccessCount = fst.SuccessCount
	fst.Results.ErrorCount = fst.ErrorCount

	// Calcular RPS real
	if fst.Results.TestDuration.Seconds() > 0 {
		fst.Results.RequestsPerSecond = float64(fst.TotalRequests) / fst.Results.TestDuration.Seconds()
	}

	// Calcular taxa de sucesso
	if fst.TotalRequests > 0 {
		fst.Results.SuccessRate = float64(fst.SuccessCount) / float64(fst.TotalRequests) * 100
	}

	// Calcular tempo médio de resposta
	if fst.TotalRequests > 0 {
		fst.Results.AvgResponseTime = float64(fst.ResponseTimeSum) / float64(fst.TotalRequests)
	}

	// Calcular throughput (estimado)
	avgResponseSize := 1024 // bytes (estimado)
	totalBytes := fst.TotalRequests * int64(avgResponseSize)
	fst.Results.ThroughputMBps = float64(totalBytes) / fst.Results.TestDuration.Seconds() / (1024 * 1024)

	// Verificar zero downtime
	fst.Results.ZeroDowntime = fst.Results.SuccessRate >= 99.5 && len(fst.Results.FailoverEvents) == 0
}

// calculatePercentile calcula percentil
func calculatePercentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Simple sort (para demonstração)
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[i] > values[j] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}

	index := int(float64(len(values)) * percentile)
	if index >= len(values) {
		index = len(values) - 1
	}

	return values[index]
}

// RunFinalStressTestSuite executa suite completa de testes
func RunFinalStressTestSuite() error {
	log.Println("=== BATERIA DE STRESS FINAL - TRANSPROTA CLUSTER ===")

	// Teste 1: Carga normal (145K req/s por 5 minutos)
	test1 := NewFinalStressTest("http://localhost:8080/api/v1/health", 5*time.Minute, 145000)
	results1 := test1.RunStressTest()

	// Teste 2: Load com failover (100K req/s por 3 minutos)
	test2 := NewFinalStressTest("http://localhost:8080/api/v1/walkability?distance=1.5", 3*time.Minute, 100000)
	results2 := test2.RunStressTest()

	// Teste 3: Stress extremo (200K req/s por 2 minutos)
	test3 := NewFinalStressTest("http://localhost:8080/api/v1/metrics", 2*time.Minute, 200000)
	results3 := test3.RunStressTest()

	// Gerar relatório final
	report := generateFinalStressReport(results1, results2, results3)

	// Salvar relatório
	if err := os.WriteFile("final_stress_test_report.txt", []byte(report), 0644); err != nil {
		return fmt.Errorf("erro ao salvar relatório: %v", err)
	}

	log.Println("Relatório final salvo em final_stress_test_report.txt")
	return nil
}

// generateFinalStressReport gera relatório final
func generateFinalStressReport(results1, results2, results3 *StressTestResults) string {
	var report strings.Builder

	report.WriteString("=== RELATÓRIO FINAL DE STRESS TEST - TRANSPROTA CLUSTER ===\n\n")
	report.WriteString(fmt.Sprintf("Data: %s\n\n", time.Now().Format("02/01/2006 15:04:05")))

	// Teste 1
	report.WriteString("## TESTE 1: CARGA NORMAL (145K req/s)\n")
	report.WriteString(fmt.Sprintf("- Duração: %v\n", results1.TestDuration))
	report.WriteString(fmt.Sprintf("- Total Requests: %d\n", results1.TotalRequests))
	report.WriteString(fmt.Sprintf("- RPS Real: %.0f\n", results1.RequestsPerSecond))
	report.WriteString(fmt.Sprintf("- Success Rate: %.2f%%\n", results1.SuccessRate))
	report.WriteString(fmt.Sprintf("- Avg Response Time: %.2fms\n", results1.AvgResponseTime))
	report.WriteString(fmt.Sprintf("- P95 Response Time: %.2fms\n", results1.P95ResponseTime))
	report.WriteString(fmt.Sprintf("- Zero Downtime: %t\n\n", results1.ZeroDowntime))

	// Teste 2
	report.WriteString("## TESTE 2: LOAD COM FAILOVER (100K req/s)\n")
	report.WriteString(fmt.Sprintf("- Duração: %v\n", results2.TestDuration))
	report.WriteString(fmt.Sprintf("- Total Requests: %d\n", results2.TotalRequests))
	report.WriteString(fmt.Sprintf("- RPS Real: %.0f\n", results2.RequestsPerSecond))
	report.WriteString(fmt.Sprintf("- Success Rate: %.2f%%\n", results2.SuccessRate))
	report.WriteString(fmt.Sprintf("- Avg Response Time: %.2fms\n", results2.AvgResponseTime))
	report.WriteString(fmt.Sprintf("- P95 Response Time: %.2fms\n", results2.P95ResponseTime))
	report.WriteString(fmt.Sprintf("- Failover Events: %d\n", len(results2.FailoverEvents)))
	report.WriteString(fmt.Sprintf("- Zero Downtime: %t\n\n", results2.ZeroDowntime))

	// Teste 3
	report.WriteString("## TESTE 3: STRESS EXTREMO (200K req/s)\n")
	report.WriteString(fmt.Sprintf("- Duração: %v\n", results3.TestDuration))
	report.WriteString(fmt.Sprintf("- Total Requests: %d\n", results3.TotalRequests))
	report.WriteString(fmt.Sprintf("- RPS Real: %.0f\n", results3.RequestsPerSecond))
	report.WriteString(fmt.Sprintf("- Success Rate: %.2f%%\n", results3.SuccessRate))
	report.WriteString(fmt.Sprintf("- Avg Response Time: %.2fms\n", results3.AvgResponseTime))
	report.WriteString(fmt.Sprintf("- P95 Response Time: %.2fms\n", results3.P95ResponseTime))
	report.WriteString(fmt.Sprintf("- Zero Downtime: %t\n\n", results3.ZeroDowntime))

	// Resumo
	report.WriteString("## RESUMO FINAL\n\n")
	avgSuccessRate := (results1.SuccessRate + results2.SuccessRate + results3.SuccessRate) / 3
	avgResponseTime := (results1.AvgResponseTime + results2.AvgResponseTime + results3.AvgResponseTime) / 3
	zeroDowntimeAll := results1.ZeroDowntime && results2.ZeroDowntime && results3.ZeroDowntime

	report.WriteString(fmt.Sprintf("- Success Rate Médio: %.2f%%\n", avgSuccessRate))
	report.WriteString(fmt.Sprintf("- Response Time Médio: %.2fms\n", avgResponseTime))
	report.WriteString(fmt.Sprintf("- Zero Downtime em Todos: %t\n", zeroDowntimeAll))

	// Veredito
	report.WriteString("\n## VEREDITO FINAL\n\n")
	if avgSuccessRate >= 99.0 && avgResponseTime <= 50 && zeroDowntimeAll {
		report.WriteString("STATUS: EXCELENTE - Cluster superou todas as expectativas!\n")
		report.WriteString("O TranspRota Cluster está PRONTO PARA PRODUÇÃO com margem de segurança.\n")
	} else if avgSuccessRate >= 95.0 && avgResponseTime <= 100 {
		report.WriteString("STATUS: BOM - Cluster atende aos requisitos mínimos.\n")
		report.WriteString("Recomendadas otimizações para produção.\n")
	} else {
		report.WriteString("STATUS: REQUER MELHORIAS - Cluster não está pronto para produção.\n")
		report.WriteString("Necessárias correções críticas antes do go-live.\n")
	}

	return report.String()
}

// setupFinalStressTestRoutes configura rotas de teste final
func setupFinalStressTestRoutes(r *gin.Engine) {
	// POST /api/v1/test/stress/final - Executar bateria completa
	r.POST("/api/v1/test/stress/final", func(c *gin.Context) {
		go func() {
			if err := RunFinalStressTestSuite(); err != nil {
				log.Printf("Erro na execução do stress test: %v", err)
			}
		}()

		c.JSON(http.StatusOK, gin.H{
			"message":  "Bateria de stress final iniciada em background",
			"status":   "running",
			"duration": "10 minutos estimados",
		})
	})

	// GET /api/v1/test/stress/status - Status do teste
	r.GET("/api/v1/test/stress/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Stress test framework disponível",
			"endpoints": []string{
				"POST /api/v1/test/stress/final - Executar bateria completa",
				"GET /api/v1/test/stress/status - Status do sistema",
			},
		})
	})
}
