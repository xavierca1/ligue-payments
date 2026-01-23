package entity

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	ID              string    `json:"id"`
	CustomerID      string    `json:"customer_id"`
	PlanID          string    `json:"plan_id"`
	ProductID       string    `json:"product_id"`
	Amount          int       `json:"amount"`   // Em centavos, como o Asaas gosta
	Status          string    `json:"status"`   // PENDING, ACTIVE, etc
	Interval        string    `json:"interval"` // MONTHLY, YEARLY
	NextBillingDate time.Time `json:"next_billing_date"`
	PaymentMethodID string    `json:"payment_method_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
type SubscriptionRepository interface {
	Create(ctx context.Context, sub *Subscription) error
	GetStatusByCustomerID(customerID string) (string, error)
	UpdateStatus(customerID string, status string) error
}

// NewSubscription cria uma nova instância com ID e Timestamps
func NewSubscription(customerID, planID string, amount int) *Subscription {
	return &Subscription{
		ID:         uuid.New().String(),
		CustomerID: customerID,
		PlanID:     planID,
		Amount:     amount,
		Status:     "PENDING",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}
