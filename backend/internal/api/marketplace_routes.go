package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/logging"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ============================================================================
// Marketplace Item Models
// ============================================================================

type MarketplaceItem struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	Description          string          `json:"description"`
	ItemType             string          `json:"item_type"` // 'rule' or 'calculation'
	Version              string          `json:"version"`
	Category             string          `json:"category"`
	Subcategories        []string        `json:"subcategories"`
	Severity             *string         `json:"severity"` // For rules
	IconEmoji            string          `json:"icon_emoji"`
	ColorHex             string          `json:"color_hex"`
	Summary              string          `json:"summary"`
	LongDescription      string          `json:"long_description"`
	ImplementationJSON   json.RawMessage `json:"implementation_json"`
	Scope                string          `json:"scope"`
	RuleType             string          `json:"rule_type"`
	Frequency            string          `json:"frequency"`
	EvaluationOrder      int             `json:"evaluation_order"`
	IsPublic             bool            `json:"is_public"`
	IsOfficial           bool            `json:"is_official"`
	IsCore               bool            `json:"is_core"`
	Status               string          `json:"status"`
	ExternalAPIProviders []string        `json:"external_api_providers"`
	RequiresCredentials  bool            `json:"requires_credentials"`
	UsageCount           int             `json:"usage_count"`
	Rating               *float64        `json:"rating"`
	DownloadsCount       int             `json:"downloads_count"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
	PublishedAt          *time.Time      `json:"published_at"`
}

type TenantMarketplaceItem struct {
	ID                            string          `json:"id"`
	TenantID                      string          `json:"tenant_id"`
	MarketplaceItemID             string          `json:"marketplace_item_id"`
	CustomName                    *string         `json:"custom_name"`
	CustomParameters              json.RawMessage `json:"custom_parameters"`
	EnabledForTenant              bool            `json:"enabled_for_tenant"`
	AddedAt                       time.Time       `json:"added_at"`
	LastUsedAt                    *time.Time      `json:"last_used_at"`
	UsageCount                    int             `json:"usage_count"`
	MarketplaceVersionAtTimeOfAdd string          `json:"marketplace_version_at_time_of_add"`
	LocalVersion                  string          `json:"local_version"`
	HasLocalModifications         bool            `json:"has_local_modifications"`
	TenantRating                  *int            `json:"tenant_rating"`
	TenantFeedback                *string         `json:"tenant_feedback"`
}

type MarketplaceItemParameter struct {
	ID                string          `json:"id"`
	MarketplaceItemID string          `json:"marketplace_item_id"`
	ParamName         string          `json:"param_name"`
	ParamType         string          `json:"param_type"`
	Description       *string         `json:"description"`
	IsRequired        bool            `json:"is_required"`
	DefaultValue      json.RawMessage `json:"default_value"`
	ValidationRules   json.RawMessage `json:"validation_rules"`
	DisplayName       string          `json:"display_name"`
	DisplayOrder      int             `json:"display_order"`
	HelpText          *string         `json:"help_text"`
	CreatedAt         time.Time       `json:"created_at"`
}

type MarketplaceListRequest struct {
	Search       string   `json:"search"`
	ItemType     string   `json:"item_type"` // 'rule', 'calculation', or empty for both
	Categories   []string `json:"categories"`
	Severities   []string `json:"severities"` // For rules
	Status       string   `json:"status"`     // 'active', 'beta', etc.
	OnlyOfficial bool     `json:"only_official"`
	OnlyCore     bool     `json:"only_core"`
	SortBy       string   `json:"sort_by"` // 'relevance', 'popular', 'rating', 'newest'
	Limit        int      `json:"limit"`
	Offset       int      `json:"offset"`
}

type AddItemToTenantRequest struct {
	MarketplaceItemID string          `json:"marketplace_item_id"`
	CustomName        *string         `json:"custom_name"`
	CustomParameters  json.RawMessage `json:"custom_parameters"`
}

type MarketplaceSearchResponse struct {
	Items      []MarketplaceItem      `json:"items"`
	TotalCount int                    `json:"total_count"`
	Facets     map[string]interface{} `json:"facets"`
}

// ============================================================================
// Marketplace Routes
// ============================================================================

func RegisterMarketplaceRoutes(r chi.Router, db *sql.DB) {
	r.Get("/marketplace/items", handleListMarketplaceItems(db))
	r.Get("/marketplace/items/{id}", handleGetMarketplaceItem(db))
	r.Get("/marketplace/items/{id}/parameters", handleGetMarketplaceItemParameters(db))

	// Validation rules marketplace endpoints
	r.Get("/marketplace/validation-rules", handleListMarketplaceValidationRules(db))

	r.Post("/marketplace/items/add-to-tenant", handleAddItemToTenant(db))
	r.Get("/marketplace/tenant-items", handleListTenantMarketplaceItems(db))
	r.Get("/marketplace/tenant-items/{id}", handleGetTenantMarketplaceItem(db))
	r.Put("/marketplace/tenant-items/{id}", handleUpdateTenantMarketplaceItem(db))
	r.Delete("/marketplace/tenant-items/{id}", handleRemoveItemFromTenant(db))

	r.Post("/marketplace/items/{id}/feedback", handlePostMarketplaceItemFeedback(db))
	r.Get("/marketplace/items/{id}/feedback", handleGetMarketplaceItemFeedback(db))
}

// ============================================================================
// List Marketplace Items (Browse Catalog)
// ============================================================================

func handleListMarketplaceItems(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Parse query parameters
		search := r.URL.Query().Get("search")
		itemType := r.URL.Query().Get("item_type")
		categories := r.URL.Query()["category"]
		severities := r.URL.Query()["severity"]
		status := r.URL.Query().Get("status")
		if status == "" {
			status = "active"
		}
		onlyOfficial := r.URL.Query().Get("only_official") == "true"
		onlyCore := r.URL.Query().Get("only_core") == "true"
		sortBy := r.URL.Query().Get("sort_by")
		if sortBy == "" {
			sortBy = "relevance"
		}

		limit := 20
		offset := 0

		// Build query
		query := `
			SELECT 
				id, name, description, item_type, version, category,
				subcategories, severity, icon_emoji, color_hex,
				summary, long_description, implementation_json,
				scope, rule_type, frequency, evaluation_order,
				is_public, is_official, is_core, status,
				external_api_providers, requires_credentials,
				usage_count, rating, downloads_count,
				created_at, updated_at, published_at
			FROM marketplace_items
			WHERE is_public = true AND status = $1
		`

		args := []interface{}{status}
		argCount := 2

		// Add filters
		if itemType != "" {
			query += fmt.Sprintf(` AND item_type = $%d`, argCount)
			args = append(args, itemType)
			argCount++
		}

		if onlyOfficial {
			query += ` AND is_official = true`
		}

		if onlyCore {
			query += ` AND is_core = true`
		}

		if len(categories) > 0 {
			query += fmt.Sprintf(` AND category = ANY($%d)`, argCount)
			args = append(args, pq.Array(categories))
			argCount++
		}

		if len(severities) > 0 {
			query += fmt.Sprintf(` AND severity = ANY($%d)`, argCount)
			args = append(args, pq.Array(severities))
			argCount++
		}

		if search != "" {
			query += fmt.Sprintf(` AND (name ILIKE $%d OR description ILIKE $%d OR summary ILIKE $%d)`,
				argCount, argCount+1, argCount+2)
			searchPattern := "%" + search + "%"
			args = append(args, searchPattern, searchPattern, searchPattern)
			argCount += 3
		}

		// Add sorting
		switch sortBy {
		case "popular":
			query += ` ORDER BY usage_count DESC`
		case "rating":
			query += ` ORDER BY rating DESC NULLS LAST`
		case "newest":
			query += ` ORDER BY created_at DESC`
		default: // relevance
			query += ` ORDER BY is_official DESC, is_core DESC, rating DESC NULLS LAST`
		}

		query += fmt.Sprintf(` LIMIT %d OFFSET %d`, limit, offset)

		// Execute query
		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Database error", "db_error", err.Error())
			return
		}
		defer rows.Close()

		items := []MarketplaceItem{}
		for rows.Next() {
			var item MarketplaceItem
			// icon_emoji can be NULL in the database; scan into sql.NullString to avoid
			// a Scan error when the column is NULL, then convert to a plain string.
			var iconEmoji sql.NullString
			err := rows.Scan(
				&item.ID, &item.Name, &item.Description, &item.ItemType, &item.Version,
				&item.Category, pq.Array(&item.Subcategories), &item.Severity,
				&iconEmoji, &item.ColorHex, &item.Summary, &item.LongDescription,
				&item.ImplementationJSON, &item.Scope, &item.RuleType, &item.Frequency,
				&item.EvaluationOrder, &item.IsPublic, &item.IsOfficial, &item.IsCore,
				&item.Status, pq.Array(&item.ExternalAPIProviders), &item.RequiresCredentials,
				&item.UsageCount, &item.Rating, &item.DownloadsCount,
				&item.CreatedAt, &item.UpdatedAt, &item.PublishedAt,
			)
			if err != nil {
				// Fail fast: scanning DB rows failed. Return a structured JSON error
				// to the client rather than writing human-readable text into the
				// response body (which corrupts JSON). This makes handler behavior
				// consistent with other API endpoints.
				logging.GetLogger().Sugar().Warnf("marketplace items scan error: %v", err)
				writeJSONError(w, http.StatusInternalServerError, "Failed to scan marketplace item", "scan_error", err.Error())
				return
			}

			if iconEmoji.Valid {
				item.IconEmoji = iconEmoji.String
			} else {
				item.IconEmoji = ""
			}

			items = append(items, item)
		}

		// Get total count
		countQuery := `SELECT COUNT(*) FROM marketplace_items WHERE is_public = true AND status = $1`
		countArgs := []interface{}{status}

		// Apply same filters to count
		if itemType != "" {
			countQuery += ` AND item_type = $2`
			countArgs = append(countArgs, itemType)
		}

		totalCount := 0
		db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&totalCount)

		// Build response
		response := MarketplaceSearchResponse{
			Items:      items,
			TotalCount: totalCount,
			Facets: map[string]interface{}{
				"categories": []string{"ESG & Sustainability", "Risk Management", "Compliance & Regulatory", "Funds Accounting"},
				"severities": []string{"BLOCK", "WARNING", "INFO"},
				"item_types": []string{"rule", "calculation"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// ============================================================================
// Get Single Marketplace Item
// ============================================================================

func handleGetMarketplaceItem(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := chi.URLParam(r, "id")

		query := `
			SELECT 
				id, name, description, item_type, version, category,
				subcategories, severity, icon_emoji, color_hex,
				summary, long_description, implementation_json,
				scope, rule_type, frequency, evaluation_order,
				is_public, is_official, is_core, status,
				external_api_providers, requires_credentials,
				usage_count, rating, downloads_count,
				created_at, updated_at, published_at
			FROM marketplace_items
			WHERE id = $1 AND is_public = true
		`

		var item MarketplaceItem
		err := db.QueryRowContext(ctx, query, id).Scan(
			&item.ID, &item.Name, &item.Description, &item.ItemType, &item.Version,
			&item.Category, pq.Array(&item.Subcategories), &item.Severity,
			&item.IconEmoji, &item.ColorHex, &item.Summary, &item.LongDescription,
			&item.ImplementationJSON, &item.Scope, &item.RuleType, &item.Frequency,
			&item.EvaluationOrder, &item.IsPublic, &item.IsOfficial, &item.IsCore,
			&item.Status, pq.Array(&item.ExternalAPIProviders), &item.RequiresCredentials,
			&item.UsageCount, &item.Rating, &item.DownloadsCount,
			&item.CreatedAt, &item.UpdatedAt, &item.PublishedAt,
		)

		if err == sql.ErrNoRows {
			writeJSONError(w, http.StatusNotFound, "Item not found", "not_found", "")
			return
		}
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Database error", "db_error", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(item)
	}
}

// ============================================================================
// Get Marketplace Item Parameters
// ============================================================================

func handleGetMarketplaceItemParameters(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := chi.URLParam(r, "id")

		query := `
			SELECT 
				id, marketplace_item_id, param_name, param_type,
				description, is_required, default_value, validation_rules,
				display_name, display_order, help_text, created_at
			FROM marketplace_item_parameters
			WHERE marketplace_item_id = $1
			ORDER BY display_order ASC
		`

		rows, err := db.QueryContext(ctx, query, id)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Database error", "db_error", err.Error())
			return
		}
		defer rows.Close()

		params := []MarketplaceItemParameter{}
		for rows.Next() {
			var param MarketplaceItemParameter
			err := rows.Scan(
				&param.ID, &param.MarketplaceItemID, &param.ParamName, &param.ParamType,
				&param.Description, &param.IsRequired, &param.DefaultValue, &param.ValidationRules,
				&param.DisplayName, &param.DisplayOrder, &param.HelpText, &param.CreatedAt,
			)
			if err != nil {
				continue
			}
			params = append(params, param)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(params)
	}
}

// ============================================================================
// Add Item to Tenant
// ============================================================================

func handleAddItemToTenant(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get tenant from context (set by middleware)
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		if tenantID == "" {
			writeJSONError(w, http.StatusUnauthorized, "Tenant not specified", "auth_error", "")
			return
		}

		var req AddItemToTenantRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request", "invalid_request", err.Error())
			return
		}

		// Verify marketplace item exists
		var itemExists bool
		err := db.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM marketplace_items WHERE id = $1 AND is_public = true)`,
			req.MarketplaceItemID,
		).Scan(&itemExists)

		if err != nil || !itemExists {
			writeJSONError(w, http.StatusNotFound, "Item not found", "not_found", "")
			return
		}

		// Get marketplace item version
		var version string
		err = db.QueryRowContext(ctx,
			`SELECT version FROM marketplace_items WHERE id = $1`,
			req.MarketplaceItemID,
		).Scan(&version)

		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Database error", "db_error", err.Error())
			return
		}

		// Insert into tenant_marketplace_items
		id := uuid.New().String()
		customParams := req.CustomParameters
		if customParams == nil {
			customParams = json.RawMessage(`{}`)
		}

		_, err = db.ExecContext(ctx, `
			INSERT INTO tenant_marketplace_items (
				id, tenant_id, marketplace_item_id,
				custom_name, custom_parameters,
				marketplace_version_at_time_of_add, local_version
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (tenant_id, marketplace_item_id) DO UPDATE SET
				enabled_for_tenant = true,
				added_at = CURRENT_TIMESTAMP
		`,
			id, tenantID, req.MarketplaceItemID,
			req.CustomName, customParams,
			version, version,
		)

		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to add item", "db_error", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": id})
	}
}

