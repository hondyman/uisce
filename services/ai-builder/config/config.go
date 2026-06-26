package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

// Config holds all ai-builder service configuration
type Config struct {
	// Server configuration
	ServerPort int

	// xAI API configuration
	XAIAPIKey      string
	XAIAPIEndpoint string
	XAIModel       string

	// Temporal configuration
	TemporalHostPort  string
	TemporalNamespace string
	TemporalTaskQueue string

	// ABAC configuration
	ABACEnabled bool

	// Workflow configuration
	WorkflowTimeout  int // seconds
	DefaultTaskQueue string
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() (*Config, error) {
	cfg := &Config{
		ServerPort:        getEnvInt("AI_BUILDER_PORT", 8082),
		XAIAPIKey:         getEnv("XAI_API_KEY", ""),
		XAIAPIEndpoint:    getEnv("XAI_API_ENDPOINT", "https://api.x.ai/v1"),
		XAIModel:          getEnv("XAI_MODEL", "grok-beta"),
		TemporalHostPort:  getEnv("TEMPORAL_HOST_PORT", "localhost:7233"),
		TemporalNamespace: getEnv("TEMPORAL_NAMESPACE", "default"),
		TemporalTaskQueue: getEnv("TEMPORAL_TASK_QUEUE", "ai-builder"),
		ABACEnabled:       getEnvBool("ABAC_ENABLED", false),
		WorkflowTimeout:   getEnvInt("WORKFLOW_TIMEOUT", 300),
		DefaultTaskQueue:  getEnv("DEFAULT_TASK_QUEUE", "ai-builder"),
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

	if cfg.XAIAPIEndpoint == "" {
		return errors.New("XAI_API_ENDPOINT is required")
	}

	if cfg.XAIModel == "" {
		return errors.New("XAI_MODEL is required")
	}

	// XAI API Key is required for production but we allow running without it for development/testing
	if cfg.XAIAPIKey == "" {
		// Log warning but don't fail - will use mock responses
		fmt.Println("WARNING: XAI_API_KEY is not set - using mock responses")
	}

	if cfg.TemporalHostPort == "" {
		return errors.New("TEMPORAL_HOST_PORT is required")
	}

	return nil
}

// IsMockMode returns true if we should use mock responses (no API key)
func (cfg *Config) IsMockMode() bool {
	return cfg.XAIAPIKey == ""
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
