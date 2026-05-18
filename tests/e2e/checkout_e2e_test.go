//go:build e2e

// Testes de integração end-to-end contra o servidor real (Asaas Sandbox).
//
// PRÉ-REQUISITOS:
//   - Servidor rodando: go run ./cmd/api
//   - Variável TEST_CPF com um CPF real válido na Receita Federal
//   - Variável TEST_EMAIL com um email válido (receberá boas-vindas)
//
// COMO RODAR (um por vez):
//
//	TEST_CPF=00000000000 TEST_EMAIL=voce@email.com \
//	  go test -tags e2e -run TestE2E_SaudeEmDia_PIX -v ./tests/e2e/
//
// LIMPEZA após cada teste (execute no Supabase SQL Editor ou psql):
//
//	DELETE FROM subscriptions WHERE customer_id = (SELECT id FROM customers WHERE cpf_cnpj = 'SEU_CPF');
//	DELETE FROM dependents    WHERE customer_cpf = 'SEU_CPF';
//	DELETE FROM customers     WHERE cpf_cnpj     = 'SEU_CPF';

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Configuração
// ---------------------------------------------------------------------------

func baseURL() string {
	if u := os.Getenv("TEST_BASE_URL"); u != "" {
		return u
	}
	return "http://localhost:8080"
}

func testCPF() string {
	if c := os.Getenv("TEST_CPF"); c != "" {
		return c
	}
	return "PREENCHA_TEST_CPF" // ex: 52998224725
}

func testEmail() string {
	if e := os.Getenv("TEST_EMAIL"); e != "" {
		return e
	}
	return "teste@liguemedicina.com"
}

// Cartão de teste Asaas Sandbox (Luhn válido)
const (
	testCardNumber = "4532015112830366"
	testCardHolder = "JOSE SILVA TESTE"
	testCardMonth  = "12"
	testCardYear   = "2030"
	testCardCVV    = "123"
)

// ---------------------------------------------------------------------------
// Planos (extraídos do app.html)
// ---------------------------------------------------------------------------

type planCfg struct {
	ID     string
	Name   string
	Family bool
}

var (
	planSaudeEmDia   = planCfg{"230265d3-3d12-4530-b896-147387003271", "Ligue Saúde em Dia", false}
	planVidaPlena    = planCfg{"ca7a08cd-018e-4aec-8bb1-715ce54eb6fd", "Ligue Vida Plena", false}
	planViverBem     = planCfg{"d473ee83-bf52-4927-82c6-d0128e030c3c", "Ligue Viver Bem", false}
	planMaisCuidado  = planCfg{"8f305d38-4c1e-435a-a57b-7c77e2a5f141", "Ligue Mais Cuidado", true}
	planCuidadoTotal = planCfg{"255f7631-751d-4edd-a165-65a0c55824eb", "Ligue Cuidado Total", true}
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type checkoutReq struct {
	Name          string      `json:"name"`
	Email         string      `json:"email"`
	CPF           string      `json:"cpf"`
	Phone         string      `json:"phone"`
	BirthDate     string      `json:"birth_date"`
	Gender        string      `json:"gender"`
	MaritalStatus string      `json:"marital_status"`
	Street        string      `json:"street"`
	Number        string      `json:"number"`
	Complement    string      `json:"complement"`
	District      string      `json:"district"`
	City          string      `json:"city"`
	State         string      `json:"state"`
	ZipCode       string      `json:"zip_code"`
	PlanID        string      `json:"plan_id"`
	PaymentMethod string      `json:"payment_method"`
	CardHolder    string      `json:"card_holder,omitempty"`
	CardNumber    string      `json:"card_number,omitempty"`
	CardMonth     string      `json:"card_month,omitempty"`
	CardYear      string      `json:"card_year,omitempty"`
	CardCVV       string      `json:"card_cvv,omitempty"`
	TermsAccepted bool        `json:"terms_accepted"`
	TermsAcceptedAt string   `json:"terms_accepted_at"`
	TermsVersion  string      `json:"terms_version"`
	Dependents    []dependent `json:"dependents,omitempty"`
}

type dependent struct {
	Name      string `json:"name"`
	CPF       string `json:"cpf"`
	BirthDate string `json:"birth_date"`
	Gender    string `json:"gender"`
	Kinship   string `json:"kinship"`
}

type checkoutResp struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Msg          string `json:"msg"`
	PixCode      string `json:"pix_code"`
	PixQRCodeURL string `json:"pix_qr_code_url"`
	Error        string `json:"error"`
	Message      string `json:"message"`
}

