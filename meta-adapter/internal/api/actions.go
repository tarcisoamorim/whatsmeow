package api

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/repository"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/whatsapp"
	pkglogger "github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/logger"
)

type ActionsHandler struct {
	instanceRepo *repository.InstanceRepository
	messageRepo  *repository.MessageRepository
	waManager    *whatsapp.Manager
}

func NewActionsHandler(db *repository.Database, waManager *whatsapp.Manager) *ActionsHandler {
	return &ActionsHandler{
		instanceRepo: repository.NewInstanceRepository(db),
		messageRepo:  repository.NewMessageRepository(db),
		waManager:    waManager,
	}
}

// MarkMessageRead marks a message as read
// POST /v1/{phone_number_id}/messages/{message_id}/read
func (h *ActionsHandler) MarkMessageRead(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	phoneNumberID := c.Params("phone_number_id")
	messageID := c.Params("message_id")
	logger := pkglogger.Get()

	// Parse request
	var req struct {
		Status string `json:"status"` // "read"
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
	if req.Status != "read" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Status must be 'read'",
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

	// Get message from database to find chat JID
	message, err := h.messageRepo.FindByMessageID(ctx, tenantID, instance.ID, messageID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Message not found",
				"type":    "NotFoundError",
				"code":    404,
			},
		})
	}

	// Determine chat JID (use From for inbound messages)
	chatJID := message.From
	if chatJID == "" {
		chatJID = message.To
	}

	// Mark message as read
	err = h.waManager.MarkMessageRead(ctx, tenantID, instance.ID, chatJID, []string{messageID}, message.Timestamp)
	if err != nil {
		logger.Error("Failed to mark message as read", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Failed to mark message as read: " + err.Error(),
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	// Update message in database
	now := time.Now()
	message.ReadAt = &now
	if err := h.messageRepo.Update(ctx, message); err != nil {
		logger.Error("Failed to update message read status", zap.Error(err))
		// Don't fail the request - WhatsApp was updated successfully
	}

	logger.Info("Message marked as read",
		zap.String("message_id", messageID),
		zap.String("chat_jid", chatJID),
	)

	return c.JSON(fiber.Map{
		"success": true,
	})
}

// DeleteMessage deletes a message for everyone
// DELETE /v1/{phone_number_id}/messages/{message_id}
func (h *ActionsHandler) DeleteMessage(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	phoneNumberID := c.Params("phone_number_id")
	messageID := c.Params("message_id")
	logger := pkglogger.Get()

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

	// Get message from database to find chat JID
	message, err := h.messageRepo.FindByMessageID(ctx, tenantID, instance.ID, messageID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Message not found",
				"type":    "NotFoundError",
				"code":    404,
			},
		})
	}

	// Determine chat JID
	chatJID := message.To
	if message.Direction == "inbound" {
		chatJID = message.From
	}

	// Delete message via WhatsApp
	err = h.waManager.DeleteMessage(ctx, tenantID, instance.ID, chatJID, messageID)
	if err != nil {
		logger.Error("Failed to delete message", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Failed to delete message: " + err.Error(),
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	// Soft delete message in database
	if err := h.messageRepo.SoftDelete(ctx, tenantID, instance.ID, message.ID); err != nil {
		logger.Error("Failed to soft delete message in database", zap.Error(err))
		// Don't fail the request - WhatsApp was updated successfully
	}

	logger.Info("Message deleted",
		zap.String("message_id", messageID),
		zap.String("chat_jid", chatJID),
	)

	return c.JSON(fiber.Map{
		"success": true,
	})
}

// ReactToMessage sends a reaction to a message
// POST /v1/{phone_number_id}/messages/{message_id}/react
func (h *ActionsHandler) ReactToMessage(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	phoneNumberID := c.Params("phone_number_id")
	messageID := c.Params("message_id")
	logger := pkglogger.Get()

	// Parse request
	var req struct {
		Emoji string `json:"emoji"` // Emoji reaction (e.g., "👍", "❤️", "😂")
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

	// Validate emoji
	if req.Emoji == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Missing 'emoji' field",
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

	// Get message from database to find chat JID
	message, err := h.messageRepo.FindByMessageID(ctx, tenantID, instance.ID, messageID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Message not found",
				"type":    "NotFoundError",
				"code":    404,
			},
		})
	}

	// Determine chat JID
	chatJID := message.From
	if chatJID == "" {
		chatJID = message.To
	}

	// Send reaction
	reactionID, err := h.waManager.ReactToMessage(ctx, tenantID, instance.ID, chatJID, messageID, req.Emoji)
	if err != nil {
		logger.Error("Failed to send reaction", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Failed to send reaction: " + err.Error(),
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	logger.Info("Reaction sent",
		zap.String("message_id", messageID),
		zap.String("emoji", req.Emoji),
		zap.String("reaction_id", reactionID),
	)

	return c.JSON(fiber.Map{
		"success": true,
		"id":      reactionID,
	})
}

// EditMessage edits a previously sent text message
// PATCH /v1/{phone_number_id}/messages/{message_id}
func (h *ActionsHandler) EditMessage(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	phoneNumberID := c.Params("phone_number_id")
	messageID := c.Params("message_id")
	logger := pkglogger.Get()

	// Parse request
	var req struct {
		Text *struct {
			Body string `json:"body"` // New text content
		} `json:"text"`
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

	// Validate text content
	if req.Text == nil || req.Text.Body == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Missing 'text.body' field",
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

	// Get message from database to find chat JID
	message, err := h.messageRepo.FindByMessageID(ctx, tenantID, instance.ID, messageID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Message not found",
				"type":    "NotFoundError",
				"code":    404,
			},
		})
	}

	// Verify message is outbound (can only edit own messages)
	if message.Direction != "outbound" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Can only edit outbound messages",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// Verify message is a text message
	if message.Type != "text" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Can only edit text messages",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// Determine chat JID
	chatJID := message.To
	if message.Direction == "inbound" {
		chatJID = message.From
	}

	// Edit message via WhatsApp
	editID, err := h.waManager.EditMessage(ctx, tenantID, instance.ID, chatJID, messageID, req.Text.Body)
	if err != nil {
		logger.Error("Failed to edit message", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Failed to edit message: " + err.Error(),
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	// Update message content in database
	if message.Content == nil {
		message.Content = make(map[string]interface{})
	}
	message.Content["text"] = map[string]interface{}{
		"body": req.Text.Body,
	}
	// Track edit history
	if message.Content["edit_history"] == nil {
		message.Content["edit_history"] = []map[string]interface{}{}
	}
	editHistory := message.Content["edit_history"].([]map[string]interface{})
	editHistory = append(editHistory, map[string]interface{}{
		"edited_at": time.Now().Format(time.RFC3339),
		"edit_id":   editID,
	})
	message.Content["edit_history"] = editHistory

	if err := h.messageRepo.Update(ctx, message); err != nil {
		logger.Error("Failed to update message in database", zap.Error(err))
		// Don't fail the request - WhatsApp was updated successfully
	}

	logger.Info("Message edited",
		zap.String("message_id", messageID),
		zap.String("edit_id", editID),
		zap.String("chat_jid", chatJID),
	)

	return c.JSON(fiber.Map{
		"success":    true,
		"message_id": editID,
	})
}
