package kommo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

type Client struct {
	apiToken           string
	baseURL            string
	pipelineB2CID      int
	productFieldID     int
	originFieldID      int
	statusEmFechamento int
}

func NewClient() *Client {
	pipelineID, _ := strconv.Atoi(os.Getenv("KOMMO_PIPELINE_B2C_ID"))
	productFieldID, _ := strconv.Atoi(os.Getenv("KOMMO_FIELD_PRODUTO_ID"))
	originFieldID, _ := strconv.Atoi(os.Getenv("KOMMO_FIELD_ORIGEM_ID"))

	return &Client{
		apiToken:           os.Getenv("KOMMO_API_TOKEN"),
		baseURL:            "https://liguemedicina.kommo.com/api/v4",
		pipelineB2CID:      pipelineID,
		productFieldID:     productFieldID,
		originFieldID:      originFieldID,
		statusEmFechamento: 142,
	}
}

func (c *Client) CreateLead(input CreateLeadInput) (int, error) {
	if c.apiToken == "" {
		log.Println("⚠️ Kommo: API_TOKEN não configurado")
		return 0, fmt.Errorf("kommo não configurado")
	}

	phoneCleaned := cleanPhone(input.Phone)
	phoneFormatted := formatPhoneWithCountryCode(phoneCleaned)
	productName := determineProductName(input.PlanName)

	originChannel := input.Origin
	if originChannel == "" {
		originChannel = "Outros"
	}

	lead := map[string]interface{}{
		"name":        input.CustomerName,
		"pipeline_id": c.pipelineB2CID,
		"status_id":   c.statusEmFechamento,
		"_embedded": map[string]interface{}{
			"tags": []map[string]interface{}{
				{"name": "pagamento_confirmado"},
			},
			"contacts": []map[string]interface{}{
				{
					"first_name": input.CustomerName,
					"custom_fields_values": []map[string]interface{}{
						{
							"field_code": "PHONE",
							"values": []map[string]interface{}{
								{"value": phoneFormatted, "enum_code": "WORK"},
							},
						},
						{
							"field_code": "EMAIL",
							"values": []map[string]interface{}{
								{"value": input.Email, "enum_code": "WORK"},
							},
						},
					},
				},
			},
		},
	}

	customFields := buildCustomFields(c, productName, originChannel)
	customFields = append(customFields, map[string]interface{}{
		"field_code": "CHECKOUT_API",
		"values": []map[string]interface{}{
			{"value": true},
		},
	})

	lead["custom_fields_values"] = customFields
	leadPayload := []map[string]interface{}{lead}
	payload, _ := json.Marshal(leadPayload)

	url := fmt.Sprintf("%s/leads/complex", c.baseURL)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	c.addAuthHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("erro ao criar lead: %d - %s", resp.StatusCode, string(body))
	}

	var result []struct {
		ID int `json:"id"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	if len(result) == 0 {
		return 0, fmt.Errorf("lead não criado")
	}

	leadID := result[0].ID
	log.Printf("✅ Kommo: Lead B2C criado #%d - %s | Produto: %s | Origem: %s",
		leadID, input.CustomerName, productName, originChannel)

	return leadID, nil
}

func cleanPhone(phone string) string {
	cleaned := ""
	for _, char := range phone {
		if char >= '0' && char <= '9' {
			cleaned += string(char)
		}
	}
	return cleaned
}

func formatPhoneWithCountryCode(phone string) string {
	cleaned := cleanPhone(phone)

	if len(cleaned) > 10 && cleaned[:2] == "55" {
		cleaned = cleaned[2:]
	}

	if len(cleaned) == 11 {
		cleaned = cleaned[:2] + cleaned[3:]
	}

	return "+55" + cleaned
}

func determineProductName(planName string) string {
	planLower := ""
	for _, char := range planName {
		if char >= 'A' && char <= 'Z' {
			planLower += string(char + 32)
		} else {
			planLower += string(char)
		}
	}

	if contains(planLower, "odonto") || contains(planLower, "dental") {
		return "Plano Odontológico"
	}

	return "Telemedicina"
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func buildCustomFields(c *Client, productName, originChannel string) []map[string]interface{} {
	fields := []map[string]interface{}{}

	if c.productFieldID > 0 && productName != "" {
		fields = append(fields, map[string]interface{}{
			"field_id": c.productFieldID,
			"values": []map[string]interface{}{
				{"value": productName},
			},
		})
	}

	return fields
}

func (c *Client) addAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")
}
