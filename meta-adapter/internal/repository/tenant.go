package repository

import (
	"context"
	"database/sql"
	"errors"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/models"
)

var ErrNotFound = errors.New("not found")

type TenantRepository struct {
	db *Database
}

func NewTenantRepository(db *Database) *TenantRepository {
	return &TenantRepository{db: db}
}

func (r *TenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	query := `
		INSERT INTO tenants (
			id, name, email, tier, max_instances, max_messages_per_day, 
			max_api_requests_per_minute, max_webhook_retries, subscription_status, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, query,
		tenant.ID, tenant.Name, tenant.Email, tenant.Tier, tenant.MaxInstances,
		tenant.MaxMessagesPerDay, tenant.MaxAPIRequestsPerMinute, tenant.MaxWebhookRetries,
		tenant.SubscriptionStatus, tenant.Status,
	)
	return err
}

func (r *TenantRepository) FindByID(ctx context.Context, id string) (*models.Tenant, error) {
	var tenant models.Tenant
	query := `SELECT * FROM tenants WHERE id = $1 AND deleted_at IS NULL`
	
	err := r.db.GetContext(ctx, &tenant, query, id)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *TenantRepository) FindByEmail(ctx context.Context, email string) (*models.Tenant, error) {
	var tenant models.Tenant
	query := `SELECT * FROM tenants WHERE email = $1 AND deleted_at IS NULL`
	
	err := r.db.GetContext(ctx, &tenant, query, email)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}
