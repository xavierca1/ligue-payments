package database

import (
	"context"
	"database/sql"

	"github.com/xavierca1/ligue-payments/internal/entity"
)

type ProductRepository struct {
	DB *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{DB: db}
}

func (r *ProductRepository) Create(ctx context.Context, p *entity.Product) error {
	query := `
		INSERT INTO products (id, name, slug, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.DB.ExecContext(ctx, query, p.ID, p.Name, p.Slug, p.CreatedAt)
	return err
}
