package asaas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type CreateCustomerRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	CpfCnpj string `json:"cpfCnpj"`
}

type CreateCustomerResponse struct {
	ID string `json:"id"` // Ex: "cus_000005105952"
}

type CreditCard struct {
	HolderName  string `json:"holderName"`
	Number      string `json:"number"`
	ExpiryMonth string `json:"expiryMonth"`
	ExpiryYear  string `json:"expiryYear"`
	CCV         string `json:"ccv"`
}

type CreateSubscriptionRequest struct {
	Customer    string     `json:"customer"`
	BillingType string     `json:"billingType"` // "CREDIT_CARD"
	Value       float64    `json:"value"`
	NextDueDate string     `json:"nextDueDate"` // "YYYY-MM-DD"
	Cycle       string     `json:"cycle"`       // "MONTHLY"
	CreditCard  CreditCard `json:"creditCard"`
	RemoteIp    string     `json:"remoteIp"` // Importante para antifraude
}

type SubscriptionResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"` // "ACTIVE"
}

type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

func NewClient(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) CreateCustomer(name, email, cpf string) (string, error) {
	url := fmt.Sprintf("%s/customers", c.baseURL)

	payload := CreateCustomerRequest{
		Name:    name,
		Email:   email,
		CpfCnpj: cpf,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("erro ao fazer marshal do payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("erro de conexão com asaas: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("erro na api asaas (status %d): %s", resp.StatusCode, string(body))
	}

	var response CreateCustomerResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return response.ID, nil
}

func (c *Client) Subscribe(customerID string, amount float64, holder, number, month, year, ccv string) (string, string, error) {
	url := fmt.Sprintf("%s/subscriptions", c.baseURL)
	today := time.Now().Format("2006-01-02")

	cc := CreditCard{
		HolderName:  holder,
		Number:      number,
		ExpiryMonth: month,
		ExpiryYear:  year,
		CCV:         ccv,
	}

	payload := CreateSubscriptionRequest{
		Customer:    customerID,
		BillingType: "CREDIT_CARD",
		Value:       amount,
		NextDueDate: today,
		Cycle:       "MONTHLY",
		CreditCard:  cc,
		RemoteIp:    "127.0.0.1",
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", "", err
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("erro asaas (status %d): %s", resp.StatusCode, string(body))
	}

	var response SubscriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", "", err
	}

	return response.ID, response.Status, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("access_token", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
}
