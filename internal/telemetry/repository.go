package telemetry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/redis/go-redis/v9"
)

// BusTelemetry representa a estrutura da tabela para migração
type BusTelemetry struct {
	ID         int64     `json:"id"`
	DeviceHash string    `json:"deviceHash"`
	RouteID    string    `json:"routeId"`
	Speed      float64   `json:"speed"`
	Latitude   float64   `json:"lat"`
	Longitude  float64   `json:"lng"`
	RecordedAt time.Time `json:"recordedAt"`
}

// TableName retorna o nome da tabela
func (BusTelemetry) TableName() string {
	return "gps_telemetry"
}

// Repository representa o repositório de dados para telemetria.
// Responsável por todas as operações de acesso a dados (PostgreSQL e Redis).
type Repository struct {
	db  *sql.DB
	rdb interface{} // Redis client (opcional)
}

// NewRepository cria uma nova instância de Repository com as dependências necessárias.
//
// Executa automaticamente a inicialização do schema (criação de tabelas e índices)
// se estas não existirem.
//
// Parameters:
//   - db: Conexão com banco de dados PostgreSQL
//   - rdb: Cliente Redis para cache (opcional, pode ser nil)
//
// Returns:
//   - *Repository: Instância configurada do repositório
func NewRepository(db *sql.DB, rdb interface{}) *Repository {
	if db == nil {
		logger.Warn("Telemetry", "Database connection is nil")
	}
	repo := &Repository{db: db, rdb: rdb}

	// AutoMigrate: garantir que a tabela existe
	if err := repo.InitSchema(); err != nil {
		logger.Error("Telemetry", "Failed to initialize schema: %v", err)
	}

	return repo
}

// InitSchema verifica e cria a tabela gps_telemetry se não existir
func (r *Repository) InitSchema() error {
	if r.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Verificar se a tabela existe
	var exists bool
	query := `SELECT EXISTS (
		SELECT 1 FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_name = 'gps_telemetry'
	)`

	err := r.db.QueryRow(query).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if !exists {
		logger.Info("Telemetry", "Table gps_telemetry does not exist, creating...")

		// Criar a tabela gps_telemetry (simplificada, sem partições para AutoMigrate)
		createTableSQL := `
			CREATE TABLE gps_telemetry (
				id BIGSERIAL PRIMARY KEY,
				device_hash VARCHAR(32) NOT NULL,
				geom GEOMETRY(POINT, 4326) NOT NULL,
				speed FLOAT,
				heading FLOAT,
				accuracy FLOAT,
				transport_mode VARCHAR(20),
				route_id VARCHAR(10),
				battery_level INTEGER,
				created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				recorded_at TIMESTAMP WITH TIME ZONE NOT NULL
			)
		`

		_, err = r.db.Exec(createTableSQL)
		if err != nil {
			// Se falhar por causa da extensão PostGIS, tentar criar extensão primeiro
			if strings.Contains(err.Error(), "postgis") || strings.Contains(err.Error(), "geometry") {
				_, _ = r.db.Exec("CREATE EXTENSION IF NOT EXISTS postgis")
				_, err = r.db.Exec(createTableSQL)
			}
			if err != nil {
				return fmt.Errorf("failed to create table: %w", err)
			}
		}

		// Criar índices essenciais
		indexes := []string{
			// Índice espacial GIST para buscas geográficas (ST_DWithin, ST_Contains)
			`CREATE INDEX IF NOT EXISTS idx_gps_telemetry_geom ON gps_telemetry USING GIST(geom)`,
			// Índice composto para buscar última posição por device
			`CREATE INDEX IF NOT EXISTS idx_gps_telemetry_device_time ON gps_telemetry(device_hash, created_at DESC)`,
			// Índice parcial para rotas (apenas onde route_id existe)
			`CREATE INDEX IF NOT EXISTS idx_gps_telemetry_route_time ON gps_telemetry(route_id, created_at DESC) WHERE route_id IS NOT NULL`,
			// Índice parcial para dados recentes (última hora) - otimiza queries de tempo real
			`CREATE INDEX IF NOT EXISTS idx_gps_telemetry_recent ON gps_telemetry(created_at DESC) WHERE created_at > NOW() - INTERVAL '1 hour'`,
			// Índice para geofences se tabela existir
			`CREATE INDEX IF NOT EXISTS idx_geofences_geom ON geofences USING GIST(polygon)`,
		}

		for _, idx := range indexes {
			if _, err := r.db.Exec(idx); err != nil {
				logger.Warn("Telemetry", "Failed to create index: %v", err)
			}
		}

		logger.Info("Telemetry", "Table gps_telemetry created successfully")
	} else {
		logger.Info("Telemetry", "Table gps_telemetry already exists")
	}

	return nil
}