// ============================================================================
// List Tenant's Marketplace Items
// ============================================================================

func handleListTenantMarketplaceItems(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		if tenantID == "" {
			writeJSONError(w, http.StatusUnauthorized, "Tenant not specified", "auth_error", "")
			return
		}

		query := `
			SELECT 
				tmi.id, tmi.tenant_id, tmi.marketplace_item_id,
				tmi.custom_name, tmi.custom_parameters, tmi.enabled_for_tenant,
				tmi.added_at, tmi.last_used_at, tmi.usage_count,
				tmi.marketplace_version_at_time_of_add, tmi.local_version,
				tmi.has_local_modifications, tmi.tenant_rating, tmi.tenant_feedback
			FROM tenant_marketplace_items tmi
			WHERE tmi.tenant_id = $1
			ORDER BY tmi.added_at DESC
		`

		rows, err := db.QueryContext(ctx, query, tenantID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Database error", "db_error", err.Error())
			return
		}
		defer rows.Close()

		items := []TenantMarketplaceItem{}
		for rows.Next() {
			var item TenantMarketplaceItem
			err := rows.Scan(
				&item.ID, &item.TenantID, &item.MarketplaceItemID,
				&item.CustomName, &item.CustomParameters, &item.EnabledForTenant,
				&item.AddedAt, &item.LastUsedAt, &item.UsageCount,
				&item.MarketplaceVersionAtTimeOfAdd, &item.LocalVersion,
				&item.HasLocalModifications, &item.TenantRating, &item.TenantFeedback,
			)
			if err != nil {
				continue
			}
			items = append(items, item)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	}
}

