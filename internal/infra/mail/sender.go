package mail

import (
	"bytes"
	"fmt"
	"path/filepath"
	"text/template"

	"gopkg.in/gomail.v2"
)

func NewEmailSender(host string, port int, user, password string) *EmailSender {
	return &EmailSender{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
	}
}

func (s *EmailSender) SendWelcome(to, name, productName, pdfLink string) error {
	data := WelcomeEmailData{
		Name:        name,
		ProductName: productName,
		PDFLink:     pdfLink,
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
	m.SetHeader("From", "nao-responda@liguemedicina.com")
	m.SetHeader("To", to)
	m.SetHeader("Subject", fmt.Sprintf("Bem-vindo à Ligue, %s! Seu acesso chegou 🚀", name))
	m.SetBody("text/html", body.String())

	d := gomail.NewDialer(s.Host, s.Port, s.User, s.Password)

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("erro ao enviar email SMTP: %w", err)
	}

	return nil
}

func (s *EmailSender) SendWelcomeEmail(name, email string) error {
	return s.SendWelcome(email, name, "Ligue Medicina", "")
}
