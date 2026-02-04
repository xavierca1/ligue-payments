package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/xavierca1/ligue-payments/internal/infra/database"
	"github.com/xavierca1/ligue-payments/internal/infra/http/handlers"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/asaas"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/doc24"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/kommo"
	"github.com/xavierca1/ligue-payments/internal/infra/mail"
	"github.com/xavierca1/ligue-payments/internal/infra/queue"
	"github.com/xavierca1/ligue-payments/internal/infra/worker"
	"github.com/xavierca1/ligue-payments/internal/usecase"
)

// KommoAdapter adapta o Kommo Client para a interface KommoService
type KommoAdapter struct {
	client *kommo.Client
}

func (a *KommoAdapter) CreateLead(customerName, phone, email, planName string, price int) (int, error) {
	return a.client.CreateLead(kommo.CreateLeadInput{
		CustomerName: customerName,
		Phone:        phone,
		Email:        email,
		PlanName:     planName,
		Price:        price,
	})
}

func main() {
	godotenv.Load()

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rabbitMQ, err := queue.NewRabbitMQ(
		os.Getenv("RABBITMQ_USER"),
		os.Getenv("RABBITMQ_PASS"),
		os.Getenv("RABBITMQ_HOST"),
		os.Getenv("RABBITMQ_PORT"),
	)
	if err != nil {
		panic(err)
	}
	defer rabbitMQ.Conn.Close()
	defer rabbitMQ.Ch.Close()

	// Repositories
	customerRepo := database.NewCustomerRepository(db)
	planRepo := database.NewPlanRepository(db)
	subRepo := database.NewSubscriptionRepository(db)
	leadRepo := &database.LeadRepository{DB: db}

	// External Services
	gateway := asaas.NewClient(os.Getenv("ASAAS_API_KEY"), os.Getenv("ASAAS_URL"))
	producer := queue.NewProducer(rabbitMQ.Conn, rabbitMQ.Ch)
	mailSender := mail.NewEmailSender(
		os.Getenv("MAIL_HOST"), 587, os.Getenv("MAIL_USER"), os.Getenv("MAIL_PASS"),
	)
	kommoClient := kommo.NewClient()
	kommoAdapter := &KommoAdapter{client: kommoClient}
	docClient := doc24.NewClient("liguemed", "J3xpZW50U2VjjkV0RG9jMjRNiOJlNDM=")

	// Background Workers
	queueWorker := queue.NewWorker(rabbitMQ.Ch, docClient, customerRepo)
	go queueWorker.Start(queue.QueueName)

	// Worker de Expiração de PIX (30 min)
	pixWorker := worker.NewPixExpirationWorker(db)
	ctx := context.Background()
	go pixWorker.Start(ctx)

	// UseCases
	createCustomerUC := usecase.NewCreateCustomerUseCase(
		customerRepo, subRepo, planRepo, gateway, producer, mailSender, kommoAdapter,
		os.Getenv("SUPABASE_STORAGE_URL"),
	)

	activateSubUC := usecase.NewActivateSubscriptionUseCase(
		subRepo, customerRepo, planRepo, producer, mailSender, kommoAdapter,
	)

	// Handlers
	customerHandler := handlers.NewCustomerHandler(createCustomerUC, subRepo)
	webhookHandler := handlers.NewWebhookHandler(customerRepo, activateSubUC)
	validationHandler := handlers.NewValidationHandler(customerRepo)
	leadHandler := handlers.NewLeadHandler(leadRepo)

	// Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173", "*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}))

	r.Post("/checkout", customerHandler.CreateCheckoutHandler)
	r.Get("/customers/{id}/status", customerHandler.GetStatusHandler)
	r.Post("/webhook", webhookHandler.Handle)
	r.Post("/validate-user", validationHandler.Handle)
	r.Post("/leads/capture", leadHandler.CaptureLead)

	// Health checks
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		// Verifica se banco de dados está acessível
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"unhealthy","reason":"database"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})

	port := ":8080"
	log.Printf("🚀 Server CorePay rodando na porta %s", port)
	http.ListenAndServe(port, r)
}
