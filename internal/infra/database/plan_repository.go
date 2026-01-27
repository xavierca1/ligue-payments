package database

import (
	"context"
	"database/sql"

	"github.com/xavierca1/ligue-payments/internal/entity"
)

type PlanRepository struct {
	DB *sql.DB
}

func NewPlanRepository(db *sql.DB) *PlanRepository {
	return &PlanRepository{DB: db}
}

func (r *PlanRepository) FindByID(ctx context.Context, id string) (*entity.Plan, error) {
	// 1. Adicione "product_id" na Query
	query := `SELECT id, name, price_cents, provider, product_id FROM plans WHERE id = $1`

	var plan entity.Plan

	// 2. Adicione "&plan.ProductID" no Scan (na mesma ordem!)
	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&plan.ID,
		&plan.Name,
		&plan.PriceCents,
		&plan.Provider,
		&plan.ProductID,
	)

	if err != nil {
		return nil, err
	}
	return &plan, nil
}
