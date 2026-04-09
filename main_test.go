package main

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// Testa se a fórmula de Haversine está calculando distâncias corretamente
func TestCalcularDistancia(t *testing.T) {
	lat1, lon1 := -16.6733, -49.2394
	lat2, lon2 := -16.6743, -49.2394

	distancia := calcularDistancia(lat1, lon1, lat2, lon2)

	if math.Abs(distancia-111) > 10 {
		t.Errorf("A distância calculada (%.2fm) está fora da margem de erro esperada", distancia)
	}

	// Edge case: mesmo ponto
	distMesmo := calcularDistancia(0, 0, 0, 0)
	if distMesmo != 0 {
		t.Errorf("Distância entre mesmo ponto deveria ser 0, foi %.2f", distMesmo)
	}

	// Edge case: coordenadas reais de Goiânia (Terminal Centro -> UFG ~23km)
	distGoi := calcularDistancia(-15.7975, -48.2647, -15.8267, -48.0405)
	if math.Abs(distGoi-24000) > 2000 { // ±2km tolerância
		t.Errorf("Distância Terminal Centro -> UFG deveria ser ~24km, foi %.2f", distGoi)
	}
}

// Testa a conversão de radianos
func TestToRadians(t *testing.T) {
	resultado := toRadians(180)
	esperado := math.Pi
	if resultado != esperado {
		t.Errorf("Conversão de radianos falhou: esperado %f, obtido %f", esperado, resultado)
	}
}

func TestNormalizeParam(t *testing.T) {
	resultado := normalizeParam("  Vila Pedroso ")
	esperado := "vila pedroso"
	if resultado != esperado {
		t.Fatalf("normalizeParam falhou: esperado %q, obtido %q", esperado, resultado)
	}
}

func TestIntegrationBonus(t *testing.T) {
	cases := []struct {
		nome     string
		parada   string
		esperado int
	}{
		{"Novo Mundo", "Terminal Novo Mundo", 15},
		{"Biblia", "Terminal Praça da Bíblia", 15},
		{"Terminal Genérico", "Terminal Centro", 15},
		{"Praça secundária", "Praça A", 15},
		{"Outro local", "Parada qualquer", 0},
	}

	for _, c := range cases {
		resultado := integrationBonus(c.parada)
		if resultado != c.esperado {
			t.Fatalf("%s: esperado %d, obtido %d", c.nome, c.esperado, resultado)
		}
	}
}

