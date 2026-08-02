package api

import (
	"context"
	"time"
)

const idempTTL = 24 * time.Hour

func idempKey(tenantID, idempHeader, fallbackID string) string {
	if idempHeader != "" {
		return "idemp:" + tenantID + ":" + idempHeader
	}
	return "idemp:" + tenantID + ":order:" + fallbackID
}

func (h *ExternalComplianceHandler) checkIdempotency(ctx context.Context, key string) ([]byte, bool) {
	if key == "" || h.redisClient == nil {
		return nil, false
	}
	val, err := h.redisClient.Get(ctx, key).Bytes()
	if err == nil && len(val) > 0 {
		return val, true
	}
	return nil, false
}

func (h *ExternalComplianceHandler) saveIdempotency(ctx context.Context, key string, response []byte) {
	if key == "" || h.redisClient == nil {
		return
	}
	h.redisClient.Set(ctx, key, response, idempTTL)
}
