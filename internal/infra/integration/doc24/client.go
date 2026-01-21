package doc24

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/xavierca1/ligue-payments/internal/entity"
)

type Client struct {
	baseURL      string
	clientID     string
	clientSecret string
	httpClient   *http.Client
	token        string
	tokenExp     time.Time
}

type AuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func NewClient(clientID, clientSecret string) *Client {
	return &Client{
		baseURL:      "https://tapi.doc24.com.ar/ws/api/v2",
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Authenticate() error {
	if c.token != "" && time.Now().Add(1*time.Minute).Before(c.tokenExp) {
		return nil
	}

	url := fmt.Sprintf("%s/authentication", c.baseURL)
	payload := map[string]string{
		"client_id":     c.clientID,
		"client_secret": c.clientSecret,
	}

	jsonBody, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("erro de conexão auth: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("falha login doc24: status %d", resp.StatusCode)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("erro decode auth: %w", err)
	}

	c.token = authResp.AccessToken
	c.tokenExp = time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second)

	return nil
}

func (c *Client) CreateBeneficiary(customer *entity.Customer) error {
	if err := c.Authenticate(); err != nil {
		return fmt.Errorf("Erro de auth: ", err)
	}

	payload := mapToDoc24(customer)
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/afiliados", c.baseURL)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("erro request doc24: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("erro doc24 status: %d", resp.StatusCode)
	}

	return nil
}
