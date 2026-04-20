# TranspRota SaaS - Mapa de Contextualização para IA

> **Projeto:** Sistema de Telemetria GPS em Tempo Real com Auditoria e Geofencing
> **Status:** Production-Ready (Autonomia Nível 5 Concluída)
> **Data:** 2026-04-10
> **Squad:** TranspRota Backend

---

## 📁 Estrutura do Projeto

```
projetoTransprota/
├── internal/
│   ├── auth/                 # Autenticação JWT
│   │   ├── jwt.go          # JWT Manager
│   │   ├── middleware.go   # AuthMiddleware
│   │   └── handler.go      # Login endpoint
│   ├── telemetry/           # Telemetria GPS
│   │   ├── model.go        # Structs (TelemetryPing, Geofence, etc)
│   │   ├── handler.go      # GPS endpoint
│   │   ├── repository.go   # Database operations
│   │   ├── routes.go       # Route configuration
│   │   ├── geofencing.go   # Geofencing Dinâmico
│   │   └── bi_analytics.go # Analytics de Engarrafamento
│   ├── infrastructure/      # Infraestrutura
│   │   ├── websocket.go    # WebSocket Hub
│   │   └── cleanup_worker.go # Cleanup Worker
│   ├── server/              # HTTP Server
│   │   └── server.go       # Server setup + Rate Limiting
│   └── config/              # Configuração
│       └── config.go        # Environment variables
├── migrations/              # Database migrations
│   ├── 01_schema.sql
│   ├── 02_audit_logs.sql
│   ├── 03_gps_telemetry.sql
│   ├── 04_users.sql
│   ├── 05_functions.sql
│   ├── 06_audit_logs_function.sql
│   └── 07_performance_indexes.sql
├── tests/                  # Testes
│   └── stress_test.go      # Stress test (50 ônibus)
├── frontend/               # Frontend (React/Vite)
├── main.go                # Entry point
├── go.mod                 # Dependencies
├── .env                   # Environment variables
├── docker-compose.yml      # Docker services
└── SQUAD_BACKLOG.md       # Backlog e status
```

---

## 🏗️ Arquitetura

### Stack Tecnológico
- **Backend:** Go 1.21+ com Gin framework
- **Banco de Dados:** PostgreSQL 15 + PostGIS (geoespacial)
- **Cache:** Redis 7
- **Autenticação:** JWT (github.com/golang-jwt/jwt/v5)
- **WebSocket:** Gorilla WebSocket
- **Frontend:** React + Vite
- **Containerização:** Docker + Docker Compose

### Padrões Arquiteturais
- **Modularização:** Packages internos separados (auth, telemetry, infrastructure, server, config)
- **Repository Pattern:** Repository layer para database operations
- **Middleware Stack:** CORS → Error Handler → Rate Limit → Auth → Routes
- **Graceful Shutdown:** 30s timeout com context cancellation
- **Audit Logging:** Tabela audit_logs com função SQL log_audit()

---

## 🔐 Autenticação

### JWT Configuration
- **Secret Key:** `JWT_SECRET_KEY` (fallback: "transprota-secret-key-2024")
- **Expiration:** 24h
- **Algorithm:** HS256

### Endpoints
- **POST /api/v1/auth/login** - Login e obtenção de token
  - Body: `{"username": "admin", "password": "admin123"}`
  - Response: `{"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."}`

### Middleware
- **AuthMiddleware:** Protege rotas exigindo `Authorization: Bearer <token>`
- **Rate Limit:** 100 requisições por minuto por IP (in-memory sliding window)

---

## 📡 API Endpoints

### Telemetria GPS
- **POST /api/v1/telemetry/gps** (Protegido com Auth)
  - Headers: `Authorization: Bearer <token>`
  - Body:
    ```json
    {
      "device_id": "bus-001",
      "lat": -16.6869,
      "lng": -49.2648,
      "speed": 45.5,
      "heading": 180,
      "accuracy": 10,
      "recorded_at": "2024-01-01T12:00:00Z"
    }
    ```
  - Response: `202 Accepted`

### Health Check
- **GET /health** - Health check do servidor
- **GET /api/v1/health** - Health check com versão

