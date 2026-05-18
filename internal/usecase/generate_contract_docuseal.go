package usecase

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/xavierca1/ligue-payments/internal/infra/integration/docuseal"
)

// GenerateContractWithDocuSealUseCase prepara um contrato para assinatura digital via DocuSeal
// Fluxo:
// 1. Gera o PDF preenchido com os dados do cliente
// 2. Envia para DocuSeal como template
// 3. Cria um submission para o cliente assinar
// 4. Retorna a URL de assinatura
type GenerateContractWithDocuSealUseCase struct {
	PdfGenerator   ContractPDFGeneratorInterface
	DocuSealClient *docuseal.Client
}

// DocuSealContractInput contém os dados para gerar contrato com assinatura digital
type DocuSealContractInput struct {
	// TemplateName é o nome do template no DocuSeal.
	// Valores válidos: ligue_saude_em_dia, ligue_mais_cuidado, ligue_vida_plena, ligue_cuidado_total, ligue_viver_bem
	// Se não especificado, usa ligue_saude_em_dia
	TemplateName string

	// Dados do cliente
	CustomerID string
	Nome       string
	Email      string
	CPF        string

	// Dados do plano
	PlanName      string
	Produto       string
	Valor         string
	Pagamento     string
	Periodicidade string

	// Dados adicionais
	Nascimento  string
	RG          string
	Orgao       string
	Sexo        string
	Civil       string
	Celular     string
	Fixo        string
	Endereco    string
	Numero      string
	Complemento string
	Bairro      string
	Cidade      string
	UF          string
	CEP         string

	// IP do cliente para auditoria
	ClientIP string

	// Field UUIDs do DocuSeal template
	// COLOCAR AQUI: Os UUIDs dos fields que serão preenchidos no DocuSeal
	// Exemplo de estrutura para DocuSeal field mapping
	FieldUUIDs map[string]string // key: field name, value: UUID no template
}

// DocuSealContractOutput contém os resultados da geração de contrato com assinatura digital
type DocuSealContractOutput struct {
	// SubmissionUUID é o identificador único da submission no DocuSeal
	SubmissionUUID string

	// SigningURL é a URL que o cliente deve acessar para assinar
	SigningURL string

	// TemplateUUID é o identificador único do template criado
	TemplateUUID string

	// Status do documento (DRAFT, SENT, SIGNED, etc)
	Status string

	// PDFBytes é o PDF preenchido (antes da assinatura)
	PDFBytes []byte
}

// NewGenerateContractWithDocuSealUseCase cria uma nova instância do usecase
func NewGenerateContractWithDocuSealUseCase(
	pdfGen ContractPDFGeneratorInterface,
	docuSealClient *docuseal.Client,
) *GenerateContractWithDocuSealUseCase {
	return &GenerateContractWithDocuSealUseCase{
		PdfGenerator:   pdfGen,
		DocuSealClient: docuSealClient,
	}
}

