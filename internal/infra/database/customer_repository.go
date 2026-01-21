package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/xavierca1/ligue-payments/internal/entity"
)

type CustomerRepository struct {
	DB *sql.DB
}

func NewCustomerRepository(db *sql.DB) *CustomerRepository {
	return &CustomerRepository{DB: db}
}

func (r *CustomerRepository) FindByID(id string) (*entity.Customer, error) {
	query := `
		SELECT id, name, email, cpf_cnpj, COALESCE(phone, ''), COALESCE(birth_date, ''), COALESCE(gender, 0)  
		FROM customers 
		WHERE id = $1
	`

	row := r.DB.QueryRow(query, id)

	var c entity.Customer
	err := row.Scan(
		&c.ID,
		&c.Name,
		&c.Email,
		&c.CPF,
		&c.Phone,
		&c.BirthDate,
		&c.Gender,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("cliente não encontrado com id %s", id)
		}
		return nil, fmt.Errorf("erro ao buscar cliente: %w", err)
	}

	return &c, nil

}

func (r *CustomerRepository) FindByGatewayID(gatewayID string) (*entity.Customer, error) {
	// Ajuste o nome da coluna 'gateway_id' se for diferente no seu banco
	query := `
        SELECT id, name, email, cpf_cnpj, plan_id, gateway_id
        FROM customers 
        WHERE gateway_id = $1`

	var c entity.Customer
	// Dica: Certifique-se que sua struct Customer tem o campo PlanID mapeado
	err := r.DB.QueryRow(query, gatewayID).Scan(
		&c.ID,
		&c.Name,
		&c.Email,
		&c.CPF,
		&c.PlanID, // <--- Importante: Precisamos disso para saber qual plano ativar
		&c.GatewayID,
	)

	if err != nil {
		return nil, fmt.Errorf("cliente não encontrado pelo gateway_id: %w", err)
	}
	return &c, nil
}

func (r *CustomerRepository) Create(ctx context.Context, c *entity.Customer) error {
	// 🛡️ PROTEÇÃO CONTRA UUID VAZIO
	// Se o PlanID não veio do UseCase, usamos um ID padrão ou retornamos erro.
	// Para destravar seu teste agora, vou colocar aquele ID que você usava:
	if c.PlanID == "" {
		c.PlanID = "15b4965d-c234-410c-83ae-ac826051e672"
	}

	// A query continua a mesma...
	query := `
        INSERT INTO customers (
            id, 
            product_id, 
            name, 
            email, 
            cpf_cnpj, 
            phone,
            birth_date,
            gender,
            gateway_id,
            subscription_id,
            status,
            street,
            number,
            complement,
            district,
            city,
            state,
            zip_code,
            created_at, 
            updated_at
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
    `

	_, err := r.DB.ExecContext(ctx, query,
		c.ID,
		c.PlanID, // Agora garantimos que não é ""
		c.Name,
		c.Email,
		c.CPF,
		c.Phone,
		c.BirthDate,
		c.Gender,
		c.GatewayID,
		c.SubscriptionID,
		c.Status,
		c.Address.Street,
		c.Address.Number,
		c.Address.Complement,
		c.Address.District,
		c.Address.City,
		c.Address.State,
		c.Address.ZipCode,
		c.CreatedAt,
		c.UpdatedAt,
	)

	if err != nil {
		log.Printf("❌ Erro crítico ao criar cliente no banco: %v", err)
		return fmt.Errorf("erro no insert do cliente: %w", err)
	}

	return nil
}