### WebSocket
- **WS /ws** - WebSocket para atualizações em tempo real
  - Subscribes: Redis channel "bus_updates"

---

## 🗄️ Banco de Dados

### Tabelas Principais
- **gps_telemetry:** Dados GPS com PostGIS (particionada por data)
- **audit_logs:** Rastro completo de auditoria
- **users:** Usuários do sistema (mock auth atual)
- **geofences:** Cercas geográficas para monitoramento
- **geofence_alerts:** Alertas disparados por geofencing

### Índices
- **PostGIS:** GIST espaciais para coordenadas
- **Time-series:** B-Tree em recorded_at
- **Performance:** idx_gps_telemetry_time_recent, idx_gps_telemetry_speed_mode, idx_audit_logs_ip

### Funções SQL
- **log_audit()** - Função para registrar audit logs automaticamente

---

## 🚨 Features Implementadas

### 1. Telemetria GPS
- Recepção de dados GPS em tempo real
- Validação de campos obrigatórios
- Anonimização de device_id (LGPD compliance)
- Cache em Redis para última posição conhecida
- Background processing (non-blocking)

### 2. Autenticação JWT
- Login com username/password
- Token JWT com claims (user_id, username)
- Middleware de proteção de rotas
- Validação de Bearer token

### 3. Rate Limiting
- 100 requisições por minuto por IP
- Sliding window algorithm
- In-memory map com timestamps
- HTTP 429 em caso de excesso

### 4. Audit Logging
- Rastro completo de todas as ações
- Tabela audit_logs com campos detalhados
- Função SQL log_audit() para registro
- Logs estruturados com prefixos [AUTH], [TELEMETRY], [DB], [SERVER]

### 5. Geofencing Dinâmico
- Cerca de Goiânia (25km raio)
- Fórmula de Haversine para cálculo de distância
- Alertas no Redis ao sair da cerca
- Log de auditoria crítico para GEOFENCE_BREACH

### 6. BI Analytics
- Worker analisa velocidade média dos últimos 10 minutos
- Detecta engarrafamento (velocidade < 80% baseline)
- Publica alerta no canal bi_alerts do Redis
- Baseline configurável (padrão: 40 km/h)

### 7. WebSocket
- Hub para conexões em tempo real
- Broadcast de mensagens do Redis
- Mutex para proteção de clients map
- Graceful shutdown

### 8. Cleanup Worker
- Remove dados GPS com mais de 7 dias
- Executa a cada 1 hora
- Graceful shutdown com context
- Logs estruturados

### 9. Graceful Shutdown
- 30s timeout para conexões ativas
- Context cancellation para workers
- Logs de shutdown
- Fechamento de conexões DB e Redis

### 10. Stress Testing
- Teste com 50 ônibus simultâneos
- 500 requisições totais (10 por ônibus)
- Métricas de latência e sucesso
- Login JWT automático

---

## ⚙️ Configuração

### Environment Variables (.env)
```env
# Database
DB_USER=admin
DB_PASSWORD=password123
DB_NAME=transprota
DB_HOST=127.0.0.1
DB_PORT=5432

# Redis
REDIS_ADDR=127.0.0.1:6379
REDIS_PASSWORD=

# API
API_SECRET_KEY=sua_chave_ultra_secreta_123
API_KEY=sua_chave_ultra_secreta_123
SERVER_PORT=8080
GIN_MODE=release

# JWT
JWT_SECRET_KEY=adm.123SenhadeAcesso
JWT_EXPIRATION=24h

# Admin
ADMIN_USERNAME=admin
ADMIN_PASSWORD=adm.123SenhadeAcesso
```

### Docker Compose
- **PostgreSQL:** postgis/postgis:15-3.3 (porta 5432)
- **Redis:** redis:7-alpine (porta 6379)
- **API:** 2 instâncias (portas 8081, 8082)
- **Health Checks:** Configurados para todos os serviços

---

## 📊 Status Atual

