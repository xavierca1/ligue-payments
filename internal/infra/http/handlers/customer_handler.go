package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/xavierca1/ligue-payments/internal/entity"
	"github.com/xavierca1/ligue-payments/internal/usecase"
)

type CustomerHandler struct {
	CreateCustomerUC *usecase.CreateCustomerUseCase
	SubRepo          entity.SubscriptionRepository
	CustomerRepo     interface {
		FindByCPF(ctx context.Context, cpf string) (*entity.Customer, error)
	}
}

func NewCustomerHandler(uc *usecase.CreateCustomerUseCase, subRepo entity.SubscriptionRepository, customerRepo interface {
	FindByCPF(ctx context.Context, cpf string) (*entity.Customer, error)
}) *CustomerHandler {
	return &CustomerHandler{
		CreateCustomerUC: uc,
		SubRepo:          subRepo,
		CustomerRepo:     customerRepo,
	}
}

func (h *CustomerHandler) CreateCheckoutHandler(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()
	var input usecase.CreateCustomerInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("[checkout] invalid_json remote=%s err=%v duration=%s", r.RemoteAddr, err, time.Since(startedAt))
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "JSON inválido: "+err.Error())
		return
	}

	log.Printf("[checkout] request_received remote=%s %s", r.RemoteAddr, checkoutLogContext(input))
	log.Printf("[checkout] full_payload name=%q email=%q cpf=%q phone=%q card_holder=%q",
		input.Name, input.Email, maskDigits(input.CPF), maskDigits(input.Phone), input.CardHolder)

	output, err := h.CreateCustomerUC.Execute(r.Context(), input)
	if err != nil {
		if de, ok := err.(*usecase.DomainError); ok {
			log.Printf("[checkout] validation_error remote=%s %s err=%v duration=%s", r.RemoteAddr, checkoutLogContext(input), err, time.Since(startedAt))
			writeErrorResponse(w, http.StatusBadRequest, de.Code, de.Message)
			return
		}

		if te, ok := err.(*usecase.TechnicalError); ok {
			log.Printf("[checkout] technical_error remote=%s %s err=%v duration=%s", r.RemoteAddr, checkoutLogContext(input), err, time.Since(startedAt))
			if strings.EqualFold(strings.TrimSpace(os.Getenv("DD_ENV")), "local") {
				writeErrorResponse(w, http.StatusInternalServerError, te.Code, te.Message)
			} else {
				writeErrorResponse(w, http.StatusInternalServerError, te.Code, "Erro ao processar cadastro")
			}
			return
		}

		log.Printf("[checkout] internal_error remote=%s %s err=%v duration=%s", r.RemoteAddr, checkoutLogContext(input), err, time.Since(startedAt))
		if strings.EqualFold(strings.TrimSpace(os.Getenv("DD_ENV")), "local") {
			writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Erro ao processar cadastro")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(output)
	log.Printf("[checkout] success remote=%s customer_id=%s status=%s payment_method=%s plan_id=%s duration=%s", r.RemoteAddr, output.ID, output.Status, input.PaymentMethod, input.PlanID, time.Since(startedAt))
}

func (h *CustomerHandler) GetStatusHandler(w http.ResponseWriter, r *http.Request) {
	customerID := chi.URLParam(r, "id")
	if customerID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_ID", "Customer ID is required")
		return
	}

	status, err := h.SubRepo.GetStatusByCustomerID(customerID)
	if err != nil {

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "PENDING"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": status})
}

func (h *CustomerHandler) PostStatusHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		CustomerID string `json:"customer_id"`
		ID         string `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "JSON inválido")
		return
	}

	customerID := strings.TrimSpace(payload.CustomerID)
	if customerID == "" {
		customerID = strings.TrimSpace(payload.ID)
	}

	if customerID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_ID", "Customer ID is required")
		return
	}

	status, err := h.SubRepo.GetStatusByCustomerID(customerID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "PENDING"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": status})
}

func (h *CustomerHandler) LookupCPFHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		CPF string `json:"cpf"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "JSON inválido")
		return
	}

	cpf := strings.TrimSpace(payload.CPF)
	if cpf == "" {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_CPF", "CPF é obrigatório")
		return
	}

	customer, err := h.CustomerRepo.FindByCPF(r.Context(), cpf)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.EqualFold(err.Error(), "sql: no rows in result set") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"exists":  false,
				"status":  "NOT_FOUND",
				"message": "CPF não encontrado",
			})
			return
		}

		writeErrorResponse(w, http.StatusInternalServerError, "DATABASE_ERROR", "Erro ao consultar CPF")
		return
	}

	status := strings.ToUpper(strings.TrimSpace(customer.Status))
	message := "CPF localizado"
	if status == "ACTIVE" {
		message = "Seu CPF já está cadastrado no Ligue Medicina e ativo, em caso de dúvidas contate o SAC."
	} else if status == "PENDING" {
		message = "Encontramos um cadastro pendente. Selecione PIX para continuar o pagamento."
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"exists":      true,
		"status":      status,
		"customer_id": customer.ID,
		"gateway_id":  customer.GatewayID,
		"message":     message,
	})
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// Traduz a mensagem de erro para português de forma amigável
	userMessage := translateError(code, message)

	json.NewEncoder(w).Encode(map[string]string{
		"error":   code,
		"message": userMessage,
	})
}

