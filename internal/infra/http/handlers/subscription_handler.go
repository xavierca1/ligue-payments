package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xavierca1/ligue-payments/internal/entity"
)

type SubscriptionHandler struct {
	SubRepo entity.SubscriptionRepository // Interface do repositório
}

func NewSubscriptionHandler(repo entity.SubscriptionRepository) *SubscriptionHandler {
	return &SubscriptionHandler{SubRepo: repo}
}

func (h *SubscriptionHandler) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
	customerID := chi.URLParam(r, "customerId")

	status, err := h.SubRepo.GetStatusByCustomerID(customerID)
	if err != nil {
		http.Error(w, "Status não encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": status})
}
