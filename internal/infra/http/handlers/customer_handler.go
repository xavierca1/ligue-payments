package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xavierca1/ligue-payments/internal/infra/database" // Importar o pacote database concreto ou a interface
	"github.com/xavierca1/ligue-payments/internal/usecase"
)

type CustomerHandler struct {
	CreateCustomerUC *usecase.CreateCustomerUseCase
	SubRepo          *database.SubscriptionRepository
}

func NewCustomerHandler(uc *usecase.CreateCustomerUseCase, subRepo *database.SubscriptionRepository) *CustomerHandler {
	return &CustomerHandler{
		CreateCustomerUC: uc,
		SubRepo:          subRepo,
	}
}

func (h *CustomerHandler) CreateCheckoutHandler(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreateCustomerInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}

	output, err := h.CreateCustomerUC.Execute(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(output)
}

// GetStatusHandler (GET /customers/{id}/status)
// AGORA SIM: Consulta a tabela subscriptions
func (h *CustomerHandler) GetStatusHandler(w http.ResponseWriter, r *http.Request) {
	customerID := chi.URLParam(r, "id")
	if customerID == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	// Busca o status da ÚLTIMA assinatura desse cliente
	status, err := h.SubRepo.GetStatusByCustomerID(customerID)
	if err != nil {
		// Se não achou assinatura ou deu erro, retorna PENDING
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "PENDING"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": status})
}
