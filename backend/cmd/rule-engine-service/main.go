package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Load configuration
	port := getEnv("PORT", "8083")
	databaseURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable")
	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")

	logger.Info("Starting Rule Engine Service",
		zap.String("port", port),
		zap.String("database_url", maskURL(databaseURL)),
		zap.String("kafka_brokers", kafkaBrokers),
	)

	// Connect to database
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		logger.Fatal("Database connection check failed", zap.Error(err))
	}
	logger.Info("Database connection established")

	// Kafka is used for routing; no RabbitMQ connection attempted here.

	// Initialize business object service (as InstanceProvider)
	boService := services.NewBusinessObjectService(db)

	// Initialize rule engine
	ruleEngine := services.NewValidationRuleEngine(db, boService)

	// Create HTTP router
	router := chi.NewRouter()

	// Middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Tenant-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// JWT Middleware - validates JWT on all routes except /health and /metrics
	publicPaths := []string{"/health", "/metrics", "/docs"}
	jwtMiddleware := jwtmiddleware.NewJWTMiddleware(publicPaths...)
	router.Use(jwtMiddleware.Handler)

	// Health check
	router.Get("/health", healthHandler(logger))

	// Metrics
	router.Get("/metrics", metricsHandler(logger))

	// Rule Engine API routes
	router.Route("/api/rules", func(r chi.Router) {
		// List rules for BP/step
		r.Get("/", listRulesHandler(db, logger))

		// Create rule
		r.Post("/", createRuleHandler(db, logger))

		// Get specific rule
		r.Get("/{ruleID}", getRuleHandler(db, logger))

		// Update rule
		r.Put("/{ruleID}", updateRuleHandler(db, logger))

		// Delete rule
		r.Delete("/{ruleID}", deleteRuleHandler(db, logger))

		// Validate rule syntax
		r.Post("/validate", validateRuleSyntaxHandler(ruleEngine, logger))

		// Evaluate rule with test data
		r.Post("/evaluate", evaluateRuleHandler(ruleEngine, logger))
	})

	// Validation Rules endpoint with faceted search and lazy loading
	router.Route("/api/validation-rules", func(r chi.Router) {
		// List rules with facets and pagination
		r.Get("/", listValidationRulesWithFacetsHandler(db, logger))
	})

	// Rule templates
	router.Route("/api/templates", func(r chi.Router) {
		// List templates
		r.Get("/", listTemplatesHandler(db, logger))

		// Get template
		r.Get("/{templateID}", getTemplateHandler(db, logger))

		// Create template
		r.Post("/", createTemplateHandler(db, logger))
	})

	// Start server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("Rule Engine Service listening", zap.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server error", zap.Error(err))
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down Rule Engine Service")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}
	logger.Info("Rule Engine Service stopped")
}

// ============================================================================
// HTTP HANDLERS
// ============================================================================

func healthHandler(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "healthy",
			"service":   "rule-engine-service",
			"timestamp": time.Now().UTC(),
		})
	}
}

func metricsHandler(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "# HELP rule_engine_rules_total Total rules in system\n")
		fmt.Fprintf(w, "# TYPE rule_engine_rules_total gauge\n")
		fmt.Fprintf(w, "rule_engine_rules_total 0\n")
	}
}

func listRulesHandler(db *sqlx.DB, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract JWT claims for tenant isolation
		claims := jwtmiddleware.GetClaimsFromContext(r)
		if claims == nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		tenantID := claims.TenantID
		bpName := r.URL.Query().Get("bp_name")
		stepName := r.URL.Query().Get("step_name")
		limit := 50

		query := `
			SELECT id, tenant_id, bp_name, step_name, name, description, 
				   condition_type, condition_json, is_active, created_at, updated_at
			FROM validation_rules
			WHERE tenant_id = $1
		`
		args := []interface{}{tenantID}

		if bpName != "" {
			query += ` AND bp_name = $` + strconv.Itoa(len(args)+1)
			args = append(args, bpName)
		}

		if stepName != "" {
			query += ` AND step_name = $` + strconv.Itoa(len(args)+1)
			args = append(args, stepName)
		}

		query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(len(args)+1)
		args = append(args, limit)

		var rules []map[string]interface{}
		err := db.SelectContext(r.Context(), &rules, query, args...)
		if err != nil {
			logger.Error("Failed to list rules", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to list rules"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count":     len(rules),
			"rules":     rules,
			"timestamp": time.Now().UTC(),
		})
	}
}

