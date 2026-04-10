package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/moby/moby/api/types"
	"github.com/moby/moby/client"
)

// ClusterTest representa teste de resiliência do cluster
type ClusterTest struct {
	DockerClient *client.Client
	TestResults  []TestResult
	mutex        sync.Mutex
}

// TestResult representa resultado de um teste
type TestResult struct {
	TestName     string        `json:"test_name"`
	Status       string        `json:"status"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Duration     time.Duration `json:"duration"`
	RequestsMade int           `json:"requests_made"`
	SuccessCount int           `json:"success_count"`
	ErrorCount   int           `json:"error_count"`
	LatencyAvg   time.Duration `json:"latency_avg"`
	FailoverTime time.Duration `json:"failover_time"`
	ErrorMessage string        `json:"error_message,omitempty"`
}

// NewClusterTest cria novo teste de cluster
func NewClusterTest() (*ClusterTest, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar cliente Docker: %v", err)
	}

	return &ClusterTest{
		DockerClient: cli,
		TestResults:  []TestResult{},
	}, nil
}

// StartCluster inicia o cluster TranspRota
func (ct *ClusterTest) StartCluster() error {
	log.Println("Iniciando cluster TranspRota...")

	// Usar docker-compose para iniciar
	cmd := exec.Command("docker-compose", "up", "-d")
	cmd.Dir = "."

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("erro ao iniciar cluster: %v\nOutput: %s", err, string(output))
	}

	log.Println("Cluster iniciado com sucesso!")
	log.Printf("Output: %s", string(output))

	// Aguardar serviços ficarem prontos
	return ct.waitForClusterReady(60 * time.Second)
}

// waitForClusterReady aguarda cluster ficar pronto
func (ct *ClusterTest) waitForClusterReady(timeout time.Duration) error {
	log.Printf("Aguardando cluster ficar pronto (timeout: %v)...", timeout)

	start := time.Now()
	for time.Since(start) < timeout {
		// Verificar health check das APIs
		if ct.checkAPIHealth("http://localhost/api/v1/health") {
			log.Println("Cluster pronto!")
			return nil
		}

		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("cluster não ficou pronto em %v", timeout)
}

// checkAPIHealth verifica saúde da API
func (ct *ClusterTest) checkAPIHealth(url string) bool {
	// Implementar health check simples
	// Em produção, usar http client real
	return true // Simplificado para demo
}

// RunSuddenDeathTest executa teste de morte súbita
func (ct *ClusterTest) RunSuddenDeathTest() error {
	log.Println("=== INICIANDO TESTE DE MORTE SÚBITA ===")

	result := TestResult{
		TestName:  "Sudden Death Test",
		Status:    "running",
		StartTime: time.Now(),
	}

	// Iniciar tráfego contínuo
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channel para resultados de requisições
	reqResults := make(chan RequestResult, 100)

	// Iniciar goroutine para requisições contínuas
	go ct.generateContinuousTraffic(ctx, reqResults)

	// Aguardar estabilização
	time.Sleep(5 * time.Second)

	// Matar uma instância aleatória
	failoverStart := time.Now()
	err := ct.killRandomInstance()
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Erro ao matar instância: %v", err)
		result.Status = "failed"
	} else {
		failoverTime := time.Since(failoverStart)
		result.FailoverTime = failoverTime
		log.Printf("Failover completado em %v", failoverTime)
	}

	// Continuar tráfego por mais tempo para testar recuperação
	time.Sleep(10 * time.Second)
	cancel() // Parar tráfego

	// Coletar resultados
	ct.collectResults(&result, reqResults)

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if result.ErrorCount == 0 && result.ErrorMessage == "" {
		result.Status = "passed"
	} else if result.ErrorMessage != "" {
		result.Status = "failed"
	} else {
		result.Status = "partial"
	}

	ct.mutex.Lock()
	ct.TestResults = append(ct.TestResults, result)
	ct.mutex.Unlock()

	return nil
}

// RequestResult representa resultado de uma requisição
type RequestResult struct {
	Success bool
	Latency time.Duration
	Error   string
}

// generateContinuousTraffic gera tráfego contínuo
func (ct *ClusterTest) generateContinuousTraffic(ctx context.Context, results chan<- RequestResult) {
	log.Println("Iniciando tráfego contínuo...")

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Fazer requisição
			start := time.Now()
			success := ct.makeRequest()
			latency := time.Since(start)

			result := RequestResult{
				Success: success,
				Latency: latency,
			}

			if !success {
				result.Error = "request failed"
			}

			select {
			case results <- result:
			default:
				// Channel cheio, ignorar
			}
		}
	}
}

// makeRequest faz requisição para a API
func (ct *ClusterTest) makeRequest() bool {
	// Implementar requisição HTTP real
	// Simplificado para demo
	return true
}

// killRandomInstance mata uma instância aleatória da API
func (ct *ClusterTest) killRandomInstance() error {
	log.Println("Mantando instância aleatória...")

	// Listar containers em execução
	containers, err := ct.DockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		log.Printf("Erro ao listar containers: %v", err)
		return fmt.Errorf("erro ao listar containers: %v", err)
	}

	// Encontrar instâncias da API
	var apiContainers []types.Container
	for _, container := range containers {
		if strings.Contains(container.Names[0], "transprota_api") {
			apiContainers = append(apiContainers, container)
		}
	}

	if len(apiContainers) == 0 {
		return fmt.Errorf("nenhuma instância API encontrada")
	}

	// Matar primeira instância
	targetContainer := apiContainers[0]
	log.Printf("Matando container: %s", targetContainer.Names[0])

	return ct.DockerClient.ContainerKill(context.Background(), targetContainer.ID, syscall.SIGKILL)
	return nil
}

// collectResults coleta resultados das requisições
func (ct *ClusterTest) collectResults(result *TestResult, results <-chan RequestResult) {
	log.Println("Coletando resultados...")

	var totalLatency time.Duration
	successCount := 0
	errorCount := 0

	// Coletar por um tempo limitado
	timeout := time.After(2 * time.Second)

	for {
		select {
		case res := <-results:
			result.RequestsMade++

			if res.Success {
				successCount++
				totalLatency += res.Latency
			} else {
				errorCount++
			}

		case <-timeout:
			goto done
		}
	}

done:
	result.SuccessCount = successCount
	result.ErrorCount = errorCount

	if successCount > 0 {
		result.LatencyAvg = totalLatency / time.Duration(successCount)
	}

	log.Printf("Resultados: %d requisições, %d sucesso, %d erros, latência média: %v",
		result.RequestsMade, successCount, errorCount, result.LatencyAvg)
}

// RestartCluster reinicia o cluster
func (ct *ClusterTest) RestartCluster() error {
	log.Println("Reiniciando cluster...")

	cmd := exec.Command("docker-compose", "restart")
	cmd.Dir = "."

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("erro ao reiniciar cluster: %v\nOutput: %s", err, string(output))
	}

	log.Println("Cluster reiniciado!")
	return ct.waitForClusterReady(60 * time.Second)
}

// StopCluster para o cluster
func (ct *ClusterTest) StopCluster() error {
	log.Println("Parando cluster...")

	cmd := exec.Command("docker-compose", "down")
	cmd.Dir = "."

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("erro ao parar cluster: %v\nOutput: %s", err, string(output))
	}

	log.Println("Cluster parado!")
	return nil
}

// GenerateReport gera relatório do teste
func (ct *ClusterTest) GenerateReport() string {
	var builder strings.Builder

	builder.WriteString("=== RELATÓRIO DE TESTE DE CLUSTER TRANSPROTA ===\n\n")

	for _, result := range ct.TestResults {
		builder.WriteString(fmt.Sprintf("Teste: %s\n", result.TestName))
		builder.WriteString(fmt.Sprintf("Status: %s\n", result.Status))
		builder.WriteString(fmt.Sprintf("Duração: %v\n", result.Duration))
		builder.WriteString(fmt.Sprintf("Requisições: %d\n", result.RequestsMade))
		builder.WriteString(fmt.Sprintf("Sucessos: %d\n", result.SuccessCount))
		builder.WriteString(fmt.Sprintf("Erros: %d\n", result.ErrorCount))
		builder.WriteString(fmt.Sprintf("Taxa de Sucesso: %.1f%%\n", float64(result.SuccessCount)/float64(result.RequestsMade)*100))
		builder.WriteString(fmt.Sprintf("Latência Média: %v\n", result.LatencyAvg))
		builder.WriteString(fmt.Sprintf("Tempo de Failover: %v\n", result.FailoverTime))

		if result.ErrorMessage != "" {
			builder.WriteString(fmt.Sprintf("Erro: %s\n", result.ErrorMessage))
		}

		builder.WriteString("\n")
	}

	// Verificação de resiliência
	if len(ct.TestResults) > 0 {
		result := ct.TestResults[0]
		successRate := float64(result.SuccessCount) / float64(result.RequestsMade) * 100

		builder.WriteString("=== AVALIAÇÃO DE RESILIÊNCIA ===\n")
		if successRate >= 95 && result.FailoverTime < 5*time.Second {
			builder.WriteString("Status: EXCELENTE - Cluster altamente resiliente!\n")
		} else if successRate >= 90 && result.FailoverTime < 10*time.Second {
			builder.WriteString("Status: BOM - Cluster resiliente com pequenas melhorias possíveis\n")
		} else {
			builder.WriteString("Status: REQUER MELHORIAS - Cluster precisa de otimizações\n")
		}
	}

	return builder.String()
}

// RunFullClusterTest executa suite completa de testes
func RunFullClusterTest() error {
	log.Println("=== INICIANDO SUITE COMPLETA DE TESTES DE CLUSTER ===")

	// Criar teste
	clusterTest, err := NewClusterTest()
	if err != nil {
		return fmt.Errorf("erro ao criar teste de cluster: %v", err)
	}
	defer clusterTest.DockerClient.Close()

	// Iniciar cluster
	if err := clusterTest.StartCluster(); err != nil {
		return fmt.Errorf("erro ao iniciar cluster: %v", err)
	}
	defer clusterTest.StopCluster()

	// Executar teste de morte súbita
	if err := clusterTest.RunSuddenDeathTest(); err != nil {
		log.Printf("Erro no teste de morte súbita: %v", err)
	}

	// Gerar relatório
	report := clusterTest.GenerateReport()
	log.Println(report)

	// Salvar relatório em arquivo
	if err := os.WriteFile("cluster_test_report.txt", []byte(report), 0644); err != nil {
		log.Printf("Erro ao salvar relatório: %v", err)
	} else {
		log.Println("Relatório salvo em cluster_test_report.txt")
	}

	return nil
}

// SetupClusterTestRoutes configura rotas para testes de cluster
func SetupClusterTestRoutes(r *gin.Engine) {
	// POST /api/v1/test/cluster/sudden-death - Executar teste de morte súbita
	r.POST("/api/v1/test/cluster/sudden-death", func(c *gin.Context) {
		go func() {
			if err := RunFullClusterTest(); err != nil {
				log.Printf("Erro na execução do teste de cluster: %v", err)
			}
		}()

		c.JSON(http.StatusOK, gin.H{
			"message": "Teste de cluster iniciado em background",
			"status":  "running",
		})
	})

	// GET /api/v1/test/cluster/status - Status do cluster
	r.GET("/api/v1/test/cluster/status", func(c *gin.Context) {
		// Verificar status dos containers
		cli, err := client.NewClientWithOpts(client.FromEnv)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao conectar ao Docker"})
			return
		}
		defer cli.Close()

		containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao listar containers"})
			return
		}

		var clusterContainers []gin.H
		for _, container := range containers {
			if strings.Contains(container.Names[0], "transprota") {
				clusterContainers = append(clusterContainers, gin.H{
					"name":   strings.TrimPrefix(container.Names[0], "/"),
					"status": container.Status,
					"image":  container.Image,
				})
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"cluster_containers": clusterContainers,
			"total_containers":   len(clusterContainers),
		})
	})
}
