#!/bin/bash

set -e

API_HOST="${API_HOST:-http://localhost:8080}"
PLAN_ID="${PLAN_ID:-plan_uuid_demo}"

echo "================================"
echo "🧪 PIX Complete Flow Test"
echo "================================"
echo ""

# 1. Create Checkout
echo "1️⃣  Criando checkout com PIX..."
CHECKOUT_RESPONSE=$(curl -sS -X POST "$API_HOST/checkout" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test User PIX",
    "email": "test.pix@example.com",
    "cpf": "12345678901",
    "phone": "11999999999",
    "birthDate": "1990-01-01",
    "gender": "1",
    "planId": "'$PLAN_ID'",
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

echo "Response: $CHECKOUT_RESPONSE"
echo ""

# Extract customer ID
CUSTOMER_ID=$(echo "$CHECKOUT_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "✅ Customer ID: $CUSTOMER_ID"
echo ""

if [ -z "$CUSTOMER_ID" ]; then
  echo "❌ Erro: Não consegui extrair o customer ID"
  exit 1
fi

# 2. Check initial status
echo "2️⃣  Verificando status inicial..."
STATUS=$(curl -sS "$API_HOST/customers/$CUSTOMER_ID/status" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
echo "📊 Status inicial: $STATUS"
echo ""

# 3. Simulate webhook payment confirmation
echo "3️⃣  Simulando webhook de pagamento confirmado (PAYMENT_CREATED com status RECEIVED)..."
WEBHOOK_RESPONSE=$(curl -sS -X POST "$API_HOST/webhook" \
  -H "Content-Type: application/json" \
  -H "X-Asaas-Signature: dummy-signature-bypass" \
  -d '{
    "event": "PAYMENT_CREATED",
    "payment": {
      "id": "pay_test_pix_001",
      "customer": "'$CUSTOMER_ID'",
      "status": "RECEIVED"
    }
  }')

echo "Webhook Response: $WEBHOOK_RESPONSE"
echo ""

# 4. Check status after webhook
echo "4️⃣  Verificando status após webhook..."
sleep 1
STATUS=$(curl -sS "$API_HOST/customers/$CUSTOMER_ID/status" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
echo "📊 Status após webhook: $STATUS"
echo ""

if [ "$STATUS" = "ACTIVE" ]; then
  echo "✅ ✅ ✅ SUCESSO! Subscription foi ativada!"
else
  echo "❌ ❌ ❌ FALHA! Status ainda é: $STATUS (esperava ACTIVE)"
fi

echo ""
echo "Test Complete!"
