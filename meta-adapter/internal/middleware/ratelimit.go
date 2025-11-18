package middleware

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/ratelimit"
	pkglogger "github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/logger"
)

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(limiter *ratelimit.Limiter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		logger := pkglogger.Get()

		// Get tenant ID and tier from context (set by auth middleware)
		tenantID, ok := c.Locals("tenant_id").(string)
		if !ok {
			// No tenant ID means unauthenticated request, skip rate limiting
			return c.Next()
		}

		// Get tier from context (set by auth middleware)
		tier, ok := c.Locals("tier").(string)
		if !ok {
			tier = "free" // Default to free tier
		}

		// Check rate limit
		allowed, info, err := limiter.Allow(c.Context(), tenantID, tier)
		if err != nil {
			logger.Error("Rate limit check failed",
				zap.Error(err),
				zap.String("tenant_id", tenantID),
			)
			// On error, allow the request to proceed (fail open)
			return c.Next()
		}

		// Set rate limit headers
		c.Set("X-RateLimit-Limit", strconv.Itoa(info.Limit))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(info.Remaining))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(info.Reset, 10))
		c.Set("X-RateLimit-Window", info.Window)

		if !allowed {
			logger.Warn("Rate limit exceeded",
				zap.String("tenant_id", tenantID),
				zap.String("tier", tier),
				zap.String("window", info.Window),
			)

			return c.Status(429).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "Rate limit exceeded. Please try again later.",
					"type":    "RateLimitError",
					"code":    429,
					"details": fiber.Map{
						"limit":     info.Limit,
						"remaining": info.Remaining,
						"reset":     info.Reset,
						"window":    info.Window,
					},
				},
			})
		}

		return c.Next()
	}
}
