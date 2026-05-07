# 🧪 Teste de Checkout no Postman

## Endpoint

```
POST http://localhost:3000/checkout
Content-Type: application/json
```

## Payloads Disponíveis

### 1. Cartão de Crédito (Simples)
**Arquivo:** `POSTMAN_CHECKOUT_PAYLOAD_COMPLETO.json`

```json
{
  "name": "João Silva Santos",
  "email": "joao.silva@test.com",
  "cpf": "12345678901",
  "plan_id": "plan_001",
  "phone": "(11) 98765-4321",
  "payment_method": "CREDIT_CARD",
  "street": "Rua das Flores",
  "number": "123",
  "district": "Centro",
  "city": "São Paulo",
  "state": "SP",
  "zip_code": "01310-100",
  "birth_date": "1990-05-15",
  "gender": "M",
  "card_holder": "JOAO SILVA SANTOS",
  "card_number": "4111111111111111",
  "card_month": "12",
  "card_year": "2025",
  "card_cvv": "123",
  "terms_accepted": true,
  "terms_accepted_at": "2025-05-07T10:30:00Z",
  "terms_version": "1.0"
}
```

### 2. PIX com Dependentes
**Arquivo:** `POSTMAN_CHECKOUT_PIX_COM_DEPENDENTES.json`

```json
{
  "name": "Maria Oliveira Costa",
  "email": "maria.oliveira@test.com",
  "cpf": "98765432109",
  "plan_id": "plan_002",
  "phone": "(21) 99999-8888",
  "coupon_code": "PROMO10",
  "payment_method": "PIX",
  "street": "Avenida Paulista",
  "number": "1000",
  "district": "Bela Vista",
  "city": "São Paulo",
  "state": "SP",
  "zip_code": "01311-100",
  "birth_date": "1985-03-22",
  "gender": "F",
  "terms_accepted": true,
  "terms_accepted_at": "2025-05-07T14:45:00Z",
  "terms_version": "1.0",
  "dependents": [
    {
      "name": "Filho Silva",
      "birth_date": "2010-06-15",
      "gender": "M",
      "relationship": "filho",
      "cpf": "12345678902"
    }
  ]
}
```

## 🔍 Monitorar Logs

Quando fizer o teste, procure nos logs por:

1. **Debug do Customer:**
```
🔍 [CUSTOMER DEBUG - OBJETO COMPLETO]:
```
Vai mostrar o objeto customer completo com a estrutura real do endereço

2. **Debug do DocuSeal:**
```
🔍 [DOCUSEAL DEBUG] Campos de Endereço Recebidos:
   Endereco: ...
   Numero: ...
   Bairro: ...
   ...
```

3. **Documento criado:**
```
✅ Documento DocuSeal gerado automaticamente (UUID=...)
```

## 📧 Resposta Esperada

```json
{
  "id": "cust_xxx...",
  "name": "João Silva Santos",
  "email": "joao.silva@test.com",
  "status": "PENDING",
  "msg": "Cadastro realizado com sucesso",
  "pix_code": "",
  "pix_qr_code_url": ""
}
```

## ⚠️ Observações

- **CPF:** Mude o CPF a cada teste (não pode ter duplicado)
- **Email:** Mude o email a cada teste (não pode ter duplicado)
- **Card:** Use cartão de teste `4111111111111111`
- **Endereço:** Os campos de endereço são os que estamos debugando

## 🎯 O que Está Sendo Testado

1. ✅ Recebimento do payload
2. ✅ Validação dos dados
3. ✅ Salvamento do customer no banco
4. ✅ Ativação da assinatura
5. ✅ **Debug do objeto customer (para ver estrutura real do endereço)**
6. ✅ **Geração do documento DocuSeal com campos de endereço**
7. ✅ Envio de email
