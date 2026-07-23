package api

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoleRoutes mounts role management handlers.
func RegisterRoleRoutes(r chi.Router, handler interface{ RegisterRoutes(chi.Router) }) {
	handler.RegisterRoutes(r)
}
