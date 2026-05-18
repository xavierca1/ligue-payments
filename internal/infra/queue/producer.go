package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/xavierca1/ligue-payments/internal/infra/http/middleware"
)

type DependentPayload struct {
	Name      string `json:"name"`
	CPF       string `json:"cpf"`
	BirthDate string `json:"birth_date"`
	Gender    int    `json:"gender"` // 1=Masculino, 2=Feminino, 3=Outro
	Kinship   string `json:"kinship"`
}

type ActivationPayload struct {
	CustomerID string `json:"customer_id"`
	PlanID     string `json:"plan_id"`

	ProviderPlanCode string `json:"provider_plan_code"`

	Provider string `json:"provider"`
	Origin   string `json:"origin"`

	Name      string `json:"name"`
	Email     string `json:"email"`
	CPF       string `json:"cpf"`
	Phone     string `json:"phone"`
	BirthDate string `json:"birth_date"`
	Gender    string `json:"gender"`

	Dependents []DependentPayload `json:"dependents,omitempty"`
}

type QueueProducerInterface interface {
	PublishActivation(ctx context.Context, payload ActivationPayload) error
}
type RabbitMQProducer struct {
	Conn *amqp.Connection
	Ch   *amqp.Channel
}

func NewProducer(conn *amqp.Connection, ch *amqp.Channel) *RabbitMQProducer {
	return &RabbitMQProducer{
		Conn: conn,
		Ch:   ch,
	}
}

func (p *RabbitMQProducer) PublishActivation(ctx context.Context, payload ActivationPayload) error {

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erro ao converter payload: %v", err)
	}

	err = p.Ch.PublishWithContext(ctx,
		ExchangeName, // ex.checkout
		RoutingKey,   // k.activation
		false,        // Mandatory
		false,        // Immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now().UTC(),
			DeliveryMode: amqp.Persistent, // Mensagem salva no disco (segurança!)
		},
	)

	if err != nil {
		return fmt.Errorf("falha ao publicar no RabbitMQ: %v", err)
	}

	middleware.RecordQueuePublished(QueueName, RoutingKey)

	return nil
}