// SaveToDatabase salva um ping de telemetria no banco de dados PostgreSQL/PostGIS.
//
// Converte as coordenadas para geometry PostGIS e insere o registro com todos os metadados.
//
// Parameters:
//   - ctx: Contexto para controle de timeout e cancelamento
//   - deviceHash: Hash SHA-256 do device_id (anonimizado)
//   - ping: Estrutura TelemetryPing com os dados GPS
//
// Returns:
//   - string: ID do registro inserido no formato "tel-{id}"
//   - error: Erro se a inserção falhar
func (r *Repository) SaveToDatabase(ctx context.Context, deviceHash string, ping *TelemetryPing) (string, error) {
	start := time.Now()

	logger.Debug("Telemetry", "SaveToDatabase: deviceHash=%s, lat=%.6f, lng=%.6f, speed=%.1f",
		deviceHash[:8]+"...", ping.Latitude, ping.Longitude, ping.Speed)

	query := `
		INSERT INTO gps_telemetry (device_hash, geom, speed, heading, accuracy, transport_mode, route_id, battery_level, recorded_at)
		VALUES ($1, ST_SetSRID(ST_MakePoint($2, $3), 4326), $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	var telemetryID int64
	err := r.db.QueryRowContext(ctx, query,
		deviceHash,
		ping.Longitude, ping.Latitude, // PostGIS: X=longitude ($2), Y=latitude ($3)
		ping.Speed,
		ping.Heading,
		ping.Accuracy,
		ping.TransportMode,
		ping.RouteID,
		ping.BatteryLevel,
		ping.RecordedAt,
	).Scan(&telemetryID)

	elapsed := time.Since(start)
	logger.Debug("Telemetry", "SaveToDatabase executed in %v", elapsed)

	if err != nil {
		logger.Error("Telemetry", "Failed to insert telemetry: %v", err)
		logger.Error("Telemetry", "ERROR Details: deviceHash=%s, lat=%.6f, lng=%.6f, speed=%.1f",
			deviceHash[:8]+"...", ping.Latitude, ping.Longitude, ping.Speed)
		return "", fmt.Errorf("failed to insert telemetry: %w", err)
	}

	logger.Info("Telemetry", "Telemetry saved with ID %d", telemetryID)
	return fmt.Sprintf("tel-%d", telemetryID), nil
}

// GetAllLatestPositions busca a última posição conhecida de cada dispositivo.
//
// Query otimizada usando DISTINCT ON (device_hash) com ORDER BY device_hash, recorded_at DESC.
//
// Retorna slice vazio [] em vez de erro quando a tabela não existe ou está vazia,
// para evitar quebrar o frontend.
//
// Parameters:
//   - ctx: Contexto para controle de timeout e cancelamento
//
// Returns:
//   - []LatestPosition: Array com a última posição de cada dispositivo
//   - error: Erro apenas se a query falhar (não retorna erro para tabela vazia)
func (r *Repository) GetAllLatestPositions(ctx context.Context) ([]LatestPosition, error) {
	start := time.Now()

	// Verificar integridade do ponteiro db
	if r.db == nil {
		logger.Warn("Telemetry", "Database connection is nil, returning empty positions")
		return []LatestPosition{}, nil
	}

	query := `
		SELECT DISTINCT ON (device_hash)
			device_hash,
			route_id,
			speed,
			ST_AsText(geom::geometry) as location,
			recorded_at
		FROM gps_telemetry
		ORDER BY device_hash, recorded_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	elapsed := time.Since(start)
	logger.Debug("Telemetry", "GetAllLatestPositions query executed in %v", elapsed)

	if err != nil {
		// Se for erro de "relation does not exist", retornar vazio ao invés de erro
		if strings.Contains(err.Error(), "relation") && strings.Contains(err.Error(), "does not exist") {
			logger.Warn("Telemetry", "Table gps_telemetry does not exist, returning empty positions")
			return []LatestPosition{}, nil
		}
		return nil, fmt.Errorf("failed to query latest positions: %w", err)
	}
	defer rows.Close()

	var positions []LatestPosition
	for rows.Next() {
		var pos LatestPosition
		var wktLocation string

		err := rows.Scan(
			&pos.DeviceHash,
			&pos.RouteID,
			&pos.Speed,
			&wktLocation,
			&pos.RecordedAt,
		)
		if err != nil {
			logger.Warn("Telemetry", "Failed to scan row: %v", err)
			continue // Skip row em vez de retornar erro
		}

		// Converter WKT para lat/lng
		// ST_AsText retorna formato: POINT(lng lat)
		lat, lng, err := parseWKTPoint(wktLocation)
		if err != nil {
			logger.Warn("Telemetry", "Failed to parse WKT: %v", err)
			continue
		}

		pos.Location = Location{
			Lat: lat,
			Lng: lng,
		}

		positions = append(positions, pos)
	}

	logger.Info("Telemetry", "GetAllLatestPositions retrieved %d positions in total %v", len(positions), time.Since(start))

	if err = rows.Err(); err != nil {
		logger.Warn("Telemetry", "Error iterating rows: %v", err)
		// Retornar o que conseguimos coletar ao invés de erro
		return positions, nil
	}

	return positions, nil
}