func createRuleHandler(db *sqlx.DB, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract JWT claims for tenant isolation
		claims := jwtmiddleware.GetClaimsFromContext(r)
		if claims == nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		tenantID := claims.TenantID

		var req struct {
			BPName        string                 `json:"bp_name"`
			StepName      string                 `json:"step_name"`
			Name          string                 `json:"name"`
			Description   string                 `json:"description"`
			ConditionType string                 `json:"condition_type"` // "AND", "OR", "NOT"
			ConditionJSON map[string]interface{} `json:"condition_json"`
			IsActive      bool                   `json:"is_active"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
			return
		}

		query := `
			INSERT INTO validation_rules 
			(tenant_id, bp_name, step_name, name, description, condition_type, condition_json, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
			RETURNING id
		`

		var ruleID string
		conditionJSON, _ := json.Marshal(req.ConditionJSON)
		err := db.GetContext(r.Context(), &ruleID, query,
			tenantID, req.BPName, req.StepName, req.Name, req.Description,
			req.ConditionType, string(conditionJSON), req.IsActive)

		if err != nil {
			logger.Error("Failed to create rule", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to create rule"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": ruleID})
	}
}

func getRuleHandler(db *sqlx.DB, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract JWT claims for tenant isolation
		claims := jwtmiddleware.GetClaimsFromContext(r)
		if claims == nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		tenantID := claims.TenantID
		ruleID := chi.URLParam(r, "ruleID")

		var rule map[string]interface{}
		query := `SELECT * FROM validation_rules WHERE id = $1 AND tenant_id = $2`
		err := db.GetContext(r.Context(), &rule, query, ruleID, tenantID)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "rule not found"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(rule)
	}
}

func updateRuleHandler(db *sqlx.DB, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract JWT claims for tenant isolation
		claims := jwtmiddleware.GetClaimsFromContext(r)
		if claims == nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		tenantID := claims.TenantID
		ruleID := chi.URLParam(r, "ruleID")

		var req struct {
			Name          string                 `json:"name"`
			Description   string                 `json:"description"`
			ConditionJSON map[string]interface{} `json:"condition_json"`
			IsActive      bool                   `json:"is_active"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
			return
		}

		query := `
			UPDATE validation_rules
			SET name = $1, description = $2, condition_json = $3, is_active = $4, updated_at = NOW()
			WHERE id = $5 AND tenant_id = $6
		`

		conditionJSON, _ := json.Marshal(req.ConditionJSON)
		result, err := db.ExecContext(r.Context(), query,
			req.Name, req.Description, string(conditionJSON), req.IsActive, ruleID, tenantID)

		if err != nil {
			logger.Error("Failed to update rule", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to update rule"})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "rule not found"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": ruleID, "status": "updated"})
	}
}

