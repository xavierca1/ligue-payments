package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/xavierca1/ligue-payments/internal/infra/database"
)

type ValidationHandler struct {
	Repo *database.CustomerRepository
}

func NewValidationHandler(repo *database.CustomerRepository) *ValidationHandler {
	return &ValidationHandler{Repo: repo}
}

func (h *ValidationHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
		CPF   string `json:"cpf"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	exists, err := h.Repo.CheckDuplicity(r.Context(), input.Email, input.CPF)
	if err != nil {
		http.Error(w, "Erro ao validar duplicidade", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if exists {
		// Retorna 409 Conflict se já existir
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "user_exists"})
		return
	}

	// Retorna 200 OK se estiver livre
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
