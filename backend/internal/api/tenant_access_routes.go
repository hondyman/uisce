package api

import (
	"github.com/go-chi/chi/v5"
)

// RegisterTenantAccessRoutes mounts tenant access endpoints.
func RegisterTenantAccessRoutes(r chi.Router, handler interface{ RegisterRoutes(chi.Router) }) {
	handler.RegisterRoutes(r)
}
