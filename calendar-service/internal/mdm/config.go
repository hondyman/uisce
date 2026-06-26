package mdm

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds MDM service configuration
type Config struct {
	Enabled             bool
	BaseURL             string
	CacheTTL            time.Duration
	Timeout             time.Duration
	FailureMode         string // "fallback" or "strict"
	HealthCheckInterval time.Duration
}

// LoadFromEnv loads MDM configuration from environment variables
func LoadFromEnv() *Config {
	config := &Config{
		Enabled:             getEnvBool("MDM_ENABLED", true),
		BaseURL:             getEnvString("MDM_SERVICE_URL", "http://localhost:8080"),
		CacheTTL:            getEnvDuration("MDM_CACHE_TTL", 5*time.Minute),
		Timeout:             getEnvDuration("MDM_TIMEOUT", 10*time.Second),
		FailureMode:         getEnvString("MDM_FAILURE_MODE", "fallback"),
		HealthCheckInterval: getEnvDuration("MDM_HEALTH_CHECK_INTERVAL", 30*time.Second),
	}

	return config
}

// ============================================================================
// Helper Functions
// ============================================================================

func getEnvString(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil // No validation needed if disabled
	}

	if c.BaseURL == "" {
		return fmt.Errorf("MDM_SERVICE_URL is required when MDM is enabled")
	}

	if c.CacheTTL <= 0 {
		return fmt.Errorf("MDM_CACHE_TTL must be positive")
	}

	if c.FailureMode != "fallback" && c.FailureMode != "strict" {
		return fmt.Errorf("MDM_FAILURE_MODE must be 'fallback' or 'strict'")
	}

	return nil
}

// String returns a masked string representation
func (c *Config) String() string {
	return fmt.Sprintf(
		"MDMConfig{Enabled:%v, BaseURL:%s, CacheTTL:%v, FailureMode:%s}",
		c.Enabled,
		c.BaseURL,
		c.CacheTTL,
		c.FailureMode,
	)
}
