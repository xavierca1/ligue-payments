package pdf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/jung-kurt/gofpdf/v2"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"golang.org/x/text/unicode/norm"
)

// ContractFormData holds the values for every AcroForm field in the contract template.
// Field names mirror the PDF form fields with the "input_" prefix stripped.
//
// ============================================================================
// MAPEAMENTO DE FIELDS PARA DOCUSEAL (Assinatura Digital)
// ============================================================================
// Para integração com DocuSeal, os seguintes fields devem ser mapeados:
//
// Signatário:
//   - Nome     -> fullname do signatário
//   - Email    -> email do signatário
//
// Dados Pessoais:
//   - CPF      -> Field: "input_cpf" no PDF -> COLOCAR FIELD UUID DO DOCUSEAL AQUI
//   - RG       -> Field: "input_rg" no PDF -> COLOCAR FIELD UUID DO DOCUSEAL AQUI
//   - Sexo     -> Field: "input_sexo" no PDF -> COLOCAR FIELD UUID DO DOCUSEAL AQUI
//   - Civil    -> Field: "input_civil" no PDF -> COLOCAR FIELD UUID DO DOCUSEAL AQUI
//   - Nascimento -> Field: "input_nascimento" no PDF -> COLOCAR FIELD UUID DO DOCUSEAL AQUI
//
// Dados de Contato:
//   - Celular  -> Field: "input_celular" no PDF -> COLOCAR FIELD UUID DO DOCUSEAL AQUI
//   - Fixo     -> Field: "input_fixo" no PDF -> COLOCAR FIELD UUID DO DOCUSEAL AQUI
//   - Email    -> Field: "input_email" no PDF -> COLOCAR FIELD UUID DO DOCUSEAL AQUI
//
// Dados de Endereço:
//   - Endereco      -> Field: "input_endereco" no PDF -> COLOCAR FIELD UUID DO DOCUSEAL AQUI
//   - Numero        -> Field: "input_numero" no PDF -> COLOCAR FIELD UUID DO DOCUSEAL AQUI
//   - Complemento   -> Field: "input_complemento" no PDF -> COLOCAR FIELD UUID DO DOCUSEAL AQUI
//   - Bairro        -> Field: "input_bairro" no PDF -> COLOCAR FIELD UUID DO DOCUSEAL AQUI
//   - Cidade        -> Field: "input_cidade" no PDF -> COLOCAR FIELD UUID DO DOCUSEAL AQUI
//   - UF            -> Field: "input_uf" no PDF -> COLOCAR FIELD UUID DO DOCUSEAL AQUI
//   - CEP           -> Field: "input_cep" no PDF -> COLOCAR FIELD UUID DO DOCUSEAL AQUI
//
// Dados do Plano/Produto:
//   - Produto      -> Field: "input_produto" no PDF -> COLOCAR FIELD UUID DO DOCUSEAL AQUI
//   - Valor        -> Field: "input_valor" no PDF -> COLOCAR FIELD UUID DO DOCUSEAL AQUI
//   - Pagamento    -> Field: "input_pagamento" no PDF -> COLOCAR FIELD UUID DO DOCUSEAL AQUI
//   - Periodicidade -> Field: "input_periodicidade" no PDF -> COLOCAR FIELD UUID DO DOCUSEAL AQUI
//
// Fluxo com DocuSeal:
// 1. O PDF é preenchido com os dados do ContractFormData
// 2. O PDF é convertido em base64 e enviado ao DocuSeal
// 3. DocuSeal cria um template com os fields definidos
// 4. Uma submission é criada com o email/nome do cliente
// 5. O cliente recebe um link de assinatura
// 6. Após assinado, o PDF assinado é recuperado do DocuSeal
// 7. O PDF assinado é enviado por email anexado
type ContractFormData struct {
	Produto       string
	ID            string
	Valor         string
	Pagamento     string
	Periodicidade string
	Nome          string
	Nascimento    string
	CPF           string
	RG            string
	Orgao         string
	Sexo          string
	Civil         string
	Celular       string
	Fixo          string
	Email         string
	Endereco      string
	Numero        string
	Complemento   string
	Bairro        string
	Cidade        string
	UF            string
	CEP           string
}

// ContractGenerator fills PDF AcroForm templates and appends a digital acceptance certificate.
// It uses pdfcpu for form filling and PDF merging, so no external PDF binaries are required.
type ContractGenerator struct {
	templateDir string
}

func NewContractGenerator(templateDir string) *ContractGenerator {
	return &ContractGenerator{templateDir: templateDir}
}

// Generate fills the plan template with customer data, appends the digital acceptance certificate page,
// and returns the final PDF as bytes.
func (g *ContractGenerator) Generate(planName string, data ContractFormData, clientIP string) ([]byte, error) {
	templatePath, err := g.resolveTemplatePath(planName)
	if err != nil {
		return nil, err
	}

	filled, err := g.fillTemplate(templatePath, data)
	if err != nil {
		return nil, fmt.Errorf("fill form: %w", err)
	}

	certPage, err := buildCertificationPage(data.Nome, clientIP)
	if err != nil {
		return nil, fmt.Errorf("build certification page: %w", err)
	}

	return mergePDFs(filled, certPage)
}