// ============================================================================
// Get Single Tenant Marketplace Item
// ============================================================================

func handleGetTenantMarketplaceItem(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		if tenantID == "" {
			writeJSONError(w, http.StatusUnauthorized, "Tenant not specified", "auth_error", "")
			return
		}

		id := chi.URLParam(r, "id")

		query := `
			SELECT 
				id, tenant_id, marketplace_item_id,
				custom_name, custom_parameters, enabled_for_tenant,
				added_at, last_used_at, usage_count,
				marketplace_version_at_time_of_add, local_version,
				has_local_modifications, tenant_rating, tenant_feedback
			FROM tenant_marketplace_items
			WHERE id = $1 AND tenant_id = $2
		`

		var item TenantMarketplaceItem
		err := db.QueryRowContext(ctx, query, id, tenantID).Scan(
			&item.ID, &item.TenantID, &item.MarketplaceItemID,
			&item.CustomName, &item.CustomParameters, &item.EnabledForTenant,
			&item.AddedAt, &item.LastUsedAt, &item.UsageCount,
			&item.MarketplaceVersionAtTimeOfAdd, &item.LocalVersion,
			&item.HasLocalModifications, &item.TenantRating, &item.TenantFeedback,
		)

		if err == sql.ErrNoRows {
			writeJSONError(w, http.StatusNotFound, "Item not found", "not_found", "")
			return
		}
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Database error", "db_error", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(item)
	}
}

