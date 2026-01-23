package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/xavierca1/ligue-payments/internal/entity"
)

type SubscriptionRepository struct {
	DB *sql.DB
}

func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{DB: db}
}

func (r *SubscriptionRepository) GetStatusByCustomerID(customerID string) (string, error) {
	query := `SELECT status FROM subscriptions WHERE customer_id = $1`

	var status string
	err := r.DB.QueryRow(query, customerID).Scan(&status)

	if err != nil {
		if err == sql.ErrNoRows {
			return "NOT_FOUND", nil
		}
		return "", fmt.Errorf("erro ao consultar status no banco: %v", err)
	}

	return status, nil
}

func (r *SubscriptionRepository) UpdateStatus(customerID string, status string) error {
	query := `UPDATE subscriptions SET status = $1, updated_at = $2 WHERE customer_id = $3`

	_, err := r.DB.Exec(query, status, time.Now(), customerID)

	if err != nil {
		return fmt.Errorf("erro ao atualizar status da assinatura do cliente %s: %w", customerID, err)
	}
	return nil
}

func (r *SubscriptionRepository) Create(ctx context.Context, c *entity.Subscription) error {
	query := ` 
		INSERT INTO subscriptions(
		id,
		plan_id,
		customer_id,
		product_id,
		amount,
		status, 
		interval, 
		next_billing_date,
		payment_method_id, 
		created_at,
		updated_at

		)
	VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`

	_, err := r.DB.ExecContext(ctx, query,
		c.ID,
		c.PlanID,
		c.CustomerID,
		c.ProductID,
		c.Amount,
		c.Status,
		c.Interval,
		c.NextBillingDate,
		c.PaymentMethodID,
		c.CreatedAt,
		c.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("Erro ao insert no subscription: %w", err)
	}
	return nil

}
