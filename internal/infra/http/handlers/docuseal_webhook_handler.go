package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/xavierca1/ligue-payments/internal/infra/integration/docuseal"
	"github.com/xavierca1/ligue-payments/internal/usecase"
)

// DocuSealWebhookHandler processa webhooks enviados pelo DocuSeal
type DocuSealWebhookHandler struct {
	DocuSealClient *docuseal.Client
	EmailService   usecase.EmailService
}

func NewDocuSealWebhookHandler(client *docuseal.Client, emailService usecase.EmailService) *DocuSealWebhookHandler {
	return &DocuSealWebhookHandler{DocuSealClient: client, EmailService: emailService}
}

func (h *DocuSealWebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ DocuSeal webhook: erro ao ler body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var event map[string]interface{}
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf("⚠️ DocuSeal webhook: payload inválido JSON: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Buscar possíveis chaves que representem o identificador da submission
	var submissionUUID string
	if v, ok := event["submission_uuid"].(string); ok && v != "" {
		submissionUUID = v
	} else if v, ok := event["uuid"].(string); ok && v != "" {
		submissionUUID = v
	} else if v, ok := event["id"].(string); ok && v != "" {
		submissionUUID = v
	}

	if submissionUUID == "" {
		log.Printf("⚠️ DocuSeal webhook: submission UUID não encontrado no payload")
		w.WriteHeader(http.StatusOK)
		return
	}

	submission, err := h.DocuSealClient.GetSubmission(submissionUUID)
	if err != nil {
		log.Printf("❌ DocuSeal webhook: falha ao obter submission %s: %v", submissionUUID, err)
		w.WriteHeader(http.StatusOK)
		return
	}

	status := strings.ToUpper(strings.TrimSpace(submission.Status))
	if status != "SIGNED" && status != "COMPLETED" {
		log.Printf("ℹ️ DocuSeal webhook: submission %s status=%s (ignorado)", submissionUUID, submission.Status)
		w.WriteHeader(http.StatusOK)
		return
	}

	if submission.DocumentURL == "" {
		log.Printf("⚠️ DocuSeal webhook: submission %s não contém document_url", submissionUUID)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Baixar PDF assinado
	resp, err := http.Get(submission.DocumentURL)
	if err != nil {
		log.Printf("❌ DocuSeal webhook: erro ao baixar documento assinado %s: %v", submission.DocumentURL, err)
		w.WriteHeader(http.StatusOK)
		return
	}
	defer resp.Body.Close()

	pdfBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ DocuSeal webhook: erro ao ler documento assinado: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Obter email do signatário
	signerEmail := ""
	signerName := ""
	if len(submission.Signers) > 0 {
		signerEmail = submission.Signers[0].Email
		signerName = submission.Signers[0].FullName
	}

	if signerEmail == "" {
		// Tentar extrair do payload do webhook
		if v, ok := event["signer_email"].(string); ok {
			signerEmail = v
		}
	}

	if signerEmail == "" {
		log.Printf("⚠️ DocuSeal webhook: não foi possível determinar email do signatário para submission %s", submissionUUID)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Enviar email com o PDF assinado anexo
	if h.EmailService != nil {
		if err := h.EmailService.SendWelcomeEmailWithContractAndDependents(signerName, signerEmail, "", "", "", nil, pdfBytes); err != nil {
			log.Printf("❌ DocuSeal webhook: falha ao enviar email com PDF assinado para %s: %v", signerEmail, err)
		} else {
			log.Printf("✅ DocuSeal webhook: email com PDF assinado enviado para %s (submission=%s)", signerEmail, submissionUUID)
		}
	}

	w.WriteHeader(http.StatusOK)
}
