package api

import (
	"github.com/go-chi/chi/v5"
)

// RegisterDAXRoutes mounts DAX function endpoints. Accepts any handler with RegisterRoutes(chi.Router).
func RegisterDAXRoutes(r chi.Router, handler interface{ RegisterRoutes(chi.Router) }) {
	handler.RegisterRoutes(r)
}
