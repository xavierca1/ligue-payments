package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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
		SELECT id, name, email, cpf_cnpj, COALESCE(phone, ''), COALESCE(birth_date, ''), COALESCE(gender, 0),
		       COALESCE(marital_status, ''),
		       COALESCE(street, ''), COALESCE(number, ''), COALESCE(complement, ''),
		       COALESCE(district, ''), COALESCE(city, ''), COALESCE(state, ''), COALESCE(zip_code, '')
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
		&c.MaritalStatus,
		&c.Address.Street,
		&c.Address.Number,
		&c.Address.Complement,
		&c.Address.District,
		&c.Address.City,
		&c.Address.State,
		&c.Address.ZipCode,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("cliente não encontrado com id %s", id)
		}
		return nil, fmt.Errorf("erro ao buscar cliente: %w", err)
	}

	return &c, nil

}

func (r *CustomerRepository) FindByCPF(ctx context.Context, cpf string) (*entity.Customer, error) {
	cleanCPF := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(cpf, ".", ""), "-", ""), " ", "")

	query := `
		SELECT id, name, email, cpf_cnpj, COALESCE(phone, ''), COALESCE(birth_date, ''), COALESCE(gender, 0),
		       COALESCE(marital_status, ''),
		       COALESCE(gateway_id, ''), COALESCE(subscription_id, ''), COALESCE(status, ''),
		       COALESCE(street, ''), COALESCE(number, ''), COALESCE(complement, ''),
		       COALESCE(district, ''), COALESCE(city, ''), COALESCE(state, ''), COALESCE(zip_code, '')
		FROM customers
		WHERE cpf_cnpj = $1
		LIMIT 1
	`

	row := r.DB.QueryRowContext(ctx, query, cleanCPF)

	var c entity.Customer
	err := row.Scan(
		&c.ID,
		&c.Name,
		&c.Email,
		&c.CPF,
		&c.Phone,
		&c.BirthDate,
		&c.Gender,
		&c.MaritalStatus,
		&c.GatewayID,
		&c.SubscriptionID,
		&c.Status,
		&c.Address.Street,
		&c.Address.Number,
		&c.Address.Complement,
		&c.Address.District,
		&c.Address.City,
		&c.Address.State,
		&c.Address.ZipCode,
	)

	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (r *CustomerRepository) FindByEmailAndProductID(ctx context.Context, email, productID string) (*entity.Customer, error) {
	query := `
		SELECT id, name, email, cpf_cnpj, COALESCE(phone, ''), COALESCE(birth_date, ''), COALESCE(gender, 0),
		       COALESCE(marital_status, ''),
		       COALESCE(gateway_id, ''), COALESCE(subscription_id, ''), COALESCE(status, ''),
		       COALESCE(street, ''), COALESCE(number, ''), COALESCE(complement, ''),
		       COALESCE(district, ''), COALESCE(city, ''), COALESCE(state, ''), COALESCE(zip_code, '')
		FROM customers
		WHERE LOWER(TRIM(email)) = LOWER(TRIM($1))
		  AND LOWER(TRIM(product_id::text)) = LOWER(TRIM($2))
		LIMIT 1
	`

	row := r.DB.QueryRowContext(ctx, query, strings.TrimSpace(email), strings.TrimSpace(productID))

	var c entity.Customer
	err := row.Scan(
		&c.ID,
		&c.Name,
		&c.Email,
		&c.CPF,
		&c.Phone,
		&c.BirthDate,
		&c.Gender,
		&c.MaritalStatus,
		&c.GatewayID,
		&c.SubscriptionID,
		&c.Status,
		&c.Address.Street,
		&c.Address.Number,
		&c.Address.Complement,
		&c.Address.District,
		&c.Address.City,
		&c.Address.State,
		&c.Address.ZipCode,
	)

	if err != nil {
		return nil, err
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
	// Helper para converter string vazia em nil (evita erro de UUID inválido)
	toNull := func(s string) interface{} {
		if s == "" {
			return nil
		}
		return s
	}

	// DEBUG: Log detalhado de todos os IDs antes de tentar inserir
	log.Printf("[DEBUG] Inserindo Customer no DB:")
	log.Printf("-> ID: '%s', ProductID: '%s', PlanID: '%s', GatewayID: '%s', SubID: '%s'",
		c.ID, c.ProductID, c.PlanID, c.GatewayID, c.SubscriptionID)

	query := `
        INSERT INTO customers (
            id, product_id, plan_id, name, email, cpf_cnpj, phone, birth_date,
            gender, marital_status, gateway_id, subscription_id, status, street, number, complement,
            district, city, state, zip_code, created_at, updated_at, terms_accepted,
            terms_accepted_at, terms_version
        )
        VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
            $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
            $21, $22, $23, $24, $25
        )
    `

	_, err := r.DB.ExecContext(ctx, query,
		c.ID,                     // $1 (UUID obrigatório)
		toNull(c.ProductID),      // $2 (UUID ou NULL)
		toNull(c.PlanID),         // $3 (UUID ou NULL)
		c.Name,                   // $4
		c.Email,                  // $5
		c.CPF,                    // $6
		c.Phone,                  // $7
		c.BirthDate,              // $8
		c.Gender,                 // $9
		c.MaritalStatus,          // $10
		toNull(c.GatewayID),      // $11
		toNull(c.SubscriptionID), // $12
		c.Status,                 // $13
		c.Address.Street,         // $14
		c.Address.Number,         // $15
		c.Address.Complement,     // $16
		c.Address.District,       // $17
		c.Address.City,           // $18
		c.Address.State,          // $19
		c.Address.ZipCode,        // $20
		c.CreatedAt,              // $21
		c.UpdatedAt,              // $22
		c.TermsAccepted,          // $23
		c.TermsAcceptedAt,        // $24
		c.TermsVersion,           // $25
	)

	if err != nil {
		// LOG DE ERRO REAL
		log.Printf("[ERROR] SQL INSERT falhou: %v", err)
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
func (r *CustomerRepository) UpdateGatewayID(ctx context.Context, customerID, gatewayID string) error {
	query := `UPDATE customers SET gateway_id = $1, updated_at = NOW() WHERE id = $2`
	result, err := r.DB.ExecContext(ctx, query, gatewayID, customerID)
	if err != nil {
		return fmt.Errorf("erro ao atualizar gateway_id: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	log.Printf("🔄 UpdateGatewayID: customer_id=%s gateway_id=%s rows_affected=%d", customerID, gatewayID, rowsAffected)
	return nil
}

func (r *CustomerRepository) UpdateStatus(ctx context.Context, customerID, status string) error {
	query := `UPDATE customers SET status = $1, updated_at = NOW() WHERE id = $2`
	result, err := r.DB.ExecContext(ctx, query, status, customerID)
	if err != nil {
		return fmt.Errorf("erro ao atualizar status do customer: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	log.Printf("🔄 UpdateStatus (customer): customer_id=%s status=%s rows_affected=%d", customerID, status, rowsAffected)
	return nil
}
