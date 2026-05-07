package middleware

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
)

var (
	metricsClient     datadogClient = noopDatadogClient{}
	metricsClientOnce sync.Once
	activeConnections int64
)

type datadogClient interface {
	Count(name string, value int64, tags []string, rate float64) error
	Gauge(name string, value float64, tags []string, rate float64) error
	Timing(name string, value time.Duration, tags []string, rate float64) error
	Close() error
}

type noopDatadogClient struct{}

func (noopDatadogClient) Count(string, int64, []string, float64) error   { return nil }
func (noopDatadogClient) Gauge(string, float64, []string, float64) error { return nil }
func (noopDatadogClient) Timing(string, time.Duration, []string, float64) error {
	return nil
}
func (noopDatadogClient) Close() error { return nil }

func getMetricsClient() datadogClient {
	metricsClientOnce.Do(func() {
		host := strings.TrimSpace(os.Getenv("DD_AGENT_HOST"))
		if host == "" {
			host = "localhost"
		}

		port := strings.TrimSpace(os.Getenv("DD_DOGSTATSD_PORT"))
		if port == "" {
			port = "8125"
		}

		client, err := statsd.New(
			host+":"+port,
			statsd.WithNamespace("ligue_payments."),
			statsd.WithTags(baseTags()),
		)
		if err == nil {
			metricsClient = client
		}
	})

	return metricsClient
}

func baseTags() []string {
	return []string{
		"service:" + envOrDefault("DD_SERVICE", "ligue-payments"),
		"env:" + envOrDefault("DD_ENV", "local"),
	}
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

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
		client := getMetricsClient()

		currentConnections := atomic.AddInt64(&activeConnections, 1)
		_ = client.Gauge("http.active_connections", float64(currentConnections), metricTags("method:"+r.Method, "path:"+normalizePath(r.URL.Path)), 1)
		defer func() {
			currentConnections := atomic.AddInt64(&activeConnections, -1)
			_ = client.Gauge("http.active_connections", float64(currentConnections), metricTags("method:"+r.Method, "path:"+normalizePath(r.URL.Path)), 1)
		}()

		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		status := strconv.Itoa(rw.statusCode)
		tags := metricTags(
			"method:"+r.Method,
			"path:"+normalizePath(r.URL.Path),
			"status:"+status,
		)

		_ = client.Count("http.requests", 1, tags, 1)
		_ = client.Timing("http.request_duration", duration, tags, 1)
	})
}

func RecordPayment(method, status string) {
	_ = getMetricsClient().Count("payments.received", 1, metricTags("method:"+method, "status:"+status), 1)
}

func RecordSubscriptionActivation() {
	_ = getMetricsClient().Count("subscriptions.activated", 1, metricTags(), 1)
}

func RecordIntegrationError(service string) {
	_ = getMetricsClient().Count("integration.errors", 1, metricTags("service:"+service), 1)
}

func RecordQueuePublished(queueName, routingKey string) {
	_ = getMetricsClient().Count("queue.messages_published", 1, metricTags("queue:"+queueName, "routing_key:"+routingKey), 1)
}

func RecordQueueConsumed(queueName, provider, status string) {
	tags := metricTags("queue:"+queueName, "status:"+status)
	if provider != "" {
		tags = append(tags, "provider:"+provider)
	}

	_ = getMetricsClient().Count("queue.messages_consumed", 1, tags, 1)
}

func RecordQueueDepth(queueName string, depth int) {
	_ = getMetricsClient().Gauge("queue.size", float64(depth), metricTags("queue:"+queueName), 1)
}

func RecordQueueMessageArrivalLatency(queueName, provider string, duration time.Duration) {
	tags := metricTags("queue:" + queueName)
	if provider != "" {
		tags = append(tags, "provider:"+provider)
	}

	_ = getMetricsClient().Timing("queue.message_arrival_latency", duration, tags, 1)
}

func RecordQueueProcessingDuration(queueName, provider string, duration time.Duration) {
	tags := metricTags("queue:" + queueName)
	if provider != "" {
		tags = append(tags, "provider:"+provider)
	}

	_ = getMetricsClient().Timing("queue.processing_duration", duration, tags, 1)
}

func metricTags(tags ...string) []string {
	return tags
}

func normalizePath(path string) string {
	if path == "" {
		return "unknown"
	}

	trimmed := strings.TrimSuffix(path, "/")
	if trimmed == "" {
		return "/"
	}

	return trimmed
}
