//go:build ignore
// +build ignore

package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xavierca1/ligue-payments/internal/infra/mail"
)

func TestGenerateMembershipCardMockPDF(t *testing.T) {
	pdfBytes, err := mail.GenerateMembershipCard(mail.MembershipCardData{
		FullName:          "Joao da Silva Teste",

































}	t.Logf("Carteirinha mock gerada em: %s", artifactPath)	}		assert.Greater(t, info.Size(), int64(100), "arquivo gerado está muito pequeno")	if info, statErr := os.Stat(artifactPath); statErr == nil {	assert.NoError(t, os.WriteFile(artifactPath, pdfBytes, 0o644))	artifactPath := filepath.Join(artifactDir, "carteirinha-mock-joao-da-silva-teste.pdf")	assert.NoError(t, os.MkdirAll(artifactDir, 0o755))	artifactDir := filepath.Join("artifacts")	assert.NoError(t, err)	})		PlatformAccessURL: "https://app.liguemedicina.com.br",		IssuedAt:          "20/04/2026",		CardNumber:        "MOCK-123456",		PlanName:          "Ligue Mais Cuidado",		FullName:          "Joao da Silva Teste",	pdfBytes, err := mail.GenerateMembershipCard(mail.MembershipCardData{func TestGenerateMembershipCardMockWriteArtifact(t *testing.T) {}	assert.True(t, strings.HasPrefix(string(pdfBytes), "%PDF"), "output não parece um PDF válido")	assert.NotEmpty(t, pdfBytes)	assert.NoError(t, err)	})		PlatformAccessURL: "https://app.liguemedicina.com.br",		IssuedAt:          "20/04/2026",		CardNumber:        "MOCK-123456",		PlanName:          "Ligue Mais Cuidado",