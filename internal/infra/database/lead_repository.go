package database

import (
	"context"
	"database/sql"

	"github.com/xavierca1/ligue-payments/internal/entity"
)

type LeadRepository struct {
	DB *sql.DB
}


func (r *LeadRepository) Upsert(ctx context.Context, lead *entity.Lead) error {
	query := `
		INSERT INTO leads (email, name, phone, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (email)
		DO UPDATE SET
			name = COALESCE(EXCLUDED.name, leads.name),
			phone = COALESCE(EXCLUDED.phone, leads.phone),
			updated_at = NOW()
		RETURNING id, created_at, updated_at, status, email_stage
	`

	err := r.DB.QueryRowContext(
		ctx,
		query,
		lead.Email,
		nullString(lead.Name),
		nullString(lead.Phone),
	).Scan(
		&lead.ID,
		&lead.CreatedAt,
		&lead.UpdatedAt,
		&lead.Status,
		&lead.EmailStage,
	)

	return err
}


func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
