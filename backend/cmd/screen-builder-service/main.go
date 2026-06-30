package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ============================================================================
// MODELS
// ============================================================================

// ScreenConfig represents a screen configuration
type ScreenConfig struct {
	ID          string          `json:"id"`
	TenantID    string          `json:"tenant_id"`
	BOType      string          `json:"bo_type"`
	ScreenName  string          `json:"screen_name"`
	ScreenType  string          `json:"screen_type"` // "detail", "list", "create", "edit"
	LayoutJSON  json.RawMessage `json:"layout_json"`
	FiltersJSON json.RawMessage `json:"filters_json"`
	ActionsJSON json.RawMessage `json:"actions_json"`
	Permissions json.RawMessage `json:"permissions_json"`
	IsPublished bool            `json:"is_published"`
	CreatedAt   string          `json:"created_at"`
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
	hasuraURL   string
	hasuraToken string
)

// ============================================================================
// INITIALIZATION
// ============================================================================

func init() {
	hasuraURL = os.Getenv("HASURA_URL")
	if hasuraURL == "" {
		hasuraURL = "http://localhost:8080"
	}
	hasuraToken = os.Getenv("HASURA_ADMIN_SECRET")

	log.Println("✓ Screen Builder Service initialized")
	log.Printf("  Hasura: %s\n", hasuraURL)
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

	// Build layout JSON
	layoutJSON, _ := json.Marshal(req.Fields)
	filtersJSON, _ := json.Marshal(req.Filters)
	actionsJSON, _ := json.Marshal(req.Actions)
	permissionsJSON, _ := json.Marshal(req.Permissions)

	query := `
		mutation CreateScreen($object: screen_configs_insert_input!) {
			insert_screen_configs_one(object: $object) {
				id
				screen_name
				bo_type
				created_at
			}
		}
	`

	object := map[string]interface{}{
		"id":               screenID,
		"tenant_id":        req.TenantID,
		"bo_type":          req.BOType,
		"screen_name":      req.ScreenName,
		"screen_type":      req.ScreenType,
		"layout_json":      json.RawMessage(layoutJSON),
		"filters_json":     json.RawMessage(filtersJSON),
		"actions_json":     json.RawMessage(actionsJSON),
		"permissions_json": json.RawMessage(permissionsJSON),
		"is_published":     false,
		"created_by":       req.UserID,
	}

	variables := map[string]interface{}{
		"object": object,
	}

	data, err := hasuraGraphQLQuery(c.Request.Context(), query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create screen: %v", err),
		})
		return
	}

	var resp struct {
		InsertScreenConfigsOne ScreenConfig `json:"insert_screen_configs_one"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to parse response: %v", err),
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
		query ListScreens($tenantID: uuid!, $boType: String!) {
			screen_configs(
				where: {tenant_id: {_eq: $tenantID}, bo_type: {_eq: $boType}}
				order_by: {created_at: desc}
			) {
				id
				screen_name
				screen_type
				is_published
				created_at
			}
		}
	`

	variables := map[string]interface{}{
		"tenantID": tenantID,
		"boType":   boType,
	}

	data, err := hasuraGraphQLQuery(c.Request.Context(), query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to fetch screens: %v", err),
		})
		return
	}

	var resp struct {
		ScreenConfigs []map[string]interface{} `json:"screen_configs"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to parse screens: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, resp.ScreenConfigs)
}

// getScreen retrieves a single screen configuration
func getScreen(c *gin.Context) {
	tenantID := c.Param("tenant_id")
	screenID := c.Param("screen_id")

	query := `
		query GetScreen($tenantID: uuid!, $screenID: uuid!) {
			screen_configs(
				where: {tenant_id: {_eq: $tenantID}, id: {_eq: $screenID}}
			) {
				id
				screen_name
				bo_type
				screen_type
				layout_json
				filters_json
				actions_json
				permissions_json
				is_published
				created_at
			}
		}
	`

	variables := map[string]interface{}{
		"tenantID": tenantID,
		"screenID": screenID,
	}

	data, err := hasuraGraphQLQuery(c.Request.Context(), query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to fetch screen: %v", err),
		})
		return
	}

	var resp struct {
		ScreenConfigs []ScreenConfig `json:"screen_configs"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to parse screen: %v", err),
		})
		return
	}

	if len(resp.ScreenConfigs) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Screen not found",
		})
		return
	}

	c.JSON(http.StatusOK, resp.ScreenConfigs[0])
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

	// Build update payload
	updates := map[string]interface{}{}

	if layout, ok := req["fields"]; ok {
		layoutJSON, _ := json.Marshal(layout)
		updates["layout_json"] = layoutJSON
	}
	if filters, ok := req["filters"]; ok {
		filtersJSON, _ := json.Marshal(filters)
		updates["filters_json"] = filtersJSON
	}
	if actions, ok := req["actions"]; ok {
		actionsJSON, _ := json.Marshal(actions)
		updates["actions_json"] = actionsJSON
	}
	if permissions, ok := req["permissions"]; ok {
		permissionsJSON, _ := json.Marshal(permissions)
		updates["permissions_json"] = permissionsJSON
	}
	if screenName, ok := req["screen_name"]; ok {
		updates["screen_name"] = screenName
	}

	query := `
		mutation UpdateScreen($tenantID: uuid!, $screenID: uuid!, $updates: screen_configs_set_input!) {
			update_screen_configs(
				where: {tenant_id: {_eq: $tenantID}, id: {_eq: $screenID}}
				_set: $updates
			) {
				affected_rows
			}
		}
	`

	variables := map[string]interface{}{
		"tenantID": tenantID,
		"screenID": screenID,
		"updates":  updates,
	}

	_, err := hasuraGraphQLQuery(c.Request.Context(), query, variables)
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

	query := `
		mutation DeleteScreen($tenantID: uuid!, $screenID: uuid!) {
			delete_screen_configs(
				where: {tenant_id: {_eq: $tenantID}, id: {_eq: $screenID}}
			) {
				affected_rows
			}
		}
	`

	variables := map[string]interface{}{
		"tenantID": tenantID,
		"screenID": screenID,
	}

	_, err := hasuraGraphQLQuery(c.Request.Context(), query, variables)
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

	query := `
		mutation PublishScreen($tenantID: uuid!, $screenID: uuid!) {
			update_screen_configs(
				where: {tenant_id: {_eq: $tenantID}, id: {_eq: $screenID}}
				_set: {is_published: true}
			) {
				affected_rows
			}
		}
	`

	variables := map[string]interface{}{
		"tenantID": tenantID,
		"screenID": screenID,
	}

	_, err := hasuraGraphQLQuery(c.Request.Context(), query, variables)
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

