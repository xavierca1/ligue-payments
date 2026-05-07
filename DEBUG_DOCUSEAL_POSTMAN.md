# Instruções para Debug DocuSeal - Postman

## 1. Endpoint de Checkout Normal
**POST** `http://localhost:3000/checkout`

Use o payload: `postman_checkout_payload.json`

Isso vai:
- Criar o cliente
- Enviar para DocuSeal
- **ENVIAR EMAIL** (cuidado!)

---

## 2. Endpoint de Teste DocuSeal (SEM EMAIL)
**POST** `http://localhost:3000/docuseal/test`

Use o payload: `postman_docuseal_test_fields.json`

Isso vai:
- Criar submission do DocuSeal
- **NÃO ENVIA EMAIL** (sem `send_email: true`)
- Retorna `signing_url` e `uuid`

### Response esperado:
```json
{
  "signing_url": "https://app.docuseal.com/s/...",
  "uuid": "uuid-da-submission",
  "error": ""
}
```

---

## 3. Para Debug dos Fields

### Opção A: Rodar a função de debug direto
No seu código Go, chame:

```go
payload := usecase.DocuSealContractInput{
    CustomerID: "123",
    Nome: "João Silva",
    Email: "test@test.com",
    // ... outros campos
}

// Printa em JSON formatado
usecase.DebugDocuSealFields(payload)

// Printa payload completo
usecase.DebugDocuSealPayload(payload, 3346712)

// Mostra campo por campo
usecase.PrintFieldDifferences(payload)
```

### Opção B: Ver no log do checkout
Faça POST no `/checkout` e veja os logs com `[checkout]`

---

## 4. Problema Atual

Se os fields não estão passando completo para o DocuSeal, pode ser:

1. **Campo vazio no input** - Use `PrintFieldDifferences()` para ver quais chegaram vazios
2. **Nome do field errado** - DocuSeal espera nomes específicos (verificar template)
3. **Formato errado** - Ex: `monthly` vs `Mensal`

---

## 5. Como Desativar Email Temporariamente

Se quiser testar sem email receber, modifique em `generate_contract_docuseal.go`:

```go
submissionReq := &docuseal.CreateSubmissionRequest{
    TemplateID: templateID,
    SendEmail:  false,  // <-- Mude para false
    // ...
}
```

---

## 6. Templates Disponíveis

- `ligue_saude_em_dia` (ID: 3346712)
- `ligue_mais_cuidado` (ID: ?)
- `ligue_vida_plena` (ID: ?)
- `ligue_cuidado_total` (ID: ?)
- `ligue_viver_bem` (ID: ?)

Verificar em `internal/infra/integration/docuseal/templates.go`
