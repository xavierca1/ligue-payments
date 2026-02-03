package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xavierca1/ligue-payments/internal/entity"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/asaas"
	"github.com/xavierca1/ligue-payments/internal/infra/queue"
	"github.com/xavierca1/ligue-payments/internal/usecase"
)

// MockPlanRepository
type MockPlanRepository struct {
	mock.Mock
}

func (m *MockPlanRepository) FindByID(ctx context.Context, id string) (*entity.Plan, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Plan), args.Error(1)
}

// MockCustomerRepository
type MockCustomerRepository struct {
	mock.Mock
}

func (m *MockCustomerRepository) Create(ctx context.Context, customer *entity.Customer) error {
	args := m.Called(ctx, customer)
	return args.Error(0)
}

func (m *MockCustomerRepository) FindByGatewayID(id string) (*entity.Customer, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Customer), args.Error(1)
}

func (m *MockCustomerRepository) FindByID(ctx context.Context, id string) (*entity.Customer, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Customer), args.Error(1)
}

func (m *MockCustomerRepository) CheckDuplicity(ctx context.Context, email, cpf string) (bool, error) {
	args := m.Called(ctx, email, cpf)
	return args.Bool(0), args.Error(1)
}

func (m *MockCustomerRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCustomerRepository) UpdateProviderID(ctx context.Context, customerID, providerID string) error {
	args := m.Called(ctx, customerID, providerID)
	return args.Error(0)
}

// MockSubscriptionRepository
type MockSubscriptionRepository struct {
	mock.Mock
}

func (m *MockSubscriptionRepository) Create(ctx context.Context, sub *entity.Subscription) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}

func (m *MockSubscriptionRepository) GetStatusByCustomerID(customerID string) (string, error) {
	args := m.Called(customerID)
	return args.String(0), args.Error(1)
}

func (m *MockSubscriptionRepository) UpdateStatus(customerID string, status string) error {
	args := m.Called(customerID, status)
	return args.Error(0)
}

func (m *MockSubscriptionRepository) FindLastByCustomerID(ctx context.Context, customerID string) (*entity.Subscription, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Subscription), args.Error(1)
}

// MockPaymentGateway
type MockPaymentGateway struct {
	mock.Mock
}

func (m *MockPaymentGateway) CreateCustomer(input asaas.CreateCustomerInput) (string, error) {
	args := m.Called(input)
	return args.String(0), args.Error(1)
}

func (m *MockPaymentGateway) Subscribe(input asaas.SubscribeInput) (string, string, error) {
	args := m.Called(input)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockPaymentGateway) SubscribePix(input asaas.SubscribePixInput) (string, *asaas.PixOutput, error) {
	args := m.Called(input)
	if args.Get(1) == nil {
		return args.String(0), nil, args.Error(2)
	}
	return args.String(0), args.Get(1).(*asaas.PixOutput), args.Error(2)
}

// MockQueueProducer
type MockQueueProducer struct {
	mock.Mock
}

func (m *MockQueueProducer) PublishActivation(ctx context.Context, payload queue.ActivationPayload) error {
	args := m.Called(ctx, payload)
	return args.Error(0)
}

// MockEmailService
type MockEmailService struct {
	mock.Mock
}

func (m *MockEmailService) SendWelcome(to, name, productName, pdfLink string) error {
	args := m.Called(to, name, productName, pdfLink)
	return args.Error(0)
}

type MockWhatsAppService struct {
	mock.Mock
}

func (m *MockWhatsAppService) SendWelcome(phone, name, planName, templateID string) error {
	args := m.Called(phone, name, planName, templateID)
	return args.Error(0)
}

// ============ TESTES ============

