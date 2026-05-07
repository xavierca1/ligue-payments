package database

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xavierca1/ligue-payments/internal/usecase"
)

type CouponRepository struct {
	DB *sql.DB
}

func NewCouponRepository(db *sql.DB) *CouponRepository {
	return &CouponRepository{DB: db}
}

func (r *CouponRepository) GetActiveCoupon(ctx context.Context, code string) (*usecase.CouponDetails, error) {
	normalizedCode := strings.ToUpper(strings.TrimSpace(code))

	query := `
		SELECT code, seller_name, COALESCE(discount_percent, 10)
		FROM coupons
		WHERE UPPER(code) = $1
		  AND is_active = TRUE
		  AND (starts_at IS NULL OR starts_at <= NOW())
		  AND (ends_at IS NULL OR ends_at >= NOW())
		LIMIT 1
	`

	var details usecase.CouponDetails
	if err := r.DB.QueryRowContext(ctx, query, normalizedCode).Scan(&details.Code, &details.SellerName, &details.DiscountPercent); err != nil {
		return nil, err
	}

	if details.DiscountPercent <= 0 {
		details.DiscountPercent = 10
	}

	return &details, nil
}

func (r *CouponRepository) TrackSale(ctx context.Context, sale usecase.CouponSaleRecord) error {
	normalizedCode := strings.ToUpper(strings.TrimSpace(sale.CouponCode))
	if normalizedCode == "" {
		return nil
	}

	if strings.TrimSpace(sale.SellerName) == "" {
		querySeller := `SELECT seller_name FROM coupons WHERE UPPER(code) = $1 LIMIT 1`
		_ = r.DB.QueryRowContext(ctx, querySeller, normalizedCode).Scan(&sale.SellerName)
	}

	query := `
		INSERT INTO coupon_sales (
			id,
			coupon_code,
			seller_name,
			customer_id,
			subscription_id,
			plan_id,
			original_amount_cents,
			discount_percent,
			discount_amount_cents,
			final_amount_cents,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`

	_, err := r.DB.ExecContext(
		ctx,
		query,
		uuid.New().String(),
		normalizedCode,
		strings.TrimSpace(sale.SellerName),
		sale.CustomerID,
		sale.SubscriptionID,
		sale.PlanID,
		sale.OriginalAmountCents,
		sale.DiscountPercent,
		sale.DiscountAmountCents,
		sale.FinalAmountCents,
		time.Now(),
	)

	return err
}
