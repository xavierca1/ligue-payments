package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/xavierca1/ligue-payments/internal/entity"
	"github.com/xavierca1/ligue-payments/internal/usecase"
)

type WebhookHandler struct {
	CustomerRepo  entity.CustomerRepositoryInterface
	ActivateSubUC *usecase.ActivateSubscriptionUseCase
}

func NewWebhookHandler(
	customerRepo entity.CustomerRepositoryInterface,
	activateSubUC *usecase.ActivateSubscriptionUseCase,
) *WebhookHandler {
	return &WebhookHandler{
		CustomerRepo:  customerRepo,
		ActivateSubUC: activateSubUC,
	}
}

func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var event struct {
		Event   string `json:"event"`
		Payment struct {
			ID       string `json:"id"`
			Customer string `json:"customer"`
		} `json:"payment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Bad JSON", 400)
		return
	}

	if event.Event != "PAYMENT_RECEIVED" && event.Event != "PAYMENT_CONFIRMED" {
		w.WriteHeader(200)
		return
	}

	localCustomer, err := h.CustomerRepo.FindByGatewayID(event.Payment.Customer)
	if err != nil {
		log.Printf("❌ Webhook: Cliente não encontrado (GatewayID: %s): %v", event.Payment.Customer, err)
		w.WriteHeader(200) // 200 pro Asaas parar de tentar
		return
	}

	input := usecase.ActivateSubscriptionInput{
		CustomerID: localCustomer.ID,
		GatewayID:  event.Payment.ID,
	}

	if err := h.ActivateSubUC.Execute(r.Context(), input); err != nil {
		log.Printf(" Erro na ativação: %v", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(200)
}
