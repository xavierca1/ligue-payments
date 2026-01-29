package queue

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Define o formato da mensagem que vai viajar na fila
type ActivationPayload struct {
	CustomerID string `json:"customer_id"`
	PlanID     string `json:"plan_id"`

	ProviderPlanCode string `json:"provider_plan_code"`
	// Campos de Controle
	Provider string `json:"provider"` // <--- Adicione
	Origin   string `json:"origin"`   // <--- Adicione

	// Dados do Cliente (Doc24)
	Name      string `json:"name"`
	Email     string `json:"email"`
	CPF       string `json:"cpf"`
	Phone     string `json:"phone"`
	BirthDate string `json:"birth_date"` // <--- Importante para Doc24
	Gender    string `json:"gender"`     // <--- Importante para Doc24
}

type QueueProducerInterface interface {
	PublishActivation(ctx context.Context, payload ActivationPayload) error
}
type RabbitMQProducer struct {
	Conn *amqp.Connection
	Ch   *amqp.Channel
}

// NewProducer reaproveita a conexão que já abrimos no main.go
func NewProducer(conn *amqp.Connection, ch *amqp.Channel) *RabbitMQProducer {
	return &RabbitMQProducer{
		Conn: conn,
		Ch:   ch,
	}
}

func (p *RabbitMQProducer) PublishActivation(ctx context.Context, payload ActivationPayload) error {
	// 1. Transforma Struct em JSON (Bytes)
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erro ao converter payload: %v", err)
	}

	// 2. Publica na Exchange "ex.checkout" com a chave "k.activation"
	err = p.Ch.PublishWithContext(ctx,
		ExchangeName, // ex.checkout
		RoutingKey,   // k.activation
		false,        // Mandatory
		false,        // Immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // Mensagem salva no disco (segurança!)
		},
	)

	if err != nil {
		return fmt.Errorf("falha ao publicar no RabbitMQ: %v", err)
	}

	return nil
}
