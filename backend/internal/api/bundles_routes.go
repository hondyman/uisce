package api

import (
	"github.com/go-chi/chi/v5"
)

// RegisterBundleRoutes is a thin wrapper that mounts bundle-related handlers.
// The handler parameter is any value that exposes RegisterRoutes(chi.Router).
func RegisterBundleRoutes(r chi.Router, handler interface{ RegisterRoutes(chi.Router) }) {
	handler.RegisterRoutes(r)
}
