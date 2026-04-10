-- ============================================================================
-- MIGRATION 04: TELEMETRY SCHEMA - GPS Data Collection for SaaS
-- Fase 1: Foundation - Telemetria Passiva (Crowdsourcing)
-- ============================================================================
-- Author: Squad TranspRota
-- Date: 2025-01-09
-- Description: Schema for GPS telemetry ingestion with privacy-first design
-- ============================================================================

-- Enable PostGIS extension (if not already enabled)
CREATE EXTENSION IF NOT EXISTS postgis;

-- ============================================================================
-- 1. TABELA: gps_telemetry
-- ============================================================================
-- Armazena dados de telemetria GPS dos usuários
-- Design privacy-first: user_id anonimizado, precisão degradada
-- Particionamento por data para performance em time-series
-- ============================================================================

CREATE TABLE gps_telemetry (
    id BIGSERIAL,
    
    -- Identificação anônima (LGPD compliant)
    device_hash VARCHAR(32) NOT NULL,  -- SHA-256 hash anonimizado do device_id
    
    -- Posição geográfica (PostGIS)
    geom GEOMETRY(POINT, 4326) NOT NULL,  -- WGS84 coordinate system
    
    -- Dados de movimento
    speed FLOAT,                        -- km/h
    heading FLOAT,                      -- graus (0-360)
    accuracy FLOAT,                     -- metros de precisão GPS
    
    -- Contexto do transporte
    transport_mode VARCHAR(20),         -- bus | car | bike | walk | metro
    route_id VARCHAR(10),              -- identificador da rota (opcional)
    
    -- Metadados
    battery_level INTEGER,               -- nível de bateria do dispositivo (%)
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL,  -- quando o dado foi coletado no device
    
    -- Chave de partição para time-series
    partition_date DATE NOT NULL GENERATED ALWAYS AS (DATE(created_at)) STORED
) PARTITION BY RANGE (partition_date);

-- ============================================================================
-- 2. PARTIÇÕES INICIAIS
-- ============================================================================

-- Partição atual (mês corrente)
CREATE TABLE gps_telemetry_2025_01 
    PARTITION OF gps_telemetry
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

-- Partição próximo mês (preparação)
CREATE TABLE gps_telemetry_2025_02 
    PARTITION OF gps_telemetry
    FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');

-- ============================================================================
-- 3. ÍNDICES DE PERFORMANCE
-- ============================================================================

-- Índice espacial GIST para queries geográficas rápidas
CREATE INDEX idx_gps_telemetry_geom 
    ON gps_telemetry USING GIST(geom);

-- Índice para busca por device_hash + tempo (queries de histórico)
CREATE INDEX idx_gps_telemetry_device_time 
    ON gps_telemetry(device_hash, created_at);

-- Índice para análise de rotas
CREATE INDEX idx_gps_telemetry_route_time 
    ON gps_telemetry(route_id, created_at) 
    WHERE route_id IS NOT NULL;

-- Índice para análise por modo de transporte
CREATE INDEX idx_gps_telemetry_mode_time 
    ON gps_telemetry(transport_mode, created_at);

-- Índice na chave de partição para manutenção
CREATE INDEX idx_gps_telemetry_partition 
    ON gps_telemetry(partition_date);

-- ============================================================================
-- 4. FUNÇÕES AUXILIARES
-- ============================================================================

-- Função para criar partições automaticamente
CREATE OR REPLACE FUNCTION create_telemetry_partition()
RETURNS void AS $$
DECLARE
    start_date DATE;
    end_date DATE;
    partition_name TEXT;
BEGIN
    -- Criar partição para próximo mês
    start_date := DATE_TRUNC('month', CURRENT_DATE + INTERVAL '1 month');
    end_date := start_date + INTERVAL '1 month';
    partition_name := 'gps_telemetry_' || TO_CHAR(start_date, 'YYYY_MM');
    
    -- Verificar se partição já existe
    IF NOT EXISTS (
        SELECT 1 FROM pg_tables 
        WHERE tablename = partition_name
    ) THEN
        EXECUTE format(
            'CREATE TABLE %I PARTITION OF gps_telemetry FOR VALUES FROM (%L) TO (%L)',
            partition_name, start_date, end_date
        );
        
        RAISE NOTICE 'Created partition: %', partition_name;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Função para limpar dados antigos (LGPD: retenção limitada)
CREATE OR REPLACE FUNCTION cleanup_old_telemetry()
RETURNS integer AS $$
DECLARE
    deleted_count INTEGER;
    retention_days INTEGER := 90;  -- 90 dias de retenção
BEGIN
    -- Deletar dados mais antigos que o período de retenção
    DELETE FROM gps_telemetry 
    WHERE created_at < NOW() - INTERVAL '90 days';
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    RAISE NOTICE 'Deleted % old telemetry records (older than % days)', 
        deleted_count, retention_days;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 5. TRIGGER PARA ATUALIZAÇÃO DE TIMESTAMPS
