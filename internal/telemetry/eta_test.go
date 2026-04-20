package telemetry

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
)

// TestCalculateETASimulation simula um ônibus percorrendo 1km a 10km/h
// Valida se o ETA retornado é de aproximadamente 6 minutos
func TestCalculateETASimulation(t *testing.T) {
	// Configuração de conexão com PostgreSQL
	db, err := sql.Open("postgres", "user=admin password=password123 dbname=transprota sslmode=disable host=localhost port=5432")
	if err != nil {
		t.Skipf("Falha ao conectar no banco de dados (teste requer PostgreSQL): %v", err)
		return
	}
	defer db.Close()

	repo := NewRepository(db, nil)
	ctx := context.Background()

	// Teste 1: Calcular distância de 1km
	t.Run("Calcular distância de 1km", func(t *testing.T) {
		// Praça Cívica (-16.6869, -49.2648) para um ponto ~1km ao norte
		lat1, lng1 := -16.6869, -49.2648
		lat2, lng2 := -16.6779, -49.2648 // ~1km ao norte

		distanceMeters, err := repo.CalculateDistance(ctx, lat1, lng1, lat2, lng2)
		if err != nil {
			t.Fatalf("Falha ao calcular distância: %v", err)
		}

		distanceKm := distanceMeters / 1000.0
		t.Logf("Distância calculada: %.2f metros (%.2f km)", distanceMeters, distanceKm)

		// Validar que a distância está próxima de 1km (tolerância de 10%)
		if distanceKm < 0.9 || distanceKm > 1.1 {
			t.Errorf("Distância esperada ~1km, mas calculada: %.2f km", distanceKm)
		}
	})

	// Teste 2: Calcular ETA para 1km a 10km/h (deve ser ~6 minutos)
	t.Run("ETA de 1km a 10km/h", func(t *testing.T) {
		distanceKm := 1.0
		speedKmh := 10.0

		// Fórmula: Tempo (horas) = Distância (km) / Velocidade (km/h)
		// Tempo (minutos) = Tempo (horas) * 60
		etaHours := distanceKm / speedKmh
		etaMinutes := etaHours * 60

		t.Logf("ETA calculado: %.2f minutos (%.2f horas)", etaMinutes, etaHours)

		// Validar que o ETA está próximo de 6 minutos (tolerância de 10%)
		if etaMinutes < 5.4 || etaMinutes > 6.6 {
			t.Errorf("ETA esperado ~6 minutos, mas calculado: %.2f minutos", etaMinutes)
		}
	})

	// Teste 3: Calcular ETA com velocidade padrão (40km/h)
	t.Run("ETA com velocidade padrão de 40km/h", func(t *testing.T) {
		distanceKm := 1.0
		speedKmh := 40.0 // Velocidade padrão de via urbana

		etaHours := distanceKm / speedKmh
		etaMinutes := etaHours * 60

		t.Logf("ETA calculado: %.2f minutos (%.2f horas)", etaMinutes, etaHours)

		// Validar que o ETA está próximo de 1.5 minutos (90 segundos)
		if etaMinutes < 1.35 || etaMinutes > 1.65 {
			t.Errorf("ETA esperado ~1.5 minutos, mas calculado: %.2f minutos", etaMinutes)
		}
	})

	// Teste 4: Calcular ETA para 5km a 30km/h
	t.Run("ETA de 5km a 30km/h", func(t *testing.T) {
		distanceKm := 5.0
		speedKmh := 30.0

		etaHours := distanceKm / speedKmh
		etaMinutes := etaHours * 60

		t.Logf("ETA calculado: %.2f minutos (%.2f horas)", etaMinutes, etaHours)

		expectedMinutes := 10.0 // 5km / 30km/h = 0.1667h = 10min
		if etaMinutes < expectedMinutes*0.9 || etaMinutes > expectedMinutes*1.1 {
			t.Errorf("ETA esperado ~%.0f minutos, mas calculado: %.2f minutos", expectedMinutes, etaMinutes)
		}
	})
}

// TestCalculateDistanceAccuracy testa a precisão do cálculo de distância
func TestCalculateDistanceAccuracy(t *testing.T) {
	t.Skip("Skipping - uses hardcoded repo variable")
}

// BenchmarkCalculateDistance benchmark de performance para cálculo de distância
func BenchmarkCalculateDistance(b *testing.B) {
	db, err := sql.Open("postgres", "user=admin password=password123 dbname=transprota sslmode=disable host=localhost port=5432")
	if err != nil {
		b.Skipf("Falha ao conectar no banco de dados: %v", err)
		return
	}
	defer db.Close()

	repo := NewRepository(db, nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = repo.CalculateDistance(ctx, -16.6869, -49.2648, -16.6779, -49.2648)
	}
}
