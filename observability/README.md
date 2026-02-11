# 📊 Observabilidade - Ligue Payments

Stack completa de monitoramento self-hosted com Grafana, Prometheus e Loki.

## 🚀 Setup Rápido

### 1. Instalar dependências Go

```bash
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promauto
go get github.com/prometheus/client_golang/prometheus/promhttp
```

### 2. Subir stack de monitoramento

```bash
# Criar diretórios necessários
mkdir -p logs observability/dashboards

# Subir containers
docker-compose -f docker-compose.monitoring.yml up -d

# Verificar se subiu
docker ps
```

### 3. Acessar Dashboards

- **Grafana**: http://localhost:3000 (admin / ligue2026)
- **Prometheus**: http://localhost:9090
- **Loki**: http://localhost:3100

### 4. Importar Dashboard Pronto

1. Acesse Grafana → Dashboards → Import
2. Cole o JSON de `observability/dashboards/ligue-payments.json`
3. Selecione datasource Prometheus
4. Pronto! 🎉

## 📈 Métricas Disponíveis

### HTTP Requests
- `http_requests_total` - Total de requisições por método, path e status
- `http_request_duration_seconds` - Latência das requisições
- `http_active_connections` - Conexões ativas

### Business Metrics
- `payments_received_total` - Pagamentos recebidos (PIX/Cartão)
- `subscriptions_activated_total` - Assinaturas ativadas
- `integration_errors_total` - Erros de integração (Asaas, Doc24, etc)

### System Health
- `/health` - Status detalhado de dependências
- `/metrics` - Endpoint Prometheus
- `/healthz` & `/ready` - Kubernetes probes

## 🔍 Queries Úteis (Prometheus)

```promql
# Taxa de requisições por segundo
rate(http_requests_total[5m])

# Latência P95
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Taxa de erro (>= 400)
rate(http_requests_total{status=~"4..|5.."}[5m])

# Pagamentos por hora
increase(payments_received_total[1h])
```

## 📋 Dashboard Mostra

✅ **Requests por segundo** (gráfico de linha)  
✅ **Latência P50/P95/P99** (gauges)  
✅ **Taxa de erro** (%)  
✅ **Top 10 IPs fazendo requests** (tabela)  
✅ **Pagamentos recebidos** (counter)  
✅ **Métodos de pagamento** (pie chart)  
✅ **Status de dependências** (health checks)  
✅ **Logs em tempo real** (Loki)

## 🎯 Alertas Recomendados

Configure no Grafana:
- Taxa de erro > 5% por 5min → Alerta
- Latência P95 > 2s → Warning
- Database down → Critical
- RabbitMQ fila > 100 mensagens → Warning

## 🔧 Deploy em Produção

### No EC2:

```bash
# 1. Clonar repo
git clone seu-repo && cd ligue-payments

# 2. Subir stack
docker-compose -f docker-compose.monitoring.yml up -d

# 3. Configurar nginx para proxy reverso
sudo nano /etc/nginx/sites-available/grafana

# /etc/nginx/sites-available/grafana
server {
    listen 80;
    server_name grafana.seudominio.com;

    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}

# 4. Ativar e recarregar
sudo ln -s /etc/nginx/sites-available/grafana /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

### Recursos Necessários:
- RAM: ~500MB para toda stack
- CPU: Negligível
- Disco: ~1GB/dia de logs (configurável)

## 🛡️ Segurança

Troque senha padrão do Grafana:
```bash
docker exec -it grafana grafana-cli admin reset-admin-password NovaSenhaSegura123
```

## 📱 Alertas por Telegram/Slack

Configure em Grafana → Alerting → Contact points

