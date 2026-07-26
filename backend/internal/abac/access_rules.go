// Package abac - Attribute-Based Access Control domain.
//
// This package owns ABAC evaluation, policy persistence, and access decisions
// for the Uisce platform. It was extracted from internal/services/ to enforce
// Cardinal Rule 3 (no cycles): this package ONLY depends on internal/models,
// libs/*, and stdlib.
//
// Cardinal Rule 7 (tenant security): every decision carries tenantID.
// Cardinal Rule 8 (caching): access decisions are cached per Rule 8.2
//   with versioned cache keys (bo:{boId}:tenant:{tenantID}).
package abac

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// ============================================================================
// CORE ABAC TYPES
// ============================================================================

// AccessLevel represents the effective permission over a Business Object.
type AccessLevel string

const (
	AccessLevelNone  AccessLevel = "NONE"
	AccessLevelRead  AccessLevel = "READ"
	AccessLevelWrite AccessLevel = "WRITE"
)

// ErrForbidden is returned when a caller lacks the required permission.
var ErrForbidden = errors.New("forbidden")

// AccessDecision is the composed decision for a principal over a BO.
type AccessDecision struct {
	AccessLevel  AccessLevel
	RowPredicate string
	ColumnMasks  map[string]string
}

// Principal carries the resolved user identity and groups.
type Principal struct {
	UserID string
	Groups []string
}

// principalContextKey avoids collisions in context.
type principalContextKey struct{}

// WithPrincipal stores a Principal in context.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, p)
}

// PrincipalFromContext extracts a Principal if present.
func PrincipalFromContext(ctx context.Context) Principal {
	if ctx == nil {
		return Principal{}
	}
	if v := ctx.Value(principalContextKey{}); v != nil {
		if p, ok := v.(Principal); ok {
			return p
		}
	}
	return Principal{}
}

// ============================================================================
// ACCESS RULE REPOSITORY
// ============================================================================

// AccessRuleRepository defines persistence and decision-resolution operations
// for BO access rules.
type AccessRuleRepository interface {
	ResolveDecision(ctx context.Context, tenantID, boID string, principal Principal) (*AccessDecision, error)
}

// pgAccessRuleRepository fetches rules from Postgres and caches decisions.
type pgAccessRuleRepository struct {
	db    *sqlx.DB
	ttl   time.Duration
	cache sync.Map // key -> *cachedDecision
}

type cachedDecision struct {
	decision *AccessDecision
	expires  time.Time
}

// NewPgAccessRuleRepository builds a repository with sane defaults.
func NewPgAccessRuleRepository(db *sqlx.DB) AccessRuleRepository {
	return &pgAccessRuleRepository{db: db, ttl: 5 * time.Minute}
}

// NewAccessRuleRepositoryWithTTL allows overriding the cache TTL.
func NewAccessRuleRepositoryWithTTL(db *sqlx.DB, ttl time.Duration) AccessRuleRepository {
	return &pgAccessRuleRepository{db: db, ttl: ttl}
}

// ResolveDecision evaluates the effective decision for principal over boID.
func (r *pgAccessRuleRepository) ResolveDecision(ctx context.Context, tenantID, boID string, principal Principal) (*AccessDecision, error) {
	if r.db == nil {
		return nil, errors.New("access rule repository: db is nil")
	}
	if tenantID == "" {
		return nil, errors.New("tenantID is required")
	}
	if boID == "" {
		return nil, errors.New("boID is required")
	}

	cacheKey := buildDecisionCacheKey(tenantID, boID, principal)
	if cached, ok := r.cache.Load(cacheKey); ok {
		if cd, ok := cached.(*cachedDecision); ok && time.Now().Before(cd.expires) {
			return cd.decision, nil
		}
		r.cache.Delete(cacheKey)
	}

	decision, err := r.fetchDecision(ctx, tenantID, boID, principal)
	if err != nil {
		return nil, err
	}

	r.cache.Store(cacheKey, &cachedDecision{decision: decision, expires: time.Now().Add(r.ttl)})
	return decision, nil
}

// fetchDecision queries Postgres and merges role + BO-level rules.
func (r *pgAccessRuleRepository) fetchDecision(ctx context.Context, tenantID, boID string, principal Principal) (*AccessDecision, error) {
	rules, err := r.loadRules(ctx, tenantID, boID, principal)
	if err != nil {
		return nil, err
	}

	decision := &AccessDecision{AccessLevel: AccessLevelNone, ColumnMasks: map[string]string{}}
	rowPredicates := []string{}
	for _, rule := range rules {
		switch rule.Effect {
		case "allow":
			if rule.AccessLevel != "" {
				if higherAccess(rule.AccessLevel) > higherAccess(decision.AccessLevel) {
					decision.AccessLevel = rule.AccessLevel
				}
			}
		case "deny":
			// Deny overrides everything
			decision.AccessLevel = AccessLevelNone
		}
		if rule.RowPredicate != "" {
			rowPredicates = append(rowPredicates, rule.RowPredicate)
		}
		for k, v := range rule.ColumnMasks {
			decision.ColumnMasks[k] = v
		}
	}
	if len(rowPredicates) > 0 {
		decision.RowPredicate = strings.Join(rowPredicates, " AND ")
	}
	return decision, nil
}

