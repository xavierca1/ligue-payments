package tests

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xavierca1/ligue-payments/internal/infra/queue"
)

// ============ TESTES DO QUEUE PRODUCER ============

// TestActivationPayloadMarshalling - Teste que o payload serializa corretamente
func TestActivationPayloadMarshalling(t *testing.T) {
	payload := queue.ActivationPayload{
		CustomerID:       "cust-123",
		PlanID:           "plan-456",
		Provider:         "DOC24",
		ProviderPlanCode: "ligue saude em dia individual",
		Origin:           "WEBHOOK_ASAAS",
		Name:             "João Silva",
		Email:            "joao@example.com",
		CPF:              "123.456.789-00",
		Phone:            "(11) 99999-9999",
		BirthDate:        "1990-05-15",
		Gender:           "1",
	}

	// Serializar
	body, err := json.Marshal(payload)
	assert.NoError(t, err)
	assert.NotEmpty(t, body)

	// Desserializar
	var received queue.ActivationPayload
	err = json.Unmarshal(body, &received)
	assert.NoError(t, err)

	// Validar
	assert.Equal(t, "cust-123", received.CustomerID)
	assert.Equal(t, "plan-456", received.PlanID)
	assert.Equal(t, "DOC24", received.Provider)
	assert.Equal(t, "João Silva", received.Name)
	assert.Equal(t, "joao@example.com", received.Email)
	assert.Equal(t, "123.456.789-00", received.CPF)
	assert.Equal(t, "(11) 99999-9999", received.Phone)
	assert.Equal(t, "1990-05-15", received.BirthDate)
	assert.Equal(t, "1", received.Gender)
	assert.Equal(t, "WEBHOOK_ASAAS", received.Origin)
	assert.Equal(t, "ligue saude em dia individual", received.ProviderPlanCode)
}

// TestActivationPayloadAllFieldsPresent - Teste que todos os campos obrigatórios estão presentes
func TestActivationPayloadAllFieldsPresent(t *testing.T) {
	payload := queue.ActivationPayload{
		CustomerID:       "cust-123",
		PlanID:           "plan-456",
		Provider:         "DOC24",
		ProviderPlanCode: "plan-code",
		Origin:           "WEBHOOK_ASAAS",
		Name:             "João Silva",
		Email:            "joao@example.com",
		CPF:              "123.456.789-00",
		Phone:            "(11) 99999-9999",
		BirthDate:        "1990-05-15",
		Gender:           "1",
	}

	body, _ := json.Marshal(payload)

	// Verificar que o JSON contém todas as chaves
	var data map[string]interface{}
	json.Unmarshal(body, &data)

	requiredFields := []string{
		"customer_id", "plan_id", "provider", "origin",
		"name", "email", "cpf", "phone", "birth_date", "gender",
	}

	for _, field := range requiredFields {
		assert.Contains(t, data, field, "field %s is missing", field)
		assert.NotEmpty(t, data[field], "field %s is empty", field)
	}
}

// TestActivationPayloadEmptyHandling - Teste com campos vazios
func TestActivationPayloadEmptyHandling(t *testing.T) {
	payload := queue.ActivationPayload{
		CustomerID: "cust-123",
		PlanID:     "", // Vazio - deveria ser preenchido
		Provider:   "DOC24",
		Name:       "João",
		Email:      "joao@example.com",
	}

	body, _ := json.Marshal(payload)

	var data map[string]interface{}
	json.Unmarshal(body, &data)

	// Mesmo com campo vazio, o JSON é válido
	assert.NotEmpty(t, body)
	assert.Equal(t, "cust-123", data["customer_id"])
	assert.Equal(t, "", data["plan_id"]) // Campo vazio mantém-se vazio
}

// TestActivationPayloadMultipleProviders - Teste com diferentes providers
func TestActivationPayloadMultipleProviders(t *testing.T) {
	providers := []string{"DOC24", "OTHER"}

	for _, provider := range providers {
		payload := queue.ActivationPayload{
			CustomerID: "cust-123",
			PlanID:     "plan-456",
			Provider:   provider,
			Name:       "João",
			Email:      "joao@example.com",
			CPF:        "123.456.789-00",
			Phone:      "(11) 99999-9999",
			BirthDate:  "1990-05-15",
			Gender:     "1",
		}

		body, err := json.Marshal(payload)
		assert.NoError(t, err)

		var received queue.ActivationPayload
		json.Unmarshal(body, &received)
		assert.Equal(t, provider, received.Provider)
	}
}

// TestActivationPayloadOriginTracking - Teste que origin é rastreado corretamente
func TestActivationPayloadOriginTracking(t *testing.T) {
	origins := []string{"WEBHOOK_ASAAS", "MANUAL", "API"}

	for _, origin := range origins {
		payload := queue.ActivationPayload{
			CustomerID: "cust-123",
			PlanID:     "plan-456",
			Provider:   "DOC24",
			Origin:     origin,
			Name:       "João",
			Email:      "joao@example.com",
			CPF:        "123.456.789-00",
			Phone:      "(11) 99999-9999",
			BirthDate:  "1990-05-15",
			Gender:     "1",
		}

		body, _ := json.Marshal(payload)

		var received queue.ActivationPayload
		json.Unmarshal(body, &received)
		assert.Equal(t, origin, received.Origin)
	}
}
