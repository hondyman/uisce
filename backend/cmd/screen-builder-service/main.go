package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// ============================================================================
// MODELS
// ============================================================================

// ScreenConfig represents a screen configuration
type ScreenConfig struct {
	ID            string          `json:"id"`
	TenantID      string          `json:"tenant_id"`
	BOType        string          `json:"bo_type"`
	ScreenName    string          `json:"screen_name"`
	ScreenType    string          `json:"screen_type"` // "detail", "list", "create", "edit"
	LayoutJSON    json.RawMessage `json:"layout_json"`
	FiltersJSON   json.RawMessage `json:"filters_json"`
	ActionsJSON   json.RawMessage `json:"actions_json"`
	PermissionsJSON json.RawMessage `json:"permissions_json"`
	IsPublished   bool            `json:"is_published"`
	CreatedAt     string          `json:"created_at"`
}

// ScreenField represents a single field in a screen layout
type ScreenField struct {
	Field      string `json:"field"`
	Label      string `json:"label"`
	Type       string `json:"type"` // "text", "number", "date", "select", "textarea"
	Order      int    `json:"order"`
	Required   bool   `json:"required"`
	Searchable bool   `json:"searchable"`
	Editable   bool   `json:"editable"`
}

// CreateScreenRequest for API
type CreateScreenRequest struct {
	TenantID    string              `json:"tenant_id" binding:"required"`
	BOType      string              `json:"bo_type" binding:"required"`
	ScreenName  string              `json:"screen_name" binding:"required"`
	ScreenType  string              `json:"screen_type" binding:"required"`
	Fields      []ScreenField       `json:"fields"`
	Filters     []ScreenField       `json:"filters"`
	Actions     []string            `json:"actions"`
	Permissions map[string][]string `json:"permissions"`
	UserID      string              `json:"user_id" binding:"required"`
}

// ============================================================================
// GLOBAL CLIENTS
// ============================================================================

var (
	db *sql.DB
)

// ============================================================================
// INITIALIZATION
// ============================================================================

func init() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		panic("DATABASE_URL environment variable is required")
	}

	var err error
	db, err = sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("✓ Screen Builder Service initialized")
	log.Printf("  Database: connected\n")
}

// ============================================================================
// MAIN
// ============================================================================

func main() {
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Create screen
	r.POST("/screens", createScreen)

	// Get screens for BO type
	r.GET("/screens/:tenant_id/:bo_type", listScreens)

	// Get single screen
	// Use a non-ambiguous path to avoid wildcard conflicts with the BO-type list route
	r.GET("/screens/:tenant_id/screen/:screen_id", getScreen)

	// Update screen
	r.PUT("/screens/:tenant_id/screen/:screen_id", updateScreen)

	// Delete screen
	r.DELETE("/screens/:tenant_id/screen/:screen_id", deleteScreen)

	// Publish screen
	r.POST("/screens/:tenant_id/screen/:screen_id/publish", publishScreen)

	port := os.Getenv("SCREEN_BUILDER_SERVICE_PORT")
	if port == "" {
		port = "8083"
	}

	log.Printf("Screen Builder Service listening on :%s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// ============================================================================
// HANDLERS
// ============================================================================

// createScreen creates a new screen configuration
func createScreen(c *gin.Context) {
	var req CreateScreenRequest

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	screenID := uuid.New().String()

	layoutJSON, _ := json.Marshal(req.Fields)
	filtersJSON, _ := json.Marshal(req.Filters)
	actionsJSON, _ := json.Marshal(req.Actions)
	permissionsJSON, _ := json.Marshal(req.Permissions)

	query := `
		INSERT INTO screen_configs (id, tenant_id, bo_type, screen_name, screen_type, layout_json, filters_json, actions_json, permissions_json, is_published, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, screen_name, bo_type, created_at
	`

	var result struct {
		ID        string    `json:"id"`
		ScreenName string   `json:"screen_name"`
		BOType    string    `json:"bo_type"`
		CreatedAt time.Time `json:"created_at"`
	}

	err := db.QueryRowContext(c.Request.Context(), query,
		screenID, req.TenantID, req.BOType, req.ScreenName, req.ScreenType,
		layoutJSON, filtersJSON, actionsJSON, permissionsJSON, false, req.UserID,
	).Scan(&result.ID, &result.ScreenName, &result.BOType, &result.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create screen: %v", err),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      screenID,
		"message": fmt.Sprintf("Screen %s created successfully", req.ScreenName),
	})
}

// listScreens retrieves all screens for a business object
func listScreens(c *gin.Context) {
	tenantID := c.Param("tenant_id")
	boType := c.Param("bo_type")

	query := `
		SELECT id, screen_name, screen_type, is_published, created_at
		FROM screen_configs
		WHERE tenant_id = $1 AND bo_type = $2
		ORDER BY created_at DESC
	`

	rows, err := db.QueryContext(c.Request.Context(), query, tenantID, boType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to fetch screens: %v", err),
		})
		return
	}
	defer rows.Close()

	var screens []map[string]interface{}
	for rows.Next() {
		var id, screenName, screenType string
		var isPublished bool
		var createdAt time.Time

		if err := rows.Scan(&id, &screenName, &screenType, &isPublished, &createdAt); err != nil {
			continue
		}

		screens = append(screens, map[string]interface{}{
			"id":           id,
			"screen_name":  screenName,
			"screen_type":  screenType,
			"is_published": isPublished,
			"created_at":   createdAt,
		})
	}

	c.JSON(http.StatusOK, screens)
}

