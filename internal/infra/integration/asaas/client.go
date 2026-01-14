package asaas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewClient(apiKey, baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// CreateCustomer: Cria o cliente no Asaas e retorna o ID (cus_xxxx)
func (c *Client) CreateCustomer(input CreateCustomerInput) (string, error) {
	url := fmt.Sprintf("%s/customers", c.baseURL)

	// 1. Converte DTO -> Request do Asaas
	payload := createCustomerRequest{
		Name:                 input.Name,
		Email:                input.Email,
		CpfCnpj:              input.CpfCnpj,
		Phone:                input.Phone,
		MobilePhone:          input.MobilePhone,
		PostalCode:           input.PostalCode,
		AddressNumber:        input.AddressNumber,
		NotificationDisabled: true, // Para não enviar email automático do Asaas agora
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("erro ao marshal customer: %w", err)
	}

	// 2. Cria Request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	c.setHeaders(req)

	// 3. Envia
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("erro request asaas: %w", err)
	}
	defer resp.Body.Close()

	// 4. Trata Erro
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ ERRO CRIAR CLIENTE ASAAS: %s\n", string(body))
		return "", fmt.Errorf("erro criar cliente asaas (status %d)", resp.StatusCode)
	}

	// 5. Decodifica
	var response customerResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("erro decode asaas: %w", err)
	}

	return response.ID, nil
}

// Subscribe: Recebe o DTO limpo, converte pro formato Asaas e envia
func (c *Client) Subscribe(input SubscribeInput) (string, string, error) {
	url := fmt.Sprintf("%s/subscriptions", c.baseURL)
	today := time.Now().Format("2006-01-02")

	// 1. De-Para: Converte seu DTO (SubscribeInput) para o JSON do Asaas
	payload := createSubscriptionRequest{
		Customer:    input.CustomerID,
		BillingType: "CREDIT_CARD",
		Value:       input.Price,
		NextDueDate: today,
		Cycle:       "MONTHLY",
		Description: "Assinatura Ligue Saúde", // Descrição na fatura

		CreditCard: creditCard{
			HolderName:  input.CardHolderName,
			Number:      input.CardNumber,
			ExpiryMonth: input.CardMonth,
			ExpiryYear:  input.CardYear,
			CCV:         input.CardCCV,
		},

		CreditCardHolderInfo: creditCardHolderInfo{
			Name:          input.CardHolderName,
			Email:         input.HolderEmail,
			CpfCnpj:       input.HolderCpfCnpj,
			PostalCode:    input.HolderPostalCode,
			AddressNumber: input.HolderAddressNum,
			Phone:         input.HolderPhone,
			MobilePhone:   input.HolderPhone,
		},
	}

	// 2. Prepara o JSON
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("erro ao gerar json: %w", err)
	}

	// 3. Cria Request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", "", err
	}
	c.setHeaders(req)

	// 4. Envia
	resp, err := c.http.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("erro na conexão com asaas: %w", err)
	}
	defer resp.Body.Close()

	// 5. Trata Erros da API (400, 401, 500)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		// Log para ajudar no debug
		fmt.Printf("❌ ERRO API ASAAS (Status %d): %s\n", resp.StatusCode, string(body))
		return "", "", fmt.Errorf("api asaas rejeitou (status %d)", resp.StatusCode)
	}

	// 6. Decodifica Sucesso
	var response subscriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", "", fmt.Errorf("erro ao ler resposta asaas: %w", err)
	}

	return response.ID, response.Status, nil
}

// setHeaders centraliza os headers obrigatórios
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("access_token", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "LiguePayments/1.0")
}
