package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"calendar-service/internal/hasura"

	"github.com/sirupsen/logrus"
)

// RegionAuthMiddleware validates tenant can access requested region
// Enforces data residency compliance
func RegionAuthMiddleware(hasuraClient *hasura.Client, logger *logrus.Entry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Extract tenant ID from header
			tenantID := r.Header.Get("X-Hasura-Tenant-Id")
			if tenantID == "" {
				http.Error(w, "Missing X-Hasura-Tenant-Id header", http.StatusUnauthorized)
				return
			}

			// Extract region from query parameter or request body
			region := r.URL.Query().Get("region")

			// If not in query, try to extract from body (for POST/PATCH)
			if region == "" && (r.Method == "POST" || r.Method == "PATCH" || r.Method == "PUT") {
				// Read body
				bodyBytes, err := io.ReadAll(r.Body)
				if err == nil {
					// Parse JSON to find region
					var reqBody map[string]interface{}
					if err := json.Unmarshal(bodyBytes, &reqBody); err == nil {
						if r, ok := reqBody["region"].(string); ok && r != "" {
							region = r
						}
					}
					// Restore body for next handler
					r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
				}
			}

			// Default to us-east-1 if not provided
			if region == "" {
				region = "us-east-1"
			}

			logger := logger.WithFields(logrus.Fields{
				"tenant_id": tenantID,
				"region":    region,
				"path":      r.RequestURI,
			})

			// Validate tenant region authorization
			authorized, err := validateTenantRegion(ctx, hasuraClient, tenantID, region)
			if err != nil {
				logger.WithError(err).Error("Failed to validate region authorization")
				http.Error(w, "Authorization check failed", http.StatusInternalServerError)
				return
			}

			if !authorized {
				logger.Warn("Unauthorized region access attempted")
				http.Error(
					w,
					fmt.Sprintf("Tenant not authorized for region %s", region),
					http.StatusForbidden,
				)
				return
			}

			logger.Debug("ℹ️ Region authorization successful")

			// Add region to context for downstream handlers
			ctx = context.WithValue(ctx, contextKeyRegion, region)
			ctx = context.WithValue(ctx, contextKeyTenantID, tenantID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ValidateTenantRegion checks if tenant is authorized for region
func validateTenantRegion(ctx context.Context, hc *hasura.Client, tenantID, region string) (bool, error) {
	if tenantID == "" {
		return false, fmt.Errorf("tenant_id required")
	}
	if region == "" {
		return false, fmt.Errorf("region required")
	}

	// Actual Hasura query to verify tenant-region mapping
	var result struct {
		TenantRegions []struct {
			TenantID string `json:"tenant_id"`
		} `json:"tenant_regions"`
	}

	query := `
	query GetTenantRegion($tenant_id: String!, $region: String!) {
		tenant_regions(
			where: {
				tenant_id: {_eq: $tenant_id},
				region: {_eq: $region}
			},
			limit: 1
		) {
			tenant_id
		}
	}
	`

	if err := hc.QueryRaw(ctx, query, map[string]interface{}{
		"tenant_id": tenantID,
		"region":    region,
	}, &result); err != nil {
		return false, fmt.Errorf("region auth query failed: %w", err)
	}

	return len(result.TenantRegions) > 0, nil
}

// GetRegionFromContext extracts region from request context
func GetRegionFromContext(ctx context.Context) string {
	if region, ok := ctx.Value(contextKeyRegion).(string); ok && region != "" {
		return region
	}
	return "us-east-1" // Default fallback
}

// GetTenantFromContext extracts tenant ID from request context
func GetTenantFromContext(ctx context.Context) string {
	if tenant, ok := ctx.Value(contextKeyTenantID).(string); ok && tenant != "" {
		return tenant
	}
	return ""
}

// Context key constants
const (
	contextKeyRegion   = "region"
	contextKeyTenantID = "tenant_id"
)
