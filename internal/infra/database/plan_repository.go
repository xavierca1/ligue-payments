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
	// 1. A Query deve usar 'price_cents' (nome no banco), não 'price'
	query := "SELECT id, name, price_cents, provider_plan_code FROM plans WHERE id = $1"

	var plan entity.Plan

	// 2. Executa a query
	row := r.DB.QueryRowContext(ctx, query, id)

	// 3. Mapeia o resultado (price_cents do banco vai para plan.Price da struct)
	err := row.Scan(
		&plan.ID,
		&plan.Name,
		&plan.ProviderPlanCode,
		&plan.PriceCents,
	)

	if err != nil {
		return nil, err
	}

	return &plan, nil
}
