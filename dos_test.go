package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestDoSAttackSimulation testa resiliência da API contra ataques de negação de serviço
func TestDoSAttackSimulation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Configurar rate limiting para teste

	// Setup router com middleware de rate limiting
	r := gin.New()
	r.Use(RateLimitMiddleware())

	// Endpoints para teste
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/planejar", func(c *gin.Context) {
		origem := c.Query("origem")
		destino := c.Query("destino")
		c.JSON(http.StatusOK, gin.H{"origem": origem, "destino": destino})
	})

	// Testar ataque de alta frequência
	t.Run("High Frequency Attack", func(t *testing.T) {
		const numRequests = 1000
		const concurrentWorkers = 50

		var successCount int64
		var rateLimitCount int64
		var errorCount int64

		start := time.Now()

		var wg sync.WaitGroup
		requestsPerWorker := numRequests / concurrentWorkers

		for i := 0; i < concurrentWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < requestsPerWorker; j++ {
					w := httptest.NewRecorder()
					req := httptest.NewRequest(http.MethodGet, "/health", nil)
					r.ServeHTTP(w, req)

					switch w.Code {
					case http.StatusOK:
						atomic.AddInt64(&successCount, 1)
					case http.StatusTooManyRequests:
						atomic.AddInt64(&rateLimitCount, 1)
					default:
						atomic.AddInt64(&errorCount, 1)
					}
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		t.Logf("High Frequency Attack Results:")
		t.Logf("  Total requests: %d", numRequests)
		t.Logf("  Success: %d", successCount)
		t.Logf("  Rate limited: %d", rateLimitCount)
		t.Logf("  Errors: %d", errorCount)
		t.Logf("  Duration: %v", duration)
		t.Logf("  Requests/sec: %.2f", float64(numRequests)/duration.Seconds())

		// Validações de segurança
		assert.Greater(t, rateLimitCount, int64(0), "Rate limiting deveria bloquear algumas requisições")
		assert.Less(t, rateLimitCount, int64(numRequests), "Não deveria bloquear todas as requisições")
		assert.Less(t, duration, 30*time.Second, "Teste deveria completar em tempo razoável")
	})

	// Testar ataque de payload grande
	t.Run("Large Payload Attack", func(t *testing.T) {
		const numRequests = 100

		var successCount int64
		var errorCount int64

		// Criar payload grande (1MB)
		largePayload := strings.Repeat("A", 1024*1024)

		for i := 0; i < numRequests; i++ {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/planejar?origem=%s&destino=Campus+Samambaia", largePayload), nil)
			r.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&errorCount, 1)
			}
		}

		t.Logf("Large Payload Attack Results:")
		t.Logf("  Total requests: %d", numRequests)
		t.Logf("  Success: %d", successCount)
		t.Logf("  Errors: %d", errorCount)

		// Validações - a maioria deveria ser rejeitada
		assert.Greater(t, errorCount, int64(numRequests/2), "Payloads grandes deveriam ser rejeitados")
	})

	// Testar ataque de conexões simultâneas
	t.Run("Concurrent Connection Attack", func(t *testing.T) {
		const numConnections = 200
		const requestsPerConnection = 10

		var successCount int64
		var rateLimitCount int64
		var errorCount int64

		start := time.Now()

		var wg sync.WaitGroup

		for i := 0; i < numConnections; i++ {
			wg.Add(1)
			go func(connID int) {
				defer wg.Done()

				for j := 0; j < requestsPerConnection; j++ {
					w := httptest.NewRecorder()
					req := httptest.NewRequest(http.MethodGet, "/planejar?origem=Setor+Bueno&destino=Campus+Samambaia", nil)
					r.ServeHTTP(w, req)

					switch w.Code {
					case http.StatusOK:
						atomic.AddInt64(&successCount, 1)
					case http.StatusTooManyRequests:
						atomic.AddInt64(&rateLimitCount, 1)
					default:
						atomic.AddInt64(&errorCount, 1)
					}
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		t.Logf("Concurrent Connection Attack Results:")
		t.Logf("  Connections: %d", numConnections)
		t.Logf("  Requests per connection: %d", requestsPerConnection)
		t.Logf("  Total requests: %d", numConnections*requestsPerConnection)
		t.Logf("  Success: %d", successCount)
		t.Logf("  Rate limited: %d", rateLimitCount)
		t.Logf("  Errors: %d", errorCount)
		t.Logf("  Duration: %v", duration)

		// Validações
		assert.Greater(t, rateLimitCount, int64(0), "Conexões simultâneas deveriam trigger rate limiting")
		assert.Less(t, duration, 60*time.Second, "Teste deveria completar em tempo razoável")
	})
}

// TestRateLimitMiddlewareEffectiveness testa eficácia do rate limiting
func TestCircuitBreakerEffectiveness(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(RateLimitMiddleware())

	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"request": 1})
	})

	// Testar requisições rápidas do mesmo IP
	t.Run("Same IP Rapid Requests", func(t *testing.T) {
		const rapidRequests = 20
		var successCount, rateLimitCount int

		for i := 0; i < rapidRequests; i++ {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			// Mesmo IP para simular ataque
			req.RemoteAddr = "192.168.1.100:12345"
			r.ServeHTTP(w, req)

			switch w.Code {
			case http.StatusOK:
				successCount++
			case http.StatusTooManyRequests:
				rateLimitCount++
			}

			// Pequeno delay para simular requisições rápidas
			time.Sleep(1 * time.Millisecond)
		}

		t.Logf("Same IP Results: Success=%d, RateLimited=%d", successCount, rateLimitCount)

		// Rate limiting deveria bloquear algumas requisições
		assert.Greater(t, rateLimitCount, 0, "Rate limiting deveria bloquear requisições rápidas")
		assert.Less(t, successCount, rapidRequests, "Não deveria permitir todas as requisições")
	})

	// Testar requisições de IPs diferentes
	t.Run("Different IPs", func(t *testing.T) {
		const numIPs = 10
		const requestsPerIP = 5

		var totalSuccess, totalRateLimit int

		for i := 0; i < numIPs; i++ {
			ip := fmt.Sprintf("192.168.1.%d", i+1)

			for j := 0; j < requestsPerIP; j++ {
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.RemoteAddr = ip + ":12345"
				r.ServeHTTP(w, req)

				switch w.Code {
				case http.StatusOK:
					totalSuccess++
				case http.StatusTooManyRequests:
					totalRateLimit++
				}
			}
		}

		t.Logf("Different IPs Results: TotalSuccess=%d, TotalRateLimited=%d", totalSuccess, totalRateLimit)

		// IPs diferentes deveriam ter melhor taxa de sucesso
		expectedTotal := numIPs * requestsPerIP
		assert.Greater(t, totalSuccess, expectedTotal/2, "IPs diferentes deveriam ter mais sucesso")
	})
}

