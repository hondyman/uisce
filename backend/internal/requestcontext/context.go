package requestcontext

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

// RequestContext holds request-scoped information
type RequestContext struct {
	RequestID string
	UserID    string
	TenantID  string
	IPAddress string
	UserAgent string
	StartTime time.Time
}

// typed key for storing RequestContext in standard context
type reqCtxKey string

const requestContextKey reqCtxKey = "semlayer_request_context"

// GetRequestContext retrieves the request context from the standard request context
func GetRequestContext(ctx context.Context) *RequestContext {
	if v := ctx.Value(requestContextKey); v != nil {
		if rc, ok := v.(*RequestContext); ok {
			return rc
		}
	}
	return nil
}

// GetRequestContextFromRequest retrieves the request context from the http request
func GetRequestContextFromRequest(r *http.Request) *RequestContext {
	return GetRequestContext(r.Context())
}

// WithRequestContext adds the request context to the context
func WithRequestContext(ctx context.Context, rc *RequestContext) context.Context {
	return context.WithValue(ctx, requestContextKey, rc)
}

// GenerateRequestID creates a unique request identifier
func GenerateRequestID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
