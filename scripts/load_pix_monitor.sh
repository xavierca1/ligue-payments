#!/bin/bash
set -euo pipefail

API_HOST="${API_HOST:-http://localhost:8080}"
CHECKOUT_PATH="${CHECKOUT_PATH:-/checkout}"
TOTAL_REQUESTS="${TOTAL_REQUESTS:-60}"
CONCURRENCY="${CONCURRENCY:-12}"
PLAN_ID="${PLAN_ID:-8ab92cfb-8e60-4ec2-b39d-4a44bcb1d2f3}"
DOMAIN="${EMAIL_DOMAIN:-example.com}"

if ! [[ "$TOTAL_REQUESTS" =~ ^[0-9]+$ ]] || [ "$TOTAL_REQUESTS" -le 0 ]; then
  echo "TOTAL_REQUESTS inválido: $TOTAL_REQUESTS"
  exit 1
fi

if ! [[ "$CONCURRENCY" =~ ^[0-9]+$ ]] || [ "$CONCURRENCY" -le 0 ]; then
  echo "CONCURRENCY inválido: $CONCURRENCY"
  exit 1
fi

TMP_RESULTS="$(mktemp)"
START_TS="$(date +%s)"

calc_digit() {
  local numbers="$1"
  local factor="$2"
  local sum=0
  local i digit

  for ((i=0; i<${#numbers}; i++)); do
    digit="${numbers:$i:1}"
    sum=$((sum + digit * factor))
    factor=$((factor - 1))
  done

  local d=$(((sum * 10) % 11))
  if [ "$d" -eq 10 ]; then
    d=0
  fi
  echo "$d"
}

generate_valid_cpf() {
  local seed="$1"
  local base
  base="$(printf "%09d" "$((seed % 1000000000))")"

  # Evita sequências óbvias repetidas
  if [[ "$base" =~ ^([0-9])\1{8}$ ]]; then
    base="12345678${seed: -1}"
  fi

  local d1 d2
  d1="$(calc_digit "$base" 10)"
  d2="$(calc_digit "${base}${d1}" 11)"
  echo "${base}${d1}${d2}"
}

run_one() {
  local i="$1"
  local seed cpf email payload code
  seed="$((START_TS + i))"
  cpf="$(generate_valid_cpf "$seed")"
  email="loadpix+${seed}@${DOMAIN}"

  payload=$(cat <<JSON
{
  "name": "Load Test PIX ${i}",
  "email": "${email}",
  "cpf": "${cpf}",
  "phone": "1199$(printf '%06d' "$i")",
  "birth_date": "1990-01-01",
  "gender": "1",
  "plan_id": "${PLAN_ID}",
  "payment_method": "PIX",
  "street": "Rua Teste",
  "number": "123",
  "district": "Centro",
  "city": "Sao Paulo",
  "state": "SP",
  "zip_code": "01234567",
  "terms_accepted": true,
  "terms_accepted_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "terms_version": "1.0"
}
JSON
)

  code=$(curl -sS -o /dev/null -w "%{http_code}" -X POST "${API_HOST}${CHECKOUT_PATH}" \
    -H "Content-Type: application/json" \
    -d "$payload" || echo "000")

  echo "$code" >> "$TMP_RESULTS"
}

export -f calc_digit generate_valid_cpf run_one
export API_HOST CHECKOUT_PATH PLAN_ID DOMAIN TMP_RESULTS START_TS

seq 1 "$TOTAL_REQUESTS" | xargs -I{} -P "$CONCURRENCY" bash -c 'run_one "$@"' _ {}

TOTAL_DONE="$(wc -l < "$TMP_RESULTS" | tr -d ' ')"
S201="$(grep -c '^201$' "$TMP_RESULTS" || true)"
S400="$(grep -c '^400$' "$TMP_RESULTS" || true)"
S401="$(grep -c '^401$' "$TMP_RESULTS" || true)"
S409="$(grep -c '^409$' "$TMP_RESULTS" || true)"
S500="$(grep -c '^500$' "$TMP_RESULTS" || true)"
S000="$(grep -c '^000$' "$TMP_RESULTS" || true)"
OTHER="$((TOTAL_DONE - S201 - S400 - S401 - S409 - S500 - S000))"

rm -f "$TMP_RESULTS"

END_TS="$(date +%s)"
DURATION="$((END_TS - START_TS))"
if [ "$DURATION" -le 0 ]; then
  DURATION=1
fi

RPS="$(awk -v t="$TOTAL_DONE" -v d="$DURATION" 'BEGIN { printf "%.2f", t/d }')"

echo "========================================"
echo "LOAD TEST PIX FINALIZADO"
echo "API: ${API_HOST}${CHECKOUT_PATH}"
echo "Requisições: $TOTAL_DONE"
echo "Concorrência: $CONCURRENCY"
echo "Duração: ${DURATION}s"
echo "RPS médio: $RPS"
echo "----------------------------------------"
echo "201: $S201"
echo "400: $S400"
echo "401: $S401"
echo "409: $S409"
echo "500: $S500"
echo "000(erro rede): $S000"
echo "Outros: $OTHER"
echo "========================================"

echo "CPFs gerados são válidos pelo algoritmo oficial de dígito verificador (formato Receita), sem consulta cadastral em base governamental."
