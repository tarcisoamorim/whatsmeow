package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/models"
)

type MessageRepository struct {
	db *Database
}

func NewMessageRepository(db *Database) *MessageRepository {
	return &MessageRepository{db: db}
}

// Create inserts a new message into the database
func (r *MessageRepository) Create(ctx context.Context, message *models.Message) error {
	// Convert content map to JSON
	contentJSON, err := json.Marshal(message.Content)
	if err != nil {
		return fmt.Errorf("failed to marshal content: %w", err)
	}

	query := `
		INSERT INTO messages (
			tenant_id, instance_id, id, message_id, direction, type, status,
			from_number, to_number, content, timestamp,
			sent_at, delivered_at, read_at, error,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)`

	_, err = r.db.ExecContext(
		ctx, query,
		message.TenantID,
		message.InstanceID,
		message.ID,
		message.MessageID,
		message.Direction,
		message.Type,
		message.Status,
		message.From,
		message.To,
		contentJSON,
		message.Timestamp,
		message.SentAt,
		message.DeliveredAt,
		message.ReadAt,
		message.Error,
		message.CreatedAt,
		message.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	return nil
}

// FindByID retrieves a message by its ID
func (r *MessageRepository) FindByID(ctx context.Context, tenantID, instanceID, messageID string) (*models.Message, error) {
	query := `
		SELECT
			tenant_id, instance_id, id, message_id, direction, type, status,
			from_number, to_number, content, timestamp,
			sent_at, delivered_at, read_at, error,
			created_at, updated_at
		FROM messages
		WHERE tenant_id = $1 AND instance_id = $2 AND id = $3
	`

	row := r.db.QueryRowContext(ctx, query, tenantID, instanceID, messageID)

	var message models.Message
	var contentJSON []byte

	err := row.Scan(
		&message.TenantID,
		&message.InstanceID,
		&message.ID,
		&message.MessageID,
		&message.Direction,
		&message.Type,
		&message.Status,
		&message.From,
		&message.To,
		&contentJSON,
		&message.Timestamp,
		&message.SentAt,
		&message.DeliveredAt,
		&message.ReadAt,
		&message.Error,
		&message.CreatedAt,
		&message.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find message: %w", err)
	}

	// Unmarshal JSON content
	if err := json.Unmarshal(contentJSON, &message.Content); err != nil {
		return nil, fmt.Errorf("failed to unmarshal content: %w", err)
	}

	return &message, nil
}

// FindByMessageID retrieves a message by its WhatsApp message ID (wamid)
func (r *MessageRepository) FindByMessageID(ctx context.Context, tenantID, instanceID, waMessageID string) (*models.Message, error) {
	query := `
		SELECT
			tenant_id, instance_id, id, message_id, direction, type, status,
			from_number, to_number, content, timestamp,
			sent_at, delivered_at, read_at, error,
			created_at, updated_at
		FROM messages
		WHERE tenant_id = $1 AND instance_id = $2 AND message_id = $3
	`

	row := r.db.QueryRowContext(ctx, query, tenantID, instanceID, waMessageID)

	var message models.Message
	var contentJSON []byte

	err := row.Scan(
		&message.TenantID,
		&message.InstanceID,
		&message.ID,
		&message.MessageID,
		&message.Direction,
		&message.Type,
		&message.Status,
		&message.From,
		&message.To,
		&contentJSON,
		&message.Timestamp,
		&message.SentAt,
		&message.DeliveredAt,
		&message.ReadAt,
		&message.Error,
		&message.CreatedAt,
		&message.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find message by message_id: %w", err)
	}

	// Unmarshal JSON content
	if err := json.Unmarshal(contentJSON, &message.Content); err != nil {
		return nil, fmt.Errorf("failed to unmarshal content: %w", err)
	}

	return &message, nil
}

// ListByInstance retrieves messages for an instance with pagination
func (r *MessageRepository) ListByInstance(ctx context.Context, tenantID, instanceID string, limit, offset int) ([]*models.Message, error) {
	query := `
		SELECT
			tenant_id, instance_id, id, message_id, direction, type, status,
			from_number, to_number, content, timestamp,
			sent_at, delivered_at, read_at, error,
			created_at, updated_at
		FROM messages
		WHERE tenant_id = $1 AND instance_id = $2
		ORDER BY timestamp DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, instanceID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	defer rows.Close()

	messages := make([]*models.Message, 0)

	for rows.Next() {
		var message models.Message
		var contentJSON []byte

		err := rows.Scan(
			&message.TenantID,
			&message.InstanceID,
			&message.ID,
			&message.MessageID,
			&message.Direction,
			&message.Type,
			&message.Status,
			&message.From,
			&message.To,
			&contentJSON,
			&message.Timestamp,
			&message.SentAt,
			&message.DeliveredAt,
			&message.ReadAt,
			&message.Error,
			&message.CreatedAt,
			&message.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		// Unmarshal JSON content
		if err := json.Unmarshal(contentJSON, &message.Content); err != nil {
			return nil, fmt.Errorf("failed to unmarshal content: %w", err)
		}

		messages = append(messages, &message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// CountByInstance counts messages for an instance
func (r *MessageRepository) CountByInstance(ctx context.Context, tenantID, instanceID string) (int, error) {
	var count int

	query := `
		SELECT COUNT(*)
		FROM messages
		WHERE tenant_id = $1 AND instance_id = $2
	`

	err := r.db.QueryRowContext(ctx, query, tenantID, instanceID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}

	return count, nil
}

// UpdateStatus updates the status of a message
func (r *MessageRepository) UpdateStatus(ctx context.Context, tenantID, instanceID, messageID, status string) error {
	now := time.Now()

	query := `
		UPDATE messages
		SET status = $1, updated_at = $2
		WHERE tenant_id = $3 AND instance_id = $4 AND id = $5
	`

	result, err := r.db.ExecContext(ctx, query, status, now, tenantID, instanceID, messageID)
	if err != nil {
		return fmt.Errorf("failed to update message status: %w", err)
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

// MarkAsSent marks a message as sent
func (r *MessageRepository) MarkAsSent(ctx context.Context, tenantID, instanceID, messageID string) error {
	now := time.Now()

	query := `
		UPDATE messages
		SET status = 'sent', sent_at = $1, updated_at = $2
		WHERE tenant_id = $3 AND instance_id = $4 AND id = $5
	`

	result, err := r.db.ExecContext(ctx, query, now, now, tenantID, instanceID, messageID)
	if err != nil {
		return fmt.Errorf("failed to mark message as sent: %w", err)
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

// MarkAsDelivered marks a message as delivered
func (r *MessageRepository) MarkAsDelivered(ctx context.Context, tenantID, instanceID, waMessageID string) error {
	now := time.Now()

	query := `
		UPDATE messages
		SET status = 'delivered', delivered_at = $1, updated_at = $2
		WHERE tenant_id = $3 AND instance_id = $4 AND message_id = $5
	`

	result, err := r.db.ExecContext(ctx, query, now, now, tenantID, instanceID, waMessageID)
	if err != nil {
		return fmt.Errorf("failed to mark message as delivered: %w", err)
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

// MarkAsRead marks a message as read
func (r *MessageRepository) MarkAsRead(ctx context.Context, tenantID, instanceID, waMessageID string) error {
	now := time.Now()

	query := `
		UPDATE messages
		SET status = 'read', read_at = $1, updated_at = $2
		WHERE tenant_id = $3 AND instance_id = $4 AND message_id = $5
	`

	result, err := r.db.ExecContext(ctx, query, now, now, tenantID, instanceID, waMessageID)
	if err != nil {
		return fmt.Errorf("failed to mark message as read: %w", err)
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

// MarkAsFailed marks a message as failed with an error message
func (r *MessageRepository) MarkAsFailed(ctx context.Context, tenantID, instanceID, messageID, errorMsg string) error {
	now := time.Now()

	query := `
		UPDATE messages
		SET status = 'failed', error = $1, updated_at = $2
		WHERE tenant_id = $3 AND instance_id = $4 AND id = $5
	`

	result, err := r.db.ExecContext(ctx, query, errorMsg, now, tenantID, instanceID, messageID)
	if err != nil {
		return fmt.Errorf("failed to mark message as failed: %w", err)
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

// Update updates a message in the database
func (r *MessageRepository) Update(ctx context.Context, message *models.Message) error {
	// Convert content map to JSON
	contentJSON, err := json.Marshal(message.Content)
	if err != nil {
		return fmt.Errorf("failed to marshal content: %w", err)
	}

	now := time.Now()
	message.UpdatedAt = now

	query := `
		UPDATE messages
		SET
			message_id = $1,
			status = $2,
			content = $3,
			sent_at = $4,
			delivered_at = $5,
			read_at = $6,
			error = $7,
			updated_at = $8
		WHERE tenant_id = $9 AND instance_id = $10 AND id = $11
	`

	result, err := r.db.ExecContext(
		ctx, query,
		message.MessageID,
		message.Status,
		contentJSON,
		message.SentAt,
		message.DeliveredAt,
		message.ReadAt,
		message.Error,
		message.UpdatedAt,
		message.TenantID,
		message.InstanceID,
		message.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
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

// SoftDelete soft deletes a message by setting deleted_at timestamp
func (r *MessageRepository) SoftDelete(ctx context.Context, tenantID, instanceID, messageID string) error {
	now := time.Now()

	query := `
		UPDATE messages
		SET deleted_at = $1, updated_at = $2
		WHERE tenant_id = $3 AND instance_id = $4 AND id = $5
	`

	result, err := r.db.ExecContext(ctx, query, now, now, tenantID, instanceID, messageID)
	if err != nil {
		return fmt.Errorf("failed to soft delete message: %w", err)
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
