-- ============================================================================
-- COMANDOS DE TESTE: Sistema de Rotas do TranspRota
-- ============================================================================
-- Use esses comandos para testar a funcionalidade de planejador de viagem
-- ============================================================================

-- 1. LISTAR TODAS AS LINHAS ATIVAS
-- Comando SQL para consultar banco diretamente:
SELECT numero_linha, nome_linha, status FROM linhas_onibus WHERE status = 'ativa' ORDER BY numero_linha;

-- Endpoint futuro (quando implementado em Go):
-- GET http://localhost:8080/linhas

-- ============================================================================

-- 2. OBTER ITINERÁRIO COMPLETO DE UMA LINHA
-- Exemplo: traçar rota da Linha 101

SELECT 
    l.numero_linha,
    l.nome_linha,
    i.ordem_parada,
    loc.name as parada,
    loc.id as parada_id,
    loc.latitude,
    loc.longitude,
    i.tempo_estimado_anterior_minutos,
    COALESCE(
        LAG(i.tempo_estimado_anterior_minutos, 1, 0) OVER (PARTITION BY l.id ORDER BY i.ordem_parada),
        0
    ) + COALESCE(i.tempo_estimado_anterior_minutos, 0) as tempo_acumulado_minutos
FROM linhas_onibus l
JOIN itinerarios i ON l.id = i.linha_id
JOIN locations loc ON i.parada_id = loc.id
WHERE l.numero_linha = '101'
ORDER BY i.ordem_parada;

-- Endpoint futuro:
-- GET http://localhost:8080/rotas/101

-- ============================================================================

-- 3. ENCONTRAR LINHAS QUE PASSAM POR UMA PARADA ESPECÍFICA
-- Exemplo: Quais linhas passam em "Terminal Centro"?

SELECT DISTINCT
    l.numero_linha,
    l.nome_linha,
    i.ordem_parada,
    FIRST_VALUE(loc.name) OVER (PARTITION BY l.id ORDER BY i.ordem_parada) as inicio_linha,
    LAST_VALUE(loc.name) OVER (PARTITION BY l.id ORDER BY i.ordem_parada ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) as fim_linha
FROM linhas_onibus l
JOIN itinerarios i ON l.id = i.linha_id
JOIN locations loc ON i.parada_id = loc.id
WHERE loc.name = 'Terminal Centro'
ORDER BY l.numero_linha;

-- Endpoint futuro:
-- GET http://localhost:8080/paradas/7/linhas  (7 é o ID do Terminal Centro)

-- ============================================================================

-- 4. BUSCAR PONTOS DE INTEGRAÇÃO (onde mudar de linha)
-- Útil para montar rotas com conexões

SELECT DISTINCT
    loc.id as parada_id,
    loc.name as nome_parada,
    loc.latitude,
    loc.longitude,
    COUNT(DISTINCT l.id) as total_linhas,
    STRING_AGG(DISTINCT l.numero_linha, ', ' ORDER BY l.numero_linha) as linhas
FROM locations loc
JOIN itinerarios i ON loc.id = i.parada_id
JOIN linhas_onibus l ON i.linha_id = l.id
WHERE i.eh_ponto_integracao = TRUE AND l.status = 'ativa'
GROUP BY loc.id, loc.name, loc.latitude, loc.longitude
ORDER BY total_linhas DESC;

-- Endpoint futuro:
-- GET http://localhost:8080/integracao/paradas

-- ============================================================================

-- 5. VERIFICAR CONEXÃO DIRETA ENTRE DOIS PONTOS
-- Retorna TRUE se há uma linha que vai de A para B sem precisar trocar

SELECT 
    l.numero_linha,
    l.nome_linha,
    min_ordem.ordem as parada_origem_ordem,
    max_ordem.ordem as parada_destino_ordem,
    (max_ordem.ordem - min_ordem.ordem) as paradas_intermediarias,
    COALESCE(SUM(i.tempo_estimado_anterior_minutos), 0) as tempo_estimado_minutos
FROM linhas_onibus l
JOIN (
    SELECT l2.id as linha_id, i2.ordem_parada as ordem
    FROM linhas_onibus l2
    JOIN itinerarios i2 ON l2.id = i2.linha_id
    JOIN locations loc2 ON i2.parada_id = loc2.id
    WHERE loc2.id = 7  -- Substituir por ID de origem (ex: Terminal Centro=7)
) min_ordem ON l.id = min_ordem.linha_id
JOIN (
    SELECT l3.id as linha_id, i3.ordem_parada as ordem
    FROM linhas_onibus l3
    JOIN itinerarios i3 ON l3.id = i3.linha_id
    JOIN locations loc3 ON i3.parada_id = loc3.id
    WHERE loc3.id = 8  -- Substituir por ID de destino (ex: Eixo Anhanguera=8)
) max_ordem ON l.id = max_ordem.linha_id
JOIN itinerarios i ON l.id = i.linha_id 
    AND i.ordem_parada BETWEEN min_ordem.ordem AND max_ordem.ordem
WHERE l.status = 'ativa' AND min_ordem.ordem < max_ordem.ordem
GROUP BY l.id, l.numero_linha, l.nome_linha, min_ordem.ordem, max_ordem.ordem
ORDER BY l.numero_linha;

-- Endpoint futuro:
-- GET http://localhost:8080/rota/direto?origem=7&destino=8

-- ============================================================================

-- 6. INTEGRAÇÃO COM GPS EM TEMPO REAL
-- Ver qual é a próxima parada de um ônibus baseado na linha e posição atual

SELECT 
    l.numero_linha,
    l.nome_linha,
    i.ordem_parada,
    loc.name as proxima_parada,
    loc.latitude,
    loc.longitude,
    i.tempo_estimado_anterior_minutos as tempo_ate_proxima
FROM linhas_onibus l
JOIN itinerarios i ON l.id = i.linha_id
JOIN locations loc ON i.parada_id = loc.id
WHERE l.numero_linha = '101'  -- Exemplo: listar próximas paradas da linha 101
ORDER BY i.ordem_parada;

-- Este será integrado com:
-- - Posição atual do ônibus (armazenada em Redis)
-- - Histórico de atrasos (tabela historico_posicoes)
-- - Cálculo de ETA dinâmico

-- ============================================================================
-- NOTAS IMPORTANTES:
-- ============================================================================

-- 1. IDs das paradas principais:
--    Terminal Centro = 7
--    Eixo Anhanguera = 8
--    Terminal Novo Mundo = 1
--    Terminal Padre Pelágio = 3
--    Terminal Praça da Bíblia = 2
--    Setor Comercial Sul = 10
--    UFG Campus Samambaia = 11

-- 2. A tabela HISTORICO_POSICOES permite rastrear:
--    - Atrasos reais vs estimados
--    - Taxa de ocupação dos ônibus
--    - Padrões de movimento para otimizar rotas

-- 3. As VIEWS criadas facilitam:
--    - v_rotas_completas: ver todas as rotas
--    - v_pontos_integracao: encontrar paradas com múltiplas linhas
--    - v_proximas_paradas: útil para exibir no mapa

-- 4. Próulos passos no Go:
--    - Endpoint /linhas: list all bus lines
--    - Endpoint /rotas/:numero_linha: get full itinerary
--    - Endpoint /paradas/:parada_id/linhas: find lines at a stop
--    - Endpoint /planejar-viagem: find best route (A* or Dijkstra)