func deleteRuleHandler(db *sqlx.DB, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract JWT claims for tenant isolation
		claims := jwtmiddleware.GetClaimsFromContext(r)
		if claims == nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		tenantID := claims.TenantID
		ruleID := chi.URLParam(r, "ruleID")

		query := `DELETE FROM validation_rules WHERE id = $1 AND tenant_id = $2`
		result, err := db.ExecContext(r.Context(), query, ruleID, tenantID)

		if err != nil {
			logger.Error("Failed to delete rule", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to delete rule"})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "rule not found"})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func validateRuleSyntaxHandler(ruleEngine services.ValidationRuleEngine, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract JWT claims for tenant isolation
		claims := jwtmiddleware.GetClaimsFromContext(r)
		if claims == nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		var req struct {
			ID            string                 `json:"id"`
			TenantID      string                 `json:"tenant_id"`
			BPName        string                 `json:"bp_name"`
			StepName      string                 `json:"step_name"`
			ConditionJSON map[string]interface{} `json:"condition_json"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
			return
		}

		// Create a ValidationRuleDefinition to validate
		rule := services.ValidationRuleDefinition{
			ID:       req.ID,
			TenantID: req.TenantID,
			BPName:   req.BPName,
			StepName: req.StepName,
		}

		// Convert map to JSON
		conditionJSON, err := json.Marshal(req.ConditionJSON)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"valid": false,
				"error": "invalid condition JSON",
			})
			return
		}
		rule.ConditionJSON = conditionJSON

		// Attempt to evaluate the condition to validate syntax
		_, err = ruleEngine.EvaluateComplexCondition(r.Context(), req.TenantID, services.ComplexCondition{}, map[string]interface{}{})
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"valid": false,
				"error": err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"valid": true})
	}
}

func evaluateRuleHandler(ruleEngine services.ValidationRuleEngine, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract JWT claims for tenant isolation
		claims := jwtmiddleware.GetClaimsFromContext(r)
		if claims == nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		var req struct {
			ID            string                 `json:"id"`
			TenantID      string                 `json:"tenant_id"`
			BPName        string                 `json:"bp_name"`
			StepName      string                 `json:"step_name"`
			ConditionJSON map[string]interface{} `json:"condition_json"`
			TestData      map[string]interface{} `json:"test_data"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
			return
		}

		// Create a ValidationRuleDefinition to evaluate
		rule := services.ValidationRuleDefinition{
			ID:       req.ID,
			TenantID: claims.TenantID,
			BPName:   req.BPName,
			StepName: req.StepName,
		}

		// Convert map to JSON
		conditionJSON, err := json.Marshal(req.ConditionJSON)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"result": false,
				"error":  "invalid condition JSON",
			})
			return
		}
		rule.ConditionJSON = conditionJSON

		// Evaluate rule
		result, err := ruleEngine.EvaluateRule(r.Context(), claims.TenantID, rule, req.TestData)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"result": false,
				"error":  err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result": result,
			"passed": result.Passed,
		})
	}
}

func listTemplatesHandler(db *sqlx.DB, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract JWT claims for tenant isolation
		claims := jwtmiddleware.GetClaimsFromContext(r)
		if claims == nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		var templates []map[string]interface{}
		query := `SELECT id, name, description, condition_type, condition_json, created_at FROM rule_templates ORDER BY created_at DESC LIMIT 50`

		err := db.SelectContext(r.Context(), &templates, query)
		if err != nil {
			logger.Error("Failed to list templates", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to list templates"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count":     len(templates),
			"templates": templates,
			"timestamp": time.Now().UTC(),
		})
	}
}

func getTemplateHandler(db *sqlx.DB, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract JWT claims for tenant isolation
		claims := jwtmiddleware.GetClaimsFromContext(r)
		if claims == nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		templateID := chi.URLParam(r, "templateID")

		var template map[string]interface{}
		query := `SELECT * FROM rule_templates WHERE id = $1`
		err := db.GetContext(r.Context(), &template, query, templateID)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "template not found"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(template)
	}
}

