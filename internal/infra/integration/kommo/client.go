package kommo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type Client struct {
	accountID string
	apiToken  string
	baseURL   string
}

func NewClient() *Client {
	return &Client{
		accountID: os.Getenv("KOMMO_ACCOUNT_ID"),
		apiToken:  os.Getenv("KOMMO_API_TOKEN"),
		baseURL:   "https://api.amocrm.com/v4",
	}
}

func (c *Client) SendWhatsAppMessage(input SendWhatsAppInput) error {
	if c.accountID == "" || c.apiToken == "" {
		log.Println("⚠️ Kommo: ACCOUNT_ID ou API_TOKEN não configurados")
		return fmt.Errorf("kommo não configurado")
	}

	phone := strings.ReplaceAll(input.PhoneNumber, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")

	contactID, err := c.findOrCreateContact(phone, input.Name)
	if err != nil {
		log.Printf("❌ Kommo: Erro ao buscar/criar contato: %v", err)
		return err
	}

	message := fmt.Sprintf("Olá %s, bem-vindo! 🎉\n\nVocê adquiriu o plano %s.\n\nAcesse seu dashboard agora!", input.Name, input.PlanName)
	if input.Message != "" {
		message = input.Message
	}
	if err := c.sendMessage(contactID, phone, message); err != nil {
		log.Printf("❌ Kommo: Erro ao enviar mensagem: %v", err)
		return err
	}

	log.Printf("✅ Kommo: Mensagem WhatsApp enviada para %s (%s)", input.Name, phone)
	return nil
}

func (c *Client) findOrCreateContact(phone, name string) (int, error) {
	contactID, err := c.findContactByPhone(phone)
	if err == nil && contactID > 0 {
		log.Printf("📱 Kommo: Contato encontrado: %d", contactID)
		return contactID, nil
	}

	return c.createContact(phone, name)
}

func (c *Client) findContactByPhone(phone string) (int, error) {
	url := fmt.Sprintf("%s/contacts?filter[phone]=%s&limit=1", c.baseURL, phone)

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
		return 0, fmt.Errorf("kommo api error: %d", resp.StatusCode)
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

func (c *Client) createContact(phone, name string) (int, error) {
	url := fmt.Sprintf("%s/contacts", c.baseURL)

	payload := map[string]interface{}{
		"first_name": name,
		"custom_fields_values": []map[string]interface{}{
			{
				"field_id": 123456, // ID do campo de telefone no Kommo (configurável)
				"values": []map[string]interface{}{
					{
						"value": phone,
					},
				},
			},
		},
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}

	c.addAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("kommo api error: %d - %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Embedded struct {
			Contacts []struct {
				ID int `json:"id"`
			} `json:"contacts"`
		} `json:"_embedded"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, err
	}

	if len(result.Embedded.Contacts) > 0 {
		return result.Embedded.Contacts[0].ID, nil
	}

	return 0, fmt.Errorf("erro ao criar contato")
}

func (c *Client) sendMessage(contactID int, phone, message string) error {
	url := fmt.Sprintf("%s/messages", c.baseURL)

	payload := map[string]interface{}{
		"to":           contactID,
		"service_code": "whatsapp",
		"text":         message,
		"phone":        phone,
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	c.addAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("kommo api error: %d - %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (c *Client) addAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiToken))
	req.Header.Set("X-Amocrm-Account-Id", c.accountID)
}
