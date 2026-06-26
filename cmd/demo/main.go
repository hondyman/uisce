package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/hondyman/semlayer/internal/activities"
	"github.com/hondyman/semlayer/internal/ai"
	"github.com/hondyman/semlayer/internal/drift"
	"github.com/hondyman/semlayer/internal/uar"
	"github.com/hondyman/semlayer/internal/workflows"
)

func main() {
	ctx := context.Background()

	// -------------------------------------------------
	// 1️⃣ Temporal client
	temporalClient, err := client.NewClient(client.Options{})
	if err != nil {
		log.Fatalf("temporal client: %v", err)
	}
	defer temporalClient.Close()

	// -------------------------------------------------
	// 2️⃣ Choose UAR store (in‑memory for quick demo, Postgres for prod)
	var uarStore uar.UARStore
	if dsn := os.Getenv("DATABASE_DSN"); dsn != "" {
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			log.Fatalf("open postgres: %v", err)
		}
		uarStore = uar.NewPostgresStore(db)
		log.Println("✅ Using Postgres UAR store")
	} else {
		uarStore = uar.NewInMemoryUAR()
		log.Println("✅ Using in-memory UAR store")
	}

	// -------------------------------------------------
	// 3️⃣ Initialise Gemini client (requires GOOGLE_GENERATIVE_AI_API_KEY)
	var gemini *ai.GeminiClient
	if os.Getenv("GOOGLE_GENERATIVE_AI_API_KEY") != "" {
		gemini, err = ai.NewGeminiClient(ctx)
		if err != nil {
			log.Printf("⚠️  Gemini init failed – will fall back to mock proposals: %v", err)
		} else {
			log.Println("✅ Gemini AI client initialized")
		}
	} else {
		log.Println("ℹ️  GOOGLE_GENERATIVE_AI_API_KEY not set – using mock AI proposals")
	}

	// -------------------------------------------------
	// 4️⃣ Initialise Drift Calculator (requires STARROCKS_HOST for real queries)
	driftCalc, err := drift.NewCalculator(ctx)
	if err != nil {
		log.Printf("⚠️  Drift calculator init failed – will use mock drift: %v", err)
		driftCalc = nil
	} else {
		if os.Getenv("STARROCKS_HOST") != "" || os.Getenv("STARROCKS_DSN") != "" {
			log.Println("✅ Drift calculator initialized (StarRocks/Iceberg)")
		} else {
			log.Println("ℹ️  STARROCKS_HOST not set – using mock drift data")
		}
	}

	// -------------------------------------------------
	// 5️⃣ Bundle activities
	acts := &activities.Activities{
		UARStore:        uarStore,
		GeminiClient:    gemini, // may be nil → mock fallback inside activity
		DriftCalculator: driftCalc,
	}

	// -------------------------------------------------
	// 6️⃣ Register worker & activities
	taskQueue := "rebalancer-tq"
	w := worker.New(temporalClient, taskQueue, worker.Options{})
	w.RegisterActivity(acts)
	w.RegisterActivity(acts.CheckDriftActivity)
	w.RegisterActivity(acts.GenerateAIProposalActivity)
	w.RegisterActivity(acts.PolicyCheckActivity)
	w.RegisterActivity(acts.NotifyAdvisorActivity)
	w.RegisterActivity(acts.ExecuteTradeSagaActivity)
	w.RegisterActivity(acts.PersistUARActivity)
	w.RegisterWorkflow(workflows.RebalanceWorkflow)

	if err := w.Start(); err != nil {
		log.Fatalf("worker start: %v", err)
	}
	log.Println("✅ Temporal worker started")

	// -------------------------------------------------
	// 7️⃣ Start a demo workflow
	we, err := temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:        "rebalancer-demo-" + time.Now().Format("150405"),
			TaskQueue: taskQueue,
		},
		workflows.RebalanceWorkflow,
		workflows.RebalanceInput{
			TenantID:    "demo_tenant",
			PortfolioID: "demo_portfolio",
		},
	)
	if err != nil {
		log.Fatalf("start workflow: %v", err)
	}
	log.Printf("🚀 workflow started – ID=%s run=%s", we.GetID(), we.GetRunID())

	// -------------------------------------------------
	// 8️⃣ Auto‑approve after the SLA timer (so the demo finishes automatically)
	go func() {
		// The workflow uses a 2‑minute SLA; we wait a bit longer.
		time.Sleep(30 * time.Second)

		sig := workflows.ApprovalSignal{
			Approved:  true,
			AdvisorID: "demo_advisor",
			Rationale: "demo auto‑approve",
			Time:      time.Now(),
		}
		if err := temporalClient.SignalWorkflow(ctx, we.GetID(), we.GetRunID(), "AdvisorApproval", sig); err != nil {
			log.Printf("auto‑approve signal error: %v", err)
		} else {
			log.Println("✅ auto‑approval signal sent")
		}
	}()

	// Let the demo run, then shut down gracefully.
	time.Sleep(90 * time.Second)
	w.Stop()
	log.Println("🛑 worker stopped")

	// Clean up drift calculator
	if driftCalc != nil {
		_ = driftCalc.Close()
	}
}