// GetFromDatabase busca a última posição de um dispositivo específico
// Fallback: se Redis falhar, esta função é chamada automaticamente pelo handler
func (r *Repository) GetFromDatabase(ctx context.Context, deviceHash string) (map[string]interface{}, error) {
	start := time.Now()

	// Verificar integridade do ponteiro db
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

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
		TransportMode string    `json:"transportMode"`
		RouteID       string    `json:"routeId"`
		BatteryLevel  int       `json:"batteryLevel"`
		RecordedAt    time.Time `json:"recordedAt"`
		CreatedAt     time.Time `json:"createdAt"`
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

	elapsed := time.Since(start)
	logger.Debug("Telemetry", "GetFromDatabase executed in %v", elapsed)

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

// GetBusesInArea busca ônibus dentro de um raio (metros) de um ponto geográfico
// Usa PostGIS ST_DWithin para query espacial otimizada com índice GIST
// Retorna lista de LatestPosition
func (r *Repository) GetBusesInArea(ctx context.Context, lat, lng float64, radiusMeters int) ([]LatestPosition, error) {
	start := time.Now()

	// Verificar integridade do ponteiro db
	if r.db == nil {
		logger.Warn("Telemetry", "Database connection is nil, returning empty positions")
		return []LatestPosition{}, nil
	}

	// Query PostGIS com ST_DWithin para busca espacial otimizada
	// ST_DWithin usa índice GIST (se existir) para performance sub-milissegundo
	query := `
		SELECT DISTINCT ON (device_hash)
			device_hash,
			route_id,
			speed,
			ST_AsText(geom::geometry) as location,
			recorded_at
		FROM gps_telemetry
		WHERE ST_DWithin(
			geom::geography,
			ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
			$3
		)
		AND recorded_at > NOW() - INTERVAL '5 minutes' -- Apenas dados recentes
		ORDER BY device_hash, recorded_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, lng, lat, radiusMeters)
	elapsed := time.Since(start)
	logger.Debug("Telemetry", "GetBusesInArea query executed in %v", elapsed)

	if err != nil {
		// Se for erro de "relation does not exist", retornar vazio ao invés de erro
		if strings.Contains(err.Error(), "relation") && strings.Contains(err.Error(), "does not exist") {
			logger.Warn("Telemetry", "Table gps_telemetry does not exist, returning empty positions")
			return []LatestPosition{}, nil
		}
		return nil, fmt.Errorf("failed to query buses in area: %w", err)
	}
	defer rows.Close()

	var positions []LatestPosition
	for rows.Next() {
		var pos LatestPosition
		var wktLocation string

		err := rows.Scan(
			&pos.DeviceHash,
			&pos.RouteID,
			&pos.Speed,
			&wktLocation,
			&pos.RecordedAt,
		)
		if err != nil {
			logger.Warn("Telemetry", "Failed to scan row: %v", err)
			continue // Skip row em vez de retornar erro
		}

		// Converter WKT para lat/lng
		lat, lng, err := parseWKTPoint(wktLocation)
		if err != nil {
			logger.Warn("Telemetry", "Failed to parse WKT: %v", err)
			continue
		}

		pos.Location = Location{
			Lat: lat,
			Lng: lng,
		}

		positions = append(positions, pos)
	}

	logger.Debug("Telemetry", " GetBusesInArea retrieved %d positions in total %v", len(positions), time.Since(start))

	if err = rows.Err(); err != nil {
		logger.Warn("Telemetry", "Error iterating rows: %v", err)
		// Retornar o que conseguimos coletar ao invés de erro
		return positions, nil
	}

	return positions, nil
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

// CheckProximity verifica se um ponto (lat, lng) está dentro de um polígono de geofence
// Usa PostGIS ST_Contains para verificação espacial otimizada com índice GIST
// Retorna true se estiver dentro, false se estiver fora
func (r *Repository) CheckProximity(ctx context.Context, lat, lng float64, geofenceID int64) (bool, error) {
	start := time.Now()

	// Verificar integridade do ponteiro db
	if r.db == nil {
		return false, fmt.Errorf("database connection is nil")
	}

	// Query PostGIS com ST_Contains para verificar se ponto está dentro do polígono
	query := `
		SELECT ST_Contains(
			polygon,
			ST_SetSRID(ST_MakePoint($1, $2), 4326)
		)
		FROM geofences
		WHERE id = $3
	`

	var isInside bool
	err := r.db.QueryRowContext(ctx, query, lng, lat, geofenceID).Scan(&isInside)
	elapsed := time.Since(start)
	logger.Debug("Telemetry", " CheckProximity executed in %v", elapsed)

	if err != nil {
		return false, fmt.Errorf("failed to check proximity: %w", err)
	}

	return isInside, nil
}

// CheckAllGeofences verifica se um ponto está dentro de qualquer geofence ativo
// Retorna lista de geofences onde o ponto está dentro
func (r *Repository) CheckAllGeofences(ctx context.Context, lat, lng float64) ([]Geofence, error) {
	start := time.Now()

	// Verificar integridade do ponteiro db
	if r.db == nil {
		return []Geofence{}, nil
	}

	// Query PostGIS para buscar todos os geofences que contêm o ponto
	query := `
		SELECT id, nome, tipo, polygon, created_at
		FROM geofences
		WHERE ST_Contains(
			polygon,
			ST_SetSRID(ST_MakePoint($1, $2), 4326)
		)
	`

	rows, err := r.db.QueryContext(ctx, query, lng, lat)
	elapsed := time.Since(start)
	logger.Debug("Telemetry", " CheckAllGeofences executed in %v", elapsed)

	if err != nil {
		// Se tabela não existir, retornar vazio
		if strings.Contains(err.Error(), "relation") && strings.Contains(err.Error(), "does not exist") {
			return []Geofence{}, nil
		}
		return nil, fmt.Errorf("failed to check all geofences: %w", err)
	}
	defer rows.Close()

	var geofences []Geofence
	for rows.Next() {
		var gf Geofence
		err := rows.Scan(&gf.ID, &gf.Nome, &gf.Tipo, &gf.Polygon, &gf.CreatedAt)
		if err != nil {
			logger.Warn("Telemetry", "Failed to scan geofence row: %v", err)
			continue
		}
		geofences = append(geofences, gf)
	}

	return geofences, nil
}

// CreateGeofence cria um novo geofence no banco de dados
func (r *Repository) CreateGeofence(ctx context.Context, nome, tipo, polygon string) (int64, error) {
	start := time.Now()

	// Validar tipo
	if tipo != "Terminal" && tipo != "Rota" {
		return 0, fmt.Errorf("invalid geofence type: must be 'Terminal' or 'Rota'")
	}

	query := `
		INSERT INTO geofences (nome, tipo, polygon)
		VALUES ($1, $2, ST_GeomFromText($3, 4326))
		RETURNING id
	`

	var id int64
	err := r.db.QueryRowContext(ctx, query, nome, tipo, polygon).Scan(&id)
	elapsed := time.Since(start)
	logger.Debug("Telemetry", " CreateGeofence executed in %v", elapsed)

	if err != nil {
		return 0, fmt.Errorf("failed to create geofence: %w", err)
	}

	return id, nil
}

// CreateGeofenceAlert cria um registro de alerta de geofencing
func (r *Repository) CreateGeofenceAlert(ctx context.Context, deviceHash string, geofenceID int64, geofenceNome, estado string, lat, lng float64) error {
	start := time.Now()

	query := `
		INSERT INTO geofence_alerts (device_hash, geofence_id, geofence_nome, estado, lat, lng, ocorrido_em)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`

	_, err := r.db.ExecContext(ctx, query, deviceHash, geofenceID, geofenceNome, estado, lat, lng)
	elapsed := time.Since(start)
	logger.Debug("Telemetry", " CreateGeofenceAlert executed in %v", elapsed)

	if err != nil {
		return fmt.Errorf("failed to create geofence alert: %w", err)
	}

	return nil
}

// GetRecentGeofenceAlerts retorna os últimos alertas de geofencing
func (r *Repository) GetRecentGeofenceAlerts(ctx context.Context, limit int) ([]GeofenceAlert, error) {
	start := time.Now()

	// Verificar integridade do ponteiro db
	if r.db == nil {
		return []GeofenceAlert{}, nil
	}

	query := `
		SELECT id, device_hash, geofence_id, geofence_nome, estado, lat, lng, ocorrido_em
		FROM geofence_alerts
		ORDER BY ocorrido_em DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	elapsed := time.Since(start)
	logger.Debug("Telemetry", " GetRecentGeofenceAlerts executed in %v", elapsed)

	if err != nil {
		// Se tabela não existir, retornar vazio
		if strings.Contains(err.Error(), "relation") && strings.Contains(err.Error(), "does not exist") {
			return []GeofenceAlert{}, nil
		}
		return nil, fmt.Errorf("failed to get recent geofence alerts: %w", err)
	}
	defer rows.Close()

	var alerts []GeofenceAlert
	for rows.Next() {
		var alert GeofenceAlert
		err := rows.Scan(
			&alert.ID,
			&alert.DeviceHash,
			&alert.GeofenceID,
			&alert.GeofenceNome,
			&alert.Estado,
			&alert.Lat,
			&alert.Lng,
			&alert.OcorridoEm,
		)
		if err != nil {
			logger.Warn("Telemetry", "Failed to scan alert row: %v", err)
			continue
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// GetAverageSpeedInPolygon calcula a velocidade média de todos os ônibus em um polígono nos últimos 15 minutos
// Usado para identificar gargalos de trânsito em tempo real
func (r *Repository) GetAverageSpeedInPolygon(ctx context.Context, geofenceID int64, minutes int) (float64, error) {
	start := time.Now()

	// Verificar integridade do ponteiro db
	if r.db == nil {
		return 0, fmt.Errorf("database connection is nil")
	}

	// Query PostGIS com ST_Contains para filtrar pontos dentro do polígono
	// Calcula velocidade média dos últimos N minutos
	query := `
		SELECT AVG(t.speed)
		FROM gps_telemetry t
		WHERE t.speed > 0
		AND t.speed < 150 -- Filtrar outliers (velocidades > 150km/h são erros)
		AND ST_Contains(
			(SELECT polygon FROM geofences WHERE id = $1),
			t.geom
		)
		AND t.recorded_at > NOW() - INTERVAL '1 minute' * $2
	`

	var avgSpeed sql.NullFloat64
	err := r.db.QueryRowContext(ctx, query, geofenceID, minutes).Scan(&avgSpeed)
	elapsed := time.Since(start)
	logger.Debug("Telemetry", " GetAverageSpeedInPolygon executed in %v", elapsed)

	if err != nil {
		// Se geofence não existir ou não houver dados, retornar 0
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to calculate average speed: %w", err)
	}

	// Se avgSpeed for NULL (sem dados), retornar 0
	if !avgSpeed.Valid {
		return 0, nil
	}

	return avgSpeed.Float64, nil
}

// GetBusesCountInPolygon conta quantos ônibus estão dentro de um polígono
// Usado para detectar congestionamento (3+ ônibus < 10km/h)
func (r *Repository) GetBusesCountInPolygon(ctx context.Context, geofenceID int64, minutes int) (int, error) {
	start := time.Now()

	// Verificar integridade do ponteiro db
	if r.db == nil {
		return 0, fmt.Errorf("database connection is nil")
	}

	// Query para contar ônibus distintos dentro do polígono
	query := `
		SELECT COUNT(DISTINCT device_hash)
		FROM gps_telemetry t
		WHERE ST_Contains(
			(SELECT polygon FROM geofences WHERE id = $1),
			t.geom
		)
		AND t.recorded_at > NOW() - INTERVAL '1 minute' * $2
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, geofenceID, minutes).Scan(&count)
	elapsed := time.Since(start)
	logger.Debug("Telemetry", " GetBusesCountInPolygon executed in %v", elapsed)

	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to count buses in polygon: %w", err)
	}

	return count, nil
}

// GetDeviceLatestPosition busca a última posição conhecida de um dispositivo
func (r *Repository) GetDeviceLatestPosition(ctx context.Context, deviceHash string) (lat, lng float64, speed float64, err error) {
	start := time.Now()

	// Verificar integridade do ponteiro db
	if r.db == nil {
		return 0, 0, 0, fmt.Errorf("database connection is nil")
	}

	query := `
		SELECT 
			ST_Y(geom::geometry) as lat,
			ST_X(geom::geometry) as lng,
			speed
		FROM gps_telemetry
		WHERE device_hash = $1
		ORDER BY recorded_at DESC
		LIMIT 1
	`

	err = r.db.QueryRowContext(ctx, query, deviceHash).Scan(&lat, &lng, &speed)
	elapsed := time.Since(start)
	logger.Debug("Telemetry", " GetDeviceLatestPosition executed in %v", elapsed)

	if err != nil {
		return 0, 0, 0, err
	}

	return lat, lng, speed, nil
}

// CalculateDistance calcula a distância em metros entre dois pontos usando ST_Distance do PostGIS
func (r *Repository) CalculateDistance(ctx context.Context, lat1, lng1, lat2, lng2 float64) (float64, error) {
	start := time.Now()

	// Verificar integridade do ponteiro db
	if r.db == nil {
		return 0, fmt.Errorf("database connection is nil")
	}

	// Usar ST_Distance com geography para cálculo em metros
	query := `
		SELECT ST_Distance(
			ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
			ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography
		)
	`

	var distance float64
	err := r.db.QueryRowContext(ctx, query, lng1, lat1, lng2, lat2).Scan(&distance)
	elapsed := time.Since(start)
	logger.Debug("Telemetry", " CalculateDistance executed in %v", elapsed)

	if err != nil {
		return 0, fmt.Errorf("failed to calculate distance: %w", err)
	}

	return distance, nil
}

// FleetHealthMetric representa métrica de saúde da frota
type FleetHealthMetric struct {
	DeviceHash    string  `json:"deviceHash"`
	RouteID       string  `json:"routeId"`
	MovingTimeMin float64 `json:"movingTimeMin"`
	TotalTimeMin  float64 `json:"totalTimeMin"`
	Efficiency    float64 `json:"efficiency"`   // (MovingTime / TotalTime) * 100
	DwellTimeMin  float64 `json:"dwellTimeMin"` // Tempo parado em terminais
	AvgSpeedKmh   float64 `json:"avgSpeedKmh"`
	TotalPings    int     `json:"totalPings"`
}

// GetFleetHealth calcula métricas de saúde da frota nas últimas 24 horas
// Eficiência = Tempo em movimento / Tempo total em serviço
func (r *Repository) GetFleetHealth(ctx context.Context, hours int) ([]FleetHealthMetric, error) {
	start := time.Now()

	// Verificar integridade do ponteiro db
	if r.db == nil {
		return []FleetHealthMetric{}, nil
	}

	// Query para calcular métricas por device_hash
	// Tempo em movimento = COUNT WHERE speed > 5 km/h
	// Tempo total = COUNT ALL
	query := `
		SELECT 
			device_hash,
			route_id,
			SUM(CASE WHEN speed > 5 THEN 1 ELSE 0 END) * 10.0 / 60.0 as moving_time_min,
			COUNT(*) * 10.0 / 60.0 as total_time_min,
			AVG(speed) as avg_speed_kmh,
			COUNT(*) as total_pings
		FROM gps_telemetry
		WHERE recorded_at > NOW() - INTERVAL '1 hour' * $1
		GROUP BY device_hash, route_id
		ORDER BY device_hash
	`

	rows, err := r.db.QueryContext(ctx, query, hours)
	elapsed := time.Since(start)
	logger.Debug("Telemetry", " GetFleetHealth executed in %v", elapsed)

	if err != nil {
		if strings.Contains(err.Error(), "relation") && strings.Contains(err.Error(), "does not exist") {
			return []FleetHealthMetric{}, nil
		}
		return nil, fmt.Errorf("failed to get fleet health: %w", err)
	}
	defer rows.Close()

	var metrics []FleetHealthMetric
	for rows.Next() {
		var m FleetHealthMetric
		err := rows.Scan(
			&m.DeviceHash,
			&m.RouteID,
			&m.MovingTimeMin,
			&m.TotalTimeMin,
			&m.AvgSpeedKmh,
			&m.TotalPings,
		)
		if err != nil {
			logger.Warn("Telemetry", "Failed to scan fleet health row: %v", err)
			continue
		}

		// Calcular eficiência (%)
		if m.TotalTimeMin > 0 {
			m.Efficiency = (m.MovingTimeMin / m.TotalTimeMin) * 100
		}

		metrics = append(metrics, m)
	}

	// Calcular Dwell Time para cada device (tempo parado em geofences Terminal)
	for i := range metrics {
		dwellTime, err := r.GetDwellTime(ctx, metrics[i].DeviceHash, hours)
		if err == nil {
			metrics[i].DwellTimeMin = dwellTime
		}
	}

	return metrics, nil
}

// GetDwellTime calcula o tempo que um dispositivo ficou parado em geofences do tipo Terminal
func (r *Repository) GetDwellTime(ctx context.Context, deviceHash string, hours int) (float64, error) {
	start := time.Now()

	// Verificar integridade do ponteiro db
	if r.db == nil {
		return 0, nil
	}

	// Query para contar pings dentro de geofences Terminal com speed < 5 km/h
	query := `
		SELECT COUNT(*) * 10.0 / 60.0 as dwell_time_min
		FROM gps_telemetry t
		INNER JOIN geofences g ON ST_Contains(g.polygon, t.geom)
		WHERE t.device_hash = $1
		AND g.tipo = 'Terminal'
		AND t.speed < 5
		AND t.recorded_at > NOW() - INTERVAL '1 hour' * $2
	`

	var dwellTime float64
	err := r.db.QueryRowContext(ctx, query, deviceHash, hours).Scan(&dwellTime)
	elapsed := time.Since(start)
	logger.Debug("Telemetry", " GetDwellTime executed in %v", elapsed)

	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get dwell time: %w", err)
	}

	return dwellTime, nil
}

// AuditLog representa um registro de auditoria
type AuditLog struct {
	ID           int64     `json:"id"`
	ActorType    string    `json:"actorType"` // user, device, system
	ActorID      string    `json:"actorId"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resourceType"`
	ResourceID   string    `json:"resourceId"`
	IPAddress    string    `json:"ipAddress"`
	UserAgent    string    `json:"userAgent"`
	Payload      string    `json:"payload"`
	OldValue     string    `json:"oldValue"`
	NewValue     string    `json:"newValue"`
	CreatedAt    time.Time `json:"createdAt"`
}

// LogAudit registra uma ação no sistema para rastro total
func (r *Repository) LogAudit(ctx context.Context, actorType, actorID, action, resourceType, resourceID, ipAddress, userAgent, payload, oldValue, newValue string) (int64, error) {
	start := time.Now()

	// Verificar integridade do ponteiro db
	if r.db == nil {
		return 0, fmt.Errorf("database connection is nil")
	}

	// Usar função SQL para inserir audit log
	query := `
		SELECT log_audit($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10::jsonb)
	`

	var logID int64
	err := r.db.QueryRowContext(ctx, query,
		actorType, actorID, action, resourceType, resourceID,
		ipAddress, userAgent, payload, oldValue, newValue).Scan(&logID)

	elapsed := time.Since(start)
	logger.Debug("Telemetry", " LogAudit executed in %v", elapsed)

	if err != nil {
		return 0, fmt.Errorf("failed to log audit: %w", err)
	}

	// Invalidar cache de conformidade após inserção de audit log
	go r.invalidateComplianceCache(context.Background())

	return logID, nil
}

// invalidateComplianceCache limpa todas as chaves de cache de conformidade
func (r *Repository) invalidateComplianceCache(ctx context.Context) {
	if r.rdb == nil {
		return
	}

	// Converter para redis.Client se possível
	redisClient, ok := r.rdb.(*redis.Client)
	if !ok {
		return
	}

	// Deletar chaves de cache de conformidade
	pattern := "compliance:*"
	keys, err := redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		logger.Info("Telemetry", " Failed to find compliance cache keys: %v", err)
		return
	}

	if len(keys) > 0 {
		err = redisClient.Del(ctx, keys...).Err()
		if err != nil {
			logger.Info("Telemetry", " Failed to invalidate compliance cache: %v", err)
		} else {
			logger.Info("Telemetry", " Invalidated %d compliance cache keys", len(keys))
		}
	}
}

// FleetStatus representa o status da frota para o CCO
type FleetStatus struct {
	TotalActiveBuses    int       `json:"totalActiveBuses"`
	TotalGeofenceAlerts int       `json:"totalGeofenceAlerts"`
	AverageSpeed        float64   `json:"averageSpeed"`
	TotalBuses          int       `json:"totalBuses"`
	OfflineBuses        int       `json:"offlineBuses"`
	LastUpdated         time.Time `json:"lastUpdated"`
}

// GetFleetStatus retorna métricas gerenciais da frota para o Centro de Controle Operacional
// Usa índice idx_gps_telemetry_time_recent para otimizar queries por tempo
// Implementa cache Redis com TTL de 2 minutos
func (r *Repository) GetFleetStatus(ctx context.Context) (*FleetStatus, error) {
	start := time.Now()

	// Tentar buscar do cache Redis primeiro
	if r.rdb != nil {
		redisClient, ok := r.rdb.(*redis.Client)
		if ok {
			cached, err := redisClient.Get(ctx, CacheKeyFleetStatus).Result()
			if err == nil && cached != "" {
				logger.Info("Telemetry", " FleetStatus cache hit")
				var status FleetStatus
				if err := json.Unmarshal([]byte(cached), &status); err == nil {
					return &status, nil
				}
			}
		}
	}

	// Verificar integridade do ponteiro db
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	status := &FleetStatus{
		LastUpdated: time.Now(),
	}

	// 1. Total de ônibus ativos (enviaram sinal nos últimos 5 min)
	// Usa índice idx_gps_telemetry_time_recent
	queryActive := `
		SELECT COUNT(DISTINCT device_hash)
		FROM gps_telemetry
		WHERE recorded_at > NOW() - INTERVAL '5 minutes'
	`
	err := r.db.QueryRowContext(ctx, queryActive).Scan(&status.TotalActiveBuses)
	if err != nil {
		logger.Debug("Telemetry", " Failed to get active buses: %v", err)
		status.TotalActiveBuses = 0
	}

	// 2. Total de alertas de Geofencing nas últimas 24h
	queryAlerts := `
		SELECT COUNT(*)
		FROM geofence_alerts
		WHERE ocorrido_em > NOW() - INTERVAL '24 hours'
	`
	err = r.db.QueryRowContext(ctx, queryAlerts).Scan(&status.TotalGeofenceAlerts)
	if err != nil {
		if !strings.Contains(err.Error(), "does not exist") {
			logger.Debug("Telemetry", " Failed to get geofence alerts: %v", err)
		}
		status.TotalGeofenceAlerts = 0
	}

	// 3. Média de velocidade de toda a frota hoje
	// Usa índice idx_gps_telemetry_time_recent
	querySpeed := `
		SELECT AVG(speed)
		FROM gps_telemetry
		WHERE speed > 0
		AND speed < 150
		AND recorded_at > NOW() - INTERVAL '24 hours'
	`
	var avgSpeed sql.NullFloat64
	err = r.db.QueryRowContext(ctx, querySpeed).Scan(&avgSpeed)
	if err != nil {
		logger.Debug("Telemetry", " Failed to get average speed: %v", err)
		status.AverageSpeed = 0
	} else if avgSpeed.Valid {
		status.AverageSpeed = avgSpeed.Float64
	}

	// 4. Total de ônibus distintos (últimas 24h)
	// Usa índice idx_gps_telemetry_time_recent
	queryTotal := `
		SELECT COUNT(DISTINCT device_hash)
		FROM gps_telemetry
		WHERE recorded_at > NOW() - INTERVAL '24 hours'
	`
	err = r.db.QueryRowContext(ctx, queryTotal).Scan(&status.TotalBuses)
	if err != nil {
		logger.Debug("Telemetry", " Failed to get total buses: %v", err)
		status.TotalBuses = status.TotalActiveBuses
	}

	// 5. Ônibus offline (total - ativos)
	status.OfflineBuses = status.TotalBuses - status.TotalActiveBuses
	if status.OfflineBuses < 0 {
		status.OfflineBuses = 0
	}

	elapsed := time.Since(start)
	logger.Debug("Telemetry", " GetFleetStatus executed in %v | Active: %d | Total: %d | Offline: %d | AvgSpeed: %.2f",
		elapsed, status.TotalActiveBuses, status.TotalBuses, status.OfflineBuses, status.AverageSpeed)

	// Salvar no cache Redis
	if r.rdb != nil {
		redisClient, ok := r.rdb.(*redis.Client)
		if ok {
			statusJSON, err := json.Marshal(status)
			if err == nil {
				redisClient.Set(ctx, CacheKeyFleetStatus, statusJSON, ComplianceCacheTTL)
				logger.Info("Telemetry", "FleetStatus cached")
			}
		}
	}

	return status, nil
}

// HistoryPoint representa um ponto no histórico de trajetória
type HistoryPoint struct {
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	Speed      float64   `json:"speed"`
	RecordedAt time.Time `json:"recordedAt"`
}

// GetHistory busca o histórico de coordenadas de um dispositivo em um intervalo de tempo
// Usa índice idx_gps_telemetry_time_recent para performance
// Retorna pontos ordenados por recorded_at ASC
func (r *Repository) GetHistory(ctx context.Context, deviceID string, start, end time.Time) ([]HistoryPoint, error) {
	startTime := time.Now()

	// Verificar integridade do ponteiro db
	if r.db == nil {
		return []HistoryPoint{}, fmt.Errorf("database connection is nil")
	}

	// Hash do deviceID para busca
	deviceHash := deviceID

	// Query com índice idx_gps_telemetry_time_recent
	query := `
		SELECT 
			ST_Y(geom::geometry) as lat,
			ST_X(geom::geometry) as lng,
			speed,
			recorded_at
		FROM gps_telemetry
		WHERE device_hash = $1
		AND recorded_at >= $2
		AND recorded_at <= $3
		ORDER BY recorded_at ASC
		LIMIT 10000
	`

	rows, err := r.db.QueryContext(ctx, query, deviceHash, start, end)
	elapsed := time.Since(startTime)
	logger.Debug("Telemetry", " GetHistory query executed in %v", elapsed)

	if err != nil {
		// Se tabela não existir, retornar vazio
		if strings.Contains(err.Error(), "relation") && strings.Contains(err.Error(), "does not exist") {
			return []HistoryPoint{}, nil
		}
		return nil, fmt.Errorf("failed to get history: %w", err)
	}
	defer rows.Close()

	var points []HistoryPoint
	for rows.Next() {
		var point HistoryPoint
		err := rows.Scan(&point.Lat, &point.Lng, &point.Speed, &point.RecordedAt)
		if err != nil {
			logger.Warn("Telemetry", "Failed to scan history row: %v", err)
			continue
		}
		points = append(points, point)
	}

	logger.Debug("Telemetry", " GetHistory retrieved %d points in total %v", len(points), time.Since(startTime))

	if err = rows.Err(); err != nil {
		logger.Warn("Telemetry", "Error iterating history rows: %v", err)
		return points, nil
	}

	return points, nil
}
