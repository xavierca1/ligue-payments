#!/bin/bash

# Debug: Simular fluxo PIX completo - checkout + webhook + status check

set -e

API_HOST="${1:-http://localhost:8080}"

echo "================================"
echo "🧪 Test PIX Complete Flow"
echo "================================"
echo "API: $API_HOST"
echo ""

# 1. Health check
echo "1️⃣  Checando saúde da API..."
if ! curl -sS "$API_HOST/health" > /dev/null 2>&1; then
  echo "❌ API não está respondendo em $API_HOST"
  exit 1
fi
echo "✅ API OK"
echo ""

# 2. Criar checkout
echo "2️⃣  Criando checkout PIX..."
CHECKOUT_RESPONSE=$(curl -sS -X POST "$API_HOST/checkout" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test PIX User",
    "email": "test.pix@example.com",
    "cpf": "12345678901",
    "phone": "11999999999",
    "birthDate": "1990-01-01",
    "gender": "1",
    "planId": "8ab92cfb-8e60-4ec2-b39d-4a44bcb1d2f3",
    "paymentMethod": "PIX",
    "street": "Rua Test",
    "number": "123",
    "district": "Centro",
    "city": "São Paulo",
    "state": "SP",
    "zipCode": "01234567",
    "termsAccepted": true,
    "termsVersion": "1.0"
  }')

CUSTOMER_ID=$(echo "$CHECKOUT_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -z "$CUSTOMER_ID" ]; then
  echo "❌ Erro ao extrair customer ID"
  echo "Response: $CHECKOUT_RESPONSE"
  exit 1
fi

echo "✅ Customer criado: $CUSTOMER_ID"
echo ""

# 3. Check status ANTES
echo "3️⃣  Status ANTES do webhook:"
STATUS_BEFORE=$(curl -sS "$API_HOST/customers/$CUSTOMER_ID/status" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
echo "   Status: $STATUS_BEFORE"
echo ""

# 4. Simular webhook
echo "4️⃣  Enviando webhook PAYMENT_CREATED (status=RECEIVED)..."
WEBHOOK_RESPONSE=$(curl -sS -X POST "$API_HOST/webhook" \
  -H "Content-Type: application/json" \
  -H "X-Asaas-Signature: dummy" \
  -d '{
    "event": "PAYMENT_CREATED",
    "payment": {
      "id": "pay_test_'$(date +%s)'",
      "customer": "'$CUSTOMER_ID'",
      "status": "RECEIVED"
    }
  }')

echo "   Webhook Response: $(echo $WEBHOOK_RESPONSE | head -c 100)"
echo ""

# 5. Aguardar e verificar
echo "5️⃣  Aguardando 2s e verificando status APÓS webhook..."
sleep 2
STATUS_AFTER=$(curl -sS "$API_HOST/customers/$CUSTOMER_ID/status" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
echo "   Status: $STATUS_AFTER"
echo ""

# 6. Resultado
echo "================================"
if [ "$STATUS_AFTER" = "ACTIVE" ]; then
  echo "✅ ✅ ✅ SUCESSO!"
  echo "Status mudou de $STATUS_BEFORE → $STATUS_AFTER"
else
  echo "❌ ❌ ❌ FALHA!"
  echo "Status antes: $STATUS_BEFORE"
  echo "Status depois: $STATUS_AFTER (esperava ACTIVE)"
fi
echo "================================"
