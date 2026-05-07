#!/bin/bash
set -u

# ==============================================================================
# Script para deploy de Dashboard no Datadog via API (US5)
# ==============================================================================

# Caminho do dashboard do projeto
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DASHBOARD_FILE="$ROOT_DIR/observability/datadog-dashboard.json"

# Credenciais
DD_API_KEY="a227789716df550d412bb855af984c16"
DD_APP_KEY="ddapp_yJXa0aIGcCZoiP4CWMu4wpUB23wc1rZSf3"

# Endpoint correto para a região US5
API_URL="https://api.us5.datadoghq.com/api/v1/dashboard"

echo "🚀 Iniciando o deploy do dashboard no Datadog..."
echo "📂 Lendo o arquivo: ${DASHBOARD_FILE}"

if [ ! -f "$DASHBOARD_FILE" ]; then
    echo "❌ Arquivo não encontrado: ${DASHBOARD_FILE}"
    exit 1
fi

# Disparo do cURL
HTTP_RESPONSE=$(curl -sS -w "\nHTTP_STATUS:%{http_code}" -X POST "$API_URL" \
    -H "Content-Type: application/json" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
    --data-binary "@${DASHBOARD_FILE}" 2>&1)

# Tratamento simples para ver se deu bom
HTTP_BODY=$(echo "$HTTP_RESPONSE" | sed -e 's/HTTP_STATUS\:.*//g')
HTTP_STATUS=$(echo "$HTTP_RESPONSE" | tr -d '\n' | sed -e 's/.*HTTP_STATUS://')

echo -e "\n📊 Status da Resposta: ${HTTP_STATUS}"

if ! [[ "$HTTP_STATUS" =~ ^[0-9]+$ ]]; then
    echo "❌ Não foi possível obter status HTTP (curl falhou antes da resposta)."
    echo "$HTTP_BODY"
    exit 1
fi

if [ "$HTTP_STATUS" -eq 200 ]; then
    echo "✅ Sucesso! O painel do hub de pagamentos já deve estar brilhando lá na conta."
else
    echo "❌ Ops, algo deu errado:"
    echo "$HTTP_BODY"
fi