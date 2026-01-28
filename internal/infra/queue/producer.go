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
	Provider   string `json:"provider"`
	Name       string `json:"name"`
	Email      string `json:"email"`

	// 👇 Campos novos vitais para a Doc24/Integrações
	Phone string `json:"phone"`
	CPF   string `json:"cpf"`

	Origin string `json:"origin"`
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
