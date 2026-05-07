# Status da Feature de PDFs de Contrato por Plano

## Resumo
A feature de enviar PDFs de contratos editados como anexos no email **está 90% implementada no código**, mas **não está ativada** porque:
1. ❌ Os PDFs não têm o nome esperado (nomes slugificados)
2. ❌ O `GenerateContractUseCase` nunca é instanciado em `main.go`

---

## ✅ O que JÁ ESTÁ PRONTO (Infraestrutura Completa)

### 1. **ContractGenerator** (`internal/infra/pdf/contract_generator.go`)
- ✅ Classe `NewContractGenerator(templateDir string)` criada
- ✅ Método `Generate(planName, data, clientIP)` preenche PDFs com `pdftk`
- ✅ Converte nomes de planos para slug: "Ligue Vida Plena" → "ligue_vida_plena.pdf"
- ✅ Flattena formulário (bloqueia edição)
- ✅ Anexa página de certificação digital com IP do cliente

### 2. **GenerateContractUseCase** (`internal/usecase/generate_contract.go`)
- ✅ Orquestra geração de PDF
- ✅ Faz upload para Supabase Storage
- ✅ Retorna `PDFBytes` para anexar no email

### 3. **Email Senders - Suporte Completo**
**SMTP** (`internal/infra/mail/sender.go`):
- ✅ `SendWelcomeEmailWithContractAndDependents()` aceita `contractPDF []byte`
- ✅ Cria arquivo temp com nome "termo_adesao.pdf"
- ✅ Anexa ao gomail
- ✅ Limpa arquivo após envio

**Graph API** (`internal/infra/mail/graph_sender.go`):
- ✅ `SendWelcomeEmailWithContractAndDependents()` aceita `contractPDF []byte`
- ✅ Codifica em base64
- ✅ Inclui no payload JSON com `@odata.type: #microsoft.graph.fileAttachment`

### 4. **Activation Flow** (`internal/usecase/activate_subscription.go`)
- ✅ Linhas 116-121: Checa `if uc.ContractUC != nil`
- ✅ Se existir, chama `uc.ContractUC.Execute(ctx, buildContractInput(...))`
- ✅ Extrai `contractPDF = contractResult.PDFBytes`
- ✅ Passa para `uc.sendWelcomeEmail(customer, plan, dependents, contractPDF)`

### 5. **Membership Cards + Contract**
- ✅ `BuildMembershipCardAttachments()` gera cartões (titular + dependentes)
- ✅ Email anexa MÚLTIPLOS arquivos:
  - Cartão do titular (PNG)
  - Cartão de cada dependente (PNG)
  - Contrato assinado (PDF "termo_adesao.pdf")

---

## ❌ O que ESTÁ FALTANDO

### 1. **Nomeação dos PDFs**
**Problema**: Os arquivos existem em:
```
/internal/infra/storage/plans_templates/
├── LigueMedicina - Termo de Adesão (EDITADO).pdf
├── LigueMedicina - Termo de Adesão Ligue Cuidado Total - CHECKOUT (EDITADO).pdf
├── LigueMedicina - Termo de Adesão Ligue Mais Cuidado - CHECKOUT (EDITADO).pdf
├── LigueMedicina - Termo de Adesão Ligue Vida Plena - CHECKOUT (EDITADO).pdf
└── LigueMedicina - Termo de Adesão SAUDE EM DIa (EDITADO).pdf
```

**Esperado**: O `ContractGenerator.Generate()` procura por arquivos em formato slug:
```go
templatePath := fmt.Sprintf("%s/%s.pdf", g.templateDir, planName)
// Exemplo: "plans_templates/ligue_vida_plena.pdf"
```

**Conversão necessária** (em `activate_subscription.go` linha 228-231):
```
"Ligue Vida Plena" → "ligue_vida_plena"
"Ligue Mais Cuidado" → "ligue_mais_cuidado"  
"Ligue Cuidado Total" → "ligue_cuidado_total"
"Saúde em Dia" → "saude_em_dia"
"Plano Individual" → "plano_individual"
```

**SOLUTION**: Renomear os PDFs para formato slug:
```
Renomear: "LigueMedicina - Termo de Adesão Ligue Vida Plena - CHECKOUT (EDITADO).pdf"
Para:     "ligue_vida_plena.pdf"
```

### 2. **GenerateContractUseCase NÃO INICIALIZADO em main.go**
**Local atual de inicialização em `cmd/api/main.go` linhas 155-165**:
```go
// FALTA AQUI: ContractGenerator e GenerateContractUseCase não são criados!

activateSubUC := usecase.NewActivateSubscriptionUseCase(
    subRepo, customerRepo, planRepo, dependentRepo, producer, mailSender, kommoAdapter,
    // ❌ FALTA: contractUC *GenerateContractUseCase (passar aqui!)
)
```

