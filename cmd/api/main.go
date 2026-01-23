package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	_ "github.com/lib/pq" // Driver do Postgres

	"github.com/joho/godotenv"
	"github.com/xavierca1/ligue-payments/internal/infra/database"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/asaas"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/doc24" // <--- Import Novo
	"github.com/xavierca1/ligue-payments/internal/infra/mail"
	"github.com/xavierca1/ligue-payments/internal/infra/queue"
	"github.com/xavierca1/ligue-payments/internal/usecase"
)

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Println("Arquivo env não encontrado ")
	}

	dbURL := os.Getenv("DATABASE_URL")
	asaasKey := os.Getenv("ASAAS_API_KEY")
	asaasURL := os.Getenv("ASAAS_URL")
	// temUrl := os.Getenv("TEM_SAUDE_URL")
	// temToken := os.Getenv("TEM_SAUDE_TOKEN")
	// temAdapter := temsaude.NewClient(temUrl, temToken)

	// Configuração do RabbitMQ
	rabbitMQ, err := queue.NewRabbitMQ("user", "password", "localhost", "5672")
	if err != nil {
		// Se não conectar na fila, mata a aplicação.
		panic(fmt.Sprintf(" Erro fatal no RabbitMQ: %v", err))
	}

	producer := queue.NewProducer(rabbitMQ.Conn, rabbitMQ.Ch)

	// Garante que fecha a conexão quando o programa parar
	defer rabbitMQ.Conn.Close()
	defer rabbitMQ.Ch.Close()

	fmt.Println("RabbitMQ 🐰 conectado e Topologia (Filas/DLQ) criada!")

	if dbURL == "" || asaasKey == "" || asaasURL == "" {
		log.Fatal("ERRO: Configure DB_URL, ASAAS_API_KEY e ASAAS_URL no .env")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Erro ao abrir conexão com banco:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Erro ao conectar no banco (Ping):", err)
	}
	log.Println("Banco de Dados Conectado com Sucesso!")

	// Repositórios
	customerRepo := database.NewCustomerRepository(db)
	planRepo := database.NewPlanRepository(db)

	// Gateways
	gateway := asaas.NewClient(asaasKey, asaasURL)

	mailSender := mail.NewEmailSender(
		os.Getenv("MAIL_HOST"),
		587,
		os.Getenv("MAIL_USER"),
		os.Getenv("MAIL_PASS"),
	)

	docClient := doc24.NewClient("liguemed", "J3xpZW50U2VjjkV0RG9jMjRNiOJlNDM=")

	worker := queue.NewWorker(rabbitMQ.Ch, docClient, customerRepo)

	// Mude de "q.activations" para:
	go worker.Start(queue.QueueName)
	log.Println("👷 Worker Doc24 iniciado e ouvindo a fila 'activation_queue'...")

	createCustomerUC := usecase.NewCreateCustomerUseCase(customerRepo,
		planRepo,
		gateway,
		// temAdapter,
		producer,
		mailSender,
		os.Getenv("SUPABASE_STORAGE_URL"))

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		// Permite o front (Vite) e localhost genérico
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000", "*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	// ---------------------------------------

	r.Use(middleware.Logger)

	r.Post("/checkout", func(w http.ResponseWriter, r *http.Request) {
		var input usecase.CreateCustomerInput

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
			return
		}

		output, err := createCustomerUC.Execute(r.Context(), input)
		if err != nil {
			log.Printf("Erro no checkout: %v", err)

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(output)
	})

	r.Post("/webhook", func(w http.ResponseWriter, r *http.Request) {
		// ... (código de autenticação e decode igual ao seu) ...

		var event struct {
			Event   string `json:"event"`
			Payment struct {
				ID          string `json:"id"`
				Customer    string `json:"customer"`
				BillingType string `json:"billingType"`
				Status      string `json:"status"`
			} `json:"payment"`
		}

		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			log.Printf("❌ Erro decode webhook: %v", err)
			http.Error(w, "Bad JSON", 400)
			return
		}

		// Filtra eventos de pagamento real
		if event.Event != "PAYMENT_RECEIVED" && event.Event != "PAYMENT_CONFIRMED" {
			w.WriteHeader(200)
			return
		}

		// Busca o cliente pelo ID do Asaas (cus_xxxx)
		localCustomer, err := customerRepo.FindByGatewayID(event.Payment.Customer)
		if err != nil {
			log.Printf("❌ Cliente não encontrado: %v", err)
			w.WriteHeader(200) // Retorna 200 pro Asaas não ficar retentando em loop infinito
			return
		}

		// 🚨 O PULO DO GATO PARA O FRONTEND FUNCIONAR 🚨
		// Precisamos atualizar o banco AGORA para o Polling pegar
		// Supondo que você tenha um método UpdateStatus no seu repo, ou rodando a query direta:

		// Opção A: Via Repo (Recomendado)
		err = customerRepo.UpdateStatus(r.Context(), localCustomer.ID, "CONFIRMED")

		// Opção B: SQL Direto (Se não tiver o método no repo ainda)
		// _, err = dbConn.ExecContext(r.Context(), "UPDATE customers SET status = 'CONFIRMED' WHERE id = $1", localCustomer.ID)

		if err != nil {
			log.Printf("❌ Erro ao atualizar status no banco: %v", err)
			// Não damos erro fatal aqui para não travar a fila, mas logamos
		} else {
			log.Printf("✅ Status do cliente %s atualizado para CONFIRMED!", localCustomer.Name)
		}
		// ---------------------------------------------------

		plan, _ := planRepo.FindByID(r.Context(), localCustomer.PlanID)
		provider := "DOC24"
		if plan != nil {
			provider = plan.Provider
		}

		payload := queue.ActivationPayload{
			CustomerID: localCustomer.ID,
			PlanID:     localCustomer.PlanID,
			Provider:   provider,
			Name:       localCustomer.Name,
			Email:      localCustomer.Email,
			Origin:     "WEBHOOK_ASAAS_REAL",
		}

		err = producer.PublishActivation(r.Context(), payload)
		if err != nil {
			log.Printf("❌ Erro ao publicar na fila: %v", err)
			w.WriteHeader(500)
			return
		}

		log.Printf("🚀 Cliente %s enviado para fila de ativação!", localCustomer.Name)
		w.WriteHeader(200)
	})
	port := ":8080"
	log.Printf("🔥 Server CorePay rodando na porta %s", port)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatal(err)
	}
}
