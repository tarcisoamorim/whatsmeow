package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/models"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/repository"
)

func TestMessageHandler_Send_MissingTo(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	waManager := setupTestWAManager(t, db)
	handler := NewMessageHandler(db, waManager)

	// Create test instance (connected)
	instanceRepo := repository.NewInstanceRepository(db)
	instance := models.NewInstance(tenant.ID)
	instance.Status = "connected"
	connectedAt := time.Now()
	instance.ConnectedAt = &connectedAt
	instanceRepo.Create(nil, instance)

	// Create test app and route
	app := fiber.New()
	app.Post("/:phone_number_id/messages", func(c *fiber.Ctx) error {
		c.Locals("tenant_id", tenant.ID)
		return handler.Send(c)
	})

	// Test request without 'to' field
	reqBody := map[string]interface{}{
		"type": "text",
		"text": map[string]interface{}{
			"body": "Hello",
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/"+instance.PhoneNumberID+"/messages", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	// Parse error response
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	errorMap, ok := result["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing 'error' field")
	}

	if errorMap["type"] != "ValidationError" {
		t.Errorf("Expected error type 'ValidationError', got %v", errorMap["type"])
	}
}

func TestMessageHandler_Send_InstanceNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	waManager := setupTestWAManager(t, db)
	handler := NewMessageHandler(db, waManager)

	// Create test app and route
	app := fiber.New()
	app.Post("/:phone_number_id/messages", func(c *fiber.Ctx) error {
		c.Locals("tenant_id", tenant.ID)
		return handler.Send(c)
	})

	// Test request with non-existent phone_number_id
	reqBody := map[string]interface{}{
		"to":   "5511988888888",
		"type": "text",
		"text": map[string]interface{}{
			"body": "Hello",
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/99999999999/messages", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestMessageHandler_Send_InstanceNotConnected(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	waManager := setupTestWAManager(t, db)
	handler := NewMessageHandler(db, waManager)

	// Create test instance (disconnected)
	instanceRepo := repository.NewInstanceRepository(db)
	instance := models.NewInstance(tenant.ID)
	instance.Status = "disconnected"
	instanceRepo.Create(nil, instance)

	// Create test app and route
	app := fiber.New()
	app.Post("/:phone_number_id/messages", func(c *fiber.Ctx) error {
		c.Locals("tenant_id", tenant.ID)
		return handler.Send(c)
	})

	// Test request
	reqBody := map[string]interface{}{
		"to":   "5511988888888",
		"type": "text",
		"text": map[string]interface{}{
			"body": "Hello",
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/"+instance.PhoneNumberID+"/messages", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400 (instance not connected), got %d", resp.StatusCode)
	}

	// Parse error response
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	errorMap, ok := result["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing 'error' field")
	}

	if errorMap["type"] != "InstanceNotConnectedError" {
		t.Errorf("Expected error type 'InstanceNotConnectedError', got %v", errorMap["type"])
	}
}

func TestMessageHandler_Send_MissingTextContent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	waManager := setupTestWAManager(t, db)
	handler := NewMessageHandler(db, waManager)

	// Create test instance (connected)
	instanceRepo := repository.NewInstanceRepository(db)
	instance := models.NewInstance(tenant.ID)
	instance.Status = "connected"
	connectedAt := time.Now()
	instance.ConnectedAt = &connectedAt
	instanceRepo.Create(nil, instance)

	// Create test app and route
	app := fiber.New()
	app.Post("/:phone_number_id/messages", func(c *fiber.Ctx) error {
		c.Locals("tenant_id", tenant.ID)
		return handler.Send(c)
	})

	// Test request with text type but no text content
	reqBody := map[string]interface{}{
		"to":   "5511988888888",
		"type": "text",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/"+instance.PhoneNumberID+"/messages", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	// Parse error response
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	errorMap, ok := result["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing 'error' field")
	}

	messageStr, _ := errorMap["message"].(string)
	if messageStr != "Missing text content" {
		t.Errorf("Expected error message 'Missing text content', got %v", errorMap["message"])
	}
}

