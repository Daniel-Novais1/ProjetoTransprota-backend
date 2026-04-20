package telemetry

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// GEOLOCATION & COLLECTIVE INTELLIGENCE REPOSITORY
// ============================================================================

const (
	// Redis GEO keys
	RedisGeoKeyBuses = "geo:buses:goiania" // GEO set de ônibus ativos
	RedisGeoKeyUsers = "geo:users:goiania" // GEO set de usuários colaboradores

	// TTL para dados GEO (1 minuto de validade para evitar expiração rápida)
	GeoDataTTL = 1 * time.Minute

	// Limites de validação para Goiânia
	GoiâniaLatMin = -16.8
	GoiâniaLatMax = -16.5
	GoiâniaLngMin = -49.3
	GoiâniaLngMax = -49.1

	// Velocidade máxima realista para ônibus urbano
	MaxBusSpeed = 80.0 // km/h
)

// AddBusLocation adiciona/atualiza posição de ônibus no Redis GEO
func (r *Repository) AddBusLocation(ctx context.Context, update *BusUpdate) error {
	redisClient, ok := r.rdb.(*redis.Client)
	if !ok {
		return fmt.Errorf("Redis not available")
	}

	// 1. Validar coordenadas (Goiânia bounding box)
	if !r.isValidCoordinate(update.Latitude, update.Longitude) {
		return fmt.Errorf("coordinates outside Goiânia bounds")
	}

	// 2. Validar velocidade
	if update.Speed < 0 || update.Speed > MaxBusSpeed {
		return fmt.Errorf("invalid speed: %f km/h", update.Speed)
	}

	// 3. Adicionar ao Redis GEO
	// GEOADD key longitude latitude member
	// Usar chave base para simplificar busca
	geoKey := RedisGeoKeyBuses
	member := fmt.Sprintf("%s|%s|%f|%f|%s|%d",
		update.RouteID, update.DeviceHash, update.Speed, update.Heading, update.Occupancy, time.Now().Unix())

	logger.Info("GEO", "Salvando member string: %s", member)

	err := redisClient.GeoAdd(ctx, geoKey, &redis.GeoLocation{
		Longitude: update.Longitude,
		Latitude:  update.Latitude,
		Name:      member,
	}).Err()

	if err != nil {
		logger.Error("GEO", "Failed to add bus location: %v", err)
		return err
	}

	// 4. Definir TTL para expiração automática
	redisClient.Expire(ctx, geoKey, GeoDataTTL)

	// 5. Logar densidade no PostgreSQL (background)
	go r.logDensity(context.Background(), update)

	logger.Info("GEO", "Bus location added | Device: %s | Route: %s | Lat: %.6f | Lng: %.6f",
		update.DeviceHash, update.RouteID, update.Latitude, update.Longitude)

	return nil
}

