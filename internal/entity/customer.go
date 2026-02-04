package entity

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Address struct {
	Street     string `json:"street"`
	Number     string `json:"number"`
	Complement string `json:"complement"`
	District   string `json:"district"`
	City       string `json:"city"`
	State      string `json:"state"`
	ZipCode    string `json:"zip_code"`
}

type Customer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	CPF   string `json:"cpf"`

	Phone     string  `json:"phone"`
	BirthDate string  `json:"birth_date"`
	Gender    int     `json:"gender"`
	Address   Address `json:"address"`
	PlanID    string  `json:"plan_id"`

	ProductID string `json:"product_id"`

	GatewayID      string    `json:"gateway_id"`
	SubscriptionID string    `json:"subscription_id"`
	ProviderID     string    `json:"provider_id"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	TermsAccepted   bool      `json:"terms_accepted"`
	TermsAcceptedAt time.Time `json:"terms_accepted_at"`
	TermsVersion    string    `json:"terms_version"`
}

func NewCustomer(name, email, cpf, phone, birthDate string, gender int, address Address) (*Customer, error) {
	customer := &Customer{
		ID:        uuid.New().String(),
		Name:      name,
		Email:     email,
		CPF:       cpf,
		Phone:     phone,
		BirthDate: birthDate,
		Gender:    gender,
		Address:   address,

		Status:    "PENDING",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := customer.Validate(); err != nil {
		return nil, err
	}

	return customer, nil
}

func (c *Customer) Validate() error {
	if c.Name == "" {
		return errors.New("name is required")
	}
	if c.Email == "" {
		return errors.New("email is required")
	}
	if c.CPF == "" {
		return errors.New("cpf is required")
	}
	if c.Address.Street == "" {
		return errors.New("address street is required")
	}
	return nil
}

type CustomerRepositoryInterface interface {
	Create(ctx context.Context, customer *Customer) error
	FindByGatewayID(id string) (*Customer, error)
	FindByID(ctx context.Context, id string) (*Customer, error)
	CheckDuplicity(ctx context.Context, email, cpf string) (bool, error)
	Delete(ctx context.Context, id string) error
	UpdateProviderID(ctx context.Context, customerID, providerID string) error
}
type PlanRepositoryInterface interface {
	FindByID(ctx context.Context, id string) (*Plan, error)
}
