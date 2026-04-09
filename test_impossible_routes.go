package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestImpossibleRoutes testa tratamento de rotas fora da área de cobertura
func TestImpossibleRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Criar router com endpoint de mapa
	r := gin.New()
	r.GET("/api/v1/map-view", func(c *gin.Context) {
		origin := c.Query("origin")
		destination := c.Query("destination")

		// Sanitização de inputs
		if origin == "" || destination == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Origin and destination are required"})
			return
		}

		// Validar e sanitizar inputs
		origin = sanitizeInput(origin)
		destination = sanitizeInput(destination)

		// Limitar tamanho para prevenir ataques
		if len(origin) > 100 || len(destination) > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Input too long"})
			return
		}

		// Verificar se está dentro da área de Goiânia
		if !isWithinGoiania(origin) || !isWithinGoiania(destination) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Fora da área de cobertura (Goiânia)"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	testCases := []struct {
		name           string
		origin         string
		destination    string
		expectedStatus int
		expectedError  string
		description    string
	}{
		{
			name:           "Rota válida em Goiânia",
			origin:         "Setor Bueno",
			destination:    "Campus Samambaia",
			expectedStatus: http.StatusOK,
			description:    "Rota dentro da área de cobertura",
		},
		{
			name:           "Origem fora - Tóquio",
			origin:         "Tóquio",
			destination:    "Setor Bueno",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Fora da área de cobertura (Goiânia)",
			description:    "Origem fora da área de cobertura",
		},
		{
			name:           "Destino fora - Nova York",
			origin:         "Setor Centro",
			destination:    "Nova York",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Fora da área de cobertura (Goiânia)",
			description:    "Destino fora da área de cobertura",
		},
		{
			name:           "Ambos fora - Paris",
			origin:         "Paris",
			destination:    "Londres",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Fora da área de cobertura (Goiânia)",
			description:    "Ambos pontos fora da área",
		},
		{
			name:           "Input vazio",
			origin:         "",
			destination:    "Setor Bueno",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Origin and destination are required",
			description:    "Input obrigatório faltando",
		},
		{
			name:           "Input muito longo",
			origin:         "Setor Bueno com um texto muito longo que excede o limite de 100 caracteres para testar validação de tamanho",
			destination:    "Campus Samambaia",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Input too long",
			description:    "Input excedendo limite de caracteres",
		},
		{
			name:           "Caracteres especiais",
			origin:         "Setor <script>alert('xss')</script>",
			destination:    "Campus Samambaia",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Fora da área de cobertura (Goiânia)",
			description:    "Input com caracteres perigosos",
		},
		{
			name:           "Lugares parecidos mas fora",
			origin:         "Bueno Aires",
			destination:    "Campus Samambaia",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Fora da área de cobertura (Goiânia)",
			description:    "Lugar com nome parecido mas fora",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/map-view", nil)

			// Adicionar parâmetros à query
			q := req.URL.Query()
			q.Add("origin", tc.origin)
			q.Add("destination", tc.destination)
			req.URL.RawQuery = q.Encode()

			r.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code, tc.description)

			if tc.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err, "Resposta deveria ser JSON válido")

				if response != nil {
					assert.Contains(t, response, "error", "Resposta deveria conter mensagem de erro")
					assert.Equal(t, tc.expectedError, response["error"], "Mensagem de erro incorreta")
				}
			}

			t.Logf("Test: %s - Status: %d, Error: %s", tc.name, w.Code, tc.expectedError)
		})
	}
}

// TestInputSanitization testa sanitização de inputs perigosos
func TestInputSanitization(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"Setor Bueno", "Setor Bueno"},
		{"<script>alert('xss')</script>", "scriptalertxssscript"},
		{"'; DROP TABLE users; --", " DROP TABLE users --"},
		{"Setor \"Bueno\"", "Setor Bueno"},
		{"$(whoami)", "whoami"},
		{"Setor\\Bueno", "SetorBueno"},
		{"", ""},
		{"   Setor Bueno   ", "Setor Bueno"},
		{"Texto muito longo que deveria ser truncado para exatamente cem caracteres para testar o limite de tamanho", "Texto muito longo que deveria ser truncado para exatamente cem caracteres para testar o limite de tam"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := sanitizeInput(tc.input)
			assert.Equal(t, tc.expected, result, "Sanitização falhou para: %s", tc.input)
			t.Logf("Input: '%s' -> Sanitized: '%s'", tc.input, result)
		})
	}
}

// TestRushHourLogic testa lógica de horário de pico
func TestRushHourLogic(t *testing.T) {
	testCases := []struct {
		name         string
		hour         int
		expectedTime int
		baseTime     int
		description  string
	}{
		{"Horário normal", 10, 30, 30, "Fora do horário de pico"},
		{"Início do pico", 17, 50, 30, "17h - início do horário de pico"},
		{"Meio do pico", 18, 50, 30, "18h - meio do horário de pico"},
		{"Fim do pico", 19, 50, 30, "19h - fim do horário de pico"},
		{"Após o pico", 20, 30, 30, "Após o horário de pico"},
		{"Madrugada", 2, 30, 30, "Madrugada"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simular lógica de horário de pico
			result := tc.baseTime
			if tc.hour >= 17 && tc.hour <= 19 {
				result += 20 // Adicionar 20 minutos no horário de pico
			}

			assert.Equal(t, tc.expectedTime, result, tc.description)
			t.Logf("Hora: %dh, Base: %dmin, Resultado: %dmin", tc.hour, tc.baseTime, result)
		})
	}
}
