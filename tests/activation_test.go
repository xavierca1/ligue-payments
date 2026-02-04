package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xavierca1/ligue-payments/internal/entity"
	"github.com/xavierca1/ligue-payments/internal/infra/queue"
	"github.com/xavierca1/ligue-payments/internal/usecase"
)

// MockDoc24Client
type MockDoc24Client struct {
	mock.Mock
}

func (m *MockDoc24Client) CreateBeneficiary(ctx context.Context, input queue.ActivationPayload) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockDoc24Client) GetBeneficiaryID(cpf string) string {
	args := m.Called(cpf)
	return args.String(0)
}

// MockKommoService
type MockKommoService struct {
	mock.Mock
}

func (m *MockKommoService) CreateLead(customerName, phone, email, planName string, price int) (int, error) {
	args := m.Called(customerName, phone, email, planName, price)
	return args.Int(0), args.Error(1)
}

// TestActivateSubscriptionDoc24Success - Teste completo de ativação com Doc24
func TestActivateSubscriptionDoc24Success(t *testing.T) {
	ctx := context.Background()

	mockSubRepo := new(MockSubscriptionRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	mockPlanRepo := new(MockPlanRepository)
	mockQueue := new(MockQueueProducer)
	mockEmailService := new(MockEmailService)
	mockKommo := new(MockKommoService)
	mockKommo.On("CreateLead", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, nil)

	// Customer existente
	customer := &entity.Customer{
		ID:        "cust-123",
		Name:      "João Silva",
		Email:     "joao@example.com",
		CPF:       "123.456.789-00",
		Phone:     "(11) 99999-9999",
		BirthDate: "1990-05-15",
		Gender:    1,
		PlanID:    "plan-456",
		GatewayID: "asaas-cust-123",
	}

	// Subscription existente
	subscription := &entity.Subscription{
		ID:         "sub-123",
		CustomerID: "cust-123",
		PlanID:     "plan-456",
		ProductID:  "prod-456",
		Status:     "PENDING",
	}

	// Plan com provider
	plan := &entity.Plan{
		ID:               "plan-456",
		Name:             "Plano Premium",
		PriceCents:       29900,
		Provider:         "DOC24",
		ProductID:        "prod-456",
		ProviderPlanCode: "ligue saude em dia individual",
	}

	mockCustomerRepo.On("FindByID", ctx, "cust-123").Return(customer, nil)
	mockSubRepo.On("FindLastByCustomerID", ctx, "cust-123").Return(subscription, nil)
	mockPlanRepo.On("FindByID", ctx, "plan-456").Return(plan, nil)
	mockSubRepo.On("UpdateStatus", "cust-123", "ACTIVE").Return(nil)
	mockQueue.On("PublishActivation", ctx, mock.MatchedBy(func(p queue.ActivationPayload) bool {
		return p.CustomerID == "cust-123" &&
			p.Provider == "DOC24" &&
			p.Name == "João Silva" &&
			p.Email == "joao@example.com"
	})).Return(nil)

	uc := usecase.NewActivateSubscriptionUseCase(
		mockSubRepo, mockCustomerRepo, mockPlanRepo,
		mockQueue, mockEmailService, mockKommo,
	)

	input := usecase.ActivateSubscriptionInput{
		CustomerID: "cust-123",
		GatewayID:  "payment-123",
	}

	err := uc.Execute(ctx, input)

	assert.NoError(t, err)
	mockCustomerRepo.AssertCalled(t, "FindByID", ctx, "cust-123")
	mockSubRepo.AssertCalled(t, "FindLastByCustomerID", ctx, "cust-123")
	mockPlanRepo.AssertCalled(t, "FindByID", ctx, "plan-456")
	mockSubRepo.AssertCalled(t, "UpdateStatus", "cust-123", "ACTIVE")
	mockQueue.AssertCalled(t, "PublishActivation", ctx, mock.Anything)
}

// TestActivateSubscriptionDoc24EndToEnd - Teste end-to-end completo com Doc24
func TestActivateSubscriptionDoc24EndToEnd(t *testing.T) {
	ctx := context.Background()

	mockSubRepo := new(MockSubscriptionRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	mockPlanRepo := new(MockPlanRepository)
	mockQueue := new(MockQueueProducer)
	mockEmailService := new(MockEmailService)
	mockDoc24Client := new(MockDoc24Client)
	mockKommo := new(MockKommoService)
	mockKommo.On("CreateLead", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, nil)

	customer := &entity.Customer{
		ID:        "cust-123",
		Name:      "João Silva",
		Email:     "joao@example.com",
		CPF:       "123.456.789-00",
		Phone:     "(11) 99999-9999",
		BirthDate: "1990-05-15",
		Gender:    1,
		PlanID:    "plan-456",
		GatewayID: "asaas-cust-123",
	}

	subscription := &entity.Subscription{
		ID:         "sub-123",
		CustomerID: "cust-123",
		PlanID:     "plan-456",
		ProductID:  "prod-456",
		Status:     "PENDING",
	}

	plan := &entity.Plan{
		ID:               "plan-456",
		Name:             "Plano Premium",
		Provider:         "DOC24",
		ProductID:        "prod-456",
		ProviderPlanCode: "ligue saude em dia individual",
	}

	mockCustomerRepo.On("FindByID", ctx, "cust-123").Return(customer, nil)
	mockSubRepo.On("FindLastByCustomerID", ctx, "cust-123").Return(subscription, nil)
	mockPlanRepo.On("FindByID", ctx, "plan-456").Return(plan, nil)
	mockSubRepo.On("UpdateStatus", "cust-123", "ACTIVE").Return(nil)

	mockQueue.On("PublishActivation", ctx, mock.MatchedBy(func(p queue.ActivationPayload) bool {
		return p.Provider == "DOC24" && p.CustomerID == "cust-123"
	})).Return(nil)

	mockDoc24Client.On("CreateBeneficiary", ctx, mock.MatchedBy(func(p queue.ActivationPayload) bool {
		return p.CustomerID == "cust-123"
	})).Return(nil)

	// 1. Webhook ativa subscription
	activateUC := usecase.NewActivateSubscriptionUseCase(
		mockSubRepo, mockCustomerRepo, mockPlanRepo,
		mockQueue, mockEmailService, mockKommo,
	)

	input := usecase.ActivateSubscriptionInput{
		CustomerID: "cust-123",
		GatewayID:  "payment-123",
	}

	err := activateUC.Execute(ctx, input)
	assert.NoError(t, err)

	// 2. Verifica que a fila foi publicada
	mockQueue.AssertCalled(t, "PublishActivation", ctx, mock.Anything)

	// 3. Simula worker processando a mensagem
	payloadArg := mockQueue.Calls[0].Arguments[1].(queue.ActivationPayload)
	err = mockDoc24Client.CreateBeneficiary(ctx, payloadArg)
	assert.NoError(t, err)

	// 4. Verifica que Doc24 foi chamado
	mockDoc24Client.AssertCalled(t, "CreateBeneficiary", ctx, mock.Anything)

	// 5. Verifica que status foi atualizado
	mockSubRepo.AssertCalled(t, "UpdateStatus", "cust-123", "ACTIVE")
}
