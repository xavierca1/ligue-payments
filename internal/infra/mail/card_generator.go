package mail

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf/v2"
	"github.com/xavierca1/ligue-payments/internal/entity"
)

type MembershipCardData struct {
	FullName          string
	PlanName          string
	CardNumber        string
	IssuedAt          string
	PlatformAccessURL string
	Tag               string
}

type MembershipCardAttachment struct {
	Filename string
	Content  []byte
}

const membershipCardLogoURL = "https://yntprscrhdlrwkgnmzrb.supabase.co/storage/v1/object/public/public-assets/logo/logo_branca.png"

const (
	cardWidthMM  = 150.0
	cardHeightMM = 90.0
)

func GenerateMembershipCard(data MembershipCardData) ([]byte, error) {
	issuedAt := strings.TrimSpace(data.IssuedAt)
	if issuedAt == "" {
		issuedAt = time.Now().Format("02/01/2006")
	}

	fullName := strings.TrimSpace(data.FullName)
	if fullName == "" {
		fullName = "Cliente Ligue"
	}
	// Normaliza o nome para ASCII padrão (remove acentos)
	fullName = normalizeASCII(fullName)
	tag := strings.ToUpper(strings.TrimSpace(data.Tag))

	planName := strings.TrimSpace(data.PlanName)
	if planName == "" {
		planName = "Ligue Medicina"
	}

	cardNumber := strings.TrimSpace(data.CardNumber)
	if cardNumber == "" {
		cardNumber = "PENDENTE"
	}

	platformURL := strings.TrimSpace(data.PlatformAccessURL)
	if platformURL == "" {
		platformURL = "https://app.liguemedicina.com.br"
	}

	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		OrientationStr: "L",
		UnitStr:        "mm",
		// No gofpdf, quando Orientation="L", largura/altura podem ser invertidas internamente.
		// Para garantir saída final em paisagem 150x90, passamos 90x150 aqui.
		Size: gofpdf.SizeType{Wd: cardHeightMM, Ht: cardWidthMM},
	})
	pdf.SetAutoPageBreak(false, 0)
	pdf.SetAcceptPageBreakFunc(func() bool { return false })
	pdf.SetMargins(0, 0, 0)
	pdf.AddPage()

	// Fundo com gradiente horizontal: primary -> secondary
	drawHorizontalGradient(pdf, cardWidthMM, cardHeightMM, [3]int{59, 91, 219}, [3]int{66, 211, 147})

	if !drawRemoteLogo(pdf, membershipCardLogoURL) {
		if logo := resolveLogoPath(); logo != "" {
			pdf.ImageOptions(logo, 105, 8, 35, 0, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
		}
	}

	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 8)
	pdf.SetXY(10, 10)
	pdf.Cell(0, 5, "CARTEIRINHA DIGITAL")

	pdf.SetFont("Arial", "B", 18)
	pdf.SetXY(10, 35)
	pdf.Cell(0, 10, truncate(fullName, 25))

	if tag != "" {
		pdf.SetFont("Arial", "B", 7)
		pdf.SetFillColor(255, 255, 255)
		pdf.SetTextColor(59, 91, 219)
		pdf.SetXY(10, 22)
		pdf.CellFormat(28, 6, tag, "1", 0, "C", true, 0, "")
		pdf.SetTextColor(255, 255, 255)
	}

	pdf.SetFont("Arial", "", 12)
	pdf.SetXY(10, 45)
	pdf.Cell(0, 8, truncate(planName, 38))

	pdf.SetFont("Arial", "B", 6)
	pdf.SetXY(10, 65)
	pdf.Cell(0, 4, "NUMERO DA CARTEIRINHA")

	pdf.SetFont("Arial", "B", 10)
	pdf.SetXY(10, 69)
	pdf.Cell(0, 5, truncate(cardNumber, 28))

	pdf.SetFont("Arial", "B", 6)
	pdf.SetXY(100, 65)
	pdf.Cell(40, 4, "EMITIDO EM")

	pdf.SetFont("Arial", "B", 10)
	pdf.SetXY(100, 69)
	pdf.Cell(40, 5, issuedAt)

	pdf.SetFont("Arial", "", 7)
	pdf.SetTextColor(230, 230, 230)
	pdf.SetXY(0, 82)
	pdf.CellFormat(cardWidthMM, 5, truncate(platformURL, 65), "", 0, "C", false, 0, "")

	pdf.SetDrawColor(255, 255, 255)
	pdf.SetLineWidth(0.3)
	pdf.Rect(1.2, 1.2, 147.6, 87.6, "D")

	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, fmt.Errorf("erro ao gerar PDF da carteirinha: %w", err)
	}

	return out.Bytes(), nil
}