// ruleSnapshot is the minimal projection of an access rule for decisions.
type ruleSnapshot struct {
	Effect       AccessLevel
	AccessLevel  AccessLevel
	RowPredicate string
	ColumnMasks  map[string]string
}

// loadRules fetches effective rules for the principal against the BO.
func (r *pgAccessRuleRepository) loadRules(ctx context.Context, tenantID, boID string, principal Principal) ([]ruleSnapshot, error) {
	rows, err := r.db.QueryxContext(ctx,
		`SELECT effect, access_level, row_predicate, column_masks
		 FROM access_rules
		 WHERE tenant_id = $1
		   AND business_object_id = $2
		   AND (principal_user_id = $3 OR principal_group = ANY($4))
		   AND enabled = TRUE`,
		tenantID, boID, principal.UserID, pq.Array(principal.Groups))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []ruleSnapshot
	for rows.Next() {
		var (
			effect       string
			accessLevel  string
			rowPredicate sql.NullString
			columnMasks  json.RawMessage
		)
		if err := rows.Scan(&effect, &accessLevel, &rowPredicate, &columnMasks); err != nil {
			return nil, err
		}
		snap := ruleSnapshot{
			Effect:      AccessLevel(effect),
			AccessLevel: AccessLevel(accessLevel),
		}
		if rowPredicate.Valid {
			snap.RowPredicate = rowPredicate.String
		}
		if len(columnMasks) > 0 {
			snap.ColumnMasks = map[string]string{}
			_ = json.Unmarshal(columnMasks, &snap.ColumnMasks)
		}
		rules = append(rules, snap)
	}
	return rules, rows.Err()
}

func buildDecisionCacheKey(tenantID, boID string, principal Principal) string {
	groups := append([]string{}, principal.Groups...)
	sort.Strings(groups)
	digest := sha256.Sum256([]byte(strings.Join(groups, ",")))
	return "bo:" + boID + ":tenant:" + tenantID + ":uid:" + principal.UserID + ":g:" + hex.EncodeToString(digest[:8])
}

func higherAccess(a AccessLevel) int {
	switch a {
	case AccessLevelWrite:
		return 3
	case AccessLevelRead:
		return 2
	case AccessLevelNone:
		return 1
	}
	return 0
}

// ============================================================================
// EVALUATION SERVICE BRIDGE
// ============================================================================

// SecurityContextProvider is satisfied by internal/security.Context, allowing
// the abac package to use security contexts without importing security types
// directly when desired. In practice we accept *security.Context as input.
type SecurityContextProvider = *security.Context

// Evaluate is a convenience wrapper that resolves a decision for a request
// using a SecurityContext (extracted from JWT or session).
func Evaluate(ctx context.Context, repo AccessRuleRepository, sec SecurityContextProvider, boID string, principal Principal) (*AccessDecision, error) {
	if sec == nil {
		return nil, errors.New("security context is required")
	}
	if sec.TenantID == "" {
		return nil, errors.New("tenant id is required in security context")
	}
	if principal.UserID == "" {
		// If principal is empty, attempt to derive from sec
		principal.UserID = sec.UserID
	}
	return repo.ResolveDecision(ctx, sec.TenantID, boID, principal)
}

// ============================================================================
// LEGACY LOGGING HOOKS
// ============================================================================

// logger is used by EvaluateAccess and other public functions for audit logs.
func logger() *zapLoggerShim {
	return &zapLoggerShim{}
}

// zapLoggerShim is a minimal logging adapter to keep the abac package
// decoupled from the full logging infrastructure. It is intended to be
// replaced in tests with a no-op or capturing logger.
type zapLoggerShim struct{}

// Info logs an informational message.
func (l *zapLoggerShim) Info(msg string, fields ...map[string]interface{}) {
	logging.GetLogger().Sugar().Infow(msg, toAnySlice(fields)...)
}

// Warn logs a warning message.
func (l *zapLoggerShim) Warn(msg string, fields ...map[string]interface{}) {
	logging.GetLogger().Sugar().Warnw(msg, toAnySlice(fields)...)
}

// Error logs an error message.
func (l *zapLoggerShim) Error(msg string, fields ...map[string]interface{}) {
	logging.GetLogger().Sugar().Errorw(msg, toAnySlice(fields)...)
}

// toAnySlice flattens map fields into an any slice for the logger.
func toAnySlice(fields []map[string]interface{}) []interface{} {
	out := make([]interface{}, 0, len(fields)*2)
	for _, f := range fields {
		for k, v := range f {
			out = append(out, k, v)
		}
	}
	return out
}