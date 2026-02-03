package usecase

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/xavierca1/ligue-payments/internal/entity"
	"github.com/xavierca1/ligue-payments/internal/infra/queue"
)




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



	customer, err := uc.CustomerRepo.FindByID(ctx, input.CustomerID)
	if err != nil {
		return fmt.Errorf("falha ao buscar dados do cliente: %w", err)
	}

	genderStr := strconv.Itoa(customer.Gender)



	sub, err := uc.SubRepo.FindLastByCustomerID(ctx, input.CustomerID)
	if err != nil {
		return fmt.Errorf("falha ao buscar assinatura do cliente: %w", err)
	}


	if sub.PlanID == "" {
		return fmt.Errorf("inconsistência: assinatura %s não tem PlanID vinculado", sub.ID)
	}



	if err := uc.SubRepo.UpdateStatus(input.CustomerID, "ACTIVE"); err != nil {
		return fmt.Errorf("erro ao ativar status no banco: %w", err)
	}


	plan, err := uc.PlanRepo.FindByID(ctx, sub.PlanID)
	if err != nil {
		return fmt.Errorf("falha ao buscar plano (%s): %w", sub.PlanID, err)
	}


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


	if err := uc.Queue.PublishActivation(ctx, payload); err != nil {

		log.Printf(" CRITICAL: Assinatura ativada no banco, mas falha ao publicar na fila: %v", err)
		return nil
	}

	log.Printf(" Ativação enviada com sucesso para %s via %s", customer.Name, plan.Provider)
	return nil
}
