// Deleted - replaced by ai_proxy_routes.go & graphql_proxy.go
// DEPRECATED: replaced by ai_proxy_routes.go and graphql_proxy.go
// DEPRECATED: moved to ai_proxy_routes.go and graphql_proxy.go.
package api

import (
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// Deprecated: moved to ai_proxy_routes.go and graphql_proxy.go

// We keep this file intentionally minimal: it provides a couple of proxy
// endpoints used by the frontend in development. The real application has
// additional security and request validation — we keep the logic isolated
// here so tests and small workflows can stub out Hasura or AI service.

// second block removed - file split into ai_proxy_routes and graphql_proxy

// Deprecated: ai proxy functionality moved to ai_proxy_routes.go and graphql_proxy.go
// internal/api/ai_proxy.go
// Lightweight proxy handlers for AI endpoints and the GraphQL endpoint.
// third duplicate block removed

// withTenant middleware extracts and validates the X-Tenant-ID header.
// All deprecated content removed. Use updated proxies in ai_proxy_routes.go and graphql_proxy.go.

// Call registerAIProxyRoutes from your main api.go file in the /api route group:
// Example in registerRoutes():
//   r.Route("/api", func(r chi.Router) {
//     ...existing routes...
//     registerAIProxyRoutes(r)
//   })
