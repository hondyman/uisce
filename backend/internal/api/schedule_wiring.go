package api

import (
	"net/http"
	"os"
	"regexp"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/activity"
	temporalclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/msgcat"
	"github.com/hondyman/uisce/backend/internal/reports"
	"github.com/hondyman/uisce/backend/internal/schedule"
	"github.com/hondyman/uisce/backend/internal/security"
)

var alphaDB = regexp.MustCompile(`/alpha([?]|$)`)

// calendarDSN is where the MDM calendars live (crims): SCHEDULE_CALENDAR_DSN,
// else the platform DSN pointed at crims.
func calendarDSN() string {
	if v := os.Getenv("SCHEDULE_CALENDAR_DSN"); v != "" {
		return v
	}
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	return alphaDB.ReplaceAllString(dsn, "/crims$1")
}

// registerScheduleRoutes wires the one scheduler: runners for reports and
// saved queries, the Temporal engine and worker, and /api/schedules.
func (s *Server) registerScheduleRoutes(r chi.Router, sqlxDB *sqlx.DB, tc temporalclient.Client,
	reportService *reports.ReportService, reportExecutor reports.ReportExecutor) {
	log := logging.GetLogger().Sugar()

	var calDB *sqlx.DB
	if dsn := calendarDSN(); dsn != "" {
		db, err := sqlx.Open("postgres", dsn)
		if err != nil {
			log.Warnf("scheduler: calendar database unavailable, calendar rules will fail: %v", err)
		} else {
			db.SetMaxOpenConns(5)
			calDB = db
		}
	}
	var gold string
	if sqlxDB == nil || sqlxDB.DB == nil {
		// Router built without a database (route-table tests, tools).
		log.Warnf("scheduler: no metadata database; schedules are unavailable")
	} else if err := sqlxDB.Get(&gold, `SELECT public.uisce_gold_copy_tenant_id()::text`); err != nil {
		log.Warnf("scheduler: gold-copy tenant not resolved; core calendars unavailable: %v", err)
	}

	runners := schedule.NewRegistry(newReportRunner(reportService, reportExecutor, sqlxDB))
	if s.SavedQueryHandler != nil {
		runners.Register(&savedQueryRunner{h: s.SavedQueryHandler})
	}
	if s.dataPipelineRunner != nil {
		runners.Register(s.dataPipelineRunner)
	}
	s.ScheduleRunners = runners

	store := &schedule.Store{DB: sqlxDB}
	cals := &schedule.MDMCalendars{DB: calDB, GoldTenantID: gold}
	svc := &schedule.Service{Store: store, Engine: &schedule.TemporalEngine{Client: tc}, Calendars: cals, Runners: runners}
	s.ScheduleService = svc

	if tc != nil {
		acts := &schedule.Activities{Store: store, Calendars: cals, Runners: runners}
		w := worker.New(tc, schedule.TaskQueue, worker.Options{})
		w.RegisterWorkflow(schedule.FireWorkflow)
		w.RegisterActivityWithOptions(acts.Gate, activity.RegisterOptions{Name: schedule.ActGate})
		w.RegisterActivityWithOptions(acts.RecordSkip, activity.RegisterOptions{Name: schedule.ActRecord})
		w.RegisterActivityWithOptions(acts.Run, activity.RegisterOptions{Name: schedule.ActRun})
		if err := w.Start(); err != nil {
			log.Errorf("scheduler: worker did not start, schedules will not fire: %v", err)
		} else {
			log.Infof("scheduler: worker started on task queue %s", schedule.TaskQueue)
		}
	} else {
		log.Warnf("scheduler: no Temporal client; schedules can be edited but will not fire")
	}

	(&schedule.Handler{Service: svc, Catalog: s.MessageCatalog, ActorFrom: s.scheduleActor}).RegisterRoutes(r)
}

// scheduleActor is the caller from the authenticated request: the tenant
// from the token (never a header on its own), and the datasource and region
// the request is working in, which scheduled runs reuse.
func (s *Server) scheduleActor(r *http.Request) (schedule.Actor, error) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", s.SecurityContextDeps)
	if err != nil || secCtx == nil || secCtx.TenantID == "" || secCtx.UserID == "" {
		return schedule.Actor{}, msgcat.Unauthenticated().Wrap(err)
	}
	a := schedule.Actor{UserID: secCtx.UserID, TenantID: secCtx.TenantID,
		DatasourceID: secCtx.ScopedDatasourceID(), Region: secCtx.Region, CanTrigger: true}
	// An enterprise scheduler's service account (Keycloak client-credentials,
	// realm role uisce_service_account) reads and triggers only, and
	// triggers only with schedule_trigger.
	if auth, ok := security.AuthInfoFromContext(r.Context()); ok {
		roles := map[string]bool{}
		for _, role := range auth.Roles {
			roles[role] = true
		}
		if roles[RoleServiceAccount] {
			a.Machine, a.CanTrigger = true, roles[RoleScheduleTrigger]
		}
	}
	return a, nil
}

// Realm roles for enterprise scheduler service accounts
// (infrastructure/keycloak/create-scheduler-client.sh).
const (
	RoleServiceAccount  = "uisce_service_account"
	RoleScheduleTrigger = "schedule_trigger"
)
