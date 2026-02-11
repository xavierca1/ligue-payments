package handlers

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/xavierca1/ligue-payments/internal/entity"
	"github.com/xavierca1/ligue-payments/internal/usecase"
)

type WebhookHandler struct {
	CustomerRepo  entity.CustomerRepositoryInterface
	ActivateSubUC usecase.ActivateSubscriptionInterface
}

func NewWebhookHandler(
	customerRepo entity.CustomerRepositoryInterface,
	activateSubUC usecase.ActivateSubscriptionInterface,
) *WebhookHandler {
	return &WebhookHandler{
		CustomerRepo:  customerRepo,
		ActivateSubUC: activateSubUC,
	}
}

func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Webhook: Failed to read body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Verificar assinatura do webhook
	signature := r.Header.Get("X-Asaas-Signature")
	if !verifyWebhookSignature(string(body), signature) {
		log.Printf("❌ Webhook: Invalid signature")
		writeErrorResponse(w, http.StatusUnauthorized, "INVALID_SIGNATURE", "invalid_signature")
		return
	}

	var event struct {
		Event   string `json:"event"`
		Payment struct {
			ID       string `json:"id"`
			Customer string `json:"customer"`
		} `json:"payment"`
	}

	if err := json.Unmarshal(body, &event); err != nil {

		log.Printf("⚠️ Webhook: Invalid JSON: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Eventos que ativam a assinatura:
	// - PAYMENT_RECEIVED: PIX confirmado
	// - PAYMENT_CONFIRMED: PIX confirmado (alternativo)
	// - PAYMENT_APPROVED: Cartão de crédito aprovado
	validEvents := []string{"PAYMENT_RECEIVED", "PAYMENT_CONFIRMED", "PAYMENT_APPROVED"}
	isValid := false
	for _, validEvent := range validEvents {
		if event.Event == validEvent {
			isValid = true
			break
		}
	}

	if !isValid {
		log.Printf("ℹ️ Webhook: Evento ignorado: %s", event.Event)
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("📥 Webhook: Evento recebido: %s para customer %s", event.Event, event.Payment.Customer)

	localCustomer, err := h.CustomerRepo.FindByGatewayID(event.Payment.Customer)
	if err != nil {
		log.Printf("❌ Webhook: Customer not found (GatewayID: %s): %v", event.Payment.Customer, err)

		w.WriteHeader(http.StatusOK)
		return
	}

	input := usecase.ActivateSubscriptionInput{
		CustomerID: localCustomer.ID,
		GatewayID:  event.Payment.ID,
	}

	if err := h.ActivateSubUC.Execute(r.Context(), input); err != nil {
		log.Printf("❌ Webhook: Activation error: %v", err)

		writeErrorResponse(w, http.StatusInternalServerError, "ACTIVATION_ERROR", "Erro ao ativar assinatura")
		return
	}

	log.Printf("✅ Webhook: Subscription activated for customer %s", localCustomer.ID)
	w.WriteHeader(http.StatusOK)
}

func verifyWebhookSignature(body, signature string) bool {
	webhookSecret := os.Getenv("ASAAS_WEBHOOK_SECRET")
	if webhookSecret == "" {
		log.Println("⚠️ ASAAS_WEBHOOK_SECRET não configurado")
		return false
	}

	hash := sha256.Sum256([]byte(body + webhookSecret))
	expectedSig := fmt.Sprintf("%x", hash)

	return subtle.ConstantTimeCompare(
		[]byte(signature),
		[]byte(expectedSig),
	) == 1
}
