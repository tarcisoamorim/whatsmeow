package api

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/adapter"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/config"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/storage"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/webhook"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/whatsapp"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/errors"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/models"
	"go.mau.fi/whatsmeow/types/events"
)

// Handler holds all dependencies for API handlers
type Handler struct {
	cfg        *config.Config
	client     *whatsapp.Client
	translator *adapter.MessageTranslator
	dispatcher *webhook.Dispatcher
	store      *storage.SQLiteStore
}

// NewHandler creates a new API handler
func NewHandler(cfg *config.Config, client *whatsapp.Client, store *storage.SQLiteStore) *Handler {
	h := &Handler{
		cfg:        cfg,
		client:     client,
		translator: adapter.NewMessageTranslator(client),
		store:      store,
	}

	// Setup webhook dispatcher if configured
	if cfg.HasWebhook() {
		phoneNumberID := "default"
		if client.IsLoggedIn() {
			phoneNumberID = client.GetOwnJID().User
		}

		h.dispatcher = webhook.NewDispatcher(
			cfg.Webhook.URL,
			cfg.Webhook.Timeout,
			cfg.Webhook.RetryAttempts,
			phoneNumberID,
		)

		// Register event handlers for webhooks
		client.AddEventHandler(func(evt interface{}) {
			h.handleWebhookEvent(evt)
		})
	}

	return h
}

// handleWebhookEvent dispatches events to webhook
func (h *Handler) handleWebhookEvent(evt interface{}) {
	if h.dispatcher == nil {
		return
	}

	switch e := evt.(type) {
	case *events.Message:
		if !e.Info.IsFromMe {
			if err := h.dispatcher.DispatchMessage(e); err != nil {
				log.Error().Err(err).Msg("Failed to dispatch message webhook")
			}
		}
	case *events.Receipt:
		if err := h.dispatcher.DispatchReceipt(e); err != nil {
			log.Error().Err(err).Msg("Failed to dispatch receipt webhook")
		}
	}
}

// Health check handler
func (h *Handler) Health(c *fiber.Ctx) error {
	response := models.HealthResponse{
		Status:  "ok",
		Version: "1.0.0",
		Time:    time.Now(),
	}

	if h.client.IsLoggedIn() {
		jid := h.client.GetOwnJID()
		response.Session = &models.SessionStatusResponse{
			SessionID:   jid.String(),
			Connected:   h.client.IsConnected(),
			PhoneNumber: jid.User,
			PushName:    h.client.GetPushName(),
		}
	}

	return successResponse(c, response)
}

// GetQRCode generates QR code for pairing
func (h *Handler) GetQRCode(c *fiber.Ctx) error {
	if h.client.IsLoggedIn() {
		return errorResponse(c, errors.NewError(
			fiber.StatusConflict,
			errors.ErrCodeInvalidParameter,
			"already_logged_in",
			"Already logged in. Logout first to generate new QR code.",
		))
	}

	ctx, cancel := context.WithTimeout(c.Context(), 60*time.Second)
	defer cancel()

	qrCode, err := h.client.GetQRCode(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate QR code")
		return HandleError(c, errors.WrapError(err, "failed to generate QR code"))
	}

	response := models.QRCodeResponse{
		QRCode:    qrCode,
		ExpiresAt: time.Now().Add(60 * time.Second),
		Timeout:   60,
	}

	return successResponse(c, response)
}

// GetSessionStatus returns current session status
func (h *Handler) GetSessionStatus(c *fiber.Ctx) error {
	response := models.SessionStatusResponse{
		Connected: h.client.IsConnected(),
	}

	if h.client.IsLoggedIn() {
		jid := h.client.GetOwnJID()
		response.SessionID = jid.String()
		response.PhoneNumber = jid.User
		response.PushName = h.client.GetPushName()
	}

	return successResponse(c, response)
}

// Logout logs out from WhatsApp
func (h *Handler) Logout(c *fiber.Ctx) error {
	if !h.client.IsLoggedIn() {
		return errorResponse(c, errors.NewError(
			fiber.StatusBadRequest,
			errors.ErrCodeInvalidParameter,
			"not_logged_in",
			"Not logged in",
		))
	}

	if err := h.client.Logout(); err != nil {
		log.Error().Err(err).Msg("Failed to logout")
		return HandleError(c, errors.WrapError(err, "failed to logout"))
	}

	return successResponse(c, models.SuccessResponse{
		Success: true,
		Message: "Logged out successfully",
	})
}

