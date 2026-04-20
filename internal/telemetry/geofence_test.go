package telemetry

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func TestCheckProximity(t *testing.T) {
	t.Skip("Skipping - requires geofences table in PostgreSQL")
}

func TestCreateGeofenceValidation(t *testing.T) {
	db, err := sql.Open("postgres", "user=admin password=password123 dbname=transprota sslmode=disable host=localhost port=5432")
	if err != nil {
		t.Skipf("Falha ao conectar no banco de dados: %v", err)
		return
	}
	defer db.Close()

	repo := NewRepository(db, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("Tipo invalido deve falhar", func(t *testing.T) {
		_, err := repo.CreateGeofence(ctx, "Teste", "Invalido", "POLYGON((-49.27 -16.69, -49.27 -16.68, -49.26 -16.68, -49.26 -16.69, -49.27 -16.69))")
		if err == nil {
			t.Error("Esperava erro para tipo invalido")
		}
	})
}

func BenchmarkCheckProximity(b *testing.B) {
	b.Skip("Skipping - requires PostgreSQL")
}