// ============================================================================
// Update Tenant Marketplace Item
// ============================================================================

func handleUpdateTenantMarketplaceItem(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		if tenantID == "" {
			writeJSONError(w, http.StatusUnauthorized, "Tenant not specified", "auth_error", "")
			return
		}

		id := chi.URLParam(r, "id")

		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request", "invalid_request", err.Error())
			return
		}

		// Update fields
		if customName, ok := req["custom_name"]; ok {
			db.ExecContext(ctx,
				`UPDATE tenant_marketplace_items SET custom_name = $1 WHERE id = $2 AND tenant_id = $3`,
				customName, id, tenantID,
			)
		}

		if enabled, ok := req["enabled_for_tenant"]; ok {
			db.ExecContext(ctx,
				`UPDATE tenant_marketplace_items SET enabled_for_tenant = $1 WHERE id = $2 AND tenant_id = $3`,
				enabled, id, tenantID,
			)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	}
}

// ============================================================================
// Remove Item from Tenant
// ============================================================================

func handleRemoveItemFromTenant(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		if tenantID == "" {
			writeJSONError(w, http.StatusUnauthorized, "Tenant not specified", "auth_error", "")
			return
		}

		id := chi.URLParam(r, "id")

		_, err := db.ExecContext(ctx,
			`DELETE FROM tenant_marketplace_items WHERE id = $1 AND tenant_id = $2`,
			id, tenantID,
		)

		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to remove item", "db_error", err.Error())
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// ============================================================================
// Post Feedback on Marketplace Item
// ============================================================================

func handlePostMarketplaceItemFeedback(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		if tenantID == "" {
			writeJSONError(w, http.StatusUnauthorized, "Tenant not specified", "auth_error", "")
			return
		}

		itemID := chi.URLParam(r, "id")

		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request", "invalid_request", err.Error())
			return
		}

		rating, ok := req["rating"].(float64)
		if !ok || rating < 1 || rating > 5 {
			writeJSONError(w, http.StatusBadRequest, "Invalid rating", "invalid_request", "Rating must be between 1 and 5")
			return
		}

		feedback, _ := req["feedback"].(string)

		_, err := db.ExecContext(ctx, `
			INSERT INTO marketplace_item_feedback (
				id, tenant_id, marketplace_item_id, rating, feedback_text
			) VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (tenant_id, marketplace_item_id) DO UPDATE SET
				rating = $4, feedback_text = $5, updated_at = CURRENT_TIMESTAMP
		`,
			uuid.New().String(), tenantID, itemID, int(rating), feedback,
		)

		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to save feedback", "db_error", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "feedback_saved"})
	}
}

