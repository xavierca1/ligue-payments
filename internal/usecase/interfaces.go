package usecase

import (
	"context"

	"github.com/xavierca1/ligue-payments/internal/entity"
)

type BenefitProvider interface {
	RegisterBeneficiary(ctx context.Context, c *entity.Customer) (string, error)
}
