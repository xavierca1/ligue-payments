package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"unicode"

	"github.com/xavierca1/ligue-payments/internal/entity"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/docuseal"
	"github.com/xavierca1/ligue-payments/internal/infra/queue"
	"golang.org/x/text/unicode/norm"
)

func NewActivateSubscriptionUseCase(
	subRepo entity.SubscriptionRepository,
	customerRepo entity.CustomerRepositoryInterface,
	planRepo entity.PlanRepositoryInterface,
	dependentRepo entity.DependentRepositoryInterface,
	queue queue.QueueProducerInterface,
	emailService EmailService,
	kommoService KommoService,
) *ActivateSubscriptionUseCase {
	return &ActivateSubscriptionUseCase{
		SubRepo:       subRepo,
		CustomerRepo:  customerRepo,
		PlanRepo:      planRepo,
		DependentRepo: dependentRepo,
		Queue:         queue,
		EmailService:  emailService,
		KommoService:  kommoService,
	}
}

func (uc *ActivateSubscriptionUseCase) Execute(ctx context.Context, input ActivateSubscriptionInput) error {
	log.Printf(" Iniciando ativação para CustomerID: %s", input.CustomerID)

	customer, err := uc.CustomerRepo.FindByID(ctx, input.CustomerID)
	if err != nil {
		return fmt.Errorf("falha ao buscar dados do cliente: %w", err)
	}

	// DEBUG: Exibir objeto customer completo em JSON
	customerJSON, _ := json.MarshalIndent(customer, "", "  ")
	log.Printf("\n🔍 [CUSTOMER DEBUG - OBJETO COMPLETO]:\n%s\n", string(customerJSON))

	var dependents []*entity.Dependent
	if uc.DependentRepo != nil {
		loadedDependents, depErr := uc.DependentRepo.FindByCustomerID(ctx, input.CustomerID)
		if depErr != nil {
			log.Printf("⚠️ Falha ao buscar dependentes do cliente (não bloqueia): %v", depErr)
		} else {
			dependents = loadedDependents
		}
	}

	// Converter Gender int para string legível
	var genderStr string
	switch customer.Gender {
	case 1:
		genderStr = "Masculino"
	case 2:
		genderStr = "Feminino"
	case 3:
		genderStr = "Outro"
	}

	sub, err := uc.SubRepo.FindLastByCustomerID(ctx, input.CustomerID)
	if err != nil {
		return fmt.Errorf("falha ao buscar assinatura do cliente: %w", err)
	}

	if sub.PlanID == "" {
		return fmt.Errorf("inconsistência: assinatura %s não tem PlanID vinculado", sub.ID)
	}

	alreadyActive := sub.Status == "ACTIVE"
	if alreadyActive {
		log.Printf("ℹ️ Assinatura já ativa para CustomerID=%s. Mantendo fluxo para e-mail de boas-vindas.", input.CustomerID)
	}

	if !alreadyActive {
		// Atualizar subscription status para ACTIVE
		if err := uc.SubRepo.UpdateStatus(input.CustomerID, "ACTIVE"); err != nil {
			return fmt.Errorf("erro ao ativar status da subscription no banco: %w", err)
		}
	}

	// Sempre atualizar o status do customer para ACTIVE quando a subscription está ativa
	// (mesmo em retries/webhooks duplicados, garante sincronização)
	if err := uc.CustomerRepo.UpdateStatus(ctx, input.CustomerID, "ACTIVE"); err != nil {
		log.Printf("⚠️ Falha ao atualizar status do customer para ACTIVE (não bloqueia): %v", err)
		// Não bloqueamos - o subscription já foi ativado
	} else {
		log.Printf("✅ Customer status atualizado para ACTIVE (customer_id=%s)", input.CustomerID)
	}

	plan, err := uc.PlanRepo.FindByID(ctx, sub.PlanID)
	if err != nil {
		return fmt.Errorf("falha ao buscar plano (%s): %w", sub.PlanID, err)
	}

	providerPlanCode := strings.TrimSpace(plan.ProviderPlanCode)
	if providerPlanCode == "" {
		providerPlanCode = strings.TrimSpace(plan.Name)
	}

	payload := queue.ActivationPayload{
		CustomerID:       customer.ID,
		PlanID:           plan.ID,
		ProviderPlanCode: providerPlanCode,
		Provider:         plan.Provider,
		Name:             customer.Name,
		Email:            customer.Email,
		Origin:           "WEBHOOK_ASAAS",
		Phone:            customer.Phone,
		CPF:              customer.CPF,
		BirthDate:        customer.BirthDate,
		Gender:           genderStr,
	}

	if uc.Queue != nil {
		if err := uc.Queue.PublishActivation(ctx, payload); err != nil {
			log.Printf("⚠️ Assinatura ativada no banco, mas falha ao publicar na fila: %v", err)
		}
	}

	// Prioridade 1: Usar DocuSeal automático (se disponível)
	// Isso gera o documento, registra aceitação dos termos com data/hora
	// e envia o PDF por email com o corpo específico do contrato
	if uc.DocuSealUseCase != nil {
		// Primeiro, mantém o template atual de boas-vindas
		uc.sendWelcomeEmail(customer, plan, dependents, nil)

		// Mapeia o nome do plano para o template correspondente no DocuSeal
		templateName := docuseal.GetTemplateFromPlanName(plan.Name)

		docuSealInput := DocuSealContractInput{
			TemplateName:  templateName,
			CustomerID:    customer.ID,
			Nome:          customer.Name,
			Email:         customer.Email,
			CPF:           customer.CPF,
			PlanName:      plan.Name,
			Produto:       plan.Name,
			Valor:         fmt.Sprintf("%.2f", float64(sub.Amount)/100), // De centavos para reais
			Pagamento:     sub.PaymentMethod,                            // PIX ou CREDIT_CARD
			Periodicidade: "Mensal",                                     // Sempre mensal por padrão
			Nascimento:    customer.BirthDate,
			Sexo:          genderStr, // Convertido de int
			Civil:         "",        // Pode extrair do customer se tiver
			Celular:       customer.Phone,
			Endereco:      customer.Address.Street,
			Numero:        customer.Address.Number,
			Bairro:        customer.Address.District,
			Cidade:        customer.Address.City,
			UF:            customer.Address.State,
			CEP:           customer.Address.ZipCode,
		}

		submissionUUID, err := uc.DocuSealUseCase.ExecuteAutomatic(ctx, docuSealInput)
		if err != nil {
			log.Printf("⚠️ Falha ao gerar documento DocuSeal (não bloqueia ativação): %v", err)
		} else {
			log.Printf("✅ Documento DocuSeal gerado automaticamente (UUID=%s) para %s", submissionUUID, customer.Email)
			// TODO: Armazenar submission_uuid na tabela de subscriptions para rastreamento
			// TODO: Usar webhook DocuSeal para enviar PDF quando assinado
		}
	} else {
		// Fallback: Gerar contrato PDF tradicional (apenas se DocuSeal não disponível)
		var contractPDF []byte
		if uc.ContractUC != nil {
			contractResult, contractErr := uc.ContractUC.Execute(ctx, buildContractInput(customer, plan))
			if contractErr != nil {
				log.Printf("⚠️ Falha ao gerar contrato (não bloqueia ativação): %v", contractErr)
			} else {
				contractPDF = contractResult.PDFBytes
			}
		}

		uc.sendWelcomeEmail(customer, plan, dependents, contractPDF)
	}

	log.Printf(" Ativação enviada com sucesso para %s via %s", customer.Name, plan.Provider)
	return nil
}

