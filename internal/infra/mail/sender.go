package mail

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/xavierca1/ligue-payments/internal/entity"
	"gopkg.in/gomail.v2"
)

func firstName(fullName string) string {
	trimmed := strings.TrimSpace(fullName)
	if trimmed == "" {
		return "Cliente"
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return "Cliente"
	}
	return parts[0]
}

func NewEmailSender(host string, port int, user, password string) *EmailSender {
	host = strings.TrimSpace(host)
	user = strings.TrimSpace(user)
	password = strings.TrimSpace(password)

	from := user
	if from == "" {
		from = "no-reply@liguemedicina.com"
	}

	return &EmailSender{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		From:     from,
	}
}

func NewEmailSenderWithFrom(host string, port int, user, password, from string) *EmailSender {
	host = strings.TrimSpace(host)
	user = strings.TrimSpace(user)
	password = strings.TrimSpace(password)
	from = strings.TrimSpace(from)

	if from == "" {
		from = user
	}

	if from == "" {
		from = "no-reply@liguemedicina.com"
	}

	return &EmailSender{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		From:     from,
	}
}

func (s *EmailSender) SendWelcome(to, name, productName, pdfLink string) error {
	cardNumber := ""
	if strings.TrimSpace(pdfLink) != "" {
		cardNumber = pdfLink
	}

	return s.sendWelcomeInternal(to, name, productName, pdfLink, cardNumber, nil, nil)
}

func (s *EmailSender) SendWelcomeEmailWithContractAndDependents(name, email, cpf, planName, providerID string, dependents []*entity.Dependent, contractPDF []byte) error {
	cardNumber := strings.TrimSpace(providerID)
	if cardNumber == "" {
		cardNumber = strings.TrimSpace(cpf)
	}
	if strings.TrimSpace(planName) == "" {
		planName = "Ligue Medicina"
	}
	return s.sendWelcomeInternal(email, name, planName, "", cardNumber, dependents, contractPDF)
}

func (s *EmailSender) sendWelcomeInternal(to, name, productName, pdfLink, cardNumber string, dependents []*entity.Dependent, contractPDF []byte) error {
	data := WelcomeEmailData{
		Name:        name,
		FirstName:   firstName(name),
		ProductName: productName,
		PDFLink:     pdfLink,
		PortalURL:   "https://app.liguemedicina.com.br",
		WhatsAppURL: "https://wa.me/5561999999999",
	}

	tmplPath := filepath.Join("templates", "welcome.html")
	t, err := template.ParseFiles(tmplPath)
	if err != nil {
		return fmt.Errorf("erro ao ler template de email: %w", err)
	}

	var body bytes.Buffer
	if err := t.Execute(&body, data); err != nil {
		return fmt.Errorf("erro ao processar template: %w", err)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", s.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Sua assinatura está ativa na Ligue Medicina")
	m.SetBody("text/html", body.String())
	m.AddAlternative("text/plain", fmt.Sprintf("Seja bem-vindo(a) à Ligue Medicina!\n\nOlá, %s!\n\nÉ um prazer ter você com a gente. Seu pagamento foi confirmado com sucesso e sua assinatura do plano já está disponível.\n\nEm anexo, sua carteirinha digital para um acesso mais fácil à nossa plataforma.\n\nComo acessar suas consultas\n- Para realizar consultas, basta acessar nosso portal e preencher as mesmas informações utilizadas no momento da contratação. Em poucos passos, você já estará conectado ao atendimento.\n\nAcesse nosso portal: %s\n\nSe tiver qualquer dúvida ou precisar de ajuda, é só falar com a gente pelo WhatsApp.\n\nTirar dúvidas com nossa equipe pelo WhatsApp: %s\n\nUm abraço,\nEquipe Ligue Medicina & Grupo Cuidarte", firstName(name), data.PortalURL, data.WhatsAppURL))

	for _, attachment := range BuildMembershipCardAttachments(name, productName, cardNumber, data.PortalURL, dependents) {
		tempFile, tempErr := os.CreateTemp("", attachment.Filename)
		if tempErr != nil {
			continue
		}

		if _, writeErr := tempFile.Write(attachment.Content); writeErr == nil {
			_ = tempFile.Close()
			m.Attach(tempFile.Name())
			defer os.Remove(tempFile.Name())
			continue
		}

		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}

	if len(contractPDF) > 0 {
		contractFile, contractErr := os.CreateTemp("", "termo_adesao_*.pdf")
		if contractErr == nil {
			if _, writeErr := contractFile.Write(contractPDF); writeErr == nil {
				_ = contractFile.Close()
				m.Attach(contractFile.Name(), gomail.Rename("termo_adesao.pdf"))
				defer os.Remove(contractFile.Name())
			} else {
				_ = contractFile.Close()
				_ = os.Remove(contractFile.Name())
			}
		}
	}

	d := gomail.NewDialer(s.Host, s.Port, s.User, s.Password)
	d.TLSConfig = &tls.Config{
		ServerName: strings.TrimSpace(s.Host),
		MinVersion: tls.VersionTLS12,
	}

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("erro ao enviar email SMTP: %w", err)
	}

	return nil
}

func (s *EmailSender) SendWelcomeEmail(name, email string) error {
	return s.SendWelcomeEmailWithCard(name, email, "", "Ligue Medicina", "")
}

func (s *EmailSender) SendWelcomeEmailWithCard(name, email, cpf, planName, providerID string) error {
	cardNumber := strings.TrimSpace(providerID)
	if cardNumber == "" {
		cardNumber = strings.TrimSpace(cpf)
	}

	if strings.TrimSpace(planName) == "" {
		planName = "Ligue Medicina"
	}

	return s.sendWelcomeInternal(email, name, planName, "", cardNumber, nil, nil)
}

func (s *EmailSender) SendWelcomeEmailWithCardAndDependents(name, email, cpf, planName, providerID string, dependents []*entity.Dependent) error {
	cardNumber := strings.TrimSpace(providerID)
	if cardNumber == "" {
		cardNumber = strings.TrimSpace(cpf)
	}

	if strings.TrimSpace(planName) == "" {
		planName = "Ligue Medicina"
	}

	return s.sendWelcomeInternal(email, name, planName, "", cardNumber, dependents, nil)
}
