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

type MessageHandler struct {
	messageRepo  *repository.MessageRepository
	instanceRepo *repository.InstanceRepository
}

func NewMessageHandler(db *repository.Database) *MessageHandler {
	return &MessageHandler{
		messageRepo:  repository.NewMessageRepository(db),
		instanceRepo: repository.NewInstanceRepository(db),
	}
}

func (h *MessageHandler) Send(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	phoneNumberID := c.Params("phone_number_id")
	logger := pkglogger.Get()

	// Parse request
	var req struct {
		MessagingProduct string `json:"messaging_product"`
		To               string `json:"to"`
		Type             string `json:"type"`
		Text             *struct {
			Body       string `json:"body"`
			PreviewURL bool   `json:"preview_url,omitempty"`
		} `json:"text,omitempty"`
		Image *struct {
			Link    string `json:"link,omitempty"`
			Caption string `json:"caption,omitempty"`
		} `json:"image,omitempty"`
		Audio *struct {
			Link string `json:"link"`
		} `json:"audio,omitempty"`
		Video *struct {
			Link    string `json:"link,omitempty"`
			Caption string `json:"caption,omitempty"`
		} `json:"video,omitempty"`
		Document *struct {
			Link     string `json:"link,omitempty"`
			Filename string `json:"filename,omitempty"`
			Caption  string `json:"caption,omitempty"`
		} `json:"document,omitempty"`
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

	if req.Type == "" {
		req.Type = "text" // Default to text
	}

	// Get instance to verify it exists and is connected
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

	// Build content based on message type
	content := make(map[string]interface{})
	switch req.Type {
	case "text":
		if req.Text == nil || req.Text.Body == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "Missing text content",
					"type":    "ValidationError",
					"code":    400,
				},
			})
		}
		content["text"] = map[string]interface{}{
			"body":        req.Text.Body,
			"preview_url": req.Text.PreviewURL,
		}
	case "image":
		if req.Image == nil {
			return c.Status(400).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "Missing image content",
					"type":    "ValidationError",
					"code":    400,
				},
			})
		}
		content["image"] = req.Image
	case "audio":
		if req.Audio == nil {
			return c.Status(400).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "Missing audio content",
					"type":    "ValidationError",
					"code":    400,
				},
			})
		}
		content["audio"] = req.Audio
	case "video":
		if req.Video == nil {
			return c.Status(400).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "Missing video content",
					"type":    "ValidationError",
					"code":    400,
				},
			})
		}
		content["video"] = req.Video
	case "document":
		if req.Document == nil {
			return c.Status(400).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "Missing document content",
					"type":    "ValidationError",
					"code":    400,
				},
			})
		}
		content["document"] = req.Document
	}

	// Create message record
	message := models.NewOutboundTextMessage(tenantID, instance.ID, phoneNumberID, req.To, "")
	message.Type = req.Type
	message.Content = content

	// Save to database
	if err := h.messageRepo.Create(ctx, message); err != nil {
		logger.Error("Failed to save message", zap.Error(err), zap.String("tenant_id", tenantID))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Failed to save message",
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	logger.Info("Message created",
		zap.String("message_id", message.MessageID),
		zap.String("tenant_id", tenantID),
		zap.String("instance_id", instance.ID),
		zap.String("to", req.To),
		zap.String("type", req.Type),
	)

	// TODO: Actually send via whatsmeow
	// For now, mark as pending - whatsmeow integration will send and update status

	// Return Meta API compatible response
	return c.JSON(fiber.Map{
		"messaging_product": "whatsapp",
		"contacts": []fiber.Map{
			{"input": req.To, "wa_id": req.To},
		},
		"messages": []fiber.Map{
			{"id": message.MessageID},
		},
	})
}

func (h *MessageHandler) List(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	phoneNumberID := c.Params("phone_number_id")
	logger := pkglogger.Get()

	// Get instance to verify it exists
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

	// Parse pagination params
	limit := c.QueryInt("limit", 25)  // Default 25 messages
	offset := c.QueryInt("offset", 0) // Default offset 0

	// Validate pagination params
	if limit > 100 {
		limit = 100 // Max 100 messages per request
	}
	if limit < 1 {
		limit = 25
	}

	// Get messages from database
	messages, err := h.messageRepo.ListByInstance(ctx, tenantID, instance.ID, limit, offset)
	if err != nil {
		logger.Error("Failed to list messages", zap.Error(err), zap.String("tenant_id", tenantID), zap.String("instance_id", instance.ID))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Failed to retrieve messages",
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	// Get total count for pagination
	totalCount, err := h.messageRepo.CountByInstance(ctx, tenantID, instance.ID)
	if err != nil {
		logger.Error("Failed to count messages", zap.Error(err), zap.String("tenant_id", tenantID), zap.String("instance_id", instance.ID))
		// Don't fail the request, just log the error
		totalCount = 0
	}

	// Convert to response format
	data := make([]fiber.Map, len(messages))
	for i, msg := range messages {
		data[i] = fiber.Map{
			"id":        msg.MessageID,
			"from":      msg.From,
			"to":        msg.To,
			"type":      msg.Type,
			"direction": msg.Direction,
			"status":    msg.Status,
			"timestamp": msg.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		}

		// Add content based on type
		if msg.Type == "text" {
			if textContent, ok := msg.Content["text"].(map[string]interface{}); ok {
				if body, ok := textContent["body"].(string); ok {
					data[i]["text"] = fiber.Map{"body": body}
				}
			}
		} else {
			data[i]["content"] = msg.Content
		}

		// Add timestamps if available
		if msg.SentAt != nil {
			data[i]["sent_at"] = msg.SentAt.Format("2006-01-02T15:04:05Z07:00")
		}
		if msg.DeliveredAt != nil {
			data[i]["delivered_at"] = msg.DeliveredAt.Format("2006-01-02T15:04:05Z07:00")
		}
		if msg.ReadAt != nil {
			data[i]["read_at"] = msg.ReadAt.Format("2006-01-02T15:04:05Z07:00")
		}
		if msg.Error != nil {
			data[i]["error"] = *msg.Error
		}
	}

	// Build paging info
	paging := fiber.Map{
		"total":  totalCount,
		"limit":  limit,
		"offset": offset,
	}

	// Add next/previous cursors if applicable
	if offset+limit < totalCount {
		paging["next"] = offset + limit
	}
	if offset > 0 {
		paging["previous"] = offset - limit
		if paging["previous"].(int) < 0 {
			paging["previous"] = 0
		}
	}

	return c.JSON(fiber.Map{
		"data":   data,
		"paging": paging,
	})
}
