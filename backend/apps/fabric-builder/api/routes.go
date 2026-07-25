package api

import (
"database/sql"

"github.com/go-chi/chi/v5"
)

// RegisterFabricRoutes registers all fabric-related API routes
func RegisterFabricRoutes(r chi.Router, db *sql.DB) {
r.Route("/api/fabric", func(r chi.Router) {
// Fabric model endpoints
r.Get("/models", GetFabricModelsHandler(db))
r.Post("/models", CreateFabricModelHandler(db))
r.Get("/models/{id}", GetFabricModelHandler(db))
r.Put("/models/{id}", UpdateFabricModelHandler(db))
r.Delete("/models/{id}", DeleteFabricModelHandler(db))

// Extension endpoints
r.Get("/extensions", GetExtensionsHandler(db))
r.Post("/extensions", CreateExtensionHandler(db))
r.Get("/extensions/{id}", GetExtensionHandler(db))
r.Put("/extensions/{id}", UpdateExtensionHandler(db))
r.Delete("/extensions/{id}", DeleteExtensionHandler(db))

// Validation endpoints
r.Post("/models/validate", ValidateFabricModelHandler(db))
r.Get("/extensions/compatibility-report", GetCompatibilityReportHandler(db))
})
}

// RegisterBusinessProcessRoutes registers all business process API routes
func RegisterBusinessProcessRoutes(r chi.Router, db *sql.DB) {
r.Route("/api/business-process", func(r chi.Router) {
// Business process CRUD
r.Get("/", ListBusinessProcessesHandler(db))
r.Post("/", CreateBusinessProcessHandler(db))
r.Get("/{id}", GetBusinessProcessHandler(db))
r.Put("/{id}", UpdateBusinessProcessHandler(db))
r.Delete("/{id}", DeleteBusinessProcessHandler(db))

// Business process execution
r.Post("/{id}/execute", ExecuteBusinessProcessHandler(db))
r.Get("/{id}/status", GetBusinessProcessStatusHandler(db))

// Designer endpoints
r.Get("/step-types", GetStepTypesHandler(db))
r.Get("/operators", GetValidationOperatorsHandler(db))
r.Get("/events", GetWorkflowEventsHandler(db))
})
}
