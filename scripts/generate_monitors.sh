#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="$ROOT_DIR/.env"

read_env_value() {
  local key="$1"
  local file="$2"
  if [ ! -f "$file" ]; then
    return 0
  fi
  sed -n "s/^${key}=//p" "$file" | head -1
}

DD_SITE="${DD_SITE:-$(read_env_value DD_SITE "$ENV_FILE")}"
DD_API_KEY="${DD_API_KEY:-$(read_env_value DD_API_KEY "$ENV_FILE")}"
DD_APP_KEY="${DD_APP_KEY:-$(read_env_value DD_APP_KEY "$ENV_FILE")}"

if [ -z "${DD_SITE:-}" ]; then
  DD_SITE="us5.datadoghq.com"
fi

if [ -z "$DD_API_KEY" ]; then
  echo "❌ DD_API_KEY não definido"
  exit 1
fi

if [ -z "$DD_APP_KEY" ]; then
  echo "❌ DD_APP_KEY não definido"
  exit 1
fi

API_URL="https://api.${DD_SITE}/api/v1/monitor"
COMMON_TAGS='["team:sre","app:ligue-payments","env:local","service:ligue-payments","component:checkout"]'

create_monitor() {
  local name="$1"
  local query="$2"
  local message="$3"
  local priority="${4:-3}"
  local critical_threshold="${5:-1}"

  local payload
  payload=$(cat <<JSON
{
  "name": "$name",
  "type": "metric alert",
  "query": "$query",
  "message": "$message",
  "tags": $COMMON_TAGS,
  "priority": $priority,
  "options": {
    "include_tags": true,
    "require_full_window": false,
    "notify_audit": false,
    "renotify_interval": 30,
    "evaluation_delay": 60,
    "thresholds": {
      "critical": ${critical_threshold}
    }
  }
}
JSON
)

  response=$(curl -sS -w "\nHTTP_STATUS:%{http_code}" -X POST "$API_URL" \
    -H "Content-Type: application/json" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
    --data-binary "$payload")

  status=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTP_STATUS://')
  body=$(echo "$response" | sed -e 's/HTTP_STATUS\:.*//g')

  if [[ "$status" =~ ^2 ]]; then
    id=$(echo "$body" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*\([0-9][0-9]*\).*/\1/p' | head -1)
    echo "✅ Monitor criado: $name (id=${id:-n/a})"
  else
    echo "❌ Falha ao criar monitor: $name"
    echo "$body"
    exit 1
  fi
}

create_monitor \
  "[Checkout] Erros 5xx > 0 (5m)" \
  "sum(last_5m):sum:ligue_payments.http.requests{env:local,service:ligue-payments,path:/checkout,status:5*}.as_count() > 0" \
  "🚨 5xx detectado no /checkout nas últimas 5 min.\n\nVerifique logs e dependências externas." \
  "3" \
  "0"

create_monitor \
  "[Checkout] Taxa de erro 4xx alta (5m)" \
  "sum(last_5m):sum:ligue_payments.http.requests{env:local,service:ligue-payments,path:/checkout,status:4*}.as_count() > 50" \
  "⚠️ Alto volume de 4xx no /checkout nas últimas 5 min.\n\nPode indicar payload inválido, problema de validação ou abuso de tráfego." \
  "4" \
  "50"

create_monitor \
  "[Checkout] Latência média alta > 1s (5m)" \
  "avg(last_5m):avg:ligue_payments.http.request_duration{env:local,service:ligue-payments,path:/checkout} > 1000" \
  "🐢 Latência média do /checkout acima de 1s nas últimas 5 min.\n\nInvestigue DB, gateway e saturação de recursos." \
  "2" \
  "1000"

echo "✅ Monitores criados com sucesso"
