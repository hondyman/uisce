// Command migrate-report-schedules copies active public.report_schedules into
// the one scheduler (public.schedules, target_kind=report), optionally
// registers each with Temporal, then deactivates the legacy row.
//
// Idempotent: skips when a core schedule already exists for the same
// (tenant_id, report_definition_id, schedule_name), or when the legacy row is
// already inactive / soft-deleted.
//
//	DATABASE_URL / POSTGRES_DSN   metadata DB (alpha)
//	TEMPORAL_*                   optional; with --apply-engine (default on)
//
//	go run ./cmd/migrate-report-schedules --dry-run
//	go run ./cmd/migrate-report-schedules
//	go run ./cmd/migrate-report-schedules --reapply-core   # Temporal Apply for existing core rows
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/hondyman/uisce/backend/internal/schedule"
	temporalclient "github.com/hondyman/uisce/libs/temporal-client"
)

type legacyRow struct {
	ID                  string         `db:"id"`
	TenantID            string         `db:"tenant_id"`
	ReportDefinitionID  string         `db:"report_definition_id"`
	OwnerID             string         `db:"owner_id"`
	ScheduleName        string         `db:"schedule_name"`
	CronExpression      string         `db:"cron_expression"`
	Region              string         `db:"region"`
	CalendarCode        sql.NullString `db:"calendar_code"`
	UnscheduledBehavior sql.NullString `db:"unscheduled_behavior"`
	BusinessDayOffset   int            `db:"business_day_offset"`
	BurstDimension      string         `db:"burst_dimension"`
	ExportFormat        string         `db:"export_format"`
	NotificationJSON    []byte         `db:"notification_channels"`
	IsActive            bool           `db:"is_active"`
}

func main() {
	dry := flag.Bool("dry-run", false, "print the plan; do not write")
	applyEngine := flag.Bool("apply-engine", true, "register timetable schedules with Temporal after insert")
	reapplyCore := flag.Bool("reapply-core", false, "Apply Temporal for existing enabled core schedules (no legacy copy)")
	flag.Parse()

	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("POSTGRES_DSN"))
	}
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL or POSTGRES_DSN is required")
		os.Exit(2)
	}

	ctx := context.Background()
	db, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect:", err)
		os.Exit(1)
	}
	defer db.Close()

	needEngine := *reapplyCore || (!*dry && *applyEngine)
	var engine schedule.Engine
	if needEngine && !*dry {
		tc, err := temporalclient.NewClientWithRetry()
		if err != nil {
			fmt.Fprintln(os.Stderr, "temporal:", err)
			fmt.Fprintln(os.Stderr, "hint: pass --apply-engine=false for DB-only, or set TEMPORAL_HOST")
			os.Exit(1)
		}
		defer tc.Close()
		engine = &schedule.TemporalEngine{Client: tc}
	}

	store := &schedule.Store{DB: db}
	if *reapplyCore {
		os.Exit(reapply(ctx, db, store, engine, *dry))
	}

	rows, err := listLegacy(ctx, db)
	if err != nil {
		fmt.Fprintln(os.Stderr, "list:", err)
		os.Exit(1)
	}
	fmt.Printf("legacy active report_schedules: %d\n", len(rows))

	var migrated, skipped, failed int
	for _, row := range rows {
		exists, err := coreExists(ctx, db, row)
		if err != nil {
			fmt.Fprintf(os.Stderr, "check %s: %v\n", row.ID, err)
			failed++
			continue
		}
		if exists {
			fmt.Printf("skip  %s %q (already on core)\n", row.ID, row.ScheduleName)
			if !*dry {
				if err := deactivateLegacy(ctx, db, row); err != nil {
					fmt.Fprintf(os.Stderr, "deactivate %s: %v\n", row.ID, err)
					failed++
					continue
				}
			}
			skipped++
			continue
		}
		in := toInput(row)
		fmt.Printf("%s %s %q → report/%s cron=%q rule=%s calendar=%q\n",
			map[bool]string{true: "plan", false: "migrate"}[*dry],
			row.ID, row.ScheduleName, row.ReportDefinitionID, in.Timing.Cron, in.Timing.CalendarRule, in.Timing.Calendar)
		if *dry {
			migrated++
			continue
		}
		sc, err := store.Create(ctx, &schedule.Schedule{
			TenantID:     row.TenantID,
			Name:         in.Name,
			Description:  in.Description,
			Target:       in.Target,
			Timing:       in.Timing,
			Enabled:      true,
			OwnerID:      row.OwnerID,
			Region:       row.Region,
		}, row.OwnerID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create %s: %v\n", row.ID, err)
			failed++
			continue
		}
		if engine != nil {
			if err := engine.Apply(ctx, sc); err != nil {
				fmt.Fprintf(os.Stderr, "apply %s (core %s): %v — rolling back core row\n", row.ID, sc.ID, err)
				_, _ = store.Delete(ctx, row.TenantID, sc.ID, "system:migrate-rollback")
				failed++
				continue
			}
		}
		if err := deactivateLegacy(ctx, db, row); err != nil {
			fmt.Fprintf(os.Stderr, "deactivate legacy %s (core %s ok): %v\n", row.ID, sc.ID, err)
			failed++
			continue
		}
		fmt.Printf("ok    legacy %s → core %s\n", row.ID, sc.ID)
		migrated++
	}

	fmt.Printf("done migrated=%d skipped=%d failed=%d dry_run=%v apply_engine=%v\n",
		migrated, skipped, failed, *dry, *applyEngine && !*dry)
	if failed > 0 {
		os.Exit(1)
	}
}

