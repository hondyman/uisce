package querybuilder

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

// Reconcile action kinds (CUBE-1.4 receipt).
const (
	ReconcileActionActiveMissingPhysical = "active_missing_physical"
	ReconcileActionStaleAttemptCleanup   = "stale_attempt_cleanup"
	ReconcileActionFailedLeftoverCleanup = "failed_leftover_cleanup"
	ReconcileActionOrphanQuarantined     = "orphan_quarantined"
)

// CubePhysicalProbe inspects and mutates StarRocks/Iceberg physicals for reconcile.
// Tests inject a fake; production uses starRocksPhysicalProbe.
type CubePhysicalProbe interface {
	HotExists(ctx context.Context, database, name string) (bool, error)
	ColdExists(ctx context.Context, icebergTable string) (bool, error)
	DropHot(ctx context.Context, database, name string) error
	DropCold(ctx context.Context, icebergTable string) error
	QuarantineHot(ctx context.Context, database, name string) (quarantineName string, err error)
	ListHotNames(ctx context.Context, database string) ([]string, error)
}

// CubeReconcileAction is one corrective step recorded on the receipt.
type CubeReconcileAction struct {
	Kind     string `json:"kind"`
	NodeID   string `json:"node_id,omitempty"`
	NodeName string `json:"node_name,omitempty"`
	TenantID string `json:"tenant_id,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

// CubeReconcileReceipt is the reconcile pass result (proof artifact for CUBE-1.4).
type CubeReconcileReceipt struct {
	StartedAt  time.Time             `json:"started_at"`
	FinishedAt time.Time             `json:"finished_at"`
	Examined   int                   `json:"examined"`
	Actions    []CubeReconcileAction `json:"actions"`
	Errors     []string              `json:"errors,omitempty"`
}

// CubeReconciler diffs catalog Active/Materializing/Failed cube grains against
// physical hot/cold existence and applies attempt-scoped cleanup + orphan quarantine.
type CubeReconciler struct {
	db             *sqlx.DB
	lifecycle      *analytics.PreAggLifecycleService
	probe          CubePhysicalProbe
	staleAttemptBy time.Duration
	now            func() time.Time
}

// NewCubeReconciler builds a reconciler. starrocksDB may be nil when probe is injected.
func NewCubeReconciler(db *sqlx.DB, starrocksDB *sql.DB) *CubeReconciler {
	var lifecycle *analytics.PreAggLifecycleService
	if db != nil {
		lifecycle = analytics.NewPreAggLifecycleService(db)
	}
	var probe CubePhysicalProbe
	if starrocksDB != nil {
		probe = &starRocksPhysicalProbe{db: starrocksDB}
	}
	return &CubeReconciler{
		db:             db,
		lifecycle:      lifecycle,
		probe:          probe,
		staleAttemptBy: defaultStaleAttemptTTL(),
		now:            func() time.Time { return time.Now().UTC() },
	}
}

// SetProbe replaces the physical probe (unit tests).
func (r *CubeReconciler) SetProbe(p CubePhysicalProbe) {
	if r != nil {
		r.probe = p
	}
}

// SetStaleAttemptTTL overrides how long Materializing may linger before cleanup.
func (r *CubeReconciler) SetStaleAttemptTTL(d time.Duration) {
	if r != nil && d > 0 {
		r.staleAttemptBy = d
	}
}

func defaultStaleAttemptTTL() time.Duration {
	if v := strings.TrimSpace(os.Getenv("CUBE_STALE_ATTEMPT_TTL_MINUTES")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Minute
		}
	}
	return 2 * time.Hour
}

type cubeGrainCatalogRow struct {
	ID         uuid.UUID       `db:"id"`
	NodeName   string          `db:"node_name"`
	Properties json.RawMessage `db:"properties"`
	Config     json.RawMessage `db:"config"`
}

// Reconcile runs one full pass and returns a receipt.
func (r *CubeReconciler) Reconcile(ctx context.Context) (*CubeReconcileReceipt, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("cube reconciler: database not configured")
	}
	if r.probe == nil {
		return nil, fmt.Errorf("cube reconciler: physical probe not configured")
	}
	if r.lifecycle == nil {
		r.lifecycle = analytics.NewPreAggLifecycleService(r.db)
	}
	nowFn := r.now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}
	receipt := &CubeReconcileReceipt{
		StartedAt: nowFn(),
		Actions:   make([]CubeReconcileAction, 0),
	}

	rows, err := r.listCubeGrainNodes(ctx)
	if err != nil {
		return nil, err
	}
	receipt.Examined = len(rows)

	knownHot := map[string]struct{}{} // key: database\x1fname
	for _, row := range rows {
		props, pErr := models.ParsePreAggProperties(row.Properties)
		if pErr != nil {
			receipt.Errors = append(receipt.Errors, fmt.Sprintf("node %s: parse properties: %v", row.ID, pErr))
			continue
		}
		cfg, cErr := models.ParsePreAggConfig(row.Config)
		if cErr != nil {
			receipt.Errors = append(receipt.Errors, fmt.Sprintf("node %s: parse config: %v", row.ID, cErr))
			continue
		}
		targetDB := strings.TrimSpace(props.TargetDatabase)
		matName := strings.TrimSpace(cfg.Materialization.TargetName)
		if matName == "" {
			matName = row.NodeName
		}
		if targetDB != "" && matName != "" {
			knownHot[targetDB+"\x1f"+matName] = struct{}{}
		}

		switch props.LifecycleStatus {
		case models.LifecycleActive:
			r.reconcileActive(ctx, receipt, row, props, targetDB, matName)
		case models.LifecycleMaterializing:
			r.reconcileStaleAttempt(ctx, receipt, row, props, targetDB, matName, nowFn())
		case models.LifecycleFailed:
			r.reconcileFailedLeftover(ctx, receipt, row, props, targetDB, matName)
		}
	}

	if qErr := r.quarantineOrphans(ctx, receipt, knownHot); qErr != nil {
		receipt.Errors = append(receipt.Errors, qErr.Error())
	}

	receipt.FinishedAt = nowFn()
	return receipt, nil
}

func (r *CubeReconciler) listCubeGrainNodes(ctx context.Context) ([]cubeGrainCatalogRow, error) {
	var rows []cubeGrainCatalogRow
	err := r.db.SelectContext(ctx, &rows, `
		SELECT cn.id, cn.node_name, cn.properties, cn.config
		FROM catalog_node cn
		JOIN catalog_node_type cnt ON cnt.id = cn.node_type_id
		WHERE cnt.catalog_type_name = 'pre_aggregation'
		  AND COALESCE(cn.properties->>'cube_id', '') <> ''
	`)
	if err != nil {
		return nil, fmt.Errorf("list cube materialization nodes: %w", err)
	}
	return rows, nil
}

func (r *CubeReconciler) reconcileActive(
	ctx context.Context,
	receipt *CubeReconcileReceipt,
	row cubeGrainCatalogRow,
	props *models.PreAggProperties,
	targetDB, matName string,
) {
	missing := make([]string, 0, 2)
	if targetDB != "" && matName != "" {
		ok, err := r.probe.HotExists(ctx, targetDB, matName)
		if err != nil {
			receipt.Errors = append(receipt.Errors, fmt.Sprintf("hot probe %s.%s: %v", targetDB, matName, err))
			return
		}
		if !ok {
			missing = append(missing, "hot")
		}
	}
	if props.DualCommitWatermark != nil || strings.TrimSpace(props.IcebergTable) != "" {
		ice := strings.TrimSpace(props.IcebergTable)
		if ice == "" {
			missing = append(missing, "cold")
		} else {
			ok, err := r.probe.ColdExists(ctx, ice)
			if err != nil {
				receipt.Errors = append(receipt.Errors, fmt.Sprintf("cold probe %s: %v", ice, err))
				return
			}
			if !ok {
				missing = append(missing, "cold")
			}
		}
	} else {
		// Active without dual-commit watermark is not dual-committed.
		missing = append(missing, "dual_commit_watermark")
	}
	if len(missing) == 0 {
		return
	}
	detail := fmt.Sprintf("missing=%s attempt_id=%s", strings.Join(missing, ","), props.AttemptID)
	cause := fmt.Errorf("cube reconcile: active grain missing physical (%s)", strings.Join(missing, ","))
	if err := r.lifecycle.MarkFailedAttempt(ctx, row.ID, props.AttemptID, cause); err != nil {
		receipt.Errors = append(receipt.Errors, fmt.Sprintf("mark failed %s: %v", row.ID, err))
		return
	}
	receipt.Actions = append(receipt.Actions, CubeReconcileAction{
		Kind:     ReconcileActionActiveMissingPhysical,
		NodeID:   row.ID.String(),
		NodeName: row.NodeName,
		TenantID: props.TenantID,
		Detail:   detail,
	})
}

func (r *CubeReconciler) reconcileStaleAttempt(
	ctx context.Context,
	receipt *CubeReconcileReceipt,
	row cubeGrainCatalogRow,
	props *models.PreAggProperties,
	targetDB, matName string,
	now time.Time,
) {
	started := props.LastMaterializedAt
	if started == nil {
		return
	}
	if now.Sub(started.UTC()) < r.staleAttemptBy {
		return
	}
	detailParts := []string{fmt.Sprintf("stale_for=%s", now.Sub(started.UTC()).Truncate(time.Second))}
	if targetDB != "" && matName != "" {
		if err := r.probe.DropHot(ctx, targetDB, matName); err != nil {
			receipt.Errors = append(receipt.Errors, fmt.Sprintf("drop hot %s.%s: %v", targetDB, matName, err))
		} else {
			detailParts = append(detailParts, "dropped_hot")
		}
	}
	if ice := strings.TrimSpace(props.IcebergTable); ice != "" {
		if err := r.probe.DropCold(ctx, ice); err != nil {
			receipt.Errors = append(receipt.Errors, fmt.Sprintf("drop cold %s: %v", ice, err))
		} else {
			detailParts = append(detailParts, "dropped_cold")
		}
	}
	cause := fmt.Errorf("cube reconcile: abandoned materializing attempt %s older than %s", props.AttemptID, r.staleAttemptBy)
	if err := r.lifecycle.MarkFailedAttempt(ctx, row.ID, props.AttemptID, cause); err != nil {
		receipt.Errors = append(receipt.Errors, fmt.Sprintf("mark failed stale %s: %v", row.ID, err))
		return
	}
	receipt.Actions = append(receipt.Actions, CubeReconcileAction{
		Kind:     ReconcileActionStaleAttemptCleanup,
		NodeID:   row.ID.String(),
		NodeName: row.NodeName,
		TenantID: props.TenantID,
		Detail:   strings.Join(detailParts, " "),
	})
}

func (r *CubeReconciler) reconcileFailedLeftover(
	ctx context.Context,
	receipt *CubeReconcileReceipt,
	row cubeGrainCatalogRow,
	props *models.PreAggProperties,
	targetDB, matName string,
) {
	// Mid-saga compensate may have failed: Failed lifecycle with hot still present.
	if targetDB == "" || matName == "" {
		return
	}
	ok, err := r.probe.HotExists(ctx, targetDB, matName)
	if err != nil {
		receipt.Errors = append(receipt.Errors, fmt.Sprintf("hot probe leftover %s.%s: %v", targetDB, matName, err))
		return
	}
	if !ok {
		return
	}
	qName, qErr := r.probe.QuarantineHot(ctx, targetDB, matName)
	if qErr != nil {
		receipt.Errors = append(receipt.Errors, fmt.Sprintf("quarantine leftover %s.%s: %v", targetDB, matName, qErr))
		return
	}
	if ice := strings.TrimSpace(props.IcebergTable); ice != "" {
		_ = r.probe.DropCold(ctx, ice)
	}
	receipt.Actions = append(receipt.Actions, CubeReconcileAction{
		Kind:     ReconcileActionFailedLeftoverCleanup,
		NodeID:   row.ID.String(),
		NodeName: row.NodeName,
		TenantID: props.TenantID,
		Detail:   fmt.Sprintf("quarantined_hot=%s", qName),
	})
}

func (r *CubeReconciler) quarantineOrphans(
	ctx context.Context,
	receipt *CubeReconcileReceipt,
	knownHot map[string]struct{},
) error {
	dbs := make(map[string]struct{})
	for k := range knownHot {
		parts := strings.SplitN(k, "\x1f", 2)
		if len(parts) == 2 && parts[0] != "" {
			dbs[parts[0]] = struct{}{}
		}
	}
	// Always scan gold + common tenant prefix databases discovered from catalog.
	for dbName := range dbs {
		names, err := r.probe.ListHotNames(ctx, dbName)
		if err != nil {
			receipt.Errors = append(receipt.Errors, fmt.Sprintf("list hot %s: %v", dbName, err))
			continue
		}
		for _, name := range names {
			if !strings.HasPrefix(name, "cube_") {
				continue
			}
			if strings.HasPrefix(name, "cube_quarantine_") {
				continue
			}
			key := dbName + "\x1f" + name
			if _, ok := knownHot[key]; ok {
				continue
			}
			qName, qErr := r.probe.QuarantineHot(ctx, dbName, name)
			if qErr != nil {
				receipt.Errors = append(receipt.Errors, fmt.Sprintf("orphan quarantine %s.%s: %v", dbName, name, qErr))
				continue
			}
			receipt.Actions = append(receipt.Actions, CubeReconcileAction{
				Kind:     ReconcileActionOrphanQuarantined,
				NodeName: name,
				Detail:   fmt.Sprintf("database=%s quarantined_as=%s", dbName, qName),
			})
		}
	}
	return nil
}

// starRocksPhysicalProbe implements CubePhysicalProbe against a StarRocks FE.
type starRocksPhysicalProbe struct {
	db *sql.DB
}

func (p *starRocksPhysicalProbe) HotExists(ctx context.Context, database, name string) (bool, error) {
	if p == nil || p.db == nil {
		return false, fmt.Errorf("starrocks probe: not configured")
	}
	var n int
	q := fmt.Sprintf(
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = %s AND table_name = %s`,
		quoteStarRocksString(database), quoteStarRocksString(name),
	)
	if err := p.db.QueryRowContext(ctx, q).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (p *starRocksPhysicalProbe) ColdExists(ctx context.Context, icebergTable string) (bool, error) {
	if p == nil || p.db == nil {
		return false, fmt.Errorf("starrocks probe: not configured")
	}
	parts := strings.Split(strings.TrimSpace(icebergTable), ".")
	if len(parts) != 3 {
		return false, fmt.Errorf("cold exists: expected catalog.db.table, got %q", icebergTable)
	}
	cat, schema, table := sanitizeIdentifier(parts[0]), sanitizeIdentifier(parts[1]), sanitizeIdentifier(parts[2])
	qualified := fmt.Sprintf("%s.%s.%s", quoteStarRocksIdent(cat), quoteStarRocksIdent(schema), quoteStarRocksIdent(table))
	// DESCRIBE fails when missing; COUNT via information_schema is catalog-specific.
	_, err := p.db.ExecContext(ctx, fmt.Sprintf("DESCRIBE %s", qualified))
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "does not exist") || strings.Contains(msg, "unknown table") || strings.Contains(msg, "not found") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (p *starRocksPhysicalProbe) DropHot(ctx context.Context, database, name string) error {
	qualified := fmt.Sprintf("%s.%s", quoteStarRocksIdent(database), quoteStarRocksIdent(name))
	_, err := p.db.ExecContext(ctx, fmt.Sprintf("DROP MATERIALIZED VIEW IF EXISTS %s", qualified))
	if err != nil {
		// Some grains may be tables rather than MVs.
		_, err2 := p.db.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", qualified))
		if err2 != nil {
			return fmt.Errorf("drop hot %s: mv=%v table=%v", qualified, err, err2)
		}
	}
	return nil
}

