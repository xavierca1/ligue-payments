package usecase

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/xavierca1/ligue-payments/internal/entity"
	"github.com/xavierca1/ligue-payments/internal/infra/queue"
)

// Input apenas com IDs (o que vem do Webhook)

// Construtor ATUALIZADO recebendo os novos repos
func NewActivateSubscriptionUseCase(
	subRepo entity.SubscriptionRepository,
	customerRepo entity.CustomerRepositoryInterface, // 👈 Novo argumento
	planRepo entity.PlanRepositoryInterface, // 👈 Novo argumento
	queue queue.QueueProducerInterface,
	emailService EmailService,
) *ActivateSubscriptionUseCase {
	return &ActivateSubscriptionUseCase{
		SubRepo:      subRepo,
		CustomerRepo: customerRepo,
		PlanRepo:     planRepo,
		Queue:        queue,
		EmailService: emailService,
	}
}

func (uc *ActivateSubscriptionUseCase) Execute(ctx context.Context, input ActivateSubscriptionInput) error {
	log.Printf(" Iniciando ativação para CustomerID: %s", input.CustomerID)

	// 1. Buscar dados do Cliente (Precisamos do Nome e Email para o Payload)

	customer, err := uc.CustomerRepo.FindByID(ctx, input.CustomerID)
	if err != nil {
		return fmt.Errorf("falha ao buscar dados do cliente: %w", err)
	}

	genderStr := strconv.Itoa(customer.Gender)

	// 2. Buscar a Assinatura (AQUI ESTÁ A CORREÇÃO 🛡️)
	// Precisamos da assinatura para saber QUAL É O PLANO REAL
	sub, err := uc.SubRepo.FindLastByCustomerID(ctx, input.CustomerID)
	if err != nil {
		return fmt.Errorf("falha ao buscar assinatura do cliente: %w", err)
	}

	// Blindagem: Verifica se o PlanID existe na assinatura
	if sub.PlanID == "" {
		return fmt.Errorf("inconsistência: assinatura %s não tem PlanID vinculado", sub.ID)
	}

	// 3. Atualizar status da assinatura para ACTIVE
	// (Podemos passar o ID da assinatura direto se seu repo suportar, ou manter customerID)
	if err := uc.SubRepo.UpdateStatus(input.CustomerID, "ACTIVE"); err != nil {
		return fmt.Errorf("erro ao ativar status no banco: %w", err)
	}

	// 4. Buscar dados do Plano (Usando o ID que veio da ASSINATURA)
	plan, err := uc.PlanRepo.FindByID(ctx, sub.PlanID)
	if err != nil {
		return fmt.Errorf("falha ao buscar plano (%s): %w", sub.PlanID, err)
	}

	// 5. Montar o Payload
	payload := queue.ActivationPayload{
		CustomerID: customer.ID,
		PlanID:     plan.ID,
		Provider:   plan.Provider, // Ex: "DOC24"
		Name:       customer.Name,
		Email:      customer.Email,
		Origin:     "WEBHOOK_ASAAS",
		Phone:      customer.Phone, // Adicionei Phone se tiver no payload
		CPF:        customer.CPF,
		BirthDate:  customer.BirthDate, // Certifique-se que o customer tem esse campo
		Gender:     genderStr,          // Certifique-se que o customer tem esse campo
	}

	// 6. Publicar na Fila
	if err := uc.Queue.PublishActivation(ctx, payload); err != nil {
		// Loga erro crítico mas não falha o request HTTP do Asaas (já salvamos no banco)
		log.Printf(" CRITICAL: Assinatura ativada no banco, mas falha ao publicar na fila: %v", err)
		return nil
	}

	log.Printf(" Ativação enviada com sucesso para %s via %s", customer.Name, plan.Provider)
	return nil
}
