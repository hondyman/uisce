package cube

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// SecurityService provides high-performance RBAC/ABAC security for Cube.js queries.
// It uses multi-tier caching to minimize performance impact on analytical workloads.
type SecurityService struct {
	db *sqlx.DB

	// In-memory L1 cache for hot paths (< 1ms latency)
	l1Cache     map[string]*CachedSecurityDecision
	l1CacheLock sync.RWMutex
	l1TTL       time.Duration

	// Background workers
	cacheRefreshCh chan string
	stopCh         chan struct{}
}

// CachedSecurityDecision represents a cached ABAC decision for fast lookup
type CachedSecurityDecision struct {
	Decision      SecurityDecision `json:"decision"`
	ComputedAt    time.Time        `json:"computed_at"`
	ExpiresAt     time.Time        `json:"expires_at"`
	HitCount      int64            `json:"hit_count"`
	PolicyVersion string           `json:"policy_version"`
}

// SecurityDecision represents the outcome of an ABAC policy evaluation
type SecurityDecision struct {
	Allowed         bool                   `json:"allowed"`
	RowFilters      []RowFilter            `json:"row_filters,omitempty"`
	ColumnMasks     []ColumnMask           `json:"column_masks,omitempty"`
	QueryLimits     *QueryLimits           `json:"query_limits,omitempty"`
	AuditMetadata   map[string]interface{} `json:"audit_metadata,omitempty"`
	AppliedPolicies []string               `json:"applied_policies,omitempty"`
	DenialReason    string                 `json:"denial_reason,omitempty"`
}

// RowFilter defines a filter to be applied to query results (row-level security)
type RowFilter struct {
	Cube       string        `json:"cube"`
	Dimension  string        `json:"dimension"`
	Operator   string        `json:"operator"` // equals, notEquals, in, notIn, contains, gt, lt, gte, lte
	Values     []interface{} `json:"values"`
	Dynamic    bool          `json:"dynamic"`              // If true, values are computed from security context
	Expression string        `json:"expression,omitempty"` // Raw SQL expression for complex filters
}

// ColumnMask defines masking rules for sensitive columns (column-level security)
type ColumnMask struct {
	Cube         string   `json:"cube"`
	Member       string   `json:"member"`                  // dimension or measure name
	MaskType     string   `json:"mask_type"`               // redact, hash, truncate, nullify, partial, custom
	MaskPattern  string   `json:"mask_pattern,omitempty"`  // For partial masking, e.g., "XXX-XX-{last4}"
	AllowedRoles []string `json:"allowed_roles,omitempty"` // Roles that see unmasked data
}

// QueryLimits defines resource limits for queries based on user tier
type QueryLimits struct {
	MaxRows           int           `json:"max_rows,omitempty"`
	MaxExecutionTime  time.Duration `json:"max_execution_time,omitempty"`
	MaxConcurrency    int           `json:"max_concurrency,omitempty"`
	AllowedCubes      []string      `json:"allowed_cubes,omitempty"`
	DeniedCubes       []string      `json:"denied_cubes,omitempty"`
	AllowedMeasures   []string      `json:"allowed_measures,omitempty"`
	DeniedMeasures    []string      `json:"denied_measures,omitempty"`
	AllowedDimensions []string      `json:"allowed_dimensions,omitempty"`
	DeniedDimensions  []string      `json:"denied_dimensions,omitempty"`
	AllowPreAgg       bool          `json:"allow_pre_agg"`
	PreAggOnly        bool          `json:"pre_agg_only"` // Force queries through pre-aggregations only
}

// SecurityContext represents the user's security attributes for ABAC evaluation
type SecurityContext struct {
	UserID           string                 `json:"user_id"`
	TenantID         string                 `json:"tenant_id"`
	DatasourceID     string                 `json:"datasource_id"`
	Roles            []string               `json:"roles"`
	Groups           []string               `json:"groups"`
	Attributes       map[string]interface{} `json:"attributes"`
	SessionID        string                 `json:"session_id"`
	IPAddress        string                 `json:"ip_address,omitempty"`
	RequestTimestamp time.Time              `json:"request_timestamp"`
}

// ABACPolicy represents a stored ABAC policy
type ABACPolicy struct {
	ID            uuid.UUID        `db:"id" json:"id"`
	TenantID      uuid.UUID        `db:"tenant_id" json:"tenant_id"`
	Name          string           `db:"name" json:"name"`
	Description   string           `db:"description" json:"description"`
	PolicyType    string           `db:"policy_type" json:"policy_type"` // row, column, query, access
	Priority      int              `db:"priority" json:"priority"`
	Enabled       bool             `db:"enabled" json:"enabled"`
	TargetCubes   []string         `db:"target_cubes" json:"target_cubes"`
	TargetMembers []string         `db:"target_members" json:"target_members"`
	Conditions    PolicyConditions `db:"conditions" json:"conditions"`
	Effects       PolicyEffects    `db:"effects" json:"effects"`
	Version       int              `db:"version" json:"version"`
	CreatedAt     time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time        `db:"updated_at" json:"updated_at"`
	CreatedBy     uuid.UUID        `db:"created_by" json:"created_by"`
}

