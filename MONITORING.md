# 📊 Monitoramento e Observabilidade

Este documento descreve como monitorar e observar o TranspRota em produção.

## 🔍 Health Checks

### Endpoint `/health`

Verifica a saúde da API e suas dependências:

```bash
curl http://localhost:8080/health
```

**Resposta:**
```json
{
  "status": "ok",
  "timestamp": "2026-04-08T14:30:00Z",
  "database": "ok",
  "redis": "ok",
  "uptime": 3600
}
```

**Status Possíveis:**
- `ok`: Todas as dependências estão saudáveis
- `degraded`: Uma ou mais dependências estão com problemas
- `offline`: API indisponível

## 📈 Métricas

### Endpoint `/metrics`

Retorna métricas em formato Prometheus:

```bash
curl http://localhost:8080/metrics
```

**Métricas Disponíveis:**

```
transprota_uptime_seconds        # Tempo de funcionamento em segundos
transprota_requests_total        # Total de requisições
transprota_errors_total          # Total de erros
transprota_error_rate            # Taxa de erro em percentual
```

## 🔗 Integração com Prometheus

Para monitorar com Prometheus, adicione ao `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'transprota'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

## 📊 Dashboards Recomendados

### Grafana

1. **Uptime**: Gráfico de tempo de funcionamento
2. **Requests/Segundo**: Taxa de requisições
3. **Taxa de Erro**: Percentual de requisições com erro
4. **Saúde do Banco de Dados**: Status de conexão PostgreSQL
5. **Saúde do Redis**: Status de conexão Redis

## 🚨 Alertas Sugeridos

### PagerDuty / Prometheus Alertmanager

```yaml
groups:
  - name: transprota
    rules:
      - alert: HighErrorRate
        expr: transprota_error_rate > 5
        for: 5m
        annotations:
          summary: "Taxa de erro acima de 5% por 5 minutos"

      - alert: APIDown
        expr: up{job="transprota"} == 0
        for: 1m
        annotations:
          summary: "API TranspRota está offline"

      - alert: DatabaseUnhealthy
        expr: transprota_database_status != 1
        for: 2m
        annotations:
          summary: "PostgreSQL indisponível"

      - alert: RedisUnhealthy
        expr: transprota_redis_status != 1
        for: 2m
        annotations:
          summary: "Redis indisponível"
```

## 📝 Logging

### Logs Estruturados

Todos os logs incluem:
- Timestamp
- Nível (INFO, ERROR, WARN)
- Mensagem
- Detalhes contextuais

**Exemplos:**
```
❌ Erro ao consultar linhas: connection refused
✅ Bancos conectados com sucesso!
⚠️ Falha ao gravar cache de rota: timeout
```

### Docker Logs

```bash
# Visualizar logs da API
docker logs transprota_api

# Seguir logs em tempo real
docker logs -f transprota_api

# Últimas 100 linhas
docker logs --tail 100 transprota_api

# Com timestamps
docker logs -f --timestamps transprota_api
```

## 🔍 Debugging

### Verificar Saúde Completa

```bash
#!/bin/bash

echo "=== API Health ==="
curl -s http://localhost:8080/health | jq .

echo -e "\n=== Database Connection ==="
docker exec transprota_db psql -U admin -d transprota -c "SELECT version();"

echo -e "\n=== Redis Connection ==="
docker exec transprota_cache redis-cli ping

echo -e "\n=== API Metrics ==="
curl -s http://localhost:8080/metrics | head -20
```

### Performance Profiling

```bash
# Go pprof CPU profile
curl http://localhost:8080/debug/pprof/profile > cpu.prof
go tool pprof cpu.prof

# Memory profile
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

## 📌 Checklist de Monitoramento em Produção

- [ ] Prometheus scraping `/metrics` a cada 15s
- [ ] Alertas configurados para taxa de erro > 5%
- [ ] Alertas para downtime de dependências
- [ ] Grafana dashboard com métricas principais
- [ ] Logs centralizados (ELK, Stackdriver, etc.)
- [ ] Health checks do k8s/Docker com liveness probe
- [ ] Backup automático de PostgreSQL
- [ ] Replicação de Redis para HA
- [ ] Rate limiting ativo (100ms/IP)
- [ ] Autoscaling configurado

## 🔐 Considerações de Segurança

- O endpoint `/metrics` é público (considere proteger em produção)
- Health check não expõe senhas ou dados sensíveis
- Logs não contêm credentials
- SSL/TLS ativado em produção

## 📞 Contato e Suporte

Em caso de problemas:
1. Verifique o endpoint `/health`
2. Consulte os logs (`docker logs -f transprota_api`)
3. Teste a conectividade com PostgreSQL e Redis
4. Abra um issue no repositório
