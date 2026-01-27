package usecase

import (
	"context"
	"fmt"
	"log"

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
	log.Printf("🔄 Iniciando ativação para CustomerID: %s", input.CustomerID)

	// 1. Atualizar status da assinatura
	if err := uc.SubRepo.UpdateStatus(input.CustomerID, "ACTIVE"); err != nil {
		return fmt.Errorf("erro ao ativar status no banco: %w", err)
	}

	// 2. Buscar dados completos do Cliente (Nome, Email)
	customer, err := uc.CustomerRepo.FindByID(ctx, input.CustomerID)
	if err != nil {
		return fmt.Errorf("falha ao buscar dados do cliente: %w", err)
	}

	// 3. Buscar dados do Plano (Provider)
	plan, err := uc.PlanRepo.FindByID(ctx, customer.PlanID)
	if err != nil {
		return fmt.Errorf("falha ao buscar plano: %w", err)
	}

	// 4. Montar o Payload RICO (igual à sua struct ActivationPayload)
	payload := queue.ActivationPayload{
		CustomerID: customer.ID,
		PlanID:     plan.ID,
		Provider:   plan.Provider, // Ex: "DOC24"
		Name:       customer.Name,
		Email:      customer.Email,
		Origin:     "WEBHOOK_ASAAS",
	}

	// 5. Publicar na Fila
	if err := uc.Queue.PublishActivation(ctx, payload); err != nil {
		log.Printf("⚠️ CRITICAL: Ativado no banco, mas falha na fila: %v", err)
		return nil
	}

	log.Printf("🚀 Ativação enviada com sucesso para %s via %s", customer.Name, plan.Provider)
	return nil
}
