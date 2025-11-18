package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

// Config holds all configuration for the application
type Config struct {
	Server    ServerConfig
	WhatsApp  WhatsAppConfig
	Webhook   WebhookConfig
	Security  SecurityConfig
	RateLimit RateLimitConfig
	Logging   LoggingConfig
	Database  DatabaseConfig
	CORS      CORSConfig
	Media     MediaConfig
	Session   SessionConfig
}

type ServerConfig struct {
	Port        string
	Host        string
	Environment string
	APIVersion  string
	BasePath    string
}

type WhatsAppConfig struct {
	StorePath      string
	AutoReconnect  bool
}

type WebhookConfig struct {
	URL            string
	VerifyToken    string
	Timeout        time.Duration
	RetryAttempts  int
}

type SecurityConfig struct {
	APIKey  string
	APIKeys []string // Multiple API keys for multi-tenant
}

type RateLimitConfig struct {
	Enabled            bool
	RequestsPerMinute  int
	Burst              int
}

type LoggingConfig struct {
	Level  string
	Format string // "json" or "console"
}

type DatabaseConfig struct {
	Path string
}

type CORSConfig struct {
	Enabled        bool
	AllowedOrigins []string
}

type MediaConfig struct {
	MaxSizeMB int
	TempDir   string
}

type SessionConfig struct {
	Timeout         time.Duration
	CleanupInterval time.Duration
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port:        getEnv("PORT", "8080"),
			Host:        getEnv("HOST", "0.0.0.0"),
			Environment: getEnv("ENVIRONMENT", "development"),
			APIVersion:  getEnv("API_VERSION", "v1"),
			BasePath:    getEnv("API_BASE_PATH", "/api/v1"),
		},
		WhatsApp: WhatsAppConfig{
			StorePath:     getEnv("WHATSAPP_STORE_PATH", "./data/sessions"),
			AutoReconnect: getEnvBool("WHATSAPP_AUTO_RECONNECT", true),
		},
		Webhook: WebhookConfig{
			URL:           getEnv("WEBHOOK_URL", ""),
			VerifyToken:   getEnv("WEBHOOK_VERIFY_TOKEN", ""),
			Timeout:       getEnvDuration("WEBHOOK_TIMEOUT", 30*time.Second),
			RetryAttempts: getEnvInt("WEBHOOK_RETRY_ATTEMPTS", 3),
		},
		Security: SecurityConfig{
			APIKey:  getEnv("API_KEY", ""),
			APIKeys: getEnvSlice("API_KEYS", ","),
		},
		RateLimit: RateLimitConfig{
			Enabled:           getEnvBool("RATE_LIMIT_ENABLED", true),
			RequestsPerMinute: getEnvInt("RATE_LIMIT_REQUESTS_PER_MINUTE", 30),
			Burst:             getEnvInt("RATE_LIMIT_BURST", 10),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Database: DatabaseConfig{
			Path: getEnv("DB_PATH", "./data/adapter.db"),
		},
		CORS: CORSConfig{
			Enabled:        getEnvBool("CORS_ENABLED", true),
			AllowedOrigins: getEnvSlice("CORS_ALLOWED_ORIGINS", ","),
		},
		Media: MediaConfig{
			MaxSizeMB: getEnvInt("MEDIA_MAX_SIZE_MB", 16),
			TempDir:   getEnv("MEDIA_TEMP_DIR", "./data/temp"),
		},
		Session: SessionConfig{
			Timeout:         getEnvDuration("SESSION_TIMEOUT", 24*time.Hour),
			CleanupInterval: getEnvDuration("SESSION_CLEANUP_INTERVAL", 1*time.Hour),
		},
	}

	// Validation
	if cfg.Security.APIKey == "" && len(cfg.Security.APIKeys) == 0 {
		log.Warn().Msg("No API keys configured - authentication disabled!")
	}

	return cfg, nil
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		b, err := strconv.ParseBool(value)
		if err != nil {
			log.Warn().Str("key", key).Str("value", value).Msg("Invalid boolean value, using default")
			return defaultValue
		}
		return b
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		i, err := strconv.Atoi(value)
		if err != nil {
			log.Warn().Str("key", key).Str("value", value).Msg("Invalid integer value, using default")
			return defaultValue
		}
		return i
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		d, err := time.ParseDuration(value)
		if err != nil {
			log.Warn().Str("key", key).Str("value", value).Msg("Invalid duration value, using default")
			return defaultValue
		}
		return d
	}
	return defaultValue
}

func getEnvSlice(key, separator string) []string {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, separator)
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	return []string{}
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

// HasWebhook returns true if webhook URL is configured
func (c *Config) HasWebhook() bool {
	return c.Webhook.URL != ""
}

// ValidAPIKey checks if the given API key is valid
func (c *Config) ValidAPIKey(key string) bool {
	if key == "" {
		return false
	}

	// Check single API key
	if c.Security.APIKey != "" && c.Security.APIKey == key {
		return true
	}

	// Check multiple API keys
	for _, k := range c.Security.APIKeys {
		if k == key {
			return true
		}
	}

	return false
}
