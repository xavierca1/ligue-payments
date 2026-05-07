package doc24

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	httptrace "github.com/DataDog/dd-trace-go/contrib/net/http/v2"
	"github.com/xavierca1/ligue-payments/internal/infra/queue"
)

const (
	BaseURL = "https://tapi.doc24.com.ar/ws/api/v2"
	AuthURL = BaseURL + "/authentication" // Endpoint correto para Client Credentials
)

var eligibilityEndpoints = []string{
	BaseURL + "/portal/elegibilidad/",
	BaseURL + "/portal/elegibilidad",
	"https://tapi.doc24.com.ar/v2/portal/elegibilidad/",
	"https://tapi.doc24.com.ar/v2/portal/elegibilidad",
}

var nonDigitsRegex = regexp.MustCompile(`\D`)

type Client struct {
	HTTPClient   *http.Client
	ClientID     string
	ClientSecret string

	mu          sync.Mutex
	token       string
	tokenExpiry time.Time
}

func NewClient(clientID, clientSecret string) *Client {
	return &Client{
		HTTPClient:   httptrace.WrapClient(&http.Client{Timeout: 30 * time.Second}),
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
}

type AffiliateInput struct {
	Nombre          string `json:"nombre"`
	Apellido        string `json:"apellido"`
	Sexo            string `json:"sexo"`             // "M" ou "F"
	FechaNacimiento string `json:"fecha_nacimiento"` // YYYY-MM-DD

	IdentificacaoTrib string `json:"identificacion_tributaria"` // CPF
	NroDocumento      string `json:"nro_documento"`             // CPF
	NroDocTitular     string `json:"nro_documento_titular"`     // CPF

	Plan       string `json:"plan"`       // Nome do plano (vem do banco)
	Empresa    string `json:"empresa"`    // Nome da empresa na Doc24
	Credencial string `json:"credencial"` // Usamos CPF
	FechaAlta  string `json:"fecha_alta"` // Data de hoje

	TelefonoMovil string `json:"telefono_movil"`
	Email         string `json:"email"`
}

type EligibilityResponse struct {
	Estado  int    `json:"estado"`  // 1 = Sucesso
	Mensaje string `json:"mensaje"` // "OK"
}

type EligibilityAlternateResponse struct {
	Afiliado map[string]interface{} `json:"afiliado"`
}

func (c *Client) EnsureAuthenticated(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Add(30*time.Second).Before(c.tokenExpiry) {
		return nil
	}

	log.Println("🔄 [Doc24] Renovando token...")

	payload := map[string]string{
		"client_id":     c.ClientID,
		"client_secret": c.ClientSecret,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", AuthURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("erro request auth: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {

		var errorBody bytes.Buffer
		errorBody.ReadFrom(resp.Body)
		log.Printf("❌ [Doc24] Erro Auth: %s", errorBody.String())
		return fmt.Errorf("erro auth doc24: status %d", resp.StatusCode)
	}

	var data struct {
		Token     string `json:"token"`
		ExpiresIn int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return fmt.Errorf("erro decode auth: %w", err)
	}

	c.token = data.Token
	exp := data.ExpiresIn
	if exp == 0 {
		exp = 3600 // Default 1h
	}
	c.tokenExpiry = time.Now().Add(time.Duration(exp) * time.Second)

	log.Println("✅ [Doc24] Token renovado com sucesso!")
	return nil
}

func (c *Client) CreateBeneficiary(ctx context.Context, input queue.ActivationPayload) error {

	if err := c.EnsureAuthenticated(ctx); err != nil {
		return err
	}

	parts := strings.SplitN(input.Name, " ", 2)
	nome := parts[0]
	sobrenome := ""
	if len(parts) > 1 {
		sobrenome = parts[1]
	}

	sexo := "M"
	if input.Gender == "0" || strings.ToUpper(input.Gender) == "F" {
		sexo = "F"
	}

	planName := strings.TrimSpace(input.ProviderPlanCode)

	if planName == "" {
		log.Println("[Doc24] ProviderPlanCode veio vazio! Usando plano default.")
		planName = "ligue saude em dia individual"
	}

	today := time.Now().Format("2006-01-02")

	payload := AffiliateInput{
		Nombre:            nome,
		Apellido:          sobrenome,
		Sexo:              sexo,
		FechaNacimiento:   input.BirthDate,
		IdentificacaoTrib: normalizeDigits(input.CPF),
		NroDocumento:      normalizeDigits(input.CPF),
		NroDocTitular:     normalizeDigits(input.CPF),

		Plan:       planName,
		Empresa:    "Ag Med", // ⚠️ CONFIRME SE É 'Ag Med' ou 'Ligue Med'
		Credencial: normalizeDigits(input.CPF),
		FechaAlta:  today,

		TelefonoMovil: normalizeDigits(input.Phone),
		Email:         input.Email,
	}

	csvFile, err := buildEligibilityCSV(payload)
	if err != nil {
		return fmt.Errorf("erro ao montar CSV de elegibilidade: %w", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("archivo_adjunto", "elegibilidad.csv")
	if err != nil {
		return fmt.Errorf("erro ao criar multipart file: %w", err)
	}

	if _, err := part.Write(csvFile); err != nil {
		return fmt.Errorf("erro ao escrever CSV no multipart: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("erro ao finalizar multipart: %w", err)
	}

	var respBody []byte
	var lastStatus int
	var lastErrBody string
	var hitEndpoint string
	success := false

	for _, endpoint := range eligibilityEndpoints {
		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body.Bytes()))
		if err != nil {
			return err
		}

		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return fmt.Errorf("falha request doc24: %w", err)
		}

		respBody, _ = io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode < 400 {
			hitEndpoint = endpoint
			success = true
			break
		}

		lastStatus = resp.StatusCode
		lastErrBody = string(respBody)

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("erro api doc24 (%d): %s", resp.StatusCode, string(respBody))
		}
	}

	if !success {
		return fmt.Errorf("erro api doc24 (%d): %s", lastStatus, lastErrBody)
	}

	trimmedBody := strings.TrimSpace(string(respBody))

	if trimmedBody == "" {
		log.Printf("ℹ️ [Doc24] Resposta vazia considerada sucesso (endpoint=%s)", hitEndpoint)
		log.Printf("🚀 [Doc24] Sucesso! Paciente %s vinculado ao plano '%s' (ID: %s) endpoint=%s", input.Name, planName, input.CPF, hitEndpoint)
		return nil
	}

	var result EligibilityResponse
	if err := json.Unmarshal(respBody, &result); err == nil {
		if result.Estado == 1 {
			log.Printf("🚀 [Doc24] Sucesso! Paciente %s vinculado ao plano '%s' (ID: %s) endpoint=%s", input.Name, planName, input.CPF, hitEndpoint)
			return nil
		}

		if result.Estado == 0 && strings.TrimSpace(result.Mensaje) == "" && strings.Contains(trimmedBody, `"mensaje":null`) {
			log.Printf("ℹ️ [Doc24] Resposta estado=0 com mensaje nulo tratada como sucesso (endpoint=%s)", hitEndpoint)
			log.Printf("🚀 [Doc24] Sucesso! Paciente %s vinculado ao plano '%s' (ID: %s) endpoint=%s", input.Name, planName, input.CPF, hitEndpoint)
			return nil
		}

		var alt EligibilityAlternateResponse
		if altErr := json.Unmarshal(respBody, &alt); altErr == nil && len(alt.Afiliado) > 0 {
			log.Printf("🚀 [Doc24] Sucesso (formato afiliado)! Paciente %s vinculado ao plano '%s' (ID: %s) endpoint=%s", input.Name, planName, input.CPF, hitEndpoint)
			return nil
		}

		return fmt.Errorf("doc24 retornou estado=%d mensagem=%q body=%s", result.Estado, result.Mensaje, trimmedBody)
	}

	var alt EligibilityAlternateResponse
	if err := json.Unmarshal(respBody, &alt); err == nil && len(alt.Afiliado) > 0 {
		log.Printf("🚀 [Doc24] Sucesso (formato afiliado)! Paciente %s vinculado ao plano '%s' (ID: %s) endpoint=%s", input.Name, planName, input.CPF, hitEndpoint)
		return nil
	}

	return fmt.Errorf("doc24 retornou payload não reconhecido: %s", trimmedBody)

}

func normalizeDigits(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return nonDigitsRegex.ReplaceAllString(value, "")
}

func buildEligibilityCSV(input AffiliateInput) ([]byte, error) {
	var b bytes.Buffer
	w := csv.NewWriter(&b)

	headers := []string{
		"nombre",
		"apellido",
		"sexo",
		"fecha_nacimiento",
		"identificacion_tributaria",
		"nro_documento",
		"nro_documento_titular",
		"plan",
		"telefono_movil",
		"email",
		"empresa",
		"credencial",
		"fecha_alta",
		"fecha_baja",
	}

	if err := w.Write(headers); err != nil {
		return nil, err
	}

	row := []string{
		input.Nombre,
		input.Apellido,
		strings.ToUpper(strings.TrimSpace(input.Sexo)),
		input.FechaNacimiento,
		input.IdentificacaoTrib,
		input.NroDocumento,
		input.NroDocTitular,
		input.Plan,
		input.TelefonoMovil,
		input.Email,
		input.Empresa,
		input.Credencial,
		input.FechaAlta,
		"",
	}

	if err := w.Write(row); err != nil {
		return nil, err
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (c *Client) GetBeneficiaryID(cpf string) string {

	return cpf
}
