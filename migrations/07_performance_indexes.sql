-- ============================================================================
-- MIGRATION 07: ÍNDICES DE PERFORMANCE ADICIONAIS
-- Autor: Squad TranspRota
-- Data: 2026-04-10
-- ============================================================================
-- Índices adicionais para otimizar queries de time-series e análise

-- Índice para queries time-series recentes (últimos dados)
CREATE INDEX IF NOT EXISTS idx_gps_telemetry_time_recent
    ON gps_telemetry(created_at DESC);

-- Índice para análise de velocidade por modo de transporte
CREATE INDEX IF NOT EXISTS idx_gps_telemetry_speed_mode
    ON gps_telemetry(speed, transport_mode)
    WHERE speed IS NOT NULL;

-- Índice forense para audit logs (investigação por IP)
CREATE INDEX IF NOT EXISTS idx_audit_logs_ip
    ON audit_logs(ip_address, created_at DESC);

-- Comentários para documentação
COMMENT ON INDEX idx_gps_telemetry_time_recent IS 'Índice para queries de dados recentes (ORDER BY created_at DESC)';
COMMENT ON INDEX idx_gps_telemetry_speed_mode IS 'Índice para análise de velocidade por modo de transporte';
COMMENT ON INDEX idx_audit_logs_ip IS 'Índice forense para investigação por IP';

-- Estatísticas do PostgreSQL para otimização
ANALYZE gps_telemetry;
ANALYZE audit_logs;
