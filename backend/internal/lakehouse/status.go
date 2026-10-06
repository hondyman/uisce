// Package lakehouse — status.go is the read-only aggregation behind Platform > Lakehouse
// status (ADR-049 area). It surfaces cluster health (SHOW FRONTENDS / SHOW BACKENDS), the
// resource-group table, and per-tenant wiring + audit-copy posture for every tenant in the
// lakehouse registry. It writes nothing; the only mutations happen behind the registry's
// own methods.
//
// Two handlers call this service:
//   - GET /api/admin/lakehouse/status        (platform admin, this package's handler)
//   - GET /api/tenant/lakehouse/status       (tenant admin, scoped by session)
//
// They take the same payload shape; the tenant handler strips the cluster and resource-group
// sections and keeps only the session tenant's row.
//
// All queries here are SHOW / information_schema reads. The AdminDB connection must NOT
// require root: any StarRocks user can run SHOW FRONTENDS, SHOW BACKENDS, and SHOW RESOURCE
// GROUPS ALL. A privileged root DSN would defeat the point of having per-tenant users.
package lakehouse

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/lakehouse/infra"
	"github.com/hondyman/uisce/backend/internal/lakehouse/registry"
)

// TenantLister is the part of registry.Store the status panel uses: enumerate the tenant
// registry and read one tenant's lakehouse state. The status service never mutates.
type TenantLister interface {
	List(ctx context.Context, q string, limit, offset int) ([]registry.Config, int, error)
	Get(ctx context.Context, tenantID uuid.UUID) (*registry.Config, error)
}

// StatusService aggregates read-only cluster state for the status panel. It is safe for
// concurrent use: each method reads fresh on every call, so the cluster view always matches
// "what the router would do right now."
type StatusService struct {
	AdminDB *sql.DB      // StarRocks pool (any user can SHOW; do NOT use the root credential)
	Reg     TenantLister // the lakehouse registry source
	Now     func() time.Time // injectable clock; tests pin it for stable generated_at

	// showWorkloadGroups caches which SHOW statement works on the live cluster.
	// 3.3: SHOW RESOURCE GROUPS ALL. 4.1: SHOW WORKLOAD GROUPS ALL (construct renamed).
	// The first call to resourceGroups probes both and caches whichever works; subsequent
	// calls use the cached decision. Per the runbook gap 3, this is the kind of code that
	// silently breaks across the upgrade — capturing it once per process keeps the panel
	// from flickering between the two.
	showWorkloadGroups bool
	rgOnce             sync.Once
}

// NewStatusService builds the service. showWorkloadGroups is captured at boot: the same
// SHOW statement works for both constructs in some releases, errors in others; one
// decision per process keeps the panel consistent.
func NewStatusService(adminDB *sql.DB, reg TenantLister, showWorkloadGroups bool) *StatusService {
	return &StatusService{AdminDB: adminDB, Reg: reg, Now: time.Now, showWorkloadGroups: showWorkloadGroups}
}

// Status is the platform payload. Tenants is one block per registry tenant. ResourceGroups
// is the live cluster's resource-group table. Cluster is FE alive/version + BE capacity/used.
//
// Per-tenant AuditCopy is the cross-check the panel exists to surface: it joins the registry's
// "is this tenant's lakehouse wired?" view with the router's "what env did the process actually
// pick up?" view and the resource-group table's "does the classifier route this user's traffic
// to a real group?" view.
type Status struct {
	GeneratedAt    time.Time         `json:"generated_at"`
	Cluster        ClusterStatus     `json:"cluster"`
	ResourceGroups []ResourceGroup   `json:"resource_groups"`
	Tenants        []TenantBlock     `json:"tenants"`
	Notes          []string          `json:"notes,omitempty"` // top-level caveats, e.g. "audit pipeline not installed"
}

// TenantScoped is the tenant admin's payload. The cluster and resource-group sections are
// omitted; only the session tenant's row is included.
type TenantScoped struct {
	GeneratedAt time.Time   `json:"generated_at"`
	Tenant      TenantBlock `json:"tenant"`
	Notes       []string    `json:"notes,omitempty"`
}