func (uc *ActivateSubscriptionUseCase) sendWelcomeEmail(customer *entity.Customer, plan *entity.Plan, dependents []*entity.Dependent, contractPDF []byte) {
	if uc.EmailService == nil {
		return
	}
	if len(contractPDF) > 0 {
		if err := uc.EmailService.SendWelcomeEmailWithContractAndDependents(customer.Name, customer.Email, customer.CPF, plan.Name, customer.ProviderID, dependents, contractPDF); err != nil {
			log.Printf("⚠️ Falha ao enviar email com contrato (não bloqueia): %v", err)
		} else {
			log.Printf("✅ Email de boas-vindas com contrato enviado para %s", customer.Email)
		}
		return
	}

	if len(dependents) > 0 {
		if err := uc.EmailService.SendWelcomeEmailWithCardAndDependents(customer.Name, customer.Email, customer.CPF, plan.Name, customer.ProviderID, dependents); err != nil {
			log.Printf("⚠️ Falha ao enviar email de boas-vindas com carteirinha e dependentes (não bloqueia): %v", err)
		} else {
			log.Printf("✅ Email de boas-vindas com carteirinha e dependentes enviado para %s", customer.Email)
		}
		return
	}

	if err := uc.EmailService.SendWelcomeEmailWithCard(customer.Name, customer.Email, customer.CPF, plan.Name, customer.ProviderID); err != nil {
		log.Printf("⚠️ Falha ao enviar email de boas-vindas com carteirinha (não bloqueia): %v", err)
	} else {
		log.Printf("✅ Email de boas-vindas com carteirinha enviado para %s", customer.Email)
	}
}

// buildContractInput maps the customer and plan data to GenerateContractInput.
// Fields not stored in the customer entity (RG, orgao, civil, fixo) are left empty.
func buildContractInput(customer *entity.Customer, plan *entity.Plan) GenerateContractInput {
	sexo := "Masculino"
	switch customer.Gender {
	case 2:
		sexo = "Feminino"
	case 3:
		sexo = "Outro"
	}

	valor := ""
	if plan.PriceCents > 0 {
		valor = fmt.Sprintf("R$ %.2f", float64(plan.PriceCents)/100.0)
	}

	return GenerateContractInput{
		CustomerID:    customer.ID,
		PlanName:      planTemplateSlug(plan.Name),
		ClientIP:      "",
		Produto:       plan.Name,
		Valor:         valor,
		Pagamento:     "",
		Periodicidade: "",
		Nome:          customer.Name,
		Nascimento:    customer.BirthDate,
		CPF:           customer.CPF,
		RG:            "",
		Orgao:         "",
		Sexo:          sexo,
		Civil:         "",
		Celular:       customer.Phone,
		Fixo:          "",
		Email:         customer.Email,
		Endereco:      customer.Address.Street,
		Numero:        customer.Address.Number,
		Complemento:   customer.Address.Complement,
		Bairro:        customer.Address.District,
		Cidade:        customer.Address.City,
		UF:            customer.Address.State,
		CEP:           customer.Address.ZipCode,
	}
}

// planTemplateSlug converts a plan display name to a safe filename slug.
// Example: "Plano Individual" → "plano_individual"
func planTemplateSlug(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	decomposed := norm.NFD.String(name)

	var builder strings.Builder
	previousUnderscore := false

	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}

		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			previousUnderscore = false
			continue
		}

		if !previousUnderscore {
			builder.WriteByte('_')
			previousUnderscore = true
		}
	}

	return strings.Trim(builder.String(), "_")
}
