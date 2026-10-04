package tenantschema

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/scanner"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/models"
)

// The template is not a directory of files. It is the gold-copy tenant's datasource and what alpha holds for it after
// that datasource was scanned (ADR-048). A datasource declares, in its config, the schemas a tenant's copy contains.

var (
	// ErrNotGoldCopy: the datasource is not the gold-copy tenant's. Only the gold copy owns templates; a tenant's own
	// datasource is never a template, whoever asks.
	ErrNotGoldCopy = errors.New("tenantschema: the template datasource does not belong to the gold-copy tenant")
	// ErrTemplateNotFound: no such datasource, or it is not visible to the platform.
	ErrTemplateNotFound = errors.New("tenantschema: template datasource not found")
	// ErrNoSchemas: the datasource names no schemas. Nothing is guessed (it is not "all of them").
	ErrNoSchemas = errors.New("tenantschema: the template datasource declares no schemas")
	// ErrBadSchemaName: a declared schema is not a plain lower-case identifier.
	ErrBadSchemaName = errors.New("tenantschema: a declared schema name is not a plain identifier")
	// ErrNoNodes: the scan holds nothing for the datasource.
	ErrNoNodes = errors.New("tenantschema: alpha holds no scan for the template datasource")
)

// Template is what the compiler is given.
type Template struct {
	DatasourceID string
	GoldTenantID string
	Schemas      []string
	Nodes        []*models.CatalogNode
}

// Store is where the template lives. AlphaStore is the real one; tests use a fake.
type Store interface {
	// Owner is the tenant that owns the datasource (the same answer request scoping uses).
	Owner(ctx context.Context, datasourceID string) (tenantID string, err error)
	// IsGoldCopy says whether that tenant is the gold copy.
	IsGoldCopy(ctx context.Context, tenantID string) (bool, error)
	// Read returns the datasource's declared schemas (the raw comma-separated config value) and the scan's nodes.
	Read(ctx context.Context, tenantID, datasourceID string) (schemas string, nodes []*models.CatalogNode, err error)
}

// Loader loads a template.
type Loader struct{ Store Store }

var schemaName = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// Load returns the template for a datasource, refusing anything that is not the gold copy's. The ownership check comes
// before anything is read, so a tenant's own metadata is never loaded as a side effect of a refused request.
func (l Loader) Load(ctx context.Context, datasourceID string) (*Template, error) {
	if l.Store == nil {
		return nil, errors.New("tenantschema: no store configured")
	}
	if _, err := uuid.Parse(datasourceID); err != nil {
		return nil, fmt.Errorf("%w: %q is not a datasource id", ErrTemplateNotFound, datasourceID)
	}
	owner, err := l.Store.Owner(ctx, datasourceID)
	if err != nil {
		return nil, err
	}
	gold, err := l.Store.IsGoldCopy(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("tenantschema: read gold copy flag: %w", err)
	}
	if !gold {
		return nil, ErrNotGoldCopy
	}
	raw, nodes, err := l.Store.Read(ctx, owner, datasourceID)
	if err != nil {
		return nil, err
	}
	var schemas []string
	seen := map[string]bool{}
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if !schemaName.MatchString(s) {
			return nil, fmt.Errorf("%w: %q", ErrBadSchemaName, s)
		}
		if !seen[s] {
			seen[s] = true
			schemas = append(schemas, s)
		}
	}
	if len(schemas) == 0 {
		return nil, ErrNoSchemas
	}
	if len(nodes) == 0 {
		return nil, ErrNoNodes
	}
	return &Template{DatasourceID: datasourceID, GoldTenantID: owner, Schemas: schemas, Nodes: nodes}, nil
}

// AlphaStore reads the template from alpha.
type AlphaStore struct {
	DB       *sql.DB
	Resolver interface {
		Resolve(ctx context.Context, datasourceID string) (*security.ResolvedDatasource, error)
	}
}

var _ Store = (*AlphaStore)(nil)

func (s *AlphaStore) Owner(ctx context.Context, datasourceID string) (string, error) {
	if s == nil || s.Resolver == nil {
		return "", errors.New("tenantschema: alpha store is not configured")
	}
	r, err := s.Resolver.Resolve(ctx, datasourceID)
	if err != nil {
		if errors.Is(err, security.ErrDatasourceNotAvailable) {
			return "", ErrTemplateNotFound
		}
		return "", fmt.Errorf("tenantschema: resolve template datasource: %w", err)
	}
	return r.TenantID, nil
}

// IsGoldCopy is a cross-tenant read of one boolean, so it needs the gold-copy-sync role, as the resolver's own lookup does.
func (s *AlphaStore) IsGoldCopy(ctx context.Context, tenantID string) (bool, error) {
	var gold bool
	err := db.WithGoldCopySync(ctx, s.DB, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, `SELECT COALESCE(gold_copy, false) FROM public.tenants WHERE id = $1`, tenantID).Scan(&gold)
	})
	return gold, err
}

// Read runs as the owning (gold-copy) tenant, so row-level security applies to everything it reads. Column nodes are
// projected to the keys the compiler uses (including scan_id, without which it ignores the column as stale): the rest of a column's properties (sample values, titles, suggested
// validation rules) are catalog content a structure does not need.
func (s *AlphaStore) Read(ctx context.Context, tenantID, datasourceID string) (string, []*models.CatalogNode, error) {
	var schemas string
	var nodes []*models.CatalogNode
	err := db.WithTenantTransaction(ctx, s.DB, tenantID, func(tx *sql.Tx) error {
		var cfg sql.NullString
		if err := tx.QueryRowContext(ctx, `SELECT config->>'schema' FROM public.tenant_product_datasource WHERE id = $1`, datasourceID).Scan(&cfg); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrTemplateNotFound
			}
			return fmt.Errorf("read the datasource: %w", err)
		}
		schemas = cfg.String
		rows, err := tx.QueryContext(ctx, `
			SELECT id, node_type_id, node_name, qualified_path,
			       CASE WHEN node_type_id = $2 THEN jsonb_strip_nulls(jsonb_build_object(
			              'is_physical_column', properties->'is_physical_column', 'format_type', properties->'format_type',
			              'default_value', properties->'default_value', 'is_nullable', properties->'is_nullable',
			              'ordinal_position', properties->'ordinal_position', 'generated', properties->'generated',
			              'identity', properties->'identity', 'collation', properties->'collation', 'scan_id', properties->'scan_id'))
			            ELSE properties END
			  FROM public.catalog_node
			 WHERE tenant_datasource_id = $1 AND is_active AND node_type_id IN ($2, $3, $4)`,
			datasourceID, scanner.NODE_TYPE_COLUMN, scanner.NODE_TYPE_TABLE, scanner.NODE_TYPE_SCHEMA)
		if err != nil {
			return fmt.Errorf("read the scan: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			n := &models.CatalogNode{TenantDatasourceId: uuid.MustParse(datasourceID)}
			var props []byte
			if err := rows.Scan(&n.ID, &n.NodeTypeID, &n.NodeName, &n.QualifiedPath, &props); err != nil {
				return fmt.Errorf("read a scanned node: %w", err)
			}
			n.Properties = json.RawMessage(props)
			nodes = append(nodes, n)
		}
		return rows.Err()
	})
	if err != nil {
		return "", nil, err
	}
	return schemas, nodes, nil
}