// Execute gera o contrato e prepara para assinatura digital
func (uc *GenerateContractWithDocuSealUseCase) Execute(ctx context.Context, input DocuSealContractInput) (*DocuSealContractOutput, error) {
	// DEBUG: Logs dos campos principais
	log.Printf("🔍 [DOCUSEAL DEBUG] Campos Recebidos:")
	log.Printf("   Valor: %q (vazio=%v)", input.Valor, input.Valor == "")
	log.Printf("   Produto: %q", input.Produto)
	log.Printf("   Pagamento: %q", input.Pagamento)
	log.Printf("   Periodicidade: %q", input.Periodicidade)

	// DEBUG: Logs dos campos de endereço
	log.Printf("🔍 [DOCUSEAL DEBUG] Campos de Endereço Recebidos:")
	log.Printf("   Endereco: %q (vazio=%v)", input.Endereco, input.Endereco == "")
	log.Printf("   Numero: %q (vazio=%v)", input.Numero, input.Numero == "")
	log.Printf("   Bairro: %q (vazio=%v)", input.Bairro, input.Bairro == "")
	log.Printf("   Cidade: %q (vazio=%v)", input.Cidade, input.Cidade == "")
	log.Printf("   UF: %q (vazio=%v)", input.UF, input.UF == "")
	log.Printf("   CEP: %q (vazio=%v)", input.CEP, input.CEP == "")
	log.Printf("   Complemento: %q (vazio=%v)", input.Complemento, input.Complemento == "")

	// Define template baseado no nome do plano se não especificado
	templateName := input.TemplateName
	if templateName == "" {
		// Tenta descobrir o template pelo nome do plano
		templateName = docuseal.GetTemplateFromPlanName(input.PlanName)
	}

	templateID, exists := docuseal.GetTemplateID(templateName)
	if !exists {
		return nil, fmt.Errorf("template inválido: %s", templateName)
	}

	log.Printf("📄 Preparando submissão DocuSeal (template: %s, ID: %d) — CustomerID=%s Email=%s", templateName, templateID, input.CustomerID, input.Email)
	if templateID == 0 {
		templateID = 3346712 // Template Saúde em Dia (teste)
	}

	fieldValues := map[string]string{
		"product":        input.Produto,
		"id":             input.CustomerID,
		"method_payment": "Método de Pagamento: " + input.Pagamento,
		"monthly":        "Mensal",
		"value":          "Valor: " + formatCurrencyForDocuSeal(input.Valor),
		"name":           input.Nome,
		"birthdate":      input.Nascimento,
		"cpf":            input.CPF,
		"genre":          input.Sexo,
		"marital_status": input.Civil,
		"cellphone":      input.Celular,
		"email":          input.Email,
		"address":        input.Endereco,
		"number":         input.Numero,
		"neighborhood":   input.Bairro,
		"city":           input.Cidade,
		"UF":             input.UF,
		"zip_code":       input.CEP,
	}

	// DEBUG: Log all field values before normalization
	log.Printf("🔍 [DOCUSEAL DEBUG] Field Values After Formatting:")
	for key, value := range fieldValues {
		log.Printf("   %s: %q", key, value)
	}

	fieldValues = normalizeDocuSealMonthly(fieldValues)

	submissionReq := &docuseal.CreateSubmissionRequest{
		TemplateID: templateID,
		SendEmail:  true, // Enviar email automático para assinatura
		Submitters: []docuseal.SignerAttribute{
			{
				Email:     input.Email,
				FullName:  input.Nome,
				Role:      docuseal.GetTemplateRole(templateName),
				Completed: true,
				Values:    fieldValues,
			},
		},
		CustomEmail: &docuseal.CustomEmailAttribute{
			Subject:  fmt.Sprintf("Cópia do Termo de Adesão - %s", input.PlanName),
			Body:     fmt.Sprintf("Olá, %s,\n\nConfirmamos o aceite do termo referente ao %s.\n\nEm anexo, você encontra a cópia do documento para seus registros, conforme estabelecido no fluxo de contratação.\n\nSe precisar de qualquer suporte técnico ou tiver dúvidas sobre o plano, conte conosco.\n\nAtenciosamente,\nEquipe Ligue Medicina", input.Nome, input.PlanName),
			FromName: "Ligue Medicina",
		},
	}
	submissionResp, err := uc.DocuSealClient.CreateSubmission(submissionReq)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar submission no DocuSeal: %w", err)
	}

	log.Printf("✅ Submission DocuSeal criada (UUID=%s) - Signing URL: %s", submissionResp.UUID, submissionResp.SigningURL)

	return &DocuSealContractOutput{
		SubmissionUUID: submissionResp.UUID,
		SigningURL:     submissionResp.SigningURL,
		TemplateUUID:   fmt.Sprintf("%d", templateID),
		Status:         submissionResp.Status,
		PDFBytes:       nil,
	}, nil
}

// GetSignedDocument recupera o documento assinado do DocuSeal
func (uc *GenerateContractWithDocuSealUseCase) GetSignedDocument(ctx context.Context, submissionUUID string) (*GetSignedDocumentOutput, error) {
	if submissionUUID == "" {
		return nil, fmt.Errorf("submissionUUID não pode ser vazio")
	}

	submission, err := uc.DocuSealClient.GetSubmission(submissionUUID)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter submission do DocuSeal: %w", err)
	}

	// Verificar se todos os signatários assinaram
	allSigned := true
	for _, signer := range submission.Signers {
		if signer.Status != "SIGNED" && signer.Status != "COMPLETED" {
			allSigned = false
			break
		}
	}

	return &GetSignedDocumentOutput{
		SubmissionUUID:   submissionUUID,
		Status:           submission.Status,
		DocumentURL:      submission.DocumentURL,
		AuditTrailURL:    submission.AuditTrailURL,
		AllSignersSigned: allSigned,
		Signers:          submission.Signers,
	}, nil
}

