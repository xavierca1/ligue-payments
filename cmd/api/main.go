package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq" // Driver do Postgres

	"github.com/joho/godotenv"
	"github.com/xavierca1/ligue-payments/internal/infra/database"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/asaas"
	"github.com/xavierca1/ligue-payments/internal/infra/mail"
	"github.com/xavierca1/ligue-payments/internal/infra/queue" // <--- Importa o pacote que você criou
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

	rabbitMQ, err := queue.NewRabbitMQ("user", "password", "localhost", "5672")
	if err != nil {
		// Se não conectar na fila, mata a aplicação.
		// É melhor cair do que rodar "quebrado".
		panic(fmt.Sprintf("❌ Erro fatal no RabbitMQ: %v", err))
	}

	producer := queue.NewProducer(rabbitMQ.Conn, rabbitMQ.Ch)

	// Garante que fecha a conexão quando o programa parar
	defer rabbitMQ.Conn.Close()
	defer rabbitMQ.Ch.Close()

	fmt.Println("RabbitMQ 🐰 conectado e Topologia (Filas/DLQ) criada!")

	if dbURL == "" {
		println("Erro em buscar o database no .env")
	}
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

	customerRepo := database.NewCustomerRepository(db)
	planRepo := database.NewPlanRepository(db)

	gateway := asaas.NewClient(asaasKey, asaasURL)

	mailSender := mail.NewEmailSender(
		os.Getenv("MAIL_HOST"),
		587, // Converta se vier como string do env
		os.Getenv("MAIL_USER"),
		os.Getenv("MAIL_PASS"),
	)
	createCustomerUC := usecase.NewCreateCustomerUseCase(customerRepo,
		planRepo,
		gateway,
		// temAdapter,
		producer,
		mailSender,
		os.Getenv("SUPABASE_STORAGE_URL"))
	r := chi.NewRouter()

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

	port := ":8080"
	log.Printf(" Server CorePay rodando na porta %s", port)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatal(err)
	}
}
