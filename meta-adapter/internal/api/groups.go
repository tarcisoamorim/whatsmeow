package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mau.fi/whatsmeow/types"
	"go.uber.org/zap"

	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/repository"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/whatsapp"
	pkglogger "github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/logger"
)

type GroupsHandler struct {
	instanceRepo *repository.InstanceRepository
	waManager    *whatsapp.Manager
}

func NewGroupsHandler(db *repository.Database, waManager *whatsapp.Manager) *GroupsHandler {
	return &GroupsHandler{
		instanceRepo: repository.NewInstanceRepository(db),
		waManager:    waManager,
	}
}

// Create creates a new group
// POST /v1/groups
func (h *GroupsHandler) Create(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	logger := pkglogger.Get()

	// Parse request
	var req struct {
		InstanceID   string   `json:"instance_id"`
		Name         string   `json:"name"`
		Participants []string `json:"participants"`
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

	// Validate fields
	if req.InstanceID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Missing 'instance_id' field",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Missing 'name' field",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	if len(req.Name) > 25 {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Group name must be 25 characters or less",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	if len(req.Participants) == 0 {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "At least one participant is required",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// Get instance
	instance, err := h.instanceRepo.FindByPhoneNumberID(ctx, tenantID, req.InstanceID)
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
		logger.Error("Failed to get instance", zap.Error(err))
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

	// Create group
	groupInfo, err := h.waManager.CreateGroup(ctx, tenantID, instance.ID, req.Name, req.Participants)
	if err != nil {
		logger.Error("Failed to create group", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": fmt.Sprintf("Failed to create group: %v", err),
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	// Format response
	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"id":           groupInfo.JID.String(),
			"name":         groupInfo.Name,
			"owner":        groupInfo.OwnerJID.String(),
			"created_at":   groupInfo.GroupCreated.Format(time.RFC3339),
			"participants": len(groupInfo.Participants),
		},
	})
}

// List retrieves all groups the instance is a member of
// GET /v1/groups?instance_id=xxx
func (h *GroupsHandler) List(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	instanceID := c.Query("instance_id")
	logger := pkglogger.Get()

	if instanceID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Missing 'instance_id' query parameter",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// Get instance
	instance, err := h.instanceRepo.FindByPhoneNumberID(ctx, tenantID, instanceID)
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
		logger.Error("Failed to get instance", zap.Error(err))
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

	// Get joined groups
	groups, err := h.waManager.GetJoinedGroups(ctx, tenantID, instance.ID)
	if err != nil {
		logger.Error("Failed to get joined groups", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": fmt.Sprintf("Failed to get groups: %v", err),
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	// Format response
	data := make([]fiber.Map, len(groups))
	for i, group := range groups {
		data[i] = fiber.Map{
			"id":           group.JID.String(),
			"name":         group.Name,
			"topic":        group.Topic,
			"owner":        group.OwnerJID.String(),
			"created_at":   group.GroupCreated.Format(time.RFC3339),
			"participants": len(group.Participants),
			"is_announce":  group.IsAnnounce,
			"is_locked":    group.IsLocked,
		}
	}

	return c.JSON(fiber.Map{
		"data": data,
	})
}

// Get retrieves information about a specific group
// GET /v1/groups/:group_id
func (h *GroupsHandler) Get(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	groupID := c.Params("group_id")
	instanceID := c.Query("instance_id")
	logger := pkglogger.Get()

	if instanceID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Missing 'instance_id' query parameter",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// Get instance
	instance, err := h.instanceRepo.FindByPhoneNumberID(ctx, tenantID, instanceID)
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
		logger.Error("Failed to get instance", zap.Error(err))
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

	// Get group info
	groupInfo, err := h.waManager.GetGroupInfo(ctx, tenantID, instance.ID, groupID)
	if err != nil {
		logger.Error("Failed to get group info", zap.Error(err))
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"message": fmt.Sprintf("Group not found or inaccessible: %v", err),
				"type":    "NotFoundError",
				"code":    404,
			},
		})
	}

	// Format participants
	participants := make([]fiber.Map, len(groupInfo.Participants))
	for i, p := range groupInfo.Participants {
		participants[i] = fiber.Map{
			"jid":      p.JID.String(),
			"is_admin": p.IsAdmin,
			"is_super_admin": p.IsSuperAdmin,
		}
	}

	// Format response
	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"id":           groupInfo.JID.String(),
			"name":         groupInfo.Name,
			"topic":        groupInfo.Topic,
			"owner":        groupInfo.OwnerJID.String(),
			"created_at":   groupInfo.GroupCreated.Format(time.RFC3339),
			"is_announce":  groupInfo.IsAnnounce,
			"is_locked":    groupInfo.IsLocked,
			"participants": participants,
		},
	})
}

