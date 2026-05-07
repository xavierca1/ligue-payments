package usecase

import (
	"context"

	"github.com/xavierca1/ligue-payments/internal/entity"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/asaas"
	"github.com/xavierca1/ligue-payments/internal/infra/pdf"
	"github.com/xavierca1/ligue-payments/internal/infra/queue"
)

type BenefitProvider interface {
	RegisterBeneficiary(ctx context.Context, c *entity.Customer) (string, error)
}

// DependentInput representa um dependente no payload de entrada
type DependentInput struct {
	Name      string `json:"name"`
	CPF       string `json:"cpf"`
	BirthDate string `json:"birth_date"` // Formato: YYYY-MM-DD
	Gender    string `json:"gender"`     // "1", "2" ou "3"
	Kinship   string `json:"kinship"`    // FILHO, CONJUGE, PAI, MAE, etc
}

type CreateCustomerInput struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	CPF    string `json:"cpf"`
	PlanID string `json:"plan_id"`

	Phone             string `json:"phone"`
	CouponCode        string `json:"coupon_code,omitempty"`
	CheckoutAction    string `json:"checkout_action,omitempty"`
	BirthDate         string `json:"birth_date"`
	Gender            string `json:"gender"`
	MaritalStatus     string `json:"marital_status,omitempty"`
	PaymentMethod     string `json:"payment_method"`
	Street            string `json:"street"`
	Number            string `json:"number"`
	Complement        string `json:"complement"`
	District          string `json:"district"`
	City              string `json:"city"`
	State             string `json:"state"`
	ZipCode           string `json:"zip_code"`
	ExternalReference string `json:"externalReference,omitempty"`
	CardHolder        string `json:"card_holder"`
	CardNumber        string `json:"card_number"`
	CardMonth         string `json:"card_month"`
	CardYear          string `json:"card_year"`
	CardCVV           string `json:"card_cvv"`

	TermsAccepted   bool   `json:"terms_accepted"`
	TermsAcceptedAt string `json:"terms_accepted_at"` // Vem como string ISO do front
	TermsVersion    string `json:"terms_version"`

	Dependents []DependentInput `json:"dependents,omitempty"` // Lista de dependentes (opcional)
}

type CreateCustomerOutput struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Status       string `json:"status"`
	Msg          string `json:"msg"`
	PixCode      string `json:"pix_code"`        // Se for cartão, vai vazio. Se for PIX, vai cheio.
	PixQRCodeURL string `json:"pix_qr_code_url"` // O Front decide o que mostrar.
}

type CustomerRepositoryInterface interface {
	Create(ctx context.Context, c *entity.Customer) error
	Delete(ctx context.Context, id string) error
	FindByCPF(ctx context.Context, cpf string) (*entity.Customer, error)
	FindByEmailAndProductID(ctx context.Context, email, productID string) (*entity.Customer, error)
	UpdateGatewayID(ctx context.Context, customerID, gatewayID string) error
	UpdateStatus(ctx context.Context, customerID, status string) error
}

type SubscriptionRepository interface {
	Create(ctx context.Context, sub *entity.Subscription) error
	GetStatusByCustomerID(customerID string) (string, error)
	UpdateStatus(customerID string, status string) error
	FindLastByCustomerID(ctx context.Context, customerID string) (*entity.Subscription, error)
	DeleteByID(ctx context.Context, id string) error
}
type PlanRepositoryInterface interface {
	FindByID(ctx context.Context, id string) (*entity.Plan, error)
}

type PaymentGateway interface {
	CreateCustomer(input asaas.CreateCustomerInput) (string, error)
	Subscribe(input asaas.SubscribeInput) (string, string, error)
	SubscribePix(input asaas.SubscribePixInput) (string, *asaas.PixOutput, error)
	GetPixBySubscriptionID(subscriptionID string) (*asaas.PixOutput, error)
	DeleteSubscription(subscriptionID string) error
}

type QueueProducerInterface interface {
	PublishActivation(ctx context.Context, payload queue.ActivationPayload) error
}

