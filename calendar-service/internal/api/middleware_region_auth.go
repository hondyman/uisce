package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"calendar-service/internal/database"

	"github.com/sirupsen/logrus"
)

func RegionAuthMiddleware(dbClient *database.Client, logger *logrus.Entry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			tenantID := r.Header.Get("X-Hasura-Tenant-Id")
			if tenantID == "" {
				http.Error(w, "Missing X-Hasura-Tenant-Id header", http.StatusUnauthorized)
				return
			}

			region := r.URL.Query().Get("region")

			if region == "" && (r.Method == "POST" || r.Method == "PATCH" || r.Method == "PUT") {
				bodyBytes, err := io.ReadAll(r.Body)
				if err == nil {
					var reqBody map[string]interface{}
					if err := json.Unmarshal(bodyBytes, &reqBody); err == nil {
						if r, ok := reqBody["region"].(string); ok && r != "" {
							region = r
						}
					}
					r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
				}
			}

			if region == "" {
				region = "us-east-1"
			}

			logger := logger.WithFields(logrus.Fields{
				"tenant_id": tenantID,
				"region":    region,
				"path":      r.RequestURI,
			})

			authorized, err := validateTenantRegion(ctx, dbClient, tenantID, region)
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

			logger.Debug("Region authorization successful")

			ctx = context.WithValue(ctx, contextKeyRegion, region)
			ctx = context.WithValue(ctx, contextKeyTenantID, tenantID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func validateTenantRegion(ctx context.Context, dc *database.Client, tenantID, region string) (bool, error) {
	if tenantID == "" {
		return false, fmt.Errorf("tenant_id required")
	}
	if region == "" {
		return false, fmt.Errorf("region required")
	}

	query := `SELECT COUNT(*) FROM tenant_regions WHERE tenant_id = $1 AND region = $2 LIMIT 1`
	var count int
	if err := dc.Pool().QueryRow(ctx, query, tenantID, region).Scan(&count); err != nil {
		return false, fmt.Errorf("region auth query failed: %w", err)
	}
	return count > 0, nil
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
