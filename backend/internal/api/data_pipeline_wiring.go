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

	store := &datapipeline.Store{DB: sqlxDB}
	if s.TemporalClient != nil {
		w := worker.New(s.TemporalClient, datapipeline.TaskQueue, worker.Options{})
		w.RegisterWorkflow(datapipeline.Workflow)
		w.RegisterWorkflow(datapipeline.ScheduledWorkflow)
		acts := &datapipeline.Activities{Store: store, Deps: deps}
		w.RegisterActivityWithOptions(acts.Run, activity.RegisterOptions{Name: datapipeline.ActivityName})
		w.RegisterActivityWithOptions(acts.ScheduledRun, activity.RegisterOptions{Name: datapipeline.ScheduledActivityName})
		if err := w.Start(); err != nil {
			log.Printf("[data-pipelines] worker failed to start: %v", err)
		}
	}
	rules := analytics.NewValidationRuleService(sqlxDB)
	catalog := func(req *http.Request, tenant string) datapipeline.PlatformCatalog {
		return &pipelineCatalog{r: req, tenant: tenant, bos: s.BusinessObjectService, crud: bo, rules: rules, deps: deps}
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
	if s.TemporalClient != nil {
		h.WithScheduler(&datapipeline.TemporalScheduler{Client: s.TemporalClient})
	}
	h.RegisterRoutes(r)
	s.DataPipelines = h
}