func reapply(ctx context.Context, db *sqlx.DB, store *schedule.Store, engine schedule.Engine, dry bool) int {
	type ref struct {
		ID       string `db:"id"`
		TenantID string `db:"tenant_id"`
		Name     string `db:"name"`
		Kind     string `db:"target_kind"`
		Mode     string `db:"trigger_mode"`
		Enabled  bool   `db:"enabled"`
	}
	var refs []ref
	// Admin connection: bypass RLS for the inventory query only.
	if err := db.SelectContext(ctx, &refs, `
		SELECT id::text, tenant_id::text, name, target_kind, trigger_mode, enabled
		FROM public.schedules
		WHERE deleted_at IS NULL AND enabled = true AND trigger_mode = 'timetable'
		ORDER BY created_at`); err != nil {
		fmt.Fprintln(os.Stderr, "list core:", err)
		return 1
	}
	fmt.Printf("core enabled timetable schedules: %d\n", len(refs))
	var okN, failN int
	for _, r := range refs {
		fmt.Printf("%s %s %q kind=%s\n", map[bool]string{true: "plan", false: "apply"}[dry], r.ID, r.Name, r.Kind)
		if dry {
			okN++
			continue
		}
		if engine == nil {
			fmt.Fprintln(os.Stderr, "engine required for --reapply-core")
			return 1
		}
		sc, err := store.Get(ctx, r.TenantID, r.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "get %s: %v\n", r.ID, err)
			failN++
			continue
		}
		if err := engine.Apply(ctx, sc); err != nil {
			fmt.Fprintf(os.Stderr, "apply %s: %v\n", r.ID, err)
			failN++
			continue
		}
		fmt.Printf("ok    %s Temporal schedule-%s-%s\n", r.ID, r.TenantID, r.ID)
		okN++
	}
	fmt.Printf("done reapply ok=%d failed=%d dry_run=%v\n", okN, failN, dry)
	if failN > 0 {
		return 1
	}
	return 0
}

func listLegacy(ctx context.Context, db *sqlx.DB) ([]legacyRow, error) {
	var rows []legacyRow
	err := db.SelectContext(ctx, &rows, `
		SELECT rs.id::text, rs.tenant_id::text, rs.report_definition_id::text, rs.owner_id,
			rs.schedule_name, rs.cron_expression, rs.region,
			c.calendar_code, rs.unscheduled_behavior, COALESCE(rs.business_day_offset, 0) AS business_day_offset,
			rs.burst_dimension, rs.export_format, COALESCE(rs.notification_channels, '{}'::jsonb) AS notification_channels,
			rs.is_active
		FROM public.report_schedules rs
		LEFT JOIN public.tenant_exchange_calendars c ON c.id = rs.calendar_id
		WHERE rs.deleted_at IS NULL AND rs.is_active = true
		ORDER BY rs.created_at`)
	return rows, err
}

