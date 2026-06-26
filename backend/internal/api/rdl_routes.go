package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/rdl"
	"github.com/jmoiron/sqlx"
)

// RegisterRDLRoutes registers all Rule Definition Language (RDL) routes for metadata-driven rebalancing
func RegisterRDLRoutes(router chi.Router, db *sqlx.DB) {
	service := rdl.NewService(db)
	handler := rdl.NewHandler(service)

	// Main RDL routes under /api/rdl
	router.Route("/api/rdl", func(r chi.Router) {
		// Rules CRUD
		r.Route("/rules", func(r chi.Router) {
			r.Post("/", handler.CreateRule)
			r.Get("/", handler.ListRules)

			r.Route("/{ruleID}", func(r chi.Router) {
				r.Get("/", handler.GetRule)
				r.Put("/", handler.UpdateRule)
				r.Delete("/", handler.DeleteRule)
			})
		})

		// Rule evaluation endpoints
		r.Route("/evaluate", func(r chi.Router) {
			// Evaluate a single rule against data
			r.Post("/", handler.EvaluateRule)

			// Evaluate all rules for a portfolio (returns applicable rules)
			r.Post("/portfolio", handler.EvaluatePortfolio)

			// Tax-Loss Harvesting specific evaluation
			r.Post("/tlh", handler.EvaluateTLH)
		})

		// Rule templates for common scenarios
		r.Route("/templates", func(r chi.Router) {
			r.Get("/", handler.ListTemplates)
			r.Get("/{templateID}", handler.GetTemplate)
		})

		// Rule validation (dry-run)
		r.Post("/validate", handler.ValidateRule)
	})

	// Shorthand routes for convenience
	router.Route("/api/rules", func(r chi.Router) {
		r.Post("/", handler.CreateRule)
		r.Get("/", handler.ListRules)

		r.Route("/{ruleID}", func(r chi.Router) {
			r.Get("/", handler.GetRule)
			r.Put("/", handler.UpdateRule)
			r.Delete("/", handler.DeleteRule)
		})
	})
}
