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

	signature := r.Header.Get("X-Asaas-Signature")
	if signature == "" {
		log.Println("❌ Webhook: Missing X-Asaas-Signature header")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid_signature"}`))
		return
	}


	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Webhook: Failed to read body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}


	if !verifyWebhookSignature(string(body), signature) {
		log.Println("❌ Webhook: Invalid signature - possible forgery!")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid_signature"}`))
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


	if event.Event != "PAYMENT_RECEIVED" && event.Event != "PAYMENT_CONFIRMED" {
		w.WriteHeader(http.StatusOK)
		return
	}

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
