package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/xavierca1/ligue-payments/internal/entity"
)

type DependentRepository struct {
	DB *sql.DB
}

func NewDependentRepository(db *sql.DB) *DependentRepository {
	return &DependentRepository{DB: db}
}

func (r *DependentRepository) Create(ctx context.Context, dependent *entity.Dependent) error {
	query := `INSERT INTO dependents (id, customer_id, name, cpf, birth_date, gender, kinship, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.DB.ExecContext(ctx, query, dependent.ID, dependent.CustomerID, dependent.Name, dependent.CPF, dependent.BirthDate, dependent.Gender, dependent.Kinship, dependent.CreatedAt, dependent.UpdatedAt)
	if err != nil {
		return fmt.Errorf("erro ao criar dependente: %w", err)
	}
	return nil
}

func (r *DependentRepository) FindByCustomerID(ctx context.Context, customerID string) ([]*entity.Dependent, error) {
	query := `SELECT id, customer_id, name, cpf, birth_date, gender, kinship, created_at, updated_at FROM dependents WHERE customer_id = $1 ORDER BY created_at ASC`
	rows, err := r.DB.QueryContext(ctx, query, customerID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar dependentes: %w", err)
	}
	defer rows.Close()
	var dependents []*entity.Dependent
	for rows.Next() {
		dep := &entity.Dependent{}
		err := rows.Scan(&dep.ID, &dep.CustomerID, &dep.Name, &dep.CPF, &dep.BirthDate, &dep.Gender, &dep.Kinship, &dep.CreatedAt, &dep.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("erro ao escanear dependente: %w", err)
		}
		dependents = append(dependents, dep)
	}
	return dependents, nil
}

func (r *DependentRepository) FindByID(ctx context.Context, id string) (*entity.Dependent, error) {
	query := `SELECT id, customer_id, name, cpf, birth_date, gender, kinship, created_at, updated_at FROM dependents WHERE id = $1`
	dep := &entity.Dependent{}
	err := r.DB.QueryRowContext(ctx, query, id).Scan(&dep.ID, &dep.CustomerID, &dep.Name, &dep.CPF, &dep.BirthDate, &dep.Gender, &dep.Kinship, &dep.CreatedAt, &dep.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("dependente não encontrado")
		}
		return nil, fmt.Errorf("erro ao buscar dependente: %w", err)
	}
	return dep, nil
}

func (r *DependentRepository) Update(ctx context.Context, dependent *entity.Dependent) error {
	query := `UPDATE dependents SET name = $2, cpf = $3, birth_date = $4, gender = $5, kinship = $6, updated_at = $7 WHERE id = $1`
	_, err := r.DB.ExecContext(ctx, query, dependent.ID, dependent.Name, dependent.CPF, dependent.BirthDate, dependent.Gender, dependent.Kinship, dependent.UpdatedAt)
	if err != nil {
		return fmt.Errorf("erro ao atualizar dependente: %w", err)
	}
	return nil
}

func (r *DependentRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM dependents WHERE id = $1`
	_, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("erro ao deletar dependente: %w", err)
	}
	return nil
}
