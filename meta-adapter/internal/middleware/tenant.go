package middleware

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/repository"
	pkglogger "github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/logger"
)

// TenantMiddleware enriches context with tenant information
func TenantMiddleware(tenantRepo *repository.TenantRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		logger := pkglogger.Get()

		// Get tenant ID from context (set by auth middleware)
		tenantID, ok := c.Locals("tenant_id").(string)
		if !ok {
			// No tenant ID, skip enrichment
			return c.Next()
		}

		// Fetch tenant from database
		ctx := context.Background()
		tenant, err := tenantRepo.FindByID(ctx, tenantID)
		if err != nil {
			logger.Error("Failed to fetch tenant",
				zap.Error(err),
				zap.String("tenant_id", tenantID),
			)
			// Don't fail the request, just proceed without tier info
			c.Locals("tier", "free") // Default to free
			return c.Next()
		}

		// Add tier to context
		c.Locals("tier", tenant.Tier)
		c.Locals("tenant", tenant)

		return c.Next()
	}
}