// GetSignedDocumentOutput contém os dados do documento assinado
type GetSignedDocumentOutput struct {
	SubmissionUUID   string
	Status           string
	DocumentURL      string
	AuditTrailURL    string
	AllSignersSigned bool
	Signers          []docuseal.SubmissionSigner
}

// ExecuteAutomatic gera o contrato no DocuSeal mas não envia signing_url
// Apenas registra que foi aceito. Será monitorado pelo webhook.
// Retorna UUID para rastreamento.
func (uc *GenerateContractWithDocuSealUseCase) ExecuteAutomatic(ctx context.Context, input DocuSealContractInput) (string, error) {
	// Define template padrão se não especificado
	templateName := input.TemplateName
	if templateName == "" {
		templateName = "ligue_saude_em_dia"
	}

	templateID, exists := docuseal.GetTemplateID(templateName)
	if !exists {
		return "", fmt.Errorf("template inválido: %s", templateName)
	}

	log.Printf("📄 Preparando submissão DocuSeal automática (template: %s, ID: %d) — CustomerID=%s Email=%s", templateName, templateID, input.CustomerID, input.Email)

	fieldValues := map[string]string{
		"product":        input.Produto,
		"id":             input.CustomerID,
		"method_payment": "Método de Pagamento: " + input.Pagamento,
		"monthly":        "Mensal",
		"value":          "Valor: " + formatCurrencyForDocuSeal(input.Valor),
		"name":           input.Nome,
		"birthdate":      input.Nascimento,
		"cpf":            input.CPF,
		"genre":          input.Sexo,
		"marital_status": input.Civil,
		"cellphone":      input.Celular,
		"email":          input.Email,
		"address":        input.Endereco,
		"number":         input.Numero,
		"neighborhood":   input.Bairro,
		"city":           input.Cidade,
		"UF":             input.UF,
		"zip_code":       input.CEP,
	}
	fieldValues = normalizeDocuSealMonthly(fieldValues)

	submissionReq := &docuseal.CreateSubmissionRequest{
		TemplateID: templateID,
		SendEmail:  true,
		Submitters: []docuseal.SignerAttribute{
			{
				Email:     input.Email,
				FullName:  input.Nome,
				Role:      docuseal.GetTemplateRole(templateName),
				Completed: true,
				Values:    fieldValues,
			},
		},
		CustomEmail: &docuseal.CustomEmailAttribute{
			Subject:  "Seu contrato de saúde - Ligue Saúde em Dia",
			Body:     fmt.Sprintf("Olá %s,\n\nSeu contrato de adesão está pronto para revisão.\n\nPor favor, acesse o link abaixo para visualizar e assinar seu documento:\n\nhttps://docuseal.com/submissions/[SUBMISSION_UUID]\n\nAtenciosamente,\nLigue Saúde em Dia", input.Nome),
			FromName: "Ligue Saúde em Dia",
		},
	}

	resp, err := uc.DocuSealClient.CreateSubmission(submissionReq)
	if err != nil {
		log.Printf("❌ Falha ao criar submission DocuSeal: %v", err)
		return "", err
	}

	log.Printf("✅ Submission DocuSeal criada (UUID=%s, CustomerID=%s)", resp.UUID, input.CustomerID)
	return resp.UUID, nil
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

// formatCurrencyForDocuSeal converte um valor numérico para formato de moeda brasileira
// Ex: "99.90" -> "R$ 99,90" ou "99,90" -> "R$ 99,90"
// Se já estiver em formato "R$ XX,XX", retorna como está
func formatCurrencyForDocuSeal(valor string) string {
	log.Printf("  DEBUG formatCurrencyForDocuSeal: input=%q", valor)

	if valor == "" {
		log.Printf("  DEBUG formatCurrencyForDocuSeal: returning empty (empty input)")
		return ""
	}

	// Remove espaços
	valor = strings.TrimSpace(valor)

	// Se já começa com R$, retorna como está
	if strings.HasPrefix(valor, "R$") {
		log.Printf("  DEBUG formatCurrencyForDocuSeal: already formatted, returning=%q", valor)
		return valor
	}

	// Substitui ponto por vírgula (conversão de formato)
	valor = strings.ReplaceAll(valor, ".", ",")

	// Adiciona R$ no início
	result := fmt.Sprintf("R$ %s", valor)
	log.Printf("  DEBUG formatCurrencyForDocuSeal: formatted result=%q", result)
	return result
}
