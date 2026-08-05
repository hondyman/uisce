package database

import (
	"context"
	"fmt"
	"os"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func ConfigFromEnv(prefix string) *Config {
	return &Config{
		Host:     getEnv(prefix+"_HOST", "localhost"),
		Port:     getEnv(prefix+"_PORT", "5432"),
		User:     getEnv(prefix+"_USER", "postgres"),
		Password: getEnv(prefix+"_PASSWORD", ""),
		DBName:   getEnv(prefix+"_DATABASE", "postgres"),
		SSLMode:  getEnv(prefix+"_SSLMODE", "disable"),
	}
}

func (c *Config) ConnectionString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode,
	)
}

func (c *Config) Connect(ctx context.Context) (*Pool, error) {
	return NewPool(ctx, c.ConnectionString())
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
