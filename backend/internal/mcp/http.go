package mcp

import (
	"context"
	"net/http"

	"github.com/mark3labs/mcp-go/server"
	"go.temporal.io/sdk/client"
)

// CutoverMarker is the unique string for the streamable HTTP cutover.
// Flip checklist: strings(deployed-binary) | grep this marker.
const CutoverMarker = "mcp-cutover-streamable-v1"

// cutoverBuildMarker keeps the literal in the binary (const alone can be
// inlined away from strings(1) probes).
var cutoverBuildMarker = []byte(CutoverMarker)

func init() {
	if len(cutoverBuildMarker) == 0 || string(cutoverBuildMarker) != CutoverMarker {
		panic("mcp cutover marker missing from binary")
	}
}

// SetTemporal wires Temporal into Path 1 tool implementations used by Server.
func (s *Server) SetTemporal(c client.Client) *Server {
	if s != nil && s.path1 != nil {
		s.path1.SetTemporal(c)
	}
	return s
}

// HTTPHandler returns the mark3labs streamable HTTP transport for /api/mcp.
// It owns POST, GET, DELETE, and HEAD — do not also register a separate
// GET info route on the same pattern (chi last-wins).
func (s *Server) HTTPHandler() http.Handler {
	return server.NewStreamableHTTPServer(
		s.registry,
		server.WithStateLess(true),
		server.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			// AuthContextMiddleware already populated AuthInfo on r.Context().
			return r.Context()
		}),
	)
}
