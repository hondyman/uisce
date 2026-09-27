package attribute

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ListEligibleEntities returns tables that have a custom_attributes column.
// Sources (merged, deduped by table_ref):
//  1. information_schema (authoritative physical presence)
//  2. catalog_node column rows named custom_attributes
//  3. attribute_def rows (defs-only fallback)
func (s *Service) ListEligibleEntities(ctx context.Context, tenantID uuid.UUID) ([]EligibleEntity, error) {
	var out []EligibleEntity
	err := s.withTenant(ctx, tenantID, func(tx *sqlx.Tx) error {
		seen := map[string]EligibleEntity{}
		fieldCounts := map[string]int{}

		countRows, err := tx.QueryxContext(ctx, `
			SELECT table_ref, entity_type, count(*) AS field_count
			FROM public.attribute_def_effective
			WHERE is_active
			GROUP BY table_ref, entity_type`)
		if err != nil {
			return err
		}
		for countRows.Next() {
			var tableRef, entityType string
			var n int
			if err := countRows.Scan(&tableRef, &entityType, &n); err != nil {
				countRows.Close()
				return err
			}
			fieldCounts[tableRef] = n
			if fieldCounts[entityType] < n {
				fieldCounts[entityType] = n
			}
		}
		if err := countRows.Err(); err != nil {
			countRows.Close()
			return err
		}
		countRows.Close()

		fieldCountFor := func(entityType, tableRef string) int {
			if n, ok := fieldCounts[tableRef]; ok {
				return n
			}
			return fieldCounts[entityType]
		}

		upsert := func(e EligibleEntity) {
			if e.TableRef == "" || e.TableName == "" {
				return
			}
			if e.FieldCount == 0 {
				e.FieldCount = fieldCountFor(e.EntityType, e.TableRef)
			}
			if prev, ok := seen[e.TableRef]; ok {
				if e.ColumnNodeID != nil && prev.ColumnNodeID == nil {
					prev.ColumnNodeID = e.ColumnNodeID
				}
				if e.TableNodeID != nil && prev.TableNodeID == nil {
					prev.TableNodeID = e.TableNodeID
				}
				if e.DatasourceID != nil && prev.DatasourceID == nil {
					prev.DatasourceID = e.DatasourceID
				}
				if e.FieldCount > prev.FieldCount {
					prev.FieldCount = e.FieldCount
				}
				if e.QualifiedPath != "" && prev.QualifiedPath == "" {
					prev.QualifiedPath = e.QualifiedPath
				}
				seen[e.TableRef] = prev
				return
			}
			seen[e.TableRef] = e
		}

		// 1) Physical tables
		infoRows, err := tx.QueryxContext(ctx, `
			SELECT table_schema, table_name
			FROM information_schema.columns
			WHERE column_name = 'custom_attributes'
			  AND table_schema NOT IN ('pg_catalog', 'information_schema')
			ORDER BY table_schema, table_name`)
		if err != nil {
			return err
		}
		for infoRows.Next() {
			var schema, table string
			if err := infoRows.Scan(&schema, &table); err != nil {
				infoRows.Close()
				return err
			}
			tableRef := schema + "." + table
			entityType := entityTypeFromTable(table)
			upsert(EligibleEntity{
				EntityType:    entityType,
				TableRef:      tableRef,
				DisplayName:   humanLabel(table),
				SchemaName:    schema,
				TableName:     table,
				QualifiedPath: "/" + schema + "/" + table,
			})
		}
		if err := infoRows.Err(); err != nil {
			infoRows.Close()
			return err
		}
		infoRows.Close()

		// 2) Enrich / add from catalog column nodes
		catRows, err := tx.QueryxContext(ctx, `
			SELECT c.id, c.qualified_path, c.tenant_datasource_id
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
			ORDER BY c.qualified_path`, tenantID)
		if err != nil {
			return err
		}
		for catRows.Next() {
			var colID uuid.UUID
			var qPath string
			var dsID *uuid.UUID
			if err := catRows.Scan(&colID, &qPath, &dsID); err != nil {
				catRows.Close()
				return err
			}
			tableRef, schema, table := normalizeCatalogTablePath(qPath)
			if table == "" {
				continue
			}
			entityType := entityTypeFromTable(table)
			idCopy := colID
			upsert(EligibleEntity{
				EntityType:    entityType,
				TableRef:      tableRef,
				DisplayName:   humanLabel(table),
				SchemaName:    schema,
				TableName:     table,
				QualifiedPath: strings.TrimSuffix(qPath, "/custom_attributes"),
				DatasourceID:  dsID,
				ColumnNodeID:  &idCopy,
			})
		}
		if err := catRows.Err(); err != nil {
			catRows.Close()
			return err
		}
		catRows.Close()

		// 3) Defs-only fallback
		for tableRef, n := range fieldCounts {
			if strings.Contains(tableRef, ".") {
				_, schema, table := normalizeCatalogTablePath(tableRef)
				upsert(EligibleEntity{
					EntityType:    entityTypeFromTable(table),
					TableRef:      tableRef,
					DisplayName:   humanLabel(table),
					SchemaName:    schema,
					TableName:     table,
					QualifiedPath: tableRef,
					FieldCount:    n,
				})
			}
		}

		out = make([]EligibleEntity, 0, len(seen))
		for _, e := range seen {
			out = append(out, e)
		}
		sortEligibleEntities(out)
		return nil
	})
	return out, err
}

func sortEligibleEntities(items []EligibleEntity) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			ai := items[i].SchemaName + "." + items[i].TableName
			aj := items[j].SchemaName + "." + items[j].TableName
			if aj < ai {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func normalizeCatalogTablePath(path string) (tableRef, schema, table string) {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "crims.")
	path = strings.TrimPrefix(path, "crims/")

	path = strings.TrimSuffix(path, "/custom_attributes")
	path = strings.TrimSuffix(path, ".custom_attributes")

	if strings.Contains(path, "/") {
		path = strings.Trim(path, "/")
		parts := strings.Split(path, "/")
		if len(parts) < 2 {
			return "", "", ""
		}
		schema = parts[len(parts)-2]
		table = parts[len(parts)-1]
		if !safeIdentRe.MatchString(schema) || !safeIdentRe.MatchString(table) {
			return "", "", ""
		}
		return schema + "." + table, schema, table
	}

	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return "", "", ""
	}
	schema = parts[len(parts)-2]
	table = parts[len(parts)-1]
	if !safeIdentRe.MatchString(schema) || !safeIdentRe.MatchString(table) {
		return "", "", ""
	}
	return schema + "." + table, schema, table
}