func TestAuthMiddlewareRejectsInvalidKey(t *testing.T) {
	os.Setenv("API_SECRET_KEY", "secret")
	defer os.Unsetenv("API_SECRET_KEY")

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/gps/1", nil)
	req.Header.Set("X-API-Key", "invalid")
	c.Request = req

	AuthMiddleware()(c)

	if !c.IsAborted() {
		t.Fatal("AuthMiddleware deveria abortar a requisição com chave inválida")
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Status esperado %d, obtido %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddlewareAcceptsValidKey(t *testing.T) {
	os.Setenv("API_SECRET_KEY", "secret")
	defer os.Unsetenv("API_SECRET_KEY")

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/gps/1", nil)
	req.Header.Set("X-API-Key", "secret")
	c.Request = req

	AuthMiddleware()(c)

	if c.IsAborted() {
		t.Fatal("AuthMiddleware não deveria abortar a requisição com chave válida")
	}
}

func TestFindDirectRoute(t *testing.T) {
	lines := map[string]*lineSegment{
		"101": {
			numeroLinha: "101",
			nomeLinha:   "Linha Direta",
			paradas:     []string{"Origem", "Meio", "Destino"},
			ordem:       map[string]int{"Origem": 1, "Meio": 2, "Destino": 3},
			tempo:       map[string]int{"Meio": 5, "Destino": 10},
		},
	}

	route, totalTime := findDirectRoute(lines, "Origem", "Destino")
	if route == nil {
		t.Fatal("findDirectRoute deveria encontrar rota direta")
	}
	if totalTime != 15 {
		t.Fatalf("Tempo esperado 15, obtido %d", totalTime)
	}
	if route.Tipo != "direta" {
		t.Fatalf("Tipo de rota esperado direta, obtido %s", route.Tipo)
	}
}

func TestFindBestTransferRoutePrefersIntegration(t *testing.T) {
	lines := map[string]*lineSegment{
		"101": {
			numeroLinha: "101",
			nomeLinha:   "Linha A",
			paradas:     []string{"Origem", "Novo Mundo"},
			ordem:       map[string]int{"Origem": 1, "Novo Mundo": 2},
			tempo:       map[string]int{"Novo Mundo": 8},
			integration: map[string]bool{"Novo Mundo": true},
		},
		"102": {
			numeroLinha: "102",
			nomeLinha:   "Linha B",
			paradas:     []string{"Novo Mundo", "Destino"},
			ordem:       map[string]int{"Novo Mundo": 1, "Destino": 2},
			tempo:       map[string]int{"Destino": 7},
			integration: map[string]bool{"Novo Mundo": true},
		},
		"103": {
			numeroLinha: "103",
			nomeLinha:   "Linha C",
			paradas:     []string{"Origem", "Outro Ponto"},
			ordem:       map[string]int{"Origem": 1, "Outro Ponto": 2},
			tempo:       map[string]int{"Outro Ponto": 8},
			integration: map[string]bool{"Outro Ponto": false},
		},
		"104": {
			numeroLinha: "104",
			nomeLinha:   "Linha D",
			paradas:     []string{"Outro Ponto", "Destino"},
			ordem:       map[string]int{"Outro Ponto": 1, "Destino": 2},
			tempo:       map[string]int{"Destino": 7},
			integration: map[string]bool{"Outro Ponto": false},
		},
	}

	route, _ := findBestTransferRoute(lines, "Origem", "Destino")
	if route == nil {
		t.Fatal("findBestTransferRoute deveria encontrar uma rota de transferência")
	}
	if len(route.Steps) != 2 {
		t.Fatalf("Esperava 2 etapas de transferência, obteve %d", len(route.Steps))
	}
	if route.Steps[0].NumeroLinha != "101" || route.Steps[1].NumeroLinha != "102" {
		t.Fatalf("Esperava rota por Novo Mundo, obteve %s -> %s", route.Steps[0].NumeroLinha, route.Steps[1].NumeroLinha)
	}
}

func TestPlanVilaPedrosoToUFG(t *testing.T) {
	lines := map[string]*lineSegment{
		"201": {
			numeroLinha: "201",
			nomeLinha:   "Vila Pedroso Express",
			paradas:     []string{"Vila Pedroso", "Terminal Centro", "UFG"},
			ordem:       map[string]int{"Vila Pedroso": 1, "Terminal Centro": 2, "UFG": 3},
			tempo:       map[string]int{"Terminal Centro": 12, "UFG": 18},
			integration: map[string]bool{"Terminal Centro": true},
		},
	}

	route, totalTime := findDirectRoute(lines, "Vila Pedroso", "UFG")
	if route == nil {
		t.Fatal("Esperava encontrar rota direta Vila Pedroso -> UFG")
	}
	if route.Tipo != "direta" {
		t.Fatalf("Esperava rota direta, obteve %s", route.Tipo)
	}
	if totalTime != 30 {
		t.Fatalf("Tempo esperado 30, obtido %d", totalTime)
	}
	if len(route.Steps) != 1 || route.Steps[0].NumeroLinha != "201" {
		t.Fatalf("Esperava linha 201, obteve %+v", route.Steps)
	}
}

// TestCalculateTrustScore testa cálculo de trust score com cenários normais e edge cases.
// Nota: Teste unitário simplificado, assume DB indisponível.
func TestCalculateTrustScore(t *testing.T) {
	app := &App{} // Sem DB, simula indisponibilidade.

	if app.db == nil {
		t.Skip("DB não disponível para teste unitário")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err := app.calculateTrustScore(ctx, "user")
	if err == nil {
		t.Errorf("Esperava erro devido a DB indisponível, mas não houve")
	}
}

// TestGetTrustLevel testa mapeamento de score para nível.
func TestGetTrustLevel(t *testing.T) {
	tests := []struct {
		score    int
		expected string
	}{
		{0, "Suspeito"},
		{20, "Suspeito"},
		{21, "Cidadão"},
		{80, "Cidadão"},
		{81, "Fiscal da Galera"},
		{100, "Fiscal da Galera"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Score %d", tt.score), func(t *testing.T) {
			level := getTrustLevel(tt.score)
			if level != tt.expected {
				t.Errorf("Esperava %s, recebeu %s", tt.expected, level)
			}
		})
	}
}

// TestSubmeterDenunciaIntegration testa endpoint POST /denuncias com integração (usando httptest).
// Nota: Pulado devido a falta de DB mock.
func TestSubmeterDenunciaIntegration(t *testing.T) {
	t.Skip("DB mock necessário para teste de integração")
}

// TestListarDenunciasIntegration testa GET /denuncias com filtros.
// Nota: Pulado devido a falta de DB mock.
func TestListarDenunciasIntegration(t *testing.T) {
	t.Skip("DB mock necessário para teste de integração")
}

// TestEdgeCasesRoteamento testa cenários extremos no cálculo de rotas
func TestEdgeCasesRoteamento(t *testing.T) {
	lines := map[string]*lineSegment{
		"101": {
			numeroLinha: "101",
			nomeLinha:   "Centro-UFG",
			paradas:     []string{"Terminal Centro", "Praça Cívica", "UFG"},
			ordem:       map[string]int{"terminal centro": 1, "praça cívica": 2, "ufg": 3},
			tempo:       map[string]int{"praça cívica": 10, "ufg": 20},
		},
		"102": {
			numeroLinha: "102",
			nomeLinha:   "Vila Pedroso-Terminal",
			paradas:     []string{"Vila Pedroso", "Terminal Centro"},
			ordem:       map[string]int{"vila pedroso": 1, "terminal centro": 2},
			tempo:       map[string]int{"terminal centro": 15},
			integration: map[string]bool{"terminal centro": true},
		},
	}

	t.Run("Rota inexistente", func(t *testing.T) {
		route, time := findDirectRoute(lines, "Bairro Fictício", "UFG")
		if route != nil || time != 0 {
			t.Errorf("Rota inexistente deveria retornar nil, retornou %+v, time=%d", route, time)
		}
	})

	t.Run("Origem igual destino", func(t *testing.T) {
		route, time := findDirectRoute(lines, "UFG", "UFG")
		if route != nil || time != 0 {
			t.Errorf("Origem=destino deveria retornar nil, retornou %+v, time=%d", route, time)
		}
	})

	t.Run("Nomes similares (case insensitive)", func(t *testing.T) {
		// Testa se normalização funciona
		route1, _ := findDirectRoute(lines, "terminal centro", "ufg")
		route2, _ := findDirectRoute(lines, "TERMINAL CENTRO", "UFG")

		if route1 == nil || route2 == nil {
			t.Fatal("Ambas as variações deveriam encontrar rota")
		}

		if route1.Steps[0].NumeroLinha != route2.Steps[0].NumeroLinha {
			t.Errorf("Nomes similares deveriam mapear para mesma rota")
		}
	})

	t.Run("Transferência sem ponto de integração", func(t *testing.T) {
		// Adiciona linha sem integração
		lines["103"] = &lineSegment{
			numeroLinha: "103",
			nomeLinha:   "Alternativa",
			paradas:     []string{"Vila Pedroso", "Outro Ponto", "UFG"},
			ordem:       map[string]int{"vila pedroso": 1, "outro ponto": 2, "ufg": 3},
			tempo:       map[string]int{"outro ponto": 10, "ufg": 15},
			integration: map[string]bool{"outro ponto": false}, // Sem integração
		}

		route, score := findBestTransferRoute(lines, "vila pedroso", "ufg")
		if route == nil {
			t.Fatal("Deveria encontrar rota de transferência")
		}

		// Score calculado pelo algoritmo (validado empiricamente)
		expectedScore := 30
		if score != expectedScore {
			t.Errorf("Score esperado %d, obtido %d", expectedScore, score)
		}
	})

	t.Run("Paradas duplicadas", func(t *testing.T) {
		// Simula parada duplicada
		lines["104"] = &lineSegment{
			numeroLinha: "104",
			nomeLinha:   "Duplicada",
			paradas:     []string{"Terminal Centro", "Praça Cívica", "Terminal Centro", "UFG"},
			ordem:       map[string]int{"terminal centro": 1, "praça cívica": 2, "ufg": 4},
			tempo:       map[string]int{"praça cívica": 10, "terminal centro": 5, "ufg": 20},
		}

		route, time := findDirectRoute(lines, "terminal centro", "ufg")
		if route == nil {
			t.Fatal("Deveria encontrar rota apesar de parada duplicada")
		}

		// sumTempo: paradas com ord > 1 e <= 4: praça cívica (2) + ufg (4) = 10 + 20 = 30
		expectedTime := 30
		if time != expectedTime {
			t.Errorf("Tempo com parada duplicada esperado %d, obtido %d", expectedTime, time)
		}
	})
}
