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
	"strings"

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
			Status   string `json:"status"`
		} `json:"payment"`
	}

	if err := json.Unmarshal(body, &event); err != nil {

		log.Printf("⚠️ Webhook: Invalid JSON: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Eventos que ativam a assinatura:
	// - Qualquer PAYMENT_* com status de pagamento confirmado
	//   (alguns tenants enviam PAYMENT_UPDATED/PAYMENT_CREATED com status final)
	eventName := strings.ToUpper(strings.TrimSpace(event.Event))
	paymentStatus := strings.ToUpper(strings.TrimSpace(event.Payment.Status))

	isPaymentEvent := strings.HasPrefix(eventName, "PAYMENT_")
	isPaidStatus := paymentStatus == "RECEIVED" || paymentStatus == "CONFIRMED" || paymentStatus == "RECEIVED_IN_CASH"
	isActivationEvent := eventName == "PAYMENT_RECEIVED" || eventName == "PAYMENT_CONFIRMED" || eventName == "PAYMENT_APPROVED"
	shouldActivate := (isPaymentEvent && isPaidStatus) || isActivationEvent

	if !shouldActivate {
		if isPaymentEvent {
			log.Printf("ℹ️ Webhook: Evento ignorado: %s (status=%s, payment_id=%s, customer=%s)", eventName, paymentStatus, strings.TrimSpace(event.Payment.ID), strings.TrimSpace(event.Payment.Customer))
		} else {
			log.Printf("ℹ️ Webhook: Evento ignorado: %s", eventName)
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("📥 Webhook: Evento de ativação: %s (status=%s, payment_id=%s, customer=%s)", eventName, paymentStatus, strings.TrimSpace(event.Payment.ID), strings.TrimSpace(event.Payment.Customer))

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
		log.Printf("❌ Webhook: Detalhes - CustomerID=%s, GatewayID=%s, PaymentID=%s", localCustomer.ID, localCustomer.GatewayID, event.Payment.ID)

		writeErrorResponse(w, http.StatusInternalServerError, "ACTIVATION_ERROR", "Erro ao ativar assinatura")
		return
	}

	log.Printf("✅ Webhook: Subscription activated for customer %s (GatewayID=%s, PaymentID=%s)", localCustomer.ID, localCustomer.GatewayID, event.Payment.ID)
	w.WriteHeader(http.StatusOK)
}

func verifyWebhookSignature(body, signature string) bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("ASAAS_WEBHOOK_SKIP_SIGNATURE")), "true") {
		log.Println("⚠️ Webhook signature validation bypassed via ASAAS_WEBHOOK_SKIP_SIGNATURE=true")
		return true
	}

	webhookSecret := os.Getenv("ASAAS_WEBHOOK_SECRET")
	if webhookSecret == "" {
		log.Println("⚠️ ASAAS_WEBHOOK_SECRET não configurado")
		return false
	}

	hash := sha256.Sum256([]byte(body + webhookSecret))
	expectedSig := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%x", hash)))
	providedSig := strings.ToLower(strings.TrimSpace(signature))

	return subtle.ConstantTimeCompare(
		[]byte(providedSig),
		[]byte(expectedSig),
	) == 1
}
