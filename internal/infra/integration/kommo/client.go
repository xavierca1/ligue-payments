package kommo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type Client struct {
	apiToken string
	baseURL  string
}

func NewClient() *Client {
	return &Client{
		apiToken: os.Getenv("KOMMO_API_TOKEN"),
		baseURL:  "https://liguemedicina.kommo.com/api/v4",
	}
}

func (c *Client) CreateLead(input CreateLeadInput) (int, error) {
	if c.apiToken == "" {
		log.Println("⚠️ Kommo: API_TOKEN não configurado")
		return 0, fmt.Errorf("kommo não configurado")
	}

	// Primeiro, criar ou buscar contato
	contactID, err := c.findOrCreateContact(input)
	if err != nil {
		return 0, fmt.Errorf("erro ao criar/buscar contato: %w", err)
	}

	// Agora criar o lead com o contato existente
	leadData := []map[string]interface{}{
		{
			"name":      fmt.Sprintf("%s - %s", input.CustomerName, input.PlanName),
			"status_id": 96648371,
			"price":     input.Price,
			"_embedded": map[string]interface{}{
				"tags": []map[string]interface{}{
					{"name": "pagamento_confirmado"},
				},
				"contacts": []map[string]interface{}{
					{"id": contactID},
				},
			},
		},
	}

	payload, _ := json.Marshal(leadData)
	req, _ := http.NewRequest("POST", c.baseURL+"/leads", bytes.NewBuffer(payload))
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

	var result struct {
		Embedded struct {
			Leads []struct {
				ID int `json:"id"`
			} `json:"leads"`
		} `json:"_embedded"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	if len(result.Embedded.Leads) == 0 {
		return 0, fmt.Errorf("lead não criado")
	}

	leadID := result.Embedded.Leads[0].ID
	log.Printf("✅ Kommo: Lead criado #%d para %s (%s)", leadID, input.CustomerName, input.PlanName)

	return leadID, nil
}

func (c *Client) findOrCreateContact(input CreateLeadInput) (int, error) {
	// Buscar contato por telefone
	contactID, err := c.findContactByPhone(input.Phone)
	if err == nil && contactID > 0 {
		log.Printf("📱 Kommo: Contato existente encontrado: %d", contactID)
		return contactID, nil
	}

	// Se não encontrou, criar novo contato
	return c.createContact(input)
}

func (c *Client) findContactByPhone(phone string) (int, error) {
	url := fmt.Sprintf("%s/contacts?query=%s", c.baseURL, phone)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	c.addAuthHeaders(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("erro ao buscar contato: %d", resp.StatusCode)
	}

	var result struct {
		Embedded struct {
			Contacts []struct {
				ID int `json:"id"`
			} `json:"contacts"`
		} `json:"_embedded"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	if len(result.Embedded.Contacts) > 0 {
		return result.Embedded.Contacts[0].ID, nil
	}

	return 0, fmt.Errorf("contato não encontrado")
}

func (c *Client) createContact(input CreateLeadInput) (int, error) {
	contactData := []map[string]interface{}{
		{
			"name": input.CustomerName,
			"custom_fields_values": []map[string]interface{}{
				{
					"field_code": "PHONE",
					"values": []map[string]interface{}{
						{"value": input.Phone, "enum_code": "WORK"},
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
	}

	payload, _ := json.Marshal(contactData)
	req, _ := http.NewRequest("POST", c.baseURL+"/contacts", bytes.NewBuffer(payload))
	c.addAuthHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("erro ao criar contato: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Embedded struct {
			Contacts []struct {
				ID int `json:"id"`
			} `json:"contacts"`
		} `json:"_embedded"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	if len(result.Embedded.Contacts) > 0 {
		contactID := result.Embedded.Contacts[0].ID
		log.Printf("✅ Kommo: Novo contato criado: %d", contactID)
		return contactID, nil
	}

	return 0, fmt.Errorf("erro ao obter ID do contato criado")
}

func (c *Client) addAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}
