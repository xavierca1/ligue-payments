package usecase

import (
	"context"
	"fmt"
)


type Transaction struct {
	operations     []Operation
	compensations  []Compensation
	customerID     string
	subscriptionID string
	customerRepoFn func(context.Context, string) error // fn para rollback de customer
	subRepoFn      func(context.Context, string) error // fn para rollback de subscription
}

type Operation struct {
	Name string
	Fn   func(context.Context) error
}

type Compensation struct {
	Name string
	Fn   func(context.Context) error
}


func NewTransaction() *Transaction {
	return &Transaction{
		operations:    []Operation{},
		compensations: []Compensation{},
	}
}


func (t *Transaction) AddOperation(name string, fn func(context.Context) error) {
	t.operations = append(t.operations, Operation{name, fn})
}


func (t *Transaction) AddCompensation(name string, fn func(context.Context) error) {
	t.compensations = append(t.compensations, Compensation{name, fn})
}



func (t *Transaction) Execute(ctx context.Context) error {
	executedOps := 0

	for i, op := range t.operations {
		if err := op.Fn(ctx); err != nil {

			t.rollback(ctx, i)
			return fmt.Errorf("operation '%s' failed: %w (rolled back %d operations)", op.Name, err, i)
		}
		executedOps++
	}

	return nil
}


func (t *Transaction) rollback(ctx context.Context, failedAtIndex int) {

	for i := failedAtIndex - 1; i >= 0; i-- {
		if i < len(t.compensations) {
			comp := t.compensations[i]
			if err := comp.Fn(ctx); err != nil {

				fmt.Printf("⚠️ WARNING: Compensation '%s' failed: %v (inconsistency risk!)\n", comp.Name, err)
			}
		}
	}
}
