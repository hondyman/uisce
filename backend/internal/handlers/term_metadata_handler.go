package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/boresolver"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ============================================================================
// Term Metadata Handler
// ============================================================================

// TermMetadataHandler handles BO term metadata operations
type TermMetadataHandler struct {
	db *sqlx.DB
}

// NewTermMetadataHandler creates a new handler
func NewTermMetadataHandler(db *sqlx.DB) *TermMetadataHandler {
	return &TermMetadataHandler{db: db}
}

// RegisterRoutes registers term metadata API routes
func (h *TermMetadataHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/bo/{boId}/terms", h.ListTermsForBO)
	r.Get("/api/bo/{boId}/term/{termId}/metadata", h.GetTermMetadata)
	r.Patch("/api/bo/{boId}/term/{termId}/metadata", h.UpdateTermMetadata)
	r.Post("/api/bo/{boId}/term/{termId}/metadata", h.CreateTermMetadata)
	r.Delete("/api/bo/{boId}/term/{termId}/metadata", h.DeleteTermMetadata)

	// Suggestions
	r.Get("/api/bo/{boId}/expression-suggestions", h.GetExpressionSuggestions)
}

// ============================================================================
// Models
// ============================================================================

// TermMetadata represents BO-specific term settings
type TermMetadata struct {
	ID           string    `json:"id" db:"id"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	BOID         string    `json:"bo_id" db:"bo_id"`
	TermID       string    `json:"term_id" db:"term_id"`
	DisplayName  *string   `json:"display_name" db:"display_name"`
	Description  *string   `json:"description" db:"description"`
	GroupName    *string   `json:"group_name" db:"group_name"`
	Required     bool      `json:"required" db:"required"`
	Visible      bool      `json:"visible" db:"visible"`
	Format       *string   `json:"format" db:"format"`
	Precision    int       `json:"precision" db:"precision"`
	CurrencyCode *string   `json:"currency_code" db:"currency_code"`
	DateFormat   *string   `json:"date_format" db:"date_format"`
	Aggregation  *string   `json:"aggregation" db:"aggregation"`
	SortOrder    int       `json:"sort_order" db:"sort_order"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// TermWithMetadata combines term info with its BO-specific metadata
type TermWithMetadata struct {
	// Term info from catalog_node
	TermID        string  `json:"term_id" db:"term_id"`
	TermName      string  `json:"term_name" db:"term_name"`
	TermTitle     *string `json:"term_title" db:"term_title"`
	SourceColumn  *string `json:"source_column" db:"source_column"`
	DataType      *string `json:"data_type" db:"data_type"`
	IsCalculation bool    `json:"is_calculation" db:"is_calculation"`

	// Metadata (nullable - may not exist yet)
	MetadataID   *string `json:"metadata_id" db:"metadata_id"`
	DisplayName  *string `json:"display_name" db:"display_name"`
	Description  *string `json:"description" db:"description"`
	GroupName    *string `json:"group_name" db:"group_name"`
	Required     bool    `json:"required" db:"required"`
	Visible      bool    `json:"visible" db:"visible"`
	Format       *string `json:"format" db:"format"`
	Precision    *int    `json:"precision" db:"precision"`
	CurrencyCode *string `json:"currency_code" db:"currency_code"`
	DateFormat   *string `json:"date_format" db:"date_format"`
	Aggregation  *string `json:"aggregation" db:"aggregation"`
	SortOrder    *int    `json:"sort_order" db:"sort_order"`

	// Enriched fields
	InferredType string `json:"inferred_type,omitempty"`
	IsAggregate  bool   `json:"is_aggregate,omitempty"`
}

// TermMetadataRequest is the request body for create/update
type TermMetadataRequest struct {
	DisplayName  *string `json:"display_name"`
	Description  *string `json:"description"`
	GroupName    *string `json:"group_name"`
	Required     *bool   `json:"required"`
	Visible      *bool   `json:"visible"`
	Format       *string `json:"format"`
	Precision    *int    `json:"precision"`
	CurrencyCode *string `json:"currency_code"`
	DateFormat   *string `json:"date_format"`
	Aggregation  *string `json:"aggregation"`
	SortOrder    *int    `json:"sort_order"`
}

// Validation constants
var validFormats = map[string]bool{
	"currency": true,
	"percent":  true,
	"number":   true,
	"date":     true,
	"string":   true,
	"integer":  true,
	"boolean":  true,
}

var validAggregations = map[string]bool{
	"none":  true,
	"sum":   true,
	"avg":   true,
	"min":   true,
	"max":   true,
	"count": true,
}

// ============================================================================
// Handlers
// ============================================================================

// ListTermsForBO returns all terms in a BO with their metadata
func (h *TermMetadataHandler) ListTermsForBO(w http.ResponseWriter, r *http.Request) {
	boID := chi.URLParam(r, "boId")
	if boID == "" {
		http.Error(w, "boId is required", http.StatusBadRequest)
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	ctx := r.Context()

	// 1. Get terms from bo_fields (persistence layer)
	query := `
		SELECT 
			COALESCE(st.id, bf.semantic_term_id::text) as term_id,
			COALESCE(st.node_name, bf.technical_name, bf.name) as term_name,
			COALESCE(st.properties->>'title', bf.display_label, st.node_name) as term_title,
			st.properties->>'source_column' as source_column,
			st.properties->>'data_type' as data_type,
			COALESCE((st.properties->>'is_calculation')::boolean, false) as is_calculation,
			tm.id as metadata_id,
			COALESCE(tm.display_name, bf.display_label, st.properties->>'title', st.node_name) as display_name,
			COALESCE(tm.description, st.description) as description,
			tm.group_name,
			COALESCE(tm.required, false) as required,
			COALESCE(tm.visible, true) as visible,
			tm.format,
			tm.precision,
			tm.currency_code,
			tm.date_format,
			tm.aggregation,
			COALESCE(tm.sort_order, bf.display_order, 0) as sort_order
		FROM bo_fields bf
		LEFT JOIN catalog_node st ON st.id = bf.semantic_term_id
		LEFT JOIN bo_term_metadata tm ON tm.bo_id = bf.business_object_id AND tm.term_id = st.id
		WHERE bf.business_object_id = $1
		ORDER BY COALESCE(tm.sort_order, bf.display_order, 999), bf.name
	`

	var terms []TermWithMetadata
	if err := h.db.SelectContext(ctx, &terms, query, boID, tenantID); err != nil {
		http.Error(w, "Failed to list terms: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Identify calculation terms
	var calcTermIDs []string
	typeEnvMap := make(map[string]boresolver.ExprType)

	for _, t := range terms {
		// Populate basic type env from catalog
		var et boresolver.ExprType
		if t.DataType != nil {
			et = mapDataType(*t.DataType)
		} else {
			et = boresolver.TypeUnknown
		}
		typeEnvMap[t.TermName] = et

		if t.IsCalculation {
			calcTermIDs = append(calcTermIDs, t.TermID)
		}
	}

	// 3. Batched fetch of formulas (if any calculations)
	formulaMap := make(map[string]string)
	if len(calcTermIDs) > 0 {
		q, args, err := sqlx.In("SELECT node_id, formula FROM calculations WHERE node_id IN (?)", calcTermIDs)
		if err == nil {
			q = h.db.Rebind(q)
			rows, err := h.db.QueryContext(ctx, q, args...)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var nodeID, formula string
					if err := rows.Scan(&nodeID, &formula); err == nil {
						formulaMap[nodeID] = formula
					}
				}
			}
		}
	}

	// 4. Enrich terms with inferred info
	// Use a custom TypeEnv implementation that reads from typeEnvMap
	env := &memTypeEnv{types: typeEnvMap}

	for i := range terms {
		t := &terms[i]
		if t.IsCalculation {
			if formula, ok := formulaMap[t.TermID]; ok && formula != "" {
				expr, err := boresolver.ParseExpression(formula)
				// If parse error, ignore or mark as valid?
				if err == nil {
					t.InferredType = string(boresolver.InferType(expr, env))
					t.IsAggregate = boresolver.IsAggregateExpr(expr)
				}
			}
		} else {
			// Direct term
			if t.DataType != nil {
				t.InferredType = string(mapDataType(*t.DataType))
			}
			t.IsAggregate = false // direct columns usually not aggregate unless specified
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"bo_id": boID,
		"terms": terms,
		"count": len(terms),
	})
}

// GetTermMetadata ... (unchanged)
func (h *TermMetadataHandler) GetTermMetadata(w http.ResponseWriter, r *http.Request) {
	boID := chi.URLParam(r, "boId")
	termID := chi.URLParam(r, "termId")
	if boID == "" || termID == "" {
		http.Error(w, "boId and termId are required", http.StatusBadRequest)
		return
	}

	_ = jwtmiddleware.GetClaimsFromContext(r).TenantID // Reserved for future tenant validation
	ctx := r.Context()

	// First verify the term is in this BO
	var termInBO bool
	err := h.db.GetContext(ctx, &termInBO, `
		SELECT EXISTS(
			SELECT 1 FROM catalog_edge 
			WHERE source_node_id = $1 
			  AND target_node_id = $2 
			  AND edge_type_name = 'HAS_ATTRIBUTE'
		)
	`, boID, termID)
	if err != nil || !termInBO {
		http.Error(w, "TERM_NOT_IN_BO: Term is not associated with this Business Object", http.StatusBadRequest)
		return
	}

	// Get metadata or defaults
	var metadata TermMetadata
	err = h.db.GetContext(ctx, &metadata, `
		SELECT * FROM bo_term_metadata WHERE bo_id = $1 AND term_id = $2
	`, boID, termID)

	if err == sql.ErrNoRows {
		// Return defaults from the semantic term
		var termDefaults struct {
			DisplayName string  `db:"display_name"`
			Description *string `db:"description"`
			DataType    *string `db:"data_type"`
		}
		h.db.GetContext(ctx, &termDefaults, `
			SELECT 
				COALESCE(properties->>'title', node_name) as display_name,
				description,
				properties->>'data_type' as data_type
			FROM catalog_node WHERE id = $1
		`, termID)

		// Return synthetic metadata with defaults
		response := map[string]interface{}{
			"bo_id":        boID,
			"term_id":      termID,
			"display_name": termDefaults.DisplayName,
			"description":  termDefaults.Description,
			"group_name":   nil,
			"required":     false,
			"visible":      true,
			"format":       inferFormat(termDefaults.DataType),
			"precision":    2,
			"aggregation":  "none",
			"sort_order":   0,
			"exists":       false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	if err != nil {
		http.Error(w, "Failed to get metadata: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Add exists flag
	response := map[string]interface{}{
		"id":            metadata.ID,
		"bo_id":         metadata.BOID,
		"term_id":       metadata.TermID,
		"display_name":  metadata.DisplayName,
		"description":   metadata.Description,
		"group_name":    metadata.GroupName,
		"required":      metadata.Required,
		"visible":       metadata.Visible,
		"format":        metadata.Format,
		"precision":     metadata.Precision,
		"currency_code": metadata.CurrencyCode,
		"date_format":   metadata.DateFormat,
		"aggregation":   metadata.Aggregation,
		"sort_order":    metadata.SortOrder,
		"created_at":    metadata.CreatedAt,
		"updated_at":    metadata.UpdatedAt,
		"exists":        true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreateTermMetadata ... (unchanged)
func (h *TermMetadataHandler) CreateTermMetadata(w http.ResponseWriter, r *http.Request) {
	boID := chi.URLParam(r, "boId")
	termID := chi.URLParam(r, "termId")
	if boID == "" || termID == "" {
		http.Error(w, "boId and termId are required", http.StatusBadRequest)
		return
	}

	var req TermMetadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validateMetadataRequest(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	ctx := r.Context()

	var termInBO bool
	h.db.GetContext(ctx, &termInBO, `
		SELECT EXISTS(SELECT 1 FROM catalog_edge WHERE source_node_id = $1 AND target_node_id = $2 AND edge_type = 'HAS_ATTRIBUTE')
	`, boID, termID)
	if !termInBO {
		http.Error(w, "TERM_NOT_IN_BO", http.StatusBadRequest)
		return
	}

	metadataID := uuid.New().String()
	now := time.Now()

	_, err := h.db.ExecContext(ctx, `
		INSERT INTO bo_term_metadata (
			id, tenant_id, bo_id, term_id, display_name, description, group_name,
			required, visible, format, precision, currency_code, date_format,
			aggregation, sort_order, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $16)
	`, metadataID, tenantID, boID, termID,
		req.DisplayName, req.Description, req.GroupName,
		coalesce(req.Required, false), coalesce(req.Visible, true),
		req.Format, coalesce(req.Precision, 2), req.CurrencyCode, req.DateFormat,
		coalesce(req.Aggregation, "none"), coalesce(req.SortOrder, 0), now)

	if err != nil {
		http.Error(w, "Failed to create metadata: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      metadataID,
		"bo_id":   boID,
		"term_id": termID,
		"message": "Metadata created successfully",
	})
}

// UpdateTermMetadata ... (unchanged)
func (h *TermMetadataHandler) UpdateTermMetadata(w http.ResponseWriter, r *http.Request) {
	boID := chi.URLParam(r, "boId")
	termID := chi.URLParam(r, "termId")
	if boID == "" || termID == "" {
		http.Error(w, "boId and termId are required", http.StatusBadRequest)
		return
	}

	var req TermMetadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validateMetadataRequest(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	ctx := r.Context()

	var exists bool
	h.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM bo_term_metadata WHERE bo_id = $1 AND term_id = $2)`, boID, termID)

	now := time.Now()

	if exists {
		_, err := h.db.ExecContext(ctx, `
			UPDATE bo_term_metadata SET
				display_name = COALESCE($3, display_name),
				description = COALESCE($4, description),
				group_name = COALESCE($5, group_name),
				required = COALESCE($6, required),
				visible = COALESCE($7, visible),
				format = COALESCE($8, format),
				precision = COALESCE($9, precision),
				currency_code = COALESCE($10, currency_code),
				date_format = COALESCE($11, date_format),
				aggregation = COALESCE($12, aggregation),
				sort_order = COALESCE($13, sort_order),
				updated_at = $14
			WHERE bo_id = $1 AND term_id = $2
		`, boID, termID,
			req.DisplayName, req.Description, req.GroupName,
			req.Required, req.Visible, req.Format, req.Precision,
			req.CurrencyCode, req.DateFormat, req.Aggregation, req.SortOrder, now)

		if err != nil {
			http.Error(w, "Failed to update metadata: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		metadataID := uuid.New().String()
		_, err := h.db.ExecContext(ctx, `
			INSERT INTO bo_term_metadata (
				id, tenant_id, bo_id, term_id, display_name, description, group_name,
				required, visible, format, precision, currency_code, date_format,
				aggregation, sort_order, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $16)
		`, metadataID, tenantID, boID, termID,
			req.DisplayName, req.Description, req.GroupName,
			coalesce(req.Required, false), coalesce(req.Visible, true),
			req.Format, coalesce(req.Precision, 2), req.CurrencyCode, req.DateFormat,
			coalesce(req.Aggregation, "none"), coalesce(req.SortOrder, 0), now)

		if err != nil {
			http.Error(w, "Failed to create metadata: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"bo_id":   boID,
		"term_id": termID,
		"message": "Metadata updated successfully",
	})
}

// DeleteTermMetadata ... (unchanged)
func (h *TermMetadataHandler) DeleteTermMetadata(w http.ResponseWriter, r *http.Request) {
	boID := chi.URLParam(r, "boId")
	termID := chi.URLParam(r, "termId")
	if boID == "" || termID == "" {
		http.Error(w, "boId and termId are required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	_, err := h.db.ExecContext(ctx, `DELETE FROM bo_term_metadata WHERE bo_id = $1 AND term_id = $2`, boID, termID)
	if err != nil {
		http.Error(w, "Failed to delete metadata: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetExpressionSuggestions
func (h *TermMetadataHandler) GetExpressionSuggestions(w http.ResponseWriter, r *http.Request) {
	boID := chi.URLParam(r, "boId")
	q := r.URL.Query().Get("q")

	// Query terms for autocomplete
	var terms []struct {
		Name        string `db:"node_name"`
		DisplayName string `db:"display_name"`
	}

	query := `
		SELECT 
			st.node_name, 
			COALESCE(st.properties->>'title', st.node_name) as display_name
		FROM catalog_edge ce
		JOIN catalog_node st ON ce.target_node_id = st.id
		WHERE ce.source_node_id = $1 
		  AND ce.edge_type = 'HAS_ATTRIBUTE'
		  AND (st.node_name ILIKE $2 OR st.properties->>'title' ILIKE $2)
		LIMIT 10
	`
	// Note: using $1 and $2 for sqlx
	err := h.db.Select(&terms, query, boID, "%"+q+"%")
	if err != nil {
		// Just return empty if error
		terms = []struct {
			Name        string `db:"node_name"`
			DisplayName string `db:"display_name"`
		}{}
	}

	functions := []string{
		"sum()", "avg()", "min()", "max()", "count()",
		"coalesce(, )",
		"round(, 2)",
		"case_when(cond, val, else)",
		"date_add('day', , )",
		"abs()", "cast()",
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"terms":     terms,
		"functions": functions,
	})
}

// ============================================================================
// Helpers
// ============================================================================

type memTypeEnv struct {
	types map[string]boresolver.ExprType
}

func (e *memTypeEnv) TermType(name string) boresolver.ExprType {
	if t, ok := e.types[name]; ok {
		return t
	}
	return boresolver.TypeUnknown
}

func mapDataType(dt string) boresolver.ExprType {
	switch strings.ToLower(dt) {
	case "integer", "bigint", "smallint", "numeric", "decimal", "double", "float", "real":
		return boresolver.TypeNumber
	case "boolean":
		return boresolver.TypeBool
	case "date", "timestamp", "timestamptz":
		return boresolver.TypeDate
	case "string", "text", "varchar":
		return boresolver.TypeString
	default:
		return boresolver.TypeString
	}
}

func validateMetadataRequest(req TermMetadataRequest) error {
	if req.Format != nil && !validFormats[*req.Format] {
		return &validationError{Code: "INVALID_FORMAT", Message: "Invalid format value"}
	}
	if req.Aggregation != nil && !validAggregations[*req.Aggregation] {
		return &validationError{Code: "INVALID_AGGREGATION", Message: "Invalid aggregation value"}
	}
	if req.GroupName != nil && len(*req.GroupName) > 100 {
		return &validationError{Code: "GROUP_NAME_TOO_LONG", Message: "Group name must be 100 characters or less"}
	}
	return nil
}

type validationError struct {
	Code    string
	Message string
}

func (e *validationError) Error() string {
	return e.Code + ": " + e.Message
}

func inferFormat(dataType *string) string {
	if dataType == nil {
		return "string"
	}
	switch *dataType {
	case "integer", "bigint", "smallint":
		return "integer"
	case "numeric", "decimal", "float", "double", "real":
		return "number"
	case "date", "timestamp", "timestamptz":
		return "date"
	case "boolean":
		return "boolean"
	default:
		return "string"
	}
}

func coalesce[T any](ptr *T, defaultVal T) T {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}
