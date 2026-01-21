package entity

import "errors"

var ErrPlanNotFound = errors.New("plano não encontrado")

type Plan struct {
	ID               string
	Name             string
	ProviderPlanCode string
	PriceCents       int
	Provider         string
}