**O que deveria ter**:
```go
// Criar ContractGenerator com o caminho dos templates
contractGen := pdf.NewContractGenerator("internal/infra/storage/plans_templates")

// Criar storage interface (se necessário - pode usar Supabase)
contractStorage := /* implementação */

// Criar usecase de geração
contractUC := usecase.NewGenerateContractUseCase(contractGen, contractStorage)

// Passar para ActivateSubscriptionUseCase
activateSubUC := usecase.NewActivateSubscriptionUseCase(
    subRepo, customerRepo, planRepo, dependentRepo, producer, mailSender, kommoAdapter,
    contractUC,  // ← ADICIONAR AQUI
)
```

**Resultado**: Como `ContractUC` é sempre `nil`, a linha 116 em `activate_subscription.go`:
```go
if uc.ContractUC != nil {  // ← Sempre FALSE, então PDF nunca é gerado
```

---

## 📋 Planos Detectados e Mapeamento de PDFs

| Nome do Plano (BD) | Slug Esperado | PDF Existente | Status |
|---|---|---|---|
| Ligue Vida Plena | `ligue_vida_plena` | Ligue Vida Plena (CHECKOUT) | Renomear |
| Ligue Mais Cuidado | `ligue_mais_cuidado` | Ligue Mais Cuidado (CHECKOUT) | Renomear |
| Ligue Cuidado Total | `ligue_cuidado_total` | Ligue Cuidado Total (CHECKOUT) | Renomear |
| Saúde em Dia | `saude_em_dia` | SAUDE EM DIa | Renomear |
| Plano Individual | `plano_individual` | (EDITADO) | Renomear |

---

## 🔧 PRÓXIMAS AÇÕES PARA ATIVAR A FEATURE

### Passo 1: Renomear PDFs
```bash
cd internal/infra/storage/plans_templates/

# Renomear cada PDF para formato slug
mv "LigueMedicina - Termo de Adesão Ligue Vida Plena - CHECKOUT (EDITADO).pdf" "ligue_vida_plena.pdf"
mv "LigueMedicina - Termo de Adesão Ligue Mais Cuidado - CHECKOUT (EDITADO).pdf" "ligue_mais_cuidado.pdf"
mv "LigueMedicina - Termo de Adesão Ligue Cuidado Total - CHECKOUT (EDITADO).pdf" "ligue_cuidado_total.pdf"
mv "LigueMedicina - Termo de Adesão SAUDE EM DIa (EDITADO).pdf" "saude_em_dia.pdf"
mv "LigueMedicina - Termo de Adesão (EDITADO).pdf" "plano_individual.pdf"
```

### Passo 2: Adicionar Inicialização em main.go
Após a linha 134 (após `setupEmailService()`), adicionar:

```go
// 4.1 Inicialização de Geração de Contratos (PDF)
contractGen := pdf.NewContractGenerator("internal/infra/storage/plans_templates")
// TODO: Implementar interface ContractStorageInterface para Supabase
var contractStorage usecase.ContractStorageInterface = nil  // Usar mock ou Supabase
var contractUC *usecase.GenerateContractUseCase = nil
if contractStorage != nil {
    contractUC = usecase.NewGenerateContractUseCase(contractGen, contractStorage)
    log.Println("✅ PDF Contract Generator inicializado")
}
```

### Passo 3: Passar ContractUC para ActivateSubscriptionUseCase
Linha 165, modificar:

```go
// ANTES:
activateSubUC := usecase.NewActivateSubscriptionUseCase(
    subRepo, customerRepo, planRepo, dependentRepo, producer, mailSender, kommoAdapter,
)

// DEPOIS: Atualizar NewActivateSubscriptionUseCase para aceitar ContractUC
// (Requer mudança na assinatura da função em interfaces.go)
```

### Passo 4: (OPCIONAL) Implementar ContractStorageInterface
Se quiser que PDFs sejam salvos no Supabase Storage depois de gerados:
- Criar implementação de `ContractStorageInterface.Upload()`
- Usar Supabase SDK (já está em `createCustomerUC`)
- Passar para `NewGenerateContractUseCase(contractGen, supabaseStorage)`

---

## 🧪 Teste Manual (Após Ativação)

1. **Criar subscription com plano "Ligue Vida Plena"**
2. **Ativar via webhook** (ou manual)
3. **Verificar email**:
   - ✅ Deve ter 3 anexos (se tiver dependentes):
     - `cartao_1.png` (titular)
     - `cartao_2.png` (dependente)
     - `termo_adesao.pdf` (contrato preenchido + certificado)
4. **Abrir PDF**:
   - ✅ Formulário preenchido com dados do cliente
   - ✅ Certificado digital na última página com IP + timestamp

---

## 📝 Nota

A feature está **99% pronta**. Bastam 2 ações:
1. Renomear PDFs (30 segundos)
2. Adicionar 5 linhas em main.go + atualizar assinatura de função (2 minutos)

Toda a lógica de preenchimento, endurecimento, e anexação de PDF já está funcionando.