func basePayload(plan planCfg, method string) checkoutReq {
	p := checkoutReq{
		Name:            "Jose Silva Teste",
		Email:           testEmail(),
		CPF:             testCPF(),
		Phone:           "11999999999",
		BirthDate:       "1990-05-15",
		Gender:          "1",
		MaritalStatus:   "solteiro",
		Street:          "Rua das Flores",
		Number:          "123",
		Complement:      "Apto 1",
		District:        "Centro",
		City:            "São Paulo",
		State:           "SP",
		ZipCode:         "01310100",
		PlanID:          plan.ID,
		PaymentMethod:   method,
		TermsAccepted:   true,
		TermsAcceptedAt: time.Now().UTC().Format(time.RFC3339),
		TermsVersion:    "v1",
	}

	if method == "CREDIT_CARD" {
		p.CardHolder = testCardHolder
		p.CardNumber = testCardNumber
		p.CardMonth = testCardMonth
		p.CardYear = testCardYear
		p.CardCVV = testCardCVV
	}

	if plan.Family {
		p.Dependents = []dependent{
			{
				Name:      "Maria Silva Dependente",
				CPF:       "PREENCHA_CPF_DEPENDENTE", // CPF válido diferente do titular
				BirthDate: "2000-03-20",
				Gender:    "2",
				Kinship:   "CONJUGE",
			},
		}
	}

	return p
}

func doCheckout(t *testing.T, plan planCfg, method string) checkoutResp {
	t.Helper()

	payload := basePayload(plan, method)
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	url := baseURL() + "/checkout"
	t.Logf("→ POST %s | plano=%q método=%s", url, plan.Name, method)

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	require.NoError(t, err, "servidor não respondeu — certifique-se de que está rodando")
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	t.Logf("← HTTP %d | body=%s", resp.StatusCode, string(raw))

	var result checkoutResp
	require.NoError(t, json.Unmarshal(raw, &result))

	return result
}

func printCleanup(t *testing.T) {
	t.Helper()
	cpf := testCPF()
	t.Logf("\n╔══════════════════════════════════════════════════════════════╗")
	t.Logf("║  LIMPEZA — execute no Supabase SQL Editor antes do próximo  ║")
	t.Logf("╠══════════════════════════════════════════════════════════════╣")
	t.Logf("║  DELETE FROM subscriptions                                   ║")
	t.Logf("║    WHERE customer_id = (                                     ║")
	t.Logf("║      SELECT id FROM customers WHERE cpf_cnpj = '%s');  ║", cpf)
	t.Logf("║  DELETE FROM dependents WHERE customer_cpf = '%s';     ║", cpf)
	t.Logf("║  DELETE FROM customers  WHERE cpf_cnpj     = '%s';     ║", cpf)
	t.Logf("╚══════════════════════════════════════════════════════════════╝\n")
}

// ---------------------------------------------------------------------------
// ✅ Plano 1 — Ligue Saúde em Dia (Individual, R$39,90)
// ---------------------------------------------------------------------------

func TestE2E_SaudeEmDia_PIX(t *testing.T) {
	result := doCheckout(t, planSaudeEmDia, "PIX")
	defer printCleanup(t)

	assert.NotEmpty(t, result.ID, "customer_id não pode ser vazio")
	assert.Equal(t, "WAITING_PAYMENT", result.Status)
	assert.NotEmpty(t, result.PixCode, "pix_code esperado")
	assert.NotEmpty(t, result.PixQRCodeURL, "pix_qr_code_url esperado")

	t.Logf("✅ customer_id=%s  pix_code_len=%d", result.ID, len(result.PixCode))
}

func TestE2E_SaudeEmDia_Card(t *testing.T) {
	result := doCheckout(t, planSaudeEmDia, "CREDIT_CARD")
	defer printCleanup(t)

	assert.NotEmpty(t, result.ID)
	assert.Empty(t, result.Error, fmt.Sprintf("erro inesperado: %s", result.Message))

	t.Logf("✅ customer_id=%s  status=%s", result.ID, result.Status)
}

// ---------------------------------------------------------------------------
// ✅ Plano 2 — Ligue Vida Plena (Individual, R$47,66)
// ---------------------------------------------------------------------------

