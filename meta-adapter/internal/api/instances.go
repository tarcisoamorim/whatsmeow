package api

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/models"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/repository"
	pkglogger "github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/logger"
)

type InstanceHandler struct {
	instanceRepo *repository.InstanceRepository
	tenantRepo   *repository.TenantRepository
}

func NewInstanceHandler(db *repository.Database) *InstanceHandler {
	return &InstanceHandler{
		instanceRepo: repository.NewInstanceRepository(db),
		tenantRepo:   repository.NewTenantRepository(db),
	}
}

func (h *InstanceHandler) List(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	logger := pkglogger.Get()

	instances, err := h.instanceRepo.FindByTenantID(ctx, tenantID)
	if err != nil {
		logger.Error("Failed to list instances", zap.Error(err), zap.String("tenant_id", tenantID))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Failed to retrieve instances",
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	// Convert to response format
	data := make([]fiber.Map, len(instances))
	for i, inst := range instances {
		data[i] = fiber.Map{
			"id":              inst.ID,
			"phone_number_id": inst.PhoneNumberID,
			"display_name":    inst.DisplayName,
			"status":          inst.Status,
			"created_at":      inst.CreatedAt,
			"updated_at":      inst.UpdatedAt,
		}
	}

	return c.JSON(fiber.Map{
		"data": data,
		"paging": fiber.Map{
			"next":     nil,
			"previous": nil,
		},
	})
}

func (h *InstanceHandler) Create(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	logger := pkglogger.Get()

	// Parse request body
	var req struct {
		DisplayName   string   `json:"display_name"`
		WebhookURL    string   `json:"webhook_url"`
		WebhookEvents []string `json:"webhook_events"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Invalid request body",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// Get tenant to check limits
	tenant, err := h.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "Tenant not found",
					"type":    "NotFoundError",
					"code":    404,
				},
			})
		}
		logger.Error("Failed to get tenant", zap.Error(err), zap.String("tenant_id", tenantID))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Internal server error",
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	// Check instance limit
	instanceCount, err := h.instanceRepo.CountByTenantID(ctx, tenantID)
	if err != nil {
		logger.Error("Failed to count instances", zap.Error(err), zap.String("tenant_id", tenantID))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Internal server error",
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	if instanceCount >= tenant.MaxInstances {
		return c.Status(403).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Instance limit reached for your tier",
				"type":    "QuotaExceededError",
				"code":    403,
			},
		})
	}

	// Create instance
	instance := models.NewInstance(tenantID)
	if req.DisplayName != "" {
		instance.DisplayName = &req.DisplayName
	}
	if req.WebhookURL != "" {
		instance.WebhookURL = &req.WebhookURL
	}
	instance.WebhookEvents = req.WebhookEvents

	// Save to database
	if err := h.instanceRepo.Create(ctx, instance); err != nil {
		logger.Error("Failed to create instance", zap.Error(err), zap.String("tenant_id", tenantID))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Failed to create instance",
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	logger.Info("Instance created", zap.String("instance_id", instance.ID), zap.String("tenant_id", tenantID))

	return c.Status(201).JSON(fiber.Map{
		"id":              instance.ID,
		"phone_number_id": instance.PhoneNumberID,
		"display_name":    instance.DisplayName,
		"status":          instance.Status,
		"webhook_url":     instance.WebhookURL,
		"webhook_events":  instance.WebhookEvents,
		"created_at":      instance.CreatedAt,
	})
}

func (h *InstanceHandler) Get(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	phoneNumberID := c.Params("phone_number_id")
	logger := pkglogger.Get()

	instance, err := h.instanceRepo.FindByPhoneNumberID(ctx, tenantID, phoneNumberID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "Instance not found",
					"type":    "NotFoundError",
					"code":    404,
				},
			})
		}
		logger.Error("Failed to get instance", zap.Error(err), zap.String("tenant_id", tenantID), zap.String("phone_number_id", phoneNumberID))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Internal server error",
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	return c.JSON(fiber.Map{
		"id":              instance.ID,
		"phone_number_id": instance.PhoneNumberID,
		"display_name":    instance.DisplayName,
		"status":          instance.Status,
		"webhook_url":     instance.WebhookURL,
		"webhook_events":  instance.WebhookEvents,
		"connected_at":    instance.ConnectedAt,
		"disconnected_at": instance.DisconnectedAt,
		"last_seen_at":    instance.LastSeenAt,
		"created_at":      instance.CreatedAt,
		"updated_at":      instance.UpdatedAt,
	})
}

func (h *InstanceHandler) GetQRCode(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	phoneNumberID := c.Params("phone_number_id")
	logger := pkglogger.Get()

	// Get instance to verify it exists and belongs to tenant
	instance, err := h.instanceRepo.FindByPhoneNumberID(ctx, tenantID, phoneNumberID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "Instance not found",
					"type":    "NotFoundError",
					"code":    404,
				},
			})
		}
		logger.Error("Failed to get instance", zap.Error(err), zap.String("tenant_id", tenantID), zap.String("phone_number_id", phoneNumberID))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Internal server error",
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	// Check if instance is already connected
	if instance.Status == "connected" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Instance is already connected",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// TODO: Generate actual QR code via whatsmeow
	// For now, return existing QR code from database or placeholder
	qrCode := "data:image/png;base64,iVBORw0KGgo..."
	expiresAt := "2025-11-18T21:30:00Z"

	if instance.QRCode != nil {
		qrCode = *instance.QRCode
	}
	if instance.QRExpiresAt != nil {
		expiresAt = instance.QRExpiresAt.Format("2006-01-02T15:04:05Z07:00")
	}

	return c.JSON(fiber.Map{
		"qr_code":    qrCode,
		"expires_at": expiresAt,
	})
}
