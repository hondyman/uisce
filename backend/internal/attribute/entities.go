package attribute

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ListEligibleEntities returns catalog tables that have a custom_attributes column.
func (s *Service) ListEligibleEntities(ctx context.Context, tenantID uuid.UUID) ([]EligibleEntity, error) {
	var out []EligibleEntity
	err := s.withTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		rows, err := tx.QueryxContext(ctx, `
			WITH col_nodes AS (
				SELECT
					c.id AS column_node_id,
					c.qualified_path AS column_path,
					c.tenant_datasource_id,
					c.tenant_id,
					CASE
						WHEN c.qualified_path LIKE '%/custom_attributes' THEN
							regexp_replace(c.qualified_path, '/custom_attributes$', '')
						ELSE NULL
					END AS table_path
				FROM public.catalog_node c
				WHERE c.node_name = 'custom_attributes'
				  AND c.is_active
				  AND (
				    c.tenant_id = $1
				    OR c.tenant_id = COALESCE(
				        NULLIF(current_setting('app.shared_reference_tenant', true), '')::uuid,
				        (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1)
				    )
				  )
			),
			tables AS (
				SELECT
					cn.column_node_id,
					cn.column_path,
					cn.tenant_datasource_id,
					COALESCE(t.id, (
						SELECT id FROM public.catalog_node p
						WHERE p.qualified_path = cn.table_path
						  AND p.is_active
						ORDER BY CASE WHEN p.tenant_id = $1 THEN 0 ELSE 1 END
						LIMIT 1
					)) AS table_node_id,
					cn.table_path AS qualified_path
				FROM col_nodes cn
				LEFT JOIN public.catalog_node t
				  ON t.qualified_path = cn.table_path
				 AND t.is_active
				 AND (t.tenant_id = $1 OR t.tenant_id = cn.tenant_id)
				WHERE cn.table_path IS NOT NULL
			)
			SELECT DISTINCT ON (qualified_path)
				table_node_id,
				column_node_id,
				qualified_path,
				tenant_datasource_id
			FROM tables
			WHERE qualified_path IS NOT NULL
			ORDER BY qualified_path, table_node_id NULLS LAST`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()

		type raw struct {
			TableNodeID  *uuid.UUID `db:"table_node_id"`
			ColumnNodeID uuid.UUID  `db:"column_node_id"`
			QualifiedPath string    `db:"qualified_path"`
			DatasourceID *uuid.UUID `db:"tenant_datasource_id"`
		}

		var raws []raw
		for rows.Next() {
			var r raw
			if err := rows.Scan(&r.TableNodeID, &r.ColumnNodeID, &r.QualifiedPath, &r.DatasourceID); err != nil {
				return err
			}
			raws = append(raws, r)
		}
		if err := rows.Err(); err != nil {
			return err
		}

		seen := map[string]bool{}
		for _, r := range raws {
			tableRef, schema, table := normalizeCatalogTablePath(r.QualifiedPath)
			if table == "" {
				continue
			}
			entityType := entityTypeFromTable(table)
			fieldCount := 0
			_ = tx.QueryRowContext(ctx, `
				SELECT count(*) FROM public.attribute_def_effective
				WHERE entity_type = $1 AND is_active`, entityType).Scan(&fieldCount)

			display := humanLabel(table)
			if r.TableNodeID != nil {
				var name string
				if err := tx.QueryRowContext(ctx, `
					SELECT COALESCE(node_name, '') FROM public.catalog_node WHERE id = $1`,
					*r.TableNodeID).Scan(&name); err == nil && name != "" {
					display = name
				}
			}

			colID := r.ColumnNodeID
			seen[entityType] = true
			out = append(out, EligibleEntity{
				EntityType:    entityType,
				TableRef:      tableRef,
				DisplayName:   display,
				SchemaName:    schema,
				TableName:     table,
				QualifiedPath: r.QualifiedPath,
				DatasourceID:  r.DatasourceID,
				FieldCount:    fieldCount,
				ColumnNodeID:  &colID,
				TableNodeID:   r.TableNodeID,
			})
		}

		// Fallback: entities that already have definitions even if catalog
		// has not scanned a custom_attributes column yet.
		defRows, err := tx.QueryxContext(ctx, `
			SELECT entity_type, min(table_ref) AS table_ref, count(*) AS field_count
			FROM public.attribute_def_effective
			WHERE is_active
			GROUP BY entity_type
			ORDER BY entity_type`)
		if err != nil {
			return err
		}
		defer defRows.Close()
		for defRows.Next() {
			var entityType, tableRef string
			var fieldCount int
			if err := defRows.Scan(&entityType, &tableRef, &fieldCount); err != nil {
				return err
			}
			if seen[entityType] {
				continue
			}
			_, schema, table := normalizeCatalogTablePath(tableRef)
			out = append(out, EligibleEntity{
				EntityType:   entityType,
				TableRef:     tableRef,
				DisplayName:  humanLabel(table),
				SchemaName:   schema,
				TableName:    table,
				QualifiedPath: tableRef,
				FieldCount:   fieldCount,
			})
		}
		return defRows.Err()
	})
	return out, err
}

func normalizeCatalogTablePath(path string) (tableRef, schema, table string) {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimPrefix(path, "crims.")
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return "", "", ""
	}
	schema = parts[len(parts)-2]
	table = parts[len(parts)-1]
	// Strip accidental column suffix if present
	if i := strings.Index(table, "/"); i >= 0 {
		table = table[:i]
	}
	tableRef = fmt.Sprintf("%s.%s", schema, table)
	return tableRef, schema, table
}
