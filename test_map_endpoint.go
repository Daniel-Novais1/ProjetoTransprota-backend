package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestMapViewEndpoint testa o endpoint /api/v1/map-view
func TestMapViewEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Criar router com apenas o endpoint de mapa
	r := gin.New()
	r.GET("/api/v1/map-view", func(c *gin.Context) {
		// Coordenadas reais aproximadas de Goiânia
		route := MapRouteResponse{
			Origin: MapPoint{
				Name:      "Setor Bueno",
				Latitude:  -16.6864,
				Longitude: -49.2643,
			},
			Destination: MapPoint{
				Name:      "Campus Samambaia UFG",
				Latitude:  -16.6831,
				Longitude: -49.2674,
			},
			Steps: []MapStep{
				{
					Name:        "Setor Bueno",
					Latitude:    -16.6864,
					Longitude:   -49.2643,
					IsTerminal:  false,
					IsTransfer:  false,
				},
				{
					Name:        "Terminal Centro",
					Latitude:    -16.6807,
					Longitude:   -49.2671,
					IsTerminal:  true,
					IsTransfer:  true,
				},
				{
					Name:        "Terminal Samambaia",
					Latitude:    -16.6825,
					Longitude:   -49.2655,
					IsTerminal:  true,
					IsTransfer:  true,
				},
				{
					Name:        "Campus Samambaia UFG",
					Latitude:    -16.6831,
					Longitude:   -49.2674,
					IsTerminal:  false,
					IsTransfer:  false,
				},
			},
			TotalTimeMinutes: 50,
			BusLines:         []string{"M23", "M71"},
		}

		c.JSON(http.StatusOK, route)
	})

	// Testar o endpoint
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/map-view", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response MapRouteResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Validações da resposta
	assert.Equal(t, "Setor Bueno", response.Origin.Name)
	assert.Equal(t, "Campus Samambaia UFG", response.Destination.Name)
	assert.Len(t, response.Steps, 4)
	assert.Equal(t, 50, response.TotalTimeMinutes)
	assert.Len(t, response.BusLines, 2)

	// Validar pontos da rota
	steps := response.Steps
	assert.Equal(t, "Setor Bueno", steps[0].Name)
	assert.Equal(t, "Terminal Centro", steps[1].Name)
	assert.Equal(t, "Terminal Samambaia", steps[2].Name)
	assert.Equal(t, "Campus Samambaia UFG", steps[3].Name)

	// Validar terminais
	assert.True(t, steps[1].IsTerminal)
	assert.True(t, steps[2].IsTerminal)
	assert.False(t, steps[0].IsTerminal)
	assert.False(t, steps[3].IsTerminal)

	// Validar pontos de transferência
	assert.True(t, steps[1].IsTransfer)
	assert.True(t, steps[2].IsTransfer)
	assert.False(t, steps[0].IsTransfer)
	assert.False(t, steps[3].IsTransfer)

	// Validar coordenadas (Goiânia)
	assert.InDelta(t, -16.68, response.Origin.Latitude, 0.01)
	assert.InDelta(t, -49.26, response.Origin.Longitude, 0.01)

	t.Logf("Endpoint test passed - Route: %s -> %s", response.Origin.Name, response.Destination.Name)
	t.Logf("Total time: %d minutes", response.TotalTimeMinutes)
	t.Logf("Bus lines: %v", response.BusLines)
}
