package usecase

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/xavierca1/ligue-payments/internal/infra/pdf"
)

func NewGenerateContractUseCase(gen ContractPDFGeneratorInterface, storage ContractStorageInterface) *GenerateContractUseCase {
	return &GenerateContractUseCase{Generator: gen, Storage: storage}
}

// Execute fills the contract PDF template for the given plan, appends the certification page,
// uploads the result to Supabase Storage, and returns the storage path and public URL.
func (uc *GenerateContractUseCase) Execute(ctx context.Context, input GenerateContractInput) (*GenerateContractOutput, error) {
	log.Printf("📄 Gerando contrato PDF — CustomerID=%s Plano=%s", input.CustomerID, input.PlanName)

	formData := pdf.ContractFormData{
		Produto:       input.Produto,
		ID:            input.CustomerID,
		Valor:         input.Valor,
		Pagamento:     input.Pagamento,
		Periodicidade: input.Periodicidade,
		Nome:          input.Nome,
		Nascimento:    input.Nascimento,
		CPF:           input.CPF,
		RG:            input.RG,
		Orgao:         input.Orgao,
		Sexo:          input.Sexo,
		Civil:         input.Civil,
		Celular:       input.Celular,
		Fixo:          input.Fixo,
		Email:         input.Email,
		Endereco:      input.Endereco,
		Numero:        input.Numero,
		Complemento:   input.Complemento,
		Bairro:        input.Bairro,
		Cidade:        input.Cidade,
		UF:            input.UF,
		CEP:           input.CEP,
	}

	pdfBytes, err := uc.Generator.Generate(input.PlanName, formData, input.ClientIP)
	if err != nil {
		return nil, &TechnicalError{
			Code:    "PDF_GENERATION_ERROR",
			Message: fmt.Sprintf("falha ao gerar contrato PDF: %v", err),
		}
	}

	if uc.Storage == nil {
		log.Printf("⚠️ Storage de contrato não configurado; enviando apenas o PDF por email")
		return &GenerateContractOutput{
			StoragePath: "",
			PublicURL:   "",
			PDFBytes:    pdfBytes,
		}, nil
	}

	timestamp := time.Now().UTC().Format("20060102150405")
	storagePath := fmt.Sprintf("%s/termo_adesao_%s_%s.pdf", input.CustomerID, input.PlanName, timestamp)

	publicURL, err := uc.Storage.Upload(ctx, storagePath, pdfBytes)
	if err != nil {
		return nil, &TechnicalError{
			Code:    "CONTRACT_UPLOAD_ERROR",
			Message: fmt.Sprintf("falha ao enviar contrato para storage: %v", err),
		}
	}

	log.Printf("✅ Contrato enviado para storage: %s", storagePath)

	return &GenerateContractOutput{
		StoragePath: storagePath,
		PublicURL:   publicURL,
		PDFBytes:    pdfBytes,
	}, nil
}
