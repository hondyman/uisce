package api

import (
	"github.com/go-chi/chi/v5"
)

// RegisterDomainRoutes mounts data-domain handlers under the provided router.
func RegisterDomainRoutes(r chi.Router, handler interface{ RegisterRoutes(chi.Router) }) {
	handler.RegisterRoutes(r)
}
