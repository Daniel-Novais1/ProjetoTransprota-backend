package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestAuthMiddlewareSecurity testa vetores de ataque no AuthMiddleware
func TestAuthMiddlewareSecurity(t *testing.T) {
	// Configura chave válida
	os.Setenv("API_SECRET_KEY", "valid-secret-key-123")
	defer os.Unsetenv("API_SECRET_KEY")

	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		apiKey         string
		expectedStatus int
		description    string
	}{
		{
			name:           "Chave válida",
			apiKey:         "valid-secret-key-123",
			expectedStatus: http.StatusOK,
			description:    "Caminho feliz - chave correta",
		},
		{
			name:           "Chave vazia",
			apiKey:         "",
			expectedStatus: http.StatusUnauthorized,
			description:    "Vetor: ausência de autenticação",
		},
		{
			name:           "Chave nula",
			apiKey:         "null",
			expectedStatus: http.StatusUnauthorized,
			description:    "Vetor: string nulo",
		},
		{
			name:           "Chave com espaços",
			apiKey:         " valid-secret-key-123 ",
			expectedStatus: http.StatusUnauthorized,
			description:    "Vetor: manipulação de espaços",
		},
		{
			name:           "Chave com whitespace",
			apiKey:         "\tvalid-secret-key-123\n",
			expectedStatus: http.StatusUnauthorized,
			description:    "Vetor: caracteres de controle",
		},
		{
			name:           "Chave parcial",
			apiKey:         "valid-secret",
			expectedStatus: http.StatusUnauthorized,
			description:    "Vetor: ataque de força bruta parcial",
		},
		{
			name:           "Chave com SQL injection",
			apiKey:         "valid-secret-key-123'; DROP TABLE users; --",
			expectedStatus: http.StatusUnauthorized,
			description:    "Vetor: SQL injection attempt",
		},
		{
			name:           "Chave com XSS",
			apiKey:         "<script>alert('xss')</script>",
			expectedStatus: http.StatusUnauthorized,
			description:    "Vetor: XSS attempt",
		},
		{
			name:           "Chave com path traversal",
			apiKey:         "../../../etc/passwd",
			expectedStatus: http.StatusUnauthorized,
			description:    "Vetor: path traversal attempt",
		},
		{
			name:           "Chave muito longa",
			apiKey:         strings.Repeat("A", 10000),
			expectedStatus: http.StatusUnauthorized,
			description:    "Vetor: buffer overflow attempt",
		},
		{
			name:           "Chave com caracteres especiais",
			apiKey:         "!@#$%^&*()_+-=[]{}|;':\",./<>?",
			expectedStatus: http.StatusUnauthorized,
			description:    "Vetor: caracteres especiais",
		},
		{
			name:           "Chave unicode",
			apiKey:         "valid-secret-key-123\u0000\u0001\u0002",
			expectedStatus: http.StatusUnauthorized,
			description:    "Vetor: caracteres de controle unicode",
		},
		{
			name:           "Chave case sensitivity",
			apiKey:         "VALID-SECRET-KEY-123",
			expectedStatus: http.StatusUnauthorized,
			description:    "Vetor: case sensitivity attack",
		},
		{
			name:           "Chave com similaridade visual",
			apiKey:         "va1id-secret-key-123", // l em vez de i
			expectedStatus: http.StatusUnauthorized,
			description:    "Vetor: homoglyph attack",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req := httptest.NewRequest(http.MethodGet, "/gps/1", nil)
			if tc.apiKey != "null" {
				req.Header.Set("X-API-Key", tc.apiKey)
			}
			c.Request = req

			// Executa middleware
			AuthMiddleware()(c)

			// Verificações de segurança
			if tc.expectedStatus == http.StatusOK {
				assert.False(t, c.IsAborted(), "Requisição válida não deveria ser abortada")
				assert.Equal(t, tc.expectedStatus, w.Code, "Status deveria ser OK para chave válida")
			} else {
				assert.True(t, c.IsAborted(), "Requisição inválida deveria ser abortada")
				assert.Equal(t, tc.expectedStatus, w.Code, "Status deveria ser Unauthorized para chave inválida")

				// Verifica se resposta não vazia informações sensíveis
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err, "Resposta deveria ser JSON válido")

				if response != nil {
					assert.Contains(t, response, "error", "Resposta deveria conter mensagem de erro genérica")
					assert.Equal(t, "Acesso negado", response["error"], "Mensagem de erro deveria ser genérica")
					assert.NotContains(t, response, "debug", "Resposta não deveria conter informações de debug")
				}
			}
		})
	}
}

