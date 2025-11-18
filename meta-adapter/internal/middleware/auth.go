package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/auth"
)

func AuthMiddleware(jwtManager *auth.JWTManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(401).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "Missing Authorization header",
					"type":    "AuthenticationError",
					"code":    401,
				},
			})
		}

		// Check Bearer prefix
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(401).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "Invalid Authorization header format",
					"type":    "AuthenticationError",
					"code":    401,
				},
			})
		}

		token := parts[1]

		// Verify JWT
		claims, err := jwtManager.Verify(token)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "Invalid or expired token",
					"type":    "AuthenticationError",
					"code":    401,
				},
			})
		}

		// Store claims in context
		c.Locals("tenant_id", claims.TenantID)
		c.Locals("client_id", claims.ClientID)
		c.Locals("scopes", claims.Scopes)
		c.Locals("claims", claims)

		return c.Next()
	}
}

func RequireScope(scope string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals("claims").(*auth.Claims)
		if !ok {
			return c.Status(401).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "Unauthorized",
					"type":    "AuthenticationError",
					"code":    401,
				},
			})
		}

		if !claims.HasScope(scope) {
			return c.Status(403).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "Insufficient permissions",
					"type":    "PermissionError",
					"code":    403,
				},
			})
		}

		return c.Next()
	}
}