func drawHorizontalGradient(pdf *gofpdf.Fpdf, width, height float64, start, end [3]int) {
	steps := int(width)
	if steps < 2 {
		steps = 2
	}

	for i := 0; i < steps; i++ {
		ratio := float64(i) / float64(steps-1)
		r := int(float64(start[0]) + (float64(end[0]-start[0]) * ratio))
		g := int(float64(start[1]) + (float64(end[1]-start[1]) * ratio))
		b := int(float64(start[2]) + (float64(end[2]-start[2]) * ratio))

		pdf.SetFillColor(r, g, b)
		pdf.Rect(float64(i), 0, 1.5, height, "F")
	}
}

func drawRemoteLogo(pdf *gofpdf.Fpdf, logoURL string) bool {
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(strings.TrimSpace(logoURL))
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	logoBytes, err := io.ReadAll(resp.Body)
	if err != nil || len(logoBytes) == 0 {
		return false
	}

	opt := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
	pdf.RegisterImageOptionsReader("membership-card-logo", opt, bytes.NewReader(logoBytes))
	pdf.ImageOptions("membership-card-logo", 105, 8, 35, 0, false, opt, 0, "")

	return true
}

func resolveLogoPath() string {
	candidates := []string{
		filepath.Join("templates", "logo-ligue-colorida.png"),
		filepath.Join("templates", "logo.png"),
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}

// normalizeASCII remove acentos e caracteres especiais, deixando apenas ASCII padrão
// Exemplo: "João" -> "Joao", "Ágata" -> "Agata"

func BuildMembershipCardAttachments(holderName, planName, cardNumber, platformURL string, dependents []*entity.Dependent) []MembershipCardAttachment {
	attachments := make([]MembershipCardAttachment, 0, 1+len(dependents))

	if cardPDF, err := GenerateMembershipCard(MembershipCardData{
		FullName:          holderName,
		PlanName:          planName,
		CardNumber:        cardNumber,
		IssuedAt:          "",
		PlatformAccessURL: platformURL,
	}); err == nil {
		attachments = append(attachments, MembershipCardAttachment{
			Filename: buildCardFilename(holderName, "titular"),
			Content:  cardPDF,
		})
	}

	for _, dependent := range dependents {
		if dependent == nil {
			continue
		}

		if cardPDF, err := GenerateMembershipCard(MembershipCardData{
			FullName:          dependent.Name,
			PlanName:          planName,
			CardNumber:        cardNumber,
			IssuedAt:          "",
			PlatformAccessURL: platformURL,
			Tag:               "DEPENDENTE",
		}); err == nil {
			attachments = append(attachments, MembershipCardAttachment{
				Filename: buildCardFilename(dependent.Name, "dependente"),
				Content:  cardPDF,
			})
		}
	}

	return attachments
}

func buildCardFilename(name, tag string) string {
	baseName := normalizeASCII(name)
	baseName = strings.ToLower(strings.TrimSpace(baseName))
	baseName = strings.Join(strings.Fields(baseName), "_")
	baseName = strings.Trim(baseName, "_")
	if baseName == "" {
		baseName = "cliente"
	}

	tag = strings.ToLower(strings.TrimSpace(normalizeASCII(tag)))
	if tag != "" {
		return fmt.Sprintf("carteirinha-%s-%s.pdf", baseName, tag)
	}

	return fmt.Sprintf("carteirinha-%s.pdf", baseName)
}
func normalizeASCII(name string) string {
	var result strings.Builder
	for _, r := range name {
		// Caracteres ASCII normais (A-Z, a-z, números, espaços, etc) passam direto
		if r < 128 {
			result.WriteRune(r)
		} else {
			// Para acentos e caracteres especiais, usar um mapeamento simples
			// Cobertura dos acentos mais comuns em português
			switch r {
			// Maiúsculas
			case 'Á', 'À', 'Â', 'Ã', 'Ä':
				result.WriteRune('A')
			case 'É', 'È', 'Ê', 'Ë':
				result.WriteRune('E')
			case 'Í', 'Ì', 'Î', 'Ï':
				result.WriteRune('I')
			case 'Ó', 'Ò', 'Ô', 'Õ', 'Ö':
				result.WriteRune('O')
			case 'Ú', 'Ù', 'Û', 'Ü':
				result.WriteRune('U')
			case 'Ç':
				result.WriteRune('C')
			// Minúsculas
			case 'á', 'à', 'â', 'ã', 'ä':
				result.WriteRune('a')
			case 'é', 'è', 'ê', 'ë':
				result.WriteRune('e')
			case 'í', 'ì', 'î', 'ï':
				result.WriteRune('i')
			case 'ó', 'ò', 'ô', 'õ', 'ö':
				result.WriteRune('o')
			case 'ú', 'ù', 'û', 'ü':
				result.WriteRune('u')
			case 'ç':
				result.WriteRune('c')
			default:
				// Para outros caracteres especiais, pula ou substitui por espaço
				if r > 127 {
					result.WriteRune(' ')
				} else {
					result.WriteRune(r)
				}
			}
		}
	}
	// Remove espaços múltiplos
	normalized := result.String()
	normalized = strings.Join(strings.Fields(normalized), " ")
	return normalized
}
