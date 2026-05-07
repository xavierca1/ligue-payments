package tests

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xavierca1/ligue-payments/internal/infra/mail"
)

func TestGenerateMembershipCardMockPDF(t *testing.T) {
	pdfBytes, err := mail.GenerateMembershipCard(mail.MembershipCardData{
		FullName:          "Joao da Silva Teste",
		PlanName:          "Ligue Mais Cuidado",
		CardNumber:        "MOCK-123456",
		IssuedAt:          "20/04/2026",
		PlatformAccessURL: "https://app.liguemedicina.com.br",
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)
	assert.True(t, strings.HasPrefix(string(pdfBytes), "%PDF"), "output não parece um PDF válido")

	matches := regexp.MustCompile(`/Count\s+(\d+)`).FindSubmatch(pdfBytes)
	if assert.Len(t, matches, 2, "metadado de páginas não encontrado no PDF") {
		assert.Equal(t, "1", string(matches[1]), "carteirinha deve ter exatamente 1 página")
	}
}

func TestGenerateMembershipCardMockWriteArtifact(t *testing.T) {
	pdfBytes, err := mail.GenerateMembershipCard(mail.MembershipCardData{
		FullName:          "Joao da Silva Teste",
		PlanName:          "Ligue Mais Cuidado",
		CardNumber:        "MOCK-123456",
		IssuedAt:          "20/04/2026",
		PlatformAccessURL: "https://app.liguemedicina.com.br",
	})
	assert.NoError(t, err)

	artifactDir := filepath.Join("artifacts")
	assert.NoError(t, os.MkdirAll(artifactDir, 0o755))

	artifactPath := filepath.Join(artifactDir, "carteirinha-mock-joao-da-silva-teste.pdf")
	assert.NoError(t, os.WriteFile(artifactPath, pdfBytes, 0o644))

	if info, statErr := os.Stat(artifactPath); statErr == nil {
		assert.Greater(t, info.Size(), int64(100), "arquivo gerado está muito pequeno")
	}

	t.Logf("Carteirinha mock gerada em: %s", artifactPath)
}
