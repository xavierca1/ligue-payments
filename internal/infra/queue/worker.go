package queue

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/xavierca1/ligue-payments/internal/infra/database"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/doc24"
)

// DTO para ler o JSON da fila
type ActivationMessage struct {
	CustomerID string `json:"customer_id"`
}

// A Struct do Worker
type Worker struct {
	channel      *amqp.Channel
	docClient    *doc24.Client
	customerRepo *database.CustomerRepository
}

// NewWorker cria a instância (Verifique se não tem nada depois do return aqui)
func NewWorker(ch *amqp.Channel, doc *doc24.Client, repo *database.CustomerRepository) *Worker {
	return &Worker{
		channel:      ch,
		docClient:    doc,
		customerRepo: repo,
	}
}

func (w *Worker) Start(queueName string) {
	// ❌ REMOVI O w.channel.QueueDeclare QUE CAUSAVA O CONFLITO
	// A fila já foi criada corretamente pelo rabbitmq.go

	// Vamos direto ao consumo!
	msgs, err := w.channel.Consume(
		queueName,
		"",    // consumer tag
		true,  // auto-ack (Cuidado: se der erro no código abaixo, a msg é perdida. Ideal é false e dar Ack manual no final)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args do consume (aqui pode ser nil de boa)
	)
	if err != nil {
		log.Fatalf("Erro ao registrar consumidor: %v", err)
	}

	log.Printf("👷 Worker rodando e ouvindo a fila: %s", queueName)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("📥 [Worker] Recebido: %s", d.Body)

			var msg ActivationMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Printf("❌ Erro JSON: %v", err)
				continue
			}

			// Busca Cliente
			customer, err := w.customerRepo.FindByID(msg.CustomerID)
			if err != nil {
				log.Printf("❌ Erro ao buscar cliente: %v", err)
				continue
			}

			// Integração Doc24
			log.Printf("🚀 Enviando %s para Doc24...", customer.Name)
			err = w.docClient.CreateBeneficiary(customer)
			if err != nil {
				log.Printf("❌ Erro Doc24: %v", err)
				// TODO: Se auto-ack fosse false, aqui daríamos Nack/Reject para cair na DLQ
				continue
			}

			log.Printf("✅ SUCESSO! Cliente ativado: %s", customer.Name)
		}
	}()

	<-forever
}
