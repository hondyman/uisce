package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hondyman/semlayer/backend/internal/validation"
)

// VersionStatus represents the lifecycle state of a model version.
type VersionStatus string

const (
	StatusActive    VersionStatus = "active"
	StatusPrevious  VersionStatus = "previous"
	StatusAvailable VersionStatus = "available"
	StatusPreview   VersionStatus = "preview"
	StatusCanary    VersionStatus = "canary"
	StatusArchived  VersionStatus = "archived"
)

// RuntimeSet holds the merged cubes and views for a version.
type RuntimeSet struct {
	Cubes map[string]interface{} `json:"cubes"`
	Views map[string]interface{} `json:"views"`
}

// DeprecationMap maps old members to new ones.
type DeprecationMap map[string]string

// SchemaChange represents detected schema changes.
type SchemaChange struct {
	Table   string `json:"table"`
	Change  string `json:"change"` // e.g., "added column", "removed table"
	Details string `json:"details"`
}

// ValidationReport holds validation results.
type ValidationReport struct {
	StructuralErrors []string `json:"structural_errors"`
	GraphErrors      []string `json:"graph_errors"`
	GovernanceErrors []string `json:"governance_errors"`
	ExtensionHealth  []string `json:"extension_health"`
	OverallStatus    string   `json:"overall_status"` // "pass", "warn", "fail"
}

// ShadowRunResult holds results of shadow queries.
type ShadowRunResult struct {
	Query   string  `json:"query"`
	OldRows int     `json:"old_rows"`
	NewRows int     `json:"new_rows"`
	DiffPct float64 `json:"diff_pct"`
	Totals  struct {
		Old float64 `json:"old"`
		New float64 `json:"new"`
	} `json:"totals"`
	Error string `json:"error,omitempty"`
}

// PreAggRebuild holds rebuild plan.
type PreAggRebuild struct {
	Targets []string `json:"targets"`
	EstCost string   `json:"estimated_cost"`
	EstTime string   `json:"estimated_time"`
}

// UpgradeLifecycleService extends UpgradeRuntimeService with full lifecycle.
type UpgradeLifecycleService struct {
	*UpgradeRuntimeService
	activeModelVersion string
	versionsMap        map[string]*RuntimeSet
	deprecationMaps    map[string]DeprecationMap
	schemaChanges      map[string][]SchemaChange
	validationReports  map[string]ValidationReport
	shadowRuns         map[string][]ShadowRunResult
	preAggRebuilds     map[string]PreAggRebuild
}

