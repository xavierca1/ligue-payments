package usecase

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/xavierca1/ligue-payments/internal/entity"
)

type CreateCustomerInput struct {
	// 1. Identificação
	Name   string `json:"name"`
	Email  string `json:"email"`
	CPF    string `json:"cpf"`
	PlanID string `json:"plan_id"`

	Phone     string `json:"phone"`
	BirthDate string `json:"birth_date"` // YYYY-MM-DD
	Gender    string `json:"gender"`     // Recebe string "1" ou "0" do front

	Street     string `json:"street"`
	Number     string `json:"number"`
	Complement string `json:"complement"`
	District   string `json:"district"`
	City       string `json:"city"`
	State      string `json:"state"`
	ZipCode    string `json:"zip_code"`

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
	Msg    string `json:"msg"` // Adicionei para feedback visual no Postman
}

type CustomerRepositoryInterface interface {
	Create(ctx context.Context, c *entity.Customer) error
}

type PlanRepositoryInterface interface {
	FindByID(ctx context.Context, id string) (*entity.Plan, error)
}

type PaymentGateway interface {
	CreateCustomer(name, email, cpf string) (string, error)
	Subscribe(customerID string, amount float64, holder, number, month, year, ccv string) (string, string, error)
}

type CreateCustomerUseCase struct {
	Repo           CustomerRepositoryInterface
	PlanRepo       PlanRepositoryInterface
	Gateway        PaymentGateway
	BenefitService BenefitProvider
}

// CORREÇÃO 1: Atribuição correta no Construtor
func NewCreateCustomerUseCase(
	repo CustomerRepositoryInterface,
	planRepo PlanRepositoryInterface,
	gateway PaymentGateway,
	benefitService BenefitProvider,
) *CreateCustomerUseCase {
	return &CreateCustomerUseCase{
		Repo:           repo,
		PlanRepo:       planRepo,
		Gateway:        gateway,
		BenefitService: benefitService, // <--- Faltava essa linha! Sem ela o código quebrava.
	}
}

func (uc *CreateCustomerUseCase) Execute(ctx context.Context, input CreateCustomerInput) (*CreateCustomerOutput, error) {
	// 1. Conversão e Criação da Entidade
	genderInt, _ := strconv.Atoi(input.Gender)

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
		input.Phone,
		input.BirthDate,
		genderInt,
		address,
	)
	if err != nil {
		return nil, err
	}
	fmt.Printf("🔍 DEBUG INPUT: ID recebido = [%s]\n", input.PlanID)

	// 2. Busca Plano
	plan, err := uc.PlanRepo.FindByID(ctx, input.PlanID)
	if err != nil {

		return nil, errors.New("plano inválido ou não encontrado")
	}
	// 3. Gateway: Cria Cliente no Asaas
	gatewayID, err := uc.Gateway.CreateCustomer(customer.Name, customer.Email, customer.CPF)
	if err != nil {
		return nil, err
	}
	customer.GatewayID = gatewayID

	// 4. Gateway: Assinatura (Cobrança)
	amount := float64(plan.PriceCents) / 100.0
	subID, status, err := uc.Gateway.Subscribe(
		gatewayID,
		amount,
		input.CardHolder,
		input.CardNumber,
		input.CardMonth,
		input.CardYear,
		input.CardCVV,
	)
	if err != nil {
		return nil, fmt.Errorf("erro no pagamento: %w", err)
	}
	customer.SubscriptionID = subID

	// CORREÇÃO 2: Chamada do Provedor de Benefício (Tem Saúde)
	// Só chegamos aqui se o pagamento passou. Agora tem que provisionar.
	providerID, err := uc.BenefitService.RegisterBeneficiary(ctx, customer)
	if err != nil {
		// ALERTA CRÍTICO: Pagou mas não levou.
		// Retornamos o erro para saber, mas o dinheiro já saiu da conta do cliente.
		return nil, fmt.Errorf("pagamento aprovado (ID %s), mas erro ao gerar carteirinha: %w", subID, err)
	}

	// Sucesso total
	customer.ProviderID = providerID
	customer.Status = status // "ACTIVE"

	// 5. Salvar no Banco
	err = uc.Repo.Create(ctx, customer)
	if err != nil {
		return nil, err
	}

	return &CreateCustomerOutput{
		ID:     customer.ID,
		Name:   customer.Name,
		Email:  customer.Email,
		Status: status,
		Msg:    "Sucesso! Carteirinha: " + providerID,
	}, nil
}
