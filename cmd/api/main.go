package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/rabbitmq/amqp091-go"

	// Driver PGX (Novo) - Compatível com Supabase Pooler
	"github.com/jackc/pgx/v5/stdlib"

	// Datadog APM
	sqltrace "github.com/DataDog/dd-trace-go/contrib/database/sql/v2"
	chitrace "github.com/DataDog/dd-trace-go/contrib/go-chi/chi.v5/v2"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"

	// Importações Internas do CorePay
	"github.com/xavierca1/ligue-payments/internal/infra/database"
	"github.com/xavierca1/ligue-payments/internal/infra/http/handlers"
	httpMiddleware "github.com/xavierca1/ligue-payments/internal/infra/http/middleware"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/asaas"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/doc24"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/docuseal"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/kommo"
	"github.com/xavierca1/ligue-payments/internal/infra/mail"
	"github.com/xavierca1/ligue-payments/internal/infra/pdf"
	"github.com/xavierca1/ligue-payments/internal/infra/queue"
	"github.com/xavierca1/ligue-payments/internal/infra/storage"
	"github.com/xavierca1/ligue-payments/internal/infra/worker"
	"github.com/xavierca1/ligue-payments/internal/usecase"
)

// ==========================================
// ADAPTERS & MIDDLEWARES
// ==========================================

// KommoAdapter adapta o Kommo Client para a interface KommoService
type KommoAdapter struct {
	client *kommo.Client
}

type noopQueueProducer struct{}

