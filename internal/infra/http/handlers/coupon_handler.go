package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/xavierca1/ligue-payments/internal/usecase"
)

type CouponHandler struct{}

func NewCouponHandler() *CouponHandler {
	return &CouponHandler{}
}

func (h *CouponHandler) Validate(w http.ResponseWriter, r *http.Request) {
	var input struct {
		CouponCode string `json:"coupon_code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "JSON inválido")
		return
	}

	code := strings.TrimSpace(input.CouponCode)
	if code == "" {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_FIELDS", "coupon_code é obrigatório")
		return
	}

	details, err := usecase.ValidateCoupon(r.Context(), code)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "COUPON_INVALID", "cupom inválido ou inativo")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":            true,
		"coupon_code":      details.Code,
		"seller_name":      details.SellerName,
		"discount_percent": details.DiscountPercent,
	})
}
