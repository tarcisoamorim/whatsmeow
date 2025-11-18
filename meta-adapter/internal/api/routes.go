package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/config"
)

// SetupRoutes configures all API routes
func SetupRoutes(app *fiber.App, handler *Handler, cfg *config.Config) {
	// Health check (no auth required)
	app.Get("/health", handler.Health)
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"name":    "WhatsApp Meta API Adapter",
			"version": "1.0.0",
			"status":  "running",
		})
	})

	// API v1 routes
	api := app.Group(cfg.Server.BasePath)

	// Apply authentication to all API routes
	api.Use(AuthMiddleware(cfg))
	api.Use(RateLimitMiddleware(cfg))

	// Session management
	session := api.Group("/session")
	session.Get("/qr", handler.GetQRCode)
	session.Get("/status", handler.GetSessionStatus)
	session.Post("/logout", handler.Logout)

	// Messages (Meta API compatible)
	// POST /v1/messages - send message
	api.Post("/messages", handler.SendMessage)

	// Media (Meta API compatible)
	// GET /v1/:media_id - get media
	// POST /v1/media - upload media
	api.Get("/:media_id", handler.GetMedia)
	api.Post("/media", handler.UploadMedia)

	// Webhook verification (Meta API requirement)
	app.Get("/webhook", handler.WebhookVerification)
}
