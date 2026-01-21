package usecase

import (
	"context"

	"github.com/xavierca1/ligue-payments/internal/entity"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/asaas"
	"github.com/xavierca1/ligue-payments/internal/infra/queue"
)

type BenefitProvider interface {
	RegisterBeneficiary(ctx context.Context, c *entity.Customer) (string, error)
}
type CreateCustomerInput struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	CPF    string `json:"cpf"`
	PlanID string `json:"plan_id"`

	Phone         string `json:"phone"`
	BirthDate     string `json:"birth_date"`
	Gender        string `json:"gender"`
	PaymentMethod string `json:"payment_method"`
	Street        string `json:"street"`
	Number        string `json:"number"`
	Complement    string `json:"complement"`
	District      string `json:"district"`
	City          string `json:"city"`
	State         string `json:"state"`
	ZipCode       string `json:"zip_code"`
	OnixCode      string `json:"onix_code"`
	CardHolder    string `json:"card_holder"`
	CardNumber    string `json:"card_number"`
	CardMonth     string `json:"card_month"`
	CardYear      string `json:"card_year"`
	CardCVV       string `json:"card_cvv"`
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
}

type PlanRepositoryInterface interface {
	FindByID(ctx context.Context, id string) (*entity.Plan, error)
}

type PaymentGateway interface {
	CreateCustomer(input asaas.CreateCustomerInput) (string, error)
	Subscribe(input asaas.SubscribeInput) (string, string, error)
	SubscribePix(input asaas.SubscribePixInput) (string, *asaas.PixOutput, error)
}

type QueueProducerInterface interface {
	PublishActivation(ctx context.Context, payload queue.ActivationPayload) error
}

type EmailService interface {
	SendWelcome(to, name, productName, pdfLink string) error
}

type CreateCustomerUseCase struct {
	Repo             CustomerRepositoryInterface
	PlanRepo         PlanRepositoryInterface
	Gateway          PaymentGateway
	BenefitService   BenefitProvider
	Queue            QueueProducerInterface
	EmailService     EmailService
	WelcomeBucketURL string
}