// PolicyConditions defines when a policy applies
type PolicyConditions struct {
	Roles              []string               `json:"roles,omitempty"`               // User must have one of these roles
	Groups             []string               `json:"groups,omitempty"`              // User must belong to one of these groups
	Attributes         map[string]interface{} `json:"attributes,omitempty"`          // Attribute-based conditions
	TimeWindow         *TimeWindowCondition   `json:"time_window,omitempty"`         // Time-based access control
	IPRanges           []string               `json:"ip_ranges,omitempty"`           // IP-based restrictions
	DataClassification []string               `json:"data_classification,omitempty"` // Data sensitivity levels
}

// TimeWindowCondition defines time-based access rules
type TimeWindowCondition struct {
	AllowedDays       []string   `json:"allowed_days,omitempty"`        // ["Monday", "Tuesday", ...]
	AllowedHoursStart int        `json:"allowed_hours_start,omitempty"` // 0-23
	AllowedHoursEnd   int        `json:"allowed_hours_end,omitempty"`   // 0-23
	Timezone          string     `json:"timezone,omitempty"`            // IANA timezone
	EffectiveFrom     *time.Time `json:"effective_from,omitempty"`
	EffectiveUntil    *time.Time `json:"effective_until,omitempty"`
}

// PolicyEffects defines what happens when a policy matches
type PolicyEffects struct {
	Action       string       `json:"action"` // allow, deny, filter, mask, limit
	RowFilters   []RowFilter  `json:"row_filters,omitempty"`
	ColumnMasks  []ColumnMask `json:"column_masks,omitempty"`
	QueryLimits  *QueryLimits `json:"query_limits,omitempty"`
	AuditLog     bool         `json:"audit_log"`
	AlertOnMatch bool         `json:"alert_on_match"`
}

// NewSecurityService creates a new security service with caching
func NewSecurityService(db *sqlx.DB) *SecurityService {
	svc := &SecurityService{
		db:             db,
		l1Cache:        make(map[string]*CachedSecurityDecision),
		l1TTL:          5 * time.Minute, // L1 cache TTL
		cacheRefreshCh: make(chan string, 1000),
		stopCh:         make(chan struct{}),
	}

	// Start background cache maintenance
	go svc.cacheMaintenanceWorker()

	return svc
}

// Stop gracefully shuts down the security service
func (s *SecurityService) Stop() {
	close(s.stopCh)
}

// EvaluateSecurity evaluates all applicable ABAC policies for a security context.
// This is the main entry point for Cube.js query security.
func (s *SecurityService) EvaluateSecurity(ctx context.Context, secCtx SecurityContext, cubes []string) (*SecurityDecision, error) {
	// Generate cache key
	cacheKey := s.generateCacheKey(secCtx, cubes)

	// Check L1 in-memory cache first (fastest path)
	if decision, found := s.checkL1Cache(cacheKey); found {
		return &decision.Decision, nil
	}

	// Check L2 database cache
	if decision, found := s.checkL2Cache(ctx, cacheKey); found {
		// Promote to L1 cache
		s.setL1Cache(cacheKey, decision)
		return &decision.Decision, nil
	}

	// Full policy evaluation (slowest path)
	decision, err := s.evaluatePolicies(ctx, secCtx, cubes)
	if err != nil {
		return nil, err
	}

	// Cache the decision
	cached := &CachedSecurityDecision{
		Decision:      *decision,
		ComputedAt:    time.Now(),
		ExpiresAt:     time.Now().Add(s.l1TTL),
		HitCount:      0,
		PolicyVersion: s.getPolicyVersion(ctx, secCtx.TenantID),
	}

	s.setL1Cache(cacheKey, cached)
	s.setL2Cache(ctx, cacheKey, cached, secCtx.TenantID)

	return decision, nil
}

// GenerateCubeQueryRewrite generates SQL WHERE clauses for Cube.js queryRewrite.
// This is optimized for direct injection into Cube query contexts.
func (s *SecurityService) GenerateCubeQueryRewrite(ctx context.Context, secCtx SecurityContext, cube string) (string, error) {
	decision, err := s.EvaluateSecurity(ctx, secCtx, []string{cube})
	if err != nil {
		return "", fmt.Errorf("security evaluation failed: %w", err)
	}

	if !decision.Allowed {
		return "1=0", nil // Block all rows
	}

	var filters []string
	for _, rf := range decision.RowFilters {
		if rf.Cube != "" && rf.Cube != cube {
			continue
		}

		filter := s.rowFilterToSQL(rf, secCtx)
		if filter != "" {
			filters = append(filters, filter)
		}
	}

	if len(filters) == 0 {
		return "1=1", nil // Allow all rows
	}

	// Combine with AND (all filters must pass)
	result := ""
	for i, f := range filters {
		if i > 0 {
			result += " AND "
		}
		result += "(" + f + ")"
	}

	return result, nil
}

