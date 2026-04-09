# 🚀 Guia de Deployment - TranspRota

## 📋 Pré-requisitos

- Docker & Docker Compose
- Node.js 18+ (para frontend)
- Go 1.21+ (para backend, opcional se usando Docker)
- PostgreSQL 15+ com PostGIS
- Redis 7+

## 🏃 Quick Start - Local (Docker Compose)

### 1. Clonar o Repositório

```bash
git clone <repo>
cd projetoTransprota
```

### 2. Configurar Variáveis de Ambiente

```bash
# Copiar arquivo de exemplo
cp .env.example .env

# Editar .env com suas configurações
nano .env
```

**Variáveis principais:**
```env
DB_USER=admin
DB_PASSWORD=password123
DB_NAME=transprota
DB_HOST=postgres
DB_PORT=5432
REDIS_ADDR=redis:6379
API_SECRET_KEY=sua-chave-secreta-super-segura
GIN_MODE=release
```

### 3. Iniciar com Docker Compose

```bash
# Construir e iniciar todos os serviços
docker-compose up -d

# Verificar status
docker-compose ps

# Ver logs
docker-compose logs -f api
```

**Serviços iniciados:**
- PostgreSQL (porta 5432)
- Redis (porta 6379)
- API TranspRota (porta 8080)
- Frontend React (porta 5173)
- Swagger UI (porta 3000)

### 4. Verificar Saúde

```bash
# Health check da API
curl http://localhost:8080/health

# Acessar frontend
open http://localhost:5173

# Acessar Swagger
open http://localhost:3000
```

## 🐳 Deployment em Kubernetes

### 1. Build da Imagem Docker

```bash
docker build -t seu-registry/transprota:latest .
docker push seu-registry/transprota:latest
```

### 2. Kubernetes Manifests

**deployment.yaml:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: transprota-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: transprota-api
  template:
    metadata:
      labels:
        app: transprota-api
    spec:
      containers:
      - name: api
        image: seu-registry/transprota:latest
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          value: "postgres-service"
        - name: REDIS_ADDR
          value: "redis-service:6379"
        - name: API_SECRET_KEY
          valueFrom:
            secretKeyRef:
              name: transprota-secrets
              key: api-key
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: transprota-api-service
spec:
  type: LoadBalancer
  selector:
    app: transprota-api
  ports:
  - port: 80
    targetPort: 8080
```

### 3. Deploy

```bash
kubectl apply -f deployment.yaml
kubectl get pods -l app=transprota-api
kubectl logs -f <pod-name>
```

## ☁️ Deployment em Cloud (AWS ECS)

### 1. ECR - Elastic Container Registry

```bash
# Criar repositório
aws ecr create-repository --repository-name transprota

# Login
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin <account>.dkr.ecr.us-east-1.amazonaws.com

# Build e push
docker build -t transprota:latest .
docker tag transprota:latest <account>.dkr.ecr.us-east-1.amazonaws.com/transprota:latest
docker push <account>.dkr.ecr.us-east-1.amazonaws.com/transprota:latest
```

### 2. ECS Task Definition

```json
{
  "name": "transprota",
  "image": "<account>.dkr.ecr.us-east-1.amazonaws.com/transprota:latest",
  "memory": 512,
  "cpu": 256,
  "portMappings": [
    {
      "containerPort": 8080,
      "hostPort": 8080
    }
  ],
  "environment": [
    {
      "name": "DB_HOST",
      "value": "postgres.xxxxx.rds.amazonaws.com"
    },
    {
      "name": "REDIS_ADDR",
      "value": "redis.xxxxx.elasticache.amazonaws.com:6379"
    }
  ],
  "logConfiguration": {
    "logDriver": "awslogs",
    "options": {
      "awslogs-group": "/ecs/transprota",
      "awslogs-region": "us-east-1",
      "awslogs-stream-prefix": "ecs"
    }
  }
}
```

## 📊 Monitoramento

Veja [MONITORING.md](./MONITORING.md)

## 🔄 CI/CD - GitHub Actions

**.github/workflows/deploy.yml:**
```yaml
name: Deploy TranspRota

on:
  push:
    branches: [main]

jobs:
  build-and-deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Build Docker Image
        run: docker build -t transprota:${{ github.sha }} .
      
      - name: Run Tests
        run: go test ./...
      
      - name: Push to Registry
        run: |
          docker tag transprota:${{ github.sha }} registry.com/transprota:latest
          docker push registry.com/transprota:latest
      
      - name: Deploy
        run: kubectl set image deployment/transprota-api api=registry.com/transprota:latest
```

## 🚨 Troubleshooting

### API não inicia
```bash
docker logs transprota_api
# Verificar variáveis de ambiente
# Verificar conectividade com PostgreSQL e Redis
```

### Erro de conexão com PostgreSQL
```bash
docker exec transprota_db psql -U admin -d transprota -c "\dt"
```

### Redis indisponível
```bash
docker exec transprota_cache redis-cli ping
```

## ✅ Checklist de Deploy

- [ ] Variáveis de ambiente configuradas
- [ ] Banco de dados migrado (schema criado)
- [ ] Redis iniciado
- [ ] Health check respondendo 200
- [ ] API respondendo /linhas, /planejar, /terminais
- [ ] Frontend rodando e conectando à API
- [ ] Monitoramento configurado
- [ ] Backups configurados
- [ ] SSL/TLS ativado
- [ ] Rate limiting ativo

## 📞 Suporte

Para problemas:
1. Verificar logs: `docker logs transprota_api`
2. Verificar saúde: `curl http://localhost:8080/health`
3. Testar conectividade: Postgres, Redis
4. Abrir issue no repositório