// SendMessage sends a message (Meta API compatible)
func (h *Handler) SendMessage(c *fiber.Ctx) error {
	// Check connection
	if !h.client.IsConnected() {
		return errorResponse(c, errors.ErrNotConnected)
	}

	// Parse request
	var req models.SendMessageRequest
	if err := c.BodyParser(&req); err != nil {
		log.Warn().Err(err).Msg("Failed to parse request body")
		return errorResponse(c, errors.ValidationError("body", "invalid JSON"))
	}

	// Validate required fields
	if req.To == "" {
		return errorResponse(c, errors.ValidationError("to", "recipient is required"))
	}

	if req.Type == "" {
		return errorResponse(c, errors.ValidationError("type", "message type is required"))
	}

	// Set messaging product if not provided
	if req.MessagingProduct == "" {
		req.MessagingProduct = "whatsapp"
	}

	// Translate to whatsmeow format
	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

	message, err := h.translator.TranslateToWhatsApp(ctx, &req)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to translate message")
		return HandleError(c, err)
	}

	// Send message
	msgID, timestamp, err := h.client.SendMessage(ctx, req.To, message)
	if err != nil {
		log.Error().Err(err).Str("to", req.To).Msg("Failed to send message")
		return HandleError(c, errors.WrapError(err, "failed to send message"))
	}

	// Build response (Meta API compatible)
	response := models.SendMessageResponse{
		MessagingProduct: "whatsapp",
		Contacts: []models.ContactResult{
			{
				Input: req.To,
				WaID:  req.To,
			},
		},
		Messages: []models.MessageResult{
			{
				ID:            string(msgID),
				MessageStatus: "accepted",
			},
		},
	}

	log.Info().
		Str("to", req.To).
		Str("message_id", string(msgID)).
		Time("timestamp", timestamp).
		Msg("Message sent successfully")

	return successResponse(c, response)
}

// GetMedia retrieves media (for incoming messages)
func (h *Handler) GetMedia(c *fiber.Ctx) error {
	mediaID := c.Params("media_id")
	if mediaID == "" {
		return errorResponse(c, errors.ValidationError("media_id", "media ID is required"))
	}

	// Check cache first
	cached, err := h.store.GetCachedMedia(mediaID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get cached media")
	}

	if cached != nil {
		// Return cached media info
		return successResponse(c, map[string]interface{}{
			"url":       cached.WhatsAppURL,
			"mime_type": cached.MimeType,
			"sha256":    cached.SHA256,
			"size":      cached.Size,
		})
	}

	// If not cached, return error
	// In a full implementation, you would download from WhatsApp and cache it
	return errorResponse(c, errors.NewError(
		fiber.StatusNotFound,
		errors.ErrCodeMediaDownloadError,
		"media_not_found",
		"Media not found or expired",
	))
}

// UploadMedia uploads media file
func (h *Handler) UploadMedia(c *fiber.Ctx) error {
	if !h.client.IsConnected() {
		return errorResponse(c, errors.ErrNotConnected)
	}

	// Parse multipart form
	file, err := c.FormFile("file")
	if err != nil {
		return errorResponse(c, errors.ValidationError("file", "file is required"))
	}

	// Read file data
	fileData, err := file.Open()
	if err != nil {
		return HandleError(c, errors.WrapError(err, "failed to read file"))
	}
	defer fileData.Close()

	// Read file content
	buf := make([]byte, file.Size)
	if _, err := fileData.Read(buf); err != nil {
		return HandleError(c, errors.WrapError(err, "failed to read file content"))
	}

	// Determine media type from content type
	// TODO: Better media type detection
	mediaType := "image"
	// Upload would happen here...

	// Generate media ID
	mediaID := uuid.New().String()

	// Store in cache
	// TODO: Actual upload to WhatsApp and cache the result

	response := models.MediaUploadResponse{
		ID: mediaID,
	}

	return successResponse(c, response)
}

// WebhookVerification handles webhook verification (Meta API requirement)
func (h *Handler) WebhookVerification(c *fiber.Ctx) error {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	if mode == "subscribe" && token == h.cfg.Webhook.VerifyToken {
		log.Info().Msg("Webhook verified successfully")
		return c.SendString(challenge)
	}

	return errorResponse(c, errors.ErrUnauthorized)
}