// ============================================================================
// Get Feedback on Marketplace Item
// ============================================================================

func handleGetMarketplaceItemFeedback(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		itemID := chi.URLParam(r, "id")

		query := `
			SELECT 
				COALESCE(AVG(rating), 0) as avg_rating,
				COUNT(*) as feedback_count,
				SUM(CASE WHEN rating >= 4 THEN 1 ELSE 0 END) as positive_count
			FROM marketplace_item_feedback
			WHERE marketplace_item_id = $1
		`

		var avgRating float64
		var feedbackCount int
		var positiveCount int

		err := db.QueryRowContext(ctx, query, itemID).Scan(&avgRating, &feedbackCount, &positiveCount)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Database error", "db_error", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"average_rating": avgRating,
			"feedback_count": feedbackCount,
			"positive_count": positiveCount,
		})
	}
}

// ============================================================================
// Marketplace Validation Rules
// ============================================================================

func handleListMarketplaceValidationRules(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Build query to fetch marketplace items where item_type = 'rule'
		query := `
			SELECT
				id,
				name,
				description,
				version,
				category,
				subcategories,
				severity,
				rule_type,
				scope,
				frequency,
				evaluation_order,
				implementation_json,
				status,
				is_public,
				is_official,
				is_core,
				usage_count,
				rating,
				downloads_count,
				created_at,
				updated_at
			FROM marketplace_items
			WHERE item_type = 'rule'
				AND is_public = true
				AND status = 'active'
			ORDER BY category, name
		`

		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Database error", "db_error", err.Error())
			return
		}
		defer rows.Close()

		var validationRules []map[string]interface{}

		for rows.Next() {
			var id, name, description, version, category, ruleType, scope, frequency, status string
			var subcategories pq.StringArray
			var severity sql.NullString
			var evaluationOrder sql.NullInt64
			var implementationJSON []byte
			var isPublic, isOfficial, isCore bool
			var usageCount, downloadsCount int
			var rating sql.NullFloat64
			var createdAt, updatedAt time.Time

			err := rows.Scan(
				&id, &name, &description, &version, &category, &subcategories,
				&severity, &ruleType, &scope, &frequency, &evaluationOrder,
				&implementationJSON, &status, &isPublic, &isOfficial, &isCore,
				&usageCount, &rating, &downloadsCount, &createdAt, &updatedAt,
			)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "Database scan error", "db_error", err.Error())
				return
			}

			// Parse implementation_json to extract parameters and other rule details
			var implData map[string]interface{}
			if err := json.Unmarshal(implementationJSON, &implData); err != nil {
				// If parsing fails, create a basic structure
				implData = map[string]interface{}{}
			}

			// Extract parameters from implementation_json if available
			parameters := implData["parameters"]
			if parameters == nil {
				parameters = map[string]interface{}{}
			}

			// Build the rule structure expected by the frontend
			rule := map[string]interface{}{
				"id":              id,
				"name":            name,
				"description":     description,
				"category":        category,
				"rule_type":       ruleType,
				"severity":        severity.String, // Convert to string, empty if null
				"scope":           scope,
				"frequency":       frequency,
				"evaluationOrder": evaluationOrder.Int64,
				"parameters":      parameters,
				"isActive":        true, // Assume active since we filter by status = 'active'
				"effectiveFrom":   createdAt.Format(time.RFC3339),
				"version":         version,
				"isOfficial":      isOfficial,
				"isCore":          isCore,
				"usageCount":      usageCount,
				"rating":          rating.Float64,
				"downloadsCount":  downloadsCount,
				"tags":            subcategories, // Use subcategories as tags
			}

			validationRules = append(validationRules, rule)
		}

		if err = rows.Err(); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Database iteration error", "db_error", err.Error())
			return
		}

		// Apply search filter if provided
		search := r.URL.Query().Get("search")
		if search != "" {
			filteredRules := []map[string]interface{}{}
			searchLower := strings.ToLower(search)
			for _, rule := range validationRules {
				name, _ := rule["name"].(string)
				description, _ := rule["description"].(string)
				category, _ := rule["category"].(string)

				if strings.Contains(strings.ToLower(name), searchLower) ||
					strings.Contains(strings.ToLower(description), searchLower) ||
					strings.Contains(strings.ToLower(category), searchLower) {
					filteredRules = append(filteredRules, rule)
				}
			}
			validationRules = filteredRules
		}

		// Apply category filter if provided
		category := r.URL.Query().Get("category")
		if category != "" {
			filteredRules := []map[string]interface{}{}
			for _, rule := range validationRules {
				ruleCategory, _ := rule["category"].(string)
				if ruleCategory == category {
					filteredRules = append(filteredRules, rule)
				}
			}
			validationRules = filteredRules
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rules":       validationRules,
			"total_count": len(validationRules),
		})
	}
}
