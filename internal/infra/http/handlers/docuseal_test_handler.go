package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/xavierca1/ligue-payments/internal/infra/integration/docuseal"
)

// DocuSealTestHandler permite criar uma submission de teste rapidamente
type DocuSealTestHandler struct {
	Client *docuseal.Client
}

func NewDocuSealTestHandler(c *docuseal.Client) *DocuSealTestHandler {
	return &DocuSealTestHandler{Client: c}
}

// TestRequest é o payload aceito pelo endpoint de teste
type TestRequest struct {
	Email    string            `json:"email"`
	Template string            `json:"template,omitempty"` // Nome do template (padrão: ligue_saude_em_dia)
	Fields   map[string]string `json:"fields,omitempty"`
}

type TestResponse struct {
	SigningURL string `json:"signing_url,omitempty"`
	UUID       string `json:"uuid,omitempty"`
	Error      string `json:"error,omitempty"`
}

func (h *DocuSealTestHandler) Handle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	var req TestRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}

	// Define template padrão se não especificado
	templateName := req.Template
	if templateName == "" {
		templateName = "ligue_saude_em_dia"
	}

	// Valida template
	templateID, exists := docuseal.GetTemplateID(templateName)
	if !exists {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(TestResponse{Error: fmt.Sprintf("template inválido: %s", templateName)})
		return
	}

	fieldValues := map[string]string{
		"product":        req.Fields["product"],
		"id":             req.Fields["id"],
		"method_payment": req.Fields["method_payment"],
		"periodicidade":  req.Fields["periodicidade"],
		"name":           req.Fields["name"],
		"birthdate":      req.Fields["birthdate"],
		"cpf":            req.Fields["cpf"],
		"genre":          req.Fields["genre"],
		"marital_status": req.Fields["marital_status"],
		"cellphone":      req.Fields["cellphone"],
		"email":          req.Fields["email"],
		"address":        req.Fields["address"],
		"number":         req.Fields["number"],
		"neighborhood":   req.Fields["neighborhood"],
		"city":           req.Fields["city"],
		"UF":             req.Fields["UF"],
		"zip_code":       req.Fields["zip_code"],
	}
	fieldValues = normalizeDocuSealMonthly(fieldValues)

	submissionReq := &docuseal.CreateSubmissionRequest{
		TemplateID: templateID,
		SendEmail:  true,
		Submitters: []docuseal.SignerAttribute{{
			Email:     req.Email,
			FullName:  req.Fields["name"],
			Role:      "Proponente",
			Completed: true,
			Values:    fieldValues,
		}},
		CustomEmail: &docuseal.CustomEmailAttribute{
			Subject:  "Seu contrato de saúde - Ligue Saúde em Dia",
			Body:     fmt.Sprintf("Olá %s,\n\nSeu contrato de adesão está pronto para revisão.\n\nPor favor, acesse o link abaixo para visualizar e assinar seu documento:\n\nhttps://docuseal.com/submissions/[SUBMISSION_UUID]\n\nAtenciosamente,\nLigue Saúde em Dia", req.Fields["name"]),
			FromName: "Ligue Saúde em Dia",
		},
	}

	respObj, err := h.Client.CreateSubmission(submissionReq)
	if err != nil {
		log.Printf("docuseal test create submission error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(TestResponse{Error: fmt.Sprintf("create submission failed: %v", err)})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(TestResponse{SigningURL: respObj.SigningURL, UUID: respObj.UUID})
}

func normalizeDocuSealMonthly(values map[string]string) map[string]string {
	normalized := make(map[string]string, len(values))
	for key, value := range values {
		trimmed := strings.TrimSpace(value)
		if strings.EqualFold(trimmed, "monthly") || strings.EqualFold(trimmed, "mensal") {
			trimmed = "Mensal"
		}
		normalized[key] = trimmed
	}
	return normalized
}