func createTemplateHandler(db *sqlx.DB, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract JWT claims for tenant isolation
		claims := jwtmiddleware.GetClaimsFromContext(r)
		if claims == nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		var req struct {
			Name          string                 `json:"name"`
			Description   string                 `json:"description"`
			ConditionType string                 `json:"condition_type"`
			ConditionJSON map[string]interface{} `json:"condition_json"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
			return
		}

		query := `
			INSERT INTO rule_templates (name, description, condition_type, condition_json, created_at)
			VALUES ($1, $2, $3, $4, NOW())
			RETURNING id
		`

		var templateID string
		conditionJSON, _ := json.Marshal(req.ConditionJSON)
		err := db.GetContext(r.Context(), &templateID, query,
			req.Name, req.Description, req.ConditionType, string(conditionJSON))

		if err != nil {
			logger.Error("Failed to create template", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to create template"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": templateID})
	}
}

// listValidationRulesWithFacetsHandler returns paginated rules with facet metadata
func listValidationRulesWithFacetsHandler(db *sqlx.DB, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract JWT claims for tenant isolation
		claims := jwtmiddleware.GetClaimsFromContext(r)
		if claims == nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		// Get pagination parameters
		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
				page = parsed
			}
		}
		limit := 20
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				limit = parsed
			}
		}

		// Get filter parameters
		searchQuery := r.URL.Query().Get("search")
		entitiesStr := r.URL.Query().Get("entities")
		ruleTypesStr := r.URL.Query().Get("rule_types")
		severitiesStr := r.URL.Query().Get("severities")

		offset := (page - 1) * limit

		// Build WHERE clause with tenant isolation
		whereClause := " WHERE tenant_id = $1"
		args := []interface{}{claims.TenantID}

		if searchQuery != "" {
			whereClause += " WHERE (name ILIKE $" + strconv.Itoa(len(args)+1) + " OR description ILIKE $" + strconv.Itoa(len(args)+1) + ")"
			searchPattern := "%" + searchQuery + "%"
			args = append(args, searchPattern, searchPattern)
		}

		// Parse entity filters
		if entitiesStr != "" {
			entities := strings.Split(entitiesStr, ",")
			if whereClause != "" {
				whereClause += " AND "
			} else {
				whereClause += " WHERE "
			}
			placeholders := []string{}
			for _, entity := range entities {
				placeholders = append(placeholders, "$"+strconv.Itoa(len(args)+1))
				args = append(args, entity)
			}
			whereClause += "bp_name IN (" + strings.Join(placeholders, ",") + ")"
		}

		// Parse rule type filters
		if ruleTypesStr != "" {
			ruleTypes := strings.Split(ruleTypesStr, ",")
			if whereClause != "" {
				whereClause += " AND "
			} else {
				whereClause += " WHERE "
			}
			placeholders := []string{}
			for _, rt := range ruleTypes {
				placeholders = append(placeholders, "$"+strconv.Itoa(len(args)+1))
				args = append(args, rt)
			}
			whereClause += "condition_type IN (" + strings.Join(placeholders, ",") + ")"
		}

		// Parse severity filters
		if severitiesStr != "" {
			// Note: This assumes a severity column exists. Adjust based on actual schema
			severities := strings.Split(severitiesStr, ",")
			if whereClause != "" {
				whereClause += " AND "
			} else {
				whereClause += " WHERE "
			}
			placeholders := []string{}
			for _, severity := range severities {
				placeholders = append(placeholders, "$"+strconv.Itoa(len(args)+1))
				args = append(args, severity)
			}
			// Use the placeholders to build an IN clause for severity
			whereClause += "severity IN (" + strings.Join(placeholders, ",") + ")"
		}

		// Get total count
		countQuery := "SELECT COUNT(*) as count FROM validation_rules" + whereClause
		var totalCount int
		err := db.GetContext(r.Context(), &totalCount, countQuery, args...)
		if err != nil {
			logger.Error("Failed to count rules", zap.Error(err))
			totalCount = 0
		}

		// Get paginated rules
		args = append(args, limit, offset)
		listQuery := `
			SELECT id, tenant_id, bp_name, step_name, name, description, 
				   condition_type, condition_json, is_active, created_at, updated_at
			FROM validation_rules
		` + whereClause + `
			ORDER BY created_at DESC
			LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args)) + `
		`

		var rules []map[string]interface{}
		err = db.SelectContext(r.Context(), &rules, listQuery, args...)
		if err != nil {
			logger.Error("Failed to list rules", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to list rules"})
			return
		}

		// Get facet counts
		facetQueries := map[string]string{
			"bp_names": `
				SELECT DISTINCT bp_name as value, COUNT(*) as count 
				FROM validation_rules` + whereClause + `
				GROUP BY bp_name 
				ORDER BY count DESC
			`,
			"condition_types": `
				SELECT DISTINCT condition_type as value, COUNT(*) as count 
				FROM validation_rules` + whereClause + `
				GROUP BY condition_type 
				ORDER BY count DESC
			`,
		}

		type FacetOption struct {
			Value string `json:"value"`
			Count int    `json:"count"`
		}

		facets := make(map[string][]FacetOption)
		for facetType, query := range facetQueries {
			var options []FacetOption
			err := db.SelectContext(r.Context(), &options, query, args[:len(args)-2]...)
			if err != nil {
				logger.Warn("Failed to get facets", zap.String("facet_type", facetType), zap.Error(err))
				options = []FacetOption{}
			}
			facets[facetType] = options
		}

		hasMore := (offset + limit) < totalCount

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rules":     rules,
			"total":     totalCount,
			"page":      page,
			"limit":     limit,
			"has_more":  hasMore,
			"facets":    facets,
			"timestamp": time.Now().UTC(),
		})
	}
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func maskURL(url string) string {
	if len(url) > 30 {
		return url[:15] + "..." + url[len(url)-10:]
	}
	return url
}