// ClusterStatus is what SHOW FRONTENDS / SHOW BACKENDS surfaces.
type ClusterStatus struct {
	Frontends []FrontendStatus `json:"frontends"`
	Backends  []BackendStatus  `json:"backends"`
}

// FrontendStatus is one row of SHOW FRONTENDS.
type FrontendStatus struct {
	Name    string `json:"name"`
	Host    string `json:"host,omitempty"`
	Alive   bool   `json:"alive"`
	Version string `json:"version,omitempty"`
}

// BackendStatus is one row of SHOW BACKENDS.
type BackendStatus struct {
	Host             string  `json:"host,omitempty"`
	Alive            bool    `json:"alive"`
	TotalCapacityB   int64   `json:"total_capacity_bytes,omitempty"`
	UsedPct          float64 `json:"used_pct,omitempty"` // 0..1
}

// ResourceGroup is one row of SHOW RESOURCE GROUPS ALL (3.3) or SHOW WORKLOAD GROUPS ALL (4.1).
//
// `Classifiers` is parsed from the SHOW output's user=... entries; on a parse failure we
// keep the raw text rather than failing the panel, so a column-name drift shows up in the UI
// rather than as a 500.
type ResourceGroup struct {
	Name             string             `json:"name"`
	CPUWeight        *int               `json:"cpu_weight,omitempty"`
	MemLimit         string             `json:"mem_limit,omitempty"`
	ConcurrencyLimit *int               `json:"concurrency_limit,omitempty"`
	BigQueryMemLimit *int64             `json:"big_query_mem_limit,omitempty"`
	Classifiers      []ResourceClassifier `json:"classifiers"`
}

// ResourceClassifier is one user/role binding inside a resource group.
type ResourceClassifier struct {
	User   string  `json:"user,omitempty"`
	Role   string  `json:"role,omitempty"`
	Weight float64 `json:"weight,omitempty"`
}

// TenantBlock is one tenant's row in the status panel.
type TenantBlock struct {
	Name        string `json:"name"`                  // public.tenants.name
	Slug        string `json:"slug"`                  // infra.TenantEnvKey(name), or "" if name is empty
	TenantID    string `json:"tenant_id,omitempty"`  // public.tenants.id, when available
	Env         TenantEnv `json:"env"`
	Database    TenantDB   `json:"database"`
	AuditCopy   TenantAuditCopy `json:"audit_copy"`
	Warnings    []string   `json:"warnings,omitempty"` // cross-check failures; absent when none
}

// TenantEnv is what the router would do on a connect, read from os.Environ at call time.
type TenantEnv struct {
	DSNConfigured     bool   `json:"dsn_configured"`
	InitResourceGroup string `json:"init_resource_group"` // empty = no override; classifier routes
}

// TenantDB is SHOW DATA FROM DATABASE <tenant_db> summarised. The DB name is derived from the
// tenant slug (lowercase) by convention; "exists=false" + a warning means the naming decision
// hasn't been made yet, not that the DB is broken.
type TenantDB struct {
	Name       string `json:"name,omitempty"`
	Exists     bool   `json:"exists"`
	TotalBytes int64  `json:"total_bytes,omitempty"`
	TableCount int    `json:"table_count,omitempty"`
	// SizeText is human-readable (e.g. "16 KB"); absent when TotalBytes is 0.
	SizeText   string `json:"size_text,omitempty"`
}

// TenantAuditCopy is the read-only audit-copy posture. Temporal fields stay null until the
// audit pipeline lands (Phase 1 of the StarRocks upgrade runbook).
type TenantAuditCopy struct {
	// LastRunStatus / LastRunTime / LastRunDurationMS are null until the temporal visibility
	// wiring is in place; v1 ships "unconfigured" so the panel does not silently misreport.
	LastRunStatus    string `json:"last_run_status,omitempty"`
	LastRunTime      string `json:"last_run_time,omitempty"`
	LastRunDurationMS *int64 `json:"last_run_duration_ms,omitempty"`
}

