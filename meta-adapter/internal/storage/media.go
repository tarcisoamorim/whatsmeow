package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// MediaStore provides Redis-based storage for uploaded media
// Implements the Meta API "Upload ID -> Send Message" pattern
type MediaStore struct {
	client *redis.Client
	ttl    time.Duration
}

// MediaMetadata contains information about stored media
type MediaMetadata struct {
	ID        string    `json:"id"`
	Data      []byte    `json:"data"`
	MimeType  string    `json:"mime_type"`
	Filename  string    `json:"filename,omitempty"`
	Size      int64     `json:"size"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// NewMediaStore creates a new Redis-based media store
func NewMediaStore(client *redis.Client) *MediaStore {
	return &MediaStore{
		client: client,
		ttl:    24 * time.Hour, // Meta API standard: 24h TTL
	}
}

// Store saves media data to Redis and returns a unique ID
// The ID can be used later in send message requests
func (s *MediaStore) Store(ctx context.Context, data []byte, mimeType, filename string) (string, error) {
	// Generate unique media ID
	id := uuid.New().String()

	// Create metadata
	metadata := MediaMetadata{
		ID:        id,
		Data:      data,
		MimeType:  mimeType,
		Filename:  filename,
		Size:      int64(len(data)),
		UploadedAt: time.Now(),
	}

	// Serialize metadata
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to serialize media metadata: %w", err)
	}

	// Store in Redis with TTL
	key := fmt.Sprintf("media:%s", id)
	if err := s.client.Set(ctx, key, jsonData, s.ttl).Err(); err != nil {
		return "", fmt.Errorf("failed to store media in Redis: %w", err)
	}

	return id, nil
}

// Get retrieves media data by ID
func (s *MediaStore) Get(ctx context.Context, id string) (*MediaMetadata, error) {
	key := fmt.Sprintf("media:%s", id)

	// Get from Redis
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("media not found or expired: %s", id)
		}
		return nil, fmt.Errorf("failed to retrieve media from Redis: %w", err)
	}

	// Deserialize metadata
	var metadata MediaMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to deserialize media metadata: %w", err)
	}

	return &metadata, nil
}

// Delete removes media from Redis
func (s *MediaStore) Delete(ctx context.Context, id string) error {
	key := fmt.Sprintf("media:%s", id)
	return s.client.Del(ctx, key).Err()
}

// Exists checks if media with the given ID exists in storage
func (s *MediaStore) Exists(ctx context.Context, id string) (bool, error) {
	key := fmt.Sprintf("media:%s", id)
	count, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check media existence: %w", err)
	}
	return count > 0, nil
}

// ExtendTTL extends the expiration time of stored media
func (s *MediaStore) ExtendTTL(ctx context.Context, id string, duration time.Duration) error {
	key := fmt.Sprintf("media:%s", id)
	return s.client.Expire(ctx, key, duration).Err()
}

// GetTTL returns the remaining TTL for stored media
func (s *MediaStore) GetTTL(ctx context.Context, id string) (time.Duration, error) {
	key := fmt.Sprintf("media:%s", id)
	return s.client.TTL(ctx, key).Result()
}

// Stats returns statistics about stored media
type MediaStats struct {
	TotalKeys   int64 `json:"total_keys"`
	TotalMemory int64 `json:"total_memory_bytes"`
}

// GetStats returns statistics about media storage (optional, for monitoring)
func (s *MediaStore) GetStats(ctx context.Context) (*MediaStats, error) {
	// Use SCAN to count media keys
	iter := s.client.Scan(ctx, 0, "media:*", 100).Iterator()
	count := int64(0)
	for iter.Next(ctx) {
		count++
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan media keys: %w", err)
	}

	// Get memory info from Redis
	info, err := s.client.Info(ctx, "memory").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis memory info: %w", err)
	}

	// Parse memory usage (simplified - just return the count for now)
	return &MediaStats{
		TotalKeys:   count,
		TotalMemory: 0, // Would need to parse INFO output for accurate value
	}, nil
}
