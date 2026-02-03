package mail

type WelcomeEmailData struct {
	Name        string
	PDFLink     string
	ProductName string
}

type EmailSender struct {
	Host     string
	Port     int
	User     string
	Password string
}
