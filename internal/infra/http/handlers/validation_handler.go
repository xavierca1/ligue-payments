package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/xavierca1/ligue-payments/internal/entity"
)

type ValidationHandler struct {
	Repo entity.CustomerRepositoryInterface
}

func NewValidationHandler(repo entity.CustomerRepositoryInterface) *ValidationHandler {
	return &ValidationHandler{Repo: repo}
}

func (h *ValidationHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
		CPF   string `json:"cpf"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "JSON inválido")
		return
	}

	if input.Email == "" || input.CPF == "" {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_FIELDS", "email and cpf are required")
		return
	}

	exists, err := h.Repo.CheckDuplicity(r.Context(), input.Email, input.CPF)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "DATABASE_ERROR", "Erro ao validar duplicidade")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if exists {

		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "user_exists",
			"message": "Um usuário com este email ou CPF já existe",
		})
		return
	}


	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
