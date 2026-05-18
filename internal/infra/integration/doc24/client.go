package doc24

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	httptrace "github.com/DataDog/dd-trace-go/contrib/net/http/v2"
	"github.com/xavierca1/ligue-payments/internal/infra/queue"
)

const (
	BaseURL = "https://api.doc24.com.ar/ws/api/v2"
	AuthURL = BaseURL + "/authentication"
	EligURL = BaseURL + "/portal/elegibilidad"
)

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
	// Doc24 rejeita HTTP/2 — desabilita via TLSNextProto vazio.
	transport := &http.Transport{
		TLSNextProto: make(map[string]func(string, *tls.Conn) http.RoundTripper),
	}
	return &Client{
		HTTPClient:   httptrace.WrapClient(&http.Client{Timeout: 30 * time.Second, Transport: transport}),
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
}

type afiliado struct {
	NroDocumento    string `json:"nro_documento"`
	Credencial      string `json:"credencial"`
	Apellido        string `json:"apellido"`
	FechaAlta       string `json:"fecha_alta"`
	FechaNacimiento string `json:"fecha_nacimiento"`
	NroDocTitular   string `json:"nro_documento_titular"`
	Sexo            string `json:"sexo"`
	Empresa         string `json:"empresa"`
	Nombre          string `json:"nombre"`
	Plan            string `json:"plan"`
	TelefonoMovil   string `json:"telefono_movil"`
	Email           string `json:"email"`
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
		var errBody bytes.Buffer
		errBody.ReadFrom(resp.Body)
		log.Printf("❌ [Doc24] Erro Auth: %s", errBody.String())
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
		exp = 3600
	}
	c.tokenExpiry = time.Now().Add(time.Duration(exp) * time.Second)

	log.Println("✅ [Doc24] Token renovado com sucesso!")
	return nil
}

func (c *Client) sendAfiliado(ctx context.Context, a afiliado) error {
	body, _ := json.Marshal(map[string]afiliado{"afiliado": a})

	req, err := http.NewRequestWithContext(ctx, "POST", EligURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	log.Printf("[Doc24] POST %s %s", EligURL, string(body))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("falha request doc24: %w", err)
	}
	defer resp.Body.Close()

	var respBuf bytes.Buffer
	respBuf.ReadFrom(resp.Body)
	log.Printf("[Doc24] Resposta HTTP %d: %s", resp.StatusCode, respBuf.String())

	if resp.StatusCode >= 400 {
		return fmt.Errorf("erro api doc24 (%d): %s", resp.StatusCode, respBuf.String())
	}

	var result struct {
		Estado  int    `json:"estado"`
		Mensaje string `json:"mensaje"`
	}
	if err := json.Unmarshal(respBuf.Bytes(), &result); err != nil {
		return fmt.Errorf("doc24 resposta inválida: %s", strings.TrimSpace(respBuf.String()))
	}
	if result.Estado == 1 && strings.TrimSpace(result.Mensaje) == "OK" {
		return nil
	}
	return fmt.Errorf("doc24 retornou estado=%d mensagem=%q", result.Estado, result.Mensaje)
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
	if strings.Contains(strings.ToLower(input.Gender), "fem") || strings.ToUpper(input.Gender) == "F" {
		sexo = "F"
	}

	planName := strings.TrimSpace(input.ProviderPlanCode)
	if planName == "" {
		log.Println("[Doc24] ProviderPlanCode veio vazio! Usando plano default.")
		planName = "ligue saude em dia individual"
	}

	today := time.Now().Format("2006-01-02")
	titularCPF := normalizeDigits(input.CPF)

	titular := afiliado{
		NroDocumento:    titularCPF,
		Credencial:      titularCPF,
		Apellido:        sobrenome,
		FechaAlta:       today,
		FechaNacimiento: input.BirthDate,
		NroDocTitular:   titularCPF,
		Sexo:            sexo,
		Empresa:         "Ligue_digital",
		Nombre:          nome,
		Plan:            planName,
		TelefonoMovil:   normalizeDigits(input.Phone),
		Email:           input.Email,
	}

	if err := c.sendAfiliado(ctx, titular); err != nil {
		return err
	}
	log.Printf("🚀 [Doc24] Titular %s vinculado ao plano '%s'", input.Name, planName)

	for _, dep := range input.Dependents {
		depParts := strings.SplitN(dep.Name, " ", 2)
		depNombre := depParts[0]
		depApellido := ""
		if len(depParts) > 1 {
			depApellido = depParts[1]
		}
		depSexo := "M"
		if dep.Gender == 2 {
			depSexo = "F"
		}

		dependent := afiliado{
			NroDocumento:    normalizeDigits(dep.CPF),
			Credencial:      normalizeDigits(dep.CPF),
			Apellido:        depApellido,
			FechaAlta:       today,
			FechaNacimiento: dep.BirthDate,
			NroDocTitular:   titularCPF,
			Sexo:            depSexo,
			Empresa:         "Ligue_digital",
			Nombre:          depNombre,
			Plan:            planName,
			TelefonoMovil:   normalizeDigits(input.Phone),
			Email:           input.Email,
		}

		if err := c.sendAfiliado(ctx, dependent); err != nil {
			log.Printf("⚠️ [Doc24] Falha ao vincular dependente %s: %v", dep.Name, err)
		} else {
			log.Printf("🚀 [Doc24] Dependente %s vinculado ao plano '%s'", dep.Name, planName)
		}
	}

	return nil
}

func normalizeDigits(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return nonDigitsRegex.ReplaceAllString(value, "")
}

func (c *Client) GetBeneficiaryID(cpf string) string {
	return cpf
}
