package queue

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)


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


	err = setupTopology(ch)
	if err != nil {
		return nil, err
	}

	return &RabbitMQ{Conn: conn, Ch: ch}, nil
}


func setupTopology(ch *amqp.Channel) error {


	err := ch.ExchangeDeclare(DLXName, "direct", true, false, false, false, nil)
	if err != nil {
		return err
	}


	_, err = ch.QueueDeclare(DLQName, true, false, false, false, nil)
	if err != nil {
		return err
	}


	err = ch.QueueBind(DLQName, RoutingKey, DLXName, false, nil)
	if err != nil {
		return err
	}




	args := amqp.Table{
		"x-dead-letter-exchange":    DLXName,    // Se der Nack, manda pra DLX
		"x-dead-letter-routing-key": RoutingKey, // Com essa chave
	}


	err = ch.ExchangeDeclare(ExchangeName, "direct", true, false, false, false, nil)
	if err != nil {
		return err
	}


	_, err = ch.QueueDeclare(QueueName, true, false, false, false, args)
	if err != nil {
		return err
	}


	err = ch.QueueBind(QueueName, RoutingKey, ExchangeName, false, nil)
	if err != nil {
		return err
	}

	return nil
}


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