// TestAuthMiddlewareTimingAttack testa vulnerabilidade a timing attacks
func TestAuthMiddlewareTimingAttack(t *testing.T) {
	os.Setenv("API_SECRET_KEY", "very-long-secret-key-for-timing-attack-testing")
	defer os.Unsetenv("API_SECRET_KEY")

	gin.SetMode(gin.TestMode)

	// Testa se o tempo de resposta é consistente entre chaves válidas e inválidas
	validKey := "very-long-secret-key-for-timing-attack-testing"
	invalidKey := "completely-different-key-of-same-length"

	// Executa múltiplas vezes para média
	validTimes := make([]time.Duration, 10)
	invalidTimes := make([]time.Duration, 10)

	for i := 0; i < 10; i++ {
		// Testa chave válida
		start := time.Now()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest(http.MethodGet, "/gps/1", nil)
		req.Header.Set("X-API-Key", validKey)
		c.Request = req
		AuthMiddleware()(c)
		validTimes[i] = time.Since(start)

		// Testa chave inválida
		start = time.Now()
		w = httptest.NewRecorder()
		c, _ = gin.CreateTestContext(w)
		req = httptest.NewRequest(http.MethodGet, "/gps/1", nil)
		req.Header.Set("X-API-Key", invalidKey)
		c.Request = req
		AuthMiddleware()(c)
		invalidTimes[i] = time.Since(start)
	}

	// Calcula médias
	avgValid := calculateAverage(validTimes)
	avgInvalid := calculateAverage(invalidTimes)

	// A diferença não deveria ser muito grande (proteção contra timing attack)
	if avgValid > 0 {
		diff := float64(avgInvalid-avgValid) / float64(avgValid) * 100
		t.Logf("Tempo médio válido: %v, inválido: %v, diferença: %.2f%%", avgValid, avgInvalid, diff)

		// Se a diferença for muito grande, pode indicar vulnerabilidade
		assert.Less(t, diff, 200.0, "Diferença de tempo muito grande pode indicar vulnerabilidade a timing attack")
	} else {
		t.Log("Tempos muito curtos para análise de timing attack")
	}
}

// TestAuthMiddlewareConcurrentAccess testa segurança em acesso concorrente
func TestAuthMiddlewareConcurrentAccess(t *testing.T) {
	os.Setenv("API_SECRET_KEY", "concurrent-test-key")
	defer os.Unsetenv("API_SECRET_KEY")

	gin.SetMode(gin.TestMode)

	const numGoroutines = 100
	results := make(chan int, numGoroutines)

	// Dispara múltiplas requisições concorrentes
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest(http.MethodGet, "/gps/1", nil)

			// Metade com chave válida, metade com inválida
			if id%2 == 0 {
				req.Header.Set("X-API-Key", "concurrent-test-key")
			} else {
				req.Header.Set("X-API-Key", "invalid-key")
			}

			c.Request = req
			AuthMiddleware()(c)
			results <- w.Code
		}(i)
	}

	// Coleta resultados
	validCount := 0
	invalidCount := 0
	for i := 0; i < numGoroutines; i++ {
		status := <-results
		switch status {
		case http.StatusOK:
			validCount++
		case http.StatusUnauthorized:
			invalidCount++
		}
	}

	// Verifica se todos os resultados foram processados corretamente
	assert.Equal(t, numGoroutines/2, validCount, "Metade das requisições deveriam ser autorizadas")
	assert.Equal(t, numGoroutines/2, invalidCount, "Metade das requisições deveriam ser rejeitadas")
}

// TestAuthMiddlewareHeaderManipulation testa manipulação de headers
func TestAuthMiddlewareHeaderManipulation(t *testing.T) {
	os.Setenv("API_SECRET_KEY", "test-key")
	defer os.Unsetenv("API_SECRET_KEY")

	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		headerName     string
		headerValue    string
		expectedStatus int
	}{
		{
			name:           "Header correto",
			headerName:     "X-API-Key",
			headerValue:    "test-key",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Header com case diferente",
			headerName:     "x-api-key",
			headerValue:    "test-key",
			expectedStatus: http.StatusOK, // HTTP headers are case-insensitive
		},
		{
			name:           "Header malformado",
			headerName:     "X-API-Key ",
			headerValue:    "test-key",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Múltiplos headers",
			headerName:     "X-API-Key",
			headerValue:    "test-key",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest(http.MethodGet, "/gps/1", nil)
			req.Header.Set(tc.headerName, tc.headerValue)

			// Adiciona headers adicionais para teste
			if tc.name == "Múltiplos headers" {
				req.Header.Add("X-API-Key", "invalid-key")
			}

			c.Request = req
			AuthMiddleware()(c)

			assert.Equal(t, tc.expectedStatus, w.Code, "Status incorreto para %s", tc.name)
		})
	}
}

// Função auxiliar para calcular média de durações
func calculateAverage(durations []time.Duration) time.Duration {
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}
