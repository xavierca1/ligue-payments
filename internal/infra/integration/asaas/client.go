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
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey, baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) post(endpoint string, body interface{}) ([]byte, error) {
	fullURL := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer marshal do body: %w", err)
	}

	req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("access_token", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro de conexão: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta: %w", err)
	}

	// 6. Valida se deu bom (200-299)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("erro na api asaas (%d): %s", resp.StatusCode, string(respBytes))
	}

	return respBytes, nil
}

func (c *Client) get(endpoint string) ([]byte, error) {
	fullURL := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("access_token", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro de conexão: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("erro na api asaas (%d): %s", resp.StatusCode, string(respBytes))
	}

	return respBytes, nil
}

func (c *Client) CreateCustomer(input CreateCustomerInput) (string, error) {
	url := fmt.Sprintf("%s/customers", c.baseURL)

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

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("erro request asaas: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ ERRO CRIAR CLIENTE ASAAS: %s\n", string(body))
		return "", fmt.Errorf("erro criar cliente asaas (status %d)", resp.StatusCode)
	}

	var response customerResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("erro decode asaas: %w", err)
	}

	return response.ID, nil
}

func (c *Client) Subscribe(input SubscribeInput) (string, string, error) {
	url := fmt.Sprintf("%s/subscriptions", c.baseURL)
	today := time.Now().Format("2006-01-02")

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

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("erro ao gerar json: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", "", err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("erro na conexão com asaas: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ ERRO API ASAAS (Status %d): %s\n", resp.StatusCode, string(body))
		return "", "", fmt.Errorf("api asaas rejeitou (status %d)", resp.StatusCode)
	}

	var response subscriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", "", fmt.Errorf("erro ao ler resposta asaas: %w", err)
	}

	return response.ID, response.Status, nil
}

func (c *Client) SubscribePix(input SubscribePixInput) (string, *PixOutput, error) {
	priceFloat := float64(input.Price) / 100.0

	loc, _ := time.LoadLocation("America/Sao_Paulo")
	nowBrazil := time.Now().In(loc)

	reqBody := map[string]interface{}{
		"customer":    input.CustomerID,
		"billingType": "PIX",
		"value":       priceFloat,
		"cycle":       "MONTHLY",
		"nextDueDate": nowBrazil.Format("2006-01-02"), // Vence hoje
		"description": "Plano Ligue - Assinatura",
	}
	respBody, err := c.post("/subscriptions", reqBody)
	if err != nil {
		return "", nil, fmt.Errorf("Erro em criar o pix por recorrencia")
	}

	var subResp asaasSubscriptionResponse
	if err := json.Unmarshal(respBody, &subResp); err != nil {
		return "", nil, fmt.Errorf("erro json assinatura: %w", err)
	}
	subscriptionID := subResp.ID
	pathList := fmt.Sprintf("/subscriptions/%s/payments?limit=1", subscriptionID)

	paymentsBody, err := c.get(pathList)
	if err != nil {
		return subscriptionID, nil, fmt.Errorf("erro ao listar pagamentos: %w", err)
	}

	var listResp asaasListPaymentsResponse
	if err := json.Unmarshal(paymentsBody, &listResp); err != nil {
		return subscriptionID, nil, fmt.Errorf("erro json lista: %w", err)
	}

	if len(listResp.Data) == 0 {
		return subscriptionID, nil, fmt.Errorf("nenhuma cobrança gerada")
	}
	paymentID := listResp.Data[0].ID

	pathQr := fmt.Sprintf("/payments/%s/pixQrCode", paymentID)

	qrBody, err := c.get(pathQr)
	if err != nil {
		return subscriptionID, nil, fmt.Errorf("erro ao pegar qrcode: %w", err)
	}
	var qrResp asaasQrCodeResponse
	if err := json.Unmarshal(qrBody, &qrResp); err != nil {
		return subscriptionID, nil, fmt.Errorf("erro json qrcode: %w", err)
	}

	return subscriptionID, &PixOutput{
		CopyPaste: qrResp.Payload,
		URL:       qrResp.EncodedImage,
	}, nil

}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("access_token", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "LiguePayments/1.0")
}
