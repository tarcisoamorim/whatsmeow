package middleware

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP request counter by method, path, and status code
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// HTTP request duration histogram
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets, // 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
		},
		[]string{"method", "path", "status"},
	)

	// WhatsApp messages sent counter
	whatsappMessagesSent = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "whatsapp_messages_sent_total",
			Help: "Total number of WhatsApp messages sent",
		},
		[]string{"tenant_id", "type", "status"},
	)

	// Active instances gauge
	activeInstances = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "whatsapp_active_instances",
			Help: "Number of active WhatsApp instances",
		},
	)

	// Database connection pool metrics
	dbConnectionsOpen = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_open",
			Help: "Number of open database connections",
		},
	)

	dbConnectionsInUse = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_in_use",
			Help: "Number of database connections in use",
		},
	)

	dbConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_idle",
			Help: "Number of idle database connections",
		},
	)
)

// MetricsMiddleware records HTTP metrics for all requests
func MetricsMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Process request
		err := c.Next()

		// Record metrics
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Response().StatusCode())
		method := c.Method()
		path := c.Route().Path

		// If path is empty, use the actual path
		if path == "" {
			path = c.Path()
		}

		// Increment request counter
		httpRequestsTotal.WithLabelValues(method, path, status).Inc()

		// Record request duration
		httpRequestDuration.WithLabelValues(method, path, status).Observe(duration)

		return err
	}
}

// RecordMessageSent records a sent WhatsApp message
func RecordMessageSent(tenantID, messageType, status string) {
	whatsappMessagesSent.WithLabelValues(tenantID, messageType, status).Inc()
}

// SetActiveInstances updates the active instances gauge
func SetActiveInstances(count int) {
	activeInstances.Set(float64(count))
}

// UpdateDBConnectionMetrics updates database connection pool metrics
func UpdateDBConnectionMetrics(open, inUse, idle int) {
	dbConnectionsOpen.Set(float64(open))
	dbConnectionsInUse.Set(float64(inUse))
	dbConnectionsIdle.Set(float64(idle))
}
