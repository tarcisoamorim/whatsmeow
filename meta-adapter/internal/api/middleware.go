package api

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/rs/zerolog/log"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/config"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/errors"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/models"
)

// SetupMiddleware sets up all middleware
func SetupMiddleware(app *fiber.App, cfg *config.Config) {
	// Recovery middleware
	app.Use(recover.New(recover.Config{
		EnableStackTrace: cfg.IsDevelopment(),
	}))

	// Logger middleware
	if cfg.IsDevelopment() {
		app.Use(logger.New(logger.Config{
			Format:     "[${time}] ${status} - ${method} ${path} (${latency})\n",
			TimeFormat: "15:04:05",
			TimeZone:   "Local",
		}))
	}

	// CORS middleware
	if cfg.CORS.Enabled {
		allowedOrigins := "*"
		if len(cfg.CORS.AllowedOrigins) > 0 {
			allowedOrigins = strings.Join(cfg.CORS.AllowedOrigins, ",")
		}

		app.Use(cors.New(cors.Config{
			AllowOrigins: allowedOrigins,
			AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
			AllowHeaders: "Origin,Content-Type,Accept,Authorization",
		}))
	}

	// Custom request logging
	app.Use(func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)

		log.Info().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", c.Response().StatusCode()).
			Dur("duration", duration).
			Str("ip", c.IP()).
			Msg("HTTP request")

		return err
	})
}

// AuthMiddleware validates API key
func AuthMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip auth if no API keys configured
		if cfg.Security.APIKey == "" && len(cfg.Security.APIKeys) == 0 {
			return c.Next()
		}

		// Get API key from header
		apiKey := c.Get("Authorization")
		if apiKey == "" {
			apiKey = c.Get("X-API-Key")
		}

		// Remove "Bearer " prefix if present
		apiKey = strings.TrimPrefix(apiKey, "Bearer ")

		// Validate API key
		if !cfg.ValidAPIKey(apiKey) {
			return errorResponse(c, errors.ErrUnauthorized)
		}

		return c.Next()
	}
}

// RateLimitMiddleware implements simple rate limiting
// Note: This is a basic implementation. For production, consider using Redis-based rate limiting
func RateLimitMiddleware(cfg *config.Config) fiber.Handler {
	if !cfg.RateLimit.Enabled {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}

	// Simple in-memory rate limiter (not suitable for multi-instance deployments)
	// TODO: Implement Redis-based rate limiting for production
	return func(c *fiber.Ctx) error {
		// For now, just pass through
		// In production, implement proper rate limiting here
		return c.Next()
	}
}

// Error response helper
func errorResponse(c *fiber.Ctx, err *errors.AdapterError) error {
	return c.Status(err.HTTPStatus).JSON(err.ToMetaError())
}

// Success response helper
func successResponse(c *fiber.Ctx, data interface{}) error {
	return c.JSON(data)
}

// Generic error handler
func HandleError(c *fiber.Ctx, err error) error {
	if adapterErr, ok := err.(*errors.AdapterError); ok {
		return errorResponse(c, adapterErr)
	}

	log.Error().Err(err).Msg("Unexpected error")
	return errorResponse(c, errors.ErrInternalServer)
}
