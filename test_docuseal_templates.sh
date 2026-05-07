#!/bin/bash

# Script para testar todos os templates do DocuSeal
# Uso: ./test_docuseal_templates.sh

source .env

BASE_URL="http://localhost:8080"

# Dados de teste
TEST_EMAIL="test@example.com"

# Dados comuns para todos os templates
COMMON_FIELDS='{
  "product": "Test Product",
  "id": "123456",
  "method_payment": "credit_card",
  "periodicidade": "monthly",
  "name": "João da Silva",
  "birthdate": "01/01/1990",
  "cpf": "123.456.789-10",
  "genre": "Masculino",
  "marital_status": "Solteiro",
  "cellphone": "(11) 98765-4321",
  "email": "joao@example.com",
  "address": "Rua Principal",
  "number": "100",
  "neighborhood": "Centro",
  "city": "São Paulo",
  "UF": "SP",
  "zip_code": "01310-100"
}'

# Templates a testar
declare -A TEMPLATES=(
  ["ligue_saude_em_dia"]="Ligue Saúde em Dia"
  ["ligue_mais_cuidado"]="Ligue Mais Cuidado"
  ["ligue_vida_plena"]="Ligue Vida Plena"
  ["ligue_cuidado_total"]="Ligue Cuidado Total"
  ["ligue_viver_bem"]="Ligue Viver Bem"
)

echo "🚀 Testando templates DocuSeal..."
echo "================================"

for template_name in "${!TEMPLATES[@]}"; do
  template_display="${TEMPLATES[$template_name]}"
  echo ""
  echo "📋 Testando: $template_display ($template_name)"
  
  RESPONSE=$(curl -s -X POST "$BASE_URL/docuseal/test" \
    -H "Content-Type: application/json" \
    -d "{
      \"email\": \"$TEST_EMAIL\",
      \"template\": \"$template_name\",
      \"fields\": $COMMON_FIELDS
    }")
  
  UUID=$(echo "$RESPONSE" | jq -r '.uuid // empty')
  ERROR=$(echo "$RESPONSE" | jq -r '.error // empty')
  
  if [ -n "$UUID" ]; then
    echo "✅ Sucesso! UUID: $UUID"
  else
    if [ -n "$ERROR" ]; then
      echo "❌ Erro: $ERROR"
    else
      echo "❌ Erro desconhecido"
      echo "Resposta: $RESPONSE"
    fi
  fi
done

echo ""
echo "================================"
echo "✨ Teste de templates concluído!"