// GenerateCubeSecurityContext generates a security context object for Cube.js.
// This is used in the securityContext callback in cube.js configuration.
func (s *SecurityService) GenerateCubeSecurityContext(ctx context.Context, secCtx SecurityContext) (map[string]interface{}, error) {
	cubes := []string{} // Evaluate for all cubes
	decision, err := s.EvaluateSecurity(ctx, secCtx, cubes)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"userId":       secCtx.UserID,
		"tenantId":     secCtx.TenantID,
		"datasourceId": secCtx.DatasourceID,
		"roles":        secCtx.Roles,
		"groups":       secCtx.Groups,
		"allowed":      decision.Allowed,
	}

	// Add row filters for queryRewrite
	if len(decision.RowFilters) > 0 {
		filters := make(map[string][]RowFilter)
		for _, rf := range decision.RowFilters {
			cube := rf.Cube
			if cube == "" {
				cube = "*"
			}
			filters[cube] = append(filters[cube], rf)
		}
		result["rowFilters"] = filters
	}

	// Add column masks for result transformation
	if len(decision.ColumnMasks) > 0 {
		masks := make(map[string][]ColumnMask)
		for _, cm := range decision.ColumnMasks {
			cube := cm.Cube
			if cube == "" {
				cube = "*"
			}
			masks[cube] = append(masks[cube], cm)
		}
		result["columnMasks"] = masks
	}

	// Add query limits
	if decision.QueryLimits != nil {
		result["queryLimits"] = decision.QueryLimits
	}

	// Add policy audit trail
	if len(decision.AppliedPolicies) > 0 {
		result["appliedPolicies"] = decision.AppliedPolicies
	}

	return result, nil
}

// --- Policy Evaluation ---

func (s *SecurityService) evaluatePolicies(ctx context.Context, secCtx SecurityContext, cubes []string) (*SecurityDecision, error) {
	// Fetch applicable policies
	policies, err := s.getApplicablePolicies(ctx, secCtx.TenantID, cubes)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch policies: %w", err)
	}

	decision := &SecurityDecision{
		Allowed:         true,
		RowFilters:      []RowFilter{},
		ColumnMasks:     []ColumnMask{},
		AuditMetadata:   make(map[string]interface{}),
		AppliedPolicies: []string{},
	}

	// Evaluate policies in priority order
	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}

		matches, err := s.evaluatePolicyConditions(policy, secCtx)
		if err != nil {
			// Log but don't fail - deny by default on evaluation errors
			decision.Allowed = false
			decision.DenialReason = fmt.Sprintf("Policy evaluation error: %v", err)
			return decision, nil
		}

		if !matches {
			continue
		}

		// Policy matched - apply effects
		decision.AppliedPolicies = append(decision.AppliedPolicies, policy.Name)

		switch policy.Effects.Action {
		case "deny":
			decision.Allowed = false
			decision.DenialReason = fmt.Sprintf("Denied by policy: %s", policy.Name)
			return decision, nil // Deny is final

		case "filter":
			decision.RowFilters = append(decision.RowFilters, policy.Effects.RowFilters...)

		case "mask":
			decision.ColumnMasks = append(decision.ColumnMasks, policy.Effects.ColumnMasks...)

		case "limit":
			if policy.Effects.QueryLimits != nil {
				decision.QueryLimits = s.mergeQueryLimits(decision.QueryLimits, policy.Effects.QueryLimits)
			}

		case "allow":
			// Explicit allow - continue to next policy
		}

		// Audit logging
		if policy.Effects.AuditLog {
			decision.AuditMetadata[policy.Name] = map[string]interface{}{
				"matched_at": time.Now(),
				"user_id":    secCtx.UserID,
				"action":     policy.Effects.Action,
			}
		}
	}

	return decision, nil
}

