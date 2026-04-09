-- Schema para sistema de rotas de ônibus

-- Tabela de pontos de parada
CREATE TABLE IF NOT EXISTS pontos_parada (
    id SERIAL PRIMARY KEY,
    nome VARCHAR(255) NOT NULL UNIQUE,
    latitude DECIMAL(10, 8) NOT NULL,
    longitude DECIMAL(11, 8) NOT NULL,
    criado_em TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de linhas de ônibus
CREATE TABLE IF NOT EXISTS linhas_onibus (
    id SERIAL PRIMARY KEY,
    numero_linha VARCHAR(50) NOT NULL UNIQUE,
    nome VARCHAR(255) NOT NULL,
    descricao TEXT,
    status VARCHAR(20) DEFAULT 'ativa',
    criado_em TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de detalhamento de rotas
CREATE TABLE IF NOT EXISTS rota_detalhe (
    id SERIAL PRIMARY KEY,
    linha_id INTEGER NOT NULL,
    parada_id INTEGER NOT NULL,
    ordem_parada INTEGER NOT NULL,
    tempo_estimado_anterior_minutos INTEGER,
    criado_em TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (linha_id) REFERENCES linhas_onibus(id) ON DELETE CASCADE,
    FOREIGN KEY (parada_id) REFERENCES pontos_parada(id) ON DELETE RESTRICT,
    UNIQUE (linha_id, ordem_parada)
);

-- Índices para melhorar performance
CREATE INDEX idx_linhas_status ON linhas_onibus(status);
CREATE INDEX idx_rota_linha ON rota_detalhe(linha_id);
CREATE INDEX idx_rota_parada ON rota_detalhe(parada_id);
CREATE INDEX idx_rota_ordem ON rota_detalhe(linha_id, ordem_parada);

-- Exemplo de inserção de dados
INSERT INTO pontos_parada (nome, latitude, longitude) VALUES
    ('Terminal Centro', -15.7975, -47.8919),
    ('Eixo Anhanguera', -15.7800, -47.9050),
    ('Estação Rodoviária', -15.7834, -47.8873)
ON CONFLICT (nome) DO NOTHING;

INSERT INTO linhas_onibus (numero_linha, nome, descricao) VALUES
    ('101', 'Eixo Anhanguera', 'Linha que percorre o Eixo Anhanguera'),
    ('102', 'Setor Comercial Sul', 'Linha que vai para o Setor Comercial Sul')
ON CONFLICT (numero_linha) DO NOTHING;

INSERT INTO rota_detalhe (linha_id, parada_id, ordem_parada, tempo_estimado_anterior_minutos) VALUES
    (1, 1, 1, NULL),
    (1, 2, 2, 10),
    (1, 3, 3, 8),
    (2, 1, 1, NULL),
    (2, 3, 2, 12)
ON CONFLICT (linha_id, ordem_parada) DO NOTHING;
