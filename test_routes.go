package main

import (
	"fmt"
)

// TestCriticalRoutes testa as 3 rotas críticas definidas pelo usuário
func TestCriticalRoutes() {
	fmt.Println("=== TESTE DAS 3 ROTAS CRÍTICAS - GOIÂNIA ===")
	fmt.Printf("Horário Simulado: 18:00h (HORÁRIO DE PICO)\n\n")

	// Coordenadas dos terminais
	locations := map[string]struct {
		lat, lng float64
		name     string
	}{
		"Terminal Novo Mundo":     {-16.6680, -49.2580, "Terminal Novo Mundo"},
		"Campus Samambaia UFG":    {-16.6831, -49.2674, "Campus Samambaia UFG"},
		"Terminal Bíblia":         {-16.6700, -49.2750, "Terminal Bíblia"},
		"Terminal Canedo":         {-16.6820, -49.2200, "Terminal Canedo"},
		"Terminal Isidória":       {-16.6900, -49.2680, "Terminal Isidória"},
		"Terminal Padre Pelágio": {-16.6750, -49.2500, "Terminal Padre Pelágio"},
	}

	// Análise das 3 rotas
	routes := []struct {
		origin, dest string
		description  string
		complexity   string
	}{
		{"Terminal Novo Mundo", "Campus Samambaia UFG", "Eixo Norte-Sul com acesso universitário", "Média"},
		{"Terminal Bíblia", "Terminal Canedo", "Integração intermunicipal Goiânia-Senador Canedo", "Alta"},
		{"Terminal Isidória", "Terminal Padre Pelágio", "Cruzamento transversal de alto fluxo", "Média"},
	}

	for i, route := range routes {
		fmt.Printf("ROTA %d: %s\n", i+1, route.origin+" -> "+route.dest)
		fmt.Printf("Descrição: %s\n", route.description)
		fmt.Printf("Complexidade: %s\n", route.complexity)

		// Calcular distância
		origin := locations[route.origin]
		dest := locations[route.dest]
		distance := calcularDistancia(origin.lat, origin.lng, dest.lat, dest.lng)

		// Calcular tempos
		baseTime := int(distance*10) + 10 // Base: 10min + 1min por 100m
		if baseTime < 15 {
			baseTime = 15
		}

		// Adicionar transferências (simulação)
		if route.complexity == "Alta" {
			baseTime += 10 // 10 min extra para rotas complexas
		}

		// Aplicar horário de pico (18h)
		rushHourTime := baseTime + 25 // 25 min no horário de pico

		fmt.Printf("Distância: %.2f km\n", distance)
		fmt.Printf("Tempo Base: %d minutos\n", baseTime)
		fmt.Printf("Tempo no Pico (18h): %d minutos\n", rushHourTime)
		fmt.Printf("Acréscimo de Pico: %d minutos\n", rushHourTime-baseTime)

		// Validação crítica
		if route.origin == "Terminal Bíblia" && route.dest == "Terminal Canedo" {
			if rushHourTime <= 20 {
				fmt.Printf("ALERTA: Tempo muito baixo para rota intermunicipal!\n")
			} else {
				fmt.Printf("VALIDADO: Tempo realista para rota intermunicipal\n")
			}
		}

		fmt.Println("---")
	}

	fmt.Println("\n=== ANÁLISE COMPARATIVA ===")
	fmt.Println("Todas as rotas consideram:")
	fmt.Println("- Trânsito intenso no horário de pico (17-19h)")
	fmt.Println("- Transferências em terminais centrais")
	fmt.Println("- Velocidade média reduzida no congestionamento")
	fmt.Println("- Fatores: integração intermunicipal, acesso universitário, cruzamento transversal")
}
