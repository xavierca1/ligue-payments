package handlers

import (
	"context"
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

	// Asaas envia o access token no header "Asaas-Access-Token"
	signature := r.Header.Get("Asaas-Access-Token")
	if !verifyWebhookSignature(string(body), signature, r) {
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

	eventName := strings.ToUpper(strings.TrimSpace(event.Event))
	paymentStatus := strings.ToUpper(strings.TrimSpace(event.Payment.Status))

	// Eventos de cobrança de multa/juros não devem reativar assinatura.
	isNonActivationEvent := eventName == "PAYMENT_FINE_CHARGED" ||
		eventName == "PAYMENT_INTEREST_CHARGED" ||
		eventName == "PAYMENT_PENALTY_CHARGED"

	isPaymentEvent := strings.HasPrefix(eventName, "PAYMENT_")
	isPaidStatus := paymentStatus == "RECEIVED" || paymentStatus == "CONFIRMED" || paymentStatus == "RECEIVED_IN_CASH"
	isActivationEvent := eventName == "PAYMENT_RECEIVED" || eventName == "PAYMENT_CONFIRMED" || eventName == "PAYMENT_APPROVED"
	shouldActivate := !isNonActivationEvent && ((isPaymentEvent && isPaidStatus) || isActivationEvent)

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

	customerRef := strings.TrimSpace(event.Payment.Customer)
	localCustomer, err := h.CustomerRepo.FindByGatewayID(customerRef)
	if err != nil {
		log.Printf("⚠️ Webhook: FindByGatewayID falhou para %q, tentando FindByID como fallback: %v", customerRef, err)
		localCustomer, err = h.CustomerRepo.FindByID(r.Context(), customerRef)
		if err != nil {
			log.Printf("❌ Webhook: Customer not found por GatewayID nem por ID (%s): %v", customerRef, err)
			w.WriteHeader(http.StatusOK)
			return
		}
		log.Printf("✅ Webhook: Customer encontrado via fallback FindByID (%s)", customerRef)
	}

	// Responde 200 imediatamente para evitar timeout do Asaas.
	// A ativação é processada em background para não bloquear a resposta.
	w.WriteHeader(http.StatusOK)

	input := usecase.ActivateSubscriptionInput{
		CustomerID: localCustomer.ID,
		GatewayID:  event.Payment.ID,
	}
	customerID := localCustomer.ID
	gatewayID := localCustomer.GatewayID

	go func() {
		ctx := context.Background()
		if err := h.ActivateSubUC.Execute(ctx, input); err != nil {
			log.Printf("❌ Webhook: Activation error: %v", err)
			log.Printf("❌ Webhook: Detalhes - CustomerID=%s, GatewayID=%s, PaymentID=%s", customerID, gatewayID, input.GatewayID)
			return
		}
		log.Printf("✅ Webhook: Subscription activated for customer %s (GatewayID=%s, PaymentID=%s)", customerID, gatewayID, input.GatewayID)
	}()
}

func verifyWebhookSignature(body, signature string, r *http.Request) bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("ASAAS_WEBHOOK_SKIP_SIGNATURE")), "true") {
		log.Println("⚠️ Webhook signature validation bypassed via ASAAS_WEBHOOK_SKIP_SIGNATURE=true")
		return true
	}

	webhookSecret := os.Getenv("ASAAS_WEBHOOK_SECRET")
	if webhookSecret == "" {
		log.Println("❌ Webhook: ASAAS_WEBHOOK_SECRET não configurado")
		return false
	}

	_ = body

	if strings.TrimSpace(signature) == "" {
		log.Printf("❌ Webhook: header Asaas-Access-Token ausente ou vazio. Headers recebidos: %s", webhookHeaderDiag(r))
		return false
	}

	match := subtle.ConstantTimeCompare(
		[]byte(strings.TrimSpace(signature)),
		[]byte(strings.TrimSpace(webhookSecret)),
	) == 1

	if !match {
		// Com ASAAS_WEBHOOK_DEBUG=true, loga o token completo para facilitar o diagnóstico.
		// Desative após ajustar o ASAAS_WEBHOOK_SECRET no .env.
		if strings.EqualFold(strings.TrimSpace(os.Getenv("ASAAS_WEBHOOK_DEBUG")), "true") {
			log.Printf("❌ Webhook: token mismatch — recebido: %q | esperado: %q",
				strings.TrimSpace(signature), strings.TrimSpace(webhookSecret))
		} else {
			log.Printf("❌ Webhook: token mismatch — recebido len=%d prefix=%q | esperado len=%d prefix=%q",
				len(strings.TrimSpace(signature)), maskSecret(signature),
				len(strings.TrimSpace(webhookSecret)), maskSecret(webhookSecret),
			)
		}
	}
	return match
}

// webhookHeaderDiag retorna nomes e comprimentos dos headers recebidos (sem valores) para diagnóstico.
func webhookHeaderDiag(r *http.Request) string {
	var parts []string
	for name, vals := range r.Header {
		for _, v := range vals {
			parts = append(parts, fmt.Sprintf("%s(len=%d)", name, len(v)))
		}
	}
	return strings.Join(parts, ", ")
}

// maskSecret exibe apenas os 4 primeiros caracteres seguidos de ***.
func maskSecret(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 4 {
		return "***"
	}
	return s[:4] + "***"
}
