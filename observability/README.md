# 📊 Observabilidade - Datadog

O projeto agora envia métricas para o Datadog via DogStatsD.

## Setup rápido

1. Configure as variáveis de ambiente:

```bash
export DD_API_KEY=seu_token
export DD_SITE=datadoghq.com
export DD_ENV=local
export DD_SERVICE=ligue-payments
```

2. Suba a stack com o agente:

```bash
docker-compose up -d
```

3. Confira no Datadog as métricas com prefixo `ligue_payments.`.

## Métricas enviadas

- `http.requests`
- `http.request_duration`
- `http.active_connections`
- `queue.messages_published`
- `queue.messages_consumed`
- `queue.size`
- `queue.processing_duration`
- `queue.message_arrival_latency`
- `integration.errors`

## Tags padrão

- `service:ligue-payments`
- `env:<ambiente>`
- `method:<verbo>`
- `path:<rota>`
- `status:<código>`

## Dashboard pronto (import)

Arquivo pronto para importar no Datadog:

- `observability/datadog-dashboard.json`

Passos:

1. Datadog → Dashboards → New Dashboard
2. Clique em **Import from JSON**
3. Cole o conteúdo de `observability/datadog-dashboard.json`
4. Salve o dashboard

O dashboard já inclui widgets para:

- Requests HTTP
- Latência média de request
- Conexões ativas
- Pagamentos recebidos
- Assinaturas ativadas
- Erros de integração

