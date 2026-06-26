package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

// Config holds all governance service configuration
type Config struct {
	// Server configuration
	ServerPort int

	// Hasura configuration
	HasuraEndpoint    string
	HasuraAdminSecret string

	// Temporal configuration
	TemporalHostPort  string
	TemporalNamespace string

	// ABAC configuration
	ABACEnabled bool

	// Default values
	DefaultTenantID string
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() (*Config, error) {
	cfg := &Config{
		ServerPort:        getEnvInt("GOVERNANCE_PORT", 8084),
		HasuraEndpoint:    getEnv("HASURA_GRAPHQL_ENDPOINT", "http://localhost:8080/v1/graphql"),
		HasuraAdminSecret: getEnv("HASURA_ADMIN_SECRET", ""),
		TemporalHostPort:  getEnv("TEMPORAL_HOST_PORT", "localhost:7233"),
		TemporalNamespace: getEnv("TEMPORAL_NAMESPACE", "default"),
		ABACEnabled:       getEnvBool("ABAC_ENABLED", true),
		DefaultTenantID:   getEnv("DEFAULT_TENANT_ID", "default"),
	}

	if err := Validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate ensures all required configuration is present
func Validate(cfg *Config) error {
	if cfg.HasuraEndpoint == "" {
		return errors.New("HASURA_GRAPHQL_ENDPOINT is required")
	}

	if cfg.HasuraAdminSecret == "" {
		return errors.New("HASURA_ADMIN_SECRET is required for production")
	}

	if cfg.TemporalHostPort == "" {
		return errors.New("TEMPORAL_HOST_PORT is required")
	}

	if cfg.ServerPort <= 0 || cfg.ServerPort > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.ServerPort)
	}

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

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
