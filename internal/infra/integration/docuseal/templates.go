package docuseal

import "strings"

// TemplateConfig define a configuração de templates no DocuSeal
type TemplateConfig struct {
	ID   int
	Name string
	Role string // Role do signatário conforme definido no template DocuSeal
}

// Templates é o mapa de todos os templates disponíveis no DocuSeal
// Todos compartilham os mesmos fields
var Templates = map[string]TemplateConfig{
	"ligue_saude_em_dia":  {ID: 3346712, Name: "Ligue Saúde em Dia", Role: "Proponente"},
	"ligue_mais_cuidado":  {ID: 3346739, Name: "Ligue Mais Cuidado", Role: "Proponente"},
	"ligue_vida_plena":    {ID: 3624336, Name: "Ligue Vida Plena", Role: "Cliente"},
	"ligue_cuidado_total": {ID: 3346755, Name: "Ligue Cuidado Total", Role: "Proponente"},
	"ligue_viver_bem":     {ID: 3346717, Name: "Ligue Viver Bem", Role: "Proponente"},
}

// PlanToTemplateMap mapeia nomes de plano para templates DocuSeal
// Faz matching case-insensitive e parcial
var PlanToTemplateMap = map[string]string{
	"saúde em dia":  "ligue_saude_em_dia",
	"mais cuidado":  "ligue_mais_cuidado",
	"vida plena":    "ligue_vida_plena",
	"cuidado total": "ligue_cuidado_total",
	"viver bem":     "ligue_viver_bem",
	// Variações com "ligue" no início
	"ligue saúde em dia":  "ligue_saude_em_dia",
	"ligue mais cuidado":  "ligue_mais_cuidado",
	"ligue vida plena":    "ligue_vida_plena",
	"ligue cuidado total": "ligue_cuidado_total",
	"ligue viver bem":     "ligue_viver_bem",
}

// DocuSealFields lista todos os fields suportados (mesmos para todos os templates)
var DocuSealFields = []string{
	"product",
	"id",
	"method_payment",
	"periodicidade",
	"value",
	"name",
	"birthdate",
	"cpf",
	"genre",
	"marital_status",
	"cellphone",
	"email",
	"address",
	"number",
	"neighborhood",
	"city",
	"UF",
	"zip_code",
}

// GetTemplateID retorna o ID numérico de um template pelo nome
func GetTemplateID(templateName string) (int, bool) {
	if config, exists := Templates[templateName]; exists {
		return config.ID, true
	}
	return 0, false
}

// IsValidTemplate verifica se um nome de template é válido
func IsValidTemplate(templateName string) bool {
	_, exists := Templates[templateName]
	return exists
}

// GetTemplateRole retorna o role do signatário para o template especificado.
// Default "Proponente" quando o template não é encontrado.
func GetTemplateRole(templateName string) string {
	if config, exists := Templates[templateName]; exists && config.Role != "" {
		return config.Role
	}
	return "Proponente"
}

// GetTemplateFromPlanName mapeia um nome de plano para o template DocuSeal correspondente
// Faz matching case-insensitive e parcial
// Exemplo: "Ligue Saúde em Dia" ou "Saúde em Dia" → "ligue_saude_em_dia"
func GetTemplateFromPlanName(planName string) string {
	if planName == "" {
		return "ligue_saude_em_dia" // Default
	}

	normalized := strings.ToLower(strings.TrimSpace(planName))

	// Primeiro tenta match exato com o mapa
	if template, exists := PlanToTemplateMap[normalized]; exists {
		return template
	}

	// Se não encontrar, tenta match parcial removendo acentos
	normalized = strings.ReplaceAll(normalized, "á", "a")
	normalized = strings.ReplaceAll(normalized, "à", "a")
	normalized = strings.ReplaceAll(normalized, "ã", "a")

	// Tenta novamente com normalize
	if template, exists := PlanToTemplateMap[normalized]; exists {
		return template
	}

	// Se ainda não encontrou, procura por substring
	for key, template := range PlanToTemplateMap {
		if strings.Contains(normalized, key) || strings.Contains(key, normalized) {
			return template
		}
	}

	// Default
	return "ligue_saude_em_dia"
}