func (s *SecurityService) evaluatePolicyConditions(policy ABACPolicy, secCtx SecurityContext) (bool, error) {
	conditions := policy.Conditions

	// Check role-based conditions
	if len(conditions.Roles) > 0 {
		hasRole := false
		for _, requiredRole := range conditions.Roles {
			for _, userRole := range secCtx.Roles {
				if requiredRole == userRole {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}
		if !hasRole {
			return false, nil
		}
	}

	// Check group-based conditions
	if len(conditions.Groups) > 0 {
		hasGroup := false
		for _, requiredGroup := range conditions.Groups {
			for _, userGroup := range secCtx.Groups {
				if requiredGroup == userGroup {
					hasGroup = true
					break
				}
			}
			if hasGroup {
				break
			}
		}
		if !hasGroup {
			return false, nil
		}
	}

	// Check attribute-based conditions
	for attrKey, attrValue := range conditions.Attributes {
		userValue, exists := secCtx.Attributes[attrKey]
		if !exists {
			return false, nil
		}
		if !s.compareAttributeValues(userValue, attrValue) {
			return false, nil
		}
	}

	// Check time window conditions
	if conditions.TimeWindow != nil {
		if !s.evaluateTimeWindow(conditions.TimeWindow, secCtx.RequestTimestamp) {
			return false, nil
		}
	}

	// Check IP range conditions
	if len(conditions.IPRanges) > 0 && secCtx.IPAddress != "" {
		if !s.checkIPInRanges(secCtx.IPAddress, conditions.IPRanges) {
			return false, nil
		}
	}

	return true, nil
}

func (s *SecurityService) evaluateTimeWindow(tw *TimeWindowCondition, reqTime time.Time) bool {
	// Check effective dates
	if tw.EffectiveFrom != nil && reqTime.Before(*tw.EffectiveFrom) {
		return false
	}
	if tw.EffectiveUntil != nil && reqTime.After(*tw.EffectiveUntil) {
		return false
	}

	// Apply timezone
	loc := time.UTC
	if tw.Timezone != "" {
		var err error
		loc, err = time.LoadLocation(tw.Timezone)
		if err != nil {
			loc = time.UTC
		}
	}
	reqTimeLocal := reqTime.In(loc)

	// Check allowed days
	if len(tw.AllowedDays) > 0 {
		dayName := reqTimeLocal.Weekday().String()
		dayAllowed := false
		for _, d := range tw.AllowedDays {
			if d == dayName {
				dayAllowed = true
				break
			}
		}
		if !dayAllowed {
			return false
		}
	}

	// Check allowed hours
	hour := reqTimeLocal.Hour()
	if tw.AllowedHoursStart != 0 || tw.AllowedHoursEnd != 0 {
		if tw.AllowedHoursStart <= tw.AllowedHoursEnd {
			// Normal range (e.g., 9-17)
			if hour < tw.AllowedHoursStart || hour > tw.AllowedHoursEnd {
				return false
			}
		} else {
			// Overnight range (e.g., 22-6)
			if hour < tw.AllowedHoursStart && hour > tw.AllowedHoursEnd {
				return false
			}
		}
	}

	return true
}

func (s *SecurityService) compareAttributeValues(userValue, policyValue interface{}) bool {
	// Handle slice comparisons
	switch pv := policyValue.(type) {
	case []interface{}:
		// User value must match one of the policy values
		for _, v := range pv {
			if fmt.Sprintf("%v", userValue) == fmt.Sprintf("%v", v) {
				return true
			}
		}
		return false
	case map[string]interface{}:
		// Complex comparison (operators)
		if op, ok := pv["$gt"]; ok {
			return s.compareNumeric(userValue, op, ">")
		}
		if op, ok := pv["$gte"]; ok {
			return s.compareNumeric(userValue, op, ">=")
		}
		if op, ok := pv["$lt"]; ok {
			return s.compareNumeric(userValue, op, "<")
		}
		if op, ok := pv["$lte"]; ok {
			return s.compareNumeric(userValue, op, "<=")
		}
		if op, ok := pv["$ne"]; ok {
			return fmt.Sprintf("%v", userValue) != fmt.Sprintf("%v", op)
		}
		if op, ok := pv["$in"]; ok {
			if arr, ok := op.([]interface{}); ok {
				for _, v := range arr {
					if fmt.Sprintf("%v", userValue) == fmt.Sprintf("%v", v) {
						return true
					}
				}
			}
			return false
		}
		return false
	default:
		// Simple equality
		return fmt.Sprintf("%v", userValue) == fmt.Sprintf("%v", policyValue)
	}
}

func (s *SecurityService) compareNumeric(userValue, policyValue interface{}, op string) bool {
	uv, err := s.toFloat(userValue)
	if err != nil {
		return false
	}
	pv, err := s.toFloat(policyValue)
	if err != nil {
		return false
	}

	switch op {
	case ">":
		return uv > pv
	case ">=":
		return uv >= pv
	case "<":
		return uv < pv
	case "<=":
		return uv <= pv
	default:
		return false
	}
}

func (s *SecurityService) toFloat(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case int32:
		return float64(val), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float", v)
	}
}

func (s *SecurityService) checkIPInRanges(ip string, ranges []string) bool {
	// TODO: Implement proper CIDR range checking
	for _, r := range ranges {
		if r == ip || r == "0.0.0.0/0" {
			return true
		}
	}
	return false
}

func (s *SecurityService) rowFilterToSQL(rf RowFilter, secCtx SecurityContext) string {
	dim := rf.Dimension

	// Handle dynamic values from security context
	values := rf.Values
	if rf.Dynamic {
		values = s.resolveDynamicValues(rf.Values, secCtx)
	}

	if rf.Expression != "" {
		// Raw SQL expression (use with caution)
		return rf.Expression
	}

	switch rf.Operator {
	case "equals", "eq", "=":
		if len(values) == 0 {
			return "1=0"
		}
		return fmt.Sprintf("%s = %s", dim, s.sqlValue(values[0]))

	case "notEquals", "ne", "!=":
		if len(values) == 0 {
			return "1=1"
		}
		return fmt.Sprintf("%s != %s", dim, s.sqlValue(values[0]))

	case "in":
		if len(values) == 0 {
			return "1=0"
		}
		return fmt.Sprintf("%s IN (%s)", dim, s.sqlValueList(values))

	case "notIn":
		if len(values) == 0 {
			return "1=1"
		}
		return fmt.Sprintf("%s NOT IN (%s)", dim, s.sqlValueList(values))

	case "contains":
		if len(values) == 0 {
			return "1=0"
		}
		return fmt.Sprintf("%s LIKE '%%%s%%'", dim, s.escapeSQL(fmt.Sprintf("%v", values[0])))

	case "gt", ">":
		return fmt.Sprintf("%s > %s", dim, s.sqlValue(values[0]))

	case "gte", ">=":
		return fmt.Sprintf("%s >= %s", dim, s.sqlValue(values[0]))

	case "lt", "<":
		return fmt.Sprintf("%s < %s", dim, s.sqlValue(values[0]))

	case "lte", "<=":
		return fmt.Sprintf("%s <= %s", dim, s.sqlValue(values[0]))

	case "isNull":
		return fmt.Sprintf("%s IS NULL", dim)

	case "isNotNull":
		return fmt.Sprintf("%s IS NOT NULL", dim)

	default:
		return ""
	}
}

func (s *SecurityService) resolveDynamicValues(templates []interface{}, secCtx SecurityContext) []interface{} {
	result := make([]interface{}, 0, len(templates))
	for _, t := range templates {
		tmpl := fmt.Sprintf("%v", t)
		switch tmpl {
		case "${tenant_id}", "$tenant_id":
			result = append(result, secCtx.TenantID)
		case "${user_id}", "$user_id":
			result = append(result, secCtx.UserID)
		case "${datasource_id}", "$datasource_id":
			result = append(result, secCtx.DatasourceID)
		default:
			// Check attributes
			if len(tmpl) > 2 && tmpl[0:2] == "${" && tmpl[len(tmpl)-1] == '}' {
				attrName := tmpl[2 : len(tmpl)-1]
				if val, ok := secCtx.Attributes[attrName]; ok {
					result = append(result, val)
				}
			} else {
				result = append(result, t)
			}
		}
	}
	return result
}

func (s *SecurityService) sqlValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return "'" + s.escapeSQL(val) + "'"
	case int, int64, int32, float64, float32:
		return fmt.Sprintf("%v", val)
	case bool:
		if val {
			return "TRUE"
		}
		return "FALSE"
	case nil:
		return "NULL"
	default:
		return "'" + s.escapeSQL(fmt.Sprintf("%v", val)) + "'"
	}
}

