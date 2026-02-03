package whatsapp

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
	accessToken string
	phoneID     string
	baseURL     string
}





func NewClient() *Client {
	return &Client{
		accessToken: os.Getenv("WHATSAPP_ACCESS_TOKEN"),
		phoneID:     os.Getenv("WHATSAPP_PHONE_ID"),
		baseURL:     "https://graph.instagram.com/v18.0",
	}
}









func (c *Client) SendMessage(input SendMessageInput) error {
	if c.accessToken == "" || c.phoneID == "" {
		log.Println("⚠️ WhatsApp: ACCESS_TOKEN ou PHONE_ID não configurados")
		return fmt.Errorf("whatsapp não configurado")
	}



	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                input.PhoneNumber,
		"type":              "template",
		"template": map[string]interface{}{
			"name": input.TemplateName,
			"language": map[string]string{
				"code": "pt_BR",
			},
			"components": []map[string]interface{}{
				{
					"type":       "body",
					"parameters": convertParametersToAPI(input.Parameters),
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("❌ WhatsApp: Erro ao serializar payload: %v", err)
		return err
	}


	url := fmt.Sprintf("%s/%s/messages", c.baseURL, c.phoneID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		log.Printf("❌ WhatsApp: Erro ao criar requisição: %v", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("❌ WhatsApp: Erro ao enviar mensagem: %v", err)
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)


	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Printf("❌ WhatsApp: API retornou status %d: %s", resp.StatusCode, string(respBody))
		return fmt.Errorf("whatsapp api error: %d", resp.StatusCode)
	}

	var result SendMessageResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Printf("❌ WhatsApp: Erro ao parsear resposta: %v", err)
		return err
	}

	if result.Error != nil {
		log.Printf("❌ WhatsApp: Erro na API: %s (Code: %d)", result.Error.Message, result.Error.Code)
		return fmt.Errorf("whatsapp: %s", result.Error.Message)
	}

	log.Printf("✅ WhatsApp: Mensagem enviada para %s", input.PhoneNumber)
	return nil
}


func convertParametersToAPI(params []string) []map[string]string {
	result := make([]map[string]string, 0, len(params))
	for _, param := range params {
		result = append(result, map[string]string{
			"type": "text",
			"text": param,
		})
	}
	return result
}
