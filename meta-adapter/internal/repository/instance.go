package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/models"
)

type InstanceRepository struct {
	db *Database
}

func NewInstanceRepository(db *Database) *InstanceRepository {
	return &InstanceRepository{db: db}
}

// Create inserts a new instance into the database
func (r *InstanceRepository) Create(ctx context.Context, instance *models.Instance) error {
	query := `
		INSERT INTO instances (
			id, tenant_id, phone_number_id, display_name, status,
			webhook_url, webhook_events, webhook_verify_token,
			qr_code, qr_expires_at, pair_code, pair_expires_at,
			connected_at, disconnected_at, last_seen_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)`

	_, err := r.db.ExecContext(
		ctx, query,
		instance.ID,
		instance.TenantID,
		instance.PhoneNumberID,
		instance.DisplayName,
		instance.Status,
		instance.WebhookURL,
		instance.WebhookEvents,
		instance.WebhookVerifyToken,
		instance.QRCode,
		instance.QRExpiresAt,
		instance.PairCode,
		instance.PairExpiresAt,
		instance.ConnectedAt,
		instance.DisconnectedAt,
		instance.LastSeenAt,
		instance.CreatedAt,
		instance.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	return nil
}

// FindByID retrieves an instance by tenant_id and instance_id
func (r *InstanceRepository) FindByID(ctx context.Context, tenantID, instanceID string) (*models.Instance, error) {
	var instance models.Instance

	query := `
		SELECT
			id, tenant_id, phone_number_id, display_name, status,
			webhook_url, webhook_events, webhook_verify_token,
			qr_code, qr_expires_at, pair_code, pair_expires_at,
			connected_at, disconnected_at, last_seen_at,
			created_at, updated_at
		FROM instances
		WHERE tenant_id = $1 AND id = $2
	`

	err := r.db.GetContext(ctx, &instance, query, tenantID, instanceID)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find instance: %w", err)
	}

	return &instance, nil
}

// FindByPhoneNumberID retrieves an instance by tenant_id and phone_number_id
func (r *InstanceRepository) FindByPhoneNumberID(ctx context.Context, tenantID, phoneNumberID string) (*models.Instance, error) {
	var instance models.Instance

	query := `
		SELECT
			id, tenant_id, phone_number_id, display_name, status,
			webhook_url, webhook_events, webhook_verify_token,
			qr_code, qr_expires_at, pair_code, pair_expires_at,
			connected_at, disconnected_at, last_seen_at,
			created_at, updated_at
		FROM instances
		WHERE tenant_id = $1 AND phone_number_id = $2
	`

	err := r.db.GetContext(ctx, &instance, query, tenantID, phoneNumberID)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find instance by phone number: %w", err)
	}

	return &instance, nil
}

// FindByTenantID retrieves all instances for a tenant
func (r *InstanceRepository) FindByTenantID(ctx context.Context, tenantID string) ([]*models.Instance, error) {
	var instances []*models.Instance

	query := `
		SELECT
			id, tenant_id, phone_number_id, display_name, status,
			webhook_url, webhook_events, webhook_verify_token,
			qr_code, qr_expires_at, pair_code, pair_expires_at,
			connected_at, disconnected_at, last_seen_at,
			created_at, updated_at
		FROM instances
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	err := r.db.SelectContext(ctx, &instances, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	return instances, nil
}

// Update updates an existing instance
func (r *InstanceRepository) Update(ctx context.Context, instance *models.Instance) error {
	instance.UpdatedAt = time.Now()

	query := `
		UPDATE instances SET
			phone_number_id = $1,
			display_name = $2,
			status = $3,
			webhook_url = $4,
			webhook_events = $5,
			webhook_verify_token = $6,
			qr_code = $7,
			qr_expires_at = $8,
			pair_code = $9,
			pair_expires_at = $10,
			connected_at = $11,
			disconnected_at = $12,
			last_seen_at = $13,
			updated_at = $14
		WHERE tenant_id = $15 AND id = $16
	`

	result, err := r.db.ExecContext(
		ctx, query,
		instance.PhoneNumberID,
		instance.DisplayName,
		instance.Status,
		instance.WebhookURL,
		instance.WebhookEvents,
		instance.WebhookVerifyToken,
		instance.QRCode,
		instance.QRExpiresAt,
		instance.PairCode,
		instance.PairExpiresAt,
		instance.ConnectedAt,
		instance.DisconnectedAt,
		instance.LastSeenAt,
		instance.UpdatedAt,
		instance.TenantID,
		instance.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// Delete soft-deletes an instance (updates status to deleted)
func (r *InstanceRepository) Delete(ctx context.Context, tenantID, instanceID string) error {
	query := `
		UPDATE instances
		SET status = 'deleted', updated_at = $1
		WHERE tenant_id = $2 AND id = $3
	`

	result, err := r.db.ExecContext(ctx, query, time.Now(), tenantID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// CountByTenantID returns the number of active instances for a tenant
func (r *InstanceRepository) CountByTenantID(ctx context.Context, tenantID string) (int, error) {
	var count int

	query := `
		SELECT COUNT(*)
		FROM instances
		WHERE tenant_id = $1 AND status != 'deleted'
	`

	err := r.db.GetContext(ctx, &count, query, tenantID)
	if err != nil {
		return 0, fmt.Errorf("failed to count instances: %w", err)
	}

	return count, nil
}

// UpdateStatus updates only the status field of an instance
func (r *InstanceRepository) UpdateStatus(ctx context.Context, tenantID, instanceID, status string) error {
	query := `
		UPDATE instances
		SET status = $1, updated_at = $2
		WHERE tenant_id = $3 AND id = $4
	`

	result, err := r.db.ExecContext(ctx, query, status, time.Now(), tenantID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to update instance status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// UpdateQRCode updates the QR code and expiration for an instance
func (r *InstanceRepository) UpdateQRCode(ctx context.Context, tenantID, instanceID, qrCode string, expiresAt time.Time) error {
	query := `
		UPDATE instances
		SET qr_code = $1, qr_expires_at = $2, updated_at = $3
		WHERE tenant_id = $4 AND id = $5
	`

	result, err := r.db.ExecContext(ctx, query, qrCode, expiresAt, time.Now(), tenantID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to update QR code: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// UpdateConnectionStatus updates connection-related timestamps
func (r *InstanceRepository) UpdateConnectionStatus(ctx context.Context, tenantID, instanceID string, connected bool) error {
	var query string
	now := time.Now()

	if connected {
		query = `
			UPDATE instances
			SET status = 'connected', connected_at = $1, last_seen_at = $2, updated_at = $3
			WHERE tenant_id = $4 AND id = $5
		`
	} else {
		query = `
			UPDATE instances
			SET status = 'disconnected', disconnected_at = $1, updated_at = $2
			WHERE tenant_id = $3 AND id = $4
		`
	}

	var result sql.Result
	var err error

	if connected {
		result, err = r.db.ExecContext(ctx, query, now, now, now, tenantID, instanceID)
	} else {
		result, err = r.db.ExecContext(ctx, query, now, now, tenantID, instanceID)
	}

	if err != nil {
		return fmt.Errorf("failed to update connection status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}
