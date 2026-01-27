package queue

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/xavierca1/ligue-payments/internal/entity"
)

type TelemedicinaClient interface {
	CreateBeneficiary(ctx context.Context, input ActivationPayload) error
}

type Worker struct {
	Channel   *amqp.Channel
	DocClient TelemedicinaClient
	Repo      entity.CustomerRepositoryInterface
}

func NewWorker(ch *amqp.Channel, docClient TelemedicinaClient, repo entity.CustomerRepositoryInterface) *Worker {
	return &Worker{
		Channel:   ch,
		DocClient: docClient,
		Repo:      repo,
	}
}

func (w *Worker) Start(queueName string) {
	msgs, err := w.Channel.Consume(
		queueName, // fila
		"",        // consumer
		false,     // auto-ack (vamos fazer ack manual pra garantir segurança)
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		log.Fatalf("❌ Falha ao registrar consumidor RabbitMQ: %s", err)
	}

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("📥 Mensagem Recebida: %s", d.Body)

			var payload ActivationPayload
			if err := json.Unmarshal(d.Body, &payload); err != nil {
				log.Printf("❌ Erro ao decodificar JSON: %s", err)
				d.Nack(false, false) // Rejeita e não re-encaminha (mensagem podre)
				continue
			}

			// Processamento Real
			if err := w.processMessage(context.Background(), payload); err != nil {
				log.Printf("❌ Erro ao processar ativação: %s", err)
				// Se for erro temporário, poderia usar d.Nack(false, true) pra tentar de novo
				// Aqui vamos rejeitar para não travar a fila no dev
				d.Nack(false, false)
			} else {
				log.Printf("✅ Sucesso! Cliente %s integrado na %s.", payload.Name, payload.Provider)
				d.Ack(false) // Confirma que processou
			}
		}
	}()

	log.Printf(" [*] Worker rodando e aguardando mensagens na fila %s", queueName)
	<-forever
}

func (w *Worker) processMessage(ctx context.Context, payload ActivationPayload) error {
	// Roteamento de Provedor
	switch payload.Provider {
	case "DOC24":
		log.Println("🩺 Enviando para API da Doc24...")
		return w.DocClient.CreateBeneficiary(ctx, payload)

	case "TEM":
		log.Println("🏥 Enviando para API da TEM Saúde...")
		// return w.TemClient.Create(ctx, payload)
		return nil // TODO: Implementar TEM

	default:
		log.Printf("⚠️ Provedor desconhecido: %s. Apenas logando.", payload.Provider)
		return nil
	}
}
