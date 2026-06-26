package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/db"
	"github.com/jmoiron/sqlx"
)

// AlphaLinkHandler provides endpoints to link non-gold nodes to gold copy nodes per tenant.
type AlphaLinkHandler struct {
	DB *sqlx.DB
}

func NewAlphaLinkHandler(dbx *sqlx.DB) *AlphaLinkHandler {
	return &AlphaLinkHandler{DB: dbx}
}

// RegisterRoutes registers alpha link endpoints under /api.
func (h *AlphaLinkHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/tenants/{tenant_id}/catalog/link-alpha", func(r chi.Router) {
		// Generic: all node types for tenant
		r.Post("/", h.linkAlphaAllTypes)
		// Preview (dry-run) for all node types
		r.Get("/preview", h.previewAlphaAllTypes)

		// Generic: specific node type as path param
		r.Post("/{node_type_id}", h.linkAlphaForNodeType)
		// Preview for specific node type
		r.Get("/{node_type_id}/preview", h.previewAlphaForNodeType)

		// Convenience endpoints for specific node types
		r.Post("/semantic-model", h.linkSemanticModel)
		r.Post("/semantic-column", h.linkSemanticColumn)
		r.Post("/schema", h.linkSchema)
		r.Post("/database-column", h.linkDatabaseColumn)
		// Convenience previews
		r.Get("/semantic-model/preview", h.previewSemanticModel)
		r.Get("/semantic-column/preview", h.previewSemanticColumn)
		r.Get("/schema/preview", h.previewSchema)
		r.Get("/database-column/preview", h.previewDatabaseColumn)
	})
}

// linkAlphaAllTypes links alpha nodes for all node types for a tenant.
func (h *AlphaLinkHandler) linkAlphaAllTypes(w http.ResponseWriter, r *http.Request) {
	tenantStr := chi.URLParam(r, "tenant_id")
	tenantID, err := uuid.Parse(tenantStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid tenant_id"})
		return
	}
	rows, err := db.LinkAlphaNodesForTenant(r.Context(), h.DB, tenantID, nil)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tenant_id": tenantID, "rows_affected": rows})
}

// linkAlphaForNodeType links alpha nodes for a specific node type for a tenant.
func (h *AlphaLinkHandler) linkAlphaForNodeType(w http.ResponseWriter, r *http.Request) {
	tenantStr := chi.URLParam(r, "tenant_id")
	tenantID, err := uuid.Parse(tenantStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid tenant_id"})
		return
	}
	nodeTypeStr := chi.URLParam(r, "node_type_id")
	nodeTypeID, err := uuid.Parse(nodeTypeStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid node_type_id"})
		return
	}
	rows, err := db.LinkAlphaNodesForTenant(r.Context(), h.DB, tenantID, &nodeTypeID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tenant_id": tenantID, "node_type_id": nodeTypeID, "rows_affected": rows})
}

// Convenience node type IDs
var (
	nodeTypeSemanticModel  uuid.UUID
	nodeTypeSemanticColumn uuid.UUID
	nodeTypeSchema         uuid.UUID
	nodeTypeDatabaseColumn uuid.UUID
)

func mustParseOrDefault(envKey, def string) uuid.UUID {
	if v := os.Getenv(envKey); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			return id
		}
	}
	return uuid.MustParse(def)
}

func init() {
	nodeTypeSemanticModel = mustParseOrDefault("SEMLAYER_NODETYPE_SEMANTIC_MODEL", "c53f9e99-8d02-4dfb-bc1b-914747d35edb")
	nodeTypeSemanticColumn = mustParseOrDefault("SEMLAYER_NODETYPE_SEMANTIC_COLUMN", "1439f761-606a-44cb-b4f8-7aa6b27a9bf5")
	nodeTypeSchema = mustParseOrDefault("SEMLAYER_NODETYPE_SCHEMA", "68d6d495-0992-4d92-ad2f-7f66dc1e7d78")
	nodeTypeDatabaseColumn = mustParseOrDefault("SEMLAYER_NODETYPE_DATABASE_COLUMN", "a64c1011-16e8-4ddf-b447-363bf8e15c9a")
}