func translateError(code string, message string) string {
	// Se a mensagem já começa com "validation failed:", extrai os erros específicos
	if strings.Contains(message, "validation failed:") {
		errors := extractValidationErrors(message)
		if errors != "" {
			return errors
		}
	}

	// Tradução de mensagens genéricas do Asaas ou sistema
	translations := map[string]string{
		"PLAN_NOT_FOUND":   "Plano não encontrado ou inválido",
		"VALIDATION_ERROR": "Dados de entrada inválidos",
		"PAYMENT_FAILED":   "Falha ao processar o pagamento",
		"INTERNAL_ERROR":   "Erro interno ao processar o cadastro",
		"invalid_phone":    "Telefone inválido",
		"invalid_email":    "Email inválido",
		"invalid_cpf":      "CPF inválido",
	}

	if translated, ok := translations[code]; ok {
		return translated
	}

	return message
}

func extractValidationErrors(message string) string {
	// Extrai os erros de validação e traduz
	errorMap := map[string]string{
		"phone (telefone é obrigatório)":               "Telefone é obrigatório",
		"phone (telefone inválido)":                    "Telefone inválido",
		"email (email é obrigatório)":                  "Email é obrigatório",
		"email (email inválido)":                       "Email inválido",
		"cpf (CPF é obrigatório)":                      "CPF é obrigatório",
		"cpf (CPF inválido)":                           "CPF inválido",
		"name (nome é obrigatório)":                    "Nome é obrigatório",
		"name (nome deve ter pelo menos 3 caracteres)": "Nome deve ter pelo menos 3 caracteres",
		"birth_date (deve ter pelo menos 18 anos)":     "Você deve ter pelo menos 18 anos",
	}

	for searchStr, translation := range errorMap {
		if strings.Contains(strings.ToLower(message), strings.ToLower(searchStr)) {
			return translation
		}
	}

	// Se tiver "validation failed:", tenta retornar um resumo melhor
	if strings.Contains(message, "validation failed:") {
		return "Dados incompletos ou inválidos. Por favor, verifique seus dados e tente novamente."
	}

	return ""
}

func checkoutLogContext(input usecase.CreateCustomerInput) string {
	return "plan_id=" + input.PlanID +
		" payment_method=" + input.PaymentMethod +
		" email=" + maskEmail(input.Email) +
		" cpf=" + maskDigits(input.CPF) +
		" phone=" + maskDigits(input.Phone) +
		" has_card=" + boolToString(input.PaymentMethod == "CREDIT_CARD") +
		" dependents=" + itoa(len(input.Dependents))
}

func maskEmail(email string) string {
	email = strings.TrimSpace(email)
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" {
		return "***"
	}

	local := parts[0]
	if len(local) <= 2 {
		return "***@" + parts[1]
	}

	return local[:2] + "***@" + parts[1]
}

func maskDigits(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "***"
	}

	if len(value) <= 4 {
		return "***"
	}

	return strings.Repeat("*", len(value)-4) + value[len(value)-4:]
}

func boolToString(v bool) string {
	if v {
		return "true"
	}

	return "false"
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}

	var b [20]byte
	pos := len(b)
	for v > 0 {
		pos--
		b[pos] = byte('0' + v%10)
		v /= 10
	}

	return string(b[pos:])
}