// getScreen retrieves a single screen configuration
func getScreen(c *gin.Context) {
	tenantID := c.Param("tenant_id")
	screenID := c.Param("screen_id")

	query := `
		SELECT id, tenant_id, bo_type, screen_name, screen_type, layout_json, filters_json, actions_json, permissions_json, is_published, created_at
		FROM screen_configs
		WHERE tenant_id = $1 AND id = $2
	`

	var screen ScreenConfig
	var layoutJSON, filtersJSON, actionsJSON, permissionsJSON []byte

	err := db.QueryRowContext(c.Request.Context(), query, tenantID, screenID).Scan(
		&screen.ID, &screen.TenantID, &screen.BOType, &screen.ScreenName, &screen.ScreenType,
		&layoutJSON, &filtersJSON, &actionsJSON, &permissionsJSON, &screen.IsPublished, &screen.CreatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Screen not found",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to fetch screen: %v", err),
		})
		return
	}

	screen.LayoutJSON = layoutJSON
	screen.FiltersJSON = filtersJSON
	screen.ActionsJSON = actionsJSON
	screen.PermissionsJSON = permissionsJSON

	c.JSON(http.StatusOK, screen)
}

// updateScreen updates a screen configuration
func updateScreen(c *gin.Context) {
	tenantID := c.Param("tenant_id")
	screenID := c.Param("screen_id")

	var req map[string]interface{}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	query := `UPDATE screen_configs SET `
	args := []interface{}{}
	argIdx := 1
	setParts := []string{}

	if layout, ok := req["fields"]; ok {
		layoutJSON, _ := json.Marshal(layout)
		setParts = append(setParts, fmt.Sprintf("layout_json = $%d", argIdx))
		args = append(args, layoutJSON)
		argIdx++
	}
	if filters, ok := req["filters"]; ok {
		filtersJSON, _ := json.Marshal(filters)
		setParts = append(setParts, fmt.Sprintf("filters_json = $%d", argIdx))
		args = append(args, filtersJSON)
		argIdx++
	}
	if actions, ok := req["actions"]; ok {
		actionsJSON, _ := json.Marshal(actions)
		setParts = append(setParts, fmt.Sprintf("actions_json = $%d", argIdx))
		args = append(args, actionsJSON)
		argIdx++
	}
	if permissions, ok := req["permissions"]; ok {
		permissionsJSON, _ := json.Marshal(permissions)
		setParts = append(setParts, fmt.Sprintf("permissions_json = $%d", argIdx))
		args = append(args, permissionsJSON)
		argIdx++
	}
	if screenName, ok := req["screen_name"]; ok {
		setParts = append(setParts, fmt.Sprintf("screen_name = $%d", argIdx))
		args = append(args, screenName)
		argIdx++
	}

	if len(setParts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No fields to update",
		})
		return
	}

	for i, part := range setParts {
		if i > 0 {
			query += ", "
		}
		query += part
	}

	query += fmt.Sprintf(" WHERE tenant_id = $%d AND id = $%d", argIdx, argIdx+1)
	args = append(args, tenantID, screenID)

	_, err := db.ExecContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to update screen: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Screen updated successfully",
	})
}

// deleteScreen deletes a screen configuration
func deleteScreen(c *gin.Context) {
	tenantID := c.Param("tenant_id")
	screenID := c.Param("screen_id")

	query := `DELETE FROM screen_configs WHERE tenant_id = $1 AND id = $2`

	_, err := db.ExecContext(c.Request.Context(), query, tenantID, screenID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to delete screen: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Screen deleted successfully",
	})
}

// publishScreen publishes a screen
func publishScreen(c *gin.Context) {
	tenantID := c.Param("tenant_id")
	screenID := c.Param("screen_id")

	query := `UPDATE screen_configs SET is_published = true WHERE tenant_id = $1 AND id = $2`

	_, err := db.ExecContext(c.Request.Context(), query, tenantID, screenID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to publish screen: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Screen published successfully",
	})
}