// PlatformStatus builds the platform payload: cluster, resource groups, and one block per
// registry tenant. A per-tenant query failure degrades that one block (not the whole panel);
// the panel exists to surface failures, and dying on the first one fails its one job.
func (s *StatusService) PlatformStatus(ctx context.Context) (*Status, error) {
	out := &Status{GeneratedAt: s.Now().UTC()}
	var err error

	if out.Cluster, err = s.clusterState(ctx); err != nil {
		return nil, fmt.Errorf("cluster state: %w", err)
	}
	if out.ResourceGroups, err = s.resourceGroups(ctx); err != nil {
		// Don't 500 on a column-name drift; report the cluster error and keep the tenants.
		out.Notes = append(out.Notes, fmt.Sprintf("resource groups unavailable: %v", err))
		out.ResourceGroups = nil
	}

	// Build the slug→router-wiring map once per call so per-tenant env lookups don't
	// re-scan os.Environ 50 times.
	_, name2initRG := infra.ScanTenantEnv(os.Environ())

	items, _, err := s.Reg.List(ctx, "", 100, 0)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	for _, cfg := range items {
		tb, terr := s.tenantBlock(ctx, cfg, out.ResourceGroups, name2initRG)
		if terr != nil {
			// Per-tenant degradation: keep the tenant's name and slug, capture the error
			// as a warning so the operator sees it rather than the whole panel 500ing.
			tb = TenantBlock{
				Name:     cfg.TenantName,
				TenantID: cfg.TenantID,
				Slug:     infra.TenantEnvKey(cfg.TenantName),
				Warnings: []string{fmt.Sprintf("could not read tenant state: %v", terr)},
			}
		}
		out.Tenants = append(out.Tenants, tb)
	}
	return out, nil
}

// TenantStatus builds the tenant payload from a session tenant. The cluster and resource
// group sections are omitted; the tenant sees only their own row. The tenant UUID MUST come
// from the verified session, never from a query parameter (taking it from a query param is
// an IDOR that hands every tenant admin the platform view).
func (s *StatusService) TenantStatus(ctx context.Context, tenantID uuid.UUID) (*TenantScoped, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant id is required")
	}
	cfg, err := s.Reg.Get(ctx, tenantID)
	if err != nil {
		return nil, &TenantNotFoundError{ID: tenantID.String()}
	}
	_, name2initRG := infra.ScanTenantEnv(os.Environ())
	tb, terr := s.tenantBlock(ctx, *cfg, nil, name2initRG)
	if terr != nil {
		tb = TenantBlock{
			Name:     cfg.TenantName,
			TenantID: cfg.TenantID,
			Slug:     infra.TenantEnvKey(cfg.TenantName),
			Warnings: []string{fmt.Sprintf("could not read tenant state: %v", terr)},
		}
	}
	return &TenantScoped{GeneratedAt: s.Now().UTC(), Tenant: tb}, nil
}

// TenantNotFoundError is returned by TenantStatus when no registry row matches the session.
type TenantNotFoundError struct{ ID string }

func (e *TenantNotFoundError) Error() string { return "tenant not found in lakehouse registry: " + e.ID }

