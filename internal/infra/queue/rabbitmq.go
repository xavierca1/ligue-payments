package queue

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Configuração de nomes para não errar string mágica depois
const (
	ExchangeName = "ex.checkout"
	QueueName    = "q.activations"
	DLQName      = "q.activations.dlq"
	DLXName      = "ex.dlx" // Dead Letter Exchange
	RoutingKey   = "k.activation"
)

type RabbitMQ struct {
	Conn *amqp.Connection
	Ch   *amqp.Channel
}

// NewRabbitMQ abre a conexão e o canal
func NewRabbitMQ(user, pass, host, port string) (*RabbitMQ, error) {
	dsn := fmt.Sprintf("amqp://%s:%s@%s:%s/", user, pass, host, port)

	conn, err := amqp.Dial(dsn)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar no RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir canal: %w", err)
	}

	// Chama a função que cria as filas automaticamente
	err = setupTopology(ch)
	if err != nil {
		return nil, err
	}

	return &RabbitMQ{Conn: conn, Ch: ch}, nil
}

// setupTopology cria a arquitetura de resiliência (DLQ)
func setupTopology(ch *amqp.Channel) error {
	// 1. Declarar a DLX (Exchange dos Mortos)
	// Se der erro fatal, a mensagem vem pra cá
	err := ch.ExchangeDeclare(DLXName, "direct", true, false, false, false, nil)
	if err != nil {
		return err
	}

	// 2. Declarar a Fila de DLQ
	_, err = ch.QueueDeclare(DLQName, true, false, false, false, nil)
	if err != nil {
		return err
	}

	// 3. Ligar DLQ na DLX
	err = ch.QueueBind(DLQName, RoutingKey, DLXName, false, nil)
	if err != nil {
		return err
	}

	// ============================================================
	// 4. A MÁGICA: Declarar a Fila Principal apontando pra DLX
	// ============================================================
	args := amqp.Table{
		"x-dead-letter-exchange":    DLXName,    // Se der Nack, manda pra DLX
		"x-dead-letter-routing-key": RoutingKey, // Com essa chave
	}

	// 5. Declarar a Exchange Principal
	err = ch.ExchangeDeclare(ExchangeName, "direct", true, false, false, false, nil)
	if err != nil {
		return err
	}

	// 6. Declarar a Fila Principal (com os argumentos da DLQ)
	_, err = ch.QueueDeclare(QueueName, true, false, false, false, args)
	if err != nil {
		return err
	}

	// 7. Ligar Fila Principal na Exchange Principal
	err = ch.QueueBind(QueueName, RoutingKey, ExchangeName, false, nil)
	if err != nil {
		return err
	}

	return nil
}

// Publish envia a mensagem para a fila
func (r *RabbitMQ) Publish(body []byte) error {
	return r.Ch.Publish(
		ExchangeName, // Exchange
		RoutingKey,   // Key
		false,        // Mandatory
		false,        // Immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // Salva no disco (Durável)
		},
	)
}
