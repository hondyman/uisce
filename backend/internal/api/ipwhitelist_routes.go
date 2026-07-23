package api

import (
	"github.com/go-chi/chi/v5"
)

// RegisterIPWhitelistRoutes mounts IP whitelist endpoints.
func RegisterIPWhitelistRoutes(r chi.Router, handler interface{ RegisterRoutes(chi.Router) }) {
	handler.RegisterRoutes(r)
}