func (s *SecurityService) sqlValueList(values []interface{}) string {
	result := ""
	for i, v := range values {
		if i > 0 {
			result += ", "
		}
		result += s.sqlValue(v)
	}
	return result
}

func (s *SecurityService) escapeSQL(s_str string) string {
	// Basic SQL escape - replace single quotes
	result := ""
	for _, c := range s_str {
		if c == '\'' {
			result += "''"
		} else {
			result += string(c)
		}
	}
	return result
}

func (s *SecurityService) mergeQueryLimits(existing, new *QueryLimits) *QueryLimits {
	if existing == nil {
		return new
	}
	if new == nil {
		return existing
	}

	result := *existing

	// Take the more restrictive limits
	if new.MaxRows > 0 && (existing.MaxRows == 0 || new.MaxRows < existing.MaxRows) {
		result.MaxRows = new.MaxRows
	}
	if new.MaxExecutionTime > 0 && (existing.MaxExecutionTime == 0 || new.MaxExecutionTime < existing.MaxExecutionTime) {
		result.MaxExecutionTime = new.MaxExecutionTime
	}
	if new.MaxConcurrency > 0 && (existing.MaxConcurrency == 0 || new.MaxConcurrency < existing.MaxConcurrency) {
		result.MaxConcurrency = new.MaxConcurrency
	}

	// Merge allowed/denied lists
	if len(new.AllowedCubes) > 0 {
		result.AllowedCubes = s.intersectStrings(existing.AllowedCubes, new.AllowedCubes)
	}
	if len(new.DeniedCubes) > 0 {
		result.DeniedCubes = append(result.DeniedCubes, new.DeniedCubes...)
	}

	return &result
}

