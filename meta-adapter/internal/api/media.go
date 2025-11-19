package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/repository"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/storage"
	pkglogger "github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/logger"
)

type MediaHandler struct {
	instanceRepo *repository.InstanceRepository
	mediaStore   *storage.MediaStore
}

func NewMediaHandler(db *repository.Database, mediaStore *storage.MediaStore) *MediaHandler {
	return &MediaHandler{
		instanceRepo: repository.NewInstanceRepository(db),
		mediaStore:   mediaStore,
	}
}

// UploadMedia handles media file uploads and returns a reusable media ID
// POST /v1/{phone_number_id}/media
func (h *MediaHandler) UploadMedia(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	phoneNumberID := c.Params("phone_number_id")
	logger := pkglogger.Get()

	// Verify instance exists
	_, err := h.instanceRepo.FindByPhoneNumberID(ctx, tenantID, phoneNumberID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Instance not found",
				"type":    "NotFoundError",
				"code":    404,
			},
		})
	}

	// Parse multipart form
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Missing 'file' in multipart form",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// Validate file size (max 100MB for WhatsApp)
	const maxFileSize = 100 * 1024 * 1024 // 100MB
	if file.Size > maxFileSize {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": fmt.Sprintf("File too large. Maximum size is %d bytes", maxFileSize),
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// Open file
	fileHandle, err := file.Open()
	if err != nil {
		logger.Error("Failed to open uploaded file", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Failed to read uploaded file",
				"type":    "InternalError",
				"code":    500,
			},
		})
	}
	defer fileHandle.Close()

	// Read file data
	data, err := io.ReadAll(fileHandle)
	if err != nil {
		logger.Error("Failed to read file content", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Failed to read file content",
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	// Detect MIME type
	mimeType := http.DetectContentType(data)
	if file.Header.Get("Content-Type") != "" {
		mimeType = file.Header.Get("Content-Type")
	}

	// Validate MIME type (WhatsApp supported types)
	if !isValidMediaType(mimeType) {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": fmt.Sprintf("Unsupported media type: %s", mimeType),
				"type":    "ValidationError",
				"code":    400,
				"details": "Supported types: image/*, video/*, audio/*, application/pdf",
			},
		})
	}

	// Store in Redis
	mediaID, err := h.mediaStore.Store(ctx, data, mimeType, file.Filename)
	if err != nil {
		logger.Error("Failed to store media", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Failed to store media",
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	logger.Info("Media uploaded successfully",
		zap.String("media_id", mediaID),
		zap.String("tenant_id", tenantID),
		zap.String("filename", file.Filename),
		zap.String("mime_type", mimeType),
		zap.Int64("size", file.Size),
	)

	// Return Meta API compatible response
	return c.Status(201).JSON(fiber.Map{
		"id": mediaID,
	})
}

// GetMedia retrieves media information by ID
// GET /v1/{phone_number_id}/media/{media_id}
func (h *MediaHandler) GetMedia(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	phoneNumberID := c.Params("phone_number_id")
	mediaID := c.Params("media_id")
	logger := pkglogger.Get()

	// Verify instance exists
	_, err := h.instanceRepo.FindByPhoneNumberID(ctx, tenantID, phoneNumberID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Instance not found",
				"type":    "NotFoundError",
				"code":    404,
			},
		})
	}

	// Get media from store
	metadata, err := h.mediaStore.Get(ctx, mediaID)
	if err != nil {
		logger.Warn("Media not found", zap.String("media_id", mediaID), zap.Error(err))
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Media not found or expired",
				"type":    "NotFoundError",
				"code":    404,
			},
		})
	}

	// Return metadata (not the actual binary data)
	return c.JSON(fiber.Map{
		"id":        metadata.ID,
		"mime_type": metadata.MimeType,
		"filename":  metadata.Filename,
		"size":      metadata.Size,
		"uploaded_at": metadata.UploadedAt,
	})
}

// DeleteMedia deletes uploaded media by ID
// DELETE /v1/{phone_number_id}/media/{media_id}
func (h *MediaHandler) DeleteMedia(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	phoneNumberID := c.Params("phone_number_id")
	mediaID := c.Params("media_id")
	logger := pkglogger.Get()

	// Verify instance exists
	_, err := h.instanceRepo.FindByPhoneNumberID(ctx, tenantID, phoneNumberID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Instance not found",
				"type":    "NotFoundError",
				"code":    404,
			},
		})
	}

	// Delete from store
	if err := h.mediaStore.Delete(ctx, mediaID); err != nil {
		logger.Error("Failed to delete media", zap.String("media_id", mediaID), zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Failed to delete media",
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	logger.Info("Media deleted", zap.String("media_id", mediaID), zap.String("tenant_id", tenantID))

	return c.JSON(fiber.Map{
		"success": true,
	})
}

// isValidMediaType checks if the MIME type is supported by WhatsApp
func isValidMediaType(mimeType string) bool {
	validPrefixes := []string{
		"image/",
		"video/",
		"audio/",
		"application/pdf",
		"application/vnd.ms-powerpoint",
		"application/msword",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument",
	}

	for _, prefix := range validPrefixes {
		if strings.HasPrefix(mimeType, prefix) {
			return true
		}
	}

	return false
}
