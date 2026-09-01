package security

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/models"
)

var (
	entitlementCacheHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "uisce_entitlement_cache_hits_total",
		Help: "Total number of entitlement cache hits",
	})
	entitlementCacheMisses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "uisce_entitlement_cache_misses_total",
		Help: "Total number of entitlement cache misses",
	})
	entitlementCacheInvalidations = promauto.NewCounter(prometheus.CounterOpts{
		Name: "uisce_entitlement_cache_invalidations_total",
		Help: "Total number of entitlement cache invalidations",
	})
)

const (
	PermissionNone  = "none"
	PermissionMask  = "mask"
	PermissionRead  = "read"
	PermissionWrite = "write"

	instanceIDDefault = "default"
)

type EntitlementsService struct {
	db    *sqlx.DB
	cache *lru.Cache[string, *EntitlementsResult]
	ttl   time.Duration
	sf    *singleflight
}

func newEmptyEntitlementsResult() *EntitlementsResult {
	return &EntitlementsResult{
		Entitlements:    map[EntitlementKey]string{},
		MaskingPatterns: map[EntitlementKey]string{},
		HiddenBOs:       map[string]struct{}{},
		MaskedFields:    map[EntitlementKey]struct{}{},
	}
}

func NewEntitlementsService(db *sqlx.DB, cacheEntries int, ttl time.Duration) *EntitlementsService {
	if cacheEntries <= 0 {
		cacheEntries = 1000
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}

	cache, err := lru.New[string, *EntitlementsResult](cacheEntries)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("Failed to create entitlement LRU cache: %v; running without cache", err)
		cache, _ = lru.New[string, *EntitlementsResult](100)
	}

	return &EntitlementsService{
		db:    db,
		cache: cache,
		ttl:   ttl,
		sf:    &singleflight{},
	}
}

func (s *EntitlementsService) cacheKey(tenantID, instanceID, userID string, roles []string) string {
	sorted := make([]string, len(roles))
	copy(sorted, roles)
	sort.Strings(sorted)
	h := sha256.New()
	h.Write([]byte(strings.Join([]string{tenantID, instanceID, userID, strings.Join(sorted, "|")}, "|")))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (s *EntitlementsService) ForUser(ctx context.Context, secCtx *Context) (*EntitlementsResult, error) {
	if secCtx.IsGlobalAdmin {
		return nil, nil
	}

	key := s.cacheKey(secCtx.TenantID, secCtx.InstanceID, secCtx.UserID, secCtx.Roles)

	if s.cache != nil {
		if v, ok := s.cache.Get(key); ok {
			entitlementCacheHits.Inc()
			return v, nil
		}
	}

	entitlementCacheMisses.Inc()
	result, err, _ := s.sf.Do(key, func() (interface{}, error) {
		return s.fetchEntitlementsForUser(ctx, secCtx)
	})
	if err != nil {
		return nil, err
	}

	if s.cache != nil && result.(*EntitlementsResult) != nil {
		s.cache.Add(key, result.(*EntitlementsResult))
	}

	return result.(*EntitlementsResult), nil
}

func (s *EntitlementsService) fetchEntitlementsForUser(ctx context.Context, secCtx *Context) (*EntitlementsResult, error) {
	if len(secCtx.Roles) == 0 {
		return newEmptyEntitlementsResult(), nil
	}

	instanceFilter := ""
	args := []interface{}{secCtx.UserID, secCtx.TenantID, secCtx.TenantID, pq.Array(secCtx.Roles)}
	if secCtx.InstanceID != "" && secCtx.InstanceID != instanceIDDefault {
		instanceFilter = "AND (fp.tenant_instance_id IS NULL OR fp.tenant_instance_id = $5)"
		args = append(args, secCtx.InstanceID)
	}

	query := fmt.Sprintf(`
		SELECT
			fp.resource_id,
			fp.resource_type,
			fp.field_name,
			fp.permission_level,
			fp.masking_pattern
		FROM bp_field_permissions fp
		JOIN bp_user_roles ur ON ur.role_id = fp.role_id
		WHERE ur.user_id = $1
		  AND ur.tenant_id = $2
		  AND fp.resource_type = 'business_object'
		  AND fp.tenant_id = $3
		  AND ur.role_key = ANY($4::text[])
		  AND ur.is_active = true
		  %s
	`, instanceFilter)

	rows, err := s.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get BO entitlements: %w", err)
	}
	defer rows.Close()

	result := newEmptyEntitlementsResult()

	for rows.Next() {
		var resID, resType, fieldName, permLevel string
		var maskingPattern *string

		if err := rows.Scan(&resID, &resType, &fieldName, &permLevel, &maskingPattern); err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return nil, fmt.Errorf("failed to scan entitlement row: %w", err)
		}

		key := EntitlementKey{ResourceID: resID, FieldName: fieldName}

		if fieldName == "*" && permLevel == PermissionNone {
			result.HiddenBOs[resID] = struct{}{}
			continue
		}

		existing, ok := result.Entitlements[key]
		if !ok || permissionPriority(permLevel) < permissionPriority(existing) {
			result.Entitlements[key] = permLevel
		}
		if maskingPattern != nil && *maskingPattern != "" {
			result.MaskingPatterns[key] = *maskingPattern
		}
	}

	return result, rows.Err()
}

