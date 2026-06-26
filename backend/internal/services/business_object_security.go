package services

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

	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

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

// AccessRule captures a single row from access_rule.
type AccessRule struct {
	RuleID           string          `db:"rule_id"`
	TenantID         string          `db:"tenant_id"`
	BusinessObjectID string          `db:"bo_id"`
	GroupDN          string          `db:"group_dn"`
	RowFilterDSL     sql.NullString  `db:"row_filter_dsl"`
	ColumnMasksRaw   json.RawMessage `db:"column_masks"`
	AccessLevel      AccessLevel     `db:"access_level"`
	Status           string          `db:"status"`
}

// columnMaskWire mirrors JSON structure stored in column_masks.
type columnMaskWire struct {
	Term     string `json:"term"`
	MaskType string `json:"mask_type"`
}

// AccessRuleRepository resolves decisions for BO access.
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

// ResolveDecision composes the effective decision for a principal.
func (r *pgAccessRuleRepository) ResolveDecision(ctx context.Context, tenantID, boID string, principal Principal) (*AccessDecision, error) {
	if r == nil || r.db == nil {
		// Fail-open only when repo is not configured.
		return &AccessDecision{AccessLevel: AccessLevelWrite, ColumnMasks: map[string]string{}}, nil
	}

	groups := dedupeAndSort(principal.Groups)
	if len(groups) == 0 {
		return &AccessDecision{AccessLevel: AccessLevelNone, ColumnMasks: map[string]string{}}, nil
	}

	key := r.cacheKey(tenantID, boID, groups)
	if d := r.loadFromCache(key); d != nil {
		return d, nil
	}

	rules, err := r.fetchRules(ctx, tenantID, boID, groups)
	if err != nil {
		return nil, err
	}

	decision := composeAccessDecision(rules)
	r.storeInCache(key, decision)
	return decision, nil
}

func (r *pgAccessRuleRepository) fetchRules(ctx context.Context, tenantID, boID string, groups []string) ([]AccessRule, error) {
	const q = `
        SELECT rule_id, tenant_id, bo_id, group_dn, row_filter_dsl, column_masks, access_level, status
        FROM access_rule
        WHERE tenant_id = $1 AND bo_id = $2 AND status = 'APPROVED' AND group_dn = ANY($3)
    `

	var rules []AccessRule
	if err := r.db.SelectContext(ctx, &rules, q, tenantID, boID, pq.Array(groups)); err != nil {
		return nil, err
	}
	return rules, nil
}

func (r *pgAccessRuleRepository) cacheKey(tenantID, boID string, groups []string) string {
	h := sha256.Sum256([]byte(strings.Join(groups, "|")))
	return tenantID + "|" + boID + "|" + hex.EncodeToString(h[:])
}

func (r *pgAccessRuleRepository) loadFromCache(key string) *AccessDecision {
	if v, ok := r.cache.Load(key); ok {
		entry := v.(cachedDecision)
		if time.Now().Before(entry.expires) {
			return entry.decision
		}
		r.cache.Delete(key)
	}
	return nil
}

func (r *pgAccessRuleRepository) storeInCache(key string, decision *AccessDecision) {
	r.cache.Store(key, cachedDecision{decision: decision, expires: time.Now().Add(r.ttl)})
}

