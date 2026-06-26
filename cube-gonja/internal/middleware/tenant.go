package middleware

import (
	"net/http"
	"strings"

	"cube-gonja/config"
	"cube-gonja/internal/tenant"

	"github.com/gin-gonic/gin"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

type TenantContext struct {
	TenantID string
	Tenant   *tenant.Tenant
}

// GinTenantAuthMiddleware extracts tenant information from headers or URL for Gin
func GinTenantAuthMiddleware(cfg config.Config, tenantMgr *tenant.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tenantID string
		var apiKey string

		// Extract tenant ID from header, query param, or URL path
		if tid := jwtmiddleware.GetGinClaimsFromContext(c).TenantID; tid != "" {
			tenantID = tid
		} else if tid := c.Query("tenant"); tid != "" {
			tenantID = tid
		} else if cfg.EnableMultiTenant {
			// Try to extract from URL path (e.g., /tenant/{tenantID}/endpoint)
			pathParts := strings.Split(strings.Trim(c.Request.URL.Path, "/"), "/")
			if len(pathParts) >= 2 && pathParts[0] == "tenant" {
				tenantID = pathParts[1]
				// Remove tenant prefix from URL for further processing
				c.Request.URL.Path = "/" + strings.Join(pathParts[2:], "/")
			}
		}

		// Default tenant if not specified
		if tenantID == "" {
			tenantID = cfg.DefaultTenant
		}

		// Extract API key
		if key := c.GetHeader("X-API-Key"); key != "" {
			apiKey = key
		} else if key := c.Query("api_key"); key != "" {
			apiKey = key
		}

		// Validate tenant access
		if cfg.RequireAuth || cfg.EnableMultiTenant {
			if !tenantMgr.ValidateTenantAccess(tenantID, apiKey) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Invalid tenant or API key"})
				c.Abort()
				return
			}
		}

		// Get or create tenant
		tenant, err := tenantMgr.GetTenant(tenantID)
		if err != nil {
			// Try to initialize the tenant
			tenant, err = tenantMgr.InitializeTenant(tenantID, tenantID, apiKey)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize tenant: " + err.Error()})
				c.Abort()
				return
			}
		}

		// Add tenant context to Gin context
		c.Set("semlayer_tenant", &TenantContext{
			TenantID: tenantID,
			Tenant:   tenant,
		})

		c.Next()
	}
}

// GetTenantFromGinContext extracts tenant information from Gin context
func GetTenantFromGinContext(c *gin.Context) (*TenantContext, bool) {
	if ctx, exists := c.Get("semlayer_tenant"); exists {
		if tenantCtx, ok := ctx.(*TenantContext); ok {
			return tenantCtx, true
		}
	}
	return nil, false
}
