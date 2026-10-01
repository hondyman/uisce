package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/datapipeline"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/mastering"
	"github.com/hondyman/uisce/backend/internal/msgcat"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/stagingbind"
	"github.com/hondyman/uisce/backend/pkg/llm"
)

// registerDataPipelineRoutes mounts the visual loader. Its dependencies are
// this server's own: catalog validation rules on the rule engine, the
// enforced BO records API, the DataFusion file engine
// (DATAPIPELINE_ENGINE_URL) and the staging database
// (DATAPIPELINE_STAGING_DSN). A missing one disables only the nodes that
// need it. With Temporal available, runs execute on an in-process worker
// (task queue datapipeline.TaskQueue) that shares these dependencies.
func (s *Server) registerDataPipelineRoutes(r chi.Router, sqlxDB *sqlx.DB, bo *BOCRUDHandler) {
	deps := datapipeline.Deps{
		Rules: &datapipeline.CatalogRuleChecker{Rules: analytics.NewValidationRuleService(sqlxDB)}, // execution path
		BO:    NewPipelineBOClient(bo),
	}
	if u := os.Getenv("DATAPIPELINE_ENGINE_URL"); u != "" {
		deps.Files = &datapipeline.HTTPFileEngine{BaseURL: u, Token: os.Getenv("DATAPIPELINE_ENGINE_TOKEN")}
	} else {
		log.Printf("[data-pipelines] DATAPIPELINE_ENGINE_URL not set: file sources and exports are disabled")
	}
	if dsn := os.Getenv("DATAPIPELINE_STAGING_DSN"); dsn != "" {
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			log.Printf("[data-pipelines] staging database: %v (staging loads disabled)", err)
		} else {
			deps.StagingDB = db
		}
	} else {
		log.Printf("[data-pipelines] DATAPIPELINE_STAGING_DSN not set: staging loads are disabled")
	}

	// Staging bindings: how rule checks in front of a staging load read its
	// rows (business object field -> staging column), maker-checker.
	bindings := &stagingbind.Store{DB: sqlxDB}
	deps.Bindings = bindings
	editor := &stagingbind.Editor{Store: bindings}
	if deps.StagingDB != nil {
		editor.Columns = stagingColumns{db: deps.StagingDB}
	}
	(&stagingbind.Handler{Store: bindings, Editor: editor, Catalog: s.MessageCatalog, ActorFrom: s.stagingBindActor}).RegisterRoutes(r)

	// Entity mastering (canonicalize -> match -> survive -> publish) over
	// the same data plane, reading staging through these bindings.
	platform := mastering.PlatformCatalog{DB: sqlxDB}
	engine := &mastering.Engine{Rules: analytics.NewValidationRuleService(sqlxDB), Bindings: bindings, Fields: platform, GoldCopy: platform.GoldCopyTenant}
	if deps.StagingDB != nil {
		engine.Data = sqlx.NewDb(deps.StagingDB, "postgres")
	}
	(&mastering.Handler{Engine: engine, Catalog: s.MessageCatalog, ActorFrom: s.masteringActor}).RegisterRoutes(r)
	s.MasteringEngine = engine
	s.masteringRunner = &mastering.Runner{Engine: engine, Bindings: bindings}
	if engine.Data != nil {
		// A pipeline's master step masters the load it just committed.
		deps.Master = pipelineMasterer{engine: engine}
	}

	store := &datapipeline.Store{DB: sqlxDB}
	acts := &datapipeline.Activities{Store: store, Deps: deps}
	// Scheduled runs go through the one scheduler (internal/schedule) with
	// this runner; this worker only runs on-demand pipeline runs.
	s.dataPipelineRunner = &dataPipelineRunner{store: store, acts: acts}
	if s.TemporalClient != nil {
		w := worker.New(s.TemporalClient, datapipeline.TaskQueue, worker.Options{})
		w.RegisterWorkflow(datapipeline.Workflow)
		w.RegisterActivityWithOptions(acts.Run, activity.RegisterOptions{Name: datapipeline.ActivityName})
		if err := w.Start(); err != nil {
			log.Printf("[data-pipelines] worker failed to start: %v", err)
		}
	}
	rules := analytics.NewValidationRuleService(sqlxDB)
	catalog := func(req *http.Request, tenant string) datapipeline.PlatformCatalog {
		return &pipelineCatalog{r: req, tenant: tenant, bos: s.BusinessObjectService, resolver: s.DatasourceResolver, crud: bo, rules: rules, deps: deps}
	}
	var assistant *datapipeline.Assistant
	if s.LLMConfigSvc != nil {
		assistant = &datapipeline.Assistant{LLM: func(ctx context.Context, prompt string) (string, error) {
			cfg, err := s.LLMConfigSvc.Get()
			if err != nil {
				return "", err
			}
			return llm.NewGeminiProvider(cfg.APIKey, cfg.Model).GenerateResponse(ctx, prompt)
		}}
	}
	h := NewDataPipelineHandler(store, deps, s.TemporalClient).WithGrounding(catalog, assistant)
	h.WithScheduler(&pipelineCoreScheduler{srv: s, store: store})
	h.RegisterRoutes(r)
	s.DataPipelines = h
}

