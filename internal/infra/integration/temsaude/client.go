package temsaude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/xavierca1/ligue-payments/internal/entity"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) RegisterBeneficiary(ctx context.Context, customer *entity.Customer) (string, error) {
	// TODO 1: Crie a Struct de Request baseada no PDF/JSON deles
	// type TemSaudeRequest struct { ... }

	// TODO 2: Mapeie o entity.Customer para essa struct
	// payload := TemSaudeRequest{ Nome: customer.Name ... }

	payload := TemAdesaoRequest{
		// Dados do Cliente
		Nome:           customer.Name,
		Cpf:            customer.CPF,
		CpfTitular:     customer.CPF, // Assumindo que o customer é o titular
		Email:          customer.Email,
		DataNascimento: customer.BirthDate,
		Sexo:           customer.Gender, // Entity int -> DTO int
		Telefone:       customer.Phone,
		IdentExterno:   customer.ID, // Vinculamos o UUID do nosso banco aqui

		// Endereço (Vindo do Value Object Address)
		Logradouro:     customer.Address.Street,
		NumeroEndereco: customer.Address.Number,
		Complemento:    customer.Address.Complement,
		Bairro:         customer.Address.District,
		Cidade:         customer.Address.City,
		Estado:         customer.Address.State,
		CEP:            customer.Address.ZipCode,

		CodOnix:   7065,
		Cnpj:      "87376109000106",
		TokenZeus: c.token,

		NumeroCartao:  0,
		NumeroDaSorte: 0,
		CN:            "",
	}
	// TODO 3: Faça o POST /beneficiarios (ou a rota deles)
	// Use c.http.Do(req) e lembre de passar o ctx no http.NewRequestWithContext
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("Erro ao serializar o payload: ", err)
	}

	url := fmt.Sprintf("%s/tem_adesao", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("Erro ao enviar a request: ", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		// Se der timeout ou erro de DNS, cai aqui.
		return "", fmt.Errorf("erro de comunicação com tem saude: %w", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("tem saude com erro: ", resp.StatusCode)
	}
	// TODO 4: Trate os erros (4xx, 5xx)

	// TODO 5: Retorne o ID da carteirinha gerada ou o ID interno deles
	// return response.Carteirinha, nil

	var response TemAdesaoResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("erro ao ler resposta json: %w", err)
	}

	if response.Status != "200" {
		return "", fmt.Errorf("erro de negócio tem saude: %s - %s", response.Status, response.Message)
	}

	return response.Message, nil
}