func (h *AlphaLinkHandler) linkSemanticModel(w http.ResponseWriter, r *http.Request) {
	h.linkConvenience(w, r, nodeTypeSemanticModel)
}
func (h *AlphaLinkHandler) linkSemanticColumn(w http.ResponseWriter, r *http.Request) {
	h.linkConvenience(w, r, nodeTypeSemanticColumn)
}
func (h *AlphaLinkHandler) linkSchema(w http.ResponseWriter, r *http.Request) {
	h.linkConvenience(w, r, nodeTypeSchema)
}
func (h *AlphaLinkHandler) linkDatabaseColumn(w http.ResponseWriter, r *http.Request) {
	h.linkConvenience(w, r, nodeTypeDatabaseColumn)
}

func (h *AlphaLinkHandler) linkConvenience(w http.ResponseWriter, r *http.Request, nodeTypeID uuid.UUID) {
	tenantStr := chi.URLParam(r, "tenant_id")
	tenantID, err := uuid.Parse(tenantStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid tenant_id"})
		return
	}
	rows, err := db.LinkAlphaNodesForTenant(r.Context(), h.DB, tenantID, &nodeTypeID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tenant_id": tenantID, "node_type_id": nodeTypeID, "rows_affected": rows})
}

// previewAll previews with optional pagination and count-only mode
func (h *AlphaLinkHandler) previewAll(w http.ResponseWriter, r *http.Request, nodeTypeID *uuid.UUID) {
	tenantStr := chi.URLParam(r, "tenant_id")
	tenantID, err := uuid.Parse(tenantStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid tenant_id"})
		return
	}

	// Optional query params: limit, offset, count_only
	type qp struct {
		Limit, Offset int
		CountOnly     bool
	}
	var params qp
	// manual parse for simplicity
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			params.Limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			params.Offset = n
		}
	}
	if v := r.URL.Query().Get("count_only"); v == "true" || v == "1" {
		params.CountOnly = true
	}

	if params.CountOnly {
		cnt, err := db.CountPreviewAlphaLinksForTenant(r.Context(), h.DB, tenantID, nodeTypeID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"tenant_id": tenantID, "count": cnt})
		return
	}

	previews, err := db.PreviewAlphaLinksForTenant(r.Context(), h.DB, tenantID, nodeTypeID, params.Limit, params.Offset)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tenant_id": tenantID, "results": previews})
}

func (h *AlphaLinkHandler) previewAlphaAllTypes(w http.ResponseWriter, r *http.Request) {
	h.previewAll(w, r, nil)
}
func (h *AlphaLinkHandler) previewAlphaForNodeType(w http.ResponseWriter, r *http.Request) {
	nodeTypeStr := chi.URLParam(r, "node_type_id")
	nodeTypeID, err := uuid.Parse(nodeTypeStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid node_type_id"})
		return
	}
	h.previewAll(w, r, &nodeTypeID)
}

func (h *AlphaLinkHandler) previewSemanticModel(w http.ResponseWriter, r *http.Request) {
	h.previewAll(w, r, &nodeTypeSemanticModel)
}
func (h *AlphaLinkHandler) previewSemanticColumn(w http.ResponseWriter, r *http.Request) {
	h.previewAll(w, r, &nodeTypeSemanticColumn)
}
func (h *AlphaLinkHandler) previewSchema(w http.ResponseWriter, r *http.Request) {
	h.previewAll(w, r, &nodeTypeSchema)
}
func (h *AlphaLinkHandler) previewDatabaseColumn(w http.ResponseWriter, r *http.Request) {
	h.previewAll(w, r, &nodeTypeDatabaseColumn)
}