type EmailService interface {
	SendWelcomeEmail(name, email string) error
	SendWelcomeEmailWithCard(name, email, cpf, planName, providerID string) error
	SendWelcomeEmailWithCardAndDependents(name, email, cpf, planName, providerID string, dependents []*entity.Dependent) error
	SendWelcomeEmailWithContractAndDependents(name, email, cpf, planName, providerID string, dependents []*entity.Dependent, contractPDF []byte) error
}
type KommoService interface {
	CreateLead(customerName, phone, email, planName string, price int) (int, error)
}

type ActivateSubscriptionInterface interface {
	Execute(ctx context.Context, input ActivateSubscriptionInput) error
}

type CreateCustomerUseCase struct {
	Repo             CustomerRepositoryInterface
	SubRepo          SubscriptionRepository
	PlanRepo         PlanRepositoryInterface
	Gateway          PaymentGateway
	BenefitService   BenefitProvider
	Queue            QueueProducerInterface
	EmailService     EmailService
	KommoService     KommoService
	WelcomeBucketURL string
	DependentRepo    entity.DependentRepositoryInterface
}

type ActivateSubscriptionInput struct {
	CustomerID string
	GatewayID  string
}

type ActivateSubscriptionUseCase struct {
	SubRepo         entity.SubscriptionRepository
	CustomerRepo    entity.CustomerRepositoryInterface
	PlanRepo        entity.PlanRepositoryInterface
	DependentRepo   entity.DependentRepositoryInterface
	Queue           queue.QueueProducerInterface
	EmailService    EmailService
	KommoService    KommoService
	ContractUC      *GenerateContractUseCase             // optional; skipped when nil
	DocuSealUseCase *GenerateContractWithDocuSealUseCase // optional; automatic document generation
}

// ContractPDFGeneratorInterface generates a filled and flattened PDF contract
// with a digital acceptance certificate appended as the last page.
type ContractPDFGeneratorInterface interface {
	Generate(planName string, data pdf.ContractFormData, clientIP string) ([]byte, error)
}

// ContractStorageInterface persists contract PDFs in a remote storage bucket.
type ContractStorageInterface interface {
	Upload(ctx context.Context, path string, data []byte) (string, error)
}

// GenerateContractInput carries all customer and plan data needed to produce the contract.
type GenerateContractInput struct {
	CustomerID string
	PlanName   string // must match the template filename without extension
	ClientIP   string

	Produto       string
	Valor         string
	Pagamento     string
	Periodicidade string
	Nome          string
	Nascimento    string
	CPF           string
	RG            string
	Orgao         string
	Sexo          string
	Civil         string
	Celular       string
	Fixo          string
	Email         string
	Endereco      string
	Numero        string
	Complemento   string
	Bairro        string
	Cidade        string
	UF            string
	CEP           string
}

type GenerateContractOutput struct {
	StoragePath string
	PublicURL   string
	PDFBytes    []byte // raw PDF kept in memory so callers can attach it to emails
}

type GenerateContractUseCase struct {
	Generator ContractPDFGeneratorInterface
	Storage   ContractStorageInterface
}

// ============================================================================
// DocuSeal Interfaces
// ============================================================================

// DocuSealClientInterface é a interface para o cliente DocuSeal
// NOTA: Implementado em internal/infra/integration/docuseal/client.go
type DocuSealClientInterface interface {
	CreateTemplate(req interface{}) (interface{}, error)
	CreateSubmission(req interface{}) (interface{}, error)
	GetSubmission(submissionUUID string) (interface{}, error)
}

// GenerateContractWithDocuSealInterface é a interface para o usecase de geração de contrato com DocuSeal
type GenerateContractWithDocuSealInterface interface {
	Execute(ctx context.Context, input DocuSealContractInput) (*DocuSealContractOutput, error)
	GetSignedDocument(ctx context.Context, submissionUUID string) (*GetSignedDocumentOutput, error)
}