func permissionPriority(p string) int {
	switch p {
	case PermissionWrite:
		return 1
	case PermissionRead:
		return 2
	case PermissionMask:
		return 3
	case PermissionNone:
		return 4
	default:
		return 5
	}
}

func (s *EntitlementsService) ListVisibleBOs(
	ctx context.Context,
	secCtx *Context,
	bos []*models.BusinessObjectDefinition,
	fieldsPerBO map[string][]*models.FieldDefinition,
) ([]*FilteredBusinessObject, *EntitlementsSummary, error) {
	if secCtx.IsGlobalAdmin {
		filtered := make([]*FilteredBusinessObject, 0, len(bos))
		summary := &EntitlementsSummary{TotalBO: len(bos), VisibleBO: len(bos)}
		for _, bo := range bos {
			filtered = append(filtered, &FilteredBusinessObject{BO: bo})
		}
		return filtered, summary, nil
	}

	entitlements, err := s.ForUser(ctx, secCtx)
	if err != nil {
		return nil, nil, err
	}

	filtered, summary := ApplyBOEntitlements(bos, fieldsPerBO, entitlements, false)
	return filtered, summary, nil
}

func (s *EntitlementsService) FilterFields(
	boID string,
	fields []*models.FieldDefinition,
	entitlements *EntitlementsResult,
) (visible []*models.FieldDefinition, hidden []string, masked map[string]string) {
	if entitlements == nil {
		return fields, nil, nil
	}

	visible = make([]*models.FieldDefinition, 0, len(fields))
	hidden = make([]string, 0)
	masked = make(map[string]string)

	for _, f := range fields {
		fieldName := f.Name
		if fieldName == "" {
			fieldName = f.Key
		}
		key := EntitlementKey{ResourceID: boID, FieldName: fieldName}

		perm := entitlements.Entitlements[key]

		switch perm {
		case PermissionNone:
			hidden = append(hidden, fieldName)
		case PermissionMask:
			fCopy := *f
			fCopy.Masked = true
			if pat, ok := entitlements.MaskingPatterns[key]; ok {
				fCopy.MaskingPattern = pat
				masked[fieldName] = pat
			}
			visible = append(visible, &fCopy)
		case PermissionRead, PermissionWrite:
			visible = append(visible, f)
		default:
			visible = append(visible, f)
		}
	}

	return visible, hidden, masked
}

func (s *EntitlementsService) CanWrite(ctx context.Context, secCtx *Context, boID string) (bool, error) {
	if secCtx.IsGlobalAdmin {
		return true, nil
	}

	entitlements, err := s.ForUser(ctx, secCtx)
	if err != nil {
		return false, err
	}

	if entitlements == nil {
		return true, nil
	}

	key := EntitlementKey{ResourceID: boID, FieldName: "*"}
	perm := entitlements.Entitlements[key]
	return perm == PermissionWrite || perm == PermissionRead, nil
}

func (s *EntitlementsService) CanRunAI(ctx context.Context, secCtx *Context, boID string) (bool, error) {
	return s.CanWrite(ctx, secCtx, boID)
}

func (s *EntitlementsService) Invalidate(tenantID, userID string) {
	if s.cache == nil {
		return
	}

	prefix := tenantID + "|"
	uidSep := "|" + userID + "|"

	removed := 0
	keys := s.cache.Keys()
	for _, key := range keys {
		if strings.HasPrefix(key, prefix) && strings.Contains(key, uidSep) {
			s.cache.Remove(key)
			removed++
		}
	}
	entitlementCacheInvalidations.Add(float64(removed))
}

func (s *EntitlementsService) GetHiddenBOIDs(ctx context.Context, secCtx *Context) (map[string]struct{}, error) {
	if secCtx.IsGlobalAdmin {
		return map[string]struct{}{}, nil
	}

	if len(secCtx.Roles) == 0 {
		return map[string]struct{}{}, nil
	}

	repo := NewBOEntitlementRepository(s.db)
	return repo.GetHiddenBOIDs(ctx, secCtx.UserID, secCtx.Roles, secCtx.TenantID, secCtx.InstanceID)
}

func (s *EntitlementsService) InvalidateAll(tenantID string) {
	if s.cache == nil {
		return
	}

	prefix := tenantID + "|"
	removed := 0
	keys := s.cache.Keys()
	for _, key := range keys {
		if strings.HasPrefix(key, prefix) {
			s.cache.Remove(key)
			removed++
		}
	}
	entitlementCacheInvalidations.Add(float64(removed))
}

type singleflight struct {
	mu sync.Mutex
	m  map[string]*singleflightCall
}

type singleflightCall struct {
	ch  chan struct{}
	val interface{}
	err error
}

func (sf *singleflight) Do(key string, fn func() (interface{}, error)) (interface{}, error, bool) {
	sf.mu.Lock()
	if sf.m == nil {
		sf.m = make(map[string]*singleflightCall)
	}
	if c, ok := sf.m[key]; ok {
		sf.mu.Unlock()
		<-c.ch
		return c.val, c.err, true
	}
	c := &singleflightCall{ch: make(chan struct{})}
	sf.m[key] = c
	sf.mu.Unlock()

	c.val, c.err = fn()
	close(c.ch)

	sf.mu.Lock()
	delete(sf.m, key)
	sf.mu.Unlock()

	return c.val, c.err, false
}
