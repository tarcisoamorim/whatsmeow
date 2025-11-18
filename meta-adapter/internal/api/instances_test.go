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
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/whatsapp"
)

// Helper function to setup test database
func setupTestDB(t *testing.T) *repository.Database {
	dbURL := "postgres://postgres:postgres@localhost:5432/whatsmeow_test?sslmode=disable"
	db, err := repository.NewDatabase(dbURL)
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}
	return db
}

// Helper function to setup test tenant
func setupTestTenant(t *testing.T, db *repository.Database) *models.Tenant {
	tenantRepo := repository.NewTenantRepository(db)
	tenant := models.NewTenant("Test Tenant", "test@example.com")
	tenant.Tier = "business" // Allow multiple instances
	tenant.MaxInstances = 5

	if err := tenantRepo.Create(nil, tenant); err != nil {
		t.Fatalf("Failed to create test tenant: %v", err)
	}
	return tenant
}

// Helper function to create test WhatsApp manager
func setupTestWAManager(t *testing.T, db *repository.Database) *whatsapp.Manager {
	// For testing, we'll skip WhatsApp manager initialization
	// In real tests, use a mock
	return nil
}

func TestInstanceHandler_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	waManager := setupTestWAManager(t, db)
	handler := NewInstanceHandler(db, waManager)

	// Create test instances
	instanceRepo := repository.NewInstanceRepository(db)
	for i := 0; i < 3; i++ {
		instance := models.NewInstance(tenant.ID)
		if err := instanceRepo.Create(nil, instance); err != nil {
			t.Fatalf("Failed to create test instance: %v", err)
		}
	}

	// Create test app and route
	app := fiber.New()
	app.Get("/instances", func(c *fiber.Ctx) error {
		c.Locals("tenant_id", tenant.ID)
		return handler.List(c)
	})

	// Test request
	req := httptest.NewRequest("GET", "/instances", nil)
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

	if len(data) < 3 {
		t.Errorf("Expected at least 3 instances, got %d", len(data))
	}
}

func TestInstanceHandler_Create_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	waManager := setupTestWAManager(t, db)
	handler := NewInstanceHandler(db, waManager)

	// Create test app and route
	app := fiber.New()
	app.Post("/instances", func(c *fiber.Ctx) error {
		c.Locals("tenant_id", tenant.ID)
		return handler.Create(c)
	})

	// Test request
	reqBody := map[string]interface{}{
		"display_name":    "Test Instance",
		"webhook_url":     "https://example.com/webhook",
		"webhook_events":  []string{"message.received", "message.sent"},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/instances", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 201, got %d. Body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["id"] == nil {
		t.Error("Response missing 'id' field")
	}
	if result["phone_number_id"] == nil {
		t.Error("Response missing 'phone_number_id' field")
	}
	if result["status"] != "disconnected" {
		t.Errorf("Expected status 'disconnected', got %v", result["status"])
	}
}

func TestInstanceHandler_Create_QuotaExceeded(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	tenant.MaxInstances = 1 // Limit to 1 instance

	// Update tenant in database
	tenantRepo := repository.NewTenantRepository(db)
	tenantRepo.Update(nil, tenant)

	// Create one instance to reach limit
	instanceRepo := repository.NewInstanceRepository(db)
	instance := models.NewInstance(tenant.ID)
	instanceRepo.Create(nil, instance)

	waManager := setupTestWAManager(t, db)
	handler := NewInstanceHandler(db, waManager)

	// Create test app and route
	app := fiber.New()
	app.Post("/instances", func(c *fiber.Ctx) error {
		c.Locals("tenant_id", tenant.ID)
		return handler.Create(c)
	})

	// Test request (should fail due to quota)
	reqBody := map[string]interface{}{
		"display_name": "Test Instance 2",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/instances", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 403 {
		t.Errorf("Expected status 403 (quota exceeded), got %d", resp.StatusCode)
	}

	// Parse error response
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	errorMap, ok := result["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing 'error' field")
	}

	if errorMap["type"] != "QuotaExceededError" {
		t.Errorf("Expected error type 'QuotaExceededError', got %v", errorMap["type"])
	}
}