// TestCreateCustomerPixFlowSuccess - Teste completo do fluxo PIX com sucesso
func TestCreateCustomerPixFlowSuccess(t *testing.T) {
	ctx := context.Background()

	// Setup dos mocks
	mockPlanRepo := new(MockPlanRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	mockSubRepo := new(MockSubscriptionRepository)
	mockGateway := new(MockPaymentGateway)
	mockQueue := new(MockQueueProducer)
	mockEmailService := new(MockEmailService)
	mockEmailService.On("SendWelcome", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Cenário: PIX com sucesso
	plan := &entity.Plan{
		ID:               "plan-123",
		Name:             "Plano Premium",
		PriceCents:       29900,
		Provider:         "DOC24",
		ProductID:        "prod-123",
		ProviderPlanCode: "ligue saude em dia individual",
	}

	mockPlanRepo.On("FindByID", ctx, "plan-123").Return(plan, nil)
	mockGateway.On("CreateCustomer", mock.Anything).Return("asaas-cust-123", nil)
	mockGateway.On("SubscribePix", mock.Anything).Return("asaas-sub-456", &asaas.PixOutput{
		CopyPaste: "00020126580014br.gov.bcb.pix",
		URL:       "data:image/png;base64,iVBORw0KG...",
	}, nil)

	mockCustomerRepo.On("Create", ctx, mock.Anything).Return(nil)
	mockSubRepo.On("Create", ctx, mock.Anything).Return(nil)
	mockEmailService.On("SendWelcome", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	uc := usecase.NewCreateCustomerUseCase(
		mockCustomerRepo, mockSubRepo, mockPlanRepo,
		mockGateway, mockQueue, mockEmailService, nil,
		"https://storage.example.com",
	)

	// Input válido - PIX
	input := usecase.CreateCustomerInput{
		Name:            "João Silva",
		Email:           "joao@example.com",
		CPF:             "123.456.789-00",
		Phone:           "(11) 99999-9999",
		BirthDate:       "1990-05-15",
		Gender:          "1", // M
		PlanID:          "plan-123",
		PaymentMethod:   "PIX",
		Street:          "Rua A",
		Number:          "123",
		Complement:      "Apto 45",
		District:        "Centro",
		City:            "São Paulo",
		State:           "SP",
		ZipCode:         "01310-100",
		OnixCode:        "7065",
		TermsAccepted:   true,
		TermsAcceptedAt: time.Now().Format(time.RFC3339),
		TermsVersion:    "1.0",
	}

	// Executar
	output, err := uc.Execute(ctx, input)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, "WAITING_PAYMENT", output.Status)
	assert.NotEmpty(t, output.ID)
	assert.NotEmpty(t, output.PixCode)
	assert.NotEmpty(t, output.PixQRCodeURL)
	assert.Equal(t, "Pré-cadastro realizado com sucesso!", output.Msg)

	// Verifica se os mocks foram chamados
	mockPlanRepo.AssertCalled(t, "FindByID", ctx, "plan-123")
	mockGateway.AssertCalled(t, "CreateCustomer", mock.Anything)
	mockGateway.AssertCalled(t, "SubscribePix", mock.Anything)
	mockCustomerRepo.AssertCalled(t, "Create", ctx, mock.Anything)
	mockSubRepo.AssertCalled(t, "Create", ctx, mock.Anything)
}

// TestCreateCustomerCreditCardFlowSuccess - Teste completo do fluxo de Cartão de Crédito com sucesso
func TestCreateCustomerCreditCardFlowSuccess(t *testing.T) {
	ctx := context.Background()

	// Setup dos mocks
	mockPlanRepo := new(MockPlanRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	mockSubRepo := new(MockSubscriptionRepository)
	mockGateway := new(MockPaymentGateway)
	mockQueue := new(MockQueueProducer)
	mockEmailService := new(MockEmailService)
	mockEmailService.On("SendWelcome", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Cenário: Cartão de Crédito com sucesso
	plan := &entity.Plan{
		ID:               "plan-456",
		Name:             "Plano Standard",
		PriceCents:       19900,
		Provider:         "DOC24",
		ProductID:        "prod-456",
		ProviderPlanCode: "ligue saude basico",
	}

	mockPlanRepo.On("FindByID", ctx, "plan-456").Return(plan, nil)
	mockGateway.On("CreateCustomer", mock.Anything).Return("asaas-cust-789", nil)
	mockGateway.On("Subscribe", mock.Anything).Return("asaas-sub-789", "ACTIVE", nil)

	mockCustomerRepo.On("Create", ctx, mock.Anything).Return(nil)
	mockSubRepo.On("Create", ctx, mock.Anything).Return(nil)

	mockWhatsApp := new(MockWhatsAppService)
	mockWhatsApp.On("SendWelcome", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	uc := usecase.NewCreateCustomerUseCase(
		mockCustomerRepo, mockSubRepo, mockPlanRepo,
		mockGateway, mockQueue, mockEmailService, mockWhatsApp,
		"https://storage.example.com",
	)

	// Input válido - Cartão de Crédito
	input := usecase.CreateCustomerInput{
		Name:            "Maria Santos",
		Email:           "maria@example.com",
		CPF:             "987.654.321-00",
		Phone:           "(21) 98888-8888",
		BirthDate:       "1985-03-22",
		Gender:          "2", // F
		PlanID:          "plan-456",
		PaymentMethod:   "CREDIT_CARD",
		Street:          "Avenida B",
		Number:          "456",
		District:        "Zona Norte",
		City:            "Rio de Janeiro",
		State:           "RJ",
		ZipCode:         "20040-020",
		OnixCode:        "7065",
		CardHolder:      "MARIA SANTOS",
		CardNumber:      "4532015112830366", // Número válido de teste (Luhn check)
		CardMonth:       "12",
		CardYear:        "26",
		CardCVV:         "123",
		TermsAccepted:   true,
		TermsAcceptedAt: time.Now().Format(time.RFC3339),
		TermsVersion:    "1.0",
	}

	// Executar
	output, err := uc.Execute(ctx, input)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, "WAITING_PAYMENT", output.Status)
	assert.NotEmpty(t, output.ID)
	assert.Empty(t, output.PixCode)      // Cartão não tem PIX
	assert.Empty(t, output.PixQRCodeURL) // Cartão não tem QRCode
	assert.Equal(t, "Pré-cadastro realizado com sucesso!", output.Msg)

	// Verifica se os mocks foram chamados
	mockPlanRepo.AssertCalled(t, "FindByID", ctx, "plan-456")
	mockGateway.AssertCalled(t, "CreateCustomer", mock.Anything)
	mockGateway.AssertCalled(t, "Subscribe", mock.Anything)
	mockCustomerRepo.AssertCalled(t, "Create", ctx, mock.Anything)
	mockSubRepo.AssertCalled(t, "Create", ctx, mock.Anything)
}

// TestCreateCustomerValidationFailure - Teste de falha de validação
func TestCreateCustomerValidationFailure(t *testing.T) {
	ctx := context.Background()

	mockPlanRepo := new(MockPlanRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	mockSubRepo := new(MockSubscriptionRepository)
	mockGateway := new(MockPaymentGateway)
	mockQueue := new(MockQueueProducer)
	mockEmailService := new(MockEmailService)
	mockEmailService.On("SendWelcome", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	uc := usecase.NewCreateCustomerUseCase(
		mockCustomerRepo, mockSubRepo, mockPlanRepo,
		mockGateway, mockQueue, mockEmailService, nil,
		"https://storage.example.com",
	)

	// Input inválido - Email faltando
	input := usecase.CreateCustomerInput{
		Name:            "João",
		Email:           "", // Email vazio!
		CPF:             "123.456.789-00",
		Phone:           "(11) 99999-9999",
		BirthDate:       "1990-05-15",
		Gender:          "1",
		PlanID:          "plan-123",
		PaymentMethod:   "PIX",
		Street:          "Rua A",
		Number:          "123",
		District:        "Centro",
		City:            "São Paulo",
		State:           "SP",
		ZipCode:         "01310-100",
		OnixCode:        "7065",
		TermsAccepted:   true,
		TermsAcceptedAt: time.Now().Format(time.RFC3339),
		TermsVersion:    "1.0",
	}

	output, err := uc.Execute(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.True(t, usecase.IsDomainError(err))

	// Nenhum mock deve ter sido chamado
	mockPlanRepo.AssertNotCalled(t, "FindByID")
	mockGateway.AssertNotCalled(t, "CreateCustomer")
}

// TestCreateCustomerPlanNotFound - Teste quando plano não existe
func TestCreateCustomerPlanNotFound(t *testing.T) {
	ctx := context.Background()

	mockPlanRepo := new(MockPlanRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	mockSubRepo := new(MockSubscriptionRepository)
	mockGateway := new(MockPaymentGateway)
	mockQueue := new(MockQueueProducer)
	mockEmailService := new(MockEmailService)
	mockEmailService.On("SendWelcome", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Plano não existe
	mockPlanRepo.On("FindByID", ctx, "plan-inexistente").Return(nil, errors.New("not found"))

	uc := usecase.NewCreateCustomerUseCase(
		mockCustomerRepo, mockSubRepo, mockPlanRepo,
		mockGateway, mockQueue, mockEmailService, nil,
		"https://storage.example.com",
	)

	input := usecase.CreateCustomerInput{
		Name:            "João Silva",
		Email:           "joao@example.com",
		CPF:             "123.456.789-00",
		Phone:           "(11) 99999-9999",
		BirthDate:       "1990-05-15",
		Gender:          "1",
		PlanID:          "plan-inexistente",
		PaymentMethod:   "PIX",
		Street:          "Rua A",
		Number:          "123",
		District:        "Centro",
		City:            "São Paulo",
		State:           "SP",
		ZipCode:         "01310-100",
		OnixCode:        "7065",
		TermsAccepted:   true,
		TermsAcceptedAt: time.Now().Format(time.RFC3339),
		TermsVersion:    "1.0",
	}

	output, err := uc.Execute(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.True(t, usecase.IsDomainError(err))
	mockPlanRepo.AssertCalled(t, "FindByID", ctx, "plan-inexistente")
}

// TestCreateCustomerPaymentFailure - Teste quando o gateway rejeita o pagamento
func TestCreateCustomerPaymentFailure(t *testing.T) {
	ctx := context.Background()

	mockPlanRepo := new(MockPlanRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	mockSubRepo := new(MockSubscriptionRepository)
	mockGateway := new(MockPaymentGateway)
	mockQueue := new(MockQueueProducer)
	mockEmailService := new(MockEmailService)
	mockEmailService.On("SendWelcome", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	plan := &entity.Plan{
		ID:         "plan-123",
		Name:       "Plano Premium",
		PriceCents: 29900,
		Provider:   "DOC24",
		ProductID:  "prod-123",
	}

	mockPlanRepo.On("FindByID", ctx, "plan-123").Return(plan, nil)
	mockGateway.On("CreateCustomer", mock.Anything).Return("asaas-cust-123", nil)
	mockGateway.On("SubscribePix", mock.Anything).Return("", nil, errors.New("payment declined"))

	uc := usecase.NewCreateCustomerUseCase(
		mockCustomerRepo, mockSubRepo, mockPlanRepo,
		mockGateway, mockQueue, mockEmailService, nil,
		"https://storage.example.com",
	)

	input := usecase.CreateCustomerInput{
		Name:            "João Silva",
		Email:           "joao@example.com",
		CPF:             "123.456.789-00",
		Phone:           "(11) 99999-9999",
		BirthDate:       "1990-05-15",
		Gender:          "1",
		PlanID:          "plan-123",
		PaymentMethod:   "PIX",
		Street:          "Rua A",
		Number:          "123",
		District:        "Centro",
		City:            "São Paulo",
		State:           "SP",
		ZipCode:         "01310-100",
		OnixCode:        "7065",
		TermsAccepted:   true,
		TermsAcceptedAt: time.Now().Format(time.RFC3339),
		TermsVersion:    "1.0",
	}

	output, err := uc.Execute(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.True(t, usecase.IsDomainError(err))

	// O customer não deve ser criado se o pagamento falhar
	mockCustomerRepo.AssertNotCalled(t, "Create")
	mockSubRepo.AssertNotCalled(t, "Create")
}

// TestCreateCustomerDatabaseFailureRollback - Teste de falha no banco com rollback
func TestCreateCustomerDatabaseFailureRollback(t *testing.T) {
	ctx := context.Background()

	mockPlanRepo := new(MockPlanRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	mockSubRepo := new(MockSubscriptionRepository)
	mockGateway := new(MockPaymentGateway)
	mockQueue := new(MockQueueProducer)
	mockEmailService := new(MockEmailService)
	mockEmailService.On("SendWelcome", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	plan := &entity.Plan{
		ID:         "plan-123",
		Name:       "Plano Premium",
		PriceCents: 29900,
		Provider:   "DOC24",
		ProductID:  "prod-123",
	}

	mockPlanRepo.On("FindByID", ctx, "plan-123").Return(plan, nil)
	mockGateway.On("CreateCustomer", mock.Anything).Return("asaas-cust-123", nil)
	mockGateway.On("SubscribePix", mock.Anything).Return("asaas-sub-456", &asaas.PixOutput{
		CopyPaste: "00020126580014br.gov.bcb.pix",
		URL:       "data:image/png",
	}, nil)

	// Customer criado com sucesso
	mockCustomerRepo.On("Create", ctx, mock.Anything).Return(nil)
	// MAS subscription falha
	mockSubRepo.On("Create", ctx, mock.Anything).Return(errors.New("database error"))
	// E o rollback do customer também é chamado
	mockCustomerRepo.On("Delete", ctx, mock.Anything).Return(nil)

	uc := usecase.NewCreateCustomerUseCase(
		mockCustomerRepo, mockSubRepo, mockPlanRepo,
		mockGateway, mockQueue, mockEmailService, nil,
		"https://storage.example.com",
	)

	input := usecase.CreateCustomerInput{
		Name:            "João Silva",
		Email:           "joao@example.com",
		CPF:             "123.456.789-00",
		Phone:           "(11) 99999-9999",
		BirthDate:       "1990-05-15",
		Gender:          "1",
		PlanID:          "plan-123",
		PaymentMethod:   "PIX",
		Street:          "Rua A",
		Number:          "123",
		District:        "Centro",
		City:            "São Paulo",
		State:           "SP",
		ZipCode:         "01310-100",
		OnixCode:        "7065",
		TermsAccepted:   true,
		TermsAcceptedAt: time.Now().Format(time.RFC3339),
		TermsVersion:    "1.0",
	}

	output, err := uc.Execute(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.True(t, usecase.IsTechnicalError(err))

	// Verifica que foi feito rollback do customer
	mockCustomerRepo.AssertCalled(t, "Delete", ctx, mock.Anything)
}
