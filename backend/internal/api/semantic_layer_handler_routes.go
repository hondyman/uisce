package api

import (
	"github.com/go-chi/chi/v5"
)

func (h *SemanticLayerHandler) RegisterRoutes(r chi.Router) {
	r.Route("/semantic", func(r chi.Router) { // Relative to /api group
		r.Get("/cubes", h.ListCubes)
		r.Post("/cubes", h.CreateCube)
		r.Get("/cubes/{name}", h.GetCube)
		r.Put("/cubes/{name}", h.UpdateCube)
		r.Post("/cubes/{name}/dimensions", h.CreateDimension)
		r.Post("/cubes/{name}/measures", h.CreateMeasure)
		r.Post("/query", h.ExecuteQuery)
		r.Post("/query/sql", h.GenerateSQL)
		r.Post("/plan", h.PlanHandler)
		r.Get("/analytics/history", h.GetQueryHistory)
		r.Get("/analytics/performance", h.GetPerformanceMetrics)
		r.Get("/bundles/{domain}", h.GetBundle)
	})
}
