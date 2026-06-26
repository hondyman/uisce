package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/platform"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// bundleServiceDBWrapper wraps an in-memory BundleService but synchronizes
// persistence with the backing database when one is configured.
type bundleServiceDBWrapper struct {
	db        *sqlx.DB
	fallback  BundleService
	policySvc platform.PolicyService
}

// NewBundleServiceWithDB returns a BundleService that keeps the existing
// in-memory implementation for business rules while persisting bundle state to
// the provided database. When no database is configured, the original
// in-memory service is returned.
func NewBundleServiceWithDB(policySvc platform.PolicyService, db *sqlx.DB) (BundleService, BundleRoleManager) {
	fallback, roleMgr := NewBundleService(policySvc)
	if db == nil || db.DB == nil {
		return fallback, roleMgr
	}

	if err := LoadBundlesIntoService(fallback, db); err != nil {
		logging.GetLogger().Warn("bundle_service.load_from_db_failed", zap.Error(err))
	}

	svc := &bundleServiceDBWrapper{
		db:        db,
		fallback:  fallback,
		policySvc: policySvc,
	}
	return svc, roleMgr
}

func (s *bundleServiceDBWrapper) bundleImpl() (*bundleServiceImpl, bool) {
	impl, ok := s.fallback.(*bundleServiceImpl)
	return impl, ok
}

func (s *bundleServiceDBWrapper) setInMemoryBundle(bundle *models.DataBundle) {
	if bundle == nil {
		return
	}
	if impl, ok := s.bundleImpl(); ok {
		impl.store.mu.Lock()
		impl.store.bundles[bundle.ID] = bundle
		impl.store.mu.Unlock()
	}
}

func (s *bundleServiceDBWrapper) restoreInMemoryBundle(bundle *models.DataBundle) {
	if bundle == nil {
		return
	}
	if impl, ok := s.bundleImpl(); ok {
		impl.store.mu.Lock()
		impl.store.bundles[bundle.ID] = bundle
		impl.store.mu.Unlock()
	}
}

func (s *bundleServiceDBWrapper) removeInMemoryBundle(id string) {
	if impl, ok := s.bundleImpl(); ok {
		impl.store.mu.Lock()
		delete(impl.store.bundles, id)
		impl.store.mu.Unlock()
	}
}

func (s *bundleServiceDBWrapper) cloneInMemoryBundle(id string) (*models.DataBundle, error) {
	impl, ok := s.bundleImpl()
	if !ok {
		return nil, nil
	}
	impl.store.mu.RLock()
	existing, exists := impl.store.bundles[id]
	impl.store.mu.RUnlock()
	if !exists || existing == nil {
		return nil, nil
	}
	return deepCloneBundle(existing)
}

func (s *bundleServiceDBWrapper) fetchBundleFromDB(id string) (*models.DataBundle, error) {
	if s.db == nil || s.db.DB == nil {
		return nil, fmt.Errorf("database not configured for bundle reads")
	}

	var (
		bundleID, name, audience, version string
		modules, metrics, governance      sql.NullString
		isActive                          bool
		createdAt, updatedAt              time.Time
	)

	err := s.db.QueryRowx(`
        SELECT bundle_id, name, audience, version, modules, metrics, governance, is_active, created_at, updated_at
        FROM private_markets_bundles
        WHERE bundle_id = $1
    `, id).Scan(&bundleID, &name, &audience, &version, &modules, &metrics, &governance, &isActive, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("bundle with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to read bundle from db: %w", err)
	}

	bundle, err := bundleFromDBRow(bundleID, name, audience, version, modules, metrics, governance, isActive, createdAt, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse bundle from db: %w", err)
	}
	return bundle, nil
}

