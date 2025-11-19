package api

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/repository"
	pkglogger "github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/logger"
)

type HealthHandler struct {
	db          *repository.Database
	redisClient *redis.Client
}

func NewHealthHandler(db *repository.Database, redisClient *redis.Client) *HealthHandler {
	return &HealthHandler{
		db:          db,
		redisClient: redisClient,
	}
}

type ComponentStatus struct {
	Status  string                 `json:"status"`  // "healthy", "unhealthy", "disabled"
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

type HealthResponse struct {
	Status     string                      `json:"status"` // "healthy", "degraded", "unhealthy"
	Service    string                      `json:"service"`
	Version    string                      `json:"version"`
	Timestamp  string                      `json:"timestamp"`
	Components map[string]ComponentStatus `json:"components"`
}

// Health performs comprehensive health checks on all critical dependencies
// Returns HTTP 200 if all critical components are healthy
// Returns HTTP 503 if any critical component is down
func (h *HealthHandler) Health(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := pkglogger.Get()
	response := HealthResponse{
		Status:     "healthy",
		Service:    "whatsapp-meta-api-adapter",
		Version:    "1.0.0",
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Components: make(map[string]ComponentStatus),
	}

	allHealthy := true
	anyCriticalDown := false

	// Check Database (CRITICAL)
	dbStatus := h.checkDatabase(ctx)
	response.Components["database"] = dbStatus
	if dbStatus.Status == "unhealthy" {
		allHealthy = false
		anyCriticalDown = true
		logger.Error("Health check: Database is unhealthy", zap.String("message", dbStatus.Message))
	}

	// Check Redis (CRITICAL for rate limiting)
	if h.redisClient != nil {
		redisStatus := h.checkRedis(ctx)
		response.Components["redis"] = redisStatus
		if redisStatus.Status == "unhealthy" {
			allHealthy = false
			anyCriticalDown = true
			logger.Error("Health check: Redis is unhealthy", zap.String("message", redisStatus.Message))
		}
	} else {
		response.Components["redis"] = ComponentStatus{
			Status:  "disabled",
			Message: "Redis not configured",
		}
	}

	// Determine overall status
	if anyCriticalDown {
		response.Status = "unhealthy"
		return c.Status(fiber.StatusServiceUnavailable).JSON(response)
	}

	if !allHealthy {
		response.Status = "degraded"
		return c.Status(fiber.StatusOK).JSON(response)
	}

	// All healthy
	return c.Status(fiber.StatusOK).JSON(response)
}

// checkDatabase verifies database connectivity and basic query execution
func (h *HealthHandler) checkDatabase(ctx context.Context) ComponentStatus {
	if h.db == nil || h.db.DB == nil {
		return ComponentStatus{
			Status:  "unhealthy",
			Message: "Database connection not initialized",
		}
	}

	start := time.Now()

	// Ping database
	if err := h.db.DB.PingContext(ctx); err != nil {
		return ComponentStatus{
			Status:  "unhealthy",
			Message: "Failed to ping database: " + err.Error(),
		}
	}

	latency := time.Since(start)

	// Check connection pool stats
	stats := h.db.DB.Stats()

	return ComponentStatus{
		Status:  "healthy",
		Message: "Database connection is healthy",
		Details: map[string]interface{}{
			"latency_ms":      latency.Milliseconds(),
			"open_conns":      stats.OpenConnections,
			"in_use":          stats.InUse,
			"idle":            stats.Idle,
			"max_open_conns":  stats.MaxOpenConnections,
			"wait_count":      stats.WaitCount,
			"wait_duration_ms": stats.WaitDuration.Milliseconds(),
		},
	}
}

// checkRedis verifies Redis connectivity and basic operations
func (h *HealthHandler) checkRedis(ctx context.Context) ComponentStatus {
	if h.redisClient == nil {
		return ComponentStatus{
			Status:  "disabled",
			Message: "Redis not configured",
		}
	}

	start := time.Now()

	// Ping Redis
	if err := h.redisClient.Ping(ctx).Err(); err != nil {
		return ComponentStatus{
			Status:  "unhealthy",
			Message: "Failed to ping Redis: " + err.Error(),
		}
	}

	latency := time.Since(start)

	// Get Redis info
	poolStats := h.redisClient.PoolStats()

	return ComponentStatus{
		Status:  "healthy",
		Message: "Redis connection is healthy",
		Details: map[string]interface{}{
			"latency_ms": latency.Milliseconds(),
			"hits":       poolStats.Hits,
			"misses":     poolStats.Misses,
			"timeouts":   poolStats.Timeouts,
			"total_conns": poolStats.TotalConns,
			"idle_conns":  poolStats.IdleConns,
			"stale_conns": poolStats.StaleConns,
		},
	}
}
