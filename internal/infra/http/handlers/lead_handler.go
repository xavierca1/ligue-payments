package handlers

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/xavierca1/ligue-payments/internal/entity"
)


type LeadHandler struct {
	leadRepo    entity.LeadRepositoryInterface
	rateLimiter *RateLimiter
}


func NewLeadHandler(leadRepo entity.LeadRepositoryInterface) *LeadHandler {
	return &LeadHandler{
		leadRepo:    leadRepo,
		rateLimiter: NewRateLimiter(10, time.Minute), // 10 req/min por IP
	}
}


type CaptureLeadRequest struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
	Phone string `json:"phone,omitempty"`
}


type CaptureLeadResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}


func (h *LeadHandler) CaptureLead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()


	clientIP := getClientIP(r)
	if !h.rateLimiter.Allow(clientIP) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(CaptureLeadResponse{
			Success: false,
			Message: "Too many requests. Please try again later.",
		})
		return
	}


	var req CaptureLeadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(CaptureLeadResponse{
			Success: false,
			Message: "Invalid JSON",
		})
		return
	}


	if req.Email == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(CaptureLeadResponse{
			Success: false,
			Message: "Email is required",
		})
		return
	}


	lead := &entity.Lead{
		Email: req.Email,
		Name:  req.Name,
		Phone: req.Phone,
	}

	if err := h.leadRepo.Upsert(ctx, lead); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(CaptureLeadResponse{
			Success: false,
			Message: "Failed to capture lead",
		})
		return
	}


	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(CaptureLeadResponse{
		Success: true,
	})
}


func getClientIP(r *http.Request) string {

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	return r.RemoteAddr
}


type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    int
	window   time.Duration
}

type visitor struct {
	count     int
	lastReset time.Time
}


func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		window:   window,
	}

	go rl.cleanup()
	return rl
}


func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	now := time.Now()

	if !exists {
		rl.visitors[ip] = &visitor{count: 1, lastReset: now}
		return true
	}


	if now.Sub(v.lastReset) > rl.window {
		v.count = 1
		v.lastReset = now
		return true
	}


	v.count++
	return v.count <= rl.limit
}


func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, v := range rl.visitors {
			if now.Sub(v.lastReset) > rl.window*2 {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}
