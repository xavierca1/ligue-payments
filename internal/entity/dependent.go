package entity

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Dependent representa um dependente vinculado a um cliente titular
type Dependent struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customer_id"`
	Name       string    `json:"name"`
	CPF        string    `json:"cpf"`
	BirthDate  string    `json:"birth_date"` // Formato: YYYY-MM-DD
	Gender     int       `json:"gender"`     // 1=Masculino, 2=Feminino, 3=Outro
	Kinship    string    `json:"kinship"`    // FILHO, CONJUGE, PAI, MAE, etc
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NewDependent cria um novo dependente com validações básicas
func NewDependent(customerID, name, cpf, birthDate string, gender int, kinship string) (*Dependent, error) {
	if customerID == "" {
		return nil, errors.New("customer_id é obrigatório")
	}
	if name == "" {
		return nil, errors.New("name é obrigatório")
	}
	if cpf == "" {
		return nil, errors.New("cpf é obrigatório")
	}
	if birthDate == "" {
		return nil, errors.New("birth_date é obrigatório")
	}
	if gender < 1 || gender > 3 {
		return nil, errors.New("gender deve ser 1, 2 ou 3")
	}
	if kinship == "" {
		return nil, errors.New("kinship é obrigatório")
	}

	return &Dependent{
		ID:         uuid.New().String(),
		CustomerID: customerID,
		Name:       name,
		CPF:        cpf,
		BirthDate:  birthDate,
		Gender:     gender,
		Kinship:    kinship,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}, nil
}

// DependentRepositoryInterface define os métodos para o repositório de dependentes
type DependentRepositoryInterface interface {
	Create(ctx context.Context, dependent *Dependent) error
	FindByCustomerID(ctx context.Context, customerID string) ([]*Dependent, error)
	FindByID(ctx context.Context, id string) (*Dependent, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, dependent *Dependent) error
}
