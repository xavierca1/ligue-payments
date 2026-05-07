package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/xavierca1/ligue-payments/internal/infra/integration/docuseal"
)

// DocuSealStatusHandler retorna o status de uma submission
type DocuSealStatusHandler struct {
	Client *docuseal.Client
}

func NewDocuSealStatusHandler(c *docuseal.Client) *DocuSealStatusHandler {
	return &DocuSealStatusHandler{Client: c}
}

type StatusRequest struct {
	UUID string `json:"uuid"`
}

func (h *DocuSealStatusHandler) Handle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	var req StatusRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	if req.UUID == "" {
		http.Error(w, "uuid is required", http.StatusBadRequest)
		return
	}

	submission, err := h.Client.GetSubmission(req.UUID)
	if err != nil {
		log.Printf("❌ Erro ao obter submission: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(submission)
}
