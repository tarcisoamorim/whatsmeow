package models

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	TenantID   string                 `db:"tenant_id" json:"tenant_id"`
	InstanceID string                 `db:"instance_id" json:"instance_id"`
	ID         string                 `db:"id" json:"id"`
	MessageID  string                 `db:"message_id" json:"message_id"` // WhatsApp message ID (wamid.xxx)
	Direction  string                 `db:"direction" json:"direction"`   // inbound, outbound
	Type       string                 `db:"type" json:"type"`             // text, image, audio, video, document, location, contacts, sticker
	Status     string                 `db:"status" json:"status"`         // pending, sent, delivered, read, failed
	From       string                 `db:"from_number" json:"from"`
	To         string                 `db:"to_number" json:"to"`
	Content    map[string]interface{} `db:"content" json:"content"` // JSONB - flexible message content
	Timestamp  time.Time              `db:"timestamp" json:"timestamp"`
	SentAt     *time.Time             `db:"sent_at" json:"sent_at,omitempty"`
	DeliveredAt *time.Time            `db:"delivered_at" json:"delivered_at,omitempty"`
	ReadAt     *time.Time             `db:"read_at" json:"read_at,omitempty"`
	Error      *string                `db:"error" json:"error,omitempty"`
	CreatedAt  time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time              `db:"updated_at" json:"updated_at"`
}

func NewMessage(tenantID, instanceID, direction, messageType, from, to string, content map[string]interface{}) *Message {
	now := time.Now()
	messageID := "wamid." + uuid.New().String()

	return &Message{
		TenantID:   tenantID,
		InstanceID: instanceID,
		ID:         uuid.New().String(),
		MessageID:  messageID,
		Direction:  direction,
		Type:       messageType,
		Status:     "pending",
		From:       from,
		To:         to,
		Content:    content,
		Timestamp:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// NewOutboundTextMessage creates a new outbound text message
func NewOutboundTextMessage(tenantID, instanceID, from, to, text string) *Message {
	content := map[string]interface{}{
		"text": map[string]interface{}{
			"body": text,
		},
	}
	return NewMessage(tenantID, instanceID, "outbound", "text", from, to, content)
}

// NewInboundTextMessage creates a new inbound text message
func NewInboundTextMessage(tenantID, instanceID, from, to, text, waMessageID string) *Message {
	content := map[string]interface{}{
		"text": map[string]interface{}{
			"body": text,
		},
	}
	msg := NewMessage(tenantID, instanceID, "inbound", "text", from, to, content)
	msg.MessageID = waMessageID
	msg.Status = "received"
	return msg
}
