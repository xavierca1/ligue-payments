package main

import (
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
	"github.com/xavierca1/ligue-payments/internal/infra/http/handlers" // 👈 Import novo
	"github.com/xavierca1/ligue-payments/internal/infra/integration/asaas"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/doc24"
	"github.com/xavierca1/ligue-payments/internal/infra/mail"
	"github.com/xavierca1/ligue-payments/internal/infra/queue"
	"github.com/xavierca1/ligue-payments/internal/usecase"
)

func main() {
	godotenv.Load()

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rabbitMQ, err := queue.NewRabbitMQ("user", "password", "localhost", "5672")
	if err != nil {
		panic(err)
	}
	defer rabbitMQ.Conn.Close()
	defer rabbitMQ.Ch.Close()

	customerRepo := database.NewCustomerRepository(db)
	planRepo := database.NewPlanRepository(db)
	subRepo := database.NewSubscriptionRepository(db)
	gateway := asaas.NewClient(os.Getenv("ASAAS_API_KEY"), os.Getenv("ASAAS_URL"))
	producer := queue.NewProducer(rabbitMQ.Conn, rabbitMQ.Ch)

	mailSender := mail.NewEmailSender(
		os.Getenv("MAIL_HOST"), 587, os.Getenv("MAIL_USER"), os.Getenv("MAIL_PASS"),
	)

	docClient := doc24.NewClient("liguemed", "J3xpZW50U2VjjkV0RG9jMjRNiOJlNDM=")
	worker := queue.NewWorker(rabbitMQ.Ch, docClient, customerRepo)
	go worker.Start(queue.QueueName)

	createCustomerUC := usecase.NewCreateCustomerUseCase(
		customerRepo, subRepo, planRepo, gateway, producer, mailSender,
		os.Getenv("SUPABASE_STORAGE_URL"),
	)

	checkoutHandler := handlers.NewCheckoutHandler(createCustomerUC)
	subHandler := handlers.NewSubscriptionHandler(subRepo)
	webhookHandler := handlers.NewWebhookHandler(customerRepo, subRepo, planRepo, producer)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173", "*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}))

	r.Post("/checkout", checkoutHandler.Handle)
	r.Get("/subscription/status/{customerId}", subHandler.HandleGetStatus)
	r.Post("/webhook", webhookHandler.Handle)

	port := ":8080"
	log.Printf("🔥 Server CorePay rodando na porta %s", port)
	http.ListenAndServe(port, r)
}
