package api

import (
	"context"
	"database/sql"
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
	if err != nil || secCtx == nil || secCtx.TenantID == "" || secCtx.UserID == "" {
		return stagingbind.Actor{}, msgcat.Unauthenticated().Wrap(err)
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
