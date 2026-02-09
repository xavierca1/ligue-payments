package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/kommo"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  Aviso: arquivo .env não encontrado, usando variáveis de ambiente do sistema")
	}

	if os.Getenv("KOMMO_API_TOKEN") == "" {
		log.Fatal("❌ KOMMO_API_TOKEN deve estar configurado no .env")
	}

	client := kommo.NewClient()

	input := kommo.CreateLeadInput{
		CustomerName: "Joao Teste da Silva",
		Phone:        "+556199767638",
		Email:        "joao.teste@email.com",
		PlanName:     "Plano Medicina",
		Price:        1990,
		Origin:       "Website",
	}

	fmt.Println("🔄 Criando lead no Kommo...")
	fmt.Printf("📋 Dados:\n")
	fmt.Printf("   Nome: %s\n", input.CustomerName)
	fmt.Printf("   Telefone: %s\n", input.Phone)
	fmt.Printf("   Email: %s\n", input.Email)
	fmt.Printf("   Plano: %s\n", input.PlanName)
	fmt.Printf("   Valor: R$ %.2f\n", input.Price)
	fmt.Printf("   Origem: %s\n\n", input.Origin)

	leadID, err := client.CreateLead(input)
	if err != nil {
		log.Fatalf("Erro ao criar lead no Kommo: %v", err)
	}

	accountID := os.Getenv("KOMMO_ACCOUNT_ID")
	if accountID == "" {
		accountID = "liguemedicina"
	}

	fmt.Printf("Lead criado com sucesso no Kommo! \n")
	fmt.Printf(" ID do Lead: #%d\n", leadID)
	fmt.Printf(" Link: https://%s.kommo.com/leads/detail/%d\n", accountID, leadID)
}
