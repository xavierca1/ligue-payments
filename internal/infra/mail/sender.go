package mail

// TODO
// A Struct de Dados (EmailData): Precisa ter os campos Name, ProductName e PDFLink. Lembre-se que para o template ler, os campos precisam ser Públicos (Letra Maiúscula).

// A Função SendWelcomeEmail:

// Recebe to, subject, e os dados (EmailData).

// Lê as variáveis de ambiente (os.Getenv).

// Faz o parse do arquivo HTML (template.ParseFiles).

// Executa o template jogando o resultado num bytes.Buffer.

// Configura o gomail.NewMessage (From, To, Subject, SetBody "text/html").

// Configura o gomail.NewDialer e envia (DialAndSend).

// Dependência: go get gopkg.in/gomail.v2