func (n *noopQueueProducer) PublishActivation(ctx context.Context, payload queue.ActivationPayload) error {
	return nil
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

func permissiveCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin == "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if strings.EqualFold(origin, "null") {
			w.Header().Set("Access-Control-Allow-Origin", "null")
			w.Header().Add("Vary", "Origin")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-Asaas-Signature, ngrok-skip-browser-warning")
		w.Header().Set("Access-Control-Max-Age", "300")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ==========================================
// PONTO DE ENTRADA (MAIN)
// ==========================================

func main() {
	// 1. Variáveis de Ambiente e Monitoramento
	if os.Getenv("RUNNING_IN_DOCKER") != "true" && os.Getenv("DOCKER_CONTAINER") != "true" {
		_ = godotenv.Load()
	}
	tracer.Start()
	defer tracer.Stop()

	// 2. Conexões de Infraestrutura (DB e Filas)
	db := setupDatabase()
	defer db.Close()

	rabbitMQ, err := queue.NewRabbitMQ(
		os.Getenv("RABBITMQ_USER"),
		os.Getenv("RABBITMQ_PASS"),
		os.Getenv("RABBITMQ_HOST"),
		os.Getenv("RABBITMQ_PORT"),
	)
	if err != nil {
		log.Printf("⚠️ RabbitMQ indisponível, iniciando API sem fila: %v", err)
	}
	if rabbitMQ != nil {
		defer rabbitMQ.Conn.Close()
		defer rabbitMQ.Ch.Close()
	}

	// 3. Inicialização de Repositórios
	customerRepo := database.NewCustomerRepository(db)
	planRepo := database.NewPlanRepository(db)
	subRepo := database.NewSubscriptionRepository(db)
	leadRepo := &database.LeadRepository{DB: db}
	dependentRepo := database.NewDependentRepository(db)
	couponRepo := database.NewCouponRepository(db)
	usecase.SetCouponTracker(couponRepo)

	// 4. Integrações e Serviços Externos
	mailSender := setupEmailService()
	gateway := asaas.NewClient(os.Getenv("ASAAS_API_KEY"), os.Getenv("ASAAS_URL"))
	var producer usecase.QueueProducerInterface = &noopQueueProducer{}
	if rabbitMQ != nil {
		producer = queue.NewProducer(rabbitMQ.Conn, rabbitMQ.Ch)
	}
	kommoAdapter := &KommoAdapter{client: kommo.NewClient()}

	var contractStorage usecase.ContractStorageInterface
	contractProjectURL := strings.TrimSpace(os.Getenv("SUPABASE_CONTRACTS_PROJECT_URL"))
	contractBucket := strings.TrimSpace(os.Getenv("SUPABASE_CONTRACTS_BUCKET"))
	contractServiceKey := strings.TrimSpace(os.Getenv("SUPABASE_CONTRACTS_SERVICE_ROLE_KEY"))
	if contractProjectURL != "" && contractBucket != "" && contractServiceKey != "" {
		contractStorage = storage.NewSupabaseStorage(contractProjectURL, contractBucket, contractServiceKey)
		log.Printf("✅ Storage de contratos inicializado no bucket %s", contractBucket)
	} else {
		log.Println("⚠️ Storage de contratos não configurado; PDFs serão enviados apenas por email")
	}

	docClient := doc24.NewClient(
		strings.TrimSpace(os.Getenv("DOC24_CLIENT_ID")),
		strings.TrimSpace(os.Getenv("DOC24_CLIENT_SECRET")),
	)

	// DocuSeal client (opcional - usa variáveis de ambiente DOCUSEAL_API_URL e DOCUSEAL_API_KEY)
	docuSealClient := docuseal.NewClient(strings.TrimSpace(os.Getenv("DOCUSEAL_API_URL")), strings.TrimSpace(os.Getenv("DOCUSEAL_API_KEY")))

	// 5. Workers de Background
	if rabbitMQ != nil {
		queueWorker := queue.NewWorker(rabbitMQ.Ch, docClient, customerRepo)
		go queueWorker.Start(queue.QueueName)
	}

	pixWorker := worker.NewPixExpirationWorker(db)
	go pixWorker.Start(context.Background())

	// 6. Casos de Uso (Business Logic)
	createCustomerUC := usecase.NewCreateCustomerUseCase(
		customerRepo, subRepo, planRepo, gateway, producer, mailSender, kommoAdapter,
		os.Getenv("SUPABASE_STORAGE_URL"),
		dependentRepo,
	)

	activateSubUC := usecase.NewActivateSubscriptionUseCase(
		subRepo, customerRepo, planRepo, dependentRepo, producer, mailSender, kommoAdapter,
	)
	activateSubUC.ContractUC = usecase.NewGenerateContractUseCase(
		pdf.NewContractGenerator("internal/infra/storage/plans_templates"),
		contractStorage,
	)
	// Instanciar DocuSeal usecase e registrar no activateSubUC
	docuSealUseCase := usecase.NewGenerateContractWithDocuSealUseCase(
		pdf.NewContractGenerator("internal/infra/storage/plans_templates"),
		docuSealClient,
	)
	activateSubUC.DocuSealUseCase = docuSealUseCase
	log.Println("✅ Gerador de contrato PDF e DocuSeal inicializado")

	// 7. Handlers (Controllers HTTP)
	customerHandler := handlers.NewCustomerHandler(createCustomerUC, subRepo, customerRepo)
	webhookHandler := handlers.NewWebhookHandler(customerRepo, activateSubUC)
	docusealWebhookHandler := handlers.NewDocuSealWebhookHandler(docuSealClient, mailSender)
	docusealTestHandler := handlers.NewDocuSealTestHandler(docuSealClient)
	docusealStatusHandler := handlers.NewDocuSealStatusHandler(docuSealClient)
	validationHandler := handlers.NewValidationHandler(customerRepo)
	leadHandler := handlers.NewLeadHandler(leadRepo)
	var rabbitMQConn *amqp091.Connection
	if rabbitMQ != nil {
		rabbitMQConn = rabbitMQ.Conn
	}
	healthHandler := handlers.NewHealthHandler(db, rabbitMQConn)
	emailHandler := handlers.NewEmailHandler(mailSender)
	couponHandler := handlers.NewCouponHandler()

	// 8. Roteamento (Chi)
	r := chi.NewRouter()

	// Middlewares globais
	r.Use(permissiveCORS)
	r.Use(middleware.Logger)
	r.Use(chitrace.Middleware())
	r.Use(httpMiddleware.Metrics)

	// Rotas da API
	r.Post("/checkout", customerHandler.CreateCheckoutHandler)
	r.Post("/customers/lookup-cpf", customerHandler.LookupCPFHandler)
	r.Post("/customers/lookup-email", validationHandler.LookupEmailHandler)
	r.Get("/customers/{id}/status", customerHandler.GetStatusHandler)
	r.Post("/customers/status", customerHandler.PostStatusHandler)
	r.Post("/webhook", webhookHandler.Handle)
	r.Post("/docuseal/webhook", docusealWebhookHandler.Handle)
	r.Post("/docuseal/test", docusealTestHandler.Handle)
	r.Post("/docuseal/status", docusealStatusHandler.Handle)
	r.Post("/validate-user", validationHandler.Handle)
	r.Post("/leads/capture", leadHandler.CaptureLead)
	r.Post("/test-email", emailHandler.SendTestWelcomeEmail)
	r.Post("/coupons/validate", couponHandler.Validate)

	// Health Checks
	r.Get("/health", healthHandler.Handle)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"unhealthy","reason":"database"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})

	// 9. Start do Servidor
	port := ":8080"
	if rawPort := strings.TrimSpace(os.Getenv("PORT")); rawPort != "" {
		if !strings.HasPrefix(rawPort, ":") {
			rawPort = ":" + rawPort
		}
		port = rawPort
	}
	log.Printf("🚀 Server CorePay rodando na porta %s", port)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatalf("❌ Falha fatal no servidor HTTP: %v", err)
	}
}

