package usecase

import (
	"encoding/json"
	"fmt"
	"log"
)

// DebugDocuSealFields isola a construção dos fields do DocuSeal para debug
// Você pode chamar essa função com um payload de checkout para visualizar
// quais campos estão sendo enviados para o DocuSeal
func DebugDocuSealFields(input DocuSealContractInput) map[string]string {
	// Exatamente o mesmo mapa que é construído na função Execute
	fieldValues := map[string]string{
		"product":        input.Produto,
		"id":             input.CustomerID,
		"method_payment": input.Pagamento,
		"periodicidade":  input.Periodicidade,
		"name":           input.Nome,
		"birthdate":      input.Nascimento,
		"cpf":            input.CPF,
		"genre":          input.Sexo,
		"marital_status": input.Civil,
		"cellphone":      input.Celular,
		"email":          input.Email,
		"address":        input.Endereco,
		"number":         input.Numero,
		"neighborhood":   input.Bairro,
		"city":           input.Cidade,
		"UF":             input.UF,
		"zip_code":       input.CEP,
	}

	// Normaliza "monthly" → "Mensal"
	fieldValues = normalizeDocuSealMonthly(fieldValues)

	// Printa em JSON formatado para você ver
	jsonBytes, _ := json.MarshalIndent(fieldValues, "", "  ")
	log.Printf("\n========== DEBUG DOCUSEAL FIELDS ==========\n%s\n==========================================\n", string(jsonBytes))

	return fieldValues
}

// DebugDocuSealPayload exibe como ficaria a requisição completa para DocuSeal
func DebugDocuSealPayload(input DocuSealContractInput, templateID int) {
	fieldValues := DebugDocuSealFields(input)

	// Simula a estrutura que vai para DocuSeal
	debugPayload := map[string]interface{}{
		"template_id": templateID,
		"send_email":  true,
		"submitters": []map[string]interface{}{
			{
				"email":     input.Email,
				"name":      input.Nome,
				"role":      "Proponente",
				"completed": true,
				"values":    fieldValues,
			},
		},
		"custom_email": map[string]string{
			"subject":   fmt.Sprintf("Cópia do Termo de Adesão - %s", input.PlanName),
			"body":      fmt.Sprintf("Olá, %s,\n\nConfirmamos o aceite do termo referente ao %s.", input.Nome, input.PlanName),
			"from_name": "Ligue Medicina",
		},
	}

	jsonBytes, _ := json.MarshalIndent(debugPayload, "", "  ")
	log.Printf("\n========== DEBUG DOCUSEAL PAYLOAD COMPLETO ==========\n%s\n=====================================================\n", string(jsonBytes))
}

// PrintFieldDifferences mostra quais campos estão vazios vs preenchidos
func PrintFieldDifferences(input DocuSealContractInput) {
	fieldValues := DebugDocuSealFields(input)

	fmt.Println("\n========== CAMPO POR CAMPO ==========")
	for fieldName, fieldValue := range fieldValues {
		if fieldValue == "" {
			fmt.Printf("❌ [VAZIO] %s\n", fieldName)
		} else {
			fmt.Printf("✅ [OK] %s = %q\n", fieldName, fieldValue)
		}
	}
	fmt.Println("=====================================")
}