func (s *SecurityService) intersectStrings(a, b []string) []string {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}

	bSet := make(map[string]bool)
	for _, v := range b {
		bSet[v] = true
	}

	result := []string{}
	for _, v := range a {
		if bSet[v] {
			result = append(result, v)
		}
	}
	return result
}

// --- Caching ---

func (s *SecurityService) generateCacheKey(secCtx SecurityContext, cubes []string) string {
	keyData := map[string]interface{}{
		"tenant":     secCtx.TenantID,
		"datasource": secCtx.DatasourceID,
		"user":       secCtx.UserID,
		"roles":      secCtx.Roles,
		"groups":     secCtx.Groups,
		"cubes":      cubes,
	}

	data, _ := json.Marshal(keyData)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (s *SecurityService) checkL1Cache(key string) (*CachedSecurityDecision, bool) {
	s.l1CacheLock.RLock()
	defer s.l1CacheLock.RUnlock()

	if cached, ok := s.l1Cache[key]; ok {
		if time.Now().Before(cached.ExpiresAt) {
			cached.HitCount++
			return cached, true
		}
	}
	return nil, false
}

func (s *SecurityService) setL1Cache(key string, decision *CachedSecurityDecision) {
	s.l1CacheLock.Lock()
	defer s.l1CacheLock.Unlock()

	s.l1Cache[key] = decision

	// Prune if cache is too large
	if len(s.l1Cache) > 10000 {
		s.pruneL1Cache()
	}
}

func (s *SecurityService) pruneL1Cache() {
	// Remove expired entries first
	now := time.Now()
	for k, v := range s.l1Cache {
		if now.After(v.ExpiresAt) {
			delete(s.l1Cache, k)
		}
	}

	// If still too large, remove LRU entries
	if len(s.l1Cache) > 8000 {
		type entry struct {
			key  string
			hits int64
		}
		entries := make([]entry, 0, len(s.l1Cache))
		for k, v := range s.l1Cache {
			entries = append(entries, entry{k, v.HitCount})
		}
		// Sort by hit count (keep high-hit entries)
		for i := 0; i < len(entries)-1; i++ {
			for j := i + 1; j < len(entries); j++ {
				if entries[i].hits > entries[j].hits {
					entries[i], entries[j] = entries[j], entries[i]
				}
			}
		}
		// Remove bottom 20%
		removeCount := len(entries) / 5
		for i := 0; i < removeCount; i++ {
			delete(s.l1Cache, entries[i].key)
		}
	}
}

func (s *SecurityService) checkL2Cache(ctx context.Context, key string) (*CachedSecurityDecision, bool) {
	var cached struct {
		Decision      json.RawMessage `db:"decision"`
		ComputedAt    time.Time       `db:"computed_at"`
		ExpiresAt     time.Time       `db:"expires_at"`
		HitCount      int64           `db:"hit_count"`
		PolicyVersion string          `db:"policy_version"`
	}

	err := s.db.GetContext(ctx, &cached, `
		SELECT decision, computed_at, expires_at, hit_count, policy_version
		FROM cube_security_cache
		WHERE cache_key = $1 AND expires_at > NOW()
	`, key)

	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		return nil, false
	}

	var decision SecurityDecision
	if err := json.Unmarshal(cached.Decision, &decision); err != nil {
		return nil, false
	}

	// Update hit count asynchronously
	go func() {
		s.db.ExecContext(context.Background(), `
			UPDATE cube_security_cache SET hit_count = hit_count + 1 WHERE cache_key = $1
		`, key)
	}()

	return &CachedSecurityDecision{
		Decision:      decision,
		ComputedAt:    cached.ComputedAt,
		ExpiresAt:     cached.ExpiresAt,
		HitCount:      cached.HitCount,
		PolicyVersion: cached.PolicyVersion,
	}, true
}

