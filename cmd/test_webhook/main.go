// Simula um evento de webhook Asaas para testar o handler localmente.
//
// Uso com ID interno do banco (coluna "id" da tabela customers):
//   go run ./cmd/test_webhook --customer-id=7d9f3a8a-a920-4f75-9ede-83c40bd443d6
//
// Uso com gateway_id do Asaas (coluna "gateway_id", começa com cus_):
//   go run ./cmd/test_webhook --gateway-id=cus_000005370527
//
// Exemplos com flags opcionais:
//   go run ./cmd/test_webhook --customer-id=UUID --event=PAYMENT_CONFIRMED --status=CONFIRMED
//   go run ./cmd/test_webhook --customer-id=UUID --url=http://localhost:8080/webhook
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	customerID := flag.String("customer-id", "", "ID interno do banco (coluna 'id' da tabela customers, UUID)")
	gatewayID := flag.String("gateway-id", "", "Gateway customer ID do Asaas (coluna 'gateway_id', ex: cus_XXXXX)")
	paymentID := flag.String("payment-id", "", "ID do pagamento (gerado automaticamente se omitido)")
	event := flag.String("event", "PAYMENT_RECEIVED", "Tipo do evento Asaas")
	status := flag.String("status", "RECEIVED", "Status do pagamento")
	url := flag.String("url", "", "URL do webhook (padrão: http://localhost:<SERVER_PORT>/webhook)")
	token := flag.String("token", "", "Token de autenticação (padrão: ASAAS_WEBHOOK_SECRET do .env)")
	flag.Parse()

	customerRef := strings.TrimSpace(*gatewayID)
	if customerRef == "" {
		customerRef = strings.TrimSpace(*customerID)
	}
	if customerRef == "" {
		log.Fatal(`Informe --customer-id (UUID do banco) ou --gateway-id (cus_XXXX do Asaas).

Exemplos:
  go run ./cmd/test_webhook --customer-id=7d9f3a8a-a920-4f75-9ede-83c40bd443d6
  go run ./cmd/test_webhook --gateway-id=cus_000005370527`)
	}

	if *paymentID == "" {
		*paymentID = fmt.Sprintf("pay_sim_%d", os.Getpid())
	}

	if *token == "" {
		*token = os.Getenv("ASAAS_WEBHOOK_SECRET")
	}
	if *token == "" {
		log.Fatal("ASAAS_WEBHOOK_SECRET não configurado no .env e --token não fornecido")
	}

	if *url == "" {
		port := os.Getenv("SERVER_PORT")
		if port == "" {
			port = "8080"
		}
		*url = "http://localhost:" + port + "/webhook"
	}

	payload := map[string]interface{}{
		"event": strings.ToUpper(*event),
		"payment": map[string]interface{}{
			"id":       *paymentID,
			"customer": customerRef,
			"status":   strings.ToUpper(*status),
		},
	}

	body, _ := json.MarshalIndent(payload, "", "  ")

	fmt.Printf("→ Enviando webhook simulado para: %s\n", *url)
	fmt.Printf("→ Event: %s | Status: %s\n", strings.ToUpper(*event), strings.ToUpper(*status))
	fmt.Printf("→ Customer: %s\n", customerRef)
	fmt.Printf("→ Token: %s\n", maskToken(*token))
	fmt.Printf("→ Payload:\n%s\n\n", body)

	req, err := http.NewRequest("POST", *url, bytes.NewReader(body))
	if err != nil {
		log.Fatalf("Erro ao criar request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("asaasAccessToken", *token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Erro ao enviar request: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("← HTTP %d\n", resp.StatusCode)
	if len(respBody) > 0 {
		fmt.Printf("← Body: %s\n", respBody)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		fmt.Println("\n✅ Webhook aceito! Verifique os logs do servidor para confirmar a ativação.")
	case http.StatusUnauthorized:
		fmt.Println("\n❌ 401 — token não bate com ASAAS_WEBHOOK_SECRET no servidor.")
		fmt.Println("   Verifique se o servidor foi reiniciado após alterar o .env.")
	case http.StatusInternalServerError:
		fmt.Println("\n⚠️  500 — webhook chegou mas falhou na ativação (veja logs do servidor).")
	default:
		fmt.Printf("\n⚠️  Resposta inesperada: %d\n", resp.StatusCode)
	}
}

func maskToken(t string) string {
	if len(t) <= 8 {
		return "***"
	}
	return t[:6] + "***" + t[len(t)-4:]
}