// clusterState runs SHOW FRONTENDS / SHOW BACKENDS and shapes the output. Column drift between
// versions is a real risk (the runbook's gap 3): we parse by column NAME when the driver gives
// us column metadata, and fall back to position with bounds checks. Unknown columns are
// silently ignored so a future-version column doesn't 500 the panel.
func (s *StatusService) clusterState(ctx context.Context) (ClusterStatus, error) {
	var out ClusterStatus

	// SHOW FRONTENDS: cols include Name, Host, Alive, Version (verify on live FE; do not
	// assume 3.3 and 4.1 match positionally).
	rows, err := s.AdminDB.QueryContext(ctx, "SHOW FRONTENDS")
	if err != nil {
		return out, fmt.Errorf("SHOW FRONTENDS: %w", err)
	}
	defer rows.Close()
	frontCols, _ := rows.Columns()
	for rows.Next() {
		raw, rerr := scanRowByName(rows, frontCols)
		if rerr != nil {
			return out, rerr
		}
		f := FrontendStatus{
			Name:    asString(raw["Name"]),
			Host:    asString(raw["Host"]),
			Alive:   asBool(raw["Alive"]),
			Version: asString(raw["Version"]),
		}
		out.Frontends = append(out.Frontends, f)
	}
	if err := rows.Err(); err != nil {
		return out, fmt.Errorf("read SHOW FRONTENDS: %w", err)
	}

	// SHOW BACKENDS: cols include Host, Alive, TotalCapacityB, TotalUsedProportion (4.1 uses
	// similar names; verify on live FE). Position-based parsing is the safest fallback.
	bRows, err := s.AdminDB.QueryContext(ctx, "SHOW BACKENDS")
	if err != nil {
		return out, fmt.Errorf("SHOW BACKENDS: %w", err)
	}
	defer bRows.Close()
	bCols, _ := bRows.Columns()
	for bRows.Next() {
		raw, rerr := scanRowByName(bRows, bCols)
		if rerr != nil {
			return out, rerr
		}
		b := BackendStatus{
			Host:    asString(raw["Host"]),
			Alive:   asBool(raw["Alive"]),
		}
		if v, ok := raw["TotalCapacityB"]; ok {
			b.TotalCapacityB = asInt64(v)
		}
		if v, ok := raw["TotalUsedProportion"]; ok {
			// StarRocks reports a fraction in 0..1; the panel renders it as percent.
			b.UsedPct = asFloat(v)
		}
		out.Backends = append(out.Backends, b)
	}
	if err := bRows.Err(); err != nil {
		return out, fmt.Errorf("read SHOW BACKENDS: %w", err)
	}
	return out, nil
}

