package mail

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/xavierca1/ligue-payments/internal/entity"
)

// GraphEmailSender usa Microsoft Graph API com OAuth2 em vez de SMTP
type GraphEmailSender struct {
	ClientID     string
	ClientSecret string
	TenantID     string
	FromEmail    string
}

// GraphTokenResponse retornada pelo Azure
type GraphTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
	ErrorCodes  []int  `json:"error_codes"`
}

// NewGraphEmailSender cria um novo sender com Graph/OAuth2
func NewGraphEmailSender(clientID, clientSecret, tenantID, fromEmail string) *GraphEmailSender {
	return &GraphEmailSender{
		ClientID:     strings.TrimSpace(clientID),
		ClientSecret: strings.TrimSpace(clientSecret),
		TenantID:     strings.TrimSpace(tenantID),
		FromEmail:    strings.TrimSpace(fromEmail),
	}
}

// getAccessToken obtém um novo token do Azure AD
func (s *GraphEmailSender) getAccessToken() (string, error) {
	if s.ClientID == "" || s.ClientSecret == "" || s.TenantID == "" {
		return "", fmt.Errorf("credenciais Graph/OAuth2 incompletas")
	}

	tokenEndpoint := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", s.TenantID)

	data := strings.NewReader(fmt.Sprintf(
		"grant_type=client_credentials&client_id=%s&client_secret=%s&scope=https://graph.microsoft.com/.default",
		s.ClientID,
		s.ClientSecret,
	))

	resp, err := http.Post(tokenEndpoint, "application/x-www-form-urlencoded", data)
	if err != nil {
		return "", fmt.Errorf("erro ao fazer request para token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("erro ao ler response do token: %w", err)
	}

	var tokenResp GraphTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("erro ao parsear response do token: %w", err)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("erro Azure: %s - %s (codes: %v)", tokenResp.Error, tokenResp.ErrorDesc, tokenResp.ErrorCodes)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("nenhum access_token retornado do Azure")
	}

	return tokenResp.AccessToken, nil
}

// SendWelcome envia e-mail de boas-vindas via Graph API
func (s *GraphEmailSender) SendWelcome(to, name, productName, pdfLink string) error {
	cardNumber := ""
	if strings.TrimSpace(pdfLink) != "" {
		cardNumber = pdfLink
	}

	return s.sendWelcomeInternal(to, name, productName, pdfLink, cardNumber, nil)
}

func (s *GraphEmailSender) sendWelcomeInternal(to, name, productName, pdfLink, cardNumber string, dependents []*entity.Dependent) error {
	accessToken, err := s.getAccessToken()
	if err != nil {
		return fmt.Errorf("falha ao obter token: %w", err)
	}

	// Parse template
	data := WelcomeEmailData{
		Name:        name,
		FirstName:   firstName(name),
		ProductName: productName,
		PDFLink:     pdfLink,
		PortalURL:   "https://app.liguemedicina.com.br",
		WhatsAppURL: "https://wa.me/5561999999999",
	}

	tmplPath := filepath.Join("templates", "welcome.html")
	t, err := template.ParseFiles(tmplPath)
	if err != nil {
		return fmt.Errorf("erro ao ler template de email: %w", err)
	}

	var body bytes.Buffer
	if err := t.Execute(&body, data); err != nil {
		return fmt.Errorf("erro ao processar template: %w", err)
	}

	// Preparar payload Graph
	graphPayload := map[string]interface{}{
		"message": map[string]interface{}{
			"subject": "Sua assinatura está ativa na Ligue Medicina",
			"body": map[string]string{
				"contentType": "HTML",
				"content":     body.String(),
			},
			"toRecipients": []map[string]interface{}{
				{
					"emailAddress": map[string]string{
						"address": to,
					},
				},
			},
			"from": map[string]interface{}{
				"emailAddress": map[string]string{
					"address": s.FromEmail,
				},
			},
		},
	}

	attachments := BuildMembershipCardAttachments(name, productName, cardNumber, data.PortalURL, dependents)
	if len(attachments) > 0 {
		message, _ := graphPayload["message"].(map[string]interface{})
		graphAttachments := make([]map[string]interface{}, 0, len(attachments))
		for _, attachment := range attachments {
			graphAttachments = append(graphAttachments, map[string]interface{}{
				"@odata.type":  "#microsoft.graph.fileAttachment",
				"name":         attachment.Filename,
				"contentType":  "application/pdf",
				"contentBytes": base64.StdEncoding.EncodeToString(attachment.Content),
			})
		}
		message["attachments"] = graphAttachments
	}

	payloadBytes, err := json.Marshal(graphPayload)
	if err != nil {
		return fmt.Errorf("erro ao serializar payload: %w", err)
	}

	// Fazer request para Graph API
	endpoint := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/sendMail", s.FromEmail)
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("erro ao criar request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao enviar request Graph: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("erro ao ler response: %w", err)
	}

	// Graph retorna 202 Accepted para sucesso
	if resp.StatusCode != 202 {
		var errResp map[string]interface{}
		json.Unmarshal(respBody, &errResp)
		return fmt.Errorf("erro Graph: HTTP %d - %v", resp.StatusCode, errResp)
	}

	return nil
}

// SendWelcomeEmail wrapper para manter compatibilidade
func (s *GraphEmailSender) SendWelcomeEmail(name, email string) error {
	return s.SendWelcomeEmailWithCard(name, email, "", "Ligue Medicina", "")
}

func (s *GraphEmailSender) SendWelcomeEmailWithCard(name, email, cpf, planName, providerID string) error {
	cardNumber := strings.TrimSpace(providerID)
	if cardNumber == "" {
		cardNumber = strings.TrimSpace(cpf)
	}

	if strings.TrimSpace(planName) == "" {
		planName = "Ligue Medicina"
	}

	return s.sendWelcomeInternal(email, name, planName, "", cardNumber, nil)
}

func (s *GraphEmailSender) SendWelcomeEmailWithContractAndDependents(name, email, cpf, planName, providerID string, dependents []*entity.Dependent, contractPDF []byte) error {
	cardNumber := strings.TrimSpace(providerID)
	if cardNumber == "" {
		cardNumber = strings.TrimSpace(cpf)
	}

	if strings.TrimSpace(planName) == "" {
		planName = "Ligue Medicina"
	}

	return s.sendWelcomeInternal(email, name, planName, "", cardNumber, dependents)
}

func (s *GraphEmailSender) SendWelcomeEmailWithCardAndDependents(name, email, cpf, planName, providerID string, dependents []*entity.Dependent) error {
	cardNumber := strings.TrimSpace(providerID)
	if cardNumber == "" {
		cardNumber = strings.TrimSpace(cpf)
	}

	if strings.TrimSpace(planName) == "" {
		planName = "Ligue Medicina"
	}

	return s.sendWelcomeInternal(email, name, planName, "", cardNumber, dependents)
}
