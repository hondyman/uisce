package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/reporting"
	"github.com/jmoiron/sqlx"
)

// SemanticReportingHandler provides HTTP handlers for the semantic reporting API
type SemanticReportingHandler struct {
	handler *reporting.Handler
}

// NewSemanticReportingHandler creates a new semantic reporting handler
func NewSemanticReportingHandler(db *sqlx.DB, cubeURL string) *SemanticReportingHandler {
	// Build dependencies
	cubeClient := reporting.NewCubeClient(cubeURL)
	renderer := reporting.NewRenderer(cubeClient)
	repo := reporting.NewRepository(db)
	svc := reporting.NewService(repo, cubeClient, renderer)
	handler := reporting.NewHandler(svc)

	return &SemanticReportingHandler{
		handler: handler,
	}
}

// RegisterRoutes registers all semantic reporting routes on the chi router
// Routes are nested under /api so will be accessible at /api/reporting/*
func (h *SemanticReportingHandler) RegisterRoutes(r chi.Router) {
	// The reporting.Handler has its own RegisterRoutes method
	// that handles all the routing internally
	h.handler.RegisterRoutes(r)
}
