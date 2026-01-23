package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/xavierca1/ligue-payments/internal/entity"
	"github.com/xavierca1/ligue-payments/internal/infra/queue"
)

type WebhookHandler struct {
	CustomerRepo entity.CustomerRepositoryInterface
	SubRepo      entity.SubscriptionRepository
	PlanRepo     entity.PlanRepositoryInterface
	Producer     queue.QueueProducerInterface
}

func NewWebhookHandler(
	customerRepo entity.CustomerRepositoryInterface,
	subRepo entity.SubscriptionRepository,
	planRepo entity.PlanRepositoryInterface,
	producer queue.QueueProducerInterface,
) *WebhookHandler {
	return &WebhookHandler{
		CustomerRepo: customerRepo,
		SubRepo:      subRepo,
		PlanRepo:     planRepo,
		Producer:     producer,
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
		log.Printf("❌ Cliente não encontrado: %v", err)
		w.WriteHeader(200)
		return
	}

	if err := h.SubRepo.UpdateStatus(localCustomer.ID, "ACTIVE"); err != nil {
		log.Printf(" Erro ao atualizar status: %v", err)
	} else {
		log.Printf(" Assinatura ativada para %s", localCustomer.Name)
	}

	plan, _ := h.PlanRepo.FindByID(r.Context(), localCustomer.PlanID)
	provider := "DOC24"
	if plan != nil {
		provider = plan.Provider
	}

	payload := queue.ActivationPayload{
		CustomerID: localCustomer.ID,
		PlanID:     localCustomer.PlanID,
		Provider:   provider,
		Name:       localCustomer.Name,
		Email:      localCustomer.Email,
		Origin:     "WEBHOOK_ASAAS",
	}

	if err := h.Producer.PublishActivation(r.Context(), payload); err != nil {
		log.Printf(" Erro fila: %v", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(200)
}
