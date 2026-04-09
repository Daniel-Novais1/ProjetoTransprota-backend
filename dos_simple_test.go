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

// TestDoSAttackSimple testa ataque DoS simples
func TestDoSAttackSimple(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Criar router com rate limiting
	r := gin.New()
	r.Use(RateLimitMiddleware())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Testar ataque de alta frequência
	const numRequests = 100
	const concurrentWorkers = 10

	var successCount int64
	var rateLimitCount int64

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
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("DoS Attack Results:")
	t.Logf("  Total requests: %d", numRequests)
	t.Logf("  Success: %d", successCount)
	t.Logf("  Rate limited: %d", rateLimitCount)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Requests/sec: %.2f", float64(numRequests)/duration.Seconds())

	// Validações de segurança
	assert.Greater(t, rateLimitCount, int64(0), "Rate limiting deveria bloquear algumas requisições")
	assert.Less(t, rateLimitCount, int64(numRequests), "Não deveria bloquear todas as requisições")
	assert.Less(t, duration, 30*time.Second, "Teste deveria completar em tempo razoável")
}

// TestRateLimitEffectiveness testa eficácia do rate limiting
func TestRateLimitEffectiveness(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(RateLimitMiddleware())

	requestCount := 0
	r.GET("/test", func(c *gin.Context) {
		requestCount++
		c.JSON(http.StatusOK, gin.H{"request": requestCount})
	})

	// Testar requisições rápidas do mesmo IP
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
}

// TestLargePayloadAttack testa ataque com payloads grandes
func TestLargePayloadAttack(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(RateLimitMiddleware())

	r.POST("/process", func(c *gin.Context) {
		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"processed": true})
	})

	// Testar com JSONs muito grandes
	const numRequests = 20
	var successCount, errorCount int64

	// Criar JSON grande (1KB cada)
	largeJSON := `{"data":"` + fmt.Sprintf("%1000s", "A") + `"}`

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

	t.Logf("Large Payload Attack: Success=%d, Errors=%d", successCount, errorCount)

	// A maioria deveria ser limitada ou rejeitada
	assert.Greater(t, errorCount, int64(numRequests/2), "Payloads grandes deveriam ser rejeitados")
}