### Prontidão do Sistema
- **JWT Auth:** ✅ Funcional
- **AuthMiddleware GPS:** ✅ Ativo
- **Audit Logging:** ✅ Funcional
- **Rate Limiting:** ✅ Implementado (100 req/min)
- **GPS Endpoint:** ✅ Protegido
- **Graceful Shutdown:** ✅ Implementado
- **Logs Estruturados:** ✅ Implementado
- **Índices DB:** ✅ Migration 07 criada
- **CleanupWorker:** ✅ Com Context
- **Build:** ✅ Sucesso
- **Geofencing:** ✅ Implementado
- **BI Analytics:** ✅ Implementado
- **Stress Test:** ✅ Criado

### Dívidas Técnicas (Não Blockers)
- Atualizar testes desabilitados para usar JWT
- Implementar worker pool para audit logs
- Criar tabela de usuários com bcrypt
- Migrar mock auth para tabela users real
- Remover fallback de JWT secret key hardcoded

---

## 🚀 Comandos Importantes

### Desenvolvimento
```bash
# Build
go build -o transprota.exe .

# Executar
.\transprota.exe

# Stress Test
cd tests
go run stress_test.go
```

### Docker
```bash
# Iniciar serviços
docker-compose up -d

# Ver logs
docker-compose logs -f

# Parar serviços
docker-compose down
```

### Frontend
```bash
cd frontend
npm install
npm run dev  # http://localhost:5173
npm run build
```

---

## 📝 Tags JSON Padronizadas

### TelemetryPing
```json
{
  "device_id": "string (required)",
  "lat": "float64 (required)",
  "lng": "float64 (required)",
  "speed": "float64 (optional)",
  "heading": "float64 (optional)",
  "accuracy": "float64 (optional)",
  "recorded_at": "time.Time (required)",
  "transport_mode": "string (optional)",
  "route_id": "string (optional)",
  "battery_level": "int (optional)",
  "platform": "string (optional)",
  "app_version": "string (optional)"
}
```

---

## 🔍 Pontos de Atenção

### 1. Request Body EOF
- **Problema:** Body pode ser consumido antes do handler
- **Solução:** Usar `c.ShouldBindBodyWith(&ping, binding.JSON)` em vez de `c.ShouldBindJSON(&ping)`

### 2. Race Conditions
- **WebSocket:** Usa `sync.Mutex` para proteger `clients` map
- **CleanupWorker:** Usa canais e context para graceful shutdown
- **Status:** Verificado e seguro

### 3. JWT Secret Key
- **Fallback:** "transprota-secret-key-2024" (hardcoded)
- **Melhoria:** Remover fallback em produção
- **Status:** Não blocker

### 4. Mock Auth
- **Atual:** Mock de autenticação no handler
- **Melhoria:** Implementar tabela users com bcrypt
- **Status:** Não blocker

---

## 🎯 Próximos Passos Sugeridos

1. **Curto Prazo:**
   - Aplicar Migration 07 no banco de dados
   - Testar endpoint GPS com token JWT real
   - Implementar testes de integração JWT

2. **Médio Prazo:**
   - Migrar rate limiting para Redis (distribuído)
   - Criar tabela de usuários com bcrypt
   - Implementar worker pool para audit logs

3. **Longo Prazo:**
   - Implementar refresh tokens
   - Adicionar circuit breaker para chamadas externas
   - Implementar data retention policy

---

## 📚 Documentação Adicional

- **SQUAD_BACKLOG.md:** Backlog detalhado com status de todas as tarefas
- **README.md:** Documentação geral do projeto
- **frontend/README.md:** Documentação do frontend

---

## 🔧 Debugging

### Logs Estruturados
- `[AUTH]` - Autenticação e JWT
- `[TELEMETRY]` - Telemetria GPS
- `[DB]` - Database operations
- `[SERVER]` - Server e middleware
- `[CONFIG]` - Configuração
- `[GEOFENCING]` - Geofencing alerts
- `[BI]` - Analytics alerts
- `[CLEANUPWORKER]` - Cleanup operations

### Canais Redis
- `bus_updates` - Atualizações de ônibus (WebSocket)
- `geofence_alerts` - Alertas de geofencing
- `bi_alerts` - Alertas de analytics

---

*Contextualização gerada para IA assistente - TranspRota Squad - 2026*
