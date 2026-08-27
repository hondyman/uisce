package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/pagedesigner"
	"github.com/jmoiron/sqlx"
)

type AutoPageHandler struct {
	compiler *pagedesigner.AutoPageCompiler
}

func NewAutoPageHandler(db *sqlx.DB) *AutoPageHandler {
	return &AutoPageHandler{
		compiler: pagedesigner.NewAutoPageCompiler(db),
	}
}

func (h *AutoPageHandler) RegisterRoutes(r chi.Router) {
	r.Post("/page-designer/auto-compile", h.HandleAutoCompile)
}

func (h *AutoPageHandler) HandleAutoCompile(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantUUIDFromRequest(r)

	var req pagedesigner.AutoPageGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.TenantID.String() == "00000000-0000-0000-0000-000000000000" {
		req.TenantID = tenantID
	}

	spec, err := h.compiler.CompilePageGroup(r.Context(), &req)
	if err != nil {
		http.Error(w, "compilation error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(spec)
}
