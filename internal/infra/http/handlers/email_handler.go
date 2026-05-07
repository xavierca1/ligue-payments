package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/xavierca1/ligue-payments/internal/usecase"
)

type EmailHandler struct {
	EmailService usecase.EmailService
}

func NewEmailHandler(emailService usecase.EmailService) *EmailHandler {
	return &EmailHandler{EmailService: emailService}
}

type sendTestEmailInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (h *EmailHandler) SendTestWelcomeEmail(w http.ResponseWriter, r *http.Request) {
	if h.EmailService == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "EMAIL_NOT_CONFIGURED", "Serviço de e-mail não configurado")
		return
	}

	var input sendTestEmailInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "JSON inválido")
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)

	if input.Name == "" || input.Email == "" {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_INPUT", "Informe name e email")
		return
	}

	if err := h.EmailService.SendWelcomeEmail(input.Name, input.Email); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "EMAIL_SEND_FAILED", "Falha ao enviar e-mail de teste")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "E-mail de teste enviado com sucesso",
	})
}