func coreExists(ctx context.Context, db *sqlx.DB, row legacyRow) (bool, error) {
	var n int
	err := db.GetContext(ctx, &n, `
		SELECT COUNT(*) FROM public.schedules
		WHERE tenant_id::text = $1 AND target_kind = 'report' AND target_ref = $2
		  AND name = $3 AND deleted_at IS NULL`,
		row.TenantID, row.ReportDefinitionID, row.ScheduleName)
	return n > 0, err
}

func deactivateLegacy(ctx context.Context, db *sqlx.DB, row legacyRow) error {
	_, err := db.ExecContext(ctx, `
		UPDATE public.report_schedules
		SET is_active = false
		WHERE id::text = $1 AND tenant_id::text = $2 AND deleted_at IS NULL`,
		row.ID, row.TenantID)
	return err
}

func toInput(row legacyRow) schedule.Input {
	params := map[string]any{
		"burst_dimension": row.BurstDimension,
		"export_format":   row.ExportFormat,
		"legacy_report_schedule_id": row.ID,
	}
	var notify map[string]any
	if len(row.NotificationJSON) > 0 {
		_ = json.Unmarshal(row.NotificationJSON, &notify)
	}
	if notify != nil {
		if v, ok := notify["in_app"]; ok {
			params["notify_in_app"] = v
		}
		if v, ok := notify["email"]; ok {
			params["notify_email"] = v
		}
	}

	cal := normalizeCalendar(row.CalendarCode.String)
	rule, bd := mapBehavior(row.UnscheduledBehavior.String, row.BusinessDayOffset, cal != "")

	desc := "Migrated from report_schedules " + row.ID
	return schedule.Input{
		Name:        row.ScheduleName,
		Description: desc,
		Target: schedule.Target{
			Kind:   "report",
			Ref:    row.ReportDefinitionID,
			Params: params,
		},
		Timing: schedule.Timing{
			Mode:         schedule.ModeTimetable,
			Cron:         strings.TrimSpace(row.CronExpression),
			TimeZone:     zoneForRegion(row.Region),
			Calendar:     cal,
			CalendarRule: rule,
			BusinessDay:  bd,
		},
	}
}

func mapBehavior(behavior string, offset int, hasCalendar bool) (rule string, businessDay int) {
	b := strings.ToUpper(strings.TrimSpace(behavior))
	switch b {
	case "SKIP":
		if hasCalendar {
			return schedule.RuleSkip, 0
		}
		return schedule.RuleNone, 0
	case "RUN_NEXT_BUS_DAY", "NEXT_BUSINESS_DAY":
		if hasCalendar {
			return schedule.RuleNextBusinessDay, 0
		}
		return schedule.RuleNone, 0
	case "BUSINESS_DAY_OF_MONTH":
		if hasCalendar && offset != 0 {
			return schedule.RuleBusinessDayOfMonth, offset
		}
		return schedule.RuleNone, 0
	default:
		// RUN_PREVIOUS_BUS_DAY and unknowns: core has no previous-day rule.
		if hasCalendar {
			return schedule.RuleSkip, 0
		}
		return schedule.RuleNone, 0
	}
}

func normalizeCalendar(code string) string {
	c := strings.ToUpper(strings.TrimSpace(code))
	switch c {
	case "":
		return ""
	case "NYSE":
		return "XNYS"
	case "LSE":
		return "XLON"
	default:
		return c
	}
}

func zoneForRegion(region string) string {
	switch strings.ToLower(strings.TrimSpace(region)) {
	case "us-west", "us-west-2":
		return "America/Los_Angeles"
	case "us-east", "us-east-1":
		return "America/New_York"
	case "eu", "eu-west", "uk", "london":
		return "Europe/London"
	default:
		return "UTC"
	}
}

