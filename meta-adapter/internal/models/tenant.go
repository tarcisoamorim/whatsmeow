package models

import (
	"time"
	"github.com/google/uuid"
)

type Tenant struct {
	ID    string `db:"id" json:"id"`
	Name  string `db:"name" json:"name" validate:"required,min=1,max=255"`
	Email string `db:"email" json:"email" validate:"required,email"`

	// Auth
	PasswordHash             string     `db:"password_hash" json:"-"`
	EmailVerified            bool       `db:"email_verified" json:"email_verified"`
	EmailVerificationToken   *string    `db:"email_verification_token" json:"-"`
	EmailVerificationExpires *time.Time `db:"email_verification_expires_at" json:"-"`

	// OAuth2
	OAuthClientID         *string `db:"oauth_client_id" json:"oauth_client_id,omitempty"`
	OAuthClientSecretHash *string `db:"oauth_client_secret_hash" json:"-"`
	OAuthRedirectURIs     []string `db:"oauth_redirect_uris" json:"oauth_redirect_uris,omitempty"`

	// API Key
	APIKeyHash   *string `db:"api_key_hash" json:"-"`
	APIKeyPrefix *string `db:"api_key_prefix" json:"api_key_prefix,omitempty"`

	// Limits
	Tier                      string `db:"tier" json:"tier"`
	MaxInstances              int    `db:"max_instances" json:"max_instances"`
	MaxMessagesPerDay         int    `db:"max_messages_per_day" json:"max_messages_per_day"`
	MaxAPIRequestsPerMinute   int    `db:"max_api_requests_per_minute" json:"max_api_requests_per_minute"`
	MaxWebhookRetries         int    `db:"max_webhook_retries" json:"max_webhook_retries"`

	// Billing
	StripeCustomerID    *string    `db:"stripe_customer_id" json:"stripe_customer_id,omitempty"`
	SubscriptionStatus  string     `db:"subscription_status" json:"subscription_status"`
	SubscriptionStarted *time.Time `db:"subscription_started_at" json:"subscription_started_at,omitempty"`
	SubscriptionEnds    *time.Time `db:"subscription_ends_at" json:"subscription_ends_at,omitempty"`
	TrialEnds           *time.Time `db:"trial_ends_at" json:"trial_ends_at,omitempty"`

	// Security
	TwoFactorEnabled bool     `db:"two_factor_enabled" json:"two_factor_enabled"`
	WebhookSecret    *string  `db:"webhook_secret" json:"-"`

	// Status
	Status string `db:"status" json:"status"`

	// Timestamps
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
	LastLoginAt *time.Time `db:"last_login_at" json:"last_login_at,omitempty"`
	DeletedAt   *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

func NewTenant(name, email string) *Tenant {
	return &Tenant{
		ID:                      uuid.New().String(),
		Name:                    name,
		Email:                   email,
		Tier:                    "free",
		MaxInstances:            1,
		MaxMessagesPerDay:       100,
		MaxAPIRequestsPerMinute: 30,
		MaxWebhookRetries:       5,
		SubscriptionStatus:      "active",
		Status:                  "active",
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}
}

type Instance struct {
	TenantID        string `db:"tenant_id" json:"tenant_id"`
	ID              string `db:"id" json:"id"`
	PhoneNumber     *string `db:"phone_number" json:"phone_number,omitempty"`
	PhoneNumberID   string `db:"phone_number_id" json:"phone_number_id"`
	DisplayName     *string `db:"display_name" json:"display_name,omitempty"`
	Status          string `db:"status" json:"status"`
	ConnectionState string `db:"connection_state" json:"connection_state"`
	QRCode          *string `db:"qr_code" json:"qr_code,omitempty"`
	QRCodeExpires   *time.Time `db:"qr_code_expires_at" json:"qr_code_expires_at,omitempty"`
	WebhookURL      *string `db:"webhook_url" json:"webhook_url,omitempty"`
	WebhookEvents   []string `db:"webhook_events" json:"webhook_events,omitempty"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
	ConnectedAt     *time.Time `db:"connected_at" json:"connected_at,omitempty"`
}

func NewInstance(tenantID string) *Instance {
	return &Instance{
		TenantID:        tenantID,
		ID:              uuid.New().String(),
		PhoneNumberID:   uuid.New().String(),
		Status:          "disconnected",
		ConnectionState: "disconnected",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}
