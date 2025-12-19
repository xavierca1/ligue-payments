package entity

import (
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID        string
	Name      string
	Slug      string
	CreatedAt time.Time
}

func NewProduct(name, slug string) *Product {
	return &Product{
		ID:        uuid.New().String(),
		Name:      name,
		Slug:      slug,
		CreatedAt: time.Now(),
	}
}