func TestInstanceHandler_Create_InvalidBody(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	waManager := setupTestWAManager(t, db)
	handler := NewInstanceHandler(db, waManager)

	// Create test app and route
	app := fiber.New()
	app.Post("/instances", func(c *fiber.Ctx) error {
		c.Locals("tenant_id", tenant.ID)
		return handler.Create(c)
	})

	// Test request with invalid JSON
	req := httptest.NewRequest("POST", "/instances", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestInstanceHandler_Get_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	waManager := setupTestWAManager(t, db)
	handler := NewInstanceHandler(db, waManager)

	// Create test instance
	instanceRepo := repository.NewInstanceRepository(db)
	instance := models.NewInstance(tenant.ID)
	displayName := "Test Instance"
	instance.DisplayName = &displayName
	if err := instanceRepo.Create(nil, instance); err != nil {
		t.Fatalf("Failed to create test instance: %v", err)
	}

	// Create test app and route
	app := fiber.New()
	app.Get("/instances/:phone_number_id", func(c *fiber.Ctx) error {
		c.Locals("tenant_id", tenant.ID)
		return handler.Get(c)
	})

	// Test request
	req := httptest.NewRequest("GET", "/instances/"+instance.PhoneNumberID, nil)
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

	if result["id"] != instance.ID {
		t.Errorf("Expected ID %s, got %v", instance.ID, result["id"])
	}
	if result["phone_number_id"] != instance.PhoneNumberID {
		t.Errorf("Expected phone_number_id %s, got %v", instance.PhoneNumberID, result["phone_number_id"])
	}
}

func TestInstanceHandler_Get_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	waManager := setupTestWAManager(t, db)
	handler := NewInstanceHandler(db, waManager)

	// Create test app and route
	app := fiber.New()
	app.Get("/instances/:phone_number_id", func(c *fiber.Ctx) error {
		c.Locals("tenant_id", tenant.ID)
		return handler.Get(c)
	})

	// Test request with non-existent phone_number_id
	req := httptest.NewRequest("GET", "/instances/99999999999", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}

	// Parse error response
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	errorMap, ok := result["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing 'error' field")
	}

	if errorMap["type"] != "NotFoundError" {
		t.Errorf("Expected error type 'NotFoundError', got %v", errorMap["type"])
	}
}

func TestInstanceHandler_GetQRCode_AlreadyConnected(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tenant := setupTestTenant(t, db)
	waManager := setupTestWAManager(t, db)
	handler := NewInstanceHandler(db, waManager)

	// Create test instance that's already connected
	instanceRepo := repository.NewInstanceRepository(db)
	instance := models.NewInstance(tenant.ID)
	instance.Status = "connected"
	connectedAt := time.Now()
	instance.ConnectedAt = &connectedAt
	if err := instanceRepo.Create(nil, instance); err != nil {
		t.Fatalf("Failed to create test instance: %v", err)
	}

	// Create test app and route
	app := fiber.New()
	app.Get("/instances/:phone_number_id/qrcode", func(c *fiber.Ctx) error {
		c.Locals("tenant_id", tenant.ID)
		return handler.GetQRCode(c)
	})

	// Test request
	req := httptest.NewRequest("GET", "/instances/"+instance.PhoneNumberID+"/qrcode", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400 (already connected), got %d", resp.StatusCode)
	}

	// Parse error response
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	errorMap, ok := result["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing 'error' field")
	}

	if errorMap["message"] != "Instance is already connected" {
		t.Errorf("Expected error message about already connected, got %v", errorMap["message"])
	}
}

func TestInstanceHandler_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create two tenants
	tenantRepo := repository.NewTenantRepository(db)
	tenant1 := models.NewTenant("Tenant 1", "tenant1@example.com")
	tenant2 := models.NewTenant("Tenant 2", "tenant2@example.com")
	tenantRepo.Create(nil, tenant1)
	tenantRepo.Create(nil, tenant2)

	// Create instance for tenant1
	instanceRepo := repository.NewInstanceRepository(db)
	instance1 := models.NewInstance(tenant1.ID)
	instanceRepo.Create(nil, instance1)

	waManager := setupTestWAManager(t, db)
	handler := NewInstanceHandler(db, waManager)

	// Create test app and route
	app := fiber.New()
	app.Get("/instances/:phone_number_id", func(c *fiber.Ctx) error {
		// Simulate tenant2 trying to access tenant1's instance
		c.Locals("tenant_id", tenant2.ID)
		return handler.Get(c)
	})

	// Test request - tenant2 should NOT see tenant1's instance
	req := httptest.NewRequest("GET", "/instances/"+instance1.PhoneNumberID, nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("Tenant isolation broken: expected 404, got %d", resp.StatusCode)
	}
}

func TestInstanceHandler_Create_TenantNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	waManager := setupTestWAManager(t, db)
	handler := NewInstanceHandler(db, waManager)

	// Create test app and route
	app := fiber.New()
	app.Post("/instances", func(c *fiber.Ctx) error {
		// Use non-existent tenant ID
		c.Locals("tenant_id", uuid.New().String())
		return handler.Create(c)
	})

	// Test request
	reqBody := map[string]interface{}{
		"display_name": "Test Instance",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/instances", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404 (tenant not found), got %d", resp.StatusCode)
	}
}
