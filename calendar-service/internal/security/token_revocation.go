package security

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

// TokenRevoker manages token revocation via JTI (JWT ID) tracking
// When a token is revoked (e.g., user logout), its JTI is stored in Redis
// The expiration time of the Redis key should match the token's expiration (exp claim)
type TokenRevoker struct {
	redisClient *redis.Client
	prefix      string
	defaultTTL  time.Duration
	logger      *logrus.Entry
}

// NewTokenRevoker creates a new token revoker instance
func NewTokenRevoker(redisClient *redis.Client, prefix string, defaultTTL time.Duration, logger *logrus.Entry) *TokenRevoker {
	return &TokenRevoker{
		redisClient: redisClient,
		prefix:      prefix,
		defaultTTL:  defaultTTL,
		logger:      logger.WithField("component", "token_revoker"),
	}
}

// key formats the Redis key for a JTI
func (r *TokenRevoker) key(jti string) string {
	return r.prefix + ":revoked:" + jti
}

// userKey formats the Redis key for all JTIs for a user (for batch revocation)
func (r *TokenRevoker) userKey(userID string) string {
	return r.prefix + ":user_tokens:" + userID
}

// Revoke marks a token as revoked with optional TTL
// ttl: if 0, uses defaultTTL (typically matches token's exp claim)
func (r *TokenRevoker) Revoke(ctx context.Context, jti string, userID string, ttl time.Duration) error {
	if ttl == 0 {
		ttl = r.defaultTTL
	}

	// Store revocation flag
	if err := r.redisClient.Set(ctx, r.key(jti), "revoked", ttl).Err(); err != nil {
		r.logger.WithError(err).WithField("jti", jti).Error("Failed to revoke token")
		return err
	}

	// Store JTI under user key for batch revocation
	if userID != "" {
		_ = r.redisClient.SAdd(ctx, r.userKey(userID), jti).Err()
		// Set expiration on user key
		_ = r.redisClient.Expire(ctx, r.userKey(userID), ttl).Err()
	}

	r.logger.WithField("jti", jti).WithField("user_id", userID).Debug("Token revoked")
	return nil
}

// IsRevoked checks if a token has been revoked
func (r *TokenRevoker) IsRevoked(ctx context.Context, jti string) (bool, error) {
	val, err := r.redisClient.Get(ctx, r.key(jti)).Result()
	if err == redis.Nil {
		// Key not found = not revoked
		return false, nil
	}
	if err != nil {
		r.logger.WithError(err).WithField("jti", jti).Warn("Failed to check revocation status")
		return false, err
	}
	return val == "revoked", nil
}

// RevokeAllForUser revokes all tokens for a user (e.g., on password change)
func (r *TokenRevoker) RevokeAllForUser(ctx context.Context, userID string, jtis []string) error {
	if len(jtis) == 0 {
		r.logger.WithField("user_id", userID).Debug("No tokens to revoke")
		return nil
	}

	// Mark all JTIs as revoked with extended TTL
	ttl := r.defaultTTL
	for _, jti := range jtis {
		if err := r.redisClient.Set(ctx, r.key(jti), "revoked", ttl).Err(); err != nil {
			r.logger.WithError(err).WithField("user_id", userID).Warn("Failed to revoke user token")
		}
	}

	// Clear user's token list
	if err := r.redisClient.Del(ctx, r.userKey(userID)).Err(); err != nil {
		r.logger.WithError(err).WithField("user_id", userID).Warn("Failed to clear user token list")
	}

	r.logger.WithField("user_id", userID).WithField("count", len(jtis)).Info("All user tokens revoked")
	return nil
}

// GetRevokedCount returns count of revoked tokens for monitoring
func (r *TokenRevoker) GetRevokedCount(ctx context.Context) (int64, error) {
	// This is an approximation - count keys matching the revoked pattern
	// For production, consider using a separate counter maintenance routine
	keys, err := r.redisClient.Keys(ctx, r.prefix+":revoked:*").Result()
	if err != nil {
		r.logger.WithError(err).Warn("Failed to count revoked tokens")
		return 0, err
	}
	return int64(len(keys)), nil
}

// HealthCheck verifies Redis connectivity for token revocation
func (r *TokenRevoker) HealthCheck(ctx context.Context) error {
	// Test pingthrough the Redis connection
	return r.redisClient.Ping(ctx).Err()
}
