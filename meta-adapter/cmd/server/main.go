package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/zap"

	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/api"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/auth"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/config"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/middleware"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/ratelimit"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/repository"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/whatsapp"
	pkglogger "github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/logger"
)

const banner = `
╦ ╦┬ ┬┌─┐┌┬┐┌─┐╔═╗┌─┐┌─┐  ╔╦╗┌─┐┌┬┐┌─┐  ╔═╗╔═╗╦  ╔═╗┌┬┐┌─┐┌─┐┌┬┐┌─┐┬─┐
║║║├─┤├─┤ │ └─┐╠═╣├─┘├─┘  ║║║├┤  │ ├─┤  ╠═╣╠═╝║  ╠═╣ ││├─┤├─┘ │ ├┤ ├┬┘
╚╩╝┴ ┴┴ ┴ ┴ └─┘╩ ╩┴  ┴    ╩ ╩└─┘ ┴ ┴ ┴  ╩ ╩╩  ╩  ╩ ╩─┴┘┴ ┴┴   ┴ └─┘┴└─
                    🚀 Production-Ready Multi-Tenant API
`

func main() {
	fmt.Println(banner)

	// Initialize logger
	if err := pkglogger.Init(); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer pkglogger.Get().Sync()

	zlog := pkglogger.Get()
	zlog.Info("Starting WhatsApp Meta API Adapter")

	// Load config
	cfg, err := config.Load()
	if err != nil {
		zlog.Fatal("Failed to load config", zap.Error(err))
	}

	// Connect to database
	var db *repository.Database
	if cfg.Database.URL != "" {
		db, err = repository.NewDatabase(cfg.Database.URL)
		if err != nil {
			zlog.Fatal("Failed to connect to database", zap.Error(err))
		}
		defer db.Close()
		zlog.Info("Database connected successfully")
	} else {
		zlog.Warn("DATABASE_URL not set, running without database")
	}

	// Initialize repositories
	instanceRepo := repository.NewInstanceRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	tenantRepo := repository.NewTenantRepository(db)

	// Initialize WhatsApp manager
	waManager, err := whatsapp.NewManager(cfg.Database.URL, instanceRepo, messageRepo)
	if err != nil {
		zlog.Fatal("Failed to initialize WhatsApp manager", zap.Error(err))
	}
	defer waManager.Shutdown()
	zlog.Info("WhatsApp manager initialized")

	// Initialize rate limiter
	var rateLimiter *ratelimit.Limiter
	if cfg.Redis.URL != "" {
		rateLimiter, err = ratelimit.NewLimiter(cfg.Redis.URL)
		if err != nil {
			zlog.Fatal("Failed to initialize rate limiter", zap.Error(err))
		}
		defer rateLimiter.Close()
		zlog.Info("Rate limiter initialized")
	} else {
		zlog.Warn("REDIS_URL not set, running without rate limiting")
	}

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(cfg.JWT.Secret)
	zlog.Info("JWT manager initialized")

	// Initialize handlers
	instanceHandler := api.NewInstanceHandler(db, waManager)
	messageHandler := api.NewMessageHandler(db, waManager)
	presenceHandler := api.NewPresenceHandler(db, waManager)
	actionsHandler := api.NewActionsHandler(db, waManager)
	chatsHandler := api.NewChatsHandler(db, waManager)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: errorHandler,
		AppName:      "WhatsApp Meta API Adapter v1.0.0",
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "whatsapp-meta-api-adapter",
			"version": "1.0.0",
		})
	})

	// API routes
	v1 := app.Group("/v1")

	// OAuth endpoints (no auth required)
	v1.Get("/oauth/authorize", func(c *fiber.Ctx) error {
		return c.SendString("OAuth2 authorization endpoint - TODO: implement full flow")
	})

	v1.Post("/oauth/token", func(c *fiber.Ctx) error {
		// Demo: Generate a test token
		token, _ := jwtManager.Generate(
			"demo-tenant-id",
			"demo-client-id",
			[]string{"messages.send", "messages.read", "instances.manage"},
			time.Hour,
		)

		return c.JSON(fiber.Map{
			"access_token": token,
			"token_type":   "Bearer",
			"expires_in":   3600,
			"scope":        "messages.send messages.read instances.manage",
		})
	})

	// Protected routes (require authentication)
	authMiddleware := middleware.AuthMiddleware(jwtManager)
	tenantMiddleware := middleware.TenantMiddleware(tenantRepo)

	// Build middleware chain
	var protectedMiddlewares []fiber.Handler
	protectedMiddlewares = append(protectedMiddlewares, authMiddleware)
	protectedMiddlewares = append(protectedMiddlewares, tenantMiddleware)
	if rateLimiter != nil {
		protectedMiddlewares = append(protectedMiddlewares, middleware.RateLimitMiddleware(rateLimiter))
	}

	// Instances endpoints
	instances := v1.Group("/instances", protectedMiddlewares...)
	instances.Get("/", instanceHandler.List)
	instances.Post("/", middleware.RequireScope("instances.manage"), instanceHandler.Create)
	instances.Get("/:phone_number_id", instanceHandler.Get)
	instances.Get("/:phone_number_id/qrcode", instanceHandler.GetQRCode)

	// Messages endpoints
	messages := v1.Group("/:phone_number_id/messages", protectedMiddlewares...)
	messages.Post("/", middleware.RequireScope("messages.send"), messageHandler.Send)
	messages.Get("/", middleware.RequireScope("messages.read"), messageHandler.List)

	// Message actions endpoints
	messages.Post("/:message_id/read", middleware.RequireScope("messages.send"), actionsHandler.MarkMessageRead)
	messages.Delete("/:message_id", middleware.RequireScope("messages.send"), actionsHandler.DeleteMessage)
	messages.Post("/:message_id/react", middleware.RequireScope("messages.send"), actionsHandler.ReactToMessage)
	messages.Patch("/:message_id", middleware.RequireScope("messages.send"), actionsHandler.EditMessage)

	// Presence endpoints
	presence := v1.Group("/:phone_number_id", protectedMiddlewares...)
	presence.Patch("/presence", middleware.RequireScope("messages.send"), presenceHandler.UpdatePresence)
	presence.Post("/typing", middleware.RequireScope("messages.send"), presenceHandler.SendTyping)

	// Chat settings endpoints (ephemeral messages)
	chats := v1.Group("/:phone_number_id/chats", protectedMiddlewares...)
	chats.Patch("/:chat_jid/disappearing", middleware.RequireScope("messages.send"), chatsHandler.SetChatDisappearingTimer)

	settings := v1.Group("/:phone_number_id/settings", protectedMiddlewares...)
	settings.Patch("/disappearing", middleware.RequireScope("messages.send"), chatsHandler.SetDefaultDisappearingTimer)

	// Start server
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("\n🚀 Server starting on %s\n\n", addr)
	
	if err := app.Listen(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	
	return c.Status(code).JSON(fiber.Map{
		"error": fiber.Map{
			"message": err.Error(),
			"type":    "InternalError",
			"code":    code,
		},
	})
}