func (g *ContractGenerator) resolveTemplatePath(planName string) (string, error) {
	normalizedName := normalizeTemplateName(planName)
	candidates := []string{filepath.Join(g.templateDir, normalizedName+".pdf")}

	if alias, ok := templateAliases[normalizedName]; ok {
		candidates = append(candidates, filepath.Join(g.templateDir, alias))
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("template PDF não encontrado para o plano %q em %s", planName, g.templateDir)
}

func normalizeTemplateName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	decomposed := norm.NFD.String(name)

	var builder strings.Builder
	previousUnderscore := false

	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}

		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			previousUnderscore = false
			continue
		}

		if !previousUnderscore {
			builder.WriteByte('_')
			previousUnderscore = true
		}
	}

	return strings.Trim(builder.String(), "_")
}

var templateAliases = map[string]string{
	"ligue_vida_plena":    "LigueMedicina - Termo de Adesão Ligue Vida Plena - CHECKOUT (EDITADO).pdf",
	"ligue_mais_cuidado":  "LigueMedicina - Termo de Adesão Ligue Mais Cuidado - CHECKOUT (EDITADO).pdf",
	"ligue_cuidado_total": "LigueMedicina - Termo de Adesão Ligue Cuidado Total - CHECKOUT (EDITADO).pdf",
	"ligue_viver_bem":     "LigueMedicina - Termo de Adesão (EDITADO).pdf",
	"plano_individual":    "LigueMedicina - Termo de Adesão (EDITADO).pdf",
	"ligue_saude_em_dia":  "LigueMedicina - Termo de Adesão SAUDE EM DiA (EDITADO).pdf",
	"saude_em_dia":        "LigueMedicina - Termo de Adesão SAUDE EM DiA (EDITADO).pdf",
}

func (g *ContractGenerator) fillTemplate(templatePath string, data ContractFormData) ([]byte, error) {
	jsonPayload, err := buildPDFCPUFormJSON(data)
	if err != nil {
		return nil, err
	}

	jsonFile, err := os.CreateTemp("", "ligue-contract-*.json")
	if err != nil {
		return nil, fmt.Errorf("create temp json: %w", err)
	}
	defer os.Remove(jsonFile.Name())

	if _, err := jsonFile.Write(jsonPayload); err != nil {
		_ = jsonFile.Close()
		return nil, fmt.Errorf("write json: %w", err)
	}
	if err := jsonFile.Close(); err != nil {
		return nil, fmt.Errorf("close json: %w", err)
	}

	outputFile, err := os.CreateTemp("", "ligue-filled-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("create temp output pdf: %w", err)
	}
	outputPath := outputFile.Name()
	if err := outputFile.Close(); err != nil {
		return nil, fmt.Errorf("close temp output pdf: %w", err)
	}
	defer os.Remove(outputPath)

	if err := api.FillFormFile(templatePath, jsonFile.Name(), outputPath, nil); err != nil {
		return nil, fmt.Errorf("pdfcpu fill_form: %w", err)
	}

	return os.ReadFile(outputPath)
}

func buildPDFCPUFormJSON(data ContractFormData) ([]byte, error) {
	type textField struct {
		ID    string `json:"id,omitempty"`
		Name  string `json:"name,omitempty"`
		Value string `json:"value"`
	}

	payload := struct {
		Forms []struct {
			TextFields []textField `json:"textfield,omitempty"`
		} `json:"forms"`
	}{
		Forms: []struct {
			TextFields []textField `json:"textfield,omitempty"`
		}{{TextFields: []textField{
			{ID: "input_produto", Name: "input_produto", Value: data.Produto},
			{ID: "input_id", Name: "input_id", Value: data.ID},
			{ID: "input_valor", Name: "input_valor", Value: data.Valor},
			{ID: "input_pagamento", Name: "input_pagamento", Value: data.Pagamento},
			{ID: "input_periodicidade", Name: "input_periodicidade", Value: data.Periodicidade},
			{ID: "input_nome", Name: "input_nome", Value: data.Nome},
			{ID: "input_nascimento", Name: "input_nascimento", Value: data.Nascimento},
			{ID: "input_cpf", Name: "input_cpf", Value: data.CPF},
			{ID: "input_rg", Name: "input_rg", Value: data.RG},
			{ID: "input_orgao", Name: "input_orgao", Value: data.Orgao},
			{ID: "input_sexo", Name: "input_sexo", Value: data.Sexo},
			{ID: "input_civil", Name: "input_civil", Value: data.Civil},
			{ID: "input_celular", Name: "input_celular", Value: data.Celular},
			{ID: "input_fixo", Name: "input_fixo", Value: data.Fixo},
			{ID: "input_email", Name: "input_email", Value: data.Email},
			{ID: "input_endereco", Name: "input_endereco", Value: data.Endereco},
			{ID: "input_numero", Name: "input_numero", Value: data.Numero},
			{ID: "input_complemento", Name: "input_complemento", Value: data.Complemento},
			{ID: "input_bairro", Name: "input_bairro", Value: data.Bairro},
			{ID: "input_cidade", Name: "input_cidade", Value: data.Cidade},
			{ID: "input_uf", Name: "input_uf", Value: data.UF},
			{ID: "input_cep", Name: "input_cep", Value: data.CEP},
		}}},
	}

	return json.Marshal(payload)
}

