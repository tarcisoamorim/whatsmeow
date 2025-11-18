package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/models"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/repository"
)

type InstanceHandler struct {
	db *repository.Database
}

func NewInstanceHandler(db *repository.Database) *InstanceHandler {
	return &InstanceHandler{db: db}
}

func (h *InstanceHandler) List(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	
	// TODO: Implement actual query
	instances := []interface{}{}
	
	return c.JSON(fiber.Map{
		"data": instances,
		"paging": fiber.Map{
			"next":     nil,
			"previous": nil,
		},
	})
}

func (h *InstanceHandler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	
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
	
	// Create instance
	instance := models.NewInstance(tenantID)
	if req.DisplayName != "" {
		instance.DisplayName = &req.DisplayName
	}
	if req.WebhookURL != "" {
		instance.WebhookURL = &req.WebhookURL
	}
	instance.WebhookEvents = req.WebhookEvents
	
	// TODO: Save to database
	
	return c.Status(201).JSON(instance)
}

func (h *InstanceHandler) Get(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)
	phoneNumberID := c.Params("phone_number_id")
	
	// TODO: Implement actual query
	_ = tenantID
	_ = phoneNumberID
	
	return c.JSON(fiber.Map{
		"id":              "instance-123",
		"phone_number_id": phoneNumberID,
		"status":          "disconnected",
	})
}

func (h *InstanceHandler) GetQRCode(c *fiber.Ctx) error {
	phoneNumberID := c.Params("phone_number_id")
	
	// TODO: Generate actual QR code
	return c.JSON(fiber.Map{
		"qr_code":    "data:image/png;base64,iVBORw0KGgo...",
		"expires_at": "2025-11-18T21:30:00Z",
	})
}
