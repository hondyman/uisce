package cbo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

const (
	defaultCacheTTLSeconds = 60
)

type TelemetryCache struct {
	redis RedisClient
	cfg   *CBOConfig
}

func NewTelemetryCache(redis RedisClient, cfg *CBOConfig) *TelemetryCache {
	if cfg == nil {
		cfg = &CBOConfig{CacheTTLSeconds: defaultCacheTTLSeconds}
	}
	if cfg.CacheTTLSeconds == 0 {
		cfg.CacheTTLSeconds = defaultCacheTTLSeconds
	}
	return &TelemetryCache{redis: redis, cfg: cfg}
}

func (c *TelemetryCache) cacheKey(tenantID, boKey string) string {
	return fmt.Sprintf("tenant:%s:cbo:%s", tenantID, boKey)
}

func (c *TelemetryCache) Get(ctx context.Context, tenantID, boKey string) (*TelemetryFlavorDecision, error) {
	if c.redis == nil {
		return nil, nil
	}

	key := c.cacheKey(tenantID, boKey)
	val, err := c.redis.Get(ctx, key)
	if err != nil {
		return nil, nil
	}

	var decision TelemetryFlavorDecision
	if err := json.Unmarshal([]byte(val), &decision); err != nil {
		return nil, err
	}
	return &decision, nil
}

func (c *TelemetryCache) Set(ctx context.Context, decision *TelemetryFlavorDecision) error {
	if c.redis == nil {
		return nil
	}

	key := c.cacheKey(decision.TenantID, decision.BOName)
	data, err := json.Marshal(decision)
	if err != nil {
		return err
	}

	ttl := time.Duration(c.cfg.CacheTTLSeconds) * time.Second
	if err := c.redis.Set(ctx, key, data, ttl); err != nil {
		return err
	}
	return nil
}

func (c *TelemetryCache) IsAvailable(ctx context.Context) bool {
	if c.redis == nil {
		return false
	}
	return c.redis.Ping(ctx) == nil
}

type CBOLogger interface {
	Warn(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Debug(msg string, fields ...zap.Field)
}

type zapLogger struct {
	logger *zap.Logger
}

func NewZapLogger(logger *zap.Logger) CBOLogger {
	return &zapLogger{logger: logger}
}

func (l *zapLogger) Warn(msg string, fields ...zap.Field) {
	l.logger.Warn(msg, fields...)
}

func (l *zapLogger) Info(msg string, fields ...zap.Field) {
	l.logger.Info(msg, fields...)
}

func (l *zapLogger) Debug(msg string, fields ...zap.Field) {
	l.logger.Debug(msg, fields...)
}

type nopLogger struct{}

func NewNopLogger() CBOLogger {
	return &nopLogger{}
}

func (n *nopLogger) Warn(msg string, fields ...zap.Field) {}
func (n *nopLogger) Info(msg string, fields ...zap.Field) {}
func (n *nopLogger) Debug(msg string, fields ...zap.Field) {}
