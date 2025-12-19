package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/paemuri/brdoc"
)

type Customer struct {
	ID        string
	Name      string
	Email     string
	CPF       string
	CreatedAt time.Time
	UpdatedAt time.Time
}

var (
	ErrEmailAlreadyExists = errors.New("email já cadastrado para este produto")
	ErrInvalidCPF         = errors.New("cpf inválido")
)

func NewCustomer(name, email, cpf string) (*Customer, error) {

	if !brdoc.IsCPF(cpf) {
		return nil, ErrInvalidCPF
	}

	return &Customer{
		ID:        uuid.New().String(),
		Name:      name,
		Email:     email,
		CPF:       cpf,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}
