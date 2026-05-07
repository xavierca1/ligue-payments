package docuseal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ============================================================================
// DTOs (Data Transfer Objects)
// ============================================================================

// CreateTemplateRequest representa a requisição para criar um template no DocuSeal
type CreateTemplateRequest struct {
	// DocumentsAttributes contém os documentos que serão usados no template
	DocumentsAttributes []DocumentAttribute `json:"documents_attributes"`
	// Name é o nome do template
	Name string `json:"name"`
	// Description é a descrição do template
	Description string `json:"description,omitempty"`
}

// DocumentAttribute representa um documento no template
type DocumentAttribute struct {
	// File é o conteúdo do documento em base64
	File string `json:"file"`
	// FileName é o nome do arquivo
	FileName string `json:"file_name"`
}

// CreateTemplateResponse é a resposta ao criar um template
type CreateTemplateResponse struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	UUID     string `json:"uuid"`
	SchemaID int    `json:"schema_id"`
	Status   string `json:"status"`
}

// CreateSubmissionRequest é a requisição para criar um documento de assinatura
type CreateSubmissionRequest struct {
	// TemplateID é o ID numérico do template (ex.: 3346712)
	TemplateID int `json:"template_id,omitempty"`
	// TemplateUUID é o UUID do template
	TemplateUUID string `json:"template_uuid,omitempty"`
	// SendEmail indica se deve enviar email ao signatário
	SendEmail bool `json:"send_email"`
	// Submitters contém os dados dos signatários (campo correto da API)
	Submitters []SignerAttribute `json:"submitters"`
	// CustomEmail é o email customizado para envio
	CustomEmail *CustomEmailAttribute `json:"custom_email,omitempty"`
}

// SignerAttribute representa um signatário no documento
type SignerAttribute struct {
	// Email do signatário
	Email string `json:"email,omitempty"`
	// FullName do signatário
	FullName string `json:"name,omitempty"`
	// Role do signatário
	Role string `json:"role,omitempty"`
	// Completed marca o submitter como auto-assinado via API
	Completed bool `json:"completed,omitempty"`
	// Phone do signatário (opcional)
	Phone string `json:"phone,omitempty"`
	// Values é o mapa de campos preenchidos do template
	Values map[string]string `json:"values,omitempty"`
}

// FieldAttribute representa um campo preenchido no documento
type FieldAttribute struct {
	// Uuid do campo no template
	UUID string `json:"uuid"`
	// Name do campo no template (alternativa ao UUID)
	Name string `json:"name,omitempty"`
	// Value é o valor a ser preenchido
	Value string `json:"value"`
}

// CustomEmailAttribute contém configurações de email customizado
type CustomEmailAttribute struct {
	Subject  string `json:"subject"`
	Body     string `json:"body"`
	FromName string `json:"from_name,omitempty"`
}

// CreateSubmissionResponse é a resposta ao criar um documento para assinatura
type CreateSubmissionResponse struct {
	ID           int    `json:"id"`
	UUID         string `json:"uuid"`
	Status       string `json:"status"`
	DocumentURL  string `json:"document_url"`
	SigningURL   string `json:"signing_url"`
	TemplateUUID string `json:"template_uuid"`
}

// SubmissionSigner representa um signatário em um documento
type SubmissionSigner struct {
	ID        int    `json:"id"`
	UUID      string `json:"uuid"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	Status    string `json:"status"`
	SignedAt  string `json:"signed_at,omitempty"`
	SignURL   string `json:"sign_url"`
	AuditLink string `json:"audit_link,omitempty"`
}

// GetSubmissionResponse é a resposta ao obter detalhes de um documento
type GetSubmissionResponse struct {
	ID               int                    `json:"id"`
	UUID             string                 `json:"uuid"`
	Status           string                 `json:"status"`
	CreatedAt        string                 `json:"created_at"`
	UpdatedAt        string                 `json:"updated_at"`
	DocumentURL      string                 `json:"document_url"`
	PreviewURL       string                 `json:"preview_url"`
	AuditTrailURL    string                 `json:"audit_trail_url"`
	Signers          []SubmissionSigner     `json:"signers"`
	SourceSubmission *GetSubmissionResponse `json:"source_submission,omitempty"`
}

// ErrorResponse representa uma resposta de erro da API
type ErrorResponse struct {
	Error   string              `json:"error,omitempty"`
	Message string              `json:"message,omitempty"`
	Errors  map[string][]string `json:"errors,omitempty"`
}

// ============================================================================
// Client
// ============================================================================

// Client é o cliente HTTP para integração com DocuSeal
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient cria um novo cliente DocuSeal
func NewClient(baseURL, apiKey string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = "https://app.docuseal.com/api"
	}

	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

// CreateTemplate cria um novo template no DocuSeal
// Retorna o UUID do template criado
func (c *Client) CreateTemplate(req *CreateTemplateRequest) (*CreateTemplateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("CreateTemplateRequest não pode ser nil")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar template request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/templates", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %w", err)
	}

	c.setHeaders(httpReq)
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("erro ao executar request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp ErrorResponse
		_ = json.Unmarshal(respBody, &errResp)
		return nil, fmt.Errorf("DocuSeal API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result CreateTemplateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("erro ao desserializar response: %w", err)
	}

	return &result, nil
}

// CreateSubmission cria um documento para assinatura baseado em um template
// Retorna a URL para o signatário assinar e o UUID do documento
func (c *Client) CreateSubmission(req *CreateSubmissionRequest) (*CreateSubmissionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("CreateSubmissionRequest não pode ser nil")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar submission request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/submissions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %w", err)
	}

	c.setHeaders(httpReq)
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("erro ao executar request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp ErrorResponse
		_ = json.Unmarshal(respBody, &errResp)
		return nil, fmt.Errorf("DocuSeal API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result CreateSubmissionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		// Se falhar desserializar como objeto único, tenta como array
		var resultArray []CreateSubmissionResponse
		if err2 := json.Unmarshal(respBody, &resultArray); err2 == nil && len(resultArray) > 0 {
			result = resultArray[0]
		} else {
			fmt.Printf("DEBUG: Response body: %s\n", string(respBody))
			return nil, fmt.Errorf("erro ao desserializar response: %w", err)
		}
	}

	return &result, nil
}

// GetSubmission obtém os detalhes de um documento de assinatura
func (c *Client) GetSubmission(submissionUUID string) (*GetSubmissionResponse, error) {
	if strings.TrimSpace(submissionUUID) == "" {
		return nil, fmt.Errorf("submissionUUID não pode ser vazio")
	}

	httpReq, err := http.NewRequest("GET", c.baseURL+"/submissions/"+submissionUUID, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %w", err)
	}

	c.setHeaders(httpReq)
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("erro ao executar request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DocuSeal API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result GetSubmissionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("erro ao desserializar response: %w", err)
	}

	return &result, nil
}

// SetHeaders adiciona os headers necessários ao request
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", c.apiKey)
}
