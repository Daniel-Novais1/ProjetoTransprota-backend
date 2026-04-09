-- ============================================================================
-- MIGRATIONS: Sistema de Rotas e Planejador de Viagem
-- TranspRota - Monitoramento Inteligente do Transporte de Goiânia
-- ============================================================================

-- ============================================================================
-- TABELA 1: LINHAS DE ÔNIBUS
-- ============================================================================
-- Armazena as linhas de transporte (101, 102, etc)
-- Cada linha tem um número único, nome e status operacional
-- ============================================================================

CREATE TABLE IF NOT EXISTS linhas_onibus (
    id SERIAL PRIMARY KEY,
    numero_linha VARCHAR(20) NOT NULL UNIQUE,
    nome_linha VARCHAR(255) NOT NULL,
    descricao TEXT,
    status VARCHAR(20) DEFAULT 'ativa' CHECK (status IN ('ativa', 'inativa', 'suspensa')),
    empresa VARCHAR(100),
    tipo_servico VARCHAR(50) DEFAULT 'regular' CHECK (tipo_servico IN ('regular', 'executivo', 'especial')),
    criado_em TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    atualizado_em TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Índices para performance
CREATE INDEX IF NOT EXISTS idx_linhas_numero ON linhas_onibus(numero_linha);
CREATE INDEX IF NOT EXISTS idx_linhas_status ON linhas_onibus(status);

-- ============================================================================
-- TABELA 2: ITINERÁRIOS (Sequência de Paradas por Linha)
-- ============================================================================
-- Define a ordem das paradas para cada linha
-- A coluna "ordem_parada" determina a sequência (1, 2, 3, ...)
-- tempo_estimado_anterior_minutos: tempo entre esta parada e a anterior
-- ============================================================================

CREATE TABLE IF NOT EXISTS itinerarios (
    id SERIAL PRIMARY KEY,
    linha_id INTEGER NOT NULL,
    parada_id INTEGER NOT NULL,
    ordem_parada INTEGER NOT NULL,
    tempo_estimado_anterior_minutos INTEGER DEFAULT NULL,
    eh_ponto_integracao BOOLEAN DEFAULT FALSE,
    criado_em TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    FOREIGN KEY (linha_id) REFERENCES linhas_onibus(id) ON DELETE CASCADE,
    FOREIGN KEY (parada_id) REFERENCES locations(id) ON DELETE RESTRICT,
    
    -- Garantir que não há duplicação de ordem para a mesma linha
    UNIQUE(linha_id, ordem_parada)
);

-- Índices para otimizar buscas
CREATE INDEX IF NOT EXISTS idx_itinerarios_linha ON itinerarios(linha_id);
CREATE INDEX IF NOT EXISTS idx_itinerarios_parada ON itinerarios(parada_id);
CREATE INDEX IF NOT EXISTS idx_itinerarios_ordem ON itinerarios(linha_id, ordem_parada);
CREATE INDEX IF NOT EXISTS idx_integracao ON itinerarios(eh_ponto_integracao);

-- ============================================================================
-- TABELA 3: HISTÓRICO DE POSIÇÕES (Para análise de atrasos)
-- ============================================================================
-- Armazena o histórico de onde cada ônibus estava em cada parada
-- Útil para calcular atrasos reais vs estimados
-- ============================================================================

CREATE TABLE IF NOT EXISTS historico_posicoes (
    id SERIAL PRIMARY KEY,
    linha_id INTEGER NOT NULL,
    bus_id VARCHAR(50) NOT NULL,
    parada_id INTEGER NOT NULL,
    tempo_chegada TIMESTAMP NOT NULL,
    tempo_saida TIMESTAMP,
    atraso_minutos INTEGER DEFAULT 0,
    lotacao_percentage INTEGER CHECK (lotacao_percentage >= 0 AND lotacao_percentage <= 100),
    criado_em TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (linha_id) REFERENCES linhas_onibus(id) ON DELETE CASCADE,
    FOREIGN KEY (parada_id) REFERENCES locations(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_historico_linha ON historico_posicoes(linha_id);
CREATE INDEX IF NOT EXISTS idx_historico_parada ON historico_posicoes(parada_id);
CREATE INDEX IF NOT EXISTS idx_historico_data ON historico_posicoes(tempo_chegada);

-- ============================================================================
-- DADOS DE EXEMPLO: Linhas Reais de Goiânia
-- ============================================================================

-- Limpar exemplos antigos se existirem
DELETE FROM itinerarios WHERE linha_id IN (
    SELECT id FROM linhas_onibus WHERE numero_linha IN ('101', '102', '103')
);
DELETE FROM linhas_onibus WHERE numero_linha IN ('101', '102', '103');

-- Inserir linhas de exemplo
INSERT INTO linhas_onibus (numero_linha, nome_linha, descricao, empresa, tipo_servico) VALUES
    ('101', 'Eixo Anhanguera', 'Terminal Centro → Eixo Anhanguera → Terminal Novo Mundo', 'RMTC', 'regular'),
    ('102', 'Setor Comercial Sul', 'Terminal Centro → Setor Comercial S. → UFG Campus Samambaia', 'RMTC', 'regular'),
    ('103', 'Integração Bíblia', 'Terminal Padre Pelágio → Terminal Praça da Bíblia → Terminal Centro', 'RMTC', 'regular')
ON CONFLICT (numero_linha) DO NOTHING;

-- ============================================================================
-- ITINERÁRIO EXEMPLO 1: Linha 101 - Eixo Anhanguera
-- ============================================================================
-- Assumindo que já existem esses pontos na tabela locations:
-- 1=Terminal Centro, 2=Eixo Anhanguera, 3=Terminal Novo Mundo
-- Ajuste os parada_id conforme necessário depois de verificar a tabela locations

INSERT INTO itinerarios (linha_id, parada_id, ordem_parada, tempo_estimado_anterior_minutos, eh_ponto_integracao)
SELECT 
    (SELECT id FROM linhas_onibus WHERE numero_linha = '101'),
    id,
    ordem,
    tempo
FROM (VALUES 
    ((SELECT id FROM locations WHERE name = 'Terminal Centro'), 1, NULL, TRUE),
    ((SELECT id FROM locations WHERE name = 'Eixo Anhanguera'), 2, 10, FALSE),
    ((SELECT id FROM locations WHERE name = 'Terminal Novo Mundo'), 3, 12, TRUE)
) AS t(parada_id, ordem, tempo, eh_integracao)
ON CONFLICT (linha_id, ordem_parada) DO NOTHING;

-- ============================================================================
-- ITINERÁRIO EXEMPLO 2: Linha 102 - Setor Comercial Sul
-- ============================================================================

INSERT INTO itinerarios (linha_id, parada_id, ordem_parada, tempo_estimado_anterior_minutos, eh_ponto_integracao)
SELECT 
    (SELECT id FROM linhas_onibus WHERE numero_linha = '102'),
    id,
    ordem,
    tempo
FROM (VALUES 
    ((SELECT id FROM locations WHERE name = 'Terminal Centro'), 1, NULL, TRUE),
    ((SELECT id FROM locations WHERE name = 'Setor Comercial Sul'), 2, 8, FALSE),
    ((SELECT id FROM locations WHERE name = 'UFG Campus Samambaia'), 3, 15, FALSE)
) AS t(parada_id, ordem, tempo, eh_integracao)
ON CONFLICT (linha_id, ordem_parada) DO NOTHING;

-- ============================================================================
-- ITINERÁRIO EXEMPLO 3: Linha 103 - Ponto de Integração
-- ============================================================================

INSERT INTO itinerarios (linha_id, parada_id, ordem_parada, tempo_estimado_anterior_minutos, eh_ponto_integracao)
SELECT 
    (SELECT id FROM linhas_onibus WHERE numero_linha = '103'),
    id,
    ordem,
    tempo
FROM (VALUES 
    ((SELECT id FROM locations WHERE name = 'Terminal Padre Pelágio'), 1, NULL, TRUE),
    ((SELECT id FROM locations WHERE name = 'Terminal Praça da Bíblia'), 2, 10, TRUE),
    ((SELECT id FROM locations WHERE name = 'Terminal Centro'), 3, 12, TRUE)
) AS t(parada_id, ordem, tempo, eh_integracao)
ON CONFLICT (linha_id, ordem_parada) DO NOTHING;

-- ============================================================================
-- VIEWS: Consultas Úteis para o Planejador de Viagem
-- ============================================================================

-- View 1: Rotas Completas (Linha com todas as paradas em ordem)
CREATE OR REPLACE VIEW v_rotas_completas AS
SELECT 
    l.id as linha_id,
    l.numero_linha,
    l.nome_linha,
    l.descricao,
    l.status,
    i.ordem_parada,
    i.parada_id,
    loc.name as nome_parada,
    loc.latitude,
    loc.longitude,
    i.tempo_estimado_anterior_minutos,
    i.eh_ponto_integracao,
    LAG(loc.name) OVER (PARTITION BY l.id ORDER BY i.ordem_parada) as parada_anterior
FROM linhas_onibus l
JOIN itinerarios i ON l.id = i.linha_id
JOIN locations loc ON i.parada_id = loc.id
ORDER BY l.id, i.ordem_parada;

-- View 2: Pontos de Integração (onde o usuário pode trocar de linha)
CREATE OR REPLACE VIEW v_pontos_integracao AS
SELECT DISTINCT
    loc.id,
    loc.name as nome_parada,
    loc.latitude,
    loc.longitude,
    STRING_AGG(l.numero_linha || ' - ' || l.nome_linha, ', ') as linhas_disponiveis,
    COUNT(DISTINCT l.id) as quantidade_linhas
FROM locations loc
JOIN itinerarios i ON loc.id = i.parada_id
JOIN linhas_onibus l ON i.linha_id = l.id
WHERE i.eh_ponto_integracao = TRUE AND l.status = 'ativa'
GROUP BY loc.id, loc.name, loc.latitude, loc.longitude;

-- View 3: Próximas Paradas de uma Linha (útil para verificar itinerário em tempo real)
CREATE OR REPLACE VIEW v_proximas_paradas AS
SELECT 
    l.id as linha_id,
    l.numero_linha,
    l.nome_linha,
    i.ordem_parada,
    loc.id as parada_id,
    loc.name as nome_parada,
    loc.latitude,
    loc.longitude,
    i.tempo_estimado_anterior_minutos,
    COALESCE(LAG(i.tempo_estimado_anterior_minutos) OVER (PARTITION BY l.id ORDER BY i.ordem_parada), 0) as tempo_acumulado_minutos
FROM linhas_onibus l
JOIN itinerarios i ON l.id = i.linha_id
JOIN locations loc ON i.parada_id = loc.id
WHERE l.status = 'ativa'
ORDER BY l.id, i.ordem_parada;

-- ============================================================================
-- PERMISSÕES E FINALIZAÇÕES
-- ============================================================================

COMMENT ON TABLE linhas_onibus IS 'Catálogo de linhas de transporte em operação';
COMMENT ON TABLE itinerarios IS 'Sequência de paradas para cada linha (ordem importa!)';
COMMENT ON TABLE historico_posicoes IS 'Histórico de passagens por paradas para análise de atrasos';
COMMENT ON COLUMN itinerarios.eh_ponto_integracao IS 'TRUE se neste ponto o usuário pode trocar de linha';

-- Verificar resultado final
SELECT 'Schema de Rotas criado com sucesso!' as status;
SELECT COUNT(*) as total_linhas FROM linhas_onibus;
SELECT COUNT(*) as total_itinerarios FROM itinerarios;
