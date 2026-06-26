package api

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/jmoiron/sqlx"
)

func TestGoogleSyncRoutes(t *testing.T) {
	// Set env vars to enable sync routes
	t.Setenv("GOOGLE_CLIENT_ID", "test-client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "test-client-secret")
	t.Setenv("OAUTH_TOKEN_ENCRYPTION_KEY", "12345678901234567890123456789012") // 32 bytes

	// Mock DB
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Mock Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Mock QoSManager
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	qosManager := services.NewQoSManager(sqlxDB)

	// Call SetupRouter
	router := SetupRouter(db, nil, nil, nil, qosManager, nil, nil, nil, redisClient)

	// Verify Routes
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/sync/google/calendars"},
		{"POST", "/sync/google/sync"},
		{"GET", "/sync/google/status/123"},
		{"POST", "/sync/google/cancel/123"},
		{"GET", "/sync/google/active"},
		{"GET", "/sync/google/events"},
		{"GET", "/sync/conflicts"},
		{"POST", "/sync/conflicts/123/resolve"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			// We handle the route match check
			rctx := chi.NewRouteContext()
			if !router.Match(rctx, route.method, route.path) {
				t.Errorf("Route %s %s not found", route.method, route.path)
			}
		})
	}
}
