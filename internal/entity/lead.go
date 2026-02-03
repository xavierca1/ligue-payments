package entity

import (
	"context"
	"time"
)


type Lead struct {
	ID              string     `json:"id"`
	Email           string     `json:"email"`
	Name            string     `json:"name,omitempty"`
	Phone           string     `json:"phone,omitempty"`
	Status          string     `json:"status"` // PENDING, RECOVERED, CONVERTED
	EmailStage      int        `json:"email_stage"`
	LastEmailSentAt *time.Time `json:"last_email_sent_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}


type LeadRepositoryInterface interface {

	Upsert(ctx context.Context, lead *Lead) error
}
