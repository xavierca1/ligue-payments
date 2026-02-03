package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/xavierca1/ligue-payments/internal/entity"
)

type CustomerRepository struct {
	DB *sql.DB
}

func NewCustomerRepository(db *sql.DB) *CustomerRepository {
	return &CustomerRepository{DB: db}
}

func (r *CustomerRepository) FindByID(ctx context.Context, id string) (*entity.Customer, error) {
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

func (r *CustomerRepository) CheckDuplicity(ctx context.Context, email, cpf string) (bool, error) {
	cleanCPF := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(cpf, ".", ""), "-", ""), " ", "")

	query := `
		SELECT COUNT(*) FROM customers 
		WHERE email = $1 OR cpf_cnpj = $2
	`
	var count int
	err := r.DB.QueryRowContext(ctx, query, email, cleanCPF).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
func (r *CustomerRepository) FindByGatewayID(gatewayID string) (*entity.Customer, error) {

	query := `
        SELECT id, name, email, cpf_cnpj, plan_id, gateway_id
        FROM customers 
        WHERE gateway_id = $1`

	var c entity.Customer

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
	query := `
		INSERT INTO customers (
			id, 
			product_id,     -- 🆕 $2 (A causa do erro FK)
			plan_id,        -- 🔙 $3 (Adicionei de volta pra não perder)
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
			updated_at,
			terms_accepted,    -- $22
			terms_accepted_at, -- $23
			terms_version      -- $24
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, 
			$21, $22, $23, $24
		)
	`

	_, err := r.DB.ExecContext(ctx, query,
		c.ID,                 // $1
		c.ProductID,          // $2 (A CORREÇÃO: Passando o ID do Produto aqui)
		c.PlanID,             // $3 (Passando o ID do Plano aqui)
		c.Name,               // $4
		c.Email,              // $5
		c.CPF,                // $6
		c.Phone,              // $7
		c.BirthDate,          // $8
		c.Gender,             // $9
		c.GatewayID,          // $10
		c.SubscriptionID,     // $11
		c.Status,             // $12
		c.Address.Street,     // $13
		c.Address.Number,     // $14
		c.Address.Complement, // $15
		c.Address.District,   // $16
		c.Address.City,       // $17
		c.Address.State,      // $18
		c.Address.ZipCode,    // $19
		c.CreatedAt,          // $20
		c.UpdatedAt,          // $21
		c.TermsAccepted,      // $22
		c.TermsAcceptedAt,    // $23
		c.TermsVersion,       // $24
	)

	if err != nil {
		return fmt.Errorf("erro no insert do cliente: %w", err)
	}

	return nil
}

func (r *CustomerRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM customers WHERE id = $1`

	_, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("erro ao deletar cliente %s: %w", id, err)
	}

	return nil
}


func (r *CustomerRepository) UpdateProviderID(ctx context.Context, customerID, providerID string) error {
	query := `UPDATE customers SET provider_id = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.DB.ExecContext(ctx, query, providerID, customerID)
	if err != nil {
		return fmt.Errorf("erro ao atualizar provider_id: %w", err)
	}
	return nil
}