// stagingColumns lists a staging table's loadable columns for binding checks.
type stagingColumns struct{ db *sql.DB }

func (c stagingColumns) Columns(ctx context.Context, table string) ([]string, error) {
	tables, err := datapipeline.StagingTables(ctx, c.db)
	if err != nil {
		return nil, err
	}
	for _, t := range tables {
		if t.Table == table {
			out := make([]string, len(t.Columns))
			for i, col := range t.Columns {
				out[i] = col.Name
			}
			return out, nil
		}
	}
	return nil, nil
}

// stagingBindActor is the caller: the tenant from the token (never a header
// on its own) and the administrator roles that scope binding changes. An
// impersonating administrator can't propose or approve - maker-checker
// records must name the real people.
func (s *Server) stagingBindActor(r *http.Request) (stagingbind.Actor, error) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", s.SecurityContextDeps)
	if err != nil {
		return stagingbind.Actor{}, handlers.SecurityContextError(err)
	}
	if secCtx == nil || secCtx.TenantID == "" || secCtx.UserID == "" {
		return stagingbind.Actor{}, msgcat.Unauthenticated()
	}
	a := stagingbind.Actor{UserID: secCtx.UserID, TenantID: secCtx.TenantID, Name: msgcat.CallerName(r)}
	if auth, ok := security.AuthInfoFromContext(r.Context()); ok && !auth.ImpersonationActive {
		for _, role := range auth.Roles {
			switch role {
			case "global_admin":
				a.PlatformAdmin = true
			case "tenant_admin":
				a.TenantAdmin = true
			}
		}
	}
	return a, nil
}

// masteringActor is the caller: the tenant from the token (never a header).
// Administrators and data stewards may run mastering; anyone in the tenant
// may read it.
func (s *Server) masteringActor(r *http.Request) (mastering.Actor, error) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", s.SecurityContextDeps)
	if err != nil || secCtx == nil || secCtx.TenantID == "" || secCtx.UserID == "" {
		return mastering.Actor{}, msgcat.Unauthenticated().Wrap(err)
	}
	a := mastering.Actor{UserID: secCtx.UserID, TenantID: secCtx.TenantID, Name: msgcat.CallerName(r)}
	if auth, ok := security.AuthInfoFromContext(r.Context()); ok && !auth.ImpersonationActive {
		for _, role := range auth.Roles {
			switch role {
			case "global_admin", "tenant_admin":
				a.CanRun, a.Admin = true, true
			case "data_steward":
				a.CanRun = true
			}
		}
	}
	return a, nil
}

// pipelineMasterer masters a pipeline's committed staging load on the
// mastering engine: the load's own idempotency key (load:<id>), so a load
// already mastered returns its run; the pipeline run is recorded on it.
type pipelineMasterer struct{ engine *mastering.Engine }

func (m pipelineMasterer) MasterLoad(ctx context.Context, r datapipeline.MasterRequest) (*datapipeline.MasterResult, error) {
	req := mastering.RunRequest{Entity: r.Entity, StagingTable: r.StagingTable, LoadRunID: r.LoadRunID, Trigger: "external",
		WorkflowID: "pipeline:" + r.PipelineRunID, StartedBy: "pipeline:" + r.PipelineRunID}
	run, fresh, err := m.engine.Start(ctx, r.TenantID, req)
	if err != nil {
		return nil, err
	}
	if fresh {
		if run, err = m.engine.Execute(ctx, r.TenantID, run, req); err != nil && run == nil {
			return nil, err
		}
	}
	out := &datapipeline.MasterResult{Entity: r.Entity, RunID: run.ID, Status: run.Status, Replayed: !fresh}
	var c mastering.Counts
	if len(run.RawCounts) > 0 && json.Unmarshal(run.RawCounts, &c) == nil {
		out.Records, out.Published, out.HeldForReview, out.Exceptions = c.Records, c.Published, c.HeldForReview, c.Exceptions
		out.Valid, out.Invalid = c.Valid, c.Invalid
	}
	if err == nil && run.Status == "FAILED" {
		detail := ""
		if run.ErrorDetail != nil {
			detail = *run.ErrorDetail
		}
		err = fmt.Errorf("mastering run %s failed: %s", run.ID, detail)
	}
	return out, err
}