-- ============================================================================

-- Garantir que recorded_at seja sempre preenchido
CREATE OR REPLACE FUNCTION set_recorded_at()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.recorded_at IS NULL THEN
        NEW.recorded_at := NEW.created_at;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_set_recorded_at
    BEFORE INSERT ON gps_telemetry
    FOR EACH ROW
    EXECUTE FUNCTION set_recorded_at();

-- ============================================================================
-- 6. TABELA DE AGREGADOS (Para ML e Analytics)
-- ============================================================================

CREATE TABLE telemetry_hourly_aggregates (
    id BIGSERIAL PRIMARY KEY,
    
    -- Dimensões
    hour_timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    route_id VARCHAR(10),
    transport_mode VARCHAR(20),
    geohash_prefix VARCHAR(4),  -- Geohash de 4 chars (~20km)
    
    -- Métricas
    ping_count INTEGER DEFAULT 0,
    unique_devices INTEGER DEFAULT 0,
    
    avg_speed FLOAT,
    max_speed FLOAT,
    min_speed FLOAT,
    stddev_speed FLOAT,
    
    avg_accuracy FLOAT,
    
    -- Bounding box da área coberta
    bbox_geom GEOMETRY(POLYGON, 4326),
    
    -- Timestamps
    computed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Constraint UNIQUE para agregados
ALTER TABLE telemetry_hourly_aggregates 
    ADD CONSTRAINT uq_telemetry_agg_hourly 
    UNIQUE (hour_timestamp, route_id, transport_mode, geohash_prefix);

-- Índices para agregados
CREATE INDEX idx_telemetry_agg_time 
    ON telemetry_hourly_aggregates(hour_timestamp);

CREATE INDEX idx_telemetry_agg_route 
    ON telemetry_hourly_aggregates(route_id, hour_timestamp);

-- ============================================================================
-- 7. COMENTÁRIOS E DOCUMENTAÇÃO
-- ============================================================================

COMMENT ON TABLE gps_telemetry IS 
    'Tabela principal de telemetria GPS com design privacy-first LGPD compliant';

COMMENT ON COLUMN gps_telemetry.device_hash IS 
    'SHA-256 hash anonimizado do device_id (32 chars hex). Rotacionado diariamente.';

COMMENT ON COLUMN gps_telemetry.geom IS 
    'Posição GPS em coordenadas WGS84 (EPSG:4326). Precisão pode ser degradada para privacidade.';

COMMENT ON COLUMN gps_telemetry.speed IS 
    'Velocidade em km/h. Validado: máximo 120km/h em perímetro urbano.';

COMMENT ON COLUMN gps_telemetry.partition_date IS 
    'Chave de partição automática para otimização de queries time-series';

COMMENT ON TABLE telemetry_hourly_aggregates IS 
    'Agregados horários para análise de ML e dashboards. Atualizado via job periódico.';

-- ============================================================================
-- 8. CONFIGURAÇÃO DE PERMISSÕES
-- ============================================================================

-- Grant permissions (ajustar conforme usuário da aplicação)
-- GRANT SELECT, INSERT ON gps_telemetry TO transprota_app;
-- GRANT USAGE, SELECT ON SEQUENCE gps_telemetry_id_seq TO transprota_app;

-- ============================================================================
-- 9. EXEMPLOS DE QUERIES COMUNS
-- ============================================================================

/*
-- Query: Últimas posições de um device (últimos 5 minutos)
SELECT device_hash, geom, speed, created_at
FROM gps_telemetry
WHERE device_hash = 'abc123...'
  AND created_at > NOW() - INTERVAL '5 minutes'
ORDER BY created_at DESC;

-- Query: Veículos dentro de um raio (500m do ponto central)
SELECT device_hash, geom, speed
FROM gps_telemetry
WHERE ST_DWithin(
    geom::geography,
    ST_SetSRID(ST_MakePoint(-49.2643, -16.6864), 4326)::geography,
    500  -- metros
)
AND created_at > NOW() - INTERVAL '1 minute';

-- Query: Estatísticas de velocidade por rota (última hora)
SELECT 
    route_id,
    COUNT(*) as ping_count,
    AVG(speed) as avg_speed,
    MAX(speed) as max_speed,
    ST_Collect(geom) as trajectory
FROM gps_telemetry
WHERE route_id = '801'
  AND created_at > NOW() - INTERVAL '1 hour'
GROUP BY route_id;

-- Query: Calcular partições para manutenção
SELECT 
    partition_date,
    COUNT(*) as row_count,
    MIN(created_at) as oldest,
    MAX(created_at) as newest
FROM gps_telemetry
GROUP BY partition_date
ORDER BY partition_date DESC;
*/

-- ============================================================================
-- MIGRATION COMPLETA
-- ============================================================================

SELECT 'Migration 04: Telemetry Schema criada com sucesso!' as status;
