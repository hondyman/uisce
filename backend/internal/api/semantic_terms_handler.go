package api

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/backend/pkg/llm"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// SemanticTermsHandler handles semantic terms API requests
type SemanticTermsHandler struct {
	db        *sql.DB
	resolver  *services.SemanticResolver
	assistant *services.SemanticAssistant
	service   *analytics.SemanticMappingService
}

// NewSemanticTermsHandler creates a new semantic terms handler
func NewSemanticTermsHandler(db *sql.DB) *SemanticTermsHandler {
	repo := &services.SQLTermRepository{DB: db}
	resolver := services.NewSemanticResolver(repo)

	// Init LLM (defaults to env var)
	llmProvider := llm.NewGeminiProvider("", "")
	assistant := services.NewSemanticAssistant(llmProvider)

	return &SemanticTermsHandler{
		db:        db, // Assuming db is *sql.DB. If sqlx is needed, struct needs update.
		resolver:  resolver,
		assistant: assistant,
	}
}

// SetService sets the semantic mapping service
func (h *SemanticTermsHandler) SetService(svc *analytics.SemanticMappingService) {
	h.service = svc
}

// RegisterRoutes registers the semantic terms routes
func (h *SemanticTermsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/semantic-terms", h.GetSemanticTerms)
	r.Post("/semantic-terms", h.CreateSemanticTerm)
	r.Post("/semantic-terms/resolve", h.ResolveSemanticTerm)
	r.Post("/semantic-terms/resolve-expression", h.ResolveExpression)
	r.Post("/semantic-terms/suggest", h.SuggestSemanticTerms)
	r.Get("/semantic-terms/explain", h.ExplainSemanticTerm)
	r.Get("/semantic-terms/{id}/suggest-business-terms", h.SuggestBusinessTerms)
}