func (s *bundleServiceDBWrapper) fetchBundlesFromDB() ([]*models.DataBundle, error) {
	if s.db == nil || s.db.DB == nil {
		return nil, fmt.Errorf("database not configured for bundle reads")
	}

	rows, err := s.db.Queryx(`
        SELECT bundle_id, name, audience, version, modules, metrics, governance, is_active, created_at, updated_at
        FROM private_markets_bundles
        WHERE is_active = true
        ORDER BY name
    `)
	if err != nil {
		return nil, fmt.Errorf("failed to list bundles from db: %w", err)
	}
	defer rows.Close()

	var out []*models.DataBundle
	for rows.Next() {
		var (
			bundleID, name, audience, version string
			modules, metrics, governance      sql.NullString
			isActive                          bool
			createdAt, updatedAt              time.Time
		)
		if err := rows.Scan(&bundleID, &name, &audience, &version, &modules, &metrics, &governance, &isActive, &createdAt, &updatedAt); err != nil {
			continue
		}
		bundle, err := bundleFromDBRow(bundleID, name, audience, version, modules, metrics, governance, isActive, createdAt, updatedAt)
		if err != nil {
			continue
		}
		out = append(out, bundle)
	}
	return out, rows.Err()
}

func (s *bundleServiceDBWrapper) ensureBundleInMemory(id string) (*models.DataBundle, error) {
	if impl, ok := s.bundleImpl(); ok {
		impl.store.mu.RLock()
		if bundle, exists := impl.store.bundles[id]; exists && bundle != nil {
			impl.store.mu.RUnlock()
			return bundle, nil
		}
		impl.store.mu.RUnlock()
	}
	bundle, err := s.fetchBundleFromDB(id)
	if err != nil {
		return nil, err
	}
	s.setInMemoryBundle(bundle)
	return bundle, nil
}

