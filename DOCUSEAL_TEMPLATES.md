# DocuSeal Templates Configuration

## Todos os templates disponíveis

```json
{
  "ligue_saude_em_dia": 3346712,
  "ligue_mais_cuidado": 3346739,
  "ligue_vida_plena": 3624336,
  "ligue_cuidado_total": 3346755,
  "ligue_viver_bem": 3346717
}
```

## Campos suportados

Todos os templates suportam os mesmos campos (não há necessidade de customizar por template):

1. **product** - Nome do produto/plano
2. **id** - ID do cliente
3. **method_payment** - Método de pagamento (PIX, CREDIT_CARD)
4. **periodicidade** - Periodicidade (monthly → Mensal)
5. **name** - Nome completo
6. **birthdate** - Data de nascimento ✅ Preenchido dinamicamente
7. **cpf** - CPF
8. **genre** - Gênero
9. **marital_status** - Estado civil
10. **cellphone** - Celular
11. **email** - Email
12. **address** - Endereço ✅ Preenchido dinamicamente
13. **number** - Número ✅ Preenchido dinamicamente
14. **complement** - Complemento ✅ Preenchido dinamicamente
15. **neighborhood** - Bairro
16. **city** - Cidade ✅ Preenchido dinamicamente
17. **UF** - Estado (UF)
18. **zip_code** - CEP ✅ Preenchido dinamicamente

**Legenda**: ✅ Campos alterados dinamicamente de acordo com o plano selecionado

## Normalização

- **Periodicidade**: Qualquer valor como "monthly", "mensal" (case-insensitive) é automaticamente convertido para "Mensal"
- **Auto-assinatura**: Todos os documentos são criados já assinados (completed=true)
- **Seleção automática de template**: O sistema mapeia automaticamente o nome do plano para o template correspondente

## Mapeamento automático de planos para templates

Quando um cliente ativa uma assinatura, o sistema automaticamente detecta qual template usar baseado no nome do plano:

| Nome do Plano | Template DocuSeal |
|---|---|
| Saúde em Dia, Ligue Saúde em Dia | `ligue_saude_em_dia` (3346712) |
| Mais Cuidado, Ligue Mais Cuidado | `ligue_mais_cuidado` (3346739) |
| Vida Plena, Ligue Vida Plena | `ligue_vida_plena` (3624336) |
| Cuidado Total, Ligue Cuidado Total | `ligue_cuidado_total` (3346755) |
| Viver Bem, Ligue Viver Bem | `ligue_viver_bem` (3346717) |

O mapeamento é **case-insensitive** e remove acentos automaticamente. Se o nome do plano não corresponder a nenhum padrão, usa o template padrão `ligue_saude_em_dia`.

## Uso

### Via API (Test Handler)

```bash
POST /docuseal/test
Content-Type: application/json

{
  "email": "cliente@example.com",
  "template": "ligue_vida_plena",  # Nome do template (obrigatório)
  "fields": {
    "product": "Vida Plena Plus",
    "id": "123456",
    "method_payment": "credit_card",
    "periodicidade": "monthly",
    "name": "João Silva",
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
  }
}
```

### Via Usecase (Em código)

```go
usecase := &GenerateContractWithDocuSealUseCase{
  PdfGenerator: pdfGen,
  DocuSealClient: client,
}

input := DocuSealContractInput{
  TemplateName: "ligue_vida_plena",  // Seleciona template
  CustomerID: "123456",
  Email: "cliente@example.com",
  Nome: "João Silva",
  CPF: "123.456.789-10",
  // ... outros campos
}

output, err := usecase.Execute(context.Background(), input)
```

## Scripts de teste

```bash
# Testar todos os templates
./test_docuseal_templates.sh
```

## Implementação

- **Arquivo de configuração**: `internal/infra/integration/docuseal/templates.go`
- **Handlers**: 
  - `internal/infra/http/handlers/docuseal_test_handler.go` - Endpoint `/docuseal/test`
  - `internal/infra/http/handlers/docuseal_status_handler.go` - Endpoint `/docuseal/status`
- **Usecase**: `internal/usecase/generate_contract_docuseal.go`
  - Método `Execute()` - Com email customizável
  - Método `ExecuteAutomatic()` - Auto-assinado sem URL
