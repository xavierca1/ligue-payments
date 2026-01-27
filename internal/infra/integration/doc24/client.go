package doc24

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync" // 👈 Importante para segurança entre threads
	"time"

	"github.com/xavierca1/ligue-payments/internal/infra/queue"
)

type Client struct {
	ClientID     string
	ClientSecret string
	BaseURL      string
	HTTP         *http.Client

	// Campos para gerenciar o Token (Estado interno)
	accessToken string
	expiresAt   time.Time
	mu          sync.Mutex // Cadeado para ninguém atropelar a renovação do token
}

type AuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"` // Geralmente vem em segundos
}

// NewClient inicializa. Note que mudei User/Pass para ClientID/Secret para ficar semântico
func NewClient(clientID, clientSecret string) *Client {
	return &Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		BaseURL:      "https://api-hml.doc24.com.br", // Confirme se é essa URL de Auth mesmo
		HTTP:         &http.Client{Timeout: 15 * time.Second},
	}
}

// ensureValidToken verifica se o token existe e é válido. Se não, faz login.
func (c *Client) ensureValidToken() error {
	c.mu.Lock()         // 🔒 Trava: Só um worker renova por vez
	defer c.mu.Unlock() // 🔓 Destrava quando terminar a função

	// Margem de segurança de 1 minuto. Se faltar menos de 1 min pra vencer, renova.
	if c.accessToken != "" && time.Now().Add(1*time.Minute).Before(c.expiresAt) {
		return nil // Token ainda é válido
	}

	log.Println("🔄 Token Doc24 expirado ou inexistente. Renovando...")

	url := fmt.Sprintf("%s/authentication", c.BaseURL)
	payload := map[string]string{
		"client_id":     c.ClientID,
		"client_secret": c.ClientSecret,
	}

	jsonBody, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("erro conexão auth doc24: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return fmt.Errorf("falha login doc24: status %d", resp.StatusCode)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("erro decode auth json: %w", err)
	}

	c.accessToken = authResp.AccessToken
	c.expiresAt = time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second)

	log.Printf(" Novo Token Doc24 obtido! Válido até: %s", c.expiresAt.Format(time.RFC3339))
	return nil
}

func (c *Client) CreateBeneficiary(ctx context.Context, input queue.ActivationPayload) error {
	if err := c.ensureValidToken(); err != nil {
		return fmt.Errorf("falha na autenticação antes de criar beneficiário: %w", err)
	}

	// 2. Monta o Payload
	payload := map[string]interface{}{
		"partner_user": c.ClientID, // As vezes pedem o user aqui dentro também
		"external_id":  input.CustomerID,
		"beneficiary": map[string]string{
			"name":  input.Name,
			"email": input.Email,
			"plan":  input.PlanID,
		},
	}

	jsonBody, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/beneficiaries/create", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("erro request doc24: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("doc24 erro na criação: status %d", resp.StatusCode)
	}

	return nil
}
