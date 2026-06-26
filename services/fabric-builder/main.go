package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hondyman/semlayer/services/fabric-builder/api"
	"github.com/hondyman/semlayer/services/fabric-builder/db"
)

func main() {
	log.Println("🚀 Starting Fabric Builder Service...")

	// Initialize database connection
	database, err := db.NewConnection()
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer database.Close()
	log.Println("✅ Database connection established")

	// Initialize Hasura GraphQL client
	hasuraClient := api.NewHasuraClientFromEnv()
	log.Println("✅ Hasura GraphQL client initialized")

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Simple CORS middleware to allow Vite dev server origins during local development
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			origin := req.Header.Get("Origin")
			if origin == "http://localhost:5173" || origin == "http://localhost:5174" || strings.HasPrefix(origin, "http://127.0.0.1:517") {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization, X-Request-ID, X-Tenant-Datasource-ID, X-Tenant-ID, X-User-ID, X-Hasura-Admin-Secret")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			}
			if req.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, req)
		})
	}

	// Register CORS early so preflight (OPTIONS) requests are handled before route matching
	r.Use(corsMiddleware)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Register API routes
	api.RegisterFabricRoutes(r, database)

	// Register both SQL-based (v1) and GraphQL-based (v2) business process routes
	// v1: Legacy SQL handlers at /api/business-process/*
	api.RegisterBusinessProcessRoutes(r, database)
	// v2: GraphQL handlers at /api/business-process/v2/* (preferred for scalability)
	api.RegisterBusinessProcessGraphQLRoutes(r, hasuraClient)

	log.Println("✅ Fabric Builder Service started successfully")
	log.Println("📊 Registered endpoints:")
	log.Println("   - /api/fabric/*")
	log.Println("   - /api/business-process/* (v1 - SQL)")
	log.Println("   - /api/business-process/v2/* (v2 - GraphQL)")

	log.Fatal(http.ListenAndServe(":8081", r))
}
