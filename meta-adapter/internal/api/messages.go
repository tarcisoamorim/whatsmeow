package api

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type MessageHandler struct{}

func NewMessageHandler() *MessageHandler {
	return &MessageHandler{}
}

func (h *MessageHandler) Send(c *fiber.Ctx) error {
	phoneNumberID := c.Params("phone_number_id")
	
	// Parse request
	var req struct {
		MessagingProduct string `json:"messaging_product"`
		To               string `json:"to"`
		Type             string `json:"type"`
		Text             *struct {
			Body       string `json:"body"`
			PreviewURL bool   `json:"preview_url"`
		} `json:"text,omitempty"`
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
	
	// Basic validation
	if req.To == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Missing 'to' field",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}
	
	// Generate message ID (Meta format)
	messageID := fmt.Sprintf("wamid.%s", uuid.New().String())
	
	// TODO: Actually send via whatsmeow
	
	return c.JSON(fiber.Map{
		"messaging_product": "whatsapp",
		"contacts": []fiber.Map{
			{"input": req.To, "wa_id": req.To},
		},
		"messages": []fiber.Map{
			{"id": messageID},
		},
	})
}

func (h *MessageHandler) List(c *fiber.Ctx) error {
	phoneNumberID := c.Params("phone_number_id")
	tenantID := c.Locals("tenant_id").(string)
	
	_ = phoneNumberID
	_ = tenantID
	
	// TODO: Query from database
	
	return c.JSON(fiber.Map{
		"data": []interface{}{},
		"paging": fiber.Map{
			"next":     nil,
			"previous": nil,
		},
	})
}