// buildCertificationPage generates a one-page PDF that certifies the digital acceptance.
// Uses gofpdf with built-in fonts, so accented characters are normalized to ASCII.
func buildCertificationPage(customerName, clientIP string) ([]byte, error) {
	brt := time.FixedZone("BRT", -3*60*60)
	now := time.Now().In(brt)

	safeName := normalizeASCII(customerName)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(25, 30, 25)
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(30, 50, 120)
	pdf.CellFormat(0, 10, "CERTIFICADO DE ACEITE DIGITAL", "", 1, "C", false, 0, "")
	pdf.Ln(4)

	pdf.SetDrawColor(30, 50, 120)
	pdf.SetLineWidth(0.5)
	pdf.Line(25, pdf.GetY(), 185, pdf.GetY())
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 11)
	pdf.SetTextColor(0, 0, 0)

	body := fmt.Sprintf(
		"O presente certificado atesta que o(a) contratante identificado(a) abaixo aceitou\n"+
			"eletronicamente os Termos de Adesao ao plano de saude selecionado, em conformidade\n"+
			"com a Lei Federal no 14.063/2020 (Assinaturas Eletronicas).\n\n"+
			"Nome do Contratante : %s\n"+
			"Data e Hora do Aceite: %s\n"+
			"Endereco IP de Origem: %s\n\n"+
			"Este documento possui validade juridica plena e e equivalente a uma\n"+
			"assinatura manuscrita para todos os fins de direito.",
		safeName,
		now.Format("02/01/2006 as 15:04:05 BRT"),
		clientIP,
	)
	pdf.MultiCell(0, 7, body, "", "L", false)
	pdf.Ln(10)

	pdf.SetFont("Arial", "I", 9)
	pdf.SetTextColor(100, 100, 100)
	pdf.MultiCell(0, 6,
		"Documento gerado automaticamente pelo sistema Ligue Medicina.\n"+
			"Nao requer assinatura adicional.",
		"", "C", false,
	)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func mergePDFs(a, b []byte) ([]byte, error) {
	fileA, err := os.CreateTemp("", "ligue-filled-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("create temp filled pdf: %w", err)
	}
	defer os.Remove(fileA.Name())

	fileB, err := os.CreateTemp("", "ligue-cert-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("create temp cert pdf: %w", err)
	}
	defer os.Remove(fileB.Name())

	if _, err := fileA.Write(a); err != nil {
		_ = fileA.Close()
		return nil, err
	}
	if err := fileA.Close(); err != nil {
		return nil, err
	}

	if _, err := fileB.Write(b); err != nil {
		_ = fileB.Close()
		return nil, err
	}
	if err := fileB.Close(); err != nil {
		return nil, err
	}

	mergedFile, err := os.CreateTemp("", "ligue-merged-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("create temp merged pdf: %w", err)
	}
	mergedPath := mergedFile.Name()
	if err := mergedFile.Close(); err != nil {
		return nil, err
	}
	defer os.Remove(mergedPath)

	if err := api.MergeCreateFile([]string{fileA.Name(), fileB.Name()}, mergedPath, false, nil); err != nil {
		return nil, fmt.Errorf("pdfcpu merge: %w", err)
	}

	return os.ReadFile(mergedPath)
}

// normalizeASCII strips accented characters to their ASCII base equivalents.
// This is required because gofpdf's built-in core fonts do not support UTF-8.
func normalizeASCII(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case 'Á', 'À', 'Â', 'Ã', 'Ä':
			b.WriteByte('A')
		case 'É', 'È', 'Ê', 'Ë':
			b.WriteByte('E')
		case 'Í', 'Ì', 'Î', 'Ï':
			b.WriteByte('I')
		case 'Ó', 'Ò', 'Ô', 'Õ', 'Ö':
			b.WriteByte('O')
		case 'Ú', 'Ù', 'Û', 'Ü':
			b.WriteByte('U')
		case 'Ç':
			b.WriteByte('C')
		case 'á', 'à', 'â', 'ã', 'ä':
			b.WriteByte('a')
		case 'é', 'è', 'ê', 'ë':
			b.WriteByte('e')
		case 'í', 'ì', 'î', 'ï':
			b.WriteByte('i')
		case 'ó', 'ò', 'ô', 'õ', 'ö':
			b.WriteByte('o')
		case 'ú', 'ù', 'û', 'ü':
			b.WriteByte('u')
		case 'ç':
			b.WriteByte('c')
		default:
			if r < 128 {
				b.WriteRune(r)
			}
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
