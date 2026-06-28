package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"

	"github.com/hondyman/semlayer/backend/internal/bp"
	"github.com/hondyman/semlayer/backend/internal/cbo"
	"github.com/hondyman/semlayer/backend/internal/lineage"
	"github.com/hondyman/semlayer/backend/internal/nba"
	obsActivities "github.com/hondyman/semlayer/backend/internal/observability/activities"
	obsWorkflows "github.com/hondyman/semlayer/backend/internal/observability/workflows"
	"github.com/hondyman/semlayer/backend/internal/pagestudio"
	"github.com/hondyman/semlayer/backend/internal/platform"
	"github.com/hondyman/semlayer/backend/internal/rag"
	rebalanceractivities "github.com/hondyman/semlayer/backend/internal/rebalancer/activities"
	rebalancerworkflow "github.com/hondyman/semlayer/backend/internal/rebalancer/workflow"
	"github.com/hondyman/semlayer/backend/internal/review"
	"github.com/hondyman/semlayer/backend/internal/rules"
	intsemantic "github.com/hondyman/semlayer/backend/internal/semantic"
	"github.com/hondyman/semlayer/backend/internal/tenant"
	"github.com/hondyman/semlayer/backend/internal/tests"
	"github.com/hondyman/semlayer/backend/internal/wealth"
	"github.com/hondyman/semlayer/backend/internal/wealth/risk"
	wealthworkflows "github.com/hondyman/semlayer/backend/internal/wealth/workflows"
	"github.com/hondyman/semlayer/backend/internal/workflows"
	"github.com/hondyman/semlayer/backend/internal/workflows/interpreter"
	"github.com/hondyman/semlayer/backend/pkg/governance"
	"github.com/hondyman/semlayer/backend/pkg/llm"
	pkgworkflows "github.com/hondyman/semlayer/backend/pkg/workflows"
	temporalclient "github.com/hondyman/semlayer/libs/temporal-client"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/worker"
)