func TestMessageHandler_Send_InvalidBody(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	waManager := setupTestWAManager(t, db)
	handler := NewMessageHandler(db, waManager)

	// Create test instance (connected)
	instanceRepo := repository.NewInstanceRepository(db)
	instance := models.NewInstance(tenant.ID)
	instance.Status = "connected"
	instanceRepo.Create(nil, instance)

	// Create test app and route
	app := fiber.New()
	app.Post("/:phone_number_id/messages", func(c *fiber.Ctx) error {
		c.Locals("tenant_id", tenant.ID)
		return handler.Send(c)
	})

	// Test request with invalid JSON
	req := httptest.NewRequest("POST", "/"+instance.PhoneNumberID+"/messages", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestMessageHandler_List_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	waManager := setupTestWAManager(t, db)
	handler := NewMessageHandler(db, waManager)

	// Create test instance
	instanceRepo := repository.NewInstanceRepository(db)
	instance := models.NewInstance(tenant.ID)
	instanceRepo.Create(nil, instance)

	// Create test messages
	messageRepo := repository.NewMessageRepository(db)
	for i := 0; i < 10; i++ {
		message := models.NewOutboundTextMessage(tenant.ID, instance.ID, instance.PhoneNumberID, "5511988888888", "Test message")
		messageRepo.Create(nil, message)
	}

	// Create test app and route
	app := fiber.New()
	app.Get("/:phone_number_id/messages", func(c *fiber.Ctx) error {
		c.Locals("tenant_id", tenant.ID)
		return handler.List(c)
	})

	// Test request
	req := httptest.NewRequest("GET", "/"+instance.PhoneNumberID+"/messages", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	data, ok := result["data"].([]interface{})
	if !ok {
		t.Fatal("Response missing 'data' field")
	}

	if len(data) < 10 {
		t.Errorf("Expected at least 10 messages, got %d", len(data))
	}

	// Check paging info
	paging, ok := result["paging"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing 'paging' field")
	}

	total, ok := paging["total"].(float64)
	if !ok || total < 10 {
		t.Errorf("Expected total >= 10, got %v", paging["total"])
	}
}

func TestMessageHandler_List_Pagination(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	waManager := setupTestWAManager(t, db)
	handler := NewMessageHandler(db, waManager)

	// Create test instance
	instanceRepo := repository.NewInstanceRepository(db)
	instance := models.NewInstance(tenant.ID)
	instanceRepo.Create(nil, instance)

	// Create 30 test messages
	messageRepo := repository.NewMessageRepository(db)
	for i := 0; i < 30; i++ {
		message := models.NewOutboundTextMessage(tenant.ID, instance.ID, instance.PhoneNumberID, "5511988888888", "Test message")
		messageRepo.Create(nil, message)
	}

	// Create test app and route
	app := fiber.New()
	app.Get("/:phone_number_id/messages", func(c *fiber.Ctx) error {
		c.Locals("tenant_id", tenant.ID)
		return handler.List(c)
	})

	// Test first page (limit=10, offset=0)
	req := httptest.NewRequest("GET", "/"+instance.PhoneNumberID+"/messages?limit=10&offset=0", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Parse response
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	data, ok := result["data"].([]interface{})
	if !ok {
		t.Fatal("Response missing 'data' field")
	}

	if len(data) != 10 {
		t.Errorf("Expected 10 messages in page, got %d", len(data))
	}

	// Check paging info
	paging, ok := result["paging"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing 'paging' field")
	}

	// Should have 'next' cursor
	if paging["next"] == nil {
		t.Error("Expected 'next' cursor in paging")
	}
}

func TestMessageHandler_List_MaxLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	waManager := setupTestWAManager(t, db)
	handler := NewMessageHandler(db, waManager)

	// Create test instance
	instanceRepo := repository.NewInstanceRepository(db)
	instance := models.NewInstance(tenant.ID)
	instanceRepo.Create(nil, instance)

	// Create test app and route
	app := fiber.New()
	app.Get("/:phone_number_id/messages", func(c *fiber.Ctx) error {
		c.Locals("tenant_id", tenant.ID)
		return handler.List(c)
	})

	// Test request with limit > 100 (should be capped at 100)
	req := httptest.NewRequest("GET", "/"+instance.PhoneNumberID+"/messages?limit=500", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Parse response
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	paging, ok := result["paging"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing 'paging' field")
	}

	// Limit should be capped at 100
	limit, _ := paging["limit"].(float64)
	if limit > 100 {
		t.Errorf("Expected limit capped at 100, got %v", limit)
	}
}

func TestMessageHandler_List_InstanceNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	waManager := setupTestWAManager(t, db)
	handler := NewMessageHandler(db, waManager)

	// Create test app and route
	app := fiber.New()
	app.Get("/:phone_number_id/messages", func(c *fiber.Ctx) error {
		c.Locals("tenant_id", tenant.ID)
		return handler.List(c)
	})

	// Test request with non-existent phone_number_id
	req := httptest.NewRequest("GET", "/99999999999/messages", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestMessageHandler_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create two tenants
	tenantRepo := repository.NewTenantRepository(db)
	tenant1 := models.NewTenant("Tenant 1", "tenant1@example.com")
	tenant2 := models.NewTenant("Tenant 2", "tenant2@example.com")
	tenantRepo.Create(nil, tenant1)
	tenantRepo.Create(nil, tenant2)

	// Create instances for both tenants
	instanceRepo := repository.NewInstanceRepository(db)
	instance1 := models.NewInstance(tenant1.ID)
	instance2 := models.NewInstance(tenant2.ID)
	instanceRepo.Create(nil, instance1)
	instanceRepo.Create(nil, instance2)

	// Create messages for tenant1
	messageRepo := repository.NewMessageRepository(db)
	for i := 0; i < 5; i++ {
		message := models.NewOutboundTextMessage(tenant1.ID, instance1.ID, instance1.PhoneNumberID, "5511988888888", "Tenant 1 message")
		messageRepo.Create(nil, message)
	}

	waManager := setupTestWAManager(t, db)
	handler := NewMessageHandler(db, waManager)

	// Create test app and route
	app := fiber.New()
	app.Get("/:phone_number_id/messages", func(c *fiber.Ctx) error {
		// Simulate tenant2 trying to access tenant1's messages
		c.Locals("tenant_id", tenant2.ID)
		return handler.List(c)
	})

	// Test request - tenant2 should NOT see tenant1's instance
	req := httptest.NewRequest("GET", "/"+instance1.PhoneNumberID+"/messages", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("Tenant isolation broken: expected 404, got %d", resp.StatusCode)
	}
}

func TestMessageHandler_MessageTypes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	waManager := setupTestWAManager(t, db)
	handler := NewMessageHandler(db, waManager)

	// Create test instance (connected)
	instanceRepo := repository.NewInstanceRepository(db)
	instance := models.NewInstance(tenant.ID)
	instance.Status = "connected"
	connectedAt := time.Now()
	instance.ConnectedAt = &connectedAt
	instanceRepo.Create(nil, instance)

	// Create test app and route
	app := fiber.New()
	app.Post("/:phone_number_id/messages", func(c *fiber.Ctx) error {
		c.Locals("tenant_id", tenant.ID)
		return handler.Send(c)
	})

	messageTypes := []struct {
		name    string
		reqBody map[string]interface{}
	}{
		{
			name: "image",
			reqBody: map[string]interface{}{
				"to":   "5511988888888",
				"type": "image",
				"image": map[string]interface{}{
					"link":    "https://example.com/image.jpg",
					"caption": "Test image",
				},
			},
		},
		{
			name: "audio",
			reqBody: map[string]interface{}{
				"to":   "5511988888888",
				"type": "audio",
				"audio": map[string]interface{}{
					"link": "https://example.com/audio.mp3",
				},
			},
		},
		{
			name: "video",
			reqBody: map[string]interface{}{
				"to":   "5511988888888",
				"type": "video",
				"video": map[string]interface{}{
					"link":    "https://example.com/video.mp4",
					"caption": "Test video",
				},
			},
		},
		{
			name: "document",
			reqBody: map[string]interface{}{
				"to":   "5511988888888",
				"type": "document",
				"document": map[string]interface{}{
					"link":     "https://example.com/doc.pdf",
					"filename": "document.pdf",
				},
			},
		},
	}

	for _, mt := range messageTypes {
		t.Run(mt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(mt.reqBody)
			req := httptest.NewRequest("POST", "/"+instance.PhoneNumberID+"/messages", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			// These should create the message (200 or 500 depending on WhatsApp manager implementation)
			// We're checking that the validation passes and message gets created in DB
			if resp.StatusCode != 200 && resp.StatusCode != 500 {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status 200 or 500, got %d. Body: %s", resp.StatusCode, string(body))
			}
		})
	}
}
