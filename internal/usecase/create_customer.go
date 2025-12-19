package usecase

import (
	"context"

	"github.com/xavierca1/ligue-payments/internal/entity"
)

type CustomerRepositoryInterface interface {
	Create(ctx context.Context, c *entity.Customer) error
}

type CreateCustomerUseCase struct {
	Repo CustomerRepositoryInterface
}

func NewCreateCustomerUseCase(repo CustomerRepositoryInterface) *CreateCustomerUseCase {
	return &CreateCustomerUseCase{Repo: repo}
}

func (uc *CreateCustomerUseCase) Execute(ctx context.Context, input CreateCustomerInput) (*CreateCustomerOutput, error) {
	customer := entity.NewCustomer(input.Name, input.Email, input.CPF)

	err := uc.Repo.Create(ctx, customer)
	if err != nil {
		return nil, err
	}

	return &CreateCustomerOutput{
		ID:    customer.ID,
		Name:  customer.Name,
		Email: customer.Email,
	}, nil
}