func TestE2E_VidaPlena_PIX(t *testing.T) {
	result := doCheckout(t, planVidaPlena, "PIX")
	defer printCleanup(t)

	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "WAITING_PAYMENT", result.Status)
	assert.NotEmpty(t, result.PixCode)

	t.Logf("✅ customer_id=%s  pix_code_len=%d", result.ID, len(result.PixCode))
}

func TestE2E_VidaPlena_Card(t *testing.T) {
	result := doCheckout(t, planVidaPlena, "CREDIT_CARD")
	defer printCleanup(t)

	assert.NotEmpty(t, result.ID)
	assert.Empty(t, result.Error, fmt.Sprintf("erro inesperado: %s", result.Message))

	t.Logf("✅ customer_id=%s  status=%s", result.ID, result.Status)
}

// ---------------------------------------------------------------------------
// ✅ Plano 3 — Ligue Viver Bem (Individual, R$69,90)
// ---------------------------------------------------------------------------

func TestE2E_ViverBem_PIX(t *testing.T) {
	result := doCheckout(t, planViverBem, "PIX")
	defer printCleanup(t)

	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "WAITING_PAYMENT", result.Status)
	assert.NotEmpty(t, result.PixCode)

	t.Logf("✅ customer_id=%s  pix_code_len=%d", result.ID, len(result.PixCode))
}

func TestE2E_ViverBem_Card(t *testing.T) {
	result := doCheckout(t, planViverBem, "CREDIT_CARD")
	defer printCleanup(t)

	assert.NotEmpty(t, result.ID)
	assert.Empty(t, result.Error, fmt.Sprintf("erro inesperado: %s", result.Message))

	t.Logf("✅ customer_id=%s  status=%s", result.ID, result.Status)
}

// ---------------------------------------------------------------------------
// ✅ Plano 4 — Ligue Mais Cuidado (Familiar, R$55,27 + 1 dependente)
// ---------------------------------------------------------------------------

func TestE2E_MaisCuidado_PIX(t *testing.T) {
	result := doCheckout(t, planMaisCuidado, "PIX")
	defer printCleanup(t)

	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "WAITING_PAYMENT", result.Status)
	assert.NotEmpty(t, result.PixCode)

	t.Logf("✅ customer_id=%s  pix_code_len=%d", result.ID, len(result.PixCode))
}

func TestE2E_MaisCuidado_Card(t *testing.T) {
	result := doCheckout(t, planMaisCuidado, "CREDIT_CARD")
	defer printCleanup(t)

	assert.NotEmpty(t, result.ID)
	assert.Empty(t, result.Error, fmt.Sprintf("erro inesperado: %s", result.Message))

	t.Logf("✅ customer_id=%s  status=%s", result.ID, result.Status)
}

// ---------------------------------------------------------------------------
// ✅ Plano 5 — Ligue Cuidado Total (Familiar, R$60,00 + 1 dependente)
// ---------------------------------------------------------------------------

func TestE2E_CuidadoTotal_PIX(t *testing.T) {
	result := doCheckout(t, planCuidadoTotal, "PIX")
	defer printCleanup(t)

	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "WAITING_PAYMENT", result.Status)
	assert.NotEmpty(t, result.PixCode)

	t.Logf("✅ customer_id=%s  pix_code_len=%d", result.ID, len(result.PixCode))
}

func TestE2E_CuidadoTotal_Card(t *testing.T) {
	result := doCheckout(t, planCuidadoTotal, "CREDIT_CARD")
	defer printCleanup(t)

	assert.NotEmpty(t, result.ID)
	assert.Empty(t, result.Error, fmt.Sprintf("erro inesperado: %s", result.Message))

	t.Logf("✅ customer_id=%s  status=%s", result.ID, result.Status)
}

// ---------------------------------------------------------------------------
// 🔒 Guarda-chuva: bloquear CPF já ATIVO
// ---------------------------------------------------------------------------

// TestE2E_BloqueioAtivo valida que um customer ativo não consegue fazer novo checkout.
// Rode DEPOIS de qualquer teste de cartão que já retornou ACTIVE,
// SEM fazer a limpeza antes.
func TestE2E_BloqueioAtivo(t *testing.T) {
	result := doCheckout(t, planSaudeEmDia, "PIX")

	assert.Equal(t, "CUSTOMER_ALREADY_ACTIVE", result.Error,
		"esperado CUSTOMER_ALREADY_ACTIVE para CPF já ativo")
	t.Logf("✅ bloqueio confirmado: %s", result.Message)
}
