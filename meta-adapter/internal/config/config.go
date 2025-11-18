package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	RabbitMQ RabbitMQConfig
	JWT      JWTConfig
	Webhook  WebhookConfig
}

type ServerConfig struct {
	Host string
	Port string
}

type DatabaseConfig struct {
	URL string
}

type RedisConfig struct {
	URL string
}

type RabbitMQConfig struct {
	URL string
}

type JWTConfig struct {
	Secret string
}

type WebhookConfig struct {
	Secret string
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host: getEnv("HOST", "0.0.0.0"),
			Port: getEnv("PORT", "8080"),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", ""),
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379/0"),
		},
		RabbitMQ: RabbitMQConfig{
			URL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "change-me-in-production"),
		},
		Webhook: WebhookConfig{
			Secret: getEnv("WEBHOOK_SECRET", "change-me-in-production"),
		},
	}

	// Validate critical configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that critical configuration values are set and secure
func (c *Config) Validate() error {
	// JWT Secret validation
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET environment variable must be set")
	}
	if c.JWT.Secret == "change-me-in-production" {
		return fmt.Errorf("JWT_SECRET cannot be the default value 'change-me-in-production'. Generate a secure secret with: openssl rand -base64 48")
	}
	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters long for security. Current length: %d", len(c.JWT.Secret))
	}

	// Webhook Secret validation
	if c.Webhook.Secret == "" {
		return fmt.Errorf("WEBHOOK_SECRET environment variable must be set")
	}
	if c.Webhook.Secret == "change-me-in-production" {
		return fmt.Errorf("WEBHOOK_SECRET cannot be the default value. Generate a secure secret with: openssl rand -base64 48")
	}
	if len(c.Webhook.Secret) < 32 {
		return fmt.Errorf("WEBHOOK_SECRET must be at least 32 characters long for security. Current length: %d", len(c.Webhook.Secret))
	}

	// Database URL validation
	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL environment variable must be set")
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