func (s *bundleServiceDBWrapper) persistBundle(bundle *models.DataBundle) error {
	if s.db == nil || s.db.DB == nil {
		return fmt.Errorf("database not configured for bundle writes")
	}
	if bundle == nil {
		return fmt.Errorf("bundle is nil")
	}

	if strings.TrimSpace(bundle.Name) == "" {
		bundle.Name = "Untitled Bundle"
	}
	if strings.TrimSpace(bundle.Version) == "" {
		bundle.Version = "1.0.0"
	}
	if len(bundle.Audience) == 0 {
		bundle.Audience = []string{"lp"}
	}

	modulesJSON, metricsJSON, governanceJSON, audience, isActive, createdAt, updatedAt, err := serializeBundleForDB(bundle)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
        INSERT INTO private_markets_bundles (bundle_id, name, audience, version, modules, metrics, governance, is_active, created_at, updated_at)
        VALUES ($1,$2,$3,$4,$5::jsonb,$6::jsonb,$7::jsonb,$8,$9,$10)
        ON CONFLICT (bundle_id) DO UPDATE SET
            name = EXCLUDED.name,
            audience = EXCLUDED.audience,
            version = EXCLUDED.version,
            modules = EXCLUDED.modules,
            metrics = EXCLUDED.metrics,
            governance = EXCLUDED.governance,
            is_active = EXCLUDED.is_active,
            updated_at = EXCLUDED.updated_at
    `, bundle.ID, bundle.Name, audience, bundle.Version, string(modulesJSON), string(metricsJSON), string(governanceJSON), isActive, createdAt, updatedAt)
	if err != nil {
		return fmt.Errorf("persist bundle: %w", err)
	}

	bundle.CreatedAt = createdAt
	bundle.UpdatedAt = updatedAt
	return nil
}

func (s *bundleServiceDBWrapper) ensureBundleAccess(user models.User, action, bundleID string) error {
	impl, ok := s.bundleImpl()
	if !ok {
		return nil
	}
	resource := fmt.Sprintf("bundle:%s", bundleID)
	policies := impl.getPoliciesForResource(resource)
	allowed, err := s.policySvc.Can(user, action, resource, policies)
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("user does not have permission to %s bundle %s", action, bundleID)
	}
	return nil
}

func (s *bundleServiceDBWrapper) GetBundle(user models.User, id string) (*models.DataBundle, error) {
	bundle, err := s.fetchBundleFromDB(id)
	if err != nil {
		return nil, err
	}
	s.setInMemoryBundle(bundle)
	if err := s.ensureBundleAccess(user, "read", id); err != nil {
		return nil, err
	}
	return bundle, nil
}

// ListBundles returns active bundles from the DB for the user (audience filtering is minimal).
func (s *bundleServiceDBWrapper) ListBundles(user models.User) ([]*models.DataBundle, error) {
	bundles, err := s.fetchBundlesFromDB()
	if err != nil {
		return nil, err
	}
	if impl, ok := s.bundleImpl(); ok {
		impl.store.mu.Lock()
		impl.store.bundles = make(map[string]*models.DataBundle, len(bundles))
		for _, bundle := range bundles {
			impl.store.bundles[bundle.ID] = bundle
		}
		impl.store.mu.Unlock()
		return impl.ListBundles(user)
	}
	return bundles, nil
}

// CreateBundle delegates to the fallback (in-memory) service.
func (s *bundleServiceDBWrapper) CreateBundle(user models.User, name, description string) (*models.DataBundle, error) {
	if s.db == nil || s.db.DB == nil {
		return s.fallback.CreateBundle(user, name, description)
	}

	bundle, err := s.fallback.CreateBundle(user, name, description)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(bundle.Description) == "" {
		bundle.Description = description
	}
	if err := s.persistBundle(bundle); err != nil {
		s.removeInMemoryBundle(bundle.ID)
		return nil, err
	}
	return bundle, nil
}

func (s *bundleServiceDBWrapper) UpdateBundle(user models.User, id string, measures []models.SemanticObjectReference, dimensions []models.SemanticObjectReference) (*models.DataBundle, error) {
	if s.db == nil || s.db.DB == nil {
		return s.fallback.UpdateBundle(user, id, measures, dimensions)
	}

	if _, err := s.ensureBundleInMemory(id); err != nil {
		return nil, err
	}
	previous, err := s.cloneInMemoryBundle(id)
	if err != nil {
		return nil, err
	}

	bundle, err := s.fallback.UpdateBundle(user, id, measures, dimensions)
	if err != nil {
		return nil, err
	}

	if err := s.persistBundle(bundle); err != nil {
		s.restoreInMemoryBundle(previous)
		return nil, err
	}
	return bundle, nil
}

func (s *bundleServiceDBWrapper) UpdateBundlePolicies(user models.User, id string, rowPolicies []models.BundleRowPolicy, columnPolicies []models.BundleColumnPolicy) (*models.DataBundle, error) {
	if s.db == nil || s.db.DB == nil {
		return s.fallback.UpdateBundlePolicies(user, id, rowPolicies, columnPolicies)
	}

	if _, err := s.ensureBundleInMemory(id); err != nil {
		return nil, err
	}
	previous, err := s.cloneInMemoryBundle(id)
	if err != nil {
		return nil, err
	}

	bundle, err := s.fallback.UpdateBundlePolicies(user, id, rowPolicies, columnPolicies)
	if err != nil {
		return nil, err
	}

	if err := s.persistBundle(bundle); err != nil {
		s.restoreInMemoryBundle(previous)
		return nil, err
	}
	return bundle, nil
}

func (s *bundleServiceDBWrapper) CertifyBundle(user models.User, id string) (*models.DataBundle, error) {
	return s.transitionStatus(user, id, s.fallback.CertifyBundle)
}

func (s *bundleServiceDBWrapper) PublishBundle(user models.User, id string) (*models.DataBundle, error) {
	return s.transitionStatus(user, id, s.fallback.PublishBundle)
}

func (s *bundleServiceDBWrapper) DeprecateBundle(user models.User, id string) (*models.DataBundle, error) {
	return s.transitionStatus(user, id, s.fallback.DeprecateBundle)
}

func (s *bundleServiceDBWrapper) transitionStatus(user models.User, id string, fn func(models.User, string) (*models.DataBundle, error)) (*models.DataBundle, error) {
	if s.db == nil || s.db.DB == nil {
		return fn(user, id)
	}

	if _, err := s.ensureBundleInMemory(id); err != nil {
		return nil, err
	}
	previous, err := s.cloneInMemoryBundle(id)
	if err != nil {
		return nil, err
	}

	bundle, err := fn(user, id)
	if err != nil {
		return nil, err
	}

	if err := s.persistBundle(bundle); err != nil {
		s.restoreInMemoryBundle(previous)
		return nil, err
	}
	return bundle, nil
}

func deepCloneBundle(bundle *models.DataBundle) (*models.DataBundle, error) {
	if bundle == nil {
		return nil, nil
	}
	data, err := json.Marshal(bundle)
	if err != nil {
		return nil, fmt.Errorf("marshal bundle for clone: %w", err)
	}
	var clone models.DataBundle
	if err := json.Unmarshal(data, &clone); err != nil {
		return nil, fmt.Errorf("unmarshal bundle clone: %w", err)
	}
	return &clone, nil
}