// Update updates group name or description
// PATCH /v1/groups/:group_id
func (h *GroupsHandler) Update(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	groupID := c.Params("group_id")
	logger := pkglogger.Get()

	// Parse request
	var req struct {
		InstanceID  string `json:"instance_id"`
		Name        string `json:"name,omitempty"`
		Description string `json:"description,omitempty"`
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

	if req.InstanceID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Missing 'instance_id' field",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// Get instance
	instance, err := h.instanceRepo.FindByPhoneNumberID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Instance not found",
				"type":    "NotFoundError",
				"code":    404,
			},
		})
	}

	// Update name if provided
	if req.Name != "" {
		if len(req.Name) > 25 {
			return c.Status(400).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "Group name must be 25 characters or less",
					"type":    "ValidationError",
					"code":    400,
				},
			})
		}

		err = h.waManager.SetGroupName(ctx, tenantID, instance.ID, groupID, req.Name)
		if err != nil {
			logger.Error("Failed to update group name", zap.Error(err))
			return c.Status(500).JSON(fiber.Map{
				"error": fiber.Map{
					"message": fmt.Sprintf("Failed to update group name: %v", err),
					"type":    "InternalError",
					"code":    500,
				},
			})
		}
	}

	// Update description if provided
	if req.Description != "" {
		err = h.waManager.SetGroupTopic(ctx, tenantID, instance.ID, groupID, req.Description)
		if err != nil {
			logger.Error("Failed to update group description", zap.Error(err))
			return c.Status(500).JSON(fiber.Map{
				"error": fiber.Map{
					"message": fmt.Sprintf("Failed to update group description: %v", err),
					"type":    "InternalError",
					"code":    500,
				},
			})
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}

// Leave leaves a group
// DELETE /v1/groups/:group_id
func (h *GroupsHandler) Leave(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	groupID := c.Params("group_id")
	logger := pkglogger.Get()

	// Parse request
	var req struct {
		InstanceID string `json:"instance_id"`
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

	if req.InstanceID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Missing 'instance_id' field",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// Get instance
	instance, err := h.instanceRepo.FindByPhoneNumberID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Instance not found",
				"type":    "NotFoundError",
				"code":    404,
			},
		})
	}

	// Leave group
	err = h.waManager.LeaveGroup(ctx, tenantID, instance.ID, groupID)
	if err != nil {
		logger.Error("Failed to leave group", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": fmt.Sprintf("Failed to leave group: %v", err),
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}

// UploadPhoto uploads a new group photo
// POST /v1/groups/:group_id/photo
func (h *GroupsHandler) UploadPhoto(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	groupID := c.Params("group_id")
	instanceID := c.FormValue("instance_id")
	logger := pkglogger.Get()

	if instanceID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Missing 'instance_id' field",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// Get instance
	instance, err := h.instanceRepo.FindByPhoneNumberID(ctx, tenantID, instanceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Instance not found",
				"type":    "NotFoundError",
				"code":    404,
			},
		})
	}

	// Get uploaded file
	file, err := c.FormFile("photo")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Missing 'photo' file",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// Open file
	src, err := file.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Failed to open uploaded file",
				"type":    "InternalError",
				"code":    500,
			},
		})
	}
	defer src.Close()

	// Read file data
	imageData, err := io.ReadAll(src)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Failed to read uploaded file",
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	// Set group photo
	pictureID, err := h.waManager.SetGroupPhoto(ctx, tenantID, instance.ID, groupID, imageData)
	if err != nil {
		logger.Error("Failed to set group photo", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": fmt.Sprintf("Failed to set group photo: %v", err),
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	return c.JSON(fiber.Map{
		"success":    true,
		"picture_id": pictureID,
	})
}

// GetInviteLink gets the group invite link
// GET /v1/groups/:group_id/invite
func (h *GroupsHandler) GetInviteLink(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	groupID := c.Params("group_id")
	instanceID := c.Query("instance_id")
	reset := c.QueryBool("reset", false)
	logger := pkglogger.Get()

	if instanceID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Missing 'instance_id' query parameter",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// Get instance
	instance, err := h.instanceRepo.FindByPhoneNumberID(ctx, tenantID, instanceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Instance not found",
				"type":    "NotFoundError",
				"code":    404,
			},
		})
	}

	// Get invite link
	inviteLink, err := h.waManager.GetGroupInviteLink(ctx, tenantID, instance.ID, groupID, reset)
	if err != nil {
		logger.Error("Failed to get invite link", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": fmt.Sprintf("Failed to get invite link: %v", err),
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	return c.JSON(fiber.Map{
		"invite_link": inviteLink,
	})
}

// Join joins a group using an invite code
// POST /v1/groups/join
func (h *GroupsHandler) Join(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	logger := pkglogger.Get()

	// Parse request
	var req struct {
		InstanceID string `json:"instance_id"`
		InviteCode string `json:"invite_code"`
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

	if req.InstanceID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Missing 'instance_id' field",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	if req.InviteCode == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Missing 'invite_code' field",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// Get instance
	instance, err := h.instanceRepo.FindByPhoneNumberID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Instance not found",
				"type":    "NotFoundError",
				"code":    404,
			},
		})
	}

	// Join group
	groupJID, err := h.waManager.JoinGroupWithLink(ctx, tenantID, instance.ID, req.InviteCode)
	if err != nil {
		logger.Error("Failed to join group", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": fmt.Sprintf("Failed to join group: %v", err),
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"group_id": groupJID.String(),
	})
}

