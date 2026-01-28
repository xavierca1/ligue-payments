package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/xavierca1/ligue-payments/internal/entity"
)

type SubscriptionRepository struct {
	DB *sql.DB
}

func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{DB: db}
}

func (r *SubscriptionRepository) Create(ctx context.Context, sub *entity.Subscription) error {
	pid := strings.TrimSpace(sub.ProductID)

	query := `
		INSERT INTO subscriptions (
			id, 
			product_id,        -- 🆕 $2 (Obrigatório)
			customer_id,       -- $3
			plan_id,           -- $4
			amount,            
			status, 
			next_billing_date, 
			created_at, 
			updated_at,
            payment_method_id  -- $10 (Pode ser Null)
		) VALUES (
			$1, 
            $2::uuid,          -- 🆕 Forçamos o UUID aqui
            $3, $4, $5, $6, $7, $8, $9, 
            NULLIF($10, '')::uuid
		)
	`

	fmt.Printf(" [REPO SUBSCRIPTION] Salvando ID=%s | ProductID=%s\n", sub.ID, pid)

	_, err := r.DB.ExecContext(
		ctx,
		query,
		sub.ID,              // $1
		pid,                 // $2 (AQUI ESTÁ A CORREÇÃO)
		sub.CustomerID,      // $3
		sub.PlanID,          // $4
		sub.Amount,          // $5
		sub.Status,          // $6
		sub.NextBillingDate, // $7
		sub.CreatedAt,       // $8
		sub.UpdatedAt,       // $9

		"",
	)

	if err != nil {
		return fmt.Errorf("FALHA AO CRIAR ASSINATURA: %w", err)
	}

	return nil
}

func (r *SubscriptionRepository) UpdateStatus(id string, status string) error {
	// Atualiza status baseado no CustomerID
	query := `UPDATE subscriptions SET status = $1, updated_at = NOW() WHERE customer_id = $2`
	_, err := r.DB.Exec(query, status, id)
	return err
}

func (r *SubscriptionRepository) GetStatusByCustomerID(customerID string) (string, error) {
	query := `SELECT status FROM subscriptions WHERE customer_id = $1 ORDER BY created_at DESC LIMIT 1`
	var status string
	err := r.DB.QueryRow(query, customerID).Scan(&status)
	if err != nil {
		return "", err
	}
	return status, nil
}
func (r *SubscriptionRepository) FindLastByCustomerID(ctx context.Context, customerID string) (*entity.Subscription, error) {
	query := `
		SELECT 
			id, 
			customer_id, 
			plan_id,      -- 👈 O culpado! Tem que estar aqui
			product_id,   -- Importante também
			amount, 
			status, 
			next_billing_date, 
			created_at, 
			updated_at
		FROM subscriptions 
		WHERE customer_id = $1 
		ORDER BY created_at DESC 
		LIMIT 1
	`

	var sub entity.Subscription

	err := r.DB.QueryRowContext(ctx, query, customerID).Scan(
		&sub.ID,
		&sub.CustomerID,
		&sub.PlanID, // 👈 E tem que ter o ponteiro aqui
		&sub.ProductID,
		&sub.Amount,
		&sub.Status,
		&sub.NextBillingDate,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("assinatura não encontrada: %w", err)
	}

	return &sub, nil
}
