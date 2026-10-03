package tenantdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/dscreds"
	"github.com/hondyman/uisce/backend/internal/security"
)

// DatasourceResolver is security.DBDatasourceResolver's surface: which tenant owns a datasource.
type DatasourceResolver interface {
	Resolve(ctx context.Context, datasourceID string) (*security.ResolvedDatasource, error)
}

// AlphaRegistry implements Registry over alpha. Ownership comes from the security resolver
// (the same answer the request-scoping code uses), and the datasource row and its binding are
// then read INSIDE the owner's tenant transaction, so row-level security applies to both.
type AlphaRegistry struct {
	DB       *sql.DB
	Resolver DatasourceResolver
	Creds    *dscreds.Resolver
}

var _ Registry = (*AlphaRegistry)(nil)

func (r *AlphaRegistry) ResolveDatasource(ctx context.Context, datasourceID string) (Datasource, error) {
	if r == nil || r.DB == nil || r.Resolver == nil {
		return Datasource{}, errors.New("tenantdb: alpha registry is not configured")
	}
	owner, err := r.Resolver.Resolve(ctx, datasourceID)
	if err != nil {
		return Datasource{}, err
	}
	var config []byte
	err = db.WithTenantTransaction(ctx, r.DB, owner.TenantID, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx,
			`SELECT config FROM public.tenant_product_datasource WHERE id = $1`, owner.DatasourceID).Scan(&config)
	})
	if err != nil {
		return Datasource{}, fmt.Errorf("read datasource config: %w", err)
	}
	ds, err := parseDatasource(owner.DatasourceID, owner.TenantID, config)
	if err != nil {
		return Datasource{}, err
	}
	return ds, nil
}

func (r *AlphaRegistry) LoadBinding(ctx context.Context, datasourceID string) (Binding, error) {
	// The binding is read as the datasource's OWNER (looked up again, not trusted from the
	// caller), and a missing row is ErrUnbound, never an empty active binding.
	owner, err := r.Resolver.Resolve(ctx, datasourceID)
	if err != nil {
		return Binding{}, err
	}
	var b Binding
	err = db.WithTenantTransaction(ctx, r.DB, owner.TenantID, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx,
			`SELECT version, lifecycle_state FROM public.tenant_datasource_binding WHERE datasource_id = $1`,
			owner.DatasourceID).Scan(&b.Version, &b.Lifecycle)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Binding{}, ErrUnbound
	}
	if err != nil {
		return Binding{}, fmt.Errorf("read binding: %w", err)
	}
	return b, nil
}

func (r *AlphaRegistry) Credentials(ctx context.Context, ds Datasource) (string, string, error) {
	if r.Creds == nil {
		return "", "", errors.New("tenantdb: no credentials resolver configured")
	}
	hydrated, err := r.Creds.Hydrate(ctx, dscreds.KindDatasource, ds.TenantID, ds.ID, ds.config)
	if err != nil {
		return "", "", err
	}
	return credentialsFromConfig(hydrated)
}

// parseDatasource reads host, port and database from a datasource config. A row missing any of
// them is ErrIncomplete: there is no default host and no default database.
func parseDatasource(id, tenantID string, config []byte) (Datasource, error) {
	var cfg map[string]any
	if err := json.Unmarshal(config, &cfg); err != nil {
		return Datasource{}, fmt.Errorf("%w: config is not an object", ErrIncomplete)
	}
	host, _ := cfg["host"].(string)
	database, _ := cfg["database"].(string)
	port := 0
	switch v := cfg["port"].(type) {
	case float64:
		port = int(v)
	case string:
		port, _ = strconv.Atoi(strings.TrimSpace(v))
	}
	if strings.TrimSpace(host) == "" || strings.TrimSpace(database) == "" || port <= 0 || port > 65535 {
		return Datasource{}, ErrIncomplete
	}
	return Datasource{ID: id, TenantID: tenantID, Host: host, Port: port, Database: database, config: config}, nil
}

// credentialsFromConfig reads the user and password the hydrated config carries (auth.basic,
// or a flat username/password), the shapes dscreds.Hydrate writes.
func credentialsFromConfig(hydrated []byte) (string, string, error) {
	var cfg map[string]any
	if err := json.Unmarshal(hydrated, &cfg); err != nil {
		return "", "", fmt.Errorf("%w: hydrated config is not an object", ErrIncomplete)
	}
	var user, pass string
	if auth, ok := cfg["auth"].(map[string]any); ok {
		if basic, ok := auth["basic"].(map[string]any); ok {
			user, _ = basic["username"].(string)
			pass, _ = basic["password"].(string)
		}
	}
	if user == "" {
		user, _ = cfg["username"].(string)
	}
	if pass == "" {
		pass, _ = cfg["password"].(string)
	}
	if user == "" || pass == "" {
		return "", "", ErrIncomplete
	}
	return user, pass, nil
}

// AppDatasource finds the tenant's one active datasource for an app, read inside the tenant's
// own transaction so row-level security applies.
func (r *AlphaRegistry) AppDatasource(ctx context.Context, tenantID, app string) (string, error) {
	if r == nil || r.DB == nil {
		return "", errors.New("tenantdb: alpha registry is not configured")
	}
	var ids []string
	err := db.WithTenantTransaction(ctx, r.DB, tenantID, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `
			SELECT tpd.id
			FROM public.tenant_product_datasource tpd
			JOIN public.tenant_product tp ON tp.id = tpd.tenant_product_id
			JOIN public.tenant_instance ti ON ti.id = tp.datasource_id
			JOIN public.alpha_datasource ad ON ad.id = tpd.alpha_datasource_id
			WHERE ti.tenant_id = $1 AND ad.datasource_code = $2
			  AND tpd.is_active AND tp.is_active AND ti.is_active
			ORDER BY tpd.id`, tenantID, app)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			ids = append(ids, id)
		}
		return rows.Err()
	})
	if err != nil {
		return "", fmt.Errorf("read app datasources: %w", err)
	}
	switch len(ids) {
	case 0:
		return "", fmt.Errorf("%w: no %q datasource", ErrUnbound, app)
	case 1:
		return ids[0], nil
	default:
		return "", fmt.Errorf("%w: %d for %q", ErrAmbiguousApp, len(ids), app)
	}
}
