package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

// Config holds all semantic-engine service configuration
type Config struct {
	// Server configuration
	ServerPort int

	// Hasura configuration
	HasuraEndpoint    string
	HasuraAdminSecret string

	// Service endpoints
	AIServiceEndpoint         string
	GovernanceServiceEndpoint string

	// Temporal configuration (TODO: implement when ready)
	TemporalHostPort  string
	TemporalNamespace string
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() (*Config, error) {
	cfg := &Config{
		ServerPort:                getEnvInt("SEMANTIC_ENGINE_PORT", 8086),
		HasuraEndpoint:            getEnv("HASURA_ENDPOINT", ""),
		HasuraAdminSecret:         getEnv("HASURA_ADMIN_SECRET", ""),
		AIServiceEndpoint:         getEnv("AI_SERVICE_ENDPOINT", "http://localhost:8082"),
		GovernanceServiceEndpoint: getEnv("GOVERNANCE_SERVICE_ENDPOINT", "http://localhost:8084"),
		TemporalHostPort:          getEnv("TEMPORAL_HOST_PORT", "localhost:7233"),
		TemporalNamespace:         getEnv("TEMPORAL_NAMESPACE", "default"),
	}

	if err := Validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate ensures all required configuration is present
func Validate(cfg *Config) error {
	if cfg.ServerPort <= 0 || cfg.ServerPort > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.ServerPort)
	}

	if cfg.AIServiceEndpoint == "" {
		return errors.New("AI_SERVICE_ENDPOINT is required")
	}

	if cfg.GovernanceServiceEndpoint == "" {
		return errors.New("GOVERNANCE_SERVICE_ENDPOINT is required")
	}

	// Hasura is optional - may not be needed for all deployments
	// Temporal is optional - may not be needed for all deployments

	return nil
}

// Helper functions
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
