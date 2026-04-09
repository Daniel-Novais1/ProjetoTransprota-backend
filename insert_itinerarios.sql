-- ============================================================================
-- INSERINDO PARADAS FALTANTES E ITINERÁRIOS
-- ============================================================================

-- 1. Inserir paradas faltantes se não existirem
INSERT INTO locations (name, latitude, longitude) VALUES
    ('Setor Comercial Sul', -15.7900, -47.8800),
    ('UFG Campus Samambaia', -16.0000, -48.9500),
    ('Vila Pedroso', -16.686, -49.264)  -- Coordenadas aproximadas de Vila Pedroso, Goiânia
ON CONFLICT (name) DO NOTHING;

-- 2. Buscar IDs das linhas (já inseridas no migration anterior)
-- Vamos verificar: 101, 102, 103

-- 3. Inserir itinerários para a Linha 101 (Eixo Anhanguera)
INSERT INTO itinerarios (linha_id, parada_id, ordem_parada, tempo_estimado_anterior_minutos, eh_ponto_integracao)
VALUES 
    ((SELECT id FROM linhas_onibus WHERE numero_linha = '101'), 7, 1, NULL, TRUE),      -- Terminal Centro
    ((SELECT id FROM linhas_onibus WHERE numero_linha = '101'), 8, 2, 10, FALSE),      -- Eixo Anhanguera
    ((SELECT id FROM linhas_onibus WHERE numero_linha = '101'), 1, 3, 12, TRUE)       -- Terminal Novo Mundo
ON CONFLICT (linha_id, ordem_parada) DO NOTHING;

-- 4. Inserir itinerários para a Linha 102 (Setor Comercial Sul)
INSERT INTO itinerarios (linha_id, parada_id, ordem_parada, tempo_estimado_anterior_minutos, eh_ponto_integracao)
VALUES 
    ((SELECT id FROM linhas_onibus WHERE numero_linha = '102'), 7, 1, NULL, TRUE),      -- Terminal Centro
    ((SELECT id FROM linhas_onibus WHERE numero_linha = '102'), (SELECT id FROM locations WHERE name = 'Setor Comercial Sul'), 2, 8, FALSE),  -- Setor Comercial Sul
    ((SELECT id FROM linhas_onibus WHERE numero_linha = '102'), (SELECT id FROM locations WHERE name = 'UFG Campus Samambaia'), 3, 15, FALSE) -- UFG
ON CONFLICT (linha_id, ordem_parada) DO NOTHING;

-- 5. Inserir itinerários para a Linha 103 (Pontos de Integração)
INSERT INTO itinerarios (linha_id, parada_id, ordem_parada, tempo_estimado_anterior_minutos, eh_ponto_integracao)
VALUES 
    ((SELECT id FROM linhas_onibus WHERE numero_linha = '103'), 3, 1, NULL, TRUE),      -- Terminal Padre Pelágio
    ((SELECT id FROM linhas_onibus WHERE numero_linha = '103'), 2, 2, 10, TRUE),       -- Terminal Praça da Bíblia
    ((SELECT id FROM linhas_onibus WHERE numero_linha = '103'), 7, 3, 12, TRUE)        -- Terminal Centro
ON CONFLICT (linha_id, ordem_parada) DO NOTHING;

-- 6. Inserir linha 104 (Vila Pedroso → UFG)
INSERT INTO linhas_onibus (numero_linha, nome_linha, descricao, status, empresa, tipo_servico, created_at, updated_at)
VALUES ('104', 'Vila Pedroso - UFG', 'Linha direta Vila Pedroso para Universidade Federal de Goiás', 'ativa', 'TranspRota', 'regular', NOW(), NOW())
ON CONFLICT (numero_linha) DO NOTHING;

-- 7. Inserir itinerários para a Linha 104 (Vila Pedroso → UFG)
INSERT INTO itinerarios (linha_id, parada_id, ordem_parada, tempo_estimado_anterior_minutos, eh_ponto_integracao)
VALUES 
    ((SELECT id FROM linhas_onibus WHERE numero_linha = '104'), (SELECT id FROM locations WHERE name = 'Vila Pedroso'), 1, NULL, FALSE),  -- Vila Pedroso
    ((SELECT id FROM linhas_onibus WHERE numero_linha = '104'), 7, 2, 25, TRUE),      -- Terminal Centro (ponto de integração)
    ((SELECT id FROM linhas_onibus WHERE numero_linha = '104'), (SELECT id FROM locations WHERE name = 'UFG Campus Samambaia'), 3, 30, FALSE)  -- UFG
ON CONFLICT (linha_id, ordem_parada) DO NOTHING;

-- ============================================================================
-- CONSULTAS DE VERIFICAÇÃO
-- ============================================================================

-- Verificar dados inseridos
SELECT '--- Linhas Cadastradas ---' as info;
SELECT numero_linha, nome_linha, status FROM linhas_onibus ORDER BY numero_linha;

SELECT '--- Itinerários por Linha ---' as info;
SELECT 
    l.numero_linha,
    i.ordem_parada,
    loc.name as parada,
    i.tempo_estimado_anterior_minutos,
    i.eh_ponto_integracao
FROM linhas_onibus l
JOIN itinerarios i ON l.id = i.linha_id
JOIN locations loc ON i.parada_id = loc.id
ORDER BY l.numero_linha, i.ordem_parada;

SELECT '--- View: Rotas Completas ---' as info;
SELECT numero_linha, nome_linha, ordem_parada, nome_parada, tempo_estimado_anterior_minutos FROM v_rotas_completas LIMIT 15;

SELECT '--- View: Pontos de Integração ---' as info;
SELECT nome_parada, quantidade_linhas, linhas_disponiveis FROM v_pontos_integracao;
