package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for Trellis
type Config struct {
	// Server configuration
	Port        int    `json:"port"`
	Environment string `json:"environment"`
	LogLevel    string `json:"log_level"`

	// Warden configuration for organization-aware authentication
	Warden WardenConfig `json:"warden"`

	// ClickHouse configuration
	ClickHouse ClickHouseConfig `json:"clickhouse"`

	// Redis configuration for deduplication and caching
	Redis RedisConfig `json:"redis"`

	// Google Cloud Pub/Sub configuration
	PubSub PubSubConfig `json:"pubsub"`

	// Google Cloud Storage configuration
	GCS GCSConfig `json:"gcs"`
}

// WardenConfig holds Warden service connection settings
type WardenConfig struct {
	// Warden service address (e.g., "warden.example.com:21382")
	Address string `json:"address"`
	
	// Whether to use TLS for gRPC connection
	TLS bool `json:"tls"`
	
	// Service account API key for internal operations (optional)
	ServiceAPIKey string `json:"service_api_key"`
	
	// Connection timeout in seconds
	TimeoutSeconds int `json:"timeout_seconds"`
}

// ClickHouseConfig holds ClickHouse database settings
type ClickHouseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	
	// Connection settings
	MaxOpenConnections int `json:"max_open_connections"`
	ConnMaxLifetime    int `json:"conn_max_lifetime_minutes"`
}

// RedisConfig holds Redis connection settings
type RedisConfig struct {
	// Redis URL (redis://localhost:6379/0)
	URL string `json:"url"`
	
	// Connection pool settings
	PoolSize     int `json:"pool_size"`
	MinIdleConns int `json:"min_idle_conns"`
	
	// Organization-scoped key prefix
	KeyPrefix string `json:"key_prefix"`
}

// PubSubConfig holds Google Cloud Pub/Sub settings
type PubSubConfig struct {
	ProjectID string `json:"project_id"`
	TopicID   string `json:"topic_id"`
	
	// Subscription settings for workers
	SubscriptionID string `json:"subscription_id"`
	
	// Publishing settings
	MaxOutstandingMessages int `json:"max_outstanding_messages"`
	NumGoroutines          int `json:"num_goroutines"`
}

// GCSConfig holds Google Cloud Storage settings
type GCSConfig struct {
	ProjectID  string `json:"project_id"`
	BucketName string `json:"bucket_name"`
	
	// Archive settings
	ArchivePrefix string `json:"archive_prefix"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	config := &Config{
		Port:        getEnvInt("TRELLIS_PORT", 8080),
		Environment: getEnvString("TRELLIS_ENV", "development"),
		LogLevel:    getEnvString("TRELLIS_LOG_LEVEL", "info"),
		
		Warden: WardenConfig{
			Address:        getEnvString("WARDEN_ADDRESS", "localhost:21382"),
			TLS:            getEnvBool("WARDEN_TLS", false),
			ServiceAPIKey:  getEnvString("WARDEN_SERVICE_API_KEY", ""),
			TimeoutSeconds: getEnvInt("WARDEN_TIMEOUT_SECONDS", 30),
		},
		
		ClickHouse: ClickHouseConfig{
			Host:               getEnvString("CLICKHOUSE_HOST", "localhost"),
			Port:               getEnvInt("CLICKHOUSE_PORT", 8123),
			Database:           getEnvString("CLICKHOUSE_DATABASE", "trellis"),
			Username:           getEnvString("CLICKHOUSE_USERNAME", "default"),
			Password:           getEnvString("CLICKHOUSE_PASSWORD", ""),
			MaxOpenConnections: getEnvInt("CLICKHOUSE_MAX_OPEN_CONNS", 10),
			ConnMaxLifetime:    getEnvInt("CLICKHOUSE_CONN_MAX_LIFETIME", 60),
		},
		
		Redis: RedisConfig{
			URL:          getEnvString("REDIS_URL", "redis://localhost:6379/0"),
			PoolSize:     getEnvInt("REDIS_POOL_SIZE", 10),
			MinIdleConns: getEnvInt("REDIS_MIN_IDLE_CONNS", 2),
			KeyPrefix:    getEnvString("REDIS_KEY_PREFIX", "trellis"),
		},
		
		PubSub: PubSubConfig{
			ProjectID:              getEnvString("PUBSUB_PROJECT_ID", ""),
			TopicID:                getEnvString("PUBSUB_TOPIC_ID", "trellis-events"),
			SubscriptionID:         getEnvString("PUBSUB_SUBSCRIPTION_ID", "trellis-processor"),
			MaxOutstandingMessages: getEnvInt("PUBSUB_MAX_OUTSTANDING", 1000),
			NumGoroutines:          getEnvInt("PUBSUB_NUM_GOROUTINES", 10),
		},
		
		GCS: GCSConfig{
			ProjectID:     getEnvString("GCS_PROJECT_ID", ""),
			BucketName:    getEnvString("GCS_BUCKET_NAME", ""),
			ArchivePrefix: getEnvString("GCS_ARCHIVE_PREFIX", "events"),
		},
	}
	
	// Validate required configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	
	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}
	
	if c.Warden.Address == "" {
		return fmt.Errorf("warden address is required")
	}
	
	if c.ClickHouse.Host == "" {
		return fmt.Errorf("clickhouse host is required")
	}
	
	if c.ClickHouse.Database == "" {
		return fmt.Errorf("clickhouse database is required")
	}
	
	if c.Redis.URL == "" {
		return fmt.Errorf("redis URL is required")
	}
	
	// PubSub validation (optional for development)
	if c.Environment == "production" {
		if c.PubSub.ProjectID == "" {
			return fmt.Errorf("pubsub project ID is required in production")
		}
		if c.GCS.ProjectID == "" {
			return fmt.Errorf("gcs project ID is required in production")
		}
	}
	
	return nil
}

// GetWardenAddress returns the complete Warden gRPC address
func (c *Config) GetWardenAddress() string {
	if c.Warden.TLS {
		return c.Warden.Address // Assume TLS is handled by the gRPC client
	}
	return c.Warden.Address
}

// GetClickHouseConnectionString returns the ClickHouse connection string
func (c *Config) GetClickHouseConnectionString() string {
	if c.ClickHouse.Password != "" {
		return fmt.Sprintf("tcp://%s:%s@%s:%d/%s",
			c.ClickHouse.Username,
			c.ClickHouse.Password,
			c.ClickHouse.Host,
			c.ClickHouse.Port,
			c.ClickHouse.Database,
		)
	}
	
	return fmt.Sprintf("tcp://%s@%s:%d/%s",
		c.ClickHouse.Username,
		c.ClickHouse.Host,
		c.ClickHouse.Port,
		c.ClickHouse.Database,
	)
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return strings.ToLower(c.Environment) == "production"
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return strings.ToLower(c.Environment) == "development"
}

// Helper functions for environment variable parsing

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}