package usecase

import (
	"context"
)

// ActivateSubscriptionInput define o que precisamos para iniciar a ativação
type ActivateSubscriptionInput struct {
	CustomerID string
	// Futuramente pode ter: TransactionID string
}

// ActivateSubscriptionUseCase orquestra a liberação do acesso
type ActivateSubscriptionUseCase struct {
	SubRepo      SubscriptionRepository // Interface já definida no create_customer.go
	Queue        QueueProducerInterface // Interface do RabbitMQ
	EmailService EmailService           // Interface de Email
}

// NewActivateSubscriptionUseCase cria a instância
func NewActivateSubscriptionUseCase(
	subRepo SubscriptionRepository,
	queue QueueProducerInterface,
	emailService EmailService,
) *ActivateSubscriptionUseCase {
	return &ActivateSubscriptionUseCase{
		SubRepo:      subRepo,
		Queue:        queue,
		EmailService: emailService,
	}
}

// Execute contém a lógica de ativação chamada pelo Webhook
func (uc *ActivateSubscriptionUseCase) Execute(ctx context.Context, input ActivateSubscriptionInput) error {
	// TODO [SEGUNDA-FEIRA]: Implementar lógica de ativação
	// O fluxo será:

	// 1. Atualizar status no banco de dados para "ACTIVE"
	// err := uc.SubRepo.UpdateStatus(ctx, input.CustomerID, "ACTIVE")

	// 2. Montar payload e publicar na fila (RabbitMQ) para liberar sistemas externos (Tem/Doc24)
	// payload := ActivationPayload{...}
	// err := uc.Queue.PublishActivation(ctx, payload)

	// 3. (Opcional) Disparar e-mail de boas-vindas
	// go uc.EmailService.SendWelcome(...)

	return nil
}