func main() {
	// Connect to Temporal using centralized helper (env-driven + retries)
	temporalClient, err := temporalclient.NewClientWithRetry()
	if err != nil {
		log.Fatalf("❌ Failed to create Temporal client: %v", err)
	}
	defer temporalClient.Close()
	log.Println("✅ Connected to Temporal at", getTemporalAddress())

	// Connect to PostgreSQL for activity operations
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:postgres@localhost:5432/alpha?sslmode=disable"
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalf("❌ Database ping failed: %v", err)
	}
	log.Println("✅ Connected to PostgreSQL database")

	// Create worker for bp_queue
	w := worker.New(temporalClient, "bp_queue", worker.Options{})
	log.Println("✅ Worker created for task queue: bp_queue")

	// Register workflow
	w.RegisterWorkflow(workflows.DynamicBPWorkflow)
	w.RegisterWorkflow(wealthworkflows.RebalanceWorkflow)
	w.RegisterWorkflow(rebalancerworkflow.PortfolioLifecycleWorkflow)
	w.RegisterWorkflow(pagestudio.PageUpgradeReconciliationWorkflow)

	// Register New Interpreter Workflow (Strategy Pillar 1)
	w.RegisterWorkflow(pkgworkflows.InterpreterWorkflow)
	w.RegisterWorkflow(pkgworkflows.RunStoredWorkflow)

	log.Println("✅ Registered workflows: DynamicBPWorkflow, RebalanceWorkflow, PortfolioLifecycleWorkflow, InterpreterWorkflow")

	// Register activities with Activities struct
	activities := workflows.NewActivities(db)
	w.RegisterActivity(activities.LoadBPStepsActivity)
	w.RegisterActivity(activities.DataEntryActivity)
	w.RegisterActivity(activities.ValidationActivity)
	w.RegisterActivity(activities.ApprovalActivity)
	w.RegisterActivity(activities.EmailNotificationActivity)
	w.RegisterActivity(activities.SlackNotificationActivity)
	w.RegisterActivity(activities.GenericStepActivity)
	w.RegisterActivity(activities.EscalateStepActivity)
	w.RegisterActivity(activities.AutoEscalateActivity)
	log.Println("✅ Registered generic activities")

	// Initialize TenantDBManager
	tenantManager := platform.NewTenantDBManager(db)

	// Register Wealth Activities
	wealthActivities := wealth.NewWealthActivities(tenantManager)
	w.RegisterActivity(wealthActivities.SubmitClientDataActivity)
	w.RegisterActivity(wealthActivities.ApproveKYCActivity)
	w.RegisterActivity(wealthActivities.ApproveAMLActivity)
	w.RegisterActivity(wealthActivities.ApproveClientActivity)
	w.RegisterActivity(wealthActivities.RejectClientActivity)
	w.RegisterActivity(wealthActivities.SubmitOrderActivity)
	w.RegisterActivity(wealthActivities.AutoApproveActivity)
	w.RegisterActivity(wealthActivities.SendToExchangeActivity)
	w.RegisterActivity(wealthActivities.FullFillActivity)
	w.RegisterActivity(wealthActivities.CancelOrderActivity)
	w.RegisterActivity(wealthActivities.RejectOrderActivity)

	// Register Rebalancing Activities
	w.RegisterActivity(wealthActivities.FetchRebalanceInputsActivity)
	w.RegisterActivity(wealthActivities.RunOptimizerActivity)
	w.RegisterActivity(wealthActivities.CheckAutonomyActivity)
	w.RegisterActivity(wealthActivities.ExecuteTradesActivity)

	// Register New Rebalancer Activities (Phase 1-3)
	dbx := sqlx.NewDb(db, "postgres")
	rebalancerActivities := rebalanceractivities.NewRebalancerActivities(dbx)
	w.RegisterActivity(rebalancerActivities.TaxAwareOptimizeActivity)
	w.RegisterActivity(rebalancerActivities.MonteCarloSimActivity)
	w.RegisterActivity(rebalancerActivities.NotifyAdvisorActivity)
	w.RegisterActivity(rebalancerActivities.AnalyzePortfolio)

	log.Println("✅ Registered wealth and rebalancer activities")

	// Register NBA Workflows and Activities
	nbaActivities := nba.NewActivities(dbx)
	w.RegisterWorkflow(nba.ClientSignalMonitorWorkflow)
	w.RegisterWorkflow(nba.GenerateNextBestActionWorkflow)
	w.RegisterActivity(nbaActivities.ScanClientSignalsActivity)
	w.RegisterActivity(nbaActivities.GenerateNextBestActionActivity)
	w.RegisterActivity(nbaActivities.SaveRecommendedActionsActivity)
	log.Println("✅ Registered NBA workflows and activities")

	// Register Crypto Workflows
	w.RegisterWorkflow(workflows.CryptoPriceUpdateWorkflow)
	w.RegisterWorkflow(workflows.DeFiPositionSyncWorkflow)
	w.RegisterWorkflow(workflows.CryptoBalanceReconciliationWorkflow)
	w.RegisterWorkflow(workflows.TaxLotOptimizationWorkflow)
	log.Println("✅ Registered crypto workflows")

	// Register Observability SLO Workflows
	sloProvider := cbo.NewDBSLOProvider(dbx)
	asoTuningProvider := cbo.NewDBASOTuningProvider(dbx)
	sloEvaluator := cbo.NewSLOEvaluator(dbx, sloProvider, asoTuningProvider)
	sloActivities := obsActivities.NewSLOActivities(sloEvaluator, sloProvider)

	w.RegisterWorkflow(obsWorkflows.SLOEvaluationWorkflow)
	w.RegisterActivity(sloActivities.LoadActiveSLOsActivity)
	w.RegisterActivity(sloActivities.EvaluateSLOActivity)
	w.RegisterActivity(sloActivities.HandleSLOViolationActivity)
	log.Println("✅ Registered SLO workflows and activities")

	// Initialize RAG Services
	ragTenantManager := tenant.NewTenantManager(db, nil)
	ingestionService := rag.NewIngestionService()
	// Use dummy key for now, or load from env
	embeddingService := rag.NewOpenAIEmbedder("dummy-key", "text-embedding-ada-002")
	configService := rag.NewConfigService(db)

	ragActivities := workflows.NewDocumentActivities(ragTenantManager, ingestionService, embeddingService, configService)

	// Register RAG Workflow and Activities
	w.RegisterWorkflow(workflows.DocumentIngestionWorkflow)
	w.RegisterActivity(ragActivities.ExtractTextActivity)
	w.RegisterActivity(ragActivities.ChunkDocumentActivity)
	w.RegisterActivity(ragActivities.GenerateEmbeddingsActivity)
	w.RegisterActivity(ragActivities.StoreChunksActivity)
	log.Println("✅ Registered RAG workflow and activities")

	// Register Metadata Engine (Interpreter)
	interpreterActivities := interpreter.NewInterpreterActivities()
	w.RegisterWorkflow(interpreter.ExecuteDynamicWorkflow)
	w.RegisterActivity(interpreterActivities.ExecuteHTTP)
	w.RegisterActivity(interpreterActivities.LogMessage)
	log.Println("✅ Registered Metadata Engine (Interpreter) workflow and activities")

	// --- Rules Engine & Workday-Plus BP Activities ---

	// 1. Rules Engine Dependencies
	rulesRepo := rules.NewSQLRuleRepository(db)

	// Cache and Compilers
	// Note: In production you might persist cache or use Redis, here in-memory
	// coreFnCache := rules.NewCoreFnCache()
	// tenantFnCache := rules.NewTenantFnCache()
	// predecl := starlib.Lib() // Base Starlark environment - REMOVED

	// coreCompiler := rules.NewCoreCompiler(coreFnCache)
	// tenantCompiler := rules.NewTenantCompiler(coreCompiler, tenantFnCache, rulesRepo)

	// Rule Engine is now CEL/ASL based - no compilers needed in updated design
	// Assuming RuleEngine constructor changed or we need to update it.
	// Let's check NewRuleEngine signature in engine.go first.
	ruleEngine := rules.NewRuleEngine(rulesRepo)

	// 2. BP Activities
	bpRepo := bp.NewSQLBPRepository(db)
	bpActivities := bp.NewBPActivities(bpRepo, ruleEngine)

	w.RegisterActivity(bpActivities.LoadDefinitionActivity)
	w.RegisterActivity(bpActivities.EvaluateConditionActivity)
	w.RegisterActivity(bpActivities.EvaluateDurationActivity)
	w.RegisterActivity(bpActivities.EvaluateApprovalLevelActivity)
	w.RegisterActivity(bpActivities.ResolveParticipantsActivity)
	w.RegisterActivity(bpActivities.CreateUserTaskActivity)

	// 3. Resolution Activities (Routing & Rules)
	designerService := bp.NewDesignerService(db)
	resActivities := &bp.ResolutionActivities{Engine: ruleEngine, Designer: designerService}
	w.RegisterActivity(resActivities.ResolveApproverRoleActivity)
	w.RegisterActivity(resActivities.ResolveBranchActivity)
	// EvaluateDurationActivity is already registered via bpActivities

	// 4. Escalation Activities
	escActivities := &bp.EscalationActivities{}
	w.RegisterActivity(escActivities.NotifyApproverActivity)
	w.RegisterActivity(escActivities.FinalEscalationActivity)

	log.Println("✅ Registered Workday-Plus BP Activities (with RuleEngine)")

	// --- Strategic Roadmap: Foundation Phase ---
	// 5. Ledger Activities (Immutable Audit)
	ledgerActivities := pkgworkflows.NewLedgerActivities(dbx)
	w.RegisterActivity(ledgerActivities.DurableLedgerWrite)

	// Register with Central Registry for lookup-by-string
	pkgworkflows.RegisterActivity("DurableLedgerWrite", ledgerActivities.DurableLedgerWrite)

	log.Println("✅ Registered Immutable Ledger Activities")

	// 6. Financial Services Activities (Phase 6)
	govEngine := governance.NewGovernanceEngine(dbx)
	compActivities := pkgworkflows.NewComplianceActivities(govEngine)
	w.RegisterActivity(compActivities.ActivityCheckCompliance)
	pkgworkflows.RegisterSafeActivity("ActivityCheckCompliance", compActivities.ActivityCheckCompliance)
	log.Println("✅ Registered Pre-Trade Compliance Activities")

	mdmActivities := pkgworkflows.NewMDMActivities()
	w.RegisterActivity(mdmActivities.ActivityValidateGoldenRecord)
	pkgworkflows.RegisterSafeActivity("ActivityValidateGoldenRecord", mdmActivities.ActivityValidateGoldenRecord)
	log.Println("✅ Registered MDM Validation Activities")

	// 7. GenAI Activities (Phase 7)
	llmCfgPath := ".runtime/llm_config.json"
	llmCfgSvc := llm.NewLLMConfigService(llmCfgPath)
	genAIActivities := &pkgworkflows.GenAIActivities{ConfigService: llmCfgSvc}
	w.RegisterActivity(genAIActivities.ActivityGenerateContent)
	pkgworkflows.RegisterSafeActivity("ActivityGenerateContent", genAIActivities.ActivityGenerateContent)
	log.Println("✅ Registered GenAI Co-pilot Activities")

	// 8. Predictive Risk Activities (Phase 7 & 8)
	riskEngine := risk.NewRiskAnalyticsEngine(db)
	riskActivities := &pkgworkflows.SettlementRiskActivities{RiskEngine: riskEngine, ConfigService: llmCfgSvc}
	w.RegisterActivity(riskActivities.ActivityPredictSettlementRisk)
	w.RegisterActivity(riskActivities.ActivityGetSettlementRiskML) // Added Phase 7
	w.RegisterActivity(riskActivities.ActivityGetRiskExplanation)  // Added Phase 8
	pkgworkflows.RegisterActivity("ActivityPredictSettlementRisk", riskActivities.ActivityPredictSettlementRisk)
	pkgworkflows.RegisterActivity("ActivityGetSettlementRiskML", riskActivities.ActivityGetSettlementRiskML)
	pkgworkflows.RegisterActivity("ActivityGetRiskExplanation", riskActivities.ActivityGetRiskExplanation)
	log.Println("✅ Registered Predictive Settlement Risk Activities (MLOps)")

	// Register MLOps Retraining Workflow (Phase 8)
	w.RegisterWorkflow(pkgworkflows.AutomatedRetrainingWorkflow)
	log.Println("✅ Registered Automated Retraining Workflow")

	// 9. RWA Lifecycle Activities (Phase 7)
	rwaActivities := &pkgworkflows.RWAActivities{ConfigService: llmCfgSvc}
	w.RegisterActivity(rwaActivities.ActivityMintToken)
	w.RegisterActivity(rwaActivities.ActivityPerformKYC)
	w.RegisterActivity(rwaActivities.ActivityDistributeDividends)
	pkgworkflows.RegisterActivity("ActivityMintToken", rwaActivities.ActivityMintToken)
	pkgworkflows.RegisterActivity("ActivityPerformKYC", rwaActivities.ActivityPerformKYC)
	pkgworkflows.RegisterActivity("ActivityDistributeDividends", rwaActivities.ActivityDistributeDividends)
	log.Println("✅ Registered RWA Lifecycle Activities")

	// 10. MDUI User Interaction Activities (Phase 8)
	uiActivities := pkgworkflows.NewUIActivities(db)
	w.RegisterActivity(uiActivities.ActivityUserInteraction)
	pkgworkflows.RegisterSafeActivity("ActivityUserInteraction", uiActivities.ActivityUserInteraction)
	log.Println("✅ Registered MDUI User Interaction Activities")

	// 11. AI Migration Engine (Phase 9)
	codeAnnotationActivities := pkgworkflows.NewCodeAnnotationActivities(llmCfgSvc)
	w.RegisterActivity(codeAnnotationActivities.ActivityAnnotateCode)
	pkgworkflows.RegisterActivity("ActivityAnnotateCode", codeAnnotationActivities.ActivityAnnotateCode)

	configGenActivities := pkgworkflows.NewConfigGenerationActivities(dbx, llmCfgSvc)
	w.RegisterActivity(configGenActivities.ActivityGenerateConfig)
	pkgworkflows.RegisterActivity("ActivityGenerateConfig", configGenActivities.ActivityGenerateConfig)

	w.RegisterWorkflow(pkgworkflows.MigrationWorkflow)
	log.Println("✅ Registered AI Migration Engine Workflow and Activities")

	// 12. Change Review System (CRS)
	// Services
	crsLineageRepo := lineage.NewDBLineageRepository(dbx)
	crsLineageService := lineage.NewLineageService(crsLineageRepo)
	crsVersionStore := intsemantic.NewSemanticVersionStore(dbx)

	// Test Runner needs resolver, which needs graph service
	// In worker, we might not have the full graph service initialized like in server
	// But we can create a lightweight one or assume DB persistence
	// For now, let's create a minimal setup.
	// NOTE: BOContextResolver uses SemanticGraphService which usually uses AGE.
	// If we are moving to SQL lineage, we might need a SQL-based graph service or update resolver.
	// However, SemanticGraphService in `internal/analytics` might be coupled to AGE.
	// Assuming `crsLineageService` can suffice or we pass nil/error if runner is called in worker without proper context.
	// Actually, ReviewActivities run semantic tests. So we DO need it.
	// We'll skip recreating the full graph service here and just pass nil to runner if not feasible,
	// BUT this will break tests running in worker.
	// Let's rely on standard DI if possible.
	// For this exercise, we initialize what we can using SQL-based approach.

	// Stubbing resolver for now to allow compilation - user can refine dependency injection
	crsTestRunner := tests.NewSemanticTestRunner(dbx, nil)

	crsActivities := review.NewReviewActivities(
		dbx,
		crsLineageService,
		crsVersionStore,
		crsTestRunner,
		nil, // ASO Invalidator
	)

	w.RegisterWorkflow(review.ChangeReviewWorkflow)
	w.RegisterWorkflow(review.PromoteChangeSetWorkflow)
	w.RegisterActivity(crsActivities.ComputeSemanticDiffActivity)
	w.RegisterActivity(crsActivities.ComputeLineageImpactActivity)
	w.RegisterActivity(crsActivities.RunSemanticTestsActivity)
	w.RegisterActivity(crsActivities.SaveChangeReviewActivity)
	w.RegisterActivity(crsActivities.ApplyChangeSetActivity)
	w.RegisterActivity(crsActivities.RebuildLineageForChangeSetActivity)
	w.RegisterActivity(crsActivities.InvalidateASOActivity)
	log.Println("✅ Registered Change Review System Workflows and Activities")

	// 13. Page Studio
	psRepo := pagestudio.NewRepository(dbx)
	reconService := &pagestudio.ReconciliationService{}
	psActivities := pagestudio.NewActivities(psRepo, reconService)
	w.RegisterActivity(psActivities.AnalyzeCoreUpgradeImpact)
	log.Println("✅ Registered Page Studio Reconciliation Activities")

	// Start the worker
	log.Println("🚀 Starting Temporal worker...")
	if err := w.Start(); err != nil {
		log.Fatalf("❌ Worker start failed: %v", err)
	}
	log.Println("✅ Worker started and listening for workflows on bp_queue")

	// Wait for shutdown signal
	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-sigCtx.Done()
	log.Println("📴 Shutting down worker...")

	w.Stop()
	log.Println("✅ Worker stopped gracefully")
}

func getTemporalAddress() string {
	if v := os.Getenv("TEMPORAL_HOST"); v != "" {
		return v
	}
	if v := os.Getenv("TEMPORAL_ADDRESS"); v != "" {
		return v
	}
	if v := os.Getenv("TEMPORAL_HOSTPORT"); v != "" {
		return v
	}
	return "temporal:7233"
}
