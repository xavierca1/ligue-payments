package usecase

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/docuseal"
)

// TestDocuSealAutomaticSubmission testa a criação automática de documento no DocuSeal
func TestDocuSealAutomaticSubmission(t *testing.T) {
	// Carregar arquivo .env
	_ = godotenv.Load()

	// Carregar variáveis de ambiente
	apiKey := os.Getenv("DOCUSEAL_API_KEY")
	apiURL := os.Getenv("DOCUSEAL_API_URL")

	if apiKey == "" {
		t.Skip("DOCUSEAL_API_KEY não configurada")
	}

	// Criar cliente DocuSeal
	client := docuseal.NewClient(apiURL, apiKey)
	log.Printf("🔑 Configuração DocuSeal:")
	log.Printf("   URL: %s", apiURL)
	log.Printf("   Key: %s...%s", apiKey[:10], apiKey[len(apiKey)-10:])

	// Criar usecase
	useCase := NewGenerateContractWithDocuSealUseCase(nil, client)

	// Dados de teste
	input := DocuSealContractInput{
		TemplateName: "ligue_saude_em_dia", // Template Saúde em Dia
		CustomerID:   "CUST-TEST-001",
		Nome:         "João da Silva",
		Email:        "teste@example.com",
		CPF:          "12345678901",
		PlanName:     "Saúde em Dia",
		Produto:      "Saúde em Dia",
		Valor:        "99.90",
		Pagamento:    "PIX",
		Nascimento:   "1990-05-15",
		Sexo:         "M",
		Civil:        "Solteiro",
		Celular:      "(11) 99999-8888",
		Endereco:     "Rua das Flores",
		Numero:       "123",
		Bairro:       "Centro",
		Cidade:       "São Paulo",
		UF:           "SP",
		CEP:          "01310-100",
	}

	// Executar teste
	log.Println("🔄 Criando documento automático no DocuSeal...")
	submissionUUID, err := useCase.ExecuteAutomatic(context.Background(), input)

	if err != nil {
		t.Fatalf("❌ Erro ao criar submission: %v", err)
	}

	if submissionUUID == "" {
		t.Fatalf("❌ UUID vazio retornado")
	}

	log.Printf("✅ Documento criado com sucesso!")
	log.Printf("📋 Submission UUID: %s", submissionUUID)
	log.Printf("📧 Email: %s", input.Email)
	log.Printf("👤 Nome: %s", input.Nome)
}