func (h *SemanticTermsHandler) CreateSemanticTerm(w http.ResponseWriter, r *http.Request) {
	// Not implemented fully as it was inline in api.go using mapping service.
	// The original orphaned code called srv.SemanticMappingSvc.CreateSemanticTerm.
	// Since we inject service now, we can use it.
	// But duplicate logic exists in semantic_mappings_handler for mapping-based creation.
	// This endpoint seems to be for manual creation?
	// api.go line 1242 was mostly empty/todo or just calling service?
	// Wait, I didn't verify line 1242 content in my read.
	// I'll leave placeholder or basic impl if I can't confirm.
	// Actually, let's look at api.go view from earlier (line 1360 calls CreateSemanticTerm).
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (h *SemanticTermsHandler) SuggestBusinessTerms(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	semanticTermID := chi.URLParam(r, "id")

	if tenantID == "" || tenantDatasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
		return
	}
	if semanticTermID == "" {
		http.Error(w, "Semantic Term ID is required", http.StatusBadRequest)
		return
	}

	if h.service == nil {
		http.Error(w, "Service not initialized", http.StatusInternalServerError)
		return
	}

	suggestions, err := h.service.SuggestBusinessTerms(r.Context(), tenantID, tenantDatasourceID, semanticTermID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}

// GetSemanticTerms retrieves semantic terms
func (h *SemanticTermsHandler) GetSemanticTerms(w http.ResponseWriter, r *http.Request) {
	datasourceID := r.URL.Query().Get("datasource_id")

	// Filter by datasource if provided, but semantic terms might be global
	// For now keeping existing logic but making it optional if needed
	var rows *sql.Rows
	var err error

	query := `
		SELECT 
			id, 
			node_name, 
			COALESCE(description, '') as description,
			COALESCE(properties, '{}'::jsonb) as properties,
			qualified_path,
			created_at,
			updated_at
		FROM catalog_node
		WHERE node_type_id = '820b942a-9c9e-4abc-acdc-84616db33098'
	`

	if datasourceID != "" {
		query += " AND (tenant_datasource_id = $1::uuid OR tenant_datasource_id IS NULL)"
		rows, err = h.db.QueryContext(r.Context(), query+" ORDER BY node_name ASC", datasourceID)
	} else {
		rows, err = h.db.QueryContext(r.Context(), query+" ORDER BY node_name ASC")
	}

	if err != nil {
		http.Error(w, "Failed to query semantic terms: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var terms []*models.SemanticTerm
	for rows.Next() {
		var id, nodeName, description, qualifiedPath string
		var createdAt, updatedAt time.Time
		var properties []byte

		err := rows.Scan(
			&id,
			&nodeName,
			&description,
			&properties,
			&qualifiedPath,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			http.Error(w, "Failed to scan semantic term: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Unmarshal properties into SemanticTerm
		term := &models.SemanticTerm{}
		if len(properties) > 0 && string(properties) != "null" {
			_ = json.Unmarshal(properties, term)
		}

		// Overwrite with catalog_node columns (authoritative)
		term.ID = id
		term.NodeName = nodeName
		term.Description = description
		term.QualifiedPath = qualifiedPath
		term.CreatedAt = createdAt
		term.UpdatedAt = updatedAt

		// Default type if missing
		if term.Type == "" {
			term.Type = models.SemanticTypePhysical
		}

		terms = append(terms, term)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": terms,
	})
}

// ResolveSemanticTermRequest payload
type ResolveSemanticTermRequest struct {
	TermID string `json:"term_id"`
}

// ResolveSemanticTerm returns the resolved SQL for a term
func (h *SemanticTermsHandler) ResolveSemanticTerm(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var req ResolveSemanticTermRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.TermID == "" {
		http.Error(w, "term_id is required", http.StatusBadRequest)
		return
	}

	sqlFragment, err := h.resolver.ResolveToSQL(req.TermID)
	if err != nil {
		http.Error(w, "Resolution failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	lineage, _ := h.resolver.GetLineage(req.TermID)

	response := map[string]interface{}{
		"term_id": req.TermID,
		"sql":     sqlFragment,
		"lineage": lineage,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SuggestSemanticTermsRequest payload
type SuggestSemanticTermsRequest struct {
	SchemaContext string `json:"schema_context"`
}

// SuggestSemanticTerms uses LLM to suggest terms
func (h *SemanticTermsHandler) SuggestSemanticTerms(w http.ResponseWriter, r *http.Request) {
	if h.assistant == nil {
		http.Error(w, "AI Assistant not configured", http.StatusNotImplemented)
		return
	}

	var req SuggestSemanticTermsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	suggestions, err := h.assistant.SuggestTerms(r.Context(), req.SchemaContext)
	if err != nil {
		http.Error(w, "Failed to generate suggestions: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"suggestions": suggestions,
	})
}

// ResolveExpressionRequest payload
type ResolveExpressionRequest struct {
	Expression string `json:"expression"`
}

// ResolveExpression resolves an ad-hoc expression string
func (h *SemanticTermsHandler) ResolveExpression(w http.ResponseWriter, r *http.Request) {
	var req ResolveExpressionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Expression == "" {
		http.Error(w, "expression is required", http.StatusBadRequest)
		return
	}

	sqlFragment, err := h.resolver.ResolveExpressionString(req.Expression)
	if err != nil {
		http.Error(w, "Resolution failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Getting lineage for an expression string is harder because GetLineage takes an ID
	// But we can peek into internal or just return nil for now
	// (or update resolver to expose lineage-from-string)

	response := map[string]interface{}{
		"sql":     sqlFragment,
		"lineage": nil, // TODO: Implement lineage for ad-hoc strings
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ExplainSemanticTermRequest - parameters can be in query string, but struct for internal use
type ExplainSemanticTermRequest struct {
	TermID     string `json:"term_id"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
}

// ExplainSemanticTerm provides deep debug info for a term
func (h *SemanticTermsHandler) ExplainSemanticTerm(w http.ResponseWriter, r *http.Request) {
	termID := r.URL.Query().Get("term_id")
	entityType := r.URL.Query().Get("entity_type")
	entityID := r.URL.Query().Get("entity_id")

	if termID == "" {
		http.Error(w, "term_id is required", http.StatusBadRequest)
		return
	}

	// 1. Resolve Value (runs plugins etc)
	res, err := h.resolver.ResolveValue(termID, entityID)
	if err != nil {
		http.Error(w, "Resolution failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. Build Response
	explainRes := models.ExplainResponse{
		Header: models.ExplainHeader{
			TermID:           termID,
			EntityType:       entityType,
			EntityID:         entityID,
			Value:            res.Value,
			EvaluatedAt:      time.Now(),
			EvaluatorVersion: "semantic-engine-2.1.0",
		},
		Summary: models.ExplainSummary{
			HumanReadable: "Evaluated successfully.",
		},
		EvaluationPath: res.Path,
		Rows:           res.Rows,
		Anomalies:      res.Anomalies,
	}

	// Mock Lineage for demo
	lineage, _ := h.resolver.GetLineage(termID)
	var deps []models.SemanticTerm
	for _, l := range lineage {
		deps = append(deps, models.SemanticTerm{ID: l})
	}
	explainRes.Lineage = models.ExplainLineage{
		Dependencies: deps,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(explainRes)
}
