package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/models"
)

// Helper function to create test database
func setupTestDB(t *testing.T) *Database {
	// Use environment variable for test database
	// In production tests, this should use Testcontainers
	dbURL := "postgres://postgres:postgres@localhost:5432/whatsmeow_test?sslmode=disable"
	db, err := NewDatabase(dbURL)
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}
	return db
}

func TestInstanceRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewInstanceRepository(db)
	ctx := context.Background()

	tenantID := uuid.New().String()
	instance := &models.Instance{
		TenantID:      tenantID,
		ID:            uuid.New().String(),
		PhoneNumberID: "5511999999999",
		Name:          "Test Instance",
		Status:        "disconnected",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := repo.Create(ctx, instance)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify instance was created
	found, err := repo.FindByID(ctx, tenantID, instance.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.ID != instance.ID {
		t.Errorf("Instance ID mismatch: got %s, want %s", found.ID, instance.ID)
	}
	if found.PhoneNumberID != instance.PhoneNumberID {
		t.Errorf("PhoneNumberID mismatch: got %s, want %s", found.PhoneNumberID, instance.PhoneNumberID)
	}
}

func TestInstanceRepository_FindByPhoneNumberID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewInstanceRepository(db)
	ctx := context.Background()

	tenantID := uuid.New().String()
	phoneNumberID := "5511988888888"
	instance := &models.Instance{
		TenantID:      tenantID,
		ID:            uuid.New().String(),
		PhoneNumberID: phoneNumberID,
		Name:          "Test Instance 2",
		Status:        "connected",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := repo.Create(ctx, instance)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Find by phone number ID
	found, err := repo.FindByPhoneNumberID(ctx, tenantID, phoneNumberID)
	if err != nil {
		t.Fatalf("FindByPhoneNumberID() error = %v", err)
	}

	if found.ID != instance.ID {
		t.Errorf("Instance ID mismatch: got %s, want %s", found.ID, instance.ID)
	}
	if found.PhoneNumberID != phoneNumberID {
		t.Errorf("PhoneNumberID mismatch: got %s, want %s", found.PhoneNumberID, phoneNumberID)
	}
}

func TestInstanceRepository_FindByTenantID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewInstanceRepository(db)
	ctx := context.Background()

	tenantID := uuid.New().String()

	// Create multiple instances for the same tenant
	instances := []*models.Instance{
		{
			TenantID:      tenantID,
			ID:            uuid.New().String(),
			PhoneNumberID: "5511977777777",
			Name:          "Instance 1",
			Status:        "connected",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		{
			TenantID:      tenantID,
			ID:            uuid.New().String(),
			PhoneNumberID: "5511966666666",
			Name:          "Instance 2",
			Status:        "disconnected",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
	}

	for _, instance := range instances {
		if err := repo.Create(ctx, instance); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Find all instances for tenant
	found, err := repo.FindByTenantID(ctx, tenantID)
	if err != nil {
		t.Fatalf("FindByTenantID() error = %v", err)
	}

	if len(found) < 2 {
		t.Errorf("FindByTenantID() returned %d instances, expected at least 2", len(found))
	}
}

func TestInstanceRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewInstanceRepository(db)
	ctx := context.Background()

	tenantID := uuid.New().String()
	instance := &models.Instance{
		TenantID:      tenantID,
		ID:            uuid.New().String(),
		PhoneNumberID: "5511955555555",
		Name:          "Original Name",
		Status:        "disconnected",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := repo.Create(ctx, instance)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update instance
	instance.Name = "Updated Name"
	instance.Status = "connected"
	instance.UpdatedAt = time.Now()

	err = repo.Update(ctx, instance)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	found, err := repo.FindByID(ctx, tenantID, instance.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Name != "Updated Name" {
		t.Errorf("Name not updated: got %s, want Updated Name", found.Name)
	}
	if found.Status != "connected" {
		t.Errorf("Status not updated: got %s, want connected", found.Status)
	}
}

func TestInstanceRepository_UpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewInstanceRepository(db)
	ctx := context.Background()

	tenantID := uuid.New().String()
	instance := &models.Instance{
		TenantID:      tenantID,
		ID:            uuid.New().String(),
		PhoneNumberID: "5511944444444",
		Name:          "Status Test",
		Status:        "disconnected",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := repo.Create(ctx, instance)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update status only
	err = repo.UpdateStatus(ctx, tenantID, instance.ID, "connected")
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	// Verify status update
	found, err := repo.FindByID(ctx, tenantID, instance.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Status != "connected" {
		t.Errorf("Status not updated: got %s, want connected", found.Status)
	}
}

func TestInstanceRepository_CountByTenantID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewInstanceRepository(db)
	ctx := context.Background()

	tenantID := uuid.New().String()

	// Create 3 instances
	for i := 0; i < 3; i++ {
		instance := &models.Instance{
			TenantID:      tenantID,
			ID:            uuid.New().String(),
			PhoneNumberID: uuid.New().String(),
			Name:          "Count Test",
			Status:        "disconnected",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		if err := repo.Create(ctx, instance); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	count, err := repo.CountByTenantID(ctx, tenantID)
	if err != nil {
		t.Fatalf("CountByTenantID() error = %v", err)
	}

	if count < 3 {
		t.Errorf("CountByTenantID() = %d, want at least 3", count)
	}
}

func TestInstanceRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewInstanceRepository(db)
	ctx := context.Background()

	tenantID := uuid.New().String()
	instance := &models.Instance{
		TenantID:      tenantID,
		ID:            uuid.New().String(),
		PhoneNumberID: "5511933333333",
		Name:          "Delete Test",
		Status:        "disconnected",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := repo.Create(ctx, instance)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Delete instance
	err = repo.Delete(ctx, tenantID, instance.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deletion
	_, err = repo.FindByID(ctx, tenantID, instance.ID)
	if err != ErrNotFound {
		t.Errorf("FindByID() after delete should return ErrNotFound, got %v", err)
	}
}

func TestInstanceRepository_UpdateQRCode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewInstanceRepository(db)
	ctx := context.Background()

	tenantID := uuid.New().String()
	instance := &models.Instance{
		TenantID:      tenantID,
		ID:            uuid.New().String(),
		PhoneNumberID: "5511922222222",
		Name:          "QR Test",
		Status:        "disconnected",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := repo.Create(ctx, instance)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update QR code
	qrCode := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	expiresAt := time.Now().Add(90 * time.Second)

	err = repo.UpdateQRCode(ctx, tenantID, instance.ID, qrCode, expiresAt)
	if err != nil {
		t.Fatalf("UpdateQRCode() error = %v", err)
	}

	// Verify QR code update
	found, err := repo.FindByID(ctx, tenantID, instance.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.QRCode == nil || *found.QRCode != qrCode {
		t.Error("QR code not updated correctly")
	}
	if found.QRCodeExpiresAt == nil {
		t.Error("QR code expiration not set")
	}
}

func TestInstanceRepository_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewInstanceRepository(db)
	ctx := context.Background()

	// Create instances for two different tenants with same phone number ID
	tenant1ID := uuid.New().String()
	tenant2ID := uuid.New().String()
	phoneNumberID := "5511911111111"

	instance1 := &models.Instance{
		TenantID:      tenant1ID,
		ID:            uuid.New().String(),
		PhoneNumberID: phoneNumberID,
		Name:          "Tenant 1 Instance",
		Status:        "connected",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	instance2 := &models.Instance{
		TenantID:      tenant2ID,
		ID:            uuid.New().String(),
		PhoneNumberID: phoneNumberID,
		Name:          "Tenant 2 Instance",
		Status:        "disconnected",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Both should be created without conflict
	if err := repo.Create(ctx, instance1); err != nil {
		t.Fatalf("Create() instance1 error = %v", err)
	}
	if err := repo.Create(ctx, instance2); err != nil {
		t.Fatalf("Create() instance2 error = %v", err)
	}

	// Tenant 1 should not see Tenant 2's instance
	found1, err := repo.FindByPhoneNumberID(ctx, tenant1ID, phoneNumberID)
	if err != nil {
		t.Fatalf("FindByPhoneNumberID() tenant1 error = %v", err)
	}
	if found1.ID != instance1.ID {
		t.Errorf("Tenant isolation broken: tenant1 got wrong instance")
	}

	// Tenant 2 should not see Tenant 1's instance
	found2, err := repo.FindByPhoneNumberID(ctx, tenant2ID, phoneNumberID)
	if err != nil {
		t.Fatalf("FindByPhoneNumberID() tenant2 error = %v", err)
	}
	if found2.ID != instance2.ID {
		t.Errorf("Tenant isolation broken: tenant2 got wrong instance")
	}
}
