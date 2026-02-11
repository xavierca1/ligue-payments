package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xavierca1/ligue-payments/internal/entity"
	"github.com/xavierca1/ligue-payments/internal/infra/http/handlers"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/asaas"
	"github.com/xavierca1/ligue-payments/internal/usecase"
)

// MockSubscriptionRepositoryHandler
type MockSubscriptionRepositoryHandler struct {
	mock.Mock
}

func (m *MockSubscriptionRepositoryHandler) Create(ctx context.Context, sub *entity.Subscription) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}

func (m *MockSubscriptionRepositoryHandler) GetStatusByCustomerID(customerID string) (string, error) {
	args := m.Called(customerID)
	return args.String(0), args.Error(1)
}

func (m *MockSubscriptionRepositoryHandler) UpdateStatus(customerID string, status string) error {
	args := m.Called(customerID, status)
	return args.Error(0)
}

func (m *MockSubscriptionRepositoryHandler) FindLastByCustomerID(ctx context.Context, customerID string) (*entity.Subscription, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Subscription), args.Error(1)
}

// ============ TESTES DO HANDLER ============

// TestCreateCheckoutHandlerPixSuccess - Teste do checkout com PIX
func TestCreateCheckoutHandlerPixSuccess(t *testing.T) {
	mockPlanRepo := new(MockPlanRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	mockSubRepo := new(MockSubscriptionRepositoryHandler)
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

	mockPlanRepo.On("FindByID", mock.Anything, "plan-123").Return(plan, nil)
	mockGateway.On("CreateCustomer", mock.Anything).Return("asaas-cust-123", nil)
	mockGateway.On("SubscribePix", mock.Anything).Return("asaas-sub-456", &asaas.PixOutput{
		CopyPaste: "00020126580014br.gov.bcb.pix",
		URL:       "data:image/png;base64,iVBORw0KG...",
	}, nil)
	mockCustomerRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockSubRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	uc := usecase.NewCreateCustomerUseCase(
		mockCustomerRepo, mockSubRepo, mockPlanRepo,
		mockGateway, mockQueue, mockEmailService, nil,
		"https://storage.example.com",
		nil,
	)

	handler := handlers.NewCustomerHandler(uc, mockSubRepo)

	// Request body
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
		TermsAccepted:   true,
		TermsAcceptedAt: time.Now().Format(time.RFC3339),
		TermsVersion:    "1.0",
	}

	body, _ := json.Marshal(input)
	req := httptest.NewRequest("POST", "/checkout", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Execute
	handler.CreateCheckoutHandler(w, req)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)

	var response usecase.CreateCustomerOutput
	json.NewDecoder(w.Body).Decode(&response)

	assert.Equal(t, "WAITING_PAYMENT", response.Status)
	assert.NotEmpty(t, response.ID)
	assert.NotEmpty(t, response.PixCode)
	assert.NotEmpty(t, response.PixQRCodeURL)
}

