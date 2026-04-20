# TranspRota - Plataforma Modular de Telemetria de Precisão

**Sistema de monitoramento de transporte público em tempo real com arquitetura modular, JWT authentication, PostGIS geospatial queries e audit logging completo.**

---

## 🎯 Problema Resolvido

❌ **Antes:**
- Horários baseados em planilhas desatualizadas (2019)
- Sem saber aonde o ônibus está AGORA
- Sem previsão de chegada confiável
- Sem opções alternativas de rota

✅ **Depois:**
- GPS em tempo real de ônibus
- Previsão dinâmica de chegada
- Planejador de rota inteligente
- Sistema de denúncias colaborativas

---

## 🛠️ Stack Tecnológica

| Componente | Tecnologia | Propósito |
|-----------|-----------|----------|
| **Backend** | Go 1.21+ | API de alta performance |
| **Frontend** | React 18 | Interface responsiva |
| **Banco de Dados** | PostgreSQL 15 + PostGIS | Dados geoespaciais |
| **Cache** | Redis 7 | GPS em tempo real |
| **Documentação** | OpenAPI 3.0 | Swagger interactive |
| **Deploy** | Docker & Docker Compose | Containerização |
| **Monitoramento** | Prometheus | Métricas e alerts |

---

## 📦 Estrutura do Projeto

```
projetoTransprota/
├── main.go                    # Backend Go (API)
├── main_test.go              # Testes unitários
├── openapi.yaml              # Documentação da API (Swagger)
├── Dockerfile                # Build de imagem
├── docker-compose.yml        # Orquestração de serviços
├── schema_rotas.sql          # Schema do banco de dados
├── MONITORING.md             # Guia de monitoramento
├── DEPLOYMENT.md             # Guia de deploy
│
├── frontend/                 # React Frontend
│   ├── src/
│   │   ├── components/       # RouteCalculator, BusTracker, Reports
│   │   ├── api/             # Cliente Axios
│   │   ├── App.jsx          # Componente raiz
│   │   └── main.jsx         # Entrada
│   ├── package.json
│   ├── vite.config.js
│   └── README.md
│
└── data/
    └── locations.json        # Base de terminais
```

---

## 🚀 Começar (3 minutos)

### 1️⃣ Pré-requisitos
```bash
# Verificar versões
docker --version
docker-compose --version
```

### 2️⃣ Clonar e Configurar
```bash
git clone <repo>
cd projetoTransprota

# Copiar configurações
cp .env.example .env
```

### 3️⃣ Iniciar com Docker Compose
```bash
docker-compose up -d
```

✅ **Pronto!**
- 🔗 API: http://localhost:8080
- 📚 Swagger: http://localhost:3000
- 🗄️ PostgreSQL: localhost:5432
- 🔴 Redis: localhost:6379

---

## 📚 Documentação Completa

### 🔌 Endpoints da API
```bash
# Health checks
GET /health                   # Status da API
GET /api/v1/health            # Status da API (v1)

# Autenticação (JWT)
POST /api/v1/auth/login       # Obter token JWT

# Telemetria GPS
POST /api/v1/telemetry/gps    # Receber ping GPS
GET /api/v1/telemetry/last-position/:device_hash  # Última posição
GET /api/v1/telemetry/ws      # WebSocket para atualizações em tempo real

# Analytics
GET /api/v1/analytics/fleet-health  # Métricas de saúde da frota
```

### 🔐 Exemplos de Uso

#### Login (Obter Token JWT)
```bash
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123"
  }'

# Resposta:
# {
#   "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
#   "expires_in": 86400
# }
```

#### Enviar Telemetria GPS (com autenticação)
```bash
curl -X POST http://localhost:8081/api/v1/telemetry/gps \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -d '{
    "device_id": "bus-001",
    "lat": -16.6869,
    "lng": -49.2648,
    "speed": 45.5,
    "heading": 180,
    "accuracy": 10,
    "recorded_at": "2024-01-01T12:00:00Z"
  }'

# Resposta:
# {
#   "status": "success",
#   "telemetry_id": "tel-12345",
#   "message": "GPS data received and saved"
# }
```

