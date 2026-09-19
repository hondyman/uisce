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

// SetTemporal wires Temporal for start_fix_order_entry on the unified Server.
func (s *Server) SetTemporal(c client.Client) *Server {
	if s != nil {
		s.temporal = c
	}
	return s
}

// HTTPHandler returns the mark3labs streamable HTTP transport for /api/mcp.
// It owns POST, GET, DELETE, and HEAD — do not also register a separate
// GET info route on the same pattern (chi last-wins).
//
// Session model: SessionMode=stateless (WithStateLess(true)). No
// Mcp-Session-Id; server restarts do not invalidate clients; GET is not
// used for SSE pushes (bare GET → 405 Streaming unsupported). Liveness
// probing uses ProbeStreamable / cmd/mcp-live-probe (initialize…), not
// the retired Path 1 GET info JSON. chi r.Handle("/mcp") is exact — it
// does not swallow POST /mcp/tools/call (Path 6).
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