// TestCreateCheckoutHandlerInvalidJSON - Teste com JSON inválido
func TestCreateCheckoutHandlerInvalidJSON(t *testing.T) {
	mockSubRepo := new(MockSubscriptionRepositoryHandler)
	uc := usecase.NewCreateCustomerUseCase(nil, nil, nil, nil, nil, nil, nil, "", nil)
	handler := handlers.NewCustomerHandler(uc, mockSubRepo)

	req := httptest.NewRequest("POST", "/checkout", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	handler.CreateCheckoutHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResponse map[string]string
	json.NewDecoder(w.Body).Decode(&errResponse)
	assert.Equal(t, "INVALID_JSON", errResponse["error"])
}

// TestCreateCheckoutHandlerValidationError - Teste com erro de validação
func TestCreateCheckoutHandlerValidationError(t *testing.T) {
	mockSubRepo := new(MockSubscriptionRepositoryHandler)

	mockPlanRepo := new(MockPlanRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	mockGateway := new(MockPaymentGateway)
	mockQueue := new(MockQueueProducer)
	mockEmailService := new(MockEmailService)
	mockEmailService.On("SendWelcome", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	uc := usecase.NewCreateCustomerUseCase(
		mockCustomerRepo, mockSubRepo, mockPlanRepo,
		mockGateway, mockQueue, mockEmailService, nil,
		"https://storage.example.com",
		nil,
	)

	handler := handlers.NewCustomerHandler(uc, mockSubRepo)

	// Input com email inválido
	input := usecase.CreateCustomerInput{
		Name:            "João",
		Email:           "invalid-email", // Email inválido!
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
		TermsAccepted:   true,
		TermsAcceptedAt: time.Now().Format(time.RFC3339),
		TermsVersion:    "1.0",
	}

	body, _ := json.Marshal(input)
	req := httptest.NewRequest("POST", "/checkout", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.CreateCheckoutHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResponse map[string]string
	json.NewDecoder(w.Body).Decode(&errResponse)
	assert.Equal(t, "VALIDATION_ERROR", errResponse["error"])
}

// TestGetStatusHandlerSuccess - Teste do status com subscription existente
func TestGetStatusHandlerSuccess(t *testing.T) {
	mockSubRepo := new(MockSubscriptionRepositoryHandler)
	mockSubRepo.On("GetStatusByCustomerID", "cust-123").Return("ACTIVE", nil)

	mockPlanRepo := new(MockPlanRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	mockGateway := new(MockPaymentGateway)
	mockQueue := new(MockQueueProducer)
	mockEmailService := new(MockEmailService)
	mockEmailService.On("SendWelcome", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	uc := usecase.NewCreateCustomerUseCase(
		mockCustomerRepo, mockSubRepo, mockPlanRepo,
		mockGateway, mockQueue, mockEmailService, nil,
		"https://storage.example.com",
		nil,
	)

	handler := handlers.NewCustomerHandler(uc, mockSubRepo)

	req := httptest.NewRequest("GET", "/customers/cust-123/status", nil)

	// Simular chi routing
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "cust-123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	w := httptest.NewRecorder()

	handler.GetStatusHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "ACTIVE", response["status"])
}

// TestGetStatusHandlerMissingID - Teste sem ID de customer
func TestGetStatusHandlerMissingID(t *testing.T) {
	mockSubRepo := new(MockSubscriptionRepositoryHandler)

	mockPlanRepo := new(MockPlanRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	mockGateway := new(MockPaymentGateway)
	mockQueue := new(MockQueueProducer)
	mockEmailService := new(MockEmailService)
	mockEmailService.On("SendWelcome", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	uc := usecase.NewCreateCustomerUseCase(
		mockCustomerRepo, mockSubRepo, mockPlanRepo,
		mockGateway, mockQueue, mockEmailService, nil,
		"https://storage.example.com",
		nil,
	)

	handler := handlers.NewCustomerHandler(uc, mockSubRepo)

	req := httptest.NewRequest("GET", "/customers//status", nil)

	// Sem ID
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	w := httptest.NewRecorder()

	handler.GetStatusHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResponse map[string]string
	json.NewDecoder(w.Body).Decode(&errResponse)
	assert.Equal(t, "MISSING_ID", errResponse["error"])
}

// TestGetStatusHandlerNotFound - Teste quando não há subscription
func TestGetStatusHandlerNotFound(t *testing.T) {
	mockSubRepo := new(MockSubscriptionRepositoryHandler)
	// Retorna erro (não encontrado)
	mockSubRepo.On("GetStatusByCustomerID", "cust-999").Return("", errors.New("not found"))

	mockPlanRepo := new(MockPlanRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	mockGateway := new(MockPaymentGateway)
	mockQueue := new(MockQueueProducer)
	mockEmailService := new(MockEmailService)
	mockEmailService.On("SendWelcome", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	uc := usecase.NewCreateCustomerUseCase(
		mockCustomerRepo, mockSubRepo, mockPlanRepo,
		mockGateway, mockQueue, mockEmailService, nil,
		"https://storage.example.com",
		nil,
	)

	handler := handlers.NewCustomerHandler(uc, mockSubRepo)

	req := httptest.NewRequest("GET", "/customers/cust-999/status", nil)

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "cust-999")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	w := httptest.NewRecorder()

	handler.GetStatusHandler(w, req)

	// Retorna PENDING mesmo com erro
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "PENDING", response["status"])
}
