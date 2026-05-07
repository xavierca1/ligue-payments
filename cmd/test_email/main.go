package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/xavierca1/ligue-payments/internal/infra/mail"
)

func main() {
	_ = godotenv.Load()

	toFlag := flag.String("to", "", "Email destino para teste")
	nameFlag := flag.String("name", "Cliente Ligue", "Nome para personalização do email")
	flag.Parse()

	to := strings.TrimSpace(*toFlag)
	name := strings.TrimSpace(*nameFlag)
	if to == "" {
		log.Fatal("parâmetro obrigatório ausente: -to")
	}

	host := strings.TrimSpace(os.Getenv("MAIL_HOST"))
	user := strings.TrimSpace(os.Getenv("MAIL_USER"))
	pass := strings.TrimSpace(os.Getenv("MAIL_PASS"))
	from := strings.TrimSpace(os.Getenv("MAIL_FROM"))

	port := 587
	if raw := strings.TrimSpace(os.Getenv("MAIL_PORT")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			log.Fatalf("MAIL_PORT inválido: %v", err)
		}
		port = parsed
	}

	missing := make([]string, 0)
	if host == "" {
		missing = append(missing, "MAIL_HOST")
	}
	if user == "" {
		missing = append(missing, "MAIL_USER")
	}
	if pass == "" {
		missing = append(missing, "MAIL_PASS")
	}
	if len(missing) > 0 {
		log.Fatalf("variáveis ausentes no ambiente: %s", strings.Join(missing, ", "))
	}

	sender := mail.NewEmailSenderWithFrom(host, port, user, pass, from)
	if err := sender.SendWelcomeEmail(name, to); err != nil {
		log.Fatalf("falha ao enviar email de teste: %v", err)
	}

	fmt.Printf("email de teste enviado com sucesso para %s\n", to)
}
package main

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/xavierca1/ligue-payments/internal/infra/mail"
)

func main() {
	_ = godotenv.Load()

	toFlag := flag.String("to", "", "Email destino para teste")
	nameFlag := flag.String("name", "Cliente Ligue", "Nome para personalização do email")
	flag.Parse()

	to := strings.TrimSpace(*toFlag)
	name := strings.TrimSpace(*nameFlag)
	if to == "" {
		log.Fatal("parâmetro obrigatório ausente: -to")
	}

	host := strings.TrimSpace(os.Getenv("MAIL_HOST"))
	user := strings.TrimSpace(os.Getenv("MAIL_USER"))
	pass := strings.TrimSpace(os.Getenv("MAIL_PASS"))
	from := strings.TrimSpace(os.Getenv("MAIL_FROM"))

	port := 587
	if raw := strings.TrimSpace(os.Getenv("MAIL_PORT")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			log.Fatalf("MAIL_PORT inválido: %v", err)
		}
		port = parsed
	}

	missing := make([]string, 0)
	if host == "" {
		missing = append(missing, "MAIL_HOST")
	}
	if user == "" {
		missing = append(missing, "MAIL_USER")
	}
	if pass == "" {
		missing = append(missing, "MAIL_PASS")
	}
	if len(missing) > 0 {
		log.Fatalf("variáveis ausentes no ambiente: %s", strings.Join(missing, ", "))
	}

	sender := mail.NewEmailSenderWithFrom(host, port, user, pass, from)
	if err := sender.SendWelcomeEmail(name, to); err != nil {
		log.Fatalf("falha ao enviar email de teste: %v", err)
	}

	fmt.Printf("email de teste enviado com sucesso para %s\n", to)
}
}	fmt.Printf("email de teste enviado com sucesso para %s\n", *to)	}		log.Fatalf("falha ao enviar email de teste: %v", err)	if err := sender.SendWelcomeEmail(*name, *to); err != nil {	sender := mail.NewEmailSenderWithFrom(host, port, user, pass, from)	}		log.Fatalf("variáveis ausentes no ambiente: %s", strings.Join(missing, ", "))	if len(missing) > 0 {	}		missing = append(missing, "MAIL_PASS")	if pass == "" {	}		missing = append(missing, "MAIL_USER")	if user == "" {	}		missing = append(missing, "MAIL_HOST")	if host == "" {	missing := make([]string, 0)	}		port = parsed		}			log.Fatalf("MAIL_PORT inválido: %v", err)		if err != nil {		parsed, err := strconv.Atoi(raw)	if raw := strings.TrimSpace(os.Getenv("MAIL_PORT")); raw != "" {	port := 587	from := strings.TrimSpace(os.Getenv("MAIL_FROM"))	pass := strings.TrimSpace(os.Getenv("MAIL_PASS"))	user := strings.TrimSpace(os.Getenv("MAIL_USER"))	host := strings.TrimSpace(os.Getenv("MAIL_HOST"))	}		log.Fatal("parâmetro obrigatório ausente: -to")	if *to == "" {	flag.Parse()	name := strings.TrimSpace(flag.String("name", "Cliente Ligue", "Nome para personalização do email"))	to := strings.TrimSpace(flag.String("to", "", "Email destino para teste"))	_ = godotenv.Load()func main() {)	"github.com/xavierca1/ligue-payments/internal/infra/mail"	"github.com/joho/godotenv"	"strings"	"strconv"	"os"	"log"	"fmt"