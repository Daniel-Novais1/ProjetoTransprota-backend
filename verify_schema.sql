SELECT '=== RESUMO DO BANCO DE DADOS ===' as info;

SELECT COUNT(*) as total_linhas FROM linhas_onibus;
SELECT COUNT(*) as total_itinerarios FROM itinerarios;
SELECT COUNT(*) as total_paradas FROM locations;

SELECT '=== LINHAS CRIADAS ===' as info;
SELECT numero_linha, nome_linha, status, tipo_servico FROM linhas_onibus ORDER BY numero_linha;

SELECT '=== ITINERÁRIOS COMPLETOS ===' as info;
SELECT 
    l.numero_linha,
    l.nome_linha,
    COUNT(i.id) as quantidade_paradas,
    STRING_AGG(loc.name, ' → ' ORDER BY i.ordem_parada) as rota
FROM linhas_onibus l
JOIN itinerarios i ON l.id = i.linha_id
JOIN locations loc ON i.parada_id = loc.id
GROUP BY l.id, l.numero_linha, l.nome_linha
ORDER BY l.numero_linha;

SELECT '=== PONTOS DE INTEGRAÇÃO ===' as info;
SELECT 
    loc.name,
    STRING_AGG(DISTINCT l.numero_linha, ', ' ORDER BY l.numero_linha) as linhas,
    COUNT(DISTINCT l.id) as total_linhas
FROM locations loc
JOIN itinerarios i ON loc.id = i.parada_id
JOIN linhas_onibus l ON i.linha_id = l.id
WHERE i.eh_ponto_integracao = TRUE
GROUP BY loc.id, loc.name
ORDER BY total_linhas DESC, loc.name;