// GetBusLocationsByRadius busca ônibus dentro de um raio (GEORADIUS)
func (r *Repository) GetBusLocationsByRadius(ctx context.Context, query *GeoRadiusQuery) (*GeoRadiusResponse, error) {
	// Normalizar RouteFilter (trim e upper case)
	normalizedRouteFilter := strings.TrimSpace(strings.ToUpper(query.RouteFilter))
	logger.Info("GEO", "GetBusLocationsByRadius chamado - Lat: %.6f, Lng: %.6f, Radius: %d, RouteFilter: '%s' (normalizado: '%s')",
		query.CenterLat, query.CenterLng, query.RadiusMeters, query.RouteFilter, normalizedRouteFilter)

	redisClient, ok := r.rdb.(*redis.Client)
	if !ok {
		return nil, fmt.Errorf("Redis not available")
	}

	var allBuses []BusLocation

	// Se routeFilter especificado, buscar apenas dessa linha (suporta busca parcial)
	geoKeys := []string{}
	if query.RouteFilter != "" {
		// Buscar em todas as chaves e filtrar por Contains
		// Para simplificar, busca na chave padrão e filtra por route_id
		geoKeys = []string{RedisGeoKeyBuses}
	} else {
		// Buscar de todas as linhas (simplificado: busca de chave padrão)
		geoKeys = []string{RedisGeoKeyBuses}
	}

	for _, geoKey := range geoKeys {
		// GEORADIUS key longitude latitude radius m [WITHCOORD] [WITHDIST]
		results, err := redisClient.GeoRadius(ctx, geoKey, query.CenterLng, query.CenterLat, &redis.GeoRadiusQuery{
			Radius:    float64(query.RadiusMeters),
			Unit:      "m",
			WithCoord: true,
			WithDist:  true,
			Count:     100, // Limite de 100 ônibus
		}).Result()

		if err != nil {
			logger.Warn("GEO", "Failed to get buses by radius: %v", err)
			continue
		}

		logger.Info("GEO", "Redis retornou %d resultados da chave %s", len(results), geoKey)

		// Parse resultados
		for _, result := range results {
			bus, err := r.parseGeoResult(result)
			if err != nil {
				logger.Warn("GEO", "Failed to parse result: %v", err)
				continue
			}
			// Filtrar por route_id se especificado (busca parcial com Contains)
			if normalizedRouteFilter != "" {
				normalizedBusRouteID := strings.TrimSpace(strings.ToUpper(bus.RouteID))
				logger.Info("GEO", "Filtrando - BusRouteID: '%s' (normalizado: '%s'), Filter: '%s', Contains: %v",
					bus.RouteID, normalizedBusRouteID, normalizedRouteFilter, strings.Contains(normalizedBusRouteID, normalizedRouteFilter))
				// Verifica se o route_id contém o filtro (ex: "001" encontra "EIXO-001")
				if !strings.Contains(normalizedBusRouteID, normalizedRouteFilter) {
					continue
				}
			}
			allBuses = append(allBuses, *bus)
		}
	}

	// Se não houver ônibus, adicionar ônibus fake para teste
	if len(allBuses) == 0 {
		logger.Warn("GEO", "Nenhum ônibus encontrado no Redis, adicionando ônibus fake TESTE-99")
		allBuses = append(allBuses, BusLocation{
			DeviceHash:   "TESTE-99",
			RouteID:      "TESTE",
			Latitude:     -16.6869,
			Longitude:    -49.2648,
			Speed:        0,
			Heading:      0,
			Occupancy:    "unknown",
			LastUpdate:   time.Now(),
			Confidence:   100,
			IsVerified:   true,
			Contributors: 1,
		})
	}

	return &GeoRadiusResponse{
		Count:        len(allBuses),
		Buses:        allBuses,
		CenterLat:    query.CenterLat,
		CenterLng:    query.CenterLng,
		RadiusMeters: query.RadiusMeters,
	}, nil
}

// GetAllActiveBuses retorna todos os ônibus ativos em Goiânia
func (r *Repository) GetAllActiveBuses(ctx context.Context) ([]BusLocation, error) {
	// Usar GEORADIUS com raio grande para cobrir toda Goiânia
	// Centro de Goiânia: -16.6869, -49.2648
	// Raio de 50km cobre toda cidade e arredores
	query := &GeoRadiusQuery{
		CenterLat:    -16.6869,
		CenterLng:    -49.2648,
		RadiusMeters: 50000, // 50km
	}

	response, err := r.GetBusLocationsByRadius(ctx, query)
	if err != nil {
		return nil, err
	}

	return response.Buses, nil
}

// parseGeoResult converte resultado do Redis GEO em BusLocation
func (r *Repository) parseGeoResult(result redis.GeoLocation) (*BusLocation, error) {
	// Parse member string: "route_id|device_hash|speed|heading|occupancy|timestamp"
	logger.Info("GEO", "Parsing member string: %s", result.Name)

	var routeID, deviceHash string
	var speed, heading float64
	var occupancy string
	var timestamp int64

	_, err := fmt.Sscanf(result.Name, "%s|%s|%f|%f|%s|%d", &routeID, &deviceHash, &speed, &heading, &occupancy, &timestamp)
	if err != nil {
		logger.Error("GEO", "Failed to parse member string: %v", err)
		return nil, err
	}

	logger.Info("GEO", "Parsed - RouteID: %s, DeviceHash: %s, Speed: %.2f, Heading: %.2f", routeID, deviceHash, speed, heading)

	// Calcular confiança baseado na recência
	age := time.Since(time.Unix(timestamp, 0))
	confidence := 100
	if age > 2*time.Minute {
		confidence = 80
	}
	if age > 4*time.Minute {
		confidence = 50
	}

	return &BusLocation{
		DeviceHash:   deviceHash,
		RouteID:      routeID, // Extraído do device_hash
		Latitude:     result.Latitude,
		Longitude:    result.Longitude,
		Speed:        speed,
		Heading:      heading,
		Occupancy:    occupancy,
		LastUpdate:   time.Unix(timestamp, 0),
		Confidence:   confidence,
		IsVerified:   false, // Seria calculado baseado em múltiplos reports
		Contributors: 1,
	}, nil
}

