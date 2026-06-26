package backend

// INTEGRATION_GUIDE.go
// This file shows how to wire all security components together in your main application

import (
	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/activities"
	"github.com/hondyman/semlayer/backend/internal/api"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/backend/workflows"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// WireSecuritySubsystem initializes and wires all security components.
func WireSecuritySubsystem(db *sqlx.DB, router chi.Router, temporalClient client.Client) error {
	// 1. Initialize repositories
	accessRuleRepo := security.NewAccessRuleRepository(db)

	// 2. Initialize services
	dslValidator := security.NewDslValidator(db)
	impactAnalyzer := security.NewImpactAnalyzer(db, accessRuleRepo)
	accessRuleService := security.NewAccessRuleService(db)

	// 3. Initialize API handlers
	securityHandler := api.NewSecurityRulesHandler(accessRuleService)

	// 4. Register routes
	securityHandler.RegisterRoutes(router)

	// 5. Initialize Temporal activities (if using Temporal)
	if temporalClient != nil {
		activities := activities.NewAccessRuleActivities(accessRuleRepo, dslValidator, impactAnalyzer)

		// Create Temporal worker
		w := worker.New(temporalClient, "security-workflows", worker.Options{})

		// Register workflow
		w.RegisterWorkflow(workflows.PromoteAccessRuleWorkflow)

		// Register activities
		w.RegisterActivity(activities.LoadRuleActivity)
		w.RegisterActivity(activities.ValidateRuleSyntaxActivity)
		w.RegisterActivity(activities.ImpactAnalysisActivity)
		w.RegisterActivity(activities.RunSecurityTestsActivity)
		w.RegisterActivity(activities.PromoteRuleActivity)
		w.RegisterActivity(activities.EmitAuditAndInvalidateCacheActivity)

		// Start worker
		if err := w.Start(); err != nil {
			return err
		}
	}

	return nil
}

// Example main.go integration:

/*
func main() {
	// Load config
	cfg := loadConfig()

	// Initialize database
	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize router
	router := mux.NewRouter()

	// Initialize Temporal client (optional)
	temporalClient, err := client.Dial(client.Options{
		HostPort: cfg.TemporalHostPort,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer temporalClient.Close()

	// Wire security subsystem
	if err := WireSecuritySubsystem(db, router, temporalClient); err != nil {
		log.Fatal(err)
	}

	// Start server
	log.Printf("Starting server on %s", cfg.ServerAddr)
	log.Fatal(http.ListenAndServe(cfg.ServerAddr, router))
}
*/

// Middleware example for extracting principal from JWT/session:

/*
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract user from JWT/session
		userID := extractUserIDFromJWT(r)
		groups := extractGroupsFromLDAP(userID)

		// Create principal
		principal := services.Principal{
			UserID: userID,
			Groups: groups,
		}

		// Store in context
		ctx := services.WithPrincipal(r.Context(), principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Apply to all routes that need security:
router.Use(AuthMiddleware)
*/

// Business Object Service integration example:

/*
type BusinessObjectService struct {
	db           *sqlx.DB
	securityRepo *services.AccessRuleRepository
}

func (s *BusinessObjectService) GetInstance(ctx context.Context, boID, instanceID string) (*Instance, error) {
	// 1. Get principal from context
	principal, err := services.PrincipalFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	// 2. Resolve access decision
	decision, err := s.securityRepo.ResolveAccessDecision(ctx, principal, boID)
	if err != nil {
		return nil, err
	}

	if decision.AccessLevel == services.AccessLevelNone {
		return nil, services.ErrForbidden
	}

	// 3. Build query with row predicate
	query := fmt.Sprintf(`
		SELECT * FROM business_object_instances
		WHERE business_object_id = $1 AND instance_id = $2
	`)

	if decision.RowPredicate != "" {
		query += fmt.Sprintf(" AND (%s)", decision.RowPredicate)
	}

	// 4. Execute query
	var instance Instance
	if err := s.db.GetContext(ctx, &instance, query, boID, instanceID); err != nil {
		return nil, err
	}

	// 5. Apply column masks
	applyColumnMasks(&instance, decision.ColumnMasks)

	return &instance, nil
}

func applyColumnMasks(instance *Instance, masks map[string]string) {
	for field, maskType := range masks {
		switch maskType {
		case "HIDE":
			// Remove field from instance
			delete(instance.Fields, field)
		case "MASK":
			// Obfuscate field value
			if val, exists := instance.Fields[field]; exists {
				instance.Fields[field] = maskValue(val)
			}
		}
	}
}
*/
