# 🔍 Rastreamento de DocuSealContractInput - Análise Completa

## 📍 Locais onde DocuSealContractInput é Instanciada

### 1. **PRINCIPAL: `activate_subscription.go` (linha 135)**
Este é o ponto mais importante - onde o documento é gerado após ativação de assinatura.

**Arquivo:** [internal/usecase/activate_subscription.go](internal/usecase/activate_subscription.go#L135)

```go
docuSealInput := DocuSealContractInput{
    TemplateName:  templateName,
    CustomerID:    customer.ID,
    Nome:          customer.Name,
    Email:         customer.Email,
    CPF:           customer.CPF,
    PlanName:      plan.Name,
    Produto:       plan.Name,
    Valor:         fmt.Sprintf("%.2f", float64(sub.Amount)/100),
    Pagamento:     sub.PaymentMethod,
    Periodicidade: "Mensal",
    Nascimento:    customer.BirthDate,
    Sexo:          genderStr,
    Civil:         "",
    Celular:       customer.Phone,
    Endereco:      customer.Address.Street,    // ← DADOS DE ENDEREÇO
    Numero:        customer.Address.Number,
    Bairro:        customer.Address.District,
    Cidade:        customer.Address.City,
    UF:            customer.Address.State,
    CEP:           customer.Address.ZipCode,
}

submissionUUID, err := uc.DocuSealUseCase.ExecuteAutomatic(ctx, docuSealInput)
```

**Fluxo:**
```
checkout (CreateCheckoutHandler)
  → CreateCustomerUseCase.Execute()
    → ActivateSubscriptionUseCase.Execute()  ← AQUI! (linha 135)
      → DocuSealUseCase.ExecuteAutomatic()
        → DocuSealClient.CreateSubmission()
```

**PROBLEMA IDENTIFICADO:**
Os campos de endereço vêm de `customer.Address` que é uma struct `entity.Customer`:
- `customer.Address.Street` → `Endereco`
- `customer.Address.Number` → `Numero`
- `customer.Address.District` → `Bairro`
- `customer.Address.City` → `Cidade`
- `customer.Address.State` → `UF`
- `customer.Address.ZipCode` → `CEP`

---

### 2. **TESTE: `generate_contract_docuseal_test.go` (linha 36)**
Apenas para testes unitários - NOT used in production

**Arquivo:** [internal/usecase/generate_contract_docuseal_test.go](internal/usecase/generate_contract_docuseal_test.go#L36)

```go
input := DocuSealContractInput{
    TemplateName: "ligue_saude_em_dia",
    CustomerID:   "CUST-TEST-001",
    Nome:         "João da Silva",
    Email:        "teste@example.com",
    CPF:          "12345678901",
    PlanName:     "Saúde em Dia",
    Produto:      "Saúde em Dia",
    Valor:        "99.90",
    Pagamento:    "PIX",
    Nascimento:   "1990-05-15",
    Sexo:         "M",
    Civil:        "Solteiro",
    Celular:      "(11) 99999-8888",
    Endereco:     "Rua das Flores",
    Numero:       "123",
    Bairro:       "Centro",
    Cidade:       "São Paulo",
    UF:           "SP",
    CEP:          "01310-100",
}
```

---

### 3. **REFERÊNCIA: `DOCUSEAL_INTEGRATION_EXAMPLE.go` (linha 63)**
Apenas comentário/exemplo - NOT used

---

## 🎯 Causa Provável dos Campos Vazios

O problema está em `activate_subscription.go` linha 135-151:

```go
Endereco:      customer.Address.Street,
Numero:        customer.Address.Number,
Bairro:        customer.Address.District,
Cidade:        customer.Address.City,
UF:            customer.Address.State,
CEP:           customer.Address.ZipCode,
```

**Possíveis causas:**

1. **Address struct é nil**: Se `customer.Address` for `nil`, todos os campos ficarão vazios
2. **Campos não preenchidos no checkout**: Os dados de endereço podem não estar chegando do cliente
3. **Tipo de dado errado**: Se `Address` for um tipo diferente de struct

---

## 🔧 Debug Adicional Necessário

### Passo 1: Ver a struct `entity.Customer`
```bash
grep -n "type Customer struct" internal/entity/*.go
```

### Passo 2: Ver como o Address é preenchido no checkout
Procurar em [internal/infra/http/handlers/customer_handler.go](internal/infra/http/handlers/customer_handler.go#L35) o método `CreateCheckoutHandler`

### Passo 3: Verificar `CreateCustomerUseCase`
Ver onde `CreateCustomerInput` é convertido para `entity.Customer` e se o `Address` está sendo preenchido corretamente

---

## 📊 Sequência de Dados (Checkout → DocuSeal)

```
1. HTTP POST /checkout
   └─ JSON Body (CreateCustomerInput)
      {
        "address": "Rua X",
        "number": "123",
        "neighborhood": "Centro",
        ...
      }
      
2. CreateCheckoutHandler (customer_handler.go)
   └─ Decode CreateCustomerInput
   └─ Call CreateCustomerUseCase.Execute()
   
3. CreateCustomerUseCase.Execute()
   └─ Valida input
   └─ Converte para entity.Customer
   └─ Salva no DB
   └─ Ativa subscription
   
4. ActivateSubscriptionUseCase.Execute()
   └─ Carrega customer do DB
   └─ Cria DocuSealContractInput
      └─ customer.Address.Street ← AQUI PODE ESTAR VAZIO!
   └─ Chama DocuSealUseCase.ExecuteAutomatic()
   
5. DocuSealUseCase.ExecuteAutomatic()
   └─ Cria fieldValues com dados de endereço
   └─ Envia para DocuSeal API
```

---

## 📝 Logs Adicionados

Foram adicionados logs no início de `Execute()` em [generate_contract_docuseal.go](internal/usecase/generate_contract_docuseal.go#L98):

```
🔍 [DOCUSEAL DEBUG] Campos de Endereço Recebidos:
   Endereco: "Rua das Flores" (vazio=false)
   Numero: "123" (vazio=false)
   Bairro: "Centro" (vazio=false)
   Cidade: "São Paulo" (vazio=false)
   UF: "SP" (vazio=false)
   CEP: "01310-100" (vazio=false)
   Complemento: "" (vazio=true)
```

Se todos aparecerem vazios → problema está em `activate_subscription.go`
Se alguns vazios → problema está em como `customer.Address` é preenchido

---

## ✅ Próximos Passos

1. Rodar o servidor com os novos logs
2. Fazer checkout via Postman
3. Ver os logs de DEBUG que foram adicionados
4. Se vazios, então investigar:
   - [internal/entity/customer.go](internal/entity/customer.go) - verificar struct Customer
   - [internal/usecase/create_customer.go](internal/usecase/create_customer.go) - verificar conversão de dados
   - [internal/infra/database/customer_repository.go](internal/infra/database/customer_repository.go) - verificar salvamento