func (s *SecurityService) setL2Cache(ctx context.Context, key string, decision *CachedSecurityDecision, tenantID string) {
	decisionJSON, err := json.Marshal(decision.Decision)
	if err != nil {
		return
	}

	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO cube_security_cache (cache_key, tenant_id, decision, computed_at, expires_at, policy_version)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (cache_key) DO UPDATE SET
			decision = EXCLUDED.decision,
			computed_at = EXCLUDED.computed_at,
			expires_at = EXCLUDED.expires_at,
			policy_version = EXCLUDED.policy_version,
			hit_count = 0
	`, key, tenantID, decisionJSON, decision.ComputedAt, decision.ExpiresAt, decision.PolicyVersion)
}

func (s *SecurityService) getPolicyVersion(ctx context.Context, tenantID string) string {
	var version string
	err := s.db.GetContext(ctx, &version, `
		SELECT COALESCE(MAX(updated_at)::text, 'initial')
		FROM cube_model_security_policies
		WHERE tenant_id = $1
	`, tenantID)
	if err != nil {
		return "unknown"
	}
	return version
}

// InvalidateCache invalidates security cache for a tenant when policies change
func (s *SecurityService) InvalidateCache(ctx context.Context, tenantID string) error {
	// Clear L2 cache
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM cube_security_cache WHERE tenant_id = $1
	`, tenantID)
	if err != nil {
		return fmt.Errorf("failed to invalidate L2 cache: %w", err)
	}

	// Clear L1 cache (all entries - could be smarter with tenant prefixing)
	s.l1CacheLock.Lock()
	s.l1Cache = make(map[string]*CachedSecurityDecision)
	s.l1CacheLock.Unlock()

	return nil
}

func (s *SecurityService) cacheMaintenanceWorker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.cleanupExpiredCache()
		}
	}
}

func (s *SecurityService) cleanupExpiredCache() {
	// Clean L1
	s.l1CacheLock.Lock()
	now := time.Now()
	for k, v := range s.l1Cache {
		if now.After(v.ExpiresAt) {
			delete(s.l1Cache, k)
		}
	}
	s.l1CacheLock.Unlock()

	// Clean L2
	ctx := context.Background()
	_, _ = s.db.ExecContext(ctx, `DELETE FROM cube_security_cache WHERE expires_at < NOW()`)
}

// --- Policy CRUD ---

func (s *SecurityService) getApplicablePolicies(ctx context.Context, tenantID string, cubes []string) ([]ABACPolicy, error) {
	query := `
		SELECT id, tenant_id, name, description, policy_type, priority, enabled,
		       target_cubes, target_members, conditions, effects, version, created_at, updated_at, created_by
		FROM cube_model_security_policies
		WHERE tenant_id = $1 AND enabled = true
		ORDER BY priority DESC, created_at ASC
	`

	var rows []struct {
		ID            uuid.UUID       `db:"id"`
		TenantID      uuid.UUID       `db:"tenant_id"`
		Name          string          `db:"name"`
		Description   sql.NullString  `db:"description"`
		PolicyType    string          `db:"policy_type"`
		Priority      int             `db:"priority"`
		Enabled       bool            `db:"enabled"`
		TargetCubes   json.RawMessage `db:"target_cubes"`
		TargetMembers json.RawMessage `db:"target_members"`
		Conditions    json.RawMessage `db:"conditions"`
		Effects       json.RawMessage `db:"effects"`
		Version       int             `db:"version"`
		CreatedAt     time.Time       `db:"created_at"`
		UpdatedAt     time.Time       `db:"updated_at"`
		CreatedBy     uuid.UUID       `db:"created_by"`
	}

	if err := s.db.SelectContext(ctx, &rows, query, tenantID); err != nil {
		return nil, fmt.Errorf("failed to fetch policies: %w", err)
	}

	policies := make([]ABACPolicy, 0, len(rows))
	for _, row := range rows {
		policy := ABACPolicy{
			ID:          row.ID,
			TenantID:    row.TenantID,
			Name:        row.Name,
			Description: row.Description.String,
			PolicyType:  row.PolicyType,
			Priority:    row.Priority,
			Enabled:     row.Enabled,
			Version:     row.Version,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
			CreatedBy:   row.CreatedBy,
		}

		// Parse JSON fields
		json.Unmarshal(row.TargetCubes, &policy.TargetCubes)
		json.Unmarshal(row.TargetMembers, &policy.TargetMembers)
		json.Unmarshal(row.Conditions, &policy.Conditions)
		json.Unmarshal(row.Effects, &policy.Effects)

		// Filter by target cubes if specified
		if len(cubes) > 0 && len(policy.TargetCubes) > 0 {
			matches := false
			for _, tc := range policy.TargetCubes {
				for _, c := range cubes {
					if tc == c || tc == "*" {
						matches = true
						break
					}
				}
				if matches {
					break
				}
			}
			if !matches {
				continue
			}
		}

		policies = append(policies, policy)
	}

	return policies, nil
}

