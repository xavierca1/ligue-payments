package usecase

import "context"

type CouponDetails struct {
	Code            string
	SellerName      string
	DiscountPercent int
}

type CouponSaleRecord struct {
	CouponCode          string
	SellerName          string
	CustomerID          string
	SubscriptionID      string
	PlanID              string
	OriginalAmountCents int
	DiscountPercent     int
	DiscountAmountCents int
	FinalAmountCents    int
}

type CouponTracker interface {
	GetActiveCoupon(ctx context.Context, code string) (*CouponDetails, error)
	TrackSale(ctx context.Context, sale CouponSaleRecord) error
}

var couponTracker CouponTracker

func SetCouponTracker(tracker CouponTracker) {
	couponTracker = tracker
}

func ValidateCoupon(ctx context.Context, code string) (*CouponDetails, error) {
	if couponTracker == nil {
		return nil, &DomainError{
			Code:    "COUPON_UNAVAILABLE",
			Message: "serviço de cupom não configurado",
		}
	}

	return couponTracker.GetActiveCoupon(ctx, code)
}
