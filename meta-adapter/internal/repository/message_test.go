package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/models"
)

func TestMessageRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMessageRepository(db)
	ctx := context.Background()

	tenantID := uuid.New().String()
	instanceID := uuid.New().String()
	message := &models.Message{
		TenantID:   tenantID,
		InstanceID: instanceID,
		ID:         uuid.New().String(),
		MessageID:  "wamid.test123",
		Direction:  "outbound",
		Type:       "text",
		Status:     "pending",
		Content: map[string]interface{}{
			"text": "Hello, World!",
		},
		From:      "5511999999999",
		To:        "5511988888888",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, message)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify message was created
	found, err := repo.FindByID(ctx, tenantID, instanceID, message.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.ID != message.ID {
		t.Errorf("Message ID mismatch: got %s, want %s", found.ID, message.ID)
	}
	if found.MessageID != message.MessageID {
		t.Errorf("MessageID mismatch: got %s, want %s", found.MessageID, message.MessageID)
	}
	if found.Type != "text" {
		t.Errorf("Type mismatch: got %s, want text", found.Type)
	}
}

func TestMessageRepository_FindByMessageID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMessageRepository(db)
	ctx := context.Background()

	tenantID := uuid.New().String()
	instanceID := uuid.New().String()
	messageID := "wamid.unique123"

	message := &models.Message{
		TenantID:   tenantID,
		InstanceID: instanceID,
		ID:         uuid.New().String(),
		MessageID:  messageID,
		Direction:  "inbound",
		Type:       "text",
		Status:     "delivered",
		Content: map[string]interface{}{
			"text": "Test message",
		},
		From:      "5511977777777",
		To:        "5511999999999",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, message)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Find by message ID
	found, err := repo.FindByMessageID(ctx, tenantID, instanceID, messageID)
	if err != nil {
		t.Fatalf("FindByMessageID() error = %v", err)
	}

	if found.ID != message.ID {
		t.Errorf("Message ID mismatch: got %s, want %s", found.ID, message.ID)
	}
	if found.MessageID != messageID {
		t.Errorf("MessageID mismatch: got %s, want %s", found.MessageID, messageID)
	}
}

