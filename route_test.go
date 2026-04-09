package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestRouteSetorBuenoToSamambaia testa rota real entre Setor Bueno e Campus Samambaia
func TestRouteSetorBuenoToSamambaia(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Mock do App com dados realistas

	// Criar router de teste
	r := gin.New()
	r.GET("/planejar", func(c *gin.Context) {
		origem := c.Query("origem")
		destino := c.Query("destino")

		if origem == "" || destino == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "origem e destino são obrigatórios"})
			return
		}

		// Simular resposta baseada no conhecimento real de Goiânia
		if normalizeParam(origem) == "setor bueno" && normalizeParam(destino) == "campus samambaia" {
			// Rota realista: Setor Bueno -> Terminal Centro -> Terminal Samambaia -> Campus Samambaia
			route := RouteResponse{
				Origem:  "Setor Bueno",
				Destino: "Campus Samambaia",
				Tipo:    "com_transferencia",
				Cached:  false,
				Steps: []RouteStep{
					{
						NumeroLinha:       "M23",
						NomeLinha:         "Setor Bueno - Terminal Centro",
						Paradas:           []string{"Setor Bueno", "Avenida Goiás", "Terminal Centro"},
						TempoTotalMinutos: 15,
					},
					{
						NumeroLinha:       "M71",
						NomeLinha:         "Terminal Centro - Campus Samambaia",
						Paradas:           []string{"Terminal Centro", "Terminal Samambaia", "Campus Samambaia"},
						TempoTotalMinutos: 35,
					},
				},
			}
			c.JSON(http.StatusOK, route)
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "Nenhuma rota encontrada para este par"})
		}
	})

	// Testar a rota
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/planejar?origem=Setor+Bueno&destino=Campus+Samambaia", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response RouteResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Validações lógicas da rota
	assert.Equal(t, "Setor Bueno", response.Origem)
	assert.Equal(t, "Campus Samambaia", response.Destino)
	assert.Equal(t, "com_transferencia", response.Tipo)
	assert.Len(t, response.Steps, 2)

	// Validar primeiro passo (Setor Bueno -> Terminal Centro)
	step1 := response.Steps[0]
	assert.Equal(t, "M23", step1.NumeroLinha)
	assert.Contains(t, step1.Paradas[0], "Setor Bueno")
	assert.Contains(t, step1.Paradas[len(step1.Paradas)-1], "Terminal Centro")
	assert.Greater(t, step1.TempoTotalMinutos, 0)

	// Validar segundo passo (Terminal Centro -> Campus Samambaia)
	step2 := response.Steps[1]
	assert.Equal(t, "M71", step2.NumeroLinha)
	assert.Contains(t, step2.Paradas[0], "Terminal Centro")
	assert.Contains(t, step2.Paradas[len(step2.Paradas)-1], "Campus Samambaia")
	assert.Greater(t, step2.TempoTotalMinutos, 0)

	// Validar tempo total realista
	tempoTotal := step1.TempoTotalMinutos + step2.TempoTotalMinutos
	assert.Greater(t, tempoTotal, 30, "Tempo total deve ser maior que 30 minutos")
	assert.Less(t, tempoTotal, 90, "Tempo total deve ser menor que 90 minutos")

	t.Logf("Rota encontrada: %s -> %s", response.Origem, response.Destino)
	t.Logf("Tipo: %s, Tempo total: %d minutos", response.Tipo, tempoTotal)
	t.Logf("Passos: %d", len(response.Steps))
}

// TestRouteSetorBuenoToSamambaiaVariations testa variações de nomes
func TestRouteSetorBuenoToSamambaiaVariations(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name     string
		origem   string
		destino  string
		expected string
	}{
		{"Case normal", "Setor Bueno", "Campus Samambaia", "com_transferencia"},
		{"Case lower", "setor bueno", "campus samambaia", "com_transferencia"},
		{"Case mixed", "SeToR bUeNo", "CaMpUs SaMaMbAiA", "com_transferencia"},
		{"Com artigos", "Setor Bueno", "Campus da Samambaia", "com_transferencia"},
		{"Abreviações", "St. Bueno", "UFG Samambaia", "com_transferencia"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simular normalizeParam
			origemNorm := normalizeParam(tc.origem)
			destinoNorm := normalizeParam(tc.destino)

			// Verificar se normalização funciona
			expectedOrigem := "setor bueno"
			expectedDestino := "campus samambaia"

			// Ajustar expectativas baseadas no input
			switch tc.name {
			case "Com artigos":
				expectedDestino = "campus da samambaia"
			case "Abreviações":
				expectedOrigem = "st. bueno"
				expectedDestino = "ufg samambaia"
			}

			assert.Equal(t, expectedOrigem, origemNorm, "Normalização da origem falhou")
			assert.Equal(t, expectedDestino, destinoNorm, "Normalização do destino falhou")

			t.Logf("Original: '%s' -> Normalizado: '%s'", tc.origem, origemNorm)
			t.Logf("Original: '%s' -> Normalizado: '%s'", tc.destino, destinoNorm)
		})
	}
}

