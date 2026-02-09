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
	customerRepo entity.CustomerRepositoryInterface,
	planRepo entity.PlanRepositoryInterface,
	queue queue.QueueProducerInterface,
	emailService EmailService,
	kommoService KommoService,
) *ActivateSubscriptionUseCase {
	return &ActivateSubscriptionUseCase{
		SubRepo:      subRepo,
		CustomerRepo: customerRepo,
		PlanRepo:     planRepo,
		Queue:        queue,
		EmailService: emailService,
		KommoService: kommoService,
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

	// Enviar email de boas-vindas
	// TODO: Configurar credenciais SMTP no .env (MAIL_HOST, MAIL_PORT, MAIL_USER, MAIL_PASS)
	go func() {
		if uc.EmailService != nil {
			err := uc.EmailService.SendWelcomeEmail(customer.Name, customer.Email)
			if err != nil {
				log.Printf("⚠️ Falha ao enviar email de boas-vindas (não bloqueia): %v", err)
			} else {
				log.Printf("✅ Email de boas-vindas enviado para %s", customer.Email)
			}
		}
	}()

	log.Printf(" Ativação enviada com sucesso para %s via %s", customer.Name, plan.Provider)
	return nil
}