func TestMessageRepository_ListByInstance(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMessageRepository(db)
	ctx := context.Background()

	tenantID := uuid.New().String()
	instanceID := uuid.New().String()

	// Create multiple messages
	for i := 0; i < 5; i++ {
		message := &models.Message{
			TenantID:   tenantID,
			InstanceID: instanceID,
			ID:         uuid.New().String(),
			MessageID:  uuid.New().String(),
			Direction:  "outbound",
			Type:       "text",
			Status:     "sent",
			Content: map[string]interface{}{
				"text": "Test message",
			},
			From:      "5511999999999",
			To:        "5511988888888",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := repo.Create(ctx, message); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// List messages
	messages, err := repo.ListByInstance(ctx, tenantID, instanceID, 10, 0)
	if err != nil {
		t.Fatalf("ListByInstance() error = %v", err)
	}

	if len(messages) < 5 {
		t.Errorf("ListByInstance() returned %d messages, expected at least 5", len(messages))
	}
}

func TestMessageRepository_ListByInstance_Pagination(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMessageRepository(db)
	ctx := context.Background()

	tenantID := uuid.New().String()
	instanceID := uuid.New().String()

	// Create 10 messages
	for i := 0; i < 10; i++ {
		message := &models.Message{
			TenantID:   tenantID,
			InstanceID: instanceID,
			ID:         uuid.New().String(),
			MessageID:  uuid.New().String(),
			Direction:  "outbound",
			Type:       "text",
			Status:     "sent",
			Content:    map[string]interface{}{"text": "Test"},
			From:       "5511999999999",
			To:         "5511988888888",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if err := repo.Create(ctx, message); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Get first page (5 messages)
	page1, err := repo.ListByInstance(ctx, tenantID, instanceID, 5, 0)
	if err != nil {
		t.Fatalf("ListByInstance() page 1 error = %v", err)
	}

	// Get second page (next 5 messages)
	page2, err := repo.ListByInstance(ctx, tenantID, instanceID, 5, 5)
	if err != nil {
		t.Fatalf("ListByInstance() page 2 error = %v", err)
	}

	if len(page1) < 5 {
		t.Errorf("Page 1 returned %d messages, expected 5", len(page1))
	}
	if len(page2) < 5 {
		t.Errorf("Page 2 returned %d messages, expected 5", len(page2))
	}

	// Ensure pages don't overlap
	for _, m1 := range page1 {
		for _, m2 := range page2 {
			if m1.ID == m2.ID {
				t.Error("Pagination overlap: same message in both pages")
			}
		}
	}
}

func TestMessageRepository_CountByInstance(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMessageRepository(db)
	ctx := context.Background()

	tenantID := uuid.New().String()
	instanceID := uuid.New().String()

	// Create 7 messages
	for i := 0; i < 7; i++ {
		message := &models.Message{
			TenantID:   tenantID,
			InstanceID: instanceID,
			ID:         uuid.New().String(),
			MessageID:  uuid.New().String(),
			Direction:  "outbound",
			Type:       "text",
			Status:     "sent",
			Content:    map[string]interface{}{"text": "Test"},
			From:       "5511999999999",
			To:         "5511988888888",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if err := repo.Create(ctx, message); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	count, err := repo.CountByInstance(ctx, tenantID, instanceID)
	if err != nil {
		t.Fatalf("CountByInstance() error = %v", err)
	}

	if count < 7 {
		t.Errorf("CountByInstance() = %d, want at least 7", count)
	}
}

func TestMessageRepository_UpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMessageRepository(db)
	ctx := context.Background()

	tenantID := uuid.New().String()
	instanceID := uuid.New().String()
	message := &models.Message{
		TenantID:   tenantID,
		InstanceID: instanceID,
		ID:         uuid.New().String(),
		MessageID:  "wamid.statustest",
		Direction:  "outbound",
		Type:       "text",
		Status:     "pending",
		Content:    map[string]interface{}{"text": "Test"},
		From:       "5511999999999",
		To:         "5511988888888",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := repo.Create(ctx, message)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update status
	err = repo.UpdateStatus(ctx, tenantID, instanceID, message.ID, "sent")
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	// Verify status update
	found, err := repo.FindByID(ctx, tenantID, instanceID, message.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Status != "sent" {
		t.Errorf("Status not updated: got %s, want sent", found.Status)
	}
}

func TestMessageRepository_MarkAsSent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMessageRepository(db)
	ctx := context.Background()

	tenantID := uuid.New().String()
	instanceID := uuid.New().String()
	message := &models.Message{
		TenantID:   tenantID,
		InstanceID: instanceID,
		ID:         uuid.New().String(),
		MessageID:  "wamid.senttest",
		Direction:  "outbound",
		Type:       "text",
		Status:     "pending",
		Content:    map[string]interface{}{"text": "Test"},
		From:       "5511999999999",
		To:         "5511988888888",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := repo.Create(ctx, message)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Mark as sent
	err = repo.MarkAsSent(ctx, tenantID, instanceID, message.ID)
	if err != nil {
		t.Fatalf("MarkAsSent() error = %v", err)
	}

	// Verify
	found, err := repo.FindByID(ctx, tenantID, instanceID, message.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Status != "sent" {
		t.Errorf("Status not updated: got %s, want sent", found.Status)
	}
	if found.SentAt == nil {
		t.Error("SentAt should be set")
	}
}

func TestMessageRepository_MarkAsDelivered(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMessageRepository(db)
	ctx := context.Background()

	tenantID := uuid.New().String()
	instanceID := uuid.New().String()
	message := &models.Message{
		TenantID:   tenantID,
		InstanceID: instanceID,
		ID:         uuid.New().String(),
		MessageID:  "wamid.deliveredtest",
		Direction:  "outbound",
		Type:       "text",
		Status:     "sent",
		Content:    map[string]interface{}{"text": "Test"},
		From:       "5511999999999",
		To:         "5511988888888",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := repo.Create(ctx, message)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Mark as delivered
	err = repo.MarkAsDelivered(ctx, tenantID, instanceID, message.ID)
	if err != nil {
		t.Fatalf("MarkAsDelivered() error = %v", err)
	}

	// Verify
	found, err := repo.FindByID(ctx, tenantID, instanceID, message.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Status != "delivered" {
		t.Errorf("Status not updated: got %s, want delivered", found.Status)
	}
	if found.DeliveredAt == nil {
		t.Error("DeliveredAt should be set")
	}
}

func TestMessageRepository_MarkAsRead(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMessageRepository(db)
	ctx := context.Background()

	tenantID := uuid.New().String()
	instanceID := uuid.New().String()
	message := &models.Message{
		TenantID:   tenantID,
		InstanceID: instanceID,
		ID:         uuid.New().String(),
		MessageID:  "wamid.readtest",
		Direction:  "outbound",
		Type:       "text",
		Status:     "delivered",
		Content:    map[string]interface{}{"text": "Test"},
		From:       "5511999999999",
		To:         "5511988888888",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := repo.Create(ctx, message)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Mark as read
	err = repo.MarkAsRead(ctx, tenantID, instanceID, message.ID)
	if err != nil {
		t.Fatalf("MarkAsRead() error = %v", err)
	}

	// Verify
	found, err := repo.FindByID(ctx, tenantID, instanceID, message.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Status != "read" {
		t.Errorf("Status not updated: got %s, want read", found.Status)
	}
	if found.ReadAt == nil {
		t.Error("ReadAt should be set")
	}
}

func TestMessageRepository_MarkAsFailed(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMessageRepository(db)
	ctx := context.Background()

	tenantID := uuid.New().String()
	instanceID := uuid.New().String()
	message := &models.Message{
		TenantID:   tenantID,
		InstanceID: instanceID,
		ID:         uuid.New().String(),
		MessageID:  "wamid.failtest",
		Direction:  "outbound",
		Type:       "text",
		Status:     "pending",
		Content:    map[string]interface{}{"text": "Test"},
		From:       "5511999999999",
		To:         "5511988888888",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := repo.Create(ctx, message)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Mark as failed
	errorMsg := "Connection timeout"
	err = repo.MarkAsFailed(ctx, tenantID, instanceID, message.ID, errorMsg)
	if err != nil {
		t.Fatalf("MarkAsFailed() error = %v", err)
	}

	// Verify
	found, err := repo.FindByID(ctx, tenantID, instanceID, message.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Status != "failed" {
		t.Errorf("Status not updated: got %s, want failed", found.Status)
	}
	if found.ErrorMessage == nil || *found.ErrorMessage != errorMsg {
		t.Error("ErrorMessage not set correctly")
	}
}

func TestMessageRepository_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMessageRepository(db)
	ctx := context.Background()

	// Two different tenants
	tenant1ID := uuid.New().String()
	tenant2ID := uuid.New().String()
	instance1ID := uuid.New().String()
	instance2ID := uuid.New().String()

	message1 := &models.Message{
		TenantID:   tenant1ID,
		InstanceID: instance1ID,
		ID:         uuid.New().String(),
		MessageID:  "wamid.tenant1",
		Direction:  "outbound",
		Type:       "text",
		Status:     "sent",
		Content:    map[string]interface{}{"text": "Tenant 1"},
		From:       "5511999999999",
		To:         "5511988888888",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	message2 := &models.Message{
		TenantID:   tenant2ID,
		InstanceID: instance2ID,
		ID:         uuid.New().String(),
		MessageID:  "wamid.tenant2",
		Direction:  "outbound",
		Type:       "text",
		Status:     "sent",
		Content:    map[string]interface{}{"text": "Tenant 2"},
		From:       "5511999999999",
		To:         "5511988888888",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Create both messages
	if err := repo.Create(ctx, message1); err != nil {
		t.Fatalf("Create() message1 error = %v", err)
	}
	if err := repo.Create(ctx, message2); err != nil {
		t.Fatalf("Create() message2 error = %v", err)
	}

	// Tenant 1 should not see Tenant 2's message
	_, err := repo.FindByID(ctx, tenant1ID, instance1ID, message2.ID)
	if err != ErrNotFound {
		t.Error("Tenant isolation broken: tenant1 can access tenant2's message")
	}

	// Tenant 2 should not see Tenant 1's message
	_, err = repo.FindByID(ctx, tenant2ID, instance2ID, message1.ID)
	if err != ErrNotFound {
		t.Error("Tenant isolation broken: tenant2 can access tenant1's message")
	}
}

func TestMessageRepository_MultipleMessageTypes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMessageRepository(db)
	ctx := context.Background()

	tenantID := uuid.New().String()
	instanceID := uuid.New().String()

	messageTypes := []struct {
		msgType string
		content map[string]interface{}
	}{
		{
			msgType: "text",
			content: map[string]interface{}{"text": "Hello"},
		},
		{
			msgType: "image",
			content: map[string]interface{}{
				"url":      "https://example.com/image.jpg",
				"caption":  "Test image",
				"mimeType": "image/jpeg",
			},
		},
		{
			msgType: "audio",
			content: map[string]interface{}{
				"url":      "https://example.com/audio.mp3",
				"mimeType": "audio/mpeg",
			},
		},
		{
			msgType: "video",
			content: map[string]interface{}{
				"url":      "https://example.com/video.mp4",
				"caption":  "Test video",
				"mimeType": "video/mp4",
			},
		},
		{
			msgType: "document",
			content: map[string]interface{}{
				"url":      "https://example.com/doc.pdf",
				"filename": "document.pdf",
				"mimeType": "application/pdf",
			},
		},
	}

	for _, mt := range messageTypes {
		message := &models.Message{
			TenantID:   tenantID,
			InstanceID: instanceID,
			ID:         uuid.New().String(),
			MessageID:  uuid.New().String(),
			Direction:  "outbound",
			Type:       mt.msgType,
			Status:     "sent",
			Content:    mt.content,
			From:       "5511999999999",
			To:         "5511988888888",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		err := repo.Create(ctx, message)
		if err != nil {
			t.Fatalf("Create() %s message error = %v", mt.msgType, err)
		}

		// Verify JSONB content is stored correctly
		found, err := repo.FindByID(ctx, tenantID, instanceID, message.ID)
		if err != nil {
			t.Fatalf("FindByID() %s message error = %v", mt.msgType, err)
		}

		if found.Type != mt.msgType {
			t.Errorf("Type mismatch for %s: got %s", mt.msgType, found.Type)
		}
		if found.Content == nil {
			t.Errorf("Content is nil for %s message", mt.msgType)
		}
	}
}
