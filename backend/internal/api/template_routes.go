package api

import (
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// Template Routes Registration
// ============================================================================

// RegisterTemplateRoutes registers all template-related API endpoints
// Call this function during API initialization in main api.go setup
func RegisterTemplateRoutes(router *gin.Engine, store *TemplateStore, rbac *TemplateRBAC) {
	handler := NewTemplateHandlerWithRBAC(store, rbac)

	templateGroup := router.Group("/api/semantic/templates")
	{
		// Template CRUD operations (wrap chi-style handlers)
		templateGroup.POST("/", func(c *gin.Context) { handler.HandleCreateTemplate(c.Writer, c.Request) })
		templateGroup.GET("", func(c *gin.Context) { handler.HandleListTemplates(c.Writer, c.Request) })
		templateGroup.GET("/:id", func(c *gin.Context) { handler.HandleGetTemplate(c.Writer, c.Request) })
		templateGroup.PUT("/:id", func(c *gin.Context) { handler.HandleUpdateTemplate(c.Writer, c.Request) })
		templateGroup.DELETE("/:id", func(c *gin.Context) { handler.HandleDeleteTemplate(c.Writer, c.Request) })

		// Template execution
		templateGroup.POST("/:id/run", func(c *gin.Context) { handler.HandleRunTemplate(c.Writer, c.Request) })

		// Template versioning
		templateGroup.GET("/:id/versions", func(c *gin.Context) { handler.HandleListVersions(c.Writer, c.Request) })
		templateGroup.GET("/:id/versions/:versionNumber", func(c *gin.Context) { handler.HandleGetVersion(c.Writer, c.Request) })
		templateGroup.POST("/:id/diff", func(c *gin.Context) { handler.HandleDiffVersions(c.Writer, c.Request) })
		templateGroup.POST("/:id/promote", func(c *gin.Context) { handler.HandlePromoteVersion(c.Writer, c.Request) })

		// Template permissions
		templateGroup.POST("/:id/permissions", func(c *gin.Context) { handler.SetPermissions(c.Writer, c.Request) })
		templateGroup.GET("/:id/permissions", func(c *gin.Context) { handler.GetPermissions(c.Writer, c.Request) })
	}
}

// ============================================================================
// Integration with Main API
// ============================================================================

// InitialseTemplateSystem initializes the template system with database, store, and RBAC
// Call this function during application startup
func InitialiseTemplateSystem(db interface{}, cacheLayer interface{}, executor interface{}) (*TemplateStore, *TemplateRBAC, error) {
	// Initialize template store with database connection
	dbConn, ok := db.(*sql.DB)
	if !ok {
		return nil, nil, fmt.Errorf("invalid db connection")
	}
	store := NewTemplateStore(dbConn)

	// Initialize template RBAC
	rbac := NewTemplateRBAC(store)

	return store, rbac, nil
}

// ============================================================================
// Middleware for Template Endpoints
// ============================================================================

// TemplateAuthMiddleware validates user has access to template
func TemplateAuthMiddleware(rbac *TemplateRBAC) gin.HandlerFunc {
	return func(c *gin.Context) {
		templateID := c.Param("id")

		if templateID == "" {
			c.Next()
			return
		}

		// TODO: Implement RBAC check properly. Allowing for now.
		c.Next()
		return
	}
}

// ============================================================================
// Example Usage in main api.go
// ============================================================================

/*
In your main api.go file, add the following during API setup:

func setupAPI(db *sql.DB, cache interface{}, executor interface{}) *gin.Engine {
    router := gin.Default()

    // ... other API setup ...

    // Initialize template system
    templateStore, templateRBAC, err := InitialiseTemplateSystem(db, cache, executor)
    if err != nil {
        log.Fatalf("Failed to initialize templates: %v", err)
    }

    // Register template routes
    RegisterTemplateRoutes(router, templateStore, templateRBAC)

    // ... other route setup ...

    return router
}
*/

// ============================================================================
// NewTemplateHandler creates a new template request handler
// ============================================================================

func NewTemplateHandlerWithRBAC(store *TemplateStore, rbac *TemplateRBAC) *TemplateHandler {
	return &TemplateHandler{
		store: store,
		rbac:  rbac,
	}
}