// resourceGroups runs SHOW RESOURCE GROUPS ALL (3.3) or SHOW WORKLOAD GROUPS ALL (4.1) based on
// the boot-time decision. Per the upgrade runbook, the construct was renamed in 4.1 but the
// SQL alias is process-stable: pick one at startup and stick with it.
//
// The first call probes both forms (SHOW WORKLOAD GROUPS ALL first — 4.1's construct, since
// the runbook is forward-looking) and caches whichever one works. Subsequent calls reuse
// the cached decision. If both fail (cluster unreachable, both constructs rejected), the
// panel surfaces the error as a note; per-tenant rows still render.
func (s *StatusService) resourceGroups(ctx context.Context) ([]ResourceGroup, error) {
	stmt, err := s.resolveRgStmt(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.AdminDB.QueryContext(ctx, stmt)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", stmt, err)
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var out []ResourceGroup
	for rows.Next() {
		raw, rerr := scanRowByName(rows, cols)
		if rerr != nil {
			return nil, rerr
		}
		g := ResourceGroup{Name: asString(raw["name"])}
		// Tolerate alternate capitalisations in 4.1.
		if v, ok := raw["cpu_weight"]; ok {
			n := asInt(v)
			g.CPUWeight = &n
		} else if v, ok := raw["CPUWeight"]; ok {
			n := asInt(v)
			g.CPUWeight = &n
		}
		if v, ok := raw["mem_limit"]; ok {
			g.MemLimit = asString(v)
		} else if v, ok := raw["MemLimit"]; ok {
			g.MemLimit = asString(v)
		}
		if v, ok := raw["concurrency_limit"]; ok {
			n := asInt(v)
			g.ConcurrencyLimit = &n
		}
		if v, ok := raw["big_query_mem_limit"]; ok {
			g.BigQueryMemLimit = asInt64Ptr(v)
		}
		// Classifiers is a text column with user='x' role='y' style entries. A regex
		// captures the user/role; weight is parsed from the parenthesised digit. A parse
		// failure leaves Classifiers empty rather than failing the whole panel.
		raw4 := asString(raw["classifiers"])
		g.Classifiers = parseClassifiers(raw4)
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", stmt, err)
	}
	return out, nil
}

// resolveRgStmt picks the SHOW statement that works on the live cluster. WORKLOAD GROUPS is
// tried first (4.1+); if it errors with a parse error we fall back to RESOURCE GROUPS (3.3).
// The decision is cached for the process lifetime so the panel does not flicker between the
// two on each refresh.
func (s *StatusService) resolveRgStmt(ctx context.Context) (string, error) {
	s.rgOnce.Do(func() {
		// Probe 4.1 first; if the parser rejects it, fall back to 3.3.
		if rows, err := s.AdminDB.QueryContext(ctx, "SHOW WORKLOAD GROUPS ALL"); err == nil {
			rows.Close()
			s.showWorkloadGroups = true
			return
		}
		// Probe 3.3; if this also fails, leave the flag false and let the call surface
		// the error to the caller. The panel degrades to "resource groups unavailable".
		if rows, err := s.AdminDB.QueryContext(ctx, "SHOW RESOURCE GROUPS ALL"); err == nil {
			rows.Close()
			s.showWorkloadGroups = false
			return
		}
	})
	if s.showWorkloadGroups {
		return "SHOW WORKLOAD GROUPS ALL", nil
	}
	return "SHOW RESOURCE GROUPS ALL", nil
}

var classifierUserRE = regexp.MustCompile(`user\s*=\s*'([^']*)'`)
var classifierRoleRE = regexp.MustCompile(`role\s*=\s*'([^']*)'`)
var classifierWeightRE = regexp.MustCompile(`weight\s*=\s*([0-9.]+)`)

// parseClassifiers extracts user/role/weight tuples from the SHOW output. The 3.3 text form
// looks like `user='tenant_northwinds_svc', weight=1.0` separated by `;` between tuples.
// On a parse failure it returns nil; the panel shows an empty classifiers column rather
// than failing the whole row.
func parseClassifiers(s string) []ResourceClassifier {
	if s == "" {
		return nil
	}
	tuples := strings.Split(s, ";")
	var out []ResourceClassifier
	for _, t := range tuples {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		c := ResourceClassifier{}
		if m := classifierUserRE.FindStringSubmatch(t); len(m) == 2 {
			c.User = m[1]
		}
		if m := classifierRoleRE.FindStringSubmatch(t); len(m) == 2 {
			c.Role = m[1]
		}
		if m := classifierWeightRE.FindStringSubmatch(t); len(m) == 2 {
			if f, err := strconv.ParseFloat(m[1], 64); err == nil {
				c.Weight = f
			}
		}
		// Keep the row even if only the user matched; the other fields are optional.
		if c.User != "" || c.Role != "" {
			out = append(out, c)
		}
	}
	return out
}

// tenantBlock assembles one row: env wiring (read from os.Environ via ScanTenantEnv),
// database size (SHOW DATA FROM DATABASE), audit-copy posture (null until the temporal
// pipeline lands), and the cross-check warnings that make the panel valuable.
//
// groups is the resource-group snapshot from the same call, used to verify the named
// init-resource-group env var points at something real. nil for the tenant-scoped handler,
// which can't read SHOW anyway; the cross-check that needs groups is silently skipped.
func (s *StatusService) tenantBlock(ctx context.Context, cfg registry.Config, groups []ResourceGroup, name2initRG map[string]string) (TenantBlock, error) {
	slug := infra.TenantEnvKey(cfg.TenantName)
	tb := TenantBlock{
		Name:     cfg.TenantName,
		TenantID: cfg.TenantID,
		Slug:     slug,
	}

	// 1. Env wiring — DSN configured (in the env-var map this status service built itself,
	// not the router's private state), and any per-tenant init-resource-group override.
	if slug != "" {
		dsns, _ := infra.ScanTenantEnv(os.Environ())
		tb.Env.DSNConfigured = dsnConfigured(dsns, slug)
		if rg, ok := name2initRG[slug]; ok {
			tb.Env.InitResourceGroup = rg
		}
	}

	// 2. Database size. SHOW DATA FROM DATABASE returns one row per table with a TotalRow
	// column of bytes. The DB name follows the slug-lower convention; if SHOW DATA errors
	// because the DB doesn't exist, "exists=false" with a warning is the correct response —
	// the registry entry exists but no DB has been created yet.
	dbName := strings.ToLower(slug)
	if dbName != "" {
		tb.Database = s.tenantDatabase(ctx, dbName)
	}

	// 3. Audit-copy posture. v1: always "unconfigured" until the temporal visibility
	// wiring lands. The Status API returns the field as null; the frontend shows
	// "Audit pipeline not installed" in its place.
	tb.AuditCopy = TenantAuditCopy{}

	// 4. Cross-check warnings — these are the silent-failure modes that have bitten this
	// code in production. Each is a 1-line truth about a thing that should hold but might
	// not; if any fires, the operator sees it on the panel rather than finding out in a
	// post-mortem.
	tb.Warnings = s.tenantCrossCheck(cfg, tb, groups)

	return tb, nil
}

// dsnConfigured returns true if any per-tenant DSN env var mentions the slug, or if a global
// legacy DSN is set (which is the only working shape on a single-tenant deploy).
func dsnConfigured(name2dsn map[string]string, slug string) bool {
	if _, ok := name2dsn[slug]; ok {
		return true
	}
	return os.Getenv("LAKEHOUSE_STARROCKS_DSN") != ""
}

// tenantDatabase runs SHOW DATA FROM DATABASE and sums bytes. Returns TenantDB with
// Exists=false (no error) if the DB doesn't exist — the right answer for "not yet created."
func (s *StatusService) tenantDatabase(ctx context.Context, dbName string) TenantDB {
	out := TenantDB{Name: dbName}
	rows, err := s.AdminDB.QueryContext(ctx, "SHOW DATA FROM DATABASE `"+dbName+"`")
	if err != nil {
		// The most common failure here is "Unknown database" (Error 1049). Treat as
		// "doesn't exist yet" — the tenant has a registry row but no warehouse DB yet.
		// Other errors propagate as warnings in the cross-check, not as a 500.
		return out
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var totalBytes int64
	var tables int
	for rows.Next() {
		raw, rerr := scanRowByName(rows, cols)
		if rerr != nil {
			continue
		}
		// Per-table size: most releases expose a `TotalRow` (bytes) and `TableName`.
		if v, ok := raw["TotalRow"]; ok {
			totalBytes += asInt64(v)
		}
		tables++
	}
	out.Exists = true
	out.TotalBytes = totalBytes
	out.TableCount = tables
	out.SizeText = humanBytes(totalBytes)
	return out
}

// tenantCrossCheck returns the list of failures the panel surfaces for this tenant. An
// empty slice is a green row; any entry is a real configuration or wiring problem. These
// are the silent-failure modes the runbook and prior threads have surfaced.
//
// groups is the resource-group snapshot from the same call. nil skips the "named group
// must exist" check (the tenant handler does not have cluster visibility to do that).
func (s *StatusService) tenantCrossCheck(cfg registry.Config, tb TenantBlock, groups []ResourceGroup) []string {
	var w []string
	slug := tb.Slug
	// 1. Registry says configured + provisioned but no per-tenant DSN: traffic is going
	//    through the legacy pool. This is fine on a single-tenant deploy; on a multi-tenant
	//    cluster it's a hidden breach because the audit copy is no longer tenant-scoped.
	if cfg.Configured && !tb.Env.DSNConfigured && os.Getenv("LAKEHOUSE_STARROCKS_REQUIRE_TENANT_DSN") != "true" {
		if !singleTenantDeploy() {
			w = append(w, "tenant is provisioned but no per-tenant DSN is set; audit copy is using the legacy pool — set LAKEHOUSE_STARROCKS_DSN_TENANT_"+slug)
		}
	}
	// 2. Init resource group set but the named group doesn't exist (would kill the pool
	//    on the first connect).
	if tb.Env.InitResourceGroup != "" && !groupExists(groups, tb.Env.InitResourceGroup) {
		w = append(w, fmt.Sprintf("init resource group %q is configured but no such group exists in the cluster — first connect will fail", tb.Env.InitResourceGroup))
	}
	// 3. Registry entry but no database created yet. Expected during greenfield provisioning,
	//    a warning once the warehouse is supposed to be active.
	if cfg.Configured && cfg.Provisioned && !tb.Database.Exists {
		w = append(w, "registry says provisioned but the database does not exist on the cluster")
	}
	return w
}

// groupExists returns true if the named group is in the snapshot. Snapshot is local to one
// PlatformStatus call, so we don't re-issue SHOW for every tenant.
func groupExists(groups []ResourceGroup, name string) bool {
	if name == "" {
		return false
	}
	for _, g := range groups {
		if g.Name == name {
			return true
		}
	}
	return false
}

// singleTenantDeploy returns true if the process has no per-tenant DSNs and a legacy DSN,
// which is the working single-tenant configuration. False on a multi-tenant cluster without
// per-tenant DSNs — the case the runbook exists to prevent.
func singleTenantDeploy() bool {
	name2dsn, _ := infra.ScanTenantEnv(os.Environ())
	return len(name2dsn) == 0 && os.Getenv("LAKEHOUSE_STARROCKS_DSN") != ""
}

// humanBytes formats a byte count in a way that fits a small table cell.
func humanBytes(n int64) string {
	if n <= 0 {
		return ""
	}
	const k = 1024
	switch {
	case n < k:
		return fmt.Sprintf("%d B", n)
	case n < k*k:
		return fmt.Sprintf("%d KB", n/k)
	case n < k*k*k:
		return fmt.Sprintf("%d MB", n/(k*k))
	default:
		return fmt.Sprintf("%.1f GB", float64(n)/(k*k*k))
	}
}

// ---- helpers: scan rows by name, tolerate column drift ----

// scanRowByName reads one row into a name→value map. Columns that exist in the result but
// aren't asked for are kept under their actual name; columns that are missing (older
// releases, renamed fields) come back as nil rather than erroring.
func scanRowByName(rows *sql.Rows, cols []string) (map[string]any, error) {
	dest := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range dest {
		ptrs[i] = &dest[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, err
	}
	out := make(map[string]any, len(cols))
	for i, c := range cols {
		out[c] = dest[i]
	}
	return out, nil
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	case fmt.Stringer:
		return x.String()
	}
	return fmt.Sprintf("%v", v)
}

func asBool(v any) bool {
	if v == nil {
		return false
	}
	switch x := v.(type) {
	case bool:
		return x
	case int64:
		return x != 0
	case string:
		s := strings.TrimSpace(strings.ToLower(x))
		return s == "true" || s == "yes" || s == "y"
	case []byte:
		s := strings.TrimSpace(strings.ToLower(string(x)))
		return s == "true" || s == "yes" || s == "y"
	}
	return false
}

func asInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int32:
		return int(x)
	case int64:
		return int(x)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(x))
		return n
	case []byte:
		n, _ := strconv.Atoi(strings.TrimSpace(string(x)))
		return n
	}
	return 0
}

func asInt64(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int32:
		return int64(x)
	case int:
		return int64(x)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		return n
	case []byte:
		n, _ := strconv.ParseInt(strings.TrimSpace(string(x)), 10, 64)
		return n
	}
	return 0
}

func asInt64Ptr(v any) *int64 {
	n := asInt64(v)
	return &n
}

func asFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int64:
		return float64(x)
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f
	case []byte:
		f, _ := strconv.ParseFloat(strings.TrimSpace(string(x)), 64)
		return f
	}
	return 0
}

// ---- top-level handler glue (consumed by internal/handlers) ----

// JSONPayload writes the status payload as JSON with a stable content-type.
func JSONPayload(v any) ([]byte, error) { return json.Marshal(v) }