func (p *starRocksPhysicalProbe) DropCold(ctx context.Context, icebergTable string) error {
	parts := strings.Split(strings.TrimSpace(icebergTable), ".")
	if len(parts) != 3 {
		return fmt.Errorf("drop cold: expected catalog.db.table, got %q", icebergTable)
	}
	qualified := fmt.Sprintf("%s.%s.%s",
		quoteStarRocksIdent(sanitizeIdentifier(parts[0])),
		quoteStarRocksIdent(sanitizeIdentifier(parts[1])),
		quoteStarRocksIdent(sanitizeIdentifier(parts[2])),
	)
	_, err := p.db.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", qualified))
	return err
}

func (p *starRocksPhysicalProbe) QuarantineHot(ctx context.Context, database, name string) (string, error) {
	qName := "cube_quarantine_" + sanitizeIdentifier(name)
	if len(qName) > 120 {
		qName = qName[:120]
	}
	src := fmt.Sprintf("%s.%s", quoteStarRocksIdent(database), quoteStarRocksIdent(name))
	dst := fmt.Sprintf("%s.%s", quoteStarRocksIdent(database), quoteStarRocksIdent(qName))
	// Best-effort: drop prior quarantine target, then rename.
	_, _ = p.db.ExecContext(ctx, fmt.Sprintf("DROP MATERIALIZED VIEW IF EXISTS %s", dst))
	_, _ = p.db.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", dst))
	if _, err := p.db.ExecContext(ctx, fmt.Sprintf("ALTER MATERIALIZED VIEW %s RENAME %s", src, quoteStarRocksIdent(qName))); err != nil {
		if _, err2 := p.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s RENAME %s", src, quoteStarRocksIdent(qName))); err2 != nil {
			// Last resort: drop (still removes the orphan from the live namespace).
			if dropErr := p.DropHot(ctx, database, name); dropErr != nil {
				return "", fmt.Errorf("quarantine %s: rename mv=%v table=%v drop=%v", src, err, err2, dropErr)
			}
			return "dropped:" + name, nil
		}
	}
	return qName, nil
}

func (p *starRocksPhysicalProbe) ListHotNames(ctx context.Context, database string) ([]string, error) {
	q := fmt.Sprintf(
		`SELECT table_name FROM information_schema.tables WHERE table_schema = %s AND table_name LIKE 'cube_%%'`,
		quoteStarRocksString(database),
	)
	rows, err := p.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}
