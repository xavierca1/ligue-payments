package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

type HealthHandler struct {
	DB        *sql.DB
	RabbitMQ  *amqp091.Connection
	StartTime time.Time
}

type HealthResponse struct {
	Status       string            `json:"status"`
	Version      string            `json:"version"`
	Uptime       string            `json:"uptime"`
	Dependencies map[string]string `json:"dependencies"`
}

func NewHealthHandler(db *sql.DB, rabbitMQ *amqp091.Connection) *HealthHandler {
	return &HealthHandler{
		DB:        db,
		RabbitMQ:  rabbitMQ,
		StartTime: time.Now(),
	}
}

func (h *HealthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	deps := make(map[string]string)

	// Check Database
	if h.DB != nil {
		if err := h.DB.Ping(); err != nil {
			deps["database"] = fmt.Sprintf("unhealthy: %v", err)
		} else {
			deps["database"] = "healthy"
		}
	} else {
		deps["database"] = "not configured"
	}

	// Check RabbitMQ
	if h.RabbitMQ != nil {
		if h.RabbitMQ.IsClosed() {
			deps["rabbitmq"] = "unhealthy: connection closed"
		} else {
			deps["rabbitmq"] = "healthy"
		}
	} else {
		deps["rabbitmq"] = "not configured"
	}

	// Check Asaas API
	asaasURL := os.Getenv("ASAAS_URL")
	if asaasURL != "" {
		deps["asaas"] = "configured"
	} else {
		deps["asaas"] = "not configured"
	}

	// Determine overall status
	status := "healthy"
	for _, v := range deps {
		if v != "healthy" && v != "configured" && v != "not configured" {
			status = "degraded"
			break
		}
	}

	uptime := time.Since(h.StartTime).Round(time.Second).String()

	response := HealthResponse{
		Status:       status,
		Version:      "1.0.0",
		Uptime:       uptime,
		Dependencies: deps,
	}

	w.Header().Set("Content-Type", "application/json")
	if status == "degraded" {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	json.NewEncoder(w).Encode(response)
}
