-- ============================================================================
-- SCHEMA: Sistema de Rotas - TranspRota
-- Tabelas: pontos_parada, linhas_onibus, itinerarios
-- ============================================================================

-- ============================================================================
-- 1. TABELA: PONTOS_PARADA
-- ============================================================================
-- Armazena coordenadas geográficas de todos os pontos de parada
-- Essencial para geolocalização e cálculo de distâncias
-- ============================================================================

CREATE TABLE IF NOT EXISTS pontos_parada (
    id SERIAL PRIMARY KEY,
    nome VARCHAR(255) NOT NULL UNIQUE,
    latitude DECIMAL(10, 8) NOT NULL,
    longitude DECIMAL(11, 8) NOT NULL,
    tipo VARCHAR(50) DEFAULT 'parada' CHECK (tipo IN ('parada', 'terminal', 'integracao')),
    criado_em TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Índice para busca rápida por nome (autocomplete do frontend)
CREATE INDEX IF NOT EXISTS idx_pontos_parada_nome ON pontos_parada(nome);

COMMENT ON TABLE pontos_parada IS 'Coordenadas geográficas de paradas e terminais';
COMMENT ON COLUMN pontos_parada.tipo IS 'Classifica parada como simples, terminal ou ponto integração';

-- ============================================================================
-- 2. TABELA: LINHAS_ONIBUS
-- ============================================================================
-- Catálogo de linhas de transporte operacionais
-- Uma linha 101 sempre tem o mesmo número, mas rota pode ter variações
-- ============================================================================

CREATE TABLE IF NOT EXISTS linhas_onibus (
    id SERIAL PRIMARY KEY,
    numero_linha VARCHAR(20) NOT NULL UNIQUE,
    nome_linha VARCHAR(255) NOT NULL,
    descricao TEXT,
    status VARCHAR(20) DEFAULT 'ativa' CHECK (status IN ('ativa', 'inativa', 'suspensa')),
    empresa VARCHAR(100),
    tipo_servico VARCHAR(50) DEFAULT 'regular' CHECK (tipo_servico IN ('regular', 'executivo', 'especial')),
    criado_em TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Índices para queries de listagem
CREATE INDEX IF NOT EXISTS idx_linhas_numero ON linhas_onibus(numero_linha);
CREATE INDEX IF NOT EXISTS idx_linhas_status ON linhas_onibus(status);

COMMENT ON TABLE linhas_onibus IS 'Catálogo de linhas de ônibus operacionais';
COMMENT ON COLUMN linhas_onibus.numero_linha IS 'Identificador único (ex: 101, 102)';

-- ============================================================================
-- 3. TABELA: ITINERARIOS
-- ============================================================================
-- Define a ordem das paradas para cada linha
-- É a "rota" propriamente dita: qual parada vem primeiro, segundo, etc
-- Uma linha pode ter múltiplas variações de itinerário
-- ============================================================================

CREATE TABLE IF NOT EXISTS itinerarios (
    id SERIAL PRIMARY KEY,
    linha_id INTEGER NOT NULL,
    parada_id INTEGER NOT NULL,
    ordem_parada INTEGER NOT NULL,
    tempo_estimado_anterior_minutos INTEGER,
    eh_ponto_integracao BOOLEAN DEFAULT FALSE,
    criado_em TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints de integridade referencial
    FOREIGN KEY (linha_id) REFERENCES linhas_onibus(id) ON DELETE CASCADE,
    FOREIGN KEY (parada_id) REFERENCES pontos_parada(id) ON DELETE RESTRICT,
    
    -- Garantir sequência única por linha (não pode repetir ordem em mesma linha)
    UNIQUE(linha_id, ordem_parada)
);

-- Índices para otimizar queries de rota
CREATE INDEX IF NOT EXISTS idx_itinerarios_linha ON itinerarios(linha_id);
CREATE INDEX IF NOT EXISTS idx_itinerarios_parada ON itinerarios(parada_id);
CREATE INDEX IF NOT EXISTS idx_itinerarios_ordem ON itinerarios(linha_id, ordem_parada);
CREATE INDEX IF NOT EXISTS idx_integracao ON itinerarios(eh_ponto_integracao) WHERE eh_ponto_integracao = TRUE;

COMMENT ON TABLE itinerarios IS 'Sequência de paradas para cada linha (ordem importa!)';
COMMENT ON COLUMN itinerarios.ordem_parada IS 'Posição na sequência (1 é primeira parada)';
COMMENT ON COLUMN itinerarios.tempo_estimado_anterior_minutos IS 'Tempo entre esta parada e a anterior';
COMMENT ON COLUMN itinerarios.eh_ponto_integracao IS 'TRUE se neste ponto o usuário pode trocar de linha';

-- ============================================================================
-- DADOS DE EXEMPLO: Três linhas reais de Goiânia
-- ============================================================================

-- Inserir paradas de exemplo
INSERT INTO pontos_parada (nome, latitude, longitude, tipo) VALUES
    ('Terminal Centro', -15.7975, -47.8919, 'terminal'),
    ('Eixo Anhanguera', -15.7800, -47.9050, 'parada'),
    ('Terminal Novo Mundo', -16.6799, -49.2138, 'terminal'),
    ('Terminal Padre Pelágio', -16.6617, -49.3242, 'terminal'),
    ('Terminal Praça da Bíblia', -16.6733, -49.2394, 'terminal'),
    ('Setor Comercial Sul', -15.7900, -47.8800, 'parada'),
    ('UFG Campus Samambaia', -16.0000, -48.9500, 'parada')
ON CONFLICT (nome) DO NOTHING;

-- Inserir linhas
INSERT INTO linhas_onibus (numero_linha, nome_linha, descricao, status, empresa, tipo_servico) VALUES
    ('101', 'Eixo Anhanguera', 'Terminal Centro → Eixo Anhanguera → Terminal Novo Mundo', 'ativa', 'RMTC', 'regular'),
    ('102', 'Setor Comercial Sul', 'Terminal Centro → Setor Comercial S. → UFG Campus Samambaia', 'ativa', 'RMTC', 'regular'),
    ('103', 'Integração Bíblia', 'Terminal Padre Pelágio → Terminal Praça da Bíblia → Terminal Centro', 'ativa', 'RMTC', 'regular')
ON CONFLICT (numero_linha) DO NOTHING;

-- Inserir itinerários (rotas com ordem de paradas)
-- Linha 101: Terminal Centro → Eixo Anhanguera → Terminal Novo Mundo
INSERT INTO itinerarios (linha_id, parada_id, ordem_parada, tempo_estimado_anterior_minutos, eh_ponto_integracao) VALUES
    ((SELECT id FROM linhas_onibus WHERE numero_linha = '101'), (SELECT id FROM pontos_parada WHERE nome = 'Terminal Centro'), 1, NULL, TRUE),
    ((SELECT id FROM linhas_onibus WHERE numero_linha = '101'), (SELECT id FROM pontos_parada WHERE nome = 'Eixo Anhanguera'), 2, 10, FALSE),
    ((SELECT id FROM linhas_onibus WHERE numero_linha = '101'), (SELECT id FROM pontos_parada WHERE nome = 'Terminal Novo Mundo'), 3, 12, TRUE)
ON CONFLICT (linha_id, ordem_parada) DO NOTHING;

-- Linha 102: Terminal Centro → Setor Comercial Sul → UFG
INSERT INTO itinerarios (linha_id, parada_id, ordem_parada, tempo_estimado_anterior_minutos, eh_ponto_integracao) VALUES
    ((SELECT id FROM linhas_onibus WHERE numero_linha = '102'), (SELECT id FROM pontos_parada WHERE nome = 'Terminal Centro'), 1, NULL, TRUE),
    ((SELECT id FROM linhas_onibus WHERE numero_linha = '102'), (SELECT id FROM pontos_parada WHERE nome = 'Setor Comercial Sul'), 2, 8, FALSE),
    ((SELECT id FROM linhas_onibus WHERE numero_linha = '102'), (SELECT id FROM pontos_parada WHERE nome = 'UFG Campus Samambaia'), 3, 15, FALSE)
ON CONFLICT (linha_id, ordem_parada) DO NOTHING;

-- Linha 103: Terminal Padre Pelágio → Terminal Praça da Bíblia → Terminal Centro
INSERT INTO itinerarios (linha_id, parada_id, ordem_parada, tempo_estimado_anterior_minutos, eh_ponto_integracao) VALUES
    ((SELECT id FROM linhas_onibus WHERE numero_linha = '103'), (SELECT id FROM pontos_parada WHERE nome = 'Terminal Padre Pelágio'), 1, NULL, TRUE),
    ((SELECT id FROM linhas_onibus WHERE numero_linha = '103'), (SELECT id FROM pontos_parada WHERE nome = 'Terminal Praça da Bíblia'), 2, 10, TRUE),
    ((SELECT id FROM linhas_onibus WHERE numero_linha = '103'), (SELECT id FROM pontos_parada WHERE nome = 'Terminal Centro'), 3, 12, TRUE)
ON CONFLICT (linha_id, ordem_parada) DO NOTHING;

-- ============================================================================
-- VERIFICAÇÃO FINAL
-- ============================================================================

SELECT '✅ Schema criado com sucesso!' as status;
SELECT COUNT(*) as total_paradas FROM pontos_parada;
SELECT COUNT(*) as total_linhas FROM linhas_onibus;
SELECT COUNT(*) as total_itinerarios FROM itinerarios;
