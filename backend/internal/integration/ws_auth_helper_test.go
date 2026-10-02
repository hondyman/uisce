package integration_test

import (
	"context"
	"net/http"
	"strings"

	"github.com/hondyman/uisce/backend/internal/events"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/security"
)

type stubResolver struct{}

func (stubResolver) Resolve(context.Context, string) (*security.ResolvedDatasource, error) {
	return nil, security.ErrDatasourceNotAvailable
}

// newTestWSHandler returns the events WebSocket handler behind a test
// authenticator. The handler requires an authenticated caller and takes its
// tenant from that identity (validating any tenant_id query parameter against
// it). In production AuthContextMiddleware establishes the identity; these
// harness tests stand in for it by authenticating each connection as the tenant it
// asks for, so they still exercise the handler's own authorization path.
func newTestWSHandler(broker *events.EventStreamBroker) http.Handler {
	h := handlers.NewWebSocketEventHandler(broker, handlers.SecurityContextDeps{Resolver: stubResolver{}})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		ctx := security.WithAuthInfo(r.Context(), security.AuthInfo{
			UserID: "test-user", TenantIDs: []string{tenant}, ActiveTenantID: tenant,
		})
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}
