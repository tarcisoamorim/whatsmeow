package models

import "time"

// QRCodeResponse represents QR code for session pairing
type QRCodeResponse struct {
	QRCode    string    `json:"qr_code"`
	ExpiresAt time.Time `json:"expires_at"`
	Timeout   int       `json:"timeout"` // seconds
}

// SessionStatusResponse represents the current session status
type SessionStatusResponse struct {
	SessionID   string    `json:"session_id"`
	Connected   bool      `json:"connected"`
	PhoneNumber string    `json:"phone_number,omitempty"`
	PushName    string    `json:"push_name,omitempty"`
	ConnectedAt time.Time `json:"connected_at,omitempty"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status  string    `json:"status"` // "ok" or "error"
	Version string    `json:"version"`
	Time    time.Time `json:"time"`
	Session *SessionStatusResponse `json:"session,omitempty"`
}

// MediaUploadResponse represents media upload response
type MediaUploadResponse struct {
	ID string `json:"id"` // Internal media ID
}

// SuccessResponse generic success response
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}
