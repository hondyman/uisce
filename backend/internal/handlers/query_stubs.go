package handlers

import "net/http"

type QueryHandler struct{}

func NewQueryHandler(qs interface{}, securityDeps SecurityContextDeps) *QueryHandler {
	return &QueryHandler{}
}

func (h *QueryHandler) HandleExecuteQuery(w http.ResponseWriter, r *http.Request) {}
func (h *QueryHandler) HandleCompileQuery(w http.ResponseWriter, r *http.Request) {}
func (h *QueryHandler) HandleExportQuery(w http.ResponseWriter, r *http.Request)  {}
func (h *QueryHandler) HandleListHistory(w http.ResponseWriter, r *http.Request)  {}

// SavedQueryHandler moved to internal/querybuilder/saved_query_handler.go (a
// real implementation, not a stub) - it needs *querybuilder.QueryService,
// and querybuilder already imports this package for SecurityContextDeps, so
// keeping it here would be an import cycle.

type SaveExtensionRequest struct {
	ModelObject interface{}
}
