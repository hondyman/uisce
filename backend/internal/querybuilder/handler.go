package querybuilder

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

// Executor resolves the database connection on which to run a query.
// The concrete implementation lives in the api package where the Server holds
// the configured DB pools.
type Executor interface {
	QueryDB(datasourceID string) *sqlx.DB
}

// QueryBuilderHandler exposes the QueryDef endpoints.
type QueryBuilderHandler struct {
	service  *QueryService
	executor Executor
	deps     handlers.SecurityContextDeps
}

// NewQueryBuilderHandler creates a handler.
func NewQueryBuilderHandler(service *QueryService, executor Executor, deps handlers.SecurityContextDeps) *QueryBuilderHandler {
	return &QueryBuilderHandler{
		service:  service,
		executor: executor,
		deps:     deps,
	}
}

// Preview handles POST /api/query/preview.
func (h *QueryBuilderHandler) Preview(w http.ResponseWriter, r *http.Request) {
	secCtx, qd, ok := h.decodeAndAuthorize(w, r)
	if !ok {
		return
	}

	resp, err := h.service.Preview(r.Context(), secCtx, qd)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	h.writeJSON(w, resp)
}

// Execute handles POST /api/query/execute.
func (h *QueryBuilderHandler) Execute(w http.ResponseWriter, r *http.Request) {
	secCtx, qd, ok := h.decodeAndAuthorize(w, r)
	if !ok {
		return
	}

	db := h.executor.QueryDB(secCtx.DatasourceID)
	if db == nil {
		h.writeError(w, fmt.Errorf("no database connection for datasource %s", secCtx.DatasourceID), http.StatusInternalServerError)
		return
	}

	resp, err := h.service.Execute(r.Context(), secCtx, qd, db)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	h.writeJSON(w, resp)
}

// ListBOTerms handles GET /api/business-objects/{boId}/terms?bindingId=...
func (h *QueryBuilderHandler) ListBOTerms(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	boID := chi.URLParam(r, "boId")
	bindingID := r.URL.Query().Get("bindingId")
	if boID == "" {
		h.writeError(w, fmt.Errorf("business object id is required"), http.StatusBadRequest)
		return
	}

	terms, err := h.service.GetBOTerms(ctx, secCtx, boID, bindingID)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	h.writeJSON(w, map[string]interface{}{"terms": terms})
}

func (h *QueryBuilderHandler) decodeAndAuthorize(w http.ResponseWriter, r *http.Request) (*security.Context, *boresolver.QueryDef, bool) {
	var qd boresolver.QueryDef
	if err := json.NewDecoder(r.Body).Decode(&qd); err != nil {
		h.writeError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return nil, nil, false
	}

	// Resolve tenant/datasource from the request headers/JWT (the same
	// authoritative path every other endpoint uses) - NOT from
	// qd.Context.BindingID. BindingID identifies a specific BO's physical
	// backend binding (business_object_bindings.id); it lives in a
	// different ID space than a tenant's provisioned datasource
	// (tenant_product_datasource.id) that SecurityContextFromRequest
	// resolves. Passing it in as an override made every widget/report
	// query with a populated BindingID fail datasource resolution outright.
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return nil, nil, false
	}

	// Real tenant-ownership check in place of the old (always-false)
	// string-equality comparison: the BO this query targets must actually
	// belong to the resolved tenant. Without this, decodeAndAuthorize would
	// only be enforcing "the caller has some valid tenant," not "the caller
	// may query this specific BO" - the actual cross-tenant guard this
	// function exists for.
	if qd.Context.BOID != "" {
		owned, err := h.service.BOBelongsToTenant(qd.Context.BOID, secCtx.TenantID)
		if err != nil {
			h.writeError(w, err, http.StatusInternalServerError)
			return nil, nil, false
		}
		if !owned {
			h.writeError(w, fmt.Errorf("forbidden: business object does not belong to the caller's tenant"), http.StatusForbidden)
			return nil, nil, false
		}
	}

	_ = ctx
	return secCtx, &qd, true
}

func (h *QueryBuilderHandler) writeJSON(w http.ResponseWriter, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(body); err != nil {
		logging.GetLogger().Sugar().Errorf("failed to encode response: %v", err)
	}
}

func (h *QueryBuilderHandler) writeError(w http.ResponseWriter, err error, status int) {
	if errors.Is(err, security.ErrImpersonationScopeViolation) {
		status = http.StatusForbidden
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   http.StatusText(status),
		"details": err.Error(),
	})
}
