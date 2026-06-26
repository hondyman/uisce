package mdm

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// InitializeClient creates and initializes the MDM client
func InitializeClient(config *Config, logger *logrus.Logger) (*Client, error) {
	if !config.Enabled {
		logger.Info("MDM client initialization skipped (disabled)")
		return nil, nil
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid MDM configuration: %w", err)
	}

	entry := logger.WithFields(logrus.Fields{
		"component": "mdm",
		"service":   config.BaseURL,
	})

	// Create HTTP client
	client := NewClient(
		config.BaseURL,
		config.Timeout,
		entry,
	)

	entry.WithField("config", config).Info("MDM client initialized successfully")

	// Verify MDM service is reachable (non-blocking)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.Health(ctx); err != nil {
			entry.WithError(err).Warn("MDM service health check failed on startup")
		} else {
			entry.Debug("MDM service health check passed")
		}
	}()

	return client, nil
}