// ============================================================================
// HELPERS
// ============================================================================

func restQuery(ctx context.Context, endpoint string, method string, params map[string]string, body interface{}) (json.RawMessage, error) {
	gatewayURL := os.Getenv("BACKEND_URL")
	if gatewayURL == "" {
		gatewayURL = "http://localhost:8080"
	}
	url := gatewayURL + "/api/rest/" + endpoint
	if len(params) > 0 {
		var parts []string
		for k, v := range params {
			parts = append(parts, k+"="+v)
		}
		url += "?" + strings.Join(parts, "&")
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("REST request failed with status %d: %s", resp.StatusCode, string(b))
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}

// hasuraGraphQLQuery maps the original Hasura GraphQL queries to standard REST queries
func hasuraGraphQLQuery(ctx context.Context, query string, variables map[string]interface{}) (json.RawMessage, error) {
	if strings.Contains(query, "insert_screen_configs_one") {
		obj := variables["object"]
		res, err := restQuery(ctx, "screen-configs", "POST", nil, obj)
		if err != nil {
			return nil, err
		}
		return json.RawMessage(fmt.Sprintf(`{"insert_screen_configs_one": %s}`, string(res))), nil
	}

	if strings.Contains(query, "update_screen_configs") {
		updates := variables["updates"]
		tenantID := variables["tenantID"]
		screenID := variables["screenID"]
		body := map[string]interface{}{
			"tenant_id": tenantID,
			"screen_id": screenID,
			"updates":   updates,
		}
		res, err := restQuery(ctx, "screen-configs", "PUT", nil, body)
		if err != nil {
			return nil, err
		}
		var rowsResp struct {
			AffectedRows int `json:"affected_rows"`
		}
		json.Unmarshal(res, &rowsResp)
		return json.RawMessage(fmt.Sprintf(`{"update_screen_configs": {"affected_rows": %d}}`, rowsResp.AffectedRows)), nil
	}

	if strings.Contains(query, "delete_screen_configs") {
		tenantID := fmt.Sprintf("%v", variables["tenantID"])
		screenID := fmt.Sprintf("%v", variables["screenID"])
		params := map[string]string{
			"tenant_id": tenantID,
			"screen_id": screenID,
		}
		res, err := restQuery(ctx, "screen-configs", "DELETE", params, nil)
		if err != nil {
			return nil, err
		}
		var rowsResp struct {
			AffectedRows int `json:"affected_rows"`
		}
		json.Unmarshal(res, &rowsResp)
		return json.RawMessage(fmt.Sprintf(`{"delete_screen_configs": {"affected_rows": %d}}`, rowsResp.AffectedRows)), nil
	}

	if strings.Contains(query, "screen_configs") {
		tenantID := fmt.Sprintf("%v", variables["tenantID"])
		params := map[string]string{
			"tenant_id": tenantID,
		}
		if boType, ok := variables["boType"]; ok {
			params["bo_type"] = fmt.Sprintf("%v", boType)
		}
		if screenID, ok := variables["screenID"]; ok {
			params["id"] = fmt.Sprintf("%v", screenID)
		}
		res, err := restQuery(ctx, "screen-configs", "GET", params, nil)
		if err != nil {
			return nil, err
		}
		return json.RawMessage(fmt.Sprintf(`{"screen_configs": %s}`, string(res))), nil
	}

	return nil, fmt.Errorf("unhandled query in REST bridge: %s", query)
}
