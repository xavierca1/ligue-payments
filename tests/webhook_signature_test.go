package tests

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xavierca1/ligue-payments/internal/entity"
	"github.com/xavierca1/ligue-payments/internal/infra/http/handlers"
	"github.com/xavierca1/ligue-payments/internal/usecase"
)

// MockActivateSubscriptionUseCase - Mock para ActivateSubscriptionUseCase
type MockActivateSubscriptionUseCase struct {
	mock.Mock
}

func (m *MockActivateSubscriptionUseCase) Execute(ctx context.Context, input usecase.ActivateSubscriptionInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

// TestWebhookSignatureVerification - Testa validação de assinatura Asaas
func TestWebhookSignatureVerification(t *testing.T) {
	webhookSecret := "test-webhook-secret"
	os.Setenv("ASAAS_WEBHOOK_SECRET", webhookSecret)
	defer os.Unsetenv("ASAAS_WEBHOOK_SECRET")

	mockCustomerRepo := new(MockCustomerRepository)
	mockActivateSubUC := new(MockActivateSubscriptionUseCase)

	handler := handlers.NewWebhookHandler(
		mockCustomerRepo,
		mockActivateSubUC,
	)

	t.Run("Valid Signature", func(t *testing.T) {
		payload := map[string]interface{}{
			"event": "PAYMENT_RECEIVED",
			"payment": map[string]string{
				"id":       "pay-123",
				"customer": "asaas-cust-456",
			},
		}

		body, _ := json.Marshal(payload)
		bodyStr := string(body)

		// Calcular assinatura correta
		hash := sha256.Sum256([]byte(bodyStr + webhookSecret))
		signature := fmt.Sprintf("%x", hash)

		// Mock customer
		mockCustomerRepo.On("FindByGatewayID", "asaas-cust-456").Return(
			&entity.Customer{ID: "cust-123", Name: "Test"},
			nil,
		)

		// Mock activation
		mockActivateSubUC.On("Execute", mock.Anything, mock.Anything).Return(nil)

		req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
		req.Header.Set("X-Asaas-Signature", signature)
		w := httptest.NewRecorder()

		handler.Handle(w, req)

		// Deve aceitar (não retorna 401)
		assert.NotEqual(t, http.StatusUnauthorized, w.Code,
			"Webhook com assinatura válida deveria ser aceito")
	})

	t.Run("Invalid Signature", func(t *testing.T) {
		payload := map[string]interface{}{
			"event": "PAYMENT_RECEIVED",
			"payment": map[string]string{
				"customer": "",
			},
		}

		body, _ := json.Marshal(payload)

		// Adicionar mock para FindByGatewayID com string vazia
		mockCustomerRepo.On("FindByGatewayID", "").Return(nil, errors.New("not found"))

		req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
		req.Header.Set("X-Asaas-Signature", "invalid-abc123")
		w := httptest.NewRecorder()

		handler.Handle(w, req)

		// Deve rejeitar
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid_signature")
	})

	t.Run("Missing Signature Header", func(t *testing.T) {
		payload := map[string]interface{}{
			"event": "PAYMENT_RECEIVED",
		}

		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.Handle(w, req)

		// Deve rejeitar
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Tampered Body", func(t *testing.T) {
		originalPayload := map[string]interface{}{
			"event": "PAYMENT_RECEIVED",
			"payment": map[string]string{
				"id":       "pay-123",
				"customer": "asaas-cust-456",
			},
		}

		originalBody, _ := json.Marshal(originalPayload)

		// Assinatura do original
		hash := sha256.Sum256([]byte(string(originalBody) + webhookSecret))
		signature := fmt.Sprintf("%x", hash)

		// Enviar body diferente
		tamperedPayload := map[string]interface{}{
			"event": "PAYMENT_RECEIVED",
			"payment": map[string]string{
				"id":       "pay-999",
				"customer": "asaas-cust-456",
			},
		}
		tamperedBody, _ := json.Marshal(tamperedPayload)

		req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(tamperedBody))
		req.Header.Set("X-Asaas-Signature", signature)
		w := httptest.NewRecorder()

		handler.Handle(w, req)

		// Deve rejeitar (body foi modificado)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
