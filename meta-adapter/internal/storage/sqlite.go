package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// SQLiteStore implements a SQLite-based storage
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite store
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &SQLiteStore{db: db}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// initSchema creates the necessary tables
func (s *SQLiteStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		phone_number TEXT,
		push_name TEXT,
		connected BOOLEAN DEFAULT 0,
		connected_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS media_cache (
		id TEXT PRIMARY KEY,
		whatsapp_url TEXT NOT NULL,
		mime_type TEXT,
		size INTEGER,
		sha256 TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_phone ON sessions(phone_number);
	CREATE INDEX IF NOT EXISTS idx_media_expires ON media_cache(expires_at);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Session operations

// UpsertSession creates or updates a session
func (s *SQLiteStore) UpsertSession(id, phoneNumber, pushName string, connected bool) error {
	query := `
		INSERT INTO sessions (id, phone_number, push_name, connected, connected_at, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			phone_number = excluded.phone_number,
			push_name = excluded.push_name,
			connected = excluded.connected,
			connected_at = CASE WHEN excluded.connected = 1 THEN CURRENT_TIMESTAMP ELSE connected_at END,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := s.db.Exec(query, id, phoneNumber, pushName, connected)
	return err
}

// GetSession retrieves a session by ID
func (s *SQLiteStore) GetSession(id string) (*SessionInfo, error) {
	query := `SELECT id, phone_number, push_name, connected, connected_at FROM sessions WHERE id = ?`

	var session SessionInfo
	var phoneNumber, pushName sql.NullString
	var connectedAt sql.NullTime

	err := s.db.QueryRow(query, id).Scan(
		&session.ID,
		&phoneNumber,
		&pushName,
		&session.Connected,
		&connectedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if phoneNumber.Valid {
		session.PhoneNumber = phoneNumber.String
	}
	if pushName.Valid {
		session.PushName = pushName.String
	}
	if connectedAt.Valid {
		session.ConnectedAt = connectedAt.Time
	}

	return &session, nil
}

// DeleteSession deletes a session
func (s *SQLiteStore) DeleteSession(id string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE id = ?", id)
	return err
}

// Media cache operations

// CacheMedia stores media information
func (s *SQLiteStore) CacheMedia(id, whatsappURL, mimeType, sha256 string, size int64) error {
	query := `
		INSERT INTO media_cache (id, whatsapp_url, mime_type, size, sha256, expires_at)
		VALUES (?, ?, ?, ?, ?, datetime('now', '+24 hours'))
		ON CONFLICT(id) DO UPDATE SET
			whatsapp_url = excluded.whatsapp_url,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := s.db.Exec(query, id, whatsappURL, mimeType, size, sha256)
	return err
}

// GetCachedMedia retrieves cached media information
func (s *SQLiteStore) GetCachedMedia(id string) (*MediaInfo, error) {
	query := `SELECT id, whatsapp_url, mime_type, size, sha256 FROM media_cache WHERE id = ? AND expires_at > CURRENT_TIMESTAMP`

	var media MediaInfo
	var mimeType, sha256 sql.NullString
	var size sql.NullInt64

	err := s.db.QueryRow(query, id).Scan(
		&media.ID,
		&media.WhatsAppURL,
		&mimeType,
		&size,
		&sha256,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if mimeType.Valid {
		media.MimeType = mimeType.String
	}
	if size.Valid {
		media.Size = size.Int64
	}
	if sha256.Valid {
		media.SHA256 = sha256.String
	}

	return &media, nil
}

// CleanupExpiredMedia removes expired media cache entries
func (s *SQLiteStore) CleanupExpiredMedia() error {
	_, err := s.db.Exec("DELETE FROM media_cache WHERE expires_at < CURRENT_TIMESTAMP")
	return err
}
