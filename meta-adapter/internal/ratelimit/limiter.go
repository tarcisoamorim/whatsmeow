package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	pkglogger "github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/logger"
)

// TierLimits defines rate limits for each tenant tier
type TierLimits struct {
	RequestsPerMinute int
	RequestsPerHour   int
	RequestsPerDay    int
}

var tierLimits = map[string]TierLimits{
	"free": {
		RequestsPerMinute: 10,
		RequestsPerHour:   100,
		RequestsPerDay:    1000,
	},
	"starter": {
		RequestsPerMinute: 60,
		RequestsPerHour:   1000,
		RequestsPerDay:    10000,
	},
	"business": {
		RequestsPerMinute: 300,
		RequestsPerHour:   10000,
		RequestsPerDay:    100000,
	},
	"enterprise": {
		RequestsPerMinute: 1000,
		RequestsPerHour:   50000,
		RequestsPerDay:    1000000,
	},
}

// Limiter handles rate limiting using Redis
type Limiter struct {
	client *redis.Client
}

// NewLimiter creates a new rate limiter
func NewLimiter(redisURL string) (*Limiter, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Limiter{client: client}, nil
}

// Allow checks if a request is allowed for the given tenant and tier
func (l *Limiter) Allow(ctx context.Context, tenantID, tier string) (bool, RateLimitInfo, error) {
	logger := pkglogger.Get()

	limits, ok := tierLimits[tier]
	if !ok {
		// Default to free tier if tier is unknown
		limits = tierLimits["free"]
		logger.Warn("Unknown tier, using free tier limits", zap.String("tier", tier))
	}

	now := time.Now()

	// Check minute limit
	minuteKey := fmt.Sprintf("ratelimit:%s:minute:%s", tenantID, now.Format("2006-01-02:15:04"))
	minuteCount, err := l.increment(ctx, minuteKey, 60*time.Second)
	if err != nil {
		return false, RateLimitInfo{}, err
	}

	if minuteCount > int64(limits.RequestsPerMinute) {
		return false, RateLimitInfo{
			Limit:     limits.RequestsPerMinute,
			Remaining: 0,
			Reset:     now.Add(time.Minute).Unix(),
			Window:    "minute",
		}, nil
	}

	// Check hour limit
	hourKey := fmt.Sprintf("ratelimit:%s:hour:%s", tenantID, now.Format("2006-01-02:15"))
	hourCount, err := l.increment(ctx, hourKey, 60*time.Minute)
	if err != nil {
		return false, RateLimitInfo{}, err
	}

	if hourCount > int64(limits.RequestsPerHour) {
		return false, RateLimitInfo{
			Limit:     limits.RequestsPerHour,
			Remaining: 0,
			Reset:     now.Add(time.Hour).Unix(),
			Window:    "hour",
		}, nil
	}

	// Check day limit
	dayKey := fmt.Sprintf("ratelimit:%s:day:%s", tenantID, now.Format("2006-01-02"))
	dayCount, err := l.increment(ctx, dayKey, 24*time.Hour)
	if err != nil {
		return false, RateLimitInfo{}, err
	}

	if dayCount > int64(limits.RequestsPerDay) {
		return false, RateLimitInfo{
			Limit:     limits.RequestsPerDay,
			Remaining: 0,
			Reset:     now.AddDate(0, 0, 1).Unix(),
			Window:    "day",
		}, nil
	}

	// Request is allowed
	return true, RateLimitInfo{
		Limit:     limits.RequestsPerMinute,
		Remaining: limits.RequestsPerMinute - int(minuteCount),
		Reset:     now.Add(time.Minute).Unix(),
		Window:    "minute",
	}, nil
}

// increment increments a counter in Redis and sets expiration
func (l *Limiter) increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	pipe := l.client.Pipeline()

	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to increment counter: %w", err)
	}

	return incr.Val(), nil
}

// GetCurrentUsage returns current usage for a tenant
func (l *Limiter) GetCurrentUsage(ctx context.Context, tenantID, tier string) (UsageInfo, error) {
	limits, ok := tierLimits[tier]
	if !ok {
		limits = tierLimits["free"]
	}

	now := time.Now()

	minuteKey := fmt.Sprintf("ratelimit:%s:minute:%s", tenantID, now.Format("2006-01-02:15:04"))
	hourKey := fmt.Sprintf("ratelimit:%s:hour:%s", tenantID, now.Format("2006-01-02:15"))
	dayKey := fmt.Sprintf("ratelimit:%s:day:%s", tenantID, now.Format("2006-01-02"))

	minuteCount, _ := l.client.Get(ctx, minuteKey).Int64()
	hourCount, _ := l.client.Get(ctx, hourKey).Int64()
	dayCount, _ := l.client.Get(ctx, dayKey).Int64()

	return UsageInfo{
		Minute: LimitUsage{
			Used:  int(minuteCount),
			Limit: limits.RequestsPerMinute,
		},
		Hour: LimitUsage{
			Used:  int(hourCount),
			Limit: limits.RequestsPerHour,
		},
		Day: LimitUsage{
			Used:  int(dayCount),
			Limit: limits.RequestsPerDay,
		},
	}, nil
}

// GetClient returns the underlying Redis client for health checks
func (l *Limiter) GetClient() *redis.Client {
	return l.client
}

// Close closes the Redis connection
func (l *Limiter) Close() error {
	return l.client.Close()
}

// RateLimitInfo contains information about rate limit status
type RateLimitInfo struct {
	Limit     int    `json:"limit"`
	Remaining int    `json:"remaining"`
	Reset     int64  `json:"reset"` // Unix timestamp
	Window    string `json:"window"`
}

// UsageInfo contains current usage statistics
type UsageInfo struct {
	Minute LimitUsage `json:"minute"`
	Hour   LimitUsage `json:"hour"`
	Day    LimitUsage `json:"day"`
}

// LimitUsage represents usage for a specific time window
type LimitUsage struct {
	Used  int `json:"used"`
	Limit int `json:"limit"`
}