// ==========================================
// FUNÇÕES DE CONFIGURAÇÃO AUXILIARES
// ==========================================

// setupDatabase configura a conexão com Postgres usando PGX envolto no Datadog Tracer
func setupDatabase() *sql.DB {
	// Registra o driver do PGX no Datadog (substitui o antigo pq.Driver)
	sqltrace.Register("pgx", &stdlib.Driver{})

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("❌ DATABASE_URL não encontrada no arquivo .env")
	}

	dbURL = enableSimpleProtocol(dbURL)

	// Abre a conexão passando "pgx"
	db, err := sqltrace.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("❌ Erro ao inicializar conexão com banco: %v", err)
	}

	// Força um teste logo na subida da aplicação para evitar erro silencioso de EOF
	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Erro ao realizar Ping no banco (Verifique o EOF e se adicionou o ?sslmode=require): %v", err)
	}

	log.Println("✅ Banco de Dados conectado via PGX/Datadog!")
	return db
}

// setupEmailService decide qual provedor de email usar com base nas variáveis de ambiente
func enableSimpleProtocol(dbURL string) string {
	if strings.Contains(dbURL, "default_query_exec_mode=") {
		return dbURL
	}

	separator := "?"
	if strings.Contains(dbURL, "?") {
		separator = "&"
	}

	return dbURL + separator + "default_query_exec_mode=simple_protocol"
}

func setupEmailService() usecase.EmailService {
	useGraphEmail := strings.ToLower(os.Getenv("USE_GRAPH_EMAIL")) == "true"

	if useGraphEmail {
		clientID := strings.TrimSpace(os.Getenv("AZURE_CLIENT_ID"))
		clientSecret := strings.TrimSpace(os.Getenv("AZURE_CLIENT_SECRET"))
		tenantID := strings.TrimSpace(os.Getenv("AZURE_TENANT_ID"))

		if clientID != "" && clientSecret != "" && tenantID != "" {
			log.Println("📧 Inicializando Microsoft Graph API (OAuth2) para envio de emails")
			return mail.NewGraphEmailSender(
				clientID,
				clientSecret,
				tenantID,
				os.Getenv("MAIL_FROM"),
			)
		}

		log.Println("⚠️ USE_GRAPH_EMAIL está habilitado, mas as credenciais AZURE_* estão incompletas. Fazendo fallback para SMTP se configurado.")
	}

	log.Println("📧 Inicializando SMTP Padrão para envio de emails")
	mailPort := 587
	if p := os.Getenv("MAIL_PORT"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			mailPort = parsed
		}
	}
	return mail.NewEmailSenderWithFrom(
		os.Getenv("MAIL_HOST"),
		mailPort,
		os.Getenv("MAIL_USER"),
		os.Getenv("MAIL_PASS"),
		os.Getenv("MAIL_FROM"),
	)
}
