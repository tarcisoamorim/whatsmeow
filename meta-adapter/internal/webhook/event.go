package webhook

import (
	"time"
)

// EventType represents the type of webhook event
type EventType string

const (
	EventMessageReceived EventType = "message.received"
	EventMessageSent     EventType = "message.sent"
	EventMessageDelivered EventType = "message.delivered"
	EventMessageRead     EventType = "message.read"
	EventMessageFailed   EventType = "message.failed"
	EventInstanceConnected EventType = "instance.connected"
	EventInstanceDisconnected EventType = "instance.disconnected"
)

// Event represents a webhook event to be delivered
type Event struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	InstanceID  string                 `json:"instance_id"`
	Type        EventType              `json:"type"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data"`
	WebhookURL  string                 `json:"-"` // Not sent in payload
	RetryCount  int                    `json:"-"`
	MaxRetries  int                    `json:"-"`
	NextRetryAt time.Time              `json:"-"`
}

// WebhookPayload is the actual payload sent to webhook URL
type WebhookPayload struct {
	Event     EventType              `json:"event"`
	Timestamp string                 `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Signature string                 `json:"signature,omitempty"`
}

// DeliveryResult represents the result of a webhook delivery attempt
type DeliveryResult struct {
	Success      bool
	StatusCode   int
	ResponseBody string
	Error        error
	Duration     time.Duration
}
