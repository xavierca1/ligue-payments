package mail

type WelcomeEmailDara struct {
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
type WelcomeEmailData struct {
	Name        string
	ProductName string
	PDFLink     string
}
