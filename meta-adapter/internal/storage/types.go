package storage

import "time"

// SessionInfo represents stored session information
type SessionInfo struct {
	ID          string
	PhoneNumber string
	PushName    string
	Connected   bool
	ConnectedAt time.Time
}

// MediaInfo represents cached media information
type MediaInfo struct {
	ID          string
	WhatsAppURL string
	MimeType    string
	Size        int64
	SHA256      string
}
