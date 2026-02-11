package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	activeConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_active_connections",
			Help: "Number of active HTTP connections",
		},
	)

	paymentsReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payments_received_total",
			Help: "Total number of payments received",
		},
		[]string{"method", "status"},
	)

	subscriptionsActivated = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "subscriptions_activated_total",
			Help: "Total number of subscriptions activated",
		},
	)

	integrationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "integration_errors_total",
			Help: "Total number of integration errors",
		},
		[]string{"service"},
	)
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		activeConnections.Inc()
		defer activeConnections.Dec()

		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(rw.statusCode)

		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	})
}

func RecordPayment(method, status string) {
	paymentsReceived.WithLabelValues(method, status).Inc()
}

func RecordSubscriptionActivation() {
	subscriptionsActivated.Inc()
}

func RecordIntegrationError(service string) {
	integrationErrors.WithLabelValues(service).Inc()
}