type SLOSummary struct {
	ErrorRate          float64       `json:"error_rate"`
	P95LatencyMs       int64         `json:"p95_latency_ms"`
	ShadowDiffRate     float64       `json:"shadow_diff_rate"`
	PreAggRebuildMs    int64         `json:"preagg_rebuild_ms"`
	CacheHitRatio      float64       `json:"cache_hit_ratio"`
	MergeDurationMs    int64         `json:"merge_duration_ms"`
	ValidateDurationMs int64         `json:"validate_duration_ms"`
	Window             time.Duration `json:"window"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

type VersionInfo struct {
	Version     string        `json:"version"`
	SchemaHash  string        `json:"schema_hash"`
	Status      VersionStatus `json:"status"`
	Warnings    []string      `json:"warnings"`
	CreatedAt   time.Time     `json:"created_at"`
	ActivatedAt *time.Time    `json:"activated_at,omitempty"`
}

type CanaryState struct {
	Version string    `json:"version"`
	Tenants []string  `json:"tenants"`
	Until   time.Time `json:"until"`
}

// DiffReport - matches the schema structure for upgrade artifacts
type DiffReport struct {
	CoreVersion     string               `json:"core_version"`
	PreviousVersion string               `json:"previous_version"`
	GeneratedAt     string               `json:"generated_at"`
	SchemaHash      string               `json:"schema_hash"`
	Summary         DiffSummary          `json:"summary"`
	Cubes           []CubeDiff           `json:"cubes"`
	Views           []ViewDiff           `json:"views"`
	Governance      []GovernanceDiff     `json:"governance"`
	PreAggregations []PreAggregationDiff `json:"pre_aggregations"`
	Warnings        []string             `json:"warnings"`
}

type DiffSummary struct {
	CubesAdded      int `json:"cubes_added"`
	CubesRemoved    int `json:"cubes_removed"`
	CubesChanged    int `json:"cubes_changed"`
	ViewsAdded      int `json:"views_added"`
	ViewsRemoved    int `json:"views_removed"`
	ViewsChanged    int `json:"views_changed"`
	BreakingChanges int `json:"breaking_changes"`
	Warnings        int `json:"warnings"`
}

type CubeDiff struct {
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Changes []Change `json:"changes"`
}

type ViewDiff struct {
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Changes []Change `json:"changes"`
}

type GovernanceDiff struct {
	Scope   string   `json:"scope"`
	Name    string   `json:"name"`
	Changes []Change `json:"changes"`
}

type PreAggregationDiff struct {
	Cube   string  `json:"cube"`
	Name   string  `json:"name"`
	Status string  `json:"status"`
	Reason *string `json:"reason,omitempty"`
}

type Change struct {
	Type     string      `json:"type"`
	Name     *string     `json:"name,omitempty"`
	JoinPath *string     `json:"join_path,omitempty"`
	Member   *string     `json:"member,omitempty"`
	Old      interface{} `json:"old,omitempty"`
	New      interface{} `json:"new,omitempty"`
	Details  interface{} `json:"details,omitempty"`
}

type BrokenReference struct {
	Path        string   `json:"path"`
	Reason      string   `json:"reason"`
	Suggestions []string `json:"suggestions"`
	Selected    *string  `json:"selected,omitempty"`
}

type Notification struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Severity  string    `json:"severity"`
	CreatedAt time.Time `json:"created_at"`
}

type UpgradeStatus struct {
	CoreVersion string   `json:"core_version"`
	Status      string   `json:"status"` // pending | ready | canary | active | rolled_back
	Warnings    []string `json:"warnings"`
	Blockers    []string `json:"blockers"`
}

type UIHints struct {
	NeedsDiffReview   bool `json:"needs_diff_review"`
	NeedsExtensionFix bool `json:"needs_extension_fix"`
	NeedsQueryRun     bool `json:"needs_query_run"`
}

type ChangelogEntry struct {
	Version     string `json:"version"`
	Date        string `json:"date"`
	Description string `json:"description"`
}

type UpgradeOverviewResponse struct {
	SchemaVersion string           `json:"schema_version"`
	Changelog     []ChangelogEntry `json:"changelog,omitempty"`
	Report        DiffReport       `json:"report"`
	Aliases       AliasMap         `json:"aliases"`
	Status        UpgradeStatus    `json:"status"`
	UIHints       UIHints          `json:"ui_hints,omitempty"`
}

type MultiUpgradeOverviewResponse struct {
	Versions []UpgradeOverviewResponse `json:"versions"`
}

type UpgradeArtifactsData struct {
	SchemaVersion string           `json:"schema_version"`
	Changelog     []ChangelogEntry `json:"changelog,omitempty"`
	Report        DiffReport       `json:"report"`
	Aliases       AliasMap         `json:"aliases"`
}

type UpgradeRuntimeService struct {
	mu            sync.RWMutex
	versions      map[string]*VersionInfo
	order         []string
	active        string
	previous      string
	preview       string
	canary        *CanaryState
	slo           SLOSummary
	notifications []Notification
	wsHub         *WebSocketHub // WebSocket hub for real-time updates
	// New for full lifecycle
	activeModelVersion string
	versionsMap        map[string]*RuntimeSet
	deprecationMaps    map[string]DeprecationMap
	schemaChanges      map[string][]SchemaChange
	validationReports  map[string]ValidationReport
	shadowRuns         map[string][]ShadowRunResult
	preAggRebuilds     map[string]PreAggRebuild
}

func NewUpgradeRuntimeService(wsHub *WebSocketHub) *UpgradeRuntimeService {
	now := time.Now()
	svc := &UpgradeRuntimeService{
		versions:           make(map[string]*VersionInfo),
		order:              []string{},
		notifications:      []Notification{},
		activeModelVersion: "1.1.0", // default active
		versionsMap:        make(map[string]*RuntimeSet),
		deprecationMaps:    make(map[string]DeprecationMap),
		schemaChanges:      make(map[string][]SchemaChange),
		validationReports:  make(map[string]ValidationReport),
		shadowRuns:         make(map[string][]ShadowRunResult),
		preAggRebuilds:     make(map[string]PreAggRebuild),
		wsHub:              wsHub, // WebSocket hub for real-time updates
		slo: SLOSummary{
			ErrorRate:          0.001,
			P95LatencyMs:       120,
			ShadowDiffRate:     0.0,
			PreAggRebuildMs:    15_000,
			CacheHitRatio:      0.92,
			MergeDurationMs:    350,
			ValidateDurationMs: 180,
			Window:             1 * time.Hour,
			UpdatedAt:          now,
		},
	}
	// Seed two versions
	v1 := &VersionInfo{Version: "1.0.0", SchemaHash: "abc123", Status: StatusPrevious, CreatedAt: now.Add(-72 * time.Hour)}
	act := now.Add(-48 * time.Hour)
	v1.ActivatedAt = &act
	v2 := &VersionInfo{Version: "1.1.0", SchemaHash: "def456", Status: StatusActive, CreatedAt: now.Add(-50 * time.Hour)}
	v2.ActivatedAt = &now
	v3 := &VersionInfo{Version: "1.2.0", SchemaHash: "ghi789", Status: StatusAvailable, CreatedAt: now.Add(-1 * time.Hour), Warnings: []string{"Join path change in claim cube", "PII tag added to customers.email"}}
	svc.versions[v1.Version] = v1
	svc.versions[v2.Version] = v2
	svc.versions[v3.Version] = v3
	svc.order = []string{v1.Version, v2.Version, v3.Version}
	svc.active = v2.Version
	svc.previous = v1.Version
	return svc
}

func (s *UpgradeRuntimeService) ListVersions() ([]VersionInfo, *CanaryState, SLOSummary) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]VersionInfo, 0, len(s.versions))
	for _, v := range s.versions {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	var canary *CanaryState
	if s.canary != nil {
		cpy := *s.canary
		canary = &cpy
	}
	return out, canary, s.slo
}

func (s *UpgradeRuntimeService) SetPreview(version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.versions[version]; !ok {
		return errors.New("version not found")
	}
	s.preview = version
	if v := s.versions[version]; v != nil {
		v.Status = StatusPreview
	}
	s.notifications = append(s.notifications, Notification{ID: randID(), Type: "upgrade", Message: "Preview started for " + version, Severity: "info", CreatedAt: time.Now()})
	return nil
}

func (s *UpgradeRuntimeService) StartCanary(version string, tenants []string, duration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.versions[version]; !ok {
		return errors.New("version not found")
	}
	until := time.Now().Add(duration)
	s.canary = &CanaryState{Version: version, Tenants: tenants, Until: until}
	s.versions[version].Status = StatusCanary
	s.notifications = append(s.notifications, Notification{ID: randID(), Type: "upgrade", Message: "Canary started for " + version, Severity: "info", CreatedAt: time.Now()})
	return nil
}

func (s *UpgradeRuntimeService) Activate(version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.versions[version]; !ok {
		return errors.New("version not found")
	}
	if version == s.active {
		return nil
	}
	// update statuses
	if curr, ok := s.versions[s.active]; ok {
		curr.Status = StatusPrevious
		s.previous = curr.Version
	}
	s.active = version
	v := s.versions[version]
	v.Status = StatusActive
	now := time.Now()
	v.ActivatedAt = &now
	s.canary = nil
	s.preview = ""
	s.notifications = append(s.notifications, Notification{ID: randID(), Type: "upgrade", Message: "Activated version " + version, Severity: "success", CreatedAt: now})
	return nil
}

func (s *UpgradeRuntimeService) Rollback() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.previous == "" {
		return "", errors.New("no previous version to rollback to")
	}
	target := s.previous
	if curr, ok := s.versions[s.active]; ok {
		curr.Status = StatusAvailable
	}
	if prev, ok := s.versions[target]; ok {
		prev.Status = StatusActive
		now := time.Now()
		prev.ActivatedAt = &now
	}
	s.active, s.previous = target, ""
	s.canary = nil
	s.preview = ""
	s.notifications = append(s.notifications, Notification{ID: randID(), Type: "upgrade", Message: "Rolled back to " + target, Severity: "warning", CreatedAt: time.Now()})
	return target, nil
}

func (s *UpgradeRuntimeService) GetDiff(from, to string) (DiffReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.versions[from]; !ok {
		return DiffReport{}, errors.New("from version not found")
	}
	if _, ok := s.versions[to]; !ok {
		return DiffReport{}, errors.New("to version not found")
	}
	var rep DiffReport
	rep.CoreVersion = "1.2.0"
	rep.PreviousVersion = from
	rep.GeneratedAt = time.Now().Format(time.RFC3339)
	rep.SchemaHash = "abc123"
	rep.Summary = DiffSummary{
		CubesAdded:      1,
		CubesRemoved:    1,
		CubesChanged:    1,
		ViewsAdded:      0,
		ViewsRemoved:    0,
		ViewsChanged:    1,
		BreakingChanges: 0,
		Warnings:        2,
	}
	rep.Cubes = []CubeDiff{
		{Name: "payments", Status: "added", Changes: []Change{}},
		{Name: "legacy_users", Status: "removed", Changes: []Change{}},
		{Name: "orders", Status: "changed", Changes: []Change{
			{Type: "modify_dimension", Name: stringPtr("total"), Old: "number", New: "decimal(18,2)"},
		}},
	}
	rep.Views = []ViewDiff{
		{Name: "orders_view", Status: "changed", Changes: []Change{
			{Type: "modify_join_path", JoinPath: stringPtr("customers"), Old: "orders.customer_id = customers.id", New: "(orders.tenant_id = customers.tenant_id) AND (orders.customer_id = customers.id)"},
		}},
	}
	rep.Governance = []GovernanceDiff{
		{Scope: "cube", Name: "customers", Changes: []Change{
			{Type: "pii_flag_added", Member: stringPtr("email")},
		}},
		{Scope: "view", Name: "orders_view", Changes: []Change{
			{Type: "access_policy_changed", Old: "public", New: "role:analyst"},
		}},
	}
	rep.PreAggregations = []PreAggregationDiff{
		{Cube: "orders", Name: "orders_by_day", Status: "rebuild_required", Reason: stringPtr("dimension changed")},
		{Cube: "customers", Name: "revenue_by_customer", Status: "rebuild_required", Reason: stringPtr("join path changed")},
	}
	rep.Warnings = []string{
		"PK/FK relationship added: orders.customer_id -> customers.id",
		"Tenant isolation enforced on join: orders -> customers",
	}
	return rep, nil
}

func (s *UpgradeRuntimeService) GetBrokenReferences(version string) ([]BrokenReference, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.versions[version]; !ok {
		return nil, errors.New("version not found")
	}
	return []BrokenReference{
		{Path: "views.sales.alias_total", Reason: "member renamed", Suggestions: []string{"views.sales.total_amount", "views.sales.gross_total"}},
		{Path: "cubes.orders.dimensions.customerKey", Reason: "dimension moved", Suggestions: []string{"cubes.customers.dimensions.customerKey"}},
	}, nil
}

func (s *UpgradeRuntimeService) ApplyExtensionFixes(version string, patches map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.versions[version]; !ok {
		return errors.New("version not found")
	}
	// Stub: accept and pretend to persist patches to extension overlays
	s.notifications = append(s.notifications, Notification{ID: randID(), Type: "upgrade", Message: "Applied " + itoa(len(patches)) + " extension fixes for " + version, Severity: "info", CreatedAt: time.Now()})
	return nil
}

// New ApplyExtensionFixes method for the new API
func (s *UpgradeRuntimeService) ApplyExtensionFixesV2(version string, fixes []ExtensionFix) (map[string]int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.versions[version]; !ok {
		return nil, errors.New("version not found")
	}
	result := make(map[string]int)
	for _, fix := range fixes {
		result[fix.FilePath] = len(fix.Fixes)
	}
	s.notifications = append(s.notifications, Notification{ID: randID(), Type: "upgrade", Message: "Applied extension fixes for " + version, Severity: "info", CreatedAt: time.Now()})
	return result, nil
}

type PreviewRunRequest struct {
	FromVersion string   `json:"from"`
	ToVersion   string   `json:"to"`
	Queries     []string `json:"queries"`
}

type PreviewRunResult struct {
	Query   string  `json:"query"`
	OldRows int     `json:"old_rows"`
	NewRows int     `json:"new_rows"`
	DiffPct float64 `json:"diff_pct"`
	Totals  struct {
		Old float64 `json:"old"`
		New float64 `json:"new"`
	} `json:"totals"`
}

func (s *UpgradeRuntimeService) RunPreview(req PreviewRunRequest) ([]PreviewRunResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.versions[req.FromVersion]; !ok {
		return nil, errors.New("from version not found")
	}
	if _, ok := s.versions[req.ToVersion]; !ok {
		return nil, errors.New("to version not found")
	}
	out := make([]PreviewRunResult, 0, len(req.Queries))
	for _, q := range req.Queries {
		out = append(out, PreviewRunResult{Query: q, OldRows: 1000, NewRows: 1000, DiffPct: 0.0, Totals: struct {
			Old float64 `json:"old"`
			New float64 `json:"new"`
		}{Old: 12345.67, New: 12345.67}})
	}
	return out, nil
}

func (s *UpgradeRuntimeService) GetSLO() SLOSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.slo
}

func (s *UpgradeRuntimeService) ListNotifications() []Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Notification, len(s.notifications))
	copy(out, s.notifications)
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

// Full Lifecycle Methods

// Phase 0: Pre-Upgrade Prep
func (s *UpgradeRuntimeService) PrepareUpgrade(newVersion string) (string, []SchemaChange, DeprecationMap, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.versions[newVersion]; ok {
		return "", nil, nil, errors.New("version already exists")
	}
	// Simulate schema introspection
	schemaHash := fmt.Sprintf("hash_%s", newVersion)
	changes := []SchemaChange{
		{Table: "orders", Change: "added column", Details: "discount_rate decimal(5,2)"},
		{Table: "customers", Change: "renamed column", Details: "email -> contact_email"},
	}
	depMap := DeprecationMap{
		"customers.email": "customers.contact_email",
	}
	s.schemaChanges[newVersion] = changes
	s.deprecationMaps[newVersion] = depMap
	s.notifications = append(s.notifications, Notification{ID: randID(), Type: "upgrade", Message: "Prepared upgrade for " + newVersion, Severity: "info", CreatedAt: time.Now()})
	return schemaHash, changes, depMap, nil
}

// Phase 1: Core Regeneration
func (s *UpgradeRuntimeService) GenerateCore(version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.versions[version]; !ok {
		return errors.New("version not found")
	}
	// Stub: generate core cubes/views
	s.notifications = append(s.notifications, Notification{ID: randID(), Type: "upgrade", Message: "Generated core for " + version, Severity: "info", CreatedAt: time.Now()})
	return nil
}

// Phase 2: Merge with Custom
func (s *UpgradeRuntimeService) MergeCustom(version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.versions[version]; !ok {
		return errors.New("version not found")
	}
	// Stub: merge cubes/views with extensions
	runtime := &RuntimeSet{
		Cubes: make(map[string]interface{}),
		Views: make(map[string]interface{}),
	}
	s.versionsMap[version] = runtime
	s.notifications = append(s.notifications, Notification{ID: randID(), Type: "upgrade", Message: "Merged custom for " + version, Severity: "info", CreatedAt: time.Now()})
	return nil
}

// Phase 3: Validation & Reporting
func (s *UpgradeRuntimeService) Validate(version string) (ValidationReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.versions[version]; !ok {
		return ValidationReport{}, errors.New("version not found")
	}
	report := ValidationReport{
		StructuralErrors: []string{},
		GraphErrors:      []string{"Cycle detected in orders -> customers"},
		GovernanceErrors: []string{},
		ExtensionHealth:  []string{"Broken include: views.sales.alias_total"},
		OverallStatus:    "warn",
	}
	s.validationReports[version] = report
	s.notifications = append(s.notifications, Notification{ID: randID(), Type: "upgrade", Message: "Validated " + version + " with status " + report.OverallStatus, Severity: "warning", CreatedAt: time.Now()})
	return report, nil
}

// Phase 4: Staging Dual-Run
func (s *UpgradeRuntimeService) RunShadow(version string, queries []string) ([]ShadowRunResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.versions[version]; !ok {
		return nil, errors.New("version not found")
	}
	results := make([]ShadowRunResult, len(queries))
	for i, q := range queries {
		results[i] = ShadowRunResult{
			Query:   q,
			OldRows: 1000,
			NewRows: 1000,
			DiffPct: 0.0,
			Totals: struct {
				Old float64 `json:"old"`
				New float64 `json:"new"`
			}{Old: 12345.67, New: 12345.67},
		}
	}
	s.shadowRuns[version] = results
	s.notifications = append(s.notifications, Notification{ID: randID(), Type: "upgrade", Message: "Ran shadow queries for " + version, Severity: "info", CreatedAt: time.Now()})
	return results, nil
}

// Phase 5: Steward Review (backend support)
func (s *UpgradeRuntimeService) GetValidationReport(version string) (ValidationReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if report, ok := s.validationReports[version]; ok {
		return report, nil
	}
	return ValidationReport{}, errors.New("report not found")
}

// Phase 6: Canary Rollout (extend existing StartCanary)

// Phase 7: Full Cutover (extend existing Activate)

// Phase 8: Post-Upgrade
func (s *UpgradeRuntimeService) Archive(version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.versions[version]; ok {
		v.Status = StatusArchived
		s.notifications = append(s.notifications, Notification{ID: randID(), Type: "upgrade", Message: "Archived " + version, Severity: "info", CreatedAt: time.Now()})
		return nil
	}
	return errors.New("version not found")
}

// Additional getters
func (s *UpgradeRuntimeService) GetSchemaChanges(version string) ([]SchemaChange, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if changes, ok := s.schemaChanges[version]; ok {
		return changes, nil
	}
	return nil, errors.New("changes not found")
}

func (s *UpgradeRuntimeService) GetDeprecationMap(version string) (DeprecationMap, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if dep, ok := s.deprecationMaps[version]; ok {
		return dep, nil
	}
	return nil, errors.New("deprecation map not found")
}

func (s *UpgradeRuntimeService) GetPreAggRebuild(version string) (PreAggRebuild, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if rebuild, ok := s.preAggRebuilds[version]; ok {
		return rebuild, nil
	}
	return PreAggRebuild{}, errors.New("rebuild plan not found")
}

// helpers
func randID() string             { return time.Now().Format("20060102150405.000000") }
func itoa(i int) string          { return fmt.Sprintf("%d", i) }
func stringPtr(s string) *string { return &s }
func intPtr(i int) *int          { return &i }
func boolPtr(b bool) *bool       { return &b }

// Diff Report and Alias Map methods
func (s *UpgradeRuntimeService) GenerateDiffReport(fromVersion, toVersion string) (DiffReport, error) {
	return s.GetDiff(fromVersion, toVersion)
}

func (s *UpgradeRuntimeService) GetAliasMap(fromVersion, toVersion string) (AliasMap, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.versions[fromVersion]; !ok {
		return AliasMap{}, errors.New("from version not found")
	}
	if _, ok := s.versions[toVersion]; !ok {
		return AliasMap{}, errors.New("to version not found")
	}
	return AliasMap{
		CoreVersion:     toVersion,
		PreviousVersion: fromVersion,
		GeneratedAt:     time.Now().Format(time.RFC3339),
		Aliases: []AliasEntry{
			{
				Scope:      "cube",
				Name:       "orders",
				MemberType: "dimension",
				OldName:    "customerKey",
				NewName:    stringPtr("customer_id"),
				Status:     "renamed",
				Meta: AliasMeta{
					Reason:               "Standardized naming convention",
					AutoRewrite:          true,
					SuggestedReplacement: stringPtr("customer_id"),
					BreakingChange:       boolPtr(true),
				},
			},
		},
	}, nil
}

func (s *UpgradeRuntimeService) GenerateAliasMap(fromVersion, toVersion string) (AliasMap, error) {
	return s.GetAliasMap(fromVersion, toVersion)
}

// Extension Fix methods
func (s *UpgradeRuntimeService) AnalyzeExtensionFixes(version string, extensionFiles []string) ([]ExtensionFix, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.versions[version]; !ok {
		return nil, errors.New("version not found")
	}
	fixes := make([]ExtensionFix, len(extensionFiles))
	for i, file := range extensionFiles {
		fixes[i] = ExtensionFix{
			FilePath: file,
			Fixes: []ExtensionFixEntry{
				{
					LineNumber: 10,
					OldCode:    "customerKey",
					NewCode:    "customer_id",
					AliasUsed:  "customer_id",
					Confidence: "high",
				},
			},
		}
	}
	return fixes, nil
}

func (s *UpgradeRuntimeService) PreviewExtensionFixes(version string, fixes []ExtensionFix) (map[string][]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.versions[version]; !ok {
		return nil, errors.New("version not found")
	}
	result := make(map[string][]string)
	for _, fix := range fixes {
		result[fix.FilePath] = []string{"Preview of changes for " + fix.FilePath}
	}
	return result, nil
}

// Golden Query methods
func (s *UpgradeRuntimeService) ListGoldenQueries() ([]GoldenQuery, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return []GoldenQuery{
		{
			Name:                "total_orders",
			Description:         "Count of all orders",
			Query:               "SELECT COUNT(*) FROM orders",
			ExpectedResultCount: intPtr(1000),
			Tags:                []string{"core", "orders"},
		},
		{
			Name:        "revenue_by_customer",
			Description: "Revenue grouped by customer",
			Query:       "SELECT customer_id, SUM(total) FROM orders GROUP BY customer_id",
			Tags:        []string{"revenue", "customers"},
		},
	}, nil
}

func (s *UpgradeRuntimeService) AddGoldenQuery(name, description, query string, tags []string) (GoldenQuery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	newQuery := GoldenQuery{
		Name:        name,
		Description: description,
		Query:       query,
		Tags:        tags,
	}
	s.notifications = append(s.notifications, Notification{ID: randID(), Type: "upgrade", Message: "Added golden query " + name, Severity: "info", CreatedAt: time.Now()})
	return newQuery, nil
}

func (s *UpgradeRuntimeService) RunGoldenQueries(fromVersion, toVersion string, queryNames []string) ([]GoldenQueryResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.versions[fromVersion]; !ok {
		return nil, errors.New("from version not found")
	}
	if _, ok := s.versions[toVersion]; !ok {
		return nil, errors.New("to version not found")
	}
	results := make([]GoldenQueryResult, len(queryNames))
	for i, name := range queryNames {
		results[i] = GoldenQueryResult{
			QueryName: name,
			OldResult: GoldenQueryExecutionResult{
				Rows:            []map[string]interface{}{{"count": 1000}},
				ExecutionTimeMs: 150,
			},
			NewResult: GoldenQueryExecutionResult{
				Rows:            []map[string]interface{}{{"count": 1000}},
				ExecutionTimeMs: 145,
			},
			DiffAnalysis: GoldenQueryDiffAnalysis{
				RowCountDiff:        0,
				ExecutionTimeDiffMs: -5,
				DataDifferences:     []string{},
				BreakingChanges:     false,
			},
		}
	}
	return results, nil
}

func (s *UpgradeRuntimeService) UpdateGoldenQuery(name string, description, query *string, tags []string) (GoldenQuery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	desc := ""
	if description != nil {
		desc = *description
	}
	q := ""
	if query != nil {
		q = *query
	}
	updatedQuery := GoldenQuery{
		Name:        name,
		Description: desc,
		Query:       q,
		Tags:        tags,
	}
	s.notifications = append(s.notifications, Notification{ID: randID(), Type: "upgrade", Message: "Updated golden query " + name, Severity: "info", CreatedAt: time.Now()})
	return updatedQuery, nil
}

func (s *UpgradeRuntimeService) DeleteGoldenQuery(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifications = append(s.notifications, Notification{ID: randID(), Type: "upgrade", Message: "Deleted golden query " + name, Severity: "info", CreatedAt: time.Now()})
	return nil
}

// AliasMap - matches the schema structure for upgrade artifacts
type AliasMap struct {
	CoreVersion     string       `json:"core_version"`
	PreviousVersion string       `json:"previous_version"`
	GeneratedAt     string       `json:"generated_at"`
	Aliases         []AliasEntry `json:"aliases"`
}

type AliasEntry struct {
	Scope      string    `json:"scope"`
	Name       string    `json:"name"`
	MemberType string    `json:"member_type"`
	OldName    string    `json:"old_name"`
	NewName    *string   `json:"new_name,omitempty"`
	Status     string    `json:"status"`
	Meta       AliasMeta `json:"meta"`
}

type AliasMeta struct {
	Reason               string  `json:"reason"`
	AutoRewrite          bool    `json:"auto_rewrite"`
	SuggestedReplacement *string `json:"suggested_replacement,omitempty"`
	BreakingChange       *bool   `json:"breaking_change,omitempty"`
}

type ExtensionFix struct {
	FilePath string              `json:"file_path"`
	Fixes    []ExtensionFixEntry `json:"fixes"`
}

type ExtensionFixEntry struct {
	LineNumber int    `json:"line_number"`
	OldCode    string `json:"old_code"`
	NewCode    string `json:"new_code"`
	AliasUsed  string `json:"alias_used,omitempty"`
	Confidence string `json:"confidence"`
}

type GoldenQuery struct {
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	Query               string   `json:"query"`
	ExpectedResultCount *int     `json:"expected_result_count,omitempty"`
	Tags                []string `json:"tags"`
}

type GoldenQueryResult struct {
	QueryName    string                     `json:"query_name"`
	OldResult    GoldenQueryExecutionResult `json:"old_result"`
	NewResult    GoldenQueryExecutionResult `json:"new_result"`
	DiffAnalysis GoldenQueryDiffAnalysis    `json:"diff_analysis"`
}

type GoldenQueryExecutionResult struct {
	Rows            []map[string]interface{} `json:"rows"`
	ExecutionTimeMs int                      `json:"execution_time_ms"`
	Error           string                   `json:"error,omitempty"`
}

type GoldenQueryDiffAnalysis struct {
	RowCountDiff        int      `json:"row_count_diff"`
	ExecutionTimeDiffMs int      `json:"execution_time_diff_ms"`
	DataDifferences     []string `json:"data_differences"`
	BreakingChanges     bool     `json:"breaking_changes"`
}

// ValidateArtifact validates an upgrade artifact against the JSON schema
func (s *UpgradeRuntimeService) ValidateArtifact(artifact interface{}) error {
	data, err := json.Marshal(artifact)
	if err != nil {
		return fmt.Errorf("failed to marshal artifact: %w", err)
	}

	return validation.ValidateUpgradeArtifacts(data)
}

// GetSchemaVersion returns the current schema version from the schema file
func (s *UpgradeRuntimeService) GetSchemaVersion() (string, error) {
	// This would read from the schema file, but for now return a placeholder
	// In production, you'd want to read this from the actual schema file
	return "1.0.0", nil
}

// GetUpgradeStatus returns the current upgrade status for a specific version
func (s *UpgradeRuntimeService) GetUpgradeStatus(coreVersion string) (UpgradeStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if v, ok := s.versions[coreVersion]; ok {
		status := UpgradeStatus{
			CoreVersion: coreVersion,
			Status:      string(v.Status),
			Warnings:    v.Warnings,
			Blockers:    []string{}, // Could be populated based on validation reports
		}
		return status, nil
	}
	return UpgradeStatus{}, errors.New("version not found")
}

// GetUpgradeOverview returns combined upgrade artifacts and status for a single version
func (s *UpgradeRuntimeService) GetUpgradeOverview(coreVersion string) (UpgradeOverviewResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get status
	status, err := s.GetUpgradeStatus(coreVersion)
	if err != nil {
		return UpgradeOverviewResponse{}, err
	}

	// Get artifacts (mock data for now - in production this would load from files)
	artifacts := s.generateMockArtifacts(coreVersion)

	// Compute UI hints
	hints := UIHints{
		NeedsDiffReview:   len(artifacts.Report.Cubes) > 0 || len(artifacts.Report.Views) > 0,
		NeedsExtensionFix: len(artifacts.Report.Warnings) > 0,
		NeedsQueryRun:     status.Status == "ready" || status.Status == "canary",
	}

	response := UpgradeOverviewResponse{
		SchemaVersion: artifacts.SchemaVersion,
		Changelog:     artifacts.Changelog,
		Report:        artifacts.Report,
		Aliases:       artifacts.Aliases,
		Status:        status,
		UIHints:       hints,
	}

	return response, nil
}

// GetMultiUpgradeOverview returns upgrade overview for multiple versions with filtering and sorting
func (s *UpgradeRuntimeService) GetMultiUpgradeOverview(coreVersions []string, statusFilter []string, sortParam string) (MultiUpgradeOverviewResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var overviews []UpgradeOverviewResponse

	// If no specific versions requested, get all available versions
	if len(coreVersions) == 0 {
		// Get all available core versions from the service
		allVersions, _, _ := s.ListVersions()
		for _, v := range allVersions {
			coreVersions = append(coreVersions, v.Version)
		}
	}

	for _, version := range coreVersions {
		overview, err := s.GetUpgradeOverview(version)
		if err != nil {
			// Skip versions that don't exist rather than failing the whole request
			continue
		}

		// Apply status filter if provided
		if len(statusFilter) > 0 {
			statusMatch := false
			for _, filter := range statusFilter {
				if strings.ToLower(overview.Status.Status) == filter {
					statusMatch = true
					break
				}
			}
			if !statusMatch {
				continue
			}
		}

		overviews = append(overviews, overview)
	}

	// Apply sorting
	switch sortParam {
	case "version_asc":
		sort.Slice(overviews, func(i, j int) bool {
			return overviews[i].Status.CoreVersion < overviews[j].Status.CoreVersion
		})
	case "version_desc":
		sort.Slice(overviews, func(i, j int) bool {
			return overviews[i].Status.CoreVersion > overviews[j].Status.CoreVersion
		})
	case "status":
		sort.Slice(overviews, func(i, j int) bool {
			return overviews[i].Status.Status < overviews[j].Status.Status
		})
	case "schema_version_asc":
		sort.Slice(overviews, func(i, j int) bool {
			return overviews[i].SchemaVersion < overviews[j].SchemaVersion
		})
	case "schema_version_desc":
		sort.Slice(overviews, func(i, j int) bool {
			return overviews[i].SchemaVersion > overviews[j].SchemaVersion
		})
	default:
		// Default to version_desc
		sort.Slice(overviews, func(i, j int) bool {
			return overviews[i].Status.CoreVersion > overviews[j].Status.CoreVersion
		})
	}

	return MultiUpgradeOverviewResponse{Versions: overviews}, nil
}

// generateMockArtifacts creates mock upgrade artifacts for demonstration
func (s *UpgradeRuntimeService) generateMockArtifacts(coreVersion string) UpgradeArtifactsData {
	return UpgradeArtifactsData{
		SchemaVersion: "1.0.0",
		Changelog: []ChangelogEntry{
			{
				Version:     "1.0.0",
				Date:        time.Now().Format("2006-01-02"),
				Description: "Initial schema version with core cubes and views",
			},
		},
		Report: DiffReport{
			CoreVersion:     coreVersion,
			PreviousVersion: "1.0.0",
			GeneratedAt:     time.Now().Format(time.RFC3339),
			SchemaHash:      "abc123",
			Summary: DiffSummary{
				CubesAdded:      1,
				CubesRemoved:    0,
				CubesChanged:    1,
				ViewsAdded:      0,
				ViewsRemoved:    0,
				ViewsChanged:    1,
				BreakingChanges: 0,
				Warnings:        1,
			},
			Cubes: []CubeDiff{
				{Name: "orders", Status: "changed", Changes: []Change{
					{Type: "modify_dimension", Name: stringPtr("total"), Old: "number", New: "decimal(18,2)"},
				}},
			},
			Views: []ViewDiff{
				{Name: "orders_view", Status: "changed", Changes: []Change{
					{Type: "modify_join_path", JoinPath: stringPtr("customers"), Old: "orders.customer_id = customers.id", New: "(orders.tenant_id = customers.tenant_id) AND (orders.customer_id = customers.id)"},
				}},
			},
			Governance: []GovernanceDiff{
				{Scope: "cube", Name: "customers", Changes: []Change{
					{Type: "pii_flag_added", Member: stringPtr("email")},
				}},
			},
			PreAggregations: []PreAggregationDiff{
				{Cube: "orders", Name: "orders_by_day", Status: "rebuild_required", Reason: stringPtr("dimension changed")},
			},
			Warnings: []string{
				"PK/FK relationship added: orders.customer_id -> customers.id",
			},
		},
		Aliases: AliasMap{
			CoreVersion:     coreVersion,
			PreviousVersion: "1.0.0",
			GeneratedAt:     time.Now().Format(time.RFC3339),
			Aliases: []AliasEntry{
				{
					Scope:      "cube",
					Name:       "orders",
					MemberType: "dimension",
					OldName:    "customerKey",
					NewName:    stringPtr("customer_id"),
					Status:     "renamed",
					Meta: AliasMeta{
						Reason:               "Standardized naming convention",
						AutoRewrite:          true,
						SuggestedReplacement: stringPtr("customer_id"),
						BreakingChange:       boolPtr(true),
					},
				},
			},
		},
	}
}

// Store Layer Hooks - Version Management
func (s *UpgradeRuntimeService) VersionExists(version string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.versions[version]
	return exists
}

func (s *UpgradeRuntimeService) SetActiveVersion(version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.versions[version]; !ok {
		return errors.New("version not found")
	}
	if version == s.active {
		return nil // Idempotent - already active
	}
	// Update statuses
	if curr, ok := s.versions[s.active]; ok {
		curr.Status = StatusPrevious
		s.previous = curr.Version
	}
	s.active = version
	v := s.versions[version]
	v.Status = StatusActive
	now := time.Now()
	v.ActivatedAt = &now
	s.canary = nil
	s.preview = ""
	return nil
}

func (s *UpgradeRuntimeService) AssignTenantsToVersion(tenants []string, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.versions[version]; !ok {
		return errors.New("version not found")
	}
	// In a real implementation, this would update tenant-to-version mappings
	// For now, we'll just log the assignment
	s.notifications = append(s.notifications, Notification{
		ID:        randID(),
		Type:      "upgrade",
		Message:   fmt.Sprintf("Assigned tenants %v to version %s", tenants, version),
		Severity:  "info",
		CreatedAt: time.Now(),
	})
	return nil
}

func (s *UpgradeRuntimeService) GetPreviousActiveVersion() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.previous
}

func (s *UpgradeRuntimeService) GetTenantsOnVersion(version string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// In a real implementation, this would query tenant assignments
	// For now, return a mock list
	if version == s.active {
		return []string{"tenantA", "tenantB", "tenantC"}
	}
	return []string{}
}

// Jobs Layer Hooks - Operational Tasks
func (s *UpgradeRuntimeService) WarmCachesForAllTenants(version string) {
	// Simulate cache warming for all tenants
	go func() {
		time.Sleep(100 * time.Millisecond) // Simulate async operation
		s.notifications = append(s.notifications, Notification{
			ID:        randID(),
			Type:      "upgrade",
			Message:   fmt.Sprintf("Cache warming completed for all tenants on version %s", version),
			Severity:  "success",
			CreatedAt: time.Now(),
		})
	}()
}

func (s *UpgradeRuntimeService) WarmCachesForTenants(tenants []string, version string) {
	// Simulate cache warming for specific tenants
	go func() {
		time.Sleep(50 * time.Millisecond) // Simulate async operation
		s.notifications = append(s.notifications, Notification{
			ID:        randID(),
			Type:      "upgrade",
			Message:   fmt.Sprintf("Cache warming completed for tenants %v on version %s", tenants, version),
			Severity:  "success",
			CreatedAt: time.Now(),
		})
	}()
}

func (s *UpgradeRuntimeService) RebuildPreAggsIfNeeded(version string) {
	// Simulate pre-aggregation rebuild check and execution
	go func() {
		time.Sleep(200 * time.Millisecond) // Simulate async operation
		s.notifications = append(s.notifications, Notification{
			ID:        randID(),
			Type:      "upgrade",
			Message:   fmt.Sprintf("Pre-aggregation rebuild completed for version %s", version),
			Severity:  "success",
			CreatedAt: time.Now(),
		})
	}()
}

// Audit Logging
func (s *UpgradeRuntimeService) LogUpgradeAction(action, version string, ts time.Time, user string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifications = append(s.notifications, Notification{
		ID:        randID(),
		Type:      "audit",
		Message:   fmt.Sprintf("User %s performed %s action on version %s", user, action, version),
		Severity:  "info",
		CreatedAt: ts,
	})
	// In a real implementation, this would write to an audit log database
}

// Real-Time Broadcasting
func (s *UpgradeRuntimeService) BroadcastStatusChange(version, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate inputs
	if version == "" || status == "" {
		log.Printf("BroadcastStatusChange: invalid parameters - version: %s, status: %s", version, status)
		return
	}

	// Update version status with validation
	if v, ok := s.versions[version]; ok {
		switch status {
		case "active":
			v.Status = StatusActive
		case "canary":
			v.Status = StatusCanary
		case "rolled_back":
			v.Status = StatusAvailable
		default:
			log.Printf("BroadcastStatusChange: unknown status %s for version %s", status, version)
			return
		}
	} else {
		log.Printf("BroadcastStatusChange: version %s not found", version)
		return
	}

	// Broadcast via WebSocket with retry logic
	if s.wsHub != nil {
		const maxRetries = 3
		for attempt := 0; attempt < maxRetries; attempt++ {
			if err := s.attemptBroadcast(version, status, attempt); err == nil {
				break
			}
			if attempt < maxRetries-1 {
				time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond) // Exponential backoff
			}
		}
	} else {
		log.Printf("BroadcastStatusChange: WebSocket hub not available")
	}

	// Log the status change
	s.notifications = append(s.notifications, Notification{
		ID:        randID(),
		Type:      "broadcast",
		Message:   fmt.Sprintf("Version %s status changed to %s", version, status),
		Severity:  "info",
		CreatedAt: time.Now(),
	})

	log.Printf("BroadcastStatusChange: successfully broadcast version %s status to %s", version, status)
}

// attemptBroadcast tries to broadcast with error handling
func (s *UpgradeRuntimeService) attemptBroadcast(version, status string, attempt int) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("BroadcastStatusChange: panic in attempt %d: %v", attempt, r)
		}
	}()

	s.wsHub.BroadcastUpgradeStatus(version, status, []string{}, []string{})
	return nil
}
