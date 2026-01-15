package usecase

import (
	"context"
	"fmt"
	"strconv"

	"github.com/xavierca1/ligue-payments/internal/entity"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/asaas"
)

type CreateCustomerInput struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	CPF    string `json:"cpf"`
	PlanID string `json:"plan_id"`

	Phone     string `json:"phone"`
	BirthDate string `json:"birth_date"`
	Gender    string `json:"gender"`

	Street     string `json:"street"`
	Number     string `json:"number"`
	Complement string `json:"complement"`
	District   string `json:"district"`
	City       string `json:"city"`
	State      string `json:"state"`
	ZipCode    string `json:"zip_code"`
	OnixCode   string `json:"onix_code"`
	CardHolder string `json:"card_holder"`
	CardNumber string `json:"card_number"`
	CardMonth  string `json:"card_month"`
	CardYear   string `json:"card_year"`
	CardCVV    string `json:"card_cvv"`
}

type CreateCustomerOutput struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Status string `json:"status"`
	Msg    string `json:"msg"`
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
}

// ---------------------------------------------------------
// CORREÇÃO 1: Definindo a interface que faltava
// ---------------------------------------------------------

type EmailService interface {
	SendWelcome(to, name, productName, pdfLink string) error
}

type CreateCustomerUseCase struct {
	Repo           CustomerRepositoryInterface
	PlanRepo       PlanRepositoryInterface
	Gateway        PaymentGateway
	BenefitService BenefitProvider
	EmailService   EmailService
}

func NewCreateCustomerUseCase(
	repo CustomerRepositoryInterface,
	planRepo PlanRepositoryInterface,
	gateway PaymentGateway,
	benefitService BenefitProvider,
	emailService EmailService,
) *CreateCustomerUseCase {
	return &CreateCustomerUseCase{
		Repo:           repo,
		PlanRepo:       planRepo,
		Gateway:        gateway,
		BenefitService: benefitService,
		// ---------------------------------------------------------
		// CORREÇÃO 2: Adicionando o EmailService aqui para não dar Panic
		// ---------------------------------------------------------
		EmailService: emailService,
	}
}

func (uc *CreateCustomerUseCase) Execute(ctx context.Context, input CreateCustomerInput) (*CreateCustomerOutput, error) {
	genderInt, _ := strconv.Atoi(input.Gender)

	plan, err := uc.PlanRepo.FindByID(ctx, input.PlanID)
	if err != nil {
		return nil, fmt.Errorf("plano inválido ou não encontrado: %w", err)
	}

	asaasCustomerID, err := uc.Gateway.CreateCustomer(asaas.CreateCustomerInput{
		Name:          input.Name,
		Email:         input.Email,
		CpfCnpj:       input.CPF,
		Phone:         input.Phone,
		MobilePhone:   input.Phone,
		PostalCode:    input.ZipCode,
		AddressNumber: input.Number,
	})
	if err != nil {
		return nil, fmt.Errorf("erro ao criar cliente no asaas: %w", err)
	}

	address := entity.Address{
		Street:     input.Street,
		Number:     input.Number,
		Complement: input.Complement,
		District:   input.District,
		City:       input.City,
		State:      input.State,
		ZipCode:    input.ZipCode,
	}

	customer, err := entity.NewCustomer(
		input.Name,
		input.Email,
		input.CPF,
		input.OnixCode,
		input.Phone,
		input.BirthDate,
		genderInt,
		address,
	)
	if err != nil {
		return nil, err
	}

	customer.GatewayID = asaasCustomerID
	customer.OnixCode = plan.ProviderPlanCode
	amount := float64(plan.PriceCents) / 100.0

	subID, status, err := uc.Gateway.Subscribe(asaas.SubscribeInput{
		CustomerID:       asaasCustomerID,
		Price:            amount,
		CardHolderName:   input.CardHolder,
		CardNumber:       input.CardNumber,
		CardMonth:        input.CardMonth,
		CardYear:         input.CardYear,
		CardCCV:          input.CardCVV,
		HolderEmail:      input.Email,
		HolderCpfCnpj:    input.CPF,
		HolderPostalCode: input.ZipCode,
		HolderAddressNum: input.Number,
		HolderPhone:      input.Phone,
	})

	if err != nil {
		return nil, fmt.Errorf("erro no pagamento: %w", err)
	}
	customer.SubscriptionID = subID

	providerID, err := uc.BenefitService.RegisterBeneficiary(ctx, customer)
	if err != nil {
		return nil, fmt.Errorf("pagamento aprovado (ID %s), mas erro ao gerar carteirinha: %w", subID, err)
	}

	customer.ProviderID = providerID
	customer.Status = status

	err = uc.Repo.Create(ctx, customer)
	if err != nil {
		return nil, fmt.Errorf("erro ao salvar venda no banco: %w", err)
	}

	// TODO: Mover URL para .env
	bucketBaseURL := "https://yntprscrhdlrwkgnmzrb.supabase.co/storage/v1/object/public/public-assets/welcome-kits"

	pdfLink := fmt.Sprintf("%s/kit_%s.pdf", bucketBaseURL, plan.ProviderPlanCode)

	go func() {
		// Agora isso vai funcionar porque EmailService foi injetado corretamente
		err := uc.EmailService.SendWelcome(input.Email, input.Name, plan.Name, pdfLink)
		if err != nil {
			fmt.Printf("failed to send welcome email to %s: %v\n", input.Email, err)
		}
	}()

	return &CreateCustomerOutput{
		ID:     customer.ID,
		Name:   customer.Name,
		Email:  customer.Email,
		Status: status,
		Msg:    "Sucesso! Carteirinha: " + providerID,
	}, nil
}