// AddParticipants adds participants to a group
// POST /v1/groups/:group_id/participants
func (h *GroupsHandler) AddParticipants(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	groupID := c.Params("group_id")
	logger := pkglogger.Get()

	// Parse request
	var req struct {
		InstanceID   string   `json:"instance_id"`
		Participants []string `json:"participants"`
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

	if req.InstanceID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Missing 'instance_id' field",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	if len(req.Participants) == 0 {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "At least one participant is required",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// Get instance
	instance, err := h.instanceRepo.FindByPhoneNumberID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Instance not found",
				"type":    "NotFoundError",
				"code":    404,
			},
		})
	}

	// Add participants
	results, err := h.waManager.AddParticipants(ctx, tenantID, instance.ID, groupID, req.Participants)
	if err != nil {
		logger.Error("Failed to add participants", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": fmt.Sprintf("Failed to add participants: %v", err),
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	// Format results
	resultData := formatParticipantResults(results)

	return c.JSON(fiber.Map{
		"success": true,
		"results": resultData,
	})
}

// RemoveParticipant removes a participant from a group
// DELETE /v1/groups/:group_id/participants/:phone
func (h *GroupsHandler) RemoveParticipant(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	groupID := c.Params("group_id")
	phone := c.Params("phone")
	instanceID := c.Query("instance_id")
	logger := pkglogger.Get()

	if instanceID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Missing 'instance_id' query parameter",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// Get instance
	instance, err := h.instanceRepo.FindByPhoneNumberID(ctx, tenantID, instanceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Instance not found",
				"type":    "NotFoundError",
				"code":    404,
			},
		})
	}

	// Remove participant
	results, err := h.waManager.RemoveParticipants(ctx, tenantID, instance.ID, groupID, []string{phone})
	if err != nil {
		logger.Error("Failed to remove participant", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": fmt.Sprintf("Failed to remove participant: %v", err),
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	// Format results
	resultData := formatParticipantResults(results)

	return c.JSON(fiber.Map{
		"success": true,
		"results": resultData,
	})
}

// PromoteAdmins promotes participants to admin status
// POST /v1/groups/:group_id/admins
func (h *GroupsHandler) PromoteAdmins(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	groupID := c.Params("group_id")
	logger := pkglogger.Get()

	// Parse request
	var req struct {
		InstanceID   string   `json:"instance_id"`
		Participants []string `json:"participants"`
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

	if req.InstanceID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Missing 'instance_id' field",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// Get instance
	instance, err := h.instanceRepo.FindByPhoneNumberID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Instance not found",
				"type":    "NotFoundError",
				"code":    404,
			},
		})
	}

	// Promote participants
	results, err := h.waManager.PromoteParticipants(ctx, tenantID, instance.ID, groupID, req.Participants)
	if err != nil {
		logger.Error("Failed to promote participants", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": fmt.Sprintf("Failed to promote participants: %v", err),
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	// Format results
	resultData := formatParticipantResults(results)

	return c.JSON(fiber.Map{
		"success": true,
		"results": resultData,
	})
}

// DemoteAdmin demotes an admin to regular participant
// DELETE /v1/groups/:group_id/admins/:phone
func (h *GroupsHandler) DemoteAdmin(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	groupID := c.Params("group_id")
	phone := c.Params("phone")
	instanceID := c.Query("instance_id")
	logger := pkglogger.Get()

	if instanceID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Missing 'instance_id' query parameter",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// Get instance
	instance, err := h.instanceRepo.FindByPhoneNumberID(ctx, tenantID, instanceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Instance not found",
				"type":    "NotFoundError",
				"code":    404,
			},
		})
	}

	// Demote participant
	results, err := h.waManager.DemoteParticipants(ctx, tenantID, instance.ID, groupID, []string{phone})
	if err != nil {
		logger.Error("Failed to demote participant", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"message": fmt.Sprintf("Failed to demote participant: %v", err),
				"type":    "InternalError",
				"code":    500,
			},
		})
	}

	// Format results
	resultData := formatParticipantResults(results)

	return c.JSON(fiber.Map{
		"success": true,
		"results": resultData,
	})
}

