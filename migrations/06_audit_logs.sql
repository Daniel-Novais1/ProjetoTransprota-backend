-- Migration 06: Audit Logs System
-- Cria tabela para rastro total de alterações no sistema
-- Autor: Squad TranspRota
-- Data: 2026-04-10

-- Tabela de Audit Logs
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    actor_type VARCHAR(20) NOT NULL CHECK (actor_type IN ('user', 'device', 'system')),
    actor_id VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(255),
    ip_address VARCHAR(45),
    user_agent TEXT,
    payload JSONB,
    old_value JSONB,
    new_value JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Índices para queries de audit log
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor ON audit_logs(actor_type, actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_created ON audit_logs(actor_type, actor_id, created_at DESC);

-- Comentários para documentação
COMMENT ON TABLE audit_logs IS 'Audit log para rastro total de alterações no sistema';
COMMENT ON COLUMN audit_logs.actor_type IS 'Tipo de ator: user, device, system';
COMMENT ON COLUMN audit_logs.actor_id IS 'ID do ator (user_id, device_id, ou nome do sistema)';
COMMENT ON COLUMN audit_logs.action IS 'Ação executada (create, update, delete, cleanup, etc)';
COMMENT ON COLUMN audit_logs.resource_type IS 'Tipo de recurso afetado (geofence, gps_telemetry, etc)';
COMMENT ON COLUMN audit_logs.payload IS 'Payload completo da requisição';
COMMENT ON COLUMN audit_logs.old_value IS 'Valor anterior (para updates)';
COMMENT ON COLUMN audit_logs.new_value IS 'Valor novo (para updates)';

-- Função para registrar audit log automaticamente
CREATE OR REPLACE FUNCTION log_audit(
    p_actor_type VARCHAR,
    p_actor_id VARCHAR,
    p_action VARCHAR,
    p_resource_type VARCHAR,
    p_resource_id VARCHAR,
    p_ip_address VARCHAR,
    p_user_agent TEXT,
    p_payload JSONB,
    p_old_value JSONB,
    p_new_value JSONB
) RETURNS BIGINT AS $$
DECLARE
    log_id BIGINT;
BEGIN
    INSERT INTO audit_logs (
        actor_type, actor_id, action, resource_type, resource_id,
        ip_address, user_agent, payload, old_value, new_value
    ) VALUES (
        p_actor_type, p_actor_id, p_action, p_resource_type, p_resource_id,
        p_ip_address, p_user_agent, p_payload, p_old_value, p_new_value
    ) RETURNING id INTO log_id;

    RETURN log_id;
END;
$$ LANGUAGE plpgsql;

-- Exemplo de uso:
-- SELECT log_audit('user', 'admin', 'create', 'geofence', '1', '127.0.0.1', 'Mozilla/5.0...', '{"nome": "Terminal Centro"}'::jsonb, NULL, '{"id": 1, "nome": "Terminal Centro"}'::jsonb);

-- Estatísticas do PostgreSQL para otimização
ANALYZE audit_logs;