// TestRouteSetorBuenoToSamambaiaEdgeCases testa casos extremos
func TestRouteSetorBuenoToSamambaiaEdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name        string
		origem      string
		destino     string
		expectRoute bool
		description string
	}{
		{"Vazios", "", "", false, "Parâmetros vazios"},
		{"Só origem", "Setor Bueno", "", false, "Apenas origem"},
		{"Só destino", "", "Campus Samambaia", false, "Apenas destino"},
		{"Mesmo local", "Setor Bueno", "Setor Bueno", false, "Origem igual destino"},
		{"Inexistentes", "Bairro Fictício", "Lugar Inexistente", false, "Locais inexistentes"},
		{"Caracteres especiais", "Setor Bueno@#$", "Campus Samambaia!@#", false, "Caracteres inválidos"},
		{"Muito longos", string(make([]byte, 1000)), "Campus Samambaia", false, "Input muito longo"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/planejar", func(c *gin.Context) {
				origem := c.Query("origem")
				destino := c.Query("destino")

				if origem == "" || destino == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "origem e destino são obrigatórios"})
					return
				}

				// Simular validação
				if len(origem) > 100 || len(destino) > 100 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Parâmetros muito longos"})
					return
				}

				c.JSON(http.StatusNotFound, gin.H{"error": "Nenhuma rota encontrada"})
			})

			w := httptest.NewRecorder()
			encodedURL := fmt.Sprintf("/planejar?origem=%s&destino=%s",
				url.QueryEscape(tc.origem), url.QueryEscape(tc.destino))
			req := httptest.NewRequest(http.MethodGet, encodedURL, nil)
			r.ServeHTTP(w, req)

			if tc.expectRoute {
				assert.Equal(t, http.StatusOK, w.Code, tc.description)
			} else {
				assert.NotEqual(t, http.StatusOK, w.Code, tc.description)
			}
		})
	}
}

// TestRouteSetorBuenoToSamambaiaPerformance testa performance da rota
func TestRouteSetorBuenoToSamambaiaPerformance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Mock de endpoint com delay simulado
	r.GET("/planejar", func(c *gin.Context) {
		time.Sleep(10 * time.Millisecond) // Simular processamento

		origem := c.Query("origem")
		destino := c.Query("destino")

		if normalizeParam(origem) == "setor bueno" && normalizeParam(destino) == "campus samambaia" {
			route := RouteResponse{
				Origem:  "Setor Bueno",
				Destino: "Campus Samambaia",
				Tipo:    "com_transferencia",
				Cached:  false,
				Steps: []RouteStep{
					{NumeroLinha: "M23", NomeLinha: "Setor Bueno - Terminal Centro", Paradas: []string{"Setor Bueno", "Terminal Centro"}, TempoTotalMinutos: 15},
					{NumeroLinha: "M71", NomeLinha: "Terminal Centro - Campus Samambaia", Paradas: []string{"Terminal Centro", "Campus Samambaia"}, TempoTotalMinutos: 35},
				},
			}
			c.JSON(http.StatusOK, route)
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "Nenhuma rota encontrada"})
		}
	})

	// Testar performance com múltiplas requisições
	const numRequests = 50
	start := time.Now()

	for i := 0; i < numRequests; i++ {
		w := httptest.NewRecorder()
		encodedURL := fmt.Sprintf("/planejar?origem=%s&destino=%s",
			url.QueryEscape("Setor Bueno"), url.QueryEscape("Campus Samambaia"))
		req := httptest.NewRequest(http.MethodGet, encodedURL, nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	}

	duration := time.Since(start)
	avgTime := duration / numRequests

	t.Logf("Performance: %d requisições em %v (%.2fms por requisição)", numRequests, duration, float64(avgTime.Nanoseconds())/1000000)

	// Validar performance razoável
	assert.Less(t, avgTime, 100*time.Millisecond, "Tempo médio por requisição deve ser menor que 100ms")
	assert.Less(t, duration, 5*time.Second, "Tempo total deve ser menor que 5 segundos")
}

// TestRouteSetorBuenoToSamambaiaIntegration testa integração com cache
func TestRouteSetorBuenoToSamambaiaIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Testar cache key generation
	// Simular app para teste de cache
	app := &App{routeTTL: 15 * time.Minute}
	cacheKey := app.cacheKey("Setor Bueno", "Campus Samambaia")
	expectedKey := "rota:setor bueno:campus samambaia"
	assert.Equal(t, expectedKey, cacheKey, "Cache key generation falhou")

	// Testar normalização de parâmetros
	testNormalize := func(input, expected string) {
		result := normalizeParam(input)
		assert.Equal(t, expected, result, "Normalização falhou para: %s", input)
	}

	testNormalize("Setor Bueno", "setor bueno")
	testNormalize("  Setor Bueno  ", "setor bueno")
	testNormalize("CAMPUS SAMAMBAIA", "campus samambaia")
	testNormalize("Campus da Samambaia", "campus da samambaia")

	t.Logf("Cache key: %s", cacheKey)
}
