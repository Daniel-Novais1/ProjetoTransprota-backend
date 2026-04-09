# CHANGELOG - TranspRota

Todas as mudanças significativas neste projeto serão documentadas aqui.

O formato é baseado em [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
e este projeto segue [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Planejado
- [ ] Endpoint `/planejar-viagem` - Algoritmo A* para melhor rota
- [ ] Integração de ETA dinâmico com histórico de atrasos
- [ ] Endpoints para dados de denúncias (lotação, atrasos, problemas)
- [ ] Sistema de Trust Score para validar informações conforme
- [ ] API de histórico de posições para analytics
- [ ] WebSocket para atualizações em tempo real (GPS)

---

## [0.2.0] - 2026-04-08

### ✨ Adicionado (Fase 2: Rotas e Planejamento)

#### Schema PostgreSQL
- **Tabela `linhas_onibus`**: Catálogo de linhas de transporte
  - Campos: numero_linha (UNIQUE), nome_linha, descricao, status, empresa, tipo_servico
  - Índices: idx_linhas_numero, idx_linhas_status
  
- **Tabela `itinerarios`**: Sequência de paradas por linha
  - Campos: linha_id, parada_id, ordem_parada, tempo_estimado_anterior_minutos, eh_ponto_integracao
  - Relações: FK para linhas_onibus (CASCADE), FK para locations (RESTRICT)
  - Constraints: UNIQUE(linha_id, ordem_parada)
  
- **Tabela `historico_posicoes`**: Rastreamento de atrasos e ocupação
  - Campos: linha_id, bus_id, parada_id, tempo_chegada, tempo_saida, atraso_minutos, lotacao_percentage
  - Índices para analytics: idx_historico_linha, idx_historico_parada, idx_historico_data

#### Views SQL
- `v_rotas_completas`: Visualiza todas as rotas com paradas ordenadas
- `v_pontos_integracao`: Identifica paradas com múltiplas linhas disponíveis
- `v_proximas_paradas`: Sequência de paradas com tempo acumulado

#### Dados de Teste
- 3 linhas cadastradas: 101 (Eixo Anhanguera), 102 (Setor Comercial), 103 (Integração Bíblia)
- 9 itinerários complete (3 paradas por linha)
- 11 paradas totais incluindo pontos de integração

#### Documentação
- `migrations_rotas.sql`: Schema completo com comentários
- `insert_itinerarios.sql`: Inserção de dados de exemplo
- `verify_schema.sql`: Queries de validação
- `TESTE_ROTAS.sql`: 6 exemplos de queries úteis
- `.github/copilot-instructions.md`: Guia GSD para desenvolvimento

### ⚙️ Mudanças
- Estrutura do main.go já suporta integração com novas tabelas
- Implementação de endpoints de rotas será na versão 0.3.0

### 📋 Notas
- **Backwards-compatible**: Código Go existente não foi alterado
- **Migration**: Requer execução de `migrations_rotas.sql` no PostgreSQL
- **Validação**: Todas as tabelas criadas com sucesso, dados inseridos

---

## [0.1.0] - 2026-04-08

### ✨ Adicionado (Fase 1: Geolocalização Básica)

#### API REST (Go)
- **GET /terminais**: Lista todos os pontos de parada com coordenadas
- **GET /gps/:id**: Consulta posição atual de um ônibus
- **POST /gps**: Registra/atualiza posição de ônibus (autenticado)
- **GET /gps/:id/status**: Verifica proximidade com terminais

#### Banco de Dados
- Tabela `locations`: Armazena terminais, latitude, longitude
- Integração Redis para cache de posições em tempo real (TTL 10 min)

#### Infraestrutura
- PostgreSQL 15 com PostGIS 3.3
- Redis 7 Alpine para cache
- Docker Compose para orquestração

#### Autenticação
- Middleware de autenticação com X-API-Key
- Validação de coordenadas (latitude: [-90, 90], longitude: [-180, 180])

#### Utilitários
- Cálculo de distância usando fórmula de Haversine
- Logging estruturado com emojis para visualização
- Context com timeouts em todas as queries

### 🔧 Tecnologia
- **Linguagem**: Go 1.21+
- **Framework HTTP**: Gin
- **Banco Principal**: PostgreSQL
- **Cache**: Redis
- **Containerização**: Docker & Docker Compose

### 📊 Testes Realizados
- ✅ Listagem de terminais (9 registros)
- ✅ Armazenamento de GPS (JSON com timestamp)
- ✅ Consulta de posição (compatível com formato antigo)
- ✅ Detecção de proximidade (150m de threshold)
- ✅ Status em trânsito vs no terminal

---

## Guia de Leitura

### Para Desenvolvedores
1. Ler `.github/copilot-instructions.md` (protocolo GSD)
2. Ler `SPECS.md` (requisitos de negócio)
3. Revisar `main.go` (arquitetura atual)
4. Consultar `TESTE_ROTAS.sql` (queries úteis)

### Para Testes
1. Verificar sessão de testes em cada release
2. Seguir comandos SQL/curl/PowerShell fornecidos
3. Validar com queries específicas

### Para Migrations
1. Backup do banco antes de rodar migration
2. Executar arquivo `.sql` na ordem apropriada
3. Rodar `verify_schema.sql` para confirmação

---

## Status Atual

```
Phase 1 (Geoloc): ✅ COMPLETO
Phase 2 (Rotas):  ✅ COMPLETO (schema only, endpoints em 0.3.0)
Phase 3 (ETA):    ⏳ PRÓXIMO
Phase 4 (Denúncias): ⏳ FUTURO
Phase 5 (Trust Score): ⏳ FUTURO
```

---

## Contato & Suporte

- **Desenvolvedor**: Daniel de Novais Santos Mendonça
- **Instituição**: UFG - Engenharia de Software
- **Ano**: 2026

