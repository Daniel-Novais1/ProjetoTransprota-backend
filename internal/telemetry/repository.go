package telemetry

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"time"
)

// Repository representa o repositório de telemetria
type Repository struct {
	db *sql.DB
}

// NewRepository cria um novo repositório
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// SaveToDatabase salva ping de telemetria no banco de dados
func (r *Repository) SaveToDatabase(ctx context.Context, deviceHash string, ping *TelemetryPing) (string, error) {
	query := `
		INSERT INTO gps_telemetry (device_hash, geom, speed)
		VALUES ($1, ST_SetSRID(ST_MakePoint($2, $3), 4326), $4)
		RETURNING id
	`

	var telemetryID int64
	err := r.db.QueryRowContext(ctx, query,
		deviceHash,
		ping.Longitude, ping.Latitude, // PostGIS: X=longitude ($2), Y=latitude ($3)
		ping.Speed,
	).Scan(&telemetryID)

	if err != nil {
		return "", fmt.Errorf("failed to insert telemetry: %w", err)
	}

	return fmt.Sprintf("tel-%d", telemetryID), nil
}

// GetAllLatestPositions busca a última posição conhecida de cada dispositivo
// Query: SELECT DISTINCT ON (device_hash) com ORDER BY device_hash, recorded_at DESC
func (r *Repository) GetAllLatestPositions(ctx context.Context) ([]LatestPosition, error) {
	query := `
		SELECT DISTINCT ON (device_hash)
			id,
			device_hash,
			route_id,
			speed,
			ST_AsText(geom::geometry) as location,
			recorded_at
		FROM gps_telemetry
		ORDER BY device_hash, recorded_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest positions: %w", err)
	}
	defer rows.Close()

	var positions []LatestPosition
	for rows.Next() {
		var pos LatestPosition
		var wktLocation string
		var id int64

		err := rows.Scan(
			&id,
			&pos.DeviceHash,
			&pos.RouteID,
			&pos.Speed,
			&wktLocation,
			&pos.RecordedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Converter WKT para lat/lng
		// ST_AsText retorna formato: POINT(lng lat)
		lat, lng, err := parseWKTPoint(wktLocation)
		if err != nil {
			log.Printf("[Telemetry] Failed to parse WKT: %v", err)
			continue
		}

		pos.Location = Location{
			Lat: lat,
			Lng: lng,
		}

		positions = append(positions, pos)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return positions, nil
}

// GetFromDatabase busca a última posição de um dispositivo específico
func (r *Repository) GetFromDatabase(ctx context.Context, deviceHash string) (map[string]interface{}, error) {
	query := `
		SELECT 
			ST_X(geom::geometry) as lng,
			ST_Y(geom::geometry) as lat,
			speed,
			heading,
			accuracy,
			transport_mode,
			route_id,
			battery_level,
			recorded_at,
			created_at
		FROM gps_telemetry
		WHERE device_hash = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var position struct {
		Lng           float64   `json:"lng"`
		Lat           float64   `json:"lat"`
		Speed         float64   `json:"speed"`
		Heading       float64   `json:"heading"`
		Accuracy      float64   `json:"accuracy"`
		TransportMode string    `json:"transport_mode"`
		RouteID       string    `json:"route_id"`
		BatteryLevel  int       `json:"battery_level"`
		RecordedAt    time.Time `json:"recorded_at"`
		CreatedAt     time.Time `json:"created_at"`
	}

	err := r.db.QueryRowContext(ctx, query, deviceHash).Scan(
		&position.Lng,
		&position.Lat,
		&position.Speed,
		&position.Heading,
		&position.Accuracy,
		&position.TransportMode,
		&position.RouteID,
		&position.BatteryLevel,
		&position.RecordedAt,
		&position.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"lng":            position.Lng,
		"lat":            position.Lat,
		"speed":          position.Speed,
		"heading":        position.Heading,
		"accuracy":       position.Accuracy,
		"transport_mode": position.TransportMode,
		"route_id":       position.RouteID,
		"battery_level":  position.BatteryLevel,
		"recorded_at":    position.RecordedAt,
		"created_at":     position.CreatedAt,
	}, nil
}

// GetTelemetryStats retorna estatísticas rápidas de telemetria
func (r *Repository) GetTelemetryStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Contar pings nas últimas 24h no banco
	var pingCount24h int64
	query := `SELECT COUNT(*) FROM gps_telemetry WHERE created_at > NOW() - INTERVAL '24 hours'`
	err := r.db.QueryRowContext(ctx, query).Scan(&pingCount24h)
	if err == nil {
		stats["pings_24h"] = pingCount24h
	}

	// Velocidade média nas últimas horas
	var avgSpeed float64
	query = `SELECT AVG(speed) FROM gps_telemetry 
			 WHERE created_at > NOW() - INTERVAL '1 hour' AND speed > 0`
	err = r.db.QueryRowContext(ctx, query).Scan(&avgSpeed)
	if err == nil {
		stats["avg_speed_1h_kmh"] = math.Round(avgSpeed*10) / 10
	}

	return stats, nil
}

// parseWKTPoint converte WKT POINT(lng lat) para lat, lng
func parseWKTPoint(wkt string) (lat, lng float64, err error) {
	// Formato esperado: POINT(lng lat)
	// Exemplo: POINT(-49.2643 -16.6864)
	var parsedLng, parsedLat float64
	_, err = fmt.Sscanf(wkt, "POINT(%f %f)", &parsedLng, &parsedLat)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid WKT format: %w", err)
	}

	return parsedLat, parsedLng, nil
}
