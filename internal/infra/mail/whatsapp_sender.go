package mail

import (
	"log"

	"github.com/xavierca1/ligue-payments/internal/infra/integration/kommo"
)

type WhatsAppSender struct {
	client *kommo.Client
}

func NewWhatsAppSender(client *kommo.Client) *WhatsAppSender {
	return &WhatsAppSender{
		client: client,
	}
}

func (s *WhatsAppSender) SendWelcome(phone, name, planName, templateID string) error {
	if phone == "" || name == "" || planName == "" {
		log.Printf("⚠️ WhatsApp: Dados incompletos para envio (phone: %s, name: %s, plan: %s)", phone, name, planName)
		return nil
	}

	input := kommo.SendWhatsAppInput{
		PhoneNumber: phone,
		Name:        name,
		PlanName:    planName,
	}

	if err := s.client.SendWhatsAppMessage(input); err != nil {
		log.Printf("⚠️ WhatsApp (Kommo): Falha ao enviar para %s: %v", phone, err)
		return nil
	}

	return nil
}
