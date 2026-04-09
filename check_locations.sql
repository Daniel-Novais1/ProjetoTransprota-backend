-- ============================================================================
-- CORRIGINDO: Inserir Itinerários Manualmente
-- ============================================================================

-- Primeiro, vamos verificar os IDs das paradas existentes
SELECT id, name FROM locations WHERE name IN ('Terminal Centro', 'Eixo Anhanguera', 'Terminal Novo Mundo', 
                                               'Setor Comercial Sul', 'UFG Campus Samambaia',
                                               'Terminal Padre Pelágio', 'Terminal Praça da Bíblia') LIMIT 20;
