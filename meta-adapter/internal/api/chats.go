package api

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/repository"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/whatsapp"
	pkglogger "github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/logger"
)

type ChatsHandler struct {
	instanceRepo *repository.InstanceRepository
	waManager    *whatsapp.Manager
}

func NewChatsHandler(db *repository.Database, waManager *whatsapp.Manager) *ChatsHandler {
	return &ChatsHandler{
		instanceRepo: repository.NewInstanceRepository(db),
		waManager:    waManager,
	}
}

// SetChatDisappearingTimer sets the disappearing message timer for a specific chat
// PATCH /v1/{phone_number_id}/chats/{chat_jid}/disappearing
func (h *ChatsHandler) SetChatDisappearingTimer(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	phoneNumberID := c.Params("phone_number_id")
	chatJID := c.Params("chat_jid")
	logger := pkglogger.Get()

	// Parse request
	var req struct {
		Duration int64 `json:"duration"` // Duration in seconds (0 = off, 86400 = 1 day, 604800 = 7 days, 7776000 = 90 days)
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

	// Validate duration (WhatsApp supports: 0, 86400, 604800, 7776000)
	validDurations := map[int64]bool{
		0:       true, // Off
		86400:   true, // 1 day
		604800:  true, // 7 days
		7776000: true, // 90 days
	}

	if !validDurations[req.Duration] {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Invalid duration. Must be 0 (off), 86400 (1 day), 604800 (7 days), or 7776000 (90 days)",
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

	// Set disappearing timer
	timer := time.Duration(req.Duration) * time.Second
	err = h.waManager.SetChatDisappearingTimer(ctx, tenantID, instance.ID, chatJID, timer)
	if err != nil {
		logger.Error("Failed to set disappearing timer",
			zap.Error(err),
			zap.String("chat_jid", chatJID),
			zap.Int64("duration", req.Duration),
		)
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": fmt.Sprintf("Failed to set disappearing timer: %v", err),
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	logger.Info("Disappearing timer set",
		zap.String("chat_jid", chatJID),
		zap.Int64("duration", req.Duration),
	)

	return c.JSON(fiber.Map{
		"success":  true,
		"duration": req.Duration,
	})
}

// SetDefaultDisappearingTimer sets the default disappearing message timer for new chats
// PATCH /v1/{phone_number_id}/settings/disappearing
func (h *ChatsHandler) SetDefaultDisappearingTimer(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	phoneNumberID := c.Params("phone_number_id")
	logger := pkglogger.Get()

	// Parse request
	var req struct {
		Duration int64 `json:"duration"` // Duration in seconds (0 = off, 86400 = 1 day, 604800 = 7 days, 7776000 = 90 days)
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

	// Validate duration (WhatsApp supports: 0, 86400, 604800, 7776000)
	validDurations := map[int64]bool{
		0:       true, // Off
		86400:   true, // 1 day
		604800:  true, // 7 days
		7776000: true, // 90 days
	}

	if !validDurations[req.Duration] {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Invalid duration. Must be 0 (off), 86400 (1 day), 604800 (7 days), or 7776000 (90 days)",
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

	// Set default disappearing timer
	timer := time.Duration(req.Duration) * time.Second
	err = h.waManager.SetDefaultDisappearingTimer(ctx, tenantID, instance.ID, timer)
	if err != nil {
		logger.Error("Failed to set default disappearing timer",
			zap.Error(err),
			zap.Int64("duration", req.Duration),
		)
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": fmt.Sprintf("Failed to set default disappearing timer: %v", err),
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	logger.Info("Default disappearing timer set",
		zap.Int64("duration", req.Duration),
	)

	return c.JSON(fiber.Map{
		"success":  true,
		"duration": req.Duration,
	})
}
