package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterCatalogScanRoute mounts the catalog scan POST endpoint.
// The handler must expose HandleCatalogScan(w http.ResponseWriter, r *http.Request).
func RegisterCatalogScanRoute(r chi.Router, handler interface {
	HandleCatalogScan(http.ResponseWriter, *http.Request)
}) {
	r.Post("/catalog/scan", handler.HandleCatalogScan)
}
