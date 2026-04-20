package telemetry

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/redis/go-redis/v9"
)

// GeofencingService verifica se ônibus saíram da cerca
type GeofencingService struct {
	redis *redis.Client
}

// NewGeofencingService cria um novo serviço de geofencing
func NewGeofencingService(rdb *redis.Client) *GeofencingService {
	return &GeofencingService{
		redis: rdb,
	}
}

// GoiâniaGeofence define a cerca de Goiânia (aprox. 25km raio)
// Usa a struct Geofence de model.go com campos adaptados
var GoiâniaGeofence = struct {
	CenterLat float64
	CenterLng float64
	RadiusKM  float64
}{
	CenterLat: -16.6869,
	CenterLng: -49.2648,
	RadiusKM:  25.0,
}

// CheckGeofence verifica se as coordenadas estão dentro da cerca
func (g *GeofencingService) CheckGeofence(lat, lng float64, fence interface{}) bool {
	// Fórmula de Haversine para calcular distância em km
	fenceData := fence.(struct {
		CenterLat float64
		CenterLng float64
		RadiusKM  float64
	})
	distance := haversine(lat, lng, fenceData.CenterLat, fenceData.CenterLng)
	return distance <= fenceData.RadiusKM
}

// haversine calcula distância entre dois pontos em km
func haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371 // raio da Terra em km

	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLng/2)*math.Sin(dLng/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// ProcessTelemetryWithGeofence processa telemetria e verifica geofencing
func (g *GeofencingService) ProcessTelemetryWithGeofence(
	ctx context.Context,
	repo *Repository,
	deviceID string,
	lat, lng float64,
) error {
	// Verificar se está dentro da geofence padrão (Goiânia)
	geofenceName := "Goiânia"
	geofenceCenter := struct {
		CenterLat float64
		CenterLng float64
		RadiusKM  float64
	}{
		CenterLat: -16.6869,
		CenterLng: -49.2648,
		RadiusKM:  25.0,
	}
	inside := g.CheckGeofence(lat, lng, geofenceCenter)

	if !inside {
		logger.Warn("Geofencing", "Ônibus %s saiu da cerca de %s! Lat: %.6f, Lng: %.6f",
			deviceID, geofenceName, lat, lng)

		// Publicar alerta no Redis
		alertData := map[string]interface{}{
			"device_id": deviceID,
			"lat":       lat,
			"lng":       lng,
			"alert":     "GEOFENCE_BREACH",
			"fence":     "Cerca de Goiânia",
			"timestamp": time.Now().Format(time.RFC3339),
		}

		if err := g.redis.Publish(ctx, "geofence_alerts", alertData).Err(); err != nil {
			logger.Error("Geofencing", "Erro ao publicar alerta no Redis: %v", err)
		}

		// Log de auditoria crítico
		go func() {
			auditCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Extrair IP e user agent do contexto se disponível
			actorID := "system"
			actorType := "system"
			action := "GEOFENCE_BREACH"
			resourceType := "geofence"
			resourceID := "Cerca de Goiânia"

			repo.LogAudit(auditCtx, actorType, actorID, action, resourceType, resourceID,
				"", "", "", "", "Bus saiu da cerca autorizada")
		}()

		return nil
	}

	return nil
}

// GetLastKnownPosition obtém última posição conhecida do ônibus
func (g *GeofencingService) GetLastKnownPosition(ctx context.Context, deviceID string) (lat, lng float64, err error) {
	// Buscar do Redis
	key := "bus:" + deviceID + ":last_position"
	data, err := g.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return 0, 0, err
	}

	if len(data) == 0 {
		return 0, 0, nil // Sem posição conhecida
	}

	latStr, ok := data["lat"]
	if !ok {
		return 0, 0, nil
	}

	lngStr, ok := data["lng"]
	if !ok {
		return 0, 0, nil
	}

	// Converter strings para float
	var latFloat, lngFloat float64
	if _, err := fmt.Sscanf(latStr, "%f", &latFloat); err != nil {
		return 0, 0, err
	}
	if _, err := fmt.Sscanf(lngStr, "%f", &lngFloat); err != nil {
		return 0, 0, err
	}

	return latFloat, lngFloat, nil
}
