-- ============================================================================
-- MIGRATION 08: CORREÇÃO DE DISCREPÂNCIA - Platform e AppVersion
-- Autor: Squad TranspRota
-- Data: 2026-04-13
-- ============================================================================
-- Adiciona campos Platform e AppVersion que existem no modelo Go
-- mas não estavam na tabela SQL original
-- ============================================================================

-- Adicionar campo platform
ALTER TABLE gps_telemetry 
ADD COLUMN IF NOT EXISTS platform VARCHAR(20);

-- Adicionar campo app_version
ALTER TABLE gps_telemetry 
ADD COLUMN IF NOT EXISTS app_version VARCHAR(20);

-- Adicionar comentários
COMMENT ON COLUMN gps_telemetry.platform IS 'Plataforma do dispositivo: android | ios';
COMMENT ON COLUMN gps_telemetry.app_version IS 'Versão do app (ex: 4.0.0)';

-- Adicionar índice para análise por plataforma
CREATE INDEX IF NOT EXISTS idx_gps_telemetry_platform 
    ON gps_telemetry(platform, created_at) 
    WHERE platform IS NOT NULL;

-- Atualizar estatísticas
ANALYZE gps_telemetry;

SELECT 'Migration 08: Campos Platform e AppVersion adicionados com sucesso!' as status;
