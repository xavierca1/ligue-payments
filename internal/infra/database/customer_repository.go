package database

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/xavierca1/ligue-payments/internal/entity"
)

type CustomerRepository struct {
	DB *sql.DB
}

func NewCustomerRepository(db *sql.DB) *CustomerRepository {
	return &CustomerRepository{DB: db}
}

func (r *CustomerRepository) Create(ctx context.Context, c *entity.Customer) error {
	query := `
		INSERT INTO customers (id, product_id, name, email, cpf_cnpj, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	productID := "15b4965d-c234-410c-83ae-ac826051e672"

	_, err := r.DB.ExecContext(ctx, query,
		c.ID,
		productID,
		c.Name,
		c.Email,
		c.CPF,
		c.CreatedAt,
		c.UpdatedAt,
	)

	if err != nil {

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return entity.ErrEmailAlreadyExists
			}
		}

		log.Printf("Erro crítico no banco: %v", err)
		return err
	}

	return nil
}