// UpdateSettings updates group settings (announce, locked)
// PATCH /v1/groups/:group_id/settings
func (h *GroupsHandler) UpdateSettings(c *fiber.Ctx) error {
	ctx := context.Background()
	tenantID := c.Locals("tenant_id").(string)
	groupID := c.Params("group_id")
	logger := pkglogger.Get()

	// Parse request
	var req struct {
		InstanceID string `json:"instance_id"`
		Announce   *bool  `json:"announce,omitempty"`
		Locked     *bool  `json:"locked,omitempty"`
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

	if req.InstanceID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Missing 'instance_id' field",
				"type":    "ValidationError",
				"code":    400,
			},
		})
	}

	// Get instance
	instance, err := h.instanceRepo.FindByPhoneNumberID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "Instance not found",
				"type":    "NotFoundError",
				"code":    404,
			},
		})
	}

	// Update announce setting if provided
	if req.Announce != nil {
		err = h.waManager.SetGroupAnnounce(ctx, tenantID, instance.ID, groupID, *req.Announce)
		if err != nil {
			logger.Error("Failed to update group announce setting", zap.Error(err))
			return c.Status(500).JSON(fiber.Map{
				"error": fiber.Map{
					"message": fmt.Sprintf("Failed to update announce setting: %v", err),
					"type":    "InternalError",
					"code":    500,
				},
			})
		}
	}

	// Update locked setting if provided
	if req.Locked != nil {
		err = h.waManager.SetGroupLocked(ctx, tenantID, instance.ID, groupID, *req.Locked)
		if err != nil {
			logger.Error("Failed to update group locked setting", zap.Error(err))
			return c.Status(500).JSON(fiber.Map{
				"error": fiber.Map{
					"message": fmt.Sprintf("Failed to update locked setting: %v", err),
					"type":    "InternalError",
					"code":    500,
				},
			})
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}

// formatParticipantResults formats participant operation results
func formatParticipantResults(results []types.GroupParticipant) []fiber.Map {
	data := make([]fiber.Map, len(results))
	for i, result := range results {
		data[i] = fiber.Map{
			"jid":       result.JID.String(),
			"error":     result.Error,
			"add_request": result.AddRequest.String(),
		}
	}
	return data
}