func dedupeAndSort(values []string) []string {
	if len(values) == 0 {
		return values
	}
	m := make(map[string]struct{}, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}
		m[v] = struct{}{}
	}
	out := make([]string, 0, len(m))
	for v := range m {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func composeAccessDecision(rules []AccessRule) *AccessDecision {
	decision := &AccessDecision{
		AccessLevel: AccessLevelNone,
		ColumnMasks: make(map[string]string),
	}

	if len(rules) == 0 {
		return decision
	}

	rank := func(l AccessLevel) int {
		switch l {
		case AccessLevelWrite:
			return 2
		case AccessLevelRead:
			return 1
		default:
			return 0
		}
	}

	var predicates []string

	for _, r := range rules {
		if rank(r.AccessLevel) > rank(decision.AccessLevel) {
			decision.AccessLevel = r.AccessLevel
		}

		if r.RowFilterDSL.Valid && strings.TrimSpace(r.RowFilterDSL.String) != "" {
			predicates = append(predicates, "("+r.RowFilterDSL.String+")")
		}

		// Merge column masks with most restrictive winning (HIDE > MASK)
		masks := parseColumnMasks(r.ColumnMasksRaw)
		for term, mask := range masks {
			existing, ok := decision.ColumnMasks[term]
			if !ok || (existing == "MASK" && mask == "HIDE") {
				decision.ColumnMasks[term] = mask
			}
		}
	}

	if len(predicates) > 0 {
		decision.RowPredicate = strings.Join(predicates, " OR ")
	}

	return decision
}

func parseColumnMasks(raw json.RawMessage) map[string]string {
	out := make(map[string]string)
	if len(raw) == 0 {
		return out
	}

	var masks []columnMaskWire
	if err := json.Unmarshal(raw, &masks); err != nil {
		logging.GetLogger().Sugar().Warnf("[SECURITY] failed to unmarshal column masks: %v", err)
		return out
	}

	for _, m := range masks {
		term := strings.TrimSpace(m.Term)
		mask := strings.ToUpper(strings.TrimSpace(m.MaskType))
		if term == "" || (mask != "HIDE" && mask != "MASK") {
			continue
		}
		out[term] = mask
	}
	return out
}

// atLeast returns true if current level meets the required level.
func (l AccessLevel) atLeast(required AccessLevel) bool {
	rank := map[AccessLevel]int{
		AccessLevelNone:  0,
		AccessLevelRead:  1,
		AccessLevelWrite: 2,
	}
	return rank[l] >= rank[required]
}

// resolveAccessDecision loads the decision for a BO using the attached repository.
func (s *BusinessObjectService) resolveAccessDecision(ctx context.Context, tenantID, boID string) (*AccessDecision, error) {
	// 1. Resolve Principal from context (Handler vs Service layer gap)
	principal := PrincipalFromContext(ctx)

	// If Principal is empty, try to derive from AuthInfo (Handler context)
	if principal.UserID == "" {
		if auth, ok := security.AuthInfoFromContext(ctx); ok {
			principal = Principal{
				UserID: auth.UserID,
				Groups: auth.Roles, // Map roles to groups for now
			}
		}
	}

	// 2. Global Admin / Global Ops Bypass (Root Access)
	for _, role := range principal.Groups {
		if role == "global_admin" || role == "global_ops" {
			// Full access, no row filters, no column masks
			return &AccessDecision{
				AccessLevel: AccessLevelWrite,
				ColumnMasks: map[string]string{},
			}, nil
		}
	}

	if s.rules == nil {
		logging.GetLogger().Sugar().Warn("[SECURITY] access rule repository not configured; allowing all access")
		return &AccessDecision{AccessLevel: AccessLevelWrite, ColumnMasks: map[string]string{}}, nil
	}

	return s.rules.ResolveDecision(ctx, tenantID, boID, principal)
}

// requireAccess enforces the required level and returns the decision for downstream use.
func (s *BusinessObjectService) requireAccess(ctx context.Context, tenantID, boID string, required AccessLevel) (*AccessDecision, error) {
	decision, err := s.resolveAccessDecision(ctx, tenantID, boID)
	if err != nil {
		return nil, err
	}

	if !decision.AccessLevel.atLeast(required) {
		return nil, ErrForbidden
	}

	if decision.ColumnMasks == nil {
		decision.ColumnMasks = make(map[string]string)
	}

	return decision, nil
}

// applyColumnMasksToInstance enforces HIDE/MASK on core/custom fields.
func applyColumnMasksToInstance(inst *models.BusinessObjectInstance, masks map[string]string) {
	if inst == nil || len(masks) == 0 {
		return
	}

	if inst.CoreFieldValues == nil {
		inst.CoreFieldValues = map[string]interface{}{}
	}
	if inst.CustomFieldValues == nil {
		inst.CustomFieldValues = map[string]interface{}{}
	}

	for term, mask := range masks {
		if mask == "HIDE" {
			delete(inst.CoreFieldValues, term)
			delete(inst.CustomFieldValues, term)
			continue
		}
		if mask == "MASK" {
			if _, ok := inst.CoreFieldValues[term]; ok {
				inst.CoreFieldValues[term] = "[MASKED]"
			}
			if _, ok := inst.CustomFieldValues[term]; ok {
				inst.CustomFieldValues[term] = "[MASKED]"
			}
		}
	}
}
