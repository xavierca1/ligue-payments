package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xavierca1/ligue-payments/internal/entity"
	"github.com/xavierca1/ligue-payments/internal/usecase"
)

type CustomerHandler struct {
	CreateCustomerUC *usecase.CreateCustomerUseCase
	SubRepo          entity.SubscriptionRepository
}

func NewCustomerHandler(uc *usecase.CreateCustomerUseCase, subRepo entity.SubscriptionRepository) *CustomerHandler {
	return &CustomerHandler{
		CreateCustomerUC: uc,
		SubRepo:          subRepo,
	}
}

func (h *CustomerHandler) CreateCheckoutHandler(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreateCustomerInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "JSON inválido: "+err.Error())
		return
	}

	output, err := h.CreateCustomerUC.Execute(r.Context(), input)
	if err != nil {

		if usecase.IsDomainError(err) {
			writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}

		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Erro ao processar cadastro")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(output)
}


func (h *CustomerHandler) GetStatusHandler(w http.ResponseWriter, r *http.Request) {
	customerID := chi.URLParam(r, "id")
	if customerID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_ID", "Customer ID is required")
		return
	}


	status, err := h.SubRepo.GetStatusByCustomerID(customerID)
	if err != nil {

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "PENDING"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": status})
}


func writeErrorResponse(w http.ResponseWriter, statusCode int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   code,
		"message": message,
	})
}