// CreatePolicy creates a new ABAC policy
func (s *SecurityService) CreatePolicy(ctx context.Context, policy *ABACPolicy) error {
	policy.ID = uuid.New()
	policy.Version = 1
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()

	targetCubesJSON, _ := json.Marshal(policy.TargetCubes)
	targetMembersJSON, _ := json.Marshal(policy.TargetMembers)
	conditionsJSON, _ := json.Marshal(policy.Conditions)
	effectsJSON, _ := json.Marshal(policy.Effects)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO cube_model_security_policies 
		(id, tenant_id, name, description, policy_type, priority, enabled, target_cubes, target_members, conditions, effects, version, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, policy.ID, policy.TenantID, policy.Name, policy.Description, policy.PolicyType, policy.Priority,
		policy.Enabled, targetCubesJSON, targetMembersJSON, conditionsJSON, effectsJSON,
		policy.Version, policy.CreatedAt, policy.UpdatedAt, policy.CreatedBy)

	if err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}

	// Invalidate cache
	return s.InvalidateCache(ctx, policy.TenantID.String())
}

// UpdatePolicy updates an existing ABAC policy
func (s *SecurityService) UpdatePolicy(ctx context.Context, policy *ABACPolicy) error {
	policy.Version++
	policy.UpdatedAt = time.Now()

	targetCubesJSON, _ := json.Marshal(policy.TargetCubes)
	targetMembersJSON, _ := json.Marshal(policy.TargetMembers)
	conditionsJSON, _ := json.Marshal(policy.Conditions)
	effectsJSON, _ := json.Marshal(policy.Effects)

	result, err := s.db.ExecContext(ctx, `
		UPDATE cube_model_security_policies SET
			name = $1, description = $2, policy_type = $3, priority = $4, enabled = $5,
			target_cubes = $6, target_members = $7, conditions = $8, effects = $9,
			version = $10, updated_at = $11
		WHERE id = $12 AND tenant_id = $13
	`, policy.Name, policy.Description, policy.PolicyType, policy.Priority, policy.Enabled,
		targetCubesJSON, targetMembersJSON, conditionsJSON, effectsJSON,
		policy.Version, policy.UpdatedAt, policy.ID, policy.TenantID)

	if err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("policy not found")
	}

	// Invalidate cache
	return s.InvalidateCache(ctx, policy.TenantID.String())
}

// DeletePolicy deletes an ABAC policy
func (s *SecurityService) DeletePolicy(ctx context.Context, tenantID, policyID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM cube_model_security_policies WHERE id = $1 AND tenant_id = $2
	`, policyID, tenantID)

	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	// Invalidate cache
	return s.InvalidateCache(ctx, tenantID.String())
}

// ListPolicies lists all policies for a tenant
func (s *SecurityService) ListPolicies(ctx context.Context, tenantID uuid.UUID) ([]ABACPolicy, error) {
	return s.getApplicablePolicies(ctx, tenantID.String(), nil)
}

// GetPolicy gets a single policy by ID
func (s *SecurityService) GetPolicy(ctx context.Context, tenantID, policyID uuid.UUID) (*ABACPolicy, error) {
	policies, err := s.getApplicablePolicies(ctx, tenantID.String(), nil)
	if err != nil {
		return nil, err
	}

	for _, p := range policies {
		if p.ID == policyID {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("policy not found")
}

// --- Cube.js Integration Helpers ---

// PrecomputeSecurityForRefresh pre-computes and caches security decisions for scheduled refresh workers
func (s *SecurityService) PrecomputeSecurityForRefresh(ctx context.Context, tenantID string, cubes []string) error {
	// Create a "system" security context for refresh workers
	sysCtx := SecurityContext{
		UserID:           "system-refresh",
		TenantID:         tenantID,
		DatasourceID:     "", // Will be filled per datasource
		Roles:            []string{"system", "refresh-worker"},
		Groups:           []string{},
		Attributes:       map[string]interface{}{"is_system": true},
		RequestTimestamp: time.Now(),
	}

	_, err := s.EvaluateSecurity(ctx, sysCtx, cubes)
	return err
}

// GetCacheStats returns statistics about the security cache
func (s *SecurityService) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	s.l1CacheLock.RLock()
	l1Size := len(s.l1Cache)
	var l1Hits int64
	for _, v := range s.l1Cache {
		l1Hits += v.HitCount
	}
	s.l1CacheLock.RUnlock()

	var l2Stats struct {
		Count     int   `db:"count"`
		TotalHits int64 `db:"total_hits"`
	}
	s.db.GetContext(ctx, &l2Stats, `
		SELECT COUNT(*) as count, COALESCE(SUM(hit_count), 0) as total_hits
		FROM cube_security_cache WHERE expires_at > NOW()
	`)

	return map[string]interface{}{
		"l1_cache_size":  l1Size,
		"l1_total_hits":  l1Hits,
		"l2_cache_size":  l2Stats.Count,
		"l2_total_hits":  l2Stats.TotalHits,
		"l1_ttl_seconds": s.l1TTL.Seconds(),
	}, nil
}