📖 **Documentação Interativa:** [http://localhost:3000](http://localhost:3000) (Swagger UI)

### 🔐 Autenticação
```bash
# Headers obrigatórios para endpoints protegidos
curl -H "X-API-Key: your-secret-key" http://localhost:8080/denuncias
```

### 💳 Trust Score
```
Score = 50 (base)
        + 5 × confirmadas
        - 15 × spam
        + 10 × com_evidencia

Níveis:
  Suspeito       (0-20)   😕
  Cidadão      (21-80)   👤
  Fiscal Galera (81-100) 🚨
```

---

## 🖥️ Frontend React

### 🎨 Componentes
- **RouteCalculator**: Planeja melhor rota entre dois pontos
- **BusTracker**: Rastreia ônibus em tempo real (5s atualização)
- **Reports**: Sistema de denúncias colaborativas
- **Navigation**: Menu principal com status da API

### 🚀 Rodar Frontend
```bash
cd frontend

# Desenvolvimento
npm install
npm run dev              # http://localhost:5173

# Produção
npm run build
npm run preview
```

### 🔗 Integração com API
```javascript
import { api } from './api/client'

// Exemplo: Calcular rota
const route = await api.calcularRota('Vila Pedroso', 'UFG')
```

---

## 🧪 Testes

### Rodar Testes Go
```bash
go test -v ./...

# Resultado esperado:
# ✓ 9/11 tests passed
# ⏩ 2 integration tests skipped (need DB mocks)
```

### Exemplos de Teste
```bash
# Testar API
curl http://localhost:8080/health
curl http://localhost:8080/linhas
curl http://localhost:8080/planejar?origem=Vila+Pedroso&destino=UFG

# Com autenticação
curl -H "X-API-Key: secret-key-change-in-production" \
  http://localhost:8080/denuncias
```

---

## 📊 Monitoramento em Produção

### Health Checks
```bash
curl http://localhost:8080/health
# Retorna status de API, PostgreSQL e Redis
```

### Métricas Prometheus
```bash
curl http://localhost:8080/metrics
# Uptime, total de requisições, taxa de erro
```

📖 **Documentação Completa:** [MONITORING.md](./MONITORING.md)

---

## 🚀 Deploy em Produção

### Com Docker Compose (Simples)
```bash
docker-compose up -d --build
```

### Com Kubernetes (Escalável)
```bash
kubectl apply -f deployment.yaml
```

### Na Cloud (AWS, GCP, Azure)
Ver [DEPLOYMENT.md](./DEPLOYMENT.md)

---

## 🔒 Segurança

- ✅ **Rate Limiting**: 100ms por IP
- ✅ **Validação**: Entrada validada antes de queries
- ✅ **Auth**: X-API-Key com constant-time compare
- ✅ **Geospatial**: Coordenadas validadas antes de uso
- ✅ **Connection Pooling**: Proteção contra connection exhaustion

---

## 📈 Performance

- ⚡ **Roteamento em Cache**: 15 minutos com Redis
- 🎛️ **Sync.Pool**: Reutilização de objetos (reduz GC)
- 🔄 **Goroutines**: Processamento assíncrono
- 📝 **Context Timeouts**: Todas operações I/O têm timeout

---

## 🎯 Status do Projeto

### ✅ Completo (MVP 1.0)

| Feature | Status | Detalhes |
|---------|--------|----------|
| ✅ Planejador de Rotas | Pronto | Diretas + transferências |
| ✅ Rastreamento GPS | Pronto | Tempo real via Redis |
| ✅ Sistema de Denúncias | Pronto | Trust Score implementado |
| ✅ API RESTful | Pronto | 8 endpoints protegidos |
| ✅ Autenticação | Pronto | X-API-Key |
| ✅ Documentação API | Pronto | OpenAPI 3.0 (Swagger) |
| ✅ Rate Limiting | Pronto | 100ms/IP |
| ✅ Testes Unitários | Pronto | 9/11 passing |
| ✅ Docker | Pronto | Compose com 4 serviços |
| ✅ Frontend React | Pronto | 100% funcional |
| ✅ Monitoramento | Pronto | /health, /metrics |

### 🚀 Próximas Fases (v2.0+)

- 📱 App mobile (React Native)
- 🗺️ Mapa interativo (Leaflet)
- 📧 Notificações push
- 👤 Autenticação por OAuth
- 💳 Sistema de créditos
- 🤖 Machine Learning para previsão

---

## 📉 Diagrama de Arquitetura

```
┌─────────────────────────────────────────────────────────────┐
│                      Cliente (Passageiro)                   │
├─────────────────────────────────────────────────────────────┤
│   React Frontend (localhost:5173)                           │
│   - Route Calculator                                        │
│   - Bus Tracker                                             │
│   - Reports (Denúncias)                                     │
└──────────────────────┬──────────────────────────────────────┘
                       │ HTTP/HTTPS
                       ▼
┌─────────────────────────────────────────────────────────────┐
│          GO API (localhost:8080) - Gin Framework            │
├─────────────────────────────────────────────────────────────┤
│ - RateLimitMiddleware                                       │
│ - AuthMiddleware (X-API-Key)                               │
│ - ErrorHandlerMiddleware                                    │
│                                                             │
│ Endpoints:                                                  │
│ GET /linhas, /terminais, /planejar                         │
│ GET/POST /gps, GET /gps/:id/status                         │
│ POST/GET /denuncias (com geospatial)                       │
│ GET /health, /metrics                                       │
└──────────────┬────────────────────────┬──────────────────┬─┘
               │                        │                  │
               ▼                        ▼                  ▼
    ┌──────────────────┐  ┌─────────────────────┐  ┌──────────────┐
    │  PostgreSQL 15   │  │    Redis Cache      │  │   Swagger UI │
    │   + PostGIS      │  │      Cluster        │  │  (localhost) │
    │                  │  │                     │  │              │
    │ - linhas_onibus  │  │ bus:{id} → GPSData │  │ OpenAPI 3.0  │
    │ - itinerarios    │  │ rota:{o}:{d}       │  │              │
    │ - pontos_parada  │  │                     │  │ (Swagger UI) │
    │ - denuncias      │  │ TTL: 10-15min       │  │              │
    │ - locations      │  │                     │  │              │
    └──────────────────┘  └─────────────────────┘  └──────────────┘
         Port 5432              Port 6379            Port 3000
```

---

## 🔗 Links Úteis

- 📚 **API Swagger**: http://localhost:3000
- 🔍 **Health Check**: http://localhost:8080/health
- 📊 **Métricas**: http://localhost:8080/metrics
- 🐘 **pgAdmin** (opcional): http://localhost:5050
- 📖 [OpenAPI Spec](./openapi.yaml)
- 📋 [Guia de Deploy](./DEPLOYMENT.md)
- 📊 [Guia de Monitoring](./MONITORING.md)

---

## 👥 Contribuindo

Este projeto é um trabalho acadêmico. Sugestões e PRs são bem-vindas!

```bash
git checkout -b feature/nova-feature
git commit -m "Adiciona nova feature"
git push origin feature/nova-feature
```

---

## 📄 Licença

MIT - Veja [LICENSE](./LICENSE)

---

## 👨‍💻 Desenvolvedor

**Daniel de Novais Santos Mendonça**
- 📧 Email: daniel@transprota.com
- 🔗 GitHub: [@Daniel-Novais1](https://github.com/Daniel-Novais1)
- 🎓 UFG - Engenharia de Software

---

## 📞 Suporte

Encontrou um bug? Abra uma [issue no GitHub](https://github.com/seu-usuario/transprota/issues)

---

**Feito com ❤️ para a comunidade de Goiânia**
