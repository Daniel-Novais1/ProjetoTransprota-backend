-- PostgreSQL PostGIS Schema para TranspRota
-- Conformidade com GEOMETRY(Point, 4326) e índices GIST

-- Habilitar extensão PostGIS
CREATE EXTENSION IF NOT EXISTS postgis;

-- Tabela de localizações com PostGIS
DROP TABLE IF EXISTS locations CASCADE;
CREATE TABLE locations (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    location GEOMETRY(Point, 4326) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Índice GIST para performance sub-millisecond
CREATE INDEX idx_locations_location_gist ON locations USING GIST (location);

-- Tabela de denúncias com PostGIS
DROP TABLE IF EXISTS denuncias CASCADE;
CREATE TABLE denuncias (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    bus_line VARCHAR(10) NOT NULL,
    bus_id VARCHAR(50) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('Lotado', 'Atrasado', 'Não Parou', 'Ar Estragado', 'Sujo')),
    location GEOMETRY(Point, 4326) NOT NULL,
    evidence_url TEXT,
    trust_score INTEGER DEFAULT 0 CHECK (trust_score >= 0 AND trust_score <= 100),
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Índice GIST para queries de proximidade
CREATE INDEX idx_denuncias_location_gist ON denuncias USING GIST (location);

-- Índice temporal para performance
CREATE INDEX idx_denuncias_timestamp ON denuncias (timestamp DESC);

-- Tabela de buscas de rotas (LGPD-Compliant) com PostGIS
DROP TABLE IF EXISTS route_searches CASCADE;
CREATE TABLE route_searches (
    id SERIAL PRIMARY KEY,
    origin VARCHAR(255) NOT NULL,
    destination VARCHAR(255) NOT NULL,
    origin_location GEOMETRY(Point, 4326), -- Coordenada do ponto de origem
    destination_location GEOMETRY(Point, 4326), -- Coordenada do ponto de destino
    search_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_rush_hour BOOLEAN DEFAULT FALSE,
    day_of_week INTEGER CHECK (day_of_week >= 0 AND day_of_week <= 6),
    -- NOTA: NÃO salvamos IP ou dados identificáveis (LGPD-Compliance)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Índices para performance de analytics com PostGIS
CREATE INDEX idx_route_searches_search_time ON route_searches (search_time DESC);
CREATE INDEX idx_route_searches_origin_destination ON route_searches (origin, destination);
CREATE INDEX idx_route_searches_is_rush_hour ON route_searches (is_rush_hour);

-- Índices GIST para queries espaciais (sub-millisecond performance)
CREATE INDEX idx_route_searches_origin_location_gist ON route_searches USING GIST (origin_location);
CREATE INDEX idx_route_searches_destination_location_gist ON route_searches USING GIST (destination_location);

-- Tabela de linhas de ônibus (mantida para compatibilidade)
DROP TABLE IF EXISTS linhas_onibus CASCADE;
CREATE TABLE linhas_onibus (
    id SERIAL PRIMARY KEY,
    numero_linha VARCHAR(10) UNIQUE NOT NULL,
    nome_linha VARCHAR(255) NOT NULL,
    status VARCHAR(20) DEFAULT 'ativa' CHECK (status IN ('ativa', 'inativa', 'manutencao')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Trigger para atualizar timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_locations_updated_at 
    BEFORE UPDATE ON locations 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_linhas_onibus_updated_at 
    BEFORE UPDATE ON linhas_onibus 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Inserir dados iniciais (terminais de Goiânia)
INSERT INTO locations (name, location) VALUES
('Terminal Novo Mundo', ST_GeomFromText('POINT(-49.2628 -16.6864)', 4326)),
('Terminal Bíblia', ST_GeomFromText('POINT(-49.2534 -16.6802)', 4326)),
('Terminal Isidória', ST_GeomFromText('POINT(-49.2456 -16.6721)', 4326)),
('Terminal Padre Pelágio', ST_GeomFromText('POINT(-49.2367 -16.6643)', 4326)),
('Terminal Canedo', ST_GeomFromText('POINT(-49.3123 -16.6987)', 4326)),
('Campus Samambaia UFG', ST_GeomFromText('POINT(-49.2234 -16.6543)', 4326)),
('Setor Bueno', ST_GeomFromText('POINT(-49.2678 -16.6890)', 4326)),
('Setor Marista', ST_GeomFromText('POINT(-49.2543 -16.6765)', 4326)),
('Setor Universitário', ST_GeomFromText('POINT(-49.2456 -16.6678)', 4326)),
('Terminal Praça da Bíblia', ST_GeomFromText('POINT(-49.2534 -16.6802)', 4326)),
('Terminal Rodoviário', ST_GeomFromText('POINT(-49.2789 -16.6901)', 4326)),
('Terminal Leste', ST_GeomFromText('POINT(-49.2345 -16.6589)', 4326)),
('Terminal Oeste', ST_GeomFromText('POINT(-49.2876 -16.6943)', 4326)),
('Terminal Sul', ST_GeomFromText('POINT(-49.2567 -16.6789)', 4326)),
('Terminal Norte', ST_GeomFromText('POINT(-49.2789 -16.6943)', 4326));

-- Inserir linhas de ônibus
INSERT INTO linhas_onibus (numero_linha, nome_linha, status) VALUES
('M10', 'Terminal Novo Mundo - Terminal Bíblia', 'ativa'),
('M23', 'Terminal Novo Mundo - Campus Samambaia', 'ativa'),
('M33', 'Terminal Isidória - Terminal Padre Pelágio', 'ativa'),
('M43', 'Terminal Novo Mundo - Setor Universitário', 'ativa'),
('M55', 'Terminal Isidória - Terminal Leste', 'ativa'),
('M60', 'Terminal Bíblia - Terminal Canedo', 'ativa'),
('M71', 'Terminal Novo Mundo - Terminal Sul', 'ativa'),
('M77', 'Terminal Isidória - Terminal Oeste', 'ativa'),
('INTERMUNICIPAL', 'Goiânia - Senador Canedo', 'ativa');

-- Tabela de denúncias de usuários com PostGIS
CREATE TABLE user_reports (
    id SERIAL PRIMARY KEY,
    tipo_problema VARCHAR(50) NOT NULL CHECK (tipo_problema IN ('Lotado', 'Atrasado', 'Perigo')),
    descricao TEXT,
    report_location GEOMETRY(Point, 4326) NOT NULL,
    bus_line VARCHAR(50),
    user_ip_hash VARCHAR(64), -- IP anonimizado para anti-spam
    trust_score DECIMAL(3,2) DEFAULT 1.0, -- Score de confiança do usuário
    status VARCHAR(20) DEFAULT 'ativa' CHECK (status IN ('ativa', 'resolvida', 'falsa')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP + INTERVAL '2 hours')
);

-- Índices GIST para performance espacial das denúncias
CREATE INDEX idx_user_reports_location_gist ON user_reports USING GIST (report_location);
CREATE INDEX idx_user_reports_bus_line ON user_reports (bus_line);
CREATE INDEX idx_user_reports_tipo_problema ON user_reports (tipo_problema);
CREATE INDEX idx_user_reports_created_at ON user_reports (created_at);
CREATE INDEX idx_user_reports_expires_at ON user_reports (expires_at);
CREATE INDEX idx_user_reports_user_ip_hash ON user_reports (user_ip_hash);

-- Trigger para limpeza automática de denúncias expiradas
CREATE OR REPLACE FUNCTION cleanup_expired_reports()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM user_reports WHERE expires_at < CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger para executar limpeza a cada hora
CREATE TRIGGER trigger_cleanup_expired_reports
AFTER INSERT ON user_reports
EXECUTE FUNCTION cleanup_expired_reports();

-- Views para performance de queries
CREATE VIEW v_denuncias_recentes AS
SELECT 
    id,
    user_ip_hash,
    bus_line,
    tipo_problema,
    ST_AsText(report_location) as location_text,
    ST_AsGeoJSON(report_location) as location_geojson,
    trust_score,
    created_at
FROM user_reports 
WHERE created_at > CURRENT_TIMESTAMP - INTERVAL '1 hour'
    AND status = 'ativa'
ORDER BY created_at DESC;

-- View para mapa de calor (heatmap)
CREATE VIEW v_heatmap_data AS
SELECT 
    bus_line,
    tipo_problema,
    COUNT(*) as report_count,
    AVG(trust_score) as avg_trust_score,
    ST_Centroid(ST_Collect(report_location)) as centroid_location,
    created_at
FROM user_reports 
WHERE created_at > CURRENT_TIMESTAMP - INTERVAL '1 hour'
    AND status = 'ativa'
GROUP BY bus_line, tipo_problema, DATE_TRUNC('minute', created_at)
HAVING COUNT(*) >= 3; -- Threshold para heatmap
ORDER BY report_count DESC;

CREATE VIEW v_terminais_ativos AS
SELECT 
    id,
    name,
    ST_AsText(location) as location_text,
    ST_AsGeoJSON(location) as location_geojson,
    created_at
FROM locations
ORDER BY name;

-- Funções PostGIS para queries espaciais
CREATE OR REPLACE FUNCTION proximos_terminais(
    lat FLOAT, 
    lng FLOAT, 
    raio_metros INTEGER DEFAULT 500
)
RETURNS TABLE (
    id INTEGER,
    nome VARCHAR(255),
    distancia FLOAT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        l.id,
        l.name,
        ST_DistanceSphere(
            l.location, 
            ST_GeomFromText('POINT(' || lng || ' ' || lat || ')', 4326)
        ) as distancia
    FROM locations l
    WHERE ST_DWithin(
        l.location, 
        ST_GeomFromText('POINT(' || lng || ' ' || lat || ')', 4326), 
        raio_metros
    )
    ORDER BY distancia;
END;
$$ LANGUAGE plpgsql;

-- Função para analytics de rotas trending
CREATE OR REPLACE FUNCTION trending_routes(
    dias INTEGER DEFAULT 7,
    limite INTEGER DEFAULT 3
)
RETURNS TABLE (
    origin VARCHAR(255),
    destination VARCHAR(255),
    search_count BIGINT,
    last_search TIMESTAMP
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        rs.origin,
        rs.destination,
        COUNT(*) as search_count,
        MAX(rs.search_time) as last_search
    FROM route_searches rs
    WHERE rs.search_time >= CURRENT_TIMESTAMP - (dias || ' days')::INTERVAL
    GROUP BY rs.origin, rs.destination
    ORDER BY search_count DESC, last_search DESC
    LIMIT limite;
END;
$$ LANGUAGE plpgsql;

-- Configurar permissões (ajustar conforme necessário)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO transprota_user;
-- GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO transprota_user;

-- Estatísticas do PostgreSQL para otimização
ANALYZE locations;
ANALYZE denuncias;
ANALYZE route_searches;
ANALYZE linhas_onibus;

-- Comentários para documentação
COMMENT ON TABLE locations IS 'Terminais e pontos de interesse com coordenadas PostGIS';
COMMENT ON COLUMN locations.location IS 'Coordenada geográfica usando GEOMETRY(Point, 4326) com índice GIST';
COMMENT ON TABLE denuncias IS 'Denúncias colaborativas com localização espacial';
COMMENT ON COLUMN denuncias.location IS 'Localização da denúncia usando GEOMETRY(Point, 4326)';
COMMENT ON TABLE route_searches IS 'Buscas de rotas para analytics (LGPD-Compliant)';
COMMENT ON INDEX idx_locations_location_gist IS 'Índice GIST para queries espaciais sub-millisecond';
COMMENT ON INDEX idx_denuncias_location_gist IS 'Índice GIST para queries de proximidade de denúncias';
