package main

import (
	"encoding/json"
	"net/http"

	"github.com/xavierca1/ligue-payments/internal/usecase"
)

type WebHandler struct {
	CreateCustomerUseCase *usecase.CreateCustomerUseCase
}

func NewWebHandler(uc *usecase.CreateCustomerUseCase) *WebHandler {
	return &WebHandler{CreateCustomerUseCase: uc}
}

func (h *WebHandler) HandleCreateCustomer(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreateCustomerInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	output, err := h.CreateCustomerUseCase.Execute(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(output)
}
