package queue

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// TelemedicinaClient define o contrato para integrações (Doc24, TEM, etc)
type TelemedicinaClient interface {
	CreateBeneficiary(ctx context.Context, input ActivationPayload) error
}

type Worker struct {
	Channel   *amqp.Channel
	DocClient TelemedicinaClient
	// Repo removido! O Worker agora é 100% desacoplado do Banco de Dados. 🚀
}

// NewWorker agora só precisa do Canal e do Cliente de Telemedicina
func NewWorker(ch *amqp.Channel, docClient TelemedicinaClient) *Worker {
	return &Worker{
		Channel:   ch,
		DocClient: docClient,
	}
}

func (w *Worker) Start(queueName string) {
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
			log.Printf("📥 [WORKER] Mensagem Recebida do RabbitMQ")

			var payload ActivationPayload
			if err := json.Unmarshal(d.Body, &payload); err != nil {
				log.Printf("❌ [WORKER] JSON Inválido: %s", err)
				// Mensagem podre (malformada). Rejeita sem requeue para não travar a fila.
				d.Nack(false, false)
				continue
			}

			log.Printf("⚙️ [WORKER] Processando ativação para: %s (Provider: %s)", payload.Name, payload.Provider)

			// Processamento Real
			if err := w.processMessage(context.Background(), payload); err != nil {
				log.Printf("❌ [WORKER] Erro na integração: %s", err)

				// Estratégia de Retentativa:
				// Se for erro de timeout/rede, idealmente faríamos d.Nack(false, true) para tentar de novo.
				// Como estamos em dev/testes, vou rejeitar para limpar a fila.
				d.Nack(false, false)
			} else {
				log.Printf("✅ [WORKER] Sucesso! Cliente %s integrado na %s.", payload.Name, payload.Provider)
				d.Ack(false) // Confirma o sucesso e remove da fila
			}
		}
	}()

	log.Printf(" [*] Worker rodando e aguardando na fila '%s'", queueName)
	<-forever
}

func (w *Worker) processMessage(ctx context.Context, payload ActivationPayload) error {
	// Roteamento de Provedor
	switch payload.Provider {
	case "DOC24":
		log.Println("🩺 Enviando dados completos para API da Doc24...")
		return w.DocClient.CreateBeneficiary(ctx, payload)

	case "TEM":
		log.Println("🏥 Enviando para API da TEM Saúde...")
		// return w.TemClient.Create(ctx, payload)
		return nil

	default:
		log.Printf("⚠️ Provedor desconhecido: %s. Apenas logando.", payload.Provider)
		// Retornamos nil para dar ACK e tirar essa mensagem da fila, já que não sabemos tratar
		return nil
	}
}
