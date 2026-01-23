package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/xavierca1/ligue-payments/internal/usecase"
)

type CheckoutHandler struct {
	CreateCustomerUC *usecase.CreateCustomerUseCase
}

func NewCheckoutHandler(uc *usecase.CreateCustomerUseCase) *CheckoutHandler {
	return &CheckoutHandler{CreateCustomerUC: uc}
}

func (h *CheckoutHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreateCustomerInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}

	output, err := h.CreateCustomerUC.Execute(r.Context(), input)
	if err != nil {
		// Dica: num projeto real, você logaria o erro aqui
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(output)
}
