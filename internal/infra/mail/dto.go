package mail

type WelcomeEmailData struct {
	Name        string
	FirstName   string
	PDFLink     string
	ProductName string
	PortalURL   string
	WhatsAppURL string
}

type EmailSender struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}