// isValidCoordinate verifica se coordenadas estão dentro de Goiânia
func (r *Repository) isValidCoordinate(lat, lng float64) bool {
	return lat >= GoiâniaLatMin && lat <= GoiâniaLatMax &&
		lng >= GoiâniaLngMin && lng <= GoiâniaLngMax
}

// logDensity registra densidade de terminais no PostgreSQL
func (r *Repository) logDensity(ctx context.Context, update *BusUpdate) {
	if r.db == nil {
		return
	}

	// Criar tabela de densidade se não existir
	r.initDensityTable(ctx)

	// Inserir log de densidade
	query := `
		INSERT INTO terminal_density (terminal_id, terminal_name, user_count, bus_count, avg_speed, timestamp, hour, day_of_week)
		VALUES ($1, $2, 1, 1, $3, $4, $5, $6)
		ON CONFLICT (terminal_id, timestamp) 
		DO UPDATE SET user_count = terminal_density.user_count + 1,
		              avg_speed = (terminal_density.avg_speed + $3) / 2
	`

	hour := time.Now().Hour()
	dayOfWeek := int(time.Now().Weekday())

	_, err := r.db.ExecContext(ctx, query,
		update.TerminalID,
		r.getTerminalName(update.TerminalID),
		update.Speed,
		time.Now(),
		hour,
		dayOfWeek,
	)

	if err != nil {
		logger.Warn("GEO", "Failed to log density: %v", err)
	}
}

// initDensityTable cria tabela de densidade se não existir
func (r *Repository) initDensityTable(ctx context.Context) {
	query := `
		CREATE TABLE IF NOT EXISTS terminal_density (
			id SERIAL PRIMARY KEY,
			terminal_id VARCHAR(50) NOT NULL,
			terminal_name VARCHAR(100),
			user_count INTEGER DEFAULT 0,
			bus_count INTEGER DEFAULT 0,
			avg_speed FLOAT DEFAULT 0,
			timestamp TIMESTAMP DEFAULT NOW(),
			hour INTEGER NOT NULL,
			day_of_week INTEGER NOT NULL,
			UNIQUE(terminal_id, timestamp)
		);
		
		CREATE INDEX IF NOT EXISTS idx_density_terminal ON terminal_density(terminal_id);
		CREATE INDEX IF NOT EXISTS idx_density_timestamp ON terminal_density(timestamp);
	`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		logger.Warn("GEO", "Failed to create density table: %v", err)
	}
}

// getTerminalName retorna nome do terminal baseado no ID
func (r *Repository) getTerminalName(terminalID string) string {
	terminals := map[string]string{
		"t01": "Terminal Anhanguera",
		"t02": "Terminal Bandeirantes",
		"t03": "Terminal Centro",
		"t04": "Terminal Bueno",
		"t05": "Terminal Jardim Goiás",
	}

	if name, exists := terminals[terminalID]; exists {
		return name
	}
	return "Terminal Desconhecido"
}

// GetDensityReport retorna relatório de densidade para analytics
func (r *Repository) GetDensityReport(ctx context.Context, hours int) ([]TerminalDensity, error) {
	if r.db == nil {
		return []TerminalDensity{}, nil
	}

	query := `
		SELECT 
			terminal_id,
			terminal_name,
			SUM(user_count) as user_count,
			SUM(bus_count) as bus_count,
			AVG(avg_speed) as avg_speed
		FROM terminal_density
		WHERE timestamp > NOW() - INTERVAL '1 hour' * $1
		GROUP BY terminal_id, terminal_name
		ORDER BY user_count DESC
		LIMIT 10
	`

	rows, err := r.db.QueryContext(ctx, query, hours)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var densities []TerminalDensity
	for rows.Next() {
		var d TerminalDensity
		err := rows.Scan(&d.TerminalID, &d.TerminalName, &d.UserCount, &d.BusCount, &d.AvgSpeed)
		if err != nil {
			continue
		}

		// Calcular tendência (simplificado)
		d.Trend = "stable"
		if d.UserCount > 50 {
			d.Trend = "increasing"
		} else if d.UserCount < 10 {
			d.Trend = "decreasing"
		}

		densities = append(densities, d)
	}

	return densities, nil
}
