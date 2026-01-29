package doc24

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/xavierca1/ligue-payments/internal/infra/queue"
)

const (
	BaseURL = "https://tapi.doc24.com.ar/ws/api/v2"
	AuthURL = BaseURL + "/authentication" // Endpoint correto para Client Credentials
)

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
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
}

// --- STRUCTS (Endpoint de Elegibilidade / Afiliados) ---

type AffiliateInput struct {
	// Dados Pessoais
	Nombre          string `json:"nombre"`
	Apellido        string `json:"apellido"`
	Sexo            string `json:"sexo"`             // "M" ou "F"
	FechaNacimiento string `json:"fecha_nacimiento"` // YYYY-MM-DD

	// Documentos
	IdentificacaoTrib string `json:"identificacion_tributaria"` // CPF
	NroDocumento      string `json:"nro_documento"`             // CPF
	NroDocTitular     string `json:"nro_documento_titular"`     // CPF

	// Vínculo com o Plano
	Plan       string `json:"plan"`       // Nome do plano (vem do banco)
	Empresa    string `json:"empresa"`    // Nome da empresa na Doc24
	Credencial string `json:"credencial"` // Usamos CPF
	FechaAlta  string `json:"fecha_alta"` // Data de hoje

	// Contato
	TelefonoMovil string `json:"telefono_movil"`
	Email         string `json:"email"`
}

type EligibilityResponse struct {
	Estado  int    `json:"estado"`  // 1 = Sucesso
	Mensaje string `json:"mensaje"` // "OK"
}

// --- MÉTODOS ---

// EnsureAuthenticated: Garante Token via Client Credentials
func (c *Client) EnsureAuthenticated(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Se tem token válido, usa ele
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
		// Lê o body para debug
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

// CreateBeneficiary: Cria ou Atualiza via Elegibilidade (Upsert)
func (c *Client) CreateBeneficiary(ctx context.Context, input queue.ActivationPayload) error {
	// 1. Auth
	if err := c.EnsureAuthenticated(ctx); err != nil {
		return err
	}

	// 2. Prepara Dados
	parts := strings.SplitN(input.Name, " ", 2)
	nome := parts[0]
	sobrenome := ""
	if len(parts) > 1 {
		sobrenome = parts[1]
	}

	// Sexo
	sexo := "M"
	if input.Gender == "0" || strings.ToUpper(input.Gender) == "F" {
		sexo = "F"
	}

	planName := input.ProviderPlanCode

	if planName == "" {
		log.Println("[Doc24] ProviderPlanCode veio vazio! Usando plano default.")
		planName = "ligue saude em dia individual"
	}

	today := time.Now().Format("2006-01-02")

	// 3. Monta Payload
	payload := AffiliateInput{
		Nombre:            nome,
		Apellido:          sobrenome,
		Sexo:              sexo,
		FechaNacimiento:   input.BirthDate,
		IdentificacaoTrib: input.CPF,
		NroDocumento:      input.CPF,
		NroDocTitular:     input.CPF,

		Plan:       planName,
		Empresa:    "Ag Med", // ⚠️ CONFIRME SE É 'Ag Med' ou 'Ligue Med'
		Credencial: input.CPF,
		FechaAlta:  today,

		TelefonoMovil: input.Phone,
		Email:         input.Email,
	}

	jsonBody, _ := json.Marshal(payload)

	// 4. Request
	url := BaseURL + "/portal/elegibilidad"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("falha request doc24: %w", err)
	}
	defer resp.Body.Close()

	// 5. Validação
	if resp.StatusCode >= 400 {
		var errBody bytes.Buffer
		errBody.ReadFrom(resp.Body)
		// Loga o erro exato que a Doc24 devolveu
		return fmt.Errorf("erro api doc24 (%d): %s", resp.StatusCode, errBody.String())
	}

	// Tenta decodar resposta (opcional, só pra logar bonito)
	var result EligibilityResponse
	json.NewDecoder(resp.Body).Decode(&result)

	log.Printf("🚀 [Doc24] Sucesso! Paciente %s vinculado ao plano '%s'", input.Name, planName)
	return nil
}