// TestDoSAttackMemoryExhaustion testa proteção contra exaustão de memória
func TestDoSAttackMemoryExhaustion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(RateLimitMiddleware())

	// Endpoint que processa dados
	r.POST("/process", func(c *gin.Context) {
		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"processed": true})
	})

	// Testar com JSONs muito grandes
	t.Run("Large JSON Attack", func(t *testing.T) {
		const numRequests = 50

		var successCount, errorCount int64

		// Criar JSON grande (10KB cada)
		largeJSON := `{"data":"` + strings.Repeat("A", 10000) + `"}`

		for i := 0; i < numRequests; i++ {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/process", strings.NewReader(largeJSON))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&errorCount, 1)
			}
		}

		t.Logf("Large JSON Attack: Success=%d, Errors=%d", successCount, errorCount)

		// A maioria deveria ser limitada ou rejeitada
		assert.Greater(t, errorCount, int64(numRequests/2), "JSONs grandes deveriam ser rejeitados")
	})
}

// TestDoSAttackSlowloris testa proteção contra ataques Slowloris
func TestDoSAttackSlowloris(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(RateLimitMiddleware())

	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Simular requisições lentas
	t.Run("Slow Request Attack", func(t *testing.T) {
		const numSlowRequests = 20

		var successCount, errorCount int64

		for i := 0; i < numSlowRequests; i++ {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			// Simular requisição lenta
			go func() {
				time.Sleep(100 * time.Millisecond)
				r.ServeHTTP(w, req)
			}()

			// Esperar um pouco antes da próxima
			time.Sleep(50 * time.Millisecond)

			if w.Code == http.StatusOK {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&errorCount, 1)
			}
		}

		time.Sleep(2 * time.Second) // Esperar completar

		t.Logf("Slow Request Attack: Success=%d, Errors=%d", successCount, errorCount)

		// Requisições lentas deveriam ser tratadas adequadamente
		assert.Greater(t, successCount, int64(0), "Algumas requisições deveriam ter sucesso")
	})
}

// TestDoSAttackResourceExhaustion testa exaustão de recursos do sistema
func TestDoSAttackResourceExhaustion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(RateLimitMiddleware())

	// Endpoint intensivo
	r.GET("/intensive", func(c *gin.Context) {
		// Simular processamento intensivo
		time.Sleep(10 * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"processed": true})
	})

	// Testar carga pesada
	t.Run("Heavy Load Attack", func(t *testing.T) {
		const numWorkers = 100
		const requestsPerWorker = 10

		var successCount, rateLimitCount, errorCount int64
		start := time.Now()

		var wg sync.WaitGroup

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < requestsPerWorker; j++ {
					w := httptest.NewRecorder()
					req := httptest.NewRequest(http.MethodGet, "/intensive", nil)
					r.ServeHTTP(w, req)

					switch w.Code {
					case http.StatusOK:
						atomic.AddInt64(&successCount, 1)
					case http.StatusTooManyRequests:
						atomic.AddInt64(&rateLimitCount, 1)
					default:
						atomic.AddInt64(&errorCount, 1)
					}
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		t.Logf("Heavy Load Attack Results:")
		t.Logf("  Total requests: %d", numWorkers*requestsPerWorker)
		t.Logf("  Success: %d", successCount)
		t.Logf("  Rate limited: %d", rateLimitCount)
		t.Logf("  Errors: %d", errorCount)
		t.Logf("  Duration: %v", duration)

		// Validações de resiliência
		assert.Greater(t, rateLimitCount, int64(0), "Rate limiting deveria ativar sob carga pesada")
		assert.Less(t, duration, 60*time.Second, "Sistema deveria aguentar carga pesada")
		assert.Greater(t, successCount, int64(0), "Algumas requisições deveriam ter sucesso")
	})
}
