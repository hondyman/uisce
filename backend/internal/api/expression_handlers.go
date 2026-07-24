package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// ============================================================================
// EXPRESSION API HANDLERS
// ============================================================================

type ExpressionHandler struct {
	// engine *services.StarlarkEngine // Removed
	logger *zap.Logger
}

func NewExpressionHandler() *ExpressionHandler {
	logger, _ := zap.NewProduction()
	return &ExpressionHandler{
		logger: logger,
	}
}

func (h *ExpressionHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/expressions", func(r chi.Router) {
		r.Get("/", h.ListExpressions)
		r.Post("/", h.CreateExpression)
		r.Get("/{id}", h.GetExpression)
		r.Put("/{id}", h.UpdateExpression)
		r.Post("/test", h.TestExpression)
		r.Get("/bo/{boId}", h.GetExpressionsByBO)
	})
}

// ListExpressions returns all expressions for a tenant
func (h *ExpressionHandler) ListExpressions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]interface{}{})
}

// CreateExpression creates a new expression
func (h *ExpressionHandler) CreateExpression(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented (Starlark removed)", http.StatusNotImplemented)
}

// GetExpression returns an expression by ID
func (h *ExpressionHandler) GetExpression(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented (Starlark removed)", http.StatusNotImplemented)
}

// UpdateExpression updates an existing expression
func (h *ExpressionHandler) UpdateExpression(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented (Starlark removed)", http.StatusNotImplemented)
}

// TestExpression tests an expression with sample data
func (h *ExpressionHandler) TestExpression(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented (Starlark removed)", http.StatusNotImplemented)
}

// GetExpressionsByBO returns all expressions for a business object
func (h *ExpressionHandler) GetExpressionsByBO(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented (Starlark removed)", http.StatusNotImplemented)
}
