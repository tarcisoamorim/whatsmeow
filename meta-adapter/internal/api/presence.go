package api

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/repository"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/whatsapp"
	pkglogger "github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/logger"
)

type PresenceHandler struct {
	instanceRepo *repository.InstanceRepository
	waManager    *whatsapp.Manager
}

func NewPresenceHandler(db *repository.Database, waManager *whatsapp.Manager) *PresenceHandler {
	return &PresenceHandler{
		instanceRepo: repository.NewInstanceRepository(db),
		waManager:    waManager,
	}
}

// UpdatePresence updates the presence status (online/offline)
// PATCH /v1/{phone_number_id}/presence
func (h *PresenceHandler) UpdatePresence(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	phoneNumberID := c.Params("phone_number_id")
	logger := pkglogger.Get()

	// Parse request
	var req struct {
		Status string `json:"status"` // "online" or "offline"
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

	// Validate status
	if req.Status != "online" && req.Status != "offline" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Status must be 'online' or 'offline'",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// Get instance
	instance, err := h.instanceRepo.FindByPhoneNumberID(ctx, tenantID, phoneNumberID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Instance not found",
				"type":    "NotFoundError",
				"code":    404,
			},
		})
	}

	// Check if instance is connected
	if instance.Status != "connected" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Instance is not connected",
				"type":    "InstanceNotConnectedError",
				"code":    400,
			},
		})
	}

	// Set presence
	available := req.Status == "online"
	err = h.waManager.SetPresence(ctx, tenantID, instance.ID, available)
	if err != nil {
		logger.Error("Failed to set presence", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Failed to set presence: " + err.Error(),
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	logger.Info("Presence updated",
		zap.String("phone_number_id", phoneNumberID),
		zap.String("status", req.Status),
	)

	return c.JSON(fiber.Map{
		"success": true,
		"status":  req.Status,
	})
}

// SendTyping sends typing/recording indicator
// POST /v1/{phone_number_id}/typing
func (h *PresenceHandler) SendTyping(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	phoneNumberID := c.Params("phone_number_id")
	logger := pkglogger.Get()

	// Parse request
	var req struct {
		To    string `json:"to"`                // Recipient phone number
		State string `json:"state"`             // "composing" or "paused"
		Media string `json:"media,omitempty"`   // "text" or "audio" (optional, default: text)
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

	// Validate required fields
	if req.To == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Missing 'to' field",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	if req.State != "composing" && req.State != "paused" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "State must be 'composing' or 'paused'",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// Default media type
	if req.Media == "" {
		req.Media = "text"
	}

	// Get instance
	instance, err := h.instanceRepo.FindByPhoneNumberID(ctx, tenantID, phoneNumberID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Instance not found",
				"type":    "NotFoundError",
				"code":    404,
			},
		})
	}

	// Check if instance is connected
	if instance.Status != "connected" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Instance is not connected",
				"type":    "InstanceNotConnectedError",
				"code":    400,
			},
		})
	}

	// Send chat presence
	err = h.waManager.SendChatPresence(ctx, tenantID, instance.ID, req.To, req.State, req.Media)
	if err != nil {
		logger.Error("Failed to send typing indicator", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Failed to send typing indicator: " + err.Error(),
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	logger.Debug("Typing indicator sent",
		zap.String("to", req.To),
		zap.String("state", req.State),
	)

	return c.JSON(fiber.Map{
		"success": true,
		"state":   req.State,
	})
}
