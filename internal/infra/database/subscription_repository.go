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
			payment_method,    -- $10
            payment_method_id  -- $11 (Pode ser Null)
		) VALUES (
			$1, 
            $2::uuid,          -- 🆕 Forçamos o UUID aqui
            $3, $4, $5, $6, $7, $8, $9, $10,
			NULLIF($11, '')
		)
	`

	fmt.Printf(" [REPO SUBSCRIPTION] Salvando ID=%s | ProductID=%s | PaymentMethod=%s\n",
		sub.ID, pid, sub.PaymentMethod)

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
		sub.PaymentMethod,   // $10
		sub.PaymentMethodID, // $11
	)

	if err != nil {
		return fmt.Errorf("FALHA AO CRIAR ASSINATURA: %w", err)
	}

	return nil
}

func (r *SubscriptionRepository) UpdateStatus(id string, status string) error {
	// Tentamos atualizar por customer_id (ID local do cliente)
	query := `UPDATE subscriptions SET status = $1, updated_at = NOW() WHERE customer_id = $2`
	result, err := r.DB.Exec(query, status, id)
	if err != nil {
		fmt.Printf("❌ UpdateStatus: SQL error (customer_id=%s) para status=%s: %v\n", id, status, err)
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("🔄 UpdateStatus (customer_id): customer_id=%s status=%s rows_affected=%d\n", id, status, rowsAffected)
	if rowsAffected > 0 {
		return nil
	}

	// Se nada foi atualizado, talvez o identificador recebido seja o payment_method_id (gateway subscription/payment id)
	query = `UPDATE subscriptions SET status = $1, updated_at = NOW() WHERE payment_method_id = $2`
	result, err = r.DB.Exec(query, status, id)
	if err != nil {
		fmt.Printf("❌ UpdateStatus: SQL error (payment_method_id=%s) para status=%s: %v\n", id, status, err)
		return err
	}
	rowsAffected, _ = result.RowsAffected()
	fmt.Printf("🔄 UpdateStatus (payment_method_id): payment_method_id=%s status=%s rows_affected=%d\n", id, status, rowsAffected)
	if rowsAffected > 0 {
		return nil
	}

	// Por fim, tente atualizar pela própria subscription.id
	query = `UPDATE subscriptions SET status = $1, updated_at = NOW() WHERE id = $2`
	result, err = r.DB.Exec(query, status, id)
	if err != nil {
		fmt.Printf("❌ UpdateStatus: SQL error (id=%s) para status=%s: %v\n", id, status, err)
		return err
	}
	rowsAffected, _ = result.RowsAffected()
	fmt.Printf("🔄 UpdateStatus (id): id=%s status=%s rows_affected=%d\n", id, status, rowsAffected)
	if rowsAffected == 0 {
		fmt.Printf("⚠️ UpdateStatus: nenhuma subscription encontrada para nenhum dos campos com identificador=%s\n", id)
	}
	return nil
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

func (r *SubscriptionRepository) DeleteByID(ctx context.Context, id string) error {
	query := `DELETE FROM subscriptions WHERE id = $1`
	_, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("erro ao deletar subscription %s: %w", id, err)
	}
	return nil
}

func (r *SubscriptionRepository) FindLastByCustomerID(ctx context.Context, customerID string) (*entity.Subscription, error) {
	query := `
		SELECT
			id,
			customer_id,
			plan_id,
			product_id,
			amount,
			status,
			next_billing_date,
			created_at,
			updated_at,
			COALESCE(payment_method_id, ''),
			COALESCE(payment_method, '')
		FROM subscriptions
		WHERE customer_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var sub entity.Subscription

	err := r.DB.QueryRowContext(ctx, query, customerID).Scan(
		&sub.ID,
		&sub.CustomerID,
		&sub.PlanID,
		&sub.ProductID,
		&sub.Amount,
		&sub.Status,
		&sub.NextBillingDate,
		&sub.CreatedAt,
		&sub.UpdatedAt,
		&sub.PaymentMethodID,
		&sub.PaymentMethod,
	)

	if err != nil {
		return nil, fmt.Errorf("assinatura não encontrada: %w", err)
	}

	return &sub, nil
}
