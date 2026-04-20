-- Migration 05: Geofencing Schema
-- Cria tabelas para cercas eletrônicas e alertas de geofencing
-- Autor: Squad TranspRota
-- Data: 2026-04-10

-- Tabela de Geofences (Cercas Eletrônicas)
CREATE TABLE IF NOT EXISTS geofences (
    id BIGSERIAL PRIMARY KEY,
    nome VARCHAR(255) NOT NULL,
    tipo VARCHAR(20) NOT NULL CHECK (tipo IN ('Terminal', 'Rota')),
    polygon GEOMETRY(POLYGON, 4326) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Índice espacial GIST para queries rápidas de ST_Contains
CREATE INDEX IF NOT EXISTS idx_geofences_polygon ON geofences USING GIST(polygon);

-- Índice para busca por tipo
CREATE INDEX IF NOT EXISTS idx_geofences_tipo ON geofences(tipo);

-- Tabela de Alertas de Geofencing
CREATE TABLE IF NOT EXISTS geofence_alerts (
    id BIGSERIAL PRIMARY KEY,
    device_hash VARCHAR(32) NOT NULL,
    geofence_id BIGINT NOT NULL REFERENCES geofences(id) ON DELETE CASCADE,
    geofence_nome VARCHAR(255) NOT NULL,
    estado VARCHAR(10) NOT NULL CHECK (estado IN ('In', 'Out')),
    lat FLOAT NOT NULL,
    lng FLOAT NOT NULL,
    ocorrido_em TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Índices para queries de alertas recentes
CREATE INDEX IF NOT EXISTS idx_geofence_alerts_device ON geofence_alerts(device_hash);
CREATE INDEX IF NOT EXISTS idx_geofence_alerts_geofence ON geofence_alerts(geofence_id);
CREATE INDEX IF NOT EXISTS idx_geofence_alerts_ocorrido ON geofence_alerts(ocorrido_em DESC);
CREATE INDEX IF NOT EXISTS idx_geofence_alerts_estado ON geofence_alerts(estado);

-- Índice composto para performance
CREATE INDEX IF NOT EXISTS idx_geofence_alerts_device_ocorrido ON geofence_alerts(device_hash, ocorrido_em DESC);

-- Inserir geofence de exemplo: Terminal Centro (Goiânia)
-- Polígono aproximado ao redor do centro de Goiânia
INSERT INTO geofences (nome, tipo, polygon)
VALUES (
    'Terminal Centro',
    'Terminal',
    ST_GeomFromText('POLYGON((-49.255 -16.695, -49.255 -16.675, -49.235 -16.675, -49.235 -16.695, -49.255 -16.695))', 4326)
)
ON CONFLICT (nome) DO NOTHING;

-- Inserir geofence de exemplo: Rota 801 (Eixo Anhanguera)
-- Polígono simplificado ao longo do Eixo Anhanguera
INSERT INTO geofences (nome, tipo, polygon)
VALUES (
    'Rota 801 - Eixo Anhanguera',
    'Rota',
    ST_GeomFromText('POLYGON((-49.28 -16.69, -49.28 -16.66, -49.22 -16.66, -49.22 -16.69, -49.28 -16.69))', 4326)
)
ON CONFLICT (nome) DO NOTHING;

-- Comentários para documentação
COMMENT ON TABLE geofences IS 'Cercas eletrônicas para monitoramento de veículos';
COMMENT ON COLUMN geofences.polygon IS 'Polígono geográfico usando SRID 4326 (WGS84)';
COMMENT ON TABLE geofence_alerts IS 'Alertas de geofencing disparados quando veículos saem/entram em áreas';
COMMENT ON COLUMN geofence_alerts.estado IS 'Estado: In (dentro) ou Out (fora) do geofence';

-- Estatísticas do PostgreSQL para otimização
ANALYZE geofences;
ANALYZE geofence_alerts;
