package usecase

// ============================================================================
// EXEMPLO DE INTEGRAÇÃO DO DOCUSEAL NO activate_subscription.go
// ============================================================================
//
// Este arquivo contém código de exemplo para integrar o DocuSeal
// no fluxo de ativação de assinatura (ActivateSubscriptionUseCase).
//
// IMPORTANTE: Este é apenas um EXEMPLO. Você deve adaptar conforme sua arquitetura.
//
// Passos para integração:
//
// 1. Adicionar campo DocuSealUC no ActivateSubscriptionUseCase:
//    type ActivateSubscriptionUseCase struct {
//        // ... campos existentes ...
//        DocuSealUC *GenerateContractWithDocuSealUseCase
//    }
//
// 2. Atualizar NewActivateSubscriptionUseCase para aceitar DocuSealUC:
//    func NewActivateSubscriptionUseCase(
//        subRepo entity.SubscriptionRepository,
//        customerRepo entity.CustomerRepositoryInterface,
//        planRepo entity.PlanRepositoryInterface,
//        dependentRepo entity.DependentRepositoryInterface,
//        queue queue.QueueProducerInterface,
//        emailService EmailService,
//        kommoService KommoService,
//        docuSealUC *GenerateContractWithDocuSealUseCase, // ADICIONAR
//    ) *ActivateSubscriptionUseCase {
//        return &ActivateSubscriptionUseCase{
//            SubRepo:       subRepo,
//            CustomerRepo:  customerRepo,
//            PlanRepo:      planRepo,
//            DependentRepo: dependentRepo,
//            Queue:         queue,
//            EmailService:  emailService,
//            KommoService:  kommoService,
//            DocuSealUC:    docuSealUC, // ADICIONAR
//        }
//    }
//
// 3. No método Execute() do ActivateSubscriptionUseCase, após ativar a subscription:
//
//    Substitua a geração do PDF tradicional por DocuSeal:
//
//    --------- ANTES (Comentar) ---------
//    // Gerar contrato PDF e enviar por email
//    var contractPDF []byte
//    if uc.ContractUC != nil {
//        contractResult, contractErr := uc.ContractUC.Execute(ctx, buildContractInput(customer, plan))
//        if contractErr != nil {
//            log.Printf("⚠️ Falha ao gerar contrato (não bloqueia ativação): %v", contractErr)
//        } else {
//            contractPDF = contractResult.PDFBytes
//        }
//    }
//    uc.sendWelcomeEmail(customer, plan, dependents, contractPDF)
//
//    --------- DEPOIS (Descomente) ---------
//    // Gerar contrato com assinatura digital DocuSeal
//    if uc.DocuSealUC != nil {
//        docuSealInput := DocuSealContractInput{
//            CustomerID:    customer.ID,
//            Nome:          customer.Name,
//            Email:         customer.Email,
//            CPF:           customer.CPF,
//            PlanName:      plan.Name,
//            Produto:       plan.Name,
//            Valor:         formatPrice(plan.Price), // função auxiliar
//            Pagamento:     "Débito em Conta", // ou obter do subscription
//            Periodicidade: "Mensal", // ou obter do plan
//            Nascimento:    customer.BirthDate,
//            RG:            "", // TODO: adicionar ao customer se não tiver
//            Orgao:         "", // TODO: adicionar ao customer se não tiver
//            Sexo:          formatGender(customer.Gender),
//            Civil:         "", // TODO: adicionar ao customer se não tiver
//            Celular:       customer.Phone,
//            Fixo:          "", // TODO: adicionar ao customer se não tiver
//            Email:         customer.Email,
//            Endereco:      customer.Street, // TODO: verificar nomes dos campos
//            Numero:        customer.Number,
//            Complemento:   customer.Complement,
//            Bairro:        customer.District,
//            Cidade:        customer.City,
//            UF:            customer.State,
//            CEP:           customer.ZipCode,
//            ClientIP:      input.ClientIP, // obter do request HTTP
//            FieldUUIDs: map[string]string{
//                "cpf":          "COLOCAR_UUID", // Obter do DocuSeal
//                "rg":           "COLOCAR_UUID",
//                "sexo":         "COLOCAR_UUID",
//                "civil":        "COLOCAR_UUID",
//                "nascimento":   "COLOCAR_UUID",
//                "celular":      "COLOCAR_UUID",
//                "fixo":         "COLOCAR_UUID",
//                "email":        "COLOCAR_UUID",
//                "endereco":     "COLOCAR_UUID",
//                "numero":       "COLOCAR_UUID",
//                "complemento":  "COLOCAR_UUID",
//                "bairro":       "COLOCAR_UUID",
//                "cidade":       "COLOCAR_UUID",
//                "uf":           "COLOCAR_UUID",
//                "cep":          "COLOCAR_UUID",
//                "produto":      "COLOCAR_UUID",
//                "valor":        "COLOCAR_UUID",
//                "pagamento":    "COLOCAR_UUID",
//                "periodicidade": "COLOCAR_UUID",
//            },
//        }
//
//        docuSealOutput, err := uc.DocuSealUC.Execute(ctx, docuSealInput)
//        if err != nil {
//            log.Printf("⚠️ Erro ao gerar contrato DocuSeal (não bloqueia): %v", err)
//            // Fall back para email de boas-vindas sem contrato
//            uc.sendWelcomeEmail(customer, plan, dependents, nil)
//        } else {
//            log.Printf("✅ Contrato DocuSeal gerado - Signing URL: %s", docuSealOutput.SigningURL)
//            // TODO: Guardar docuSealOutput.SubmissionUUID no database
//            // TODO: Enviar email com link de assinatura
//            uc.sendDocuSealSigningEmail(customer, docuSealOutput.SigningURL)
//        }
//    } else {
//        // Fall back para contrato PDF tradicional se DocuSeal não está configurado
//        var contractPDF []byte
//        if uc.ContractUC != nil {
//            contractResult, contractErr := uc.ContractUC.Execute(ctx, buildContractInput(customer, plan))
//            if contractErr != nil {
//                log.Printf("⚠️ Falha ao gerar contrato (não bloqueia ativação): %v", contractErr)
//            } else {
//                contractPDF = contractResult.PDFBytes
//            }
//        }
//        uc.sendWelcomeEmail(customer, plan, dependents, contractPDF)
//    }
//
// 4. Criar novo método sendDocuSealSigningEmail() no ActivateSubscriptionUseCase:
//
//    func (uc *ActivateSubscriptionUseCase) sendDocuSealSigningEmail(customer *entity.Customer, signingURL string) {
//        if uc.EmailService == nil {
//            return
//        }
//        // TODO: Implementar método SendDocuSealSigningEmail na interface EmailService
//        // Este método deve enviar um email com:
//        // - Saudação personalisada
//        // - Link de assinatura prominente
//        // - Instruções de como assinar
//        // - Informações de suporte
//    }
//
// 5. Adicionar métodos helper:
//
//    func formatPrice(price int) string {
//        // Converter de centavos para formato legível
//        // Ex: 4990 -> "R$ 49,90"
//        return fmt.Sprintf("R$ %.2f", float64(price)/100)
//    }
//
//    func formatGender(gender int) string {
//        switch gender {
//        case 1:
//            return "Masculino"
//        case 2:
//            return "Feminino"
//        case 3:
//            return "Outro"
//        default:
//            return ""
//        }
//    }
//
// 6. Criar handler para webhook do DocuSeal (quando cliente assina):
//
//    func HandleDocuSealWebhook(w http.ResponseWriter, r *http.Request) {
//        var event map[string]interface{}
//        if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
//            http.Error(w, "Invalid payload", http.StatusBadRequest)
//            return
//        }
//
//        submissionUUID := event["submission_uuid"].(string)
//        status := event["status"].(string)
//
//        if status == "SIGNED" || status == "COMPLETED" {
//            // 1. Obter submission details do DocuSeal
//            // 2. Baixar PDF assinado
//            // 3. Guardar no database
//            // 4. Enviar por email para cliente
//            // 5. Atualizar status da subscription
//        }
//
//        w.WriteHeader(http.StatusOK)
//        json.NewEncoder(w).Encode(map[string]string{"status": "received"})
//    }
//
// ============================================================================

// ESTRUTURA AUXILIAR PARA RASTREAR ASSINATURAS (sugerida)

// DocuSealSubmissionRepository interface para persistir submissions
// type DocuSealSubmissionRepository interface {
//     Save(ctx context.Context, submission *DocuSealSubmission) error
//     FindByCustomerID(ctx context.Context, customerID string) (*DocuSealSubmission, error)
//     UpdateStatus(ctx context.Context, submissionUUID, status string) error
// }

// DocuSealSubmission entity para rastrear submissões
// type DocuSealSubmission struct {
//     ID               int64  // Primary Key
//     CustomerID       string // Foreign Key
//     SubscriptionID   string
//     SubmissionUUID   string // UUID único do DocuSeal
//     TemplateUUID     string
//     SigningURL       string
//     Status           string // SENT, VIEWED, SIGNED, COMPLETED
//     SignedAt         *time.Time
//     DocumentURL      string // URL do PDF assinado
//     CreatedAt        time.Time
//     UpdatedAt        time.Time
// }
