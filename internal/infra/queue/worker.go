package queue

import (
	"context"
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/xavierca1/ligue-payments/internal/entity"
	"github.com/xavierca1/ligue-payments/internal/infra/http/middleware"
)

type TelemedicinaClient interface {
	CreateBeneficiary(ctx context.Context, input ActivationPayload) error
	GetBeneficiaryID(cpf string) string
}

type Worker struct {
	Channel      *amqp.Channel
	DocClient    TelemedicinaClient
	CustomerRepo entity.CustomerRepositoryInterface
}

func NewWorker(ch *amqp.Channel, docClient TelemedicinaClient, customerRepo entity.CustomerRepositoryInterface) *Worker {
	return &Worker{
		Channel:      ch,
		DocClient:    docClient,
		CustomerRepo: customerRepo,
	}
}

func (w *Worker) Start(queueName string) {
	w.recordQueueDepth(queueName)

	msgs, err := w.Channel.Consume(
		queueName, // fila
		"",        // consumer
		false,     // auto-ack (manual é mais seguro)
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
			w.recordQueueDepth(queueName)
			log.Printf("📥 [WORKER] Mensagem Recebida do RabbitMQ")

			processingStart := time.Now()

			var payload ActivationPayload
			if err := json.Unmarshal(d.Body, &payload); err != nil {
				log.Printf("❌ [WORKER] JSON Inválido: %s", err)
				middleware.RecordQueueConsumed(queueName, "", "invalid_json")

				d.Nack(false, false)
				w.recordQueueDepth(queueName)
				continue
			}

			if !d.Timestamp.IsZero() {
				middleware.RecordQueueMessageArrivalLatency(queueName, payload.Provider, time.Since(d.Timestamp))
			}

			log.Printf("⚙️ [WORKER] Processando ativação para: %s (Provider: %s)", payload.Name, payload.Provider)

			if err := w.processMessage(context.Background(), payload); err != nil {
				log.Printf("❌ [WORKER] Erro na integração: %s", err)
				middleware.RecordQueueConsumed(queueName, payload.Provider, "failed")
				middleware.RecordQueueProcessingDuration(queueName, payload.Provider, time.Since(processingStart))

				d.Nack(false, false)
			} else {
				log.Printf("✅ [WORKER] Sucesso! Cliente %s integrado na %s.", payload.Name, payload.Provider)
				middleware.RecordQueueConsumed(queueName, payload.Provider, "success")
				middleware.RecordQueueProcessingDuration(queueName, payload.Provider, time.Since(processingStart))
				d.Ack(false) // Confirma o sucesso e remove da fila
			}

			w.recordQueueDepth(queueName)
		}
	}()

	log.Printf(" [*] Worker rodando e aguardando na fila '%s'", queueName)
	<-forever
}

func (w *Worker) recordQueueDepth(queueName string) {
	queueState, err := w.Channel.QueueInspect(queueName)
	if err != nil {
		return
	}

	middleware.RecordQueueDepth(queueName, queueState.Messages)
}

func (w *Worker) processMessage(ctx context.Context, payload ActivationPayload) error {

	switch payload.Provider {
	case "DOC24":
		log.Println("🩺 Enviando dados completos para API da Doc24...")

		if err := w.DocClient.CreateBeneficiary(ctx, payload); err != nil {
			return err
		}

		providerID := w.DocClient.GetBeneficiaryID(payload.CPF)

		if err := w.CustomerRepo.UpdateProviderID(ctx, payload.CustomerID, providerID); err != nil {
			log.Printf("⚠️ Falha ao salvar provider_id para customer %s: %v", payload.CustomerID, err)

		} else {
			log.Printf("✅ Provider ID salvo: customer=%s provider_id=%s", payload.CustomerID, providerID)
		}

		return nil

	case "TEM":
		log.Println("🏥 Enviando para API da TEM Saúde...")

		return nil

	default:
		log.Printf("⚠️ Provedor desconhecido: %s. Apenas logando.", payload.Provider)

		return nil
	}
}
