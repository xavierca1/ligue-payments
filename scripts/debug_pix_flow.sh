#!/bin/bash

# Debug script para testar o fluxo PIX completo
# Simula: checkout → webhook → status check

API_HOST="${1:-http://localhost:8080}"
NGROK_HOOK="${2:-}"

echo "====================================="
echo "🧪 Debug: Fluxo PIX Completo"
echo "====================================="
echo "API Host: $API_HOST"
echo ""

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. Primeiro, check health
echo -e "${YELLOW}1. Verificando saúde da API...${NC}"
HEALTH=$(curl -sS "$API_HOST/health" | head -c 100)
if [[ $HEALTH == *"healthy"* ]] || [[ $HEALTH == *"status"* ]]; then
  echo -e "${GREEN}✅ API está respondendo${NC}"
else
  echo -e "${RED}❌ API não está respondendo${NC}"
  exit 1
fi
echo ""

# 2. Criar checkout
echo -e "${YELLOW}2. Criando checkout PIX...${NC}"
CHECKOUT=$(curl -sS -X POST "$API_HOST/checkout" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test User PIX",
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

echo "Response: $(echo $CHECKOUT | head -c 200)..."
CUSTOMER_ID=$(echo "$CHECKOUT" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -z "$CUSTOMER_ID" ]; then
  echo -e "${RED}❌ Erro ao extrair customer ID${NC}"
  echo "Full response: $CHECKOUT"
  exit 1
fi

echo -e "${GREEN}✅ Customer criado: $CUSTOMER_ID${NC}"
echo ""

# 3. Check status ANTES do webhook
echo -e "${YELLOW}3. Verificando status ANTES do webhook...${NC}"
STATUS_BEFORE=$(curl -sS "$API_HOST/customers/$CUSTOMER_ID/status")
echo "Status: $STATUS_BEFORE"
echo ""

# 4. Simular webhook (se NGROK fornecido, usamos real; senão, localhost)
if [ -z "$NGROK_HOOK" ]; then
  WEBHOOK_URL="$API_HOST/webhook"
else
  WEBHOOK_URL="$NGROK_HOOK/webhook"
fi

echo -e "${YELLOW}4. Enviando webhook para: $WEBHOOK_URL${NC}"
echo "Simulando: PAYMENT_CREATED com status RECEIVED"

WEBHOOK=$(curl -sS -X POST "$WEBHOOK_URL" \
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

echo "Webhook Response: $(echo $WEBHOOK | head -c 100)..."
echo ""

# 5. Aguardar 2 segundos e verificar status
echo -e "${YELLOW}5. Aguardando e verificando status APÓS webhook...${NC}"
sleep 2

STATUS_AFTER=$(curl -sS "$API_HOST/customers/$CUSTOMER_ID/status")
echo "Status: $STATUS_AFTER"
echo ""

# 6. Resultado
echo "====================================="
if [[ $STATUS_AFTER == *"ACTIVE"* ]]; then
  echo -e "${GREEN}✅ ✅ ✅ SUCESSO! Status mudou para ACTIVE!${NC}"
else
  echo -e "${RED}❌ FALHA! Status não mudou para ACTIVE${NC}"
  echo "Status antes: $STATUS_BEFORE"
  echo "Status depois: $STATUS_AFTER"
fi
echo "====================================="
