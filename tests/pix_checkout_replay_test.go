package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

type replayCustomer struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	CPF             string `json:"cpf_cnpj"`
	Phone           string `json:"phone"`
	BirthDate       string `json:"birth_date"`
	Gender          int    `json:"gender"`
	PlanID          string `json:"plan_id"`
	Street          string `json:"street"`
	Number          string `json:"number"`
	Complement      string `json:"complement"`
	District        string `json:"district"`
	City            string `json:"city"`
	State           string `json:"state"`
	ZipCode         string `json:"zip_code"`
	TermsAccepted   bool   `json:"terms_accepted"`
	TermsAcceptedAt string `json:"terms_accepted_at"`
	TermsVersion    string `json:"terms_version"`
}

type checkoutReplayInput struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	CPF             string `json:"cpf"`
	Phone           string `json:"phone"`
	BirthDate       string `json:"birth_date"`
	Gender          string `json:"gender"`
	PlanID          string `json:"plan_id"`
	PaymentMethod   string `json:"payment_method"`
	Street          string `json:"street"`
	Number          string `json:"number"`
	Complement      string `json:"complement"`
	District        string `json:"district"`
	City            string `json:"city"`
	State           string `json:"state"`
	ZipCode         string `json:"zip_code"`
	TermsAccepted   bool   `json:"terms_accepted"`
	TermsAcceptedAt string `json:"terms_accepted_at"`
	TermsVersion    string `json:"terms_version"`
}

func TestReplayPixCheckoutFromApprovedCustomers(t *testing.T) {
	if strings.ToLower(os.Getenv("RUN_PIX_REPLAY")) != "true" {
		t.Skip("defina RUN_PIX_REPLAY=true para executar o replay PIX contra /checkout")
	}

	apiBaseURL := strings.TrimSpace(os.Getenv("PIX_REPLAY_API_BASE_URL"))
	if apiBaseURL == "" {
		apiBaseURL = "http://localhost:8081"
	}

	artifactPath := strings.TrimSpace(os.Getenv("PIX_REPLAY_CUSTOMERS_FILE"))
	if artifactPath == "" {
		artifactPath = resolveArtifactPath()
	}

	uniquify := strings.ToLower(strings.TrimSpace(os.Getenv("PIX_REPLAY_UNIQUIFY")))
	if uniquify == "" {
		uniquify = "false"
	}

	raw, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("erro ao ler artifact %s: %v", artifactPath, err)
	}

	var customers []replayCustomer
	if err := json.Unmarshal(raw, &customers); err != nil {
		t.Fatalf("erro ao parsear artifact JSON: %v", err)
	}

	if len(customers) == 0 {
		t.Fatalf("nenhum cliente no artifact: %s", artifactPath)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	checkoutURL := strings.TrimRight(apiBaseURL, "/") + "/checkout"

	statusCount := map[int]int{}
	bodySamples := map[int]string{}

	for i, c := range customers {
		payload := checkoutReplayInput{
			Name:            c.Name,
			Email:           c.Email,
			CPF:             digitsOnly(c.CPF),
			Phone:           digitsOnly(c.Phone),
			BirthDate:       c.BirthDate,
			Gender:          strconv.Itoa(c.Gender),
			PlanID:          c.PlanID,
			PaymentMethod:   "PIX",
			Street:          c.Street,
			Number:          c.Number,
			Complement:      c.Complement,
			District:        c.District,
			City:            c.City,
			State:           c.State,
			ZipCode:         digitsOnly(c.ZipCode),
			TermsAccepted:   true,
			TermsAcceptedAt: time.Now().UTC().Format(time.RFC3339),
			TermsVersion:    firstNonEmpty(c.TermsVersion, "1.0"),
		}

		if uniquify == "true" {
			payload.Email = uniqueEmail(payload.Email, i)
			payload.CPF = uniqueCPFFromSeed(i + 1)
		}

		b, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("erro ao serializar payload index=%d: %v", i, err)
		}

		req, err := http.NewRequest(http.MethodPost, checkoutURL, bytes.NewReader(b))
		if err != nil {
			t.Fatalf("erro ao criar request index=%d: %v", i, err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("erro ao chamar checkout index=%d: %v", i, err)
		}

		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		statusCount[resp.StatusCode]++
		if _, ok := bodySamples[resp.StatusCode]; !ok {
			bodySamples[resp.StatusCode] = truncate(string(respBody), 240)
		}

		if resp.StatusCode >= 500 {
			t.Fatalf("checkout retornou %d no index=%d body=%s", resp.StatusCode, i, string(respBody))
		}
	}

	t.Logf("PIX replay concluído em %s", checkoutURL)
	for code, count := range statusCount {
		t.Logf("status %d => %d", code, count)
		if sample := bodySamples[code]; sample != "" {
			t.Logf("sample[%d]: %s", code, sample)
		}
	}

	if statusCount[http.StatusCreated] == 0 {
		t.Log("nenhum 201 criado neste replay (ambiente pode estar sem tabela/plano), mas o tráfego foi gerado para observabilidade")
	}
}

func firstNonEmpty(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}

func digitsOnly(v string) string {
	var b strings.Builder
	b.Grow(len(v))
	for _, r := range v {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func uniqueEmail(email string, idx int) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Sprintf("pix-replay+%d@example.com", idx+1)
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return fmt.Sprintf("pix-replay+%d@example.com", idx+1)
	}
	return fmt.Sprintf("%s+pixreplay%d@%s", parts[0], idx+1, parts[1])
}

func uniqueCPFFromSeed(seed int) string {
	base := fmt.Sprintf("%09d", (seed*7919)%1000000000)
	d1 := cpfDigit(base, 10)
	d2 := cpfDigit(base+strconv.Itoa(d1), 11)
	return base + strconv.Itoa(d1) + strconv.Itoa(d2)
}

func cpfDigit(numbers string, factor int) int {
	sum := 0
	for _, r := range numbers {
		sum += int(r-'0') * factor
		factor--
	}
	d := (sum * 10) % 11
	if d == 10 {
		return 0
	}
	return d
}

func truncate(v string, max int) string {
	if len(v) <= max {
		return v
	}
	if max <= 3 {
		return v[:max]
	}
	return v[:max-3] + "..."
}

func resolveArtifactPath() string {
	candidates := []string{
		filepath.FromSlash("tests/artifacts/customers.json"),
		filepath.FromSlash("artifacts/customers.json"),
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	return candidates[0]
}
