package asaas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	httptrace "github.com/DataDog/dd-trace-go/contrib/net/http/v2"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey, baseURL string) *Client {
	timeout := 30 * time.Second
	if rawTimeout := strings.TrimSpace(os.Getenv("ASAAS_HTTP_TIMEOUT_SECONDS")); rawTimeout != "" {
		if seconds, err := strconv.Atoi(rawTimeout); err == nil && seconds > 0 {
			timeout = time.Duration(seconds) * time.Second
		}
	}

	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: httptrace.WrapClient(&http.Client{Timeout: timeout}),
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

	fmt.Printf("[asaas] CreateCustomer payload: name=%q email=%q cpf=%q phone=%q mobile=%q zip=%q number=%q\n",
		payload.Name, payload.Email, payload.CpfCnpj, payload.Phone, payload.MobilePhone, payload.PostalCode, payload.AddressNumber)

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
		fmt.Printf("❌ ERRO CRIAR CLIENTE ASAAS (status %d): %s\n", resp.StatusCode, string(body))

		// Tenta parsear a resposta de erro para logar campos específicos
		var errResp struct {
			Errors []struct {
				Code        string `json:"code"`
				Description string `json:"description"`
				Field       string `json:"field"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(body, &errResp); err == nil && len(errResp.Errors) > 0 {
			fmt.Printf("[asaas] Campos com erro:\n")
			for _, e := range errResp.Errors {
				fmt.Printf("  - Field: %q | Code: %q | Description: %q\n", e.Field, e.Code, e.Description)
			}
		}

		return "", fmt.Errorf("erro criar cliente asaas (status %d)", resp.StatusCode)
	}

	var response customerResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("erro decode asaas: %w", err)
	}

	fmt.Printf("[asaas] Cliente criado com sucesso: id=%q\n", response.ID)
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

	fmt.Printf("[asaas] Subscribe payload: customer=%q value=%.2f card_holder=%q phone=%q email=%q\n",
		input.CustomerID, input.Price, input.CardHolderName, input.HolderPhone, input.HolderEmail)

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
		fmt.Printf("❌ ERRO API ASAAS Subscribe (Status %d): %s\n", resp.StatusCode, string(body))

		// Tenta parsear a resposta de erro
		var errResp struct {
			Errors []struct {
				Code        string `json:"code"`
				Description string `json:"description"`
				Field       string `json:"field"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(body, &errResp); err == nil && len(errResp.Errors) > 0 {
			fmt.Printf("[asaas] Campos com erro na subscrição:\n")
			for _, e := range errResp.Errors {
				fmt.Printf("  - Field: %q | Code: %q | Description: %q\n", e.Field, e.Code, e.Description)
			}
		}

		return "", "", fmt.Errorf("api asaas rejeitou (status %d)", resp.StatusCode)
	}

	var response subscriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", "", fmt.Errorf("erro ao ler resposta asaas: %w", err)
	}

	fmt.Printf("[asaas] Subscrição criada com sucesso: id=%q status=%q\n", response.ID, response.Status)
	return response.ID, response.Status, nil
}

func (c *Client) SubscribePix(input SubscribePixInput) (string, *PixOutput, error) {
	priceFloat := float64(input.Price) / 100.0

	loc, _ := time.LoadLocation("America/Sao_Paulo")
	nowBrazil := time.Now().In(loc)
	expirationDate := nowBrazil.Add(30 * time.Minute) // PIX expira em 30 min

	reqBody := map[string]interface{}{
		"customer":    input.CustomerID,
		"billingType": "PIX",
		"value":       priceFloat,
		"cycle":       "MONTHLY",
		"nextDueDate": nowBrazil.Format("2006-01-02"),      // Vence hoje
		"dueDate":     expirationDate.Format("2006-01-02"), // Data de expiração
		"description": "Plano Ligue - Assinatura",
	}

	fmt.Printf("[asaas] SubscribePix: customer=%q value=%.2f cycle=MONTHLY\n", input.CustomerID, priceFloat)

	respBody, err := c.post("/subscriptions", reqBody)
	if err != nil {
		fmt.Printf("❌ ERRO ao criar PIX por recorrência: %v\n", err)
		return "", nil, fmt.Errorf("Erro em criar o pix por recorrencia: %w", err)
	}

	var subResp asaasSubscriptionResponse
	if err := json.Unmarshal(respBody, &subResp); err != nil {
		return "", nil, fmt.Errorf("erro json assinatura: %w", err)
	}
	subscriptionID := subResp.ID

	var lastErr error
	for attempt := 0; attempt < 6; attempt++ {
		pix, err := c.GetPixBySubscriptionID(subscriptionID)
		if err == nil && pix != nil {
			return subscriptionID, pix, nil
		}
		lastErr = err
		if attempt < 5 {
			time.Sleep(600 * time.Millisecond)
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("não foi possível recuperar o QR Code do PIX")
	}
	return subscriptionID, nil, lastErr

}

func (c *Client) GetPixBySubscriptionID(subscriptionID string) (*PixOutput, error) {
	subscriptionID = strings.TrimSpace(subscriptionID)
	if subscriptionID == "" {
		return nil, fmt.Errorf("subscriptionID vazio")
	}

	pathList := fmt.Sprintf("/subscriptions/%s/payments?limit=1", subscriptionID)

	paymentsBody, err := c.get(pathList)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar pagamentos: %w", err)
	}

	var listResp asaasListPaymentsResponse
	if err := json.Unmarshal(paymentsBody, &listResp); err != nil {
		return nil, fmt.Errorf("erro json lista: %w", err)
	}

	if len(listResp.Data) == 0 {
		return nil, fmt.Errorf("nenhuma cobrança gerada")
	}

	paymentID := strings.TrimSpace(listResp.Data[0].ID)
	if paymentID == "" {
		return nil, fmt.Errorf("pagamento inválido para assinatura")
	}

	pathQr := fmt.Sprintf("/payments/%s/pixQrCode", paymentID)
	qrBody, err := c.get(pathQr)
	if err != nil {
		return nil, fmt.Errorf("erro ao pegar qrcode: %w", err)
	}

	var qrResp asaasQrCodeResponse
	if err := json.Unmarshal(qrBody, &qrResp); err != nil {
		return nil, fmt.Errorf("erro json qrcode: %w", err)
	}

	fmt.Printf("[asaas] PIX qrCode payload: subscription=%q payment=%q code_len=%d qr_len=%d\n",
		subscriptionID, paymentID, len(strings.TrimSpace(qrResp.Payload)), len(strings.TrimSpace(qrResp.EncodedImage)))

	return &PixOutput{
		CopyPaste: qrResp.Payload,
		URL:       qrResp.EncodedImage,
	}, nil
}

func (c *Client) DeleteSubscription(subscriptionID string) error {
	subscriptionID = strings.TrimSpace(subscriptionID)
	if subscriptionID == "" {
		return fmt.Errorf("subscriptionID vazio")
	}

	url := fmt.Sprintf("%s/subscriptions/%s", c.baseURL, subscriptionID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("erro ao criar request de delete: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("erro de conexão ao deletar assinatura: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("erro ao deletar assinatura asaas (%d): %s", resp.StatusCode, string(body))
	}

	fmt.Printf("[asaas] Assinatura deletada: id=%q\n", subscriptionID)
	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("access_token", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "LiguePayments/1.0")
}
