package scanner

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/models"
)

// A scan has to record enough to rebuild the structure it scanned, because a tenant structure is built from what
// alpha holds after the gold copy's scan is synced, never straight from the source (ADR-048). Columns, primary and
// unique keys and foreign keys were already recorded. This records the rest, so nothing a deploy needs lives only in
// the source database:
//
//	table node   constraints        [{name, type, definition}]   every local primary key (p), unique (u), check (c),
//	                                foreign key (f) and exclusion (x) constraint, in PostgreSQL's own words, so nothing about
//	                                key order, cascade rules, deferrability or NULLS NOT DISTINCT has to be inferred
//	             indexes            [{name, definition, unique, method}]   every index that is not a primary key or a
//	                                unique constraint (those are the key properties) and not a copy of a partitioned index
//	             partition          {key}            for a partitioned parent  (e.g. "RANGE (quote_time)")
//	                                {parent, bound}  for a partition           (e.g. "FOR VALUES FROM (...) TO (...)")
//	             triggers           [{name, definition, function}]         only triggers the user created, not clones on partitions
//	             persistence, options   only when not the default (an unlogged table; storage options like fillfactor)
//	column node  format_type        the exact type as pg_catalog.format_type prints it ("character varying(150)", "uuid[]")
//	             generated, identity, collation   only when set; a compiler that does not model them must refuse
//	schema node  extensions         [{name, version, schema}]   what the source database has installed (not plpgsql)
//	             routines           [{name, arguments, kind, language, definition}]   functions and procedures, not extension-owned
//	             definitions_captured  true only if every query below succeeded; false (with definitions_error) otherwise
//	             definitions_version   the shape of the above
//
// Definitions are PostgreSQL's own deparse (pg_get_constraintdef, pg_get_indexdef, pg_get_triggerdef, pg_get_functiondef,
// pg_get_partkeydef, pg_get_expr), the same text pg_dump emits, so a compiler can reproduce them exactly.
//
// A scan that could not record them says so (definitions_captured = false) rather than recording nothing silently, so a
// deploy built from it can refuse.

// DefinitionsVersion is the shape of the properties above.
const DefinitionsVersion = 3

type tableDefs struct {
	constraints []map[string]interface{}
	indexes     []map[string]interface{}
	triggers    []map[string]interface{}
	partition   map[string]interface{}
}

// neutralRows is the rows of a query that ran with an empty search_path. Closing it ends the (read-only) transaction.
type neutralRows struct {
	*sql.Rows
	tx *sql.Tx
}

func (r *neutralRows) Close() error {
	err := r.Rows.Close()
	_ = r.tx.Rollback() // read only: nothing to commit
	return err
}

// neutralQuery runs a query in a read-only transaction whose search_path is EMPTY, the way pg_dump reads a database.
//
// This is not cosmetic. PostgreSQL's own deparse (pg_get_constraintdef, pg_get_indexdef, pg_get_triggerdef,
// pg_get_functiondef, pg_get_expr, format_type, and the column default information_schema reports) leaves a name
// unqualified whenever its schema is on the scanning session's search_path. Scanned by a session that could see `mdm`, a foreign key
// is recorded as `REFERENCES party(id)`; scanned by one that could not, as `REFERENCES mdm.party(id)`. The first cannot be
// applied to a tenant's database, where `party` resolves to nothing, and the same text could resolve to the WRONG table where two
// schemas share a name. With an empty path every name outside pg_catalog is qualified, whoever scans and however
// their connection is configured. (pg_catalog is always searched, so its functions and tables need no qualifying.)
func (s *AnsiScanner) neutralQuery(query string, args ...interface{}) (*neutralRows, error) {
	tx, err := s.sourceDB.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`SET LOCAL search_path = ''`); err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("set a neutral search_path: %w", err)
	}
	rows, err := tx.Query(query, args...)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return &neutralRows{Rows: rows, tx: tx}, nil
}

func (s *AnsiScanner) schemaFilter(col string, args *[]interface{}) string {
	if len(s.schemaWhitelist) > 0 {
		ph := make([]string, len(s.schemaWhitelist))
		for i, v := range s.schemaWhitelist {
			*args = append(*args, v)
			ph[i] = fmt.Sprintf("$%d", len(*args))
		}
		return fmt.Sprintf("%s IN (%s)", col, strings.Join(ph, ", "))
	}
	return fmt.Sprintf("%s NOT IN ('pg_catalog', 'information_schema', 'pg_toast')", col)
}

// processDefinitions records check constraints, indexes, partitioning, triggers and routines.
func (s *AnsiScanner) processDefinitions() error {
	schemaNodes := map[string]*models.CatalogNode{}
	tableNodes := map[string]*models.CatalogNode{}
	for _, n := range s.nodes {
		switch n.NodeTypeID {
		case NODE_TYPE_SCHEMA:
			schemaNodes[n.NodeName] = n
		case NODE_TYPE_TABLE:
			tableNodes[strings.TrimPrefix(n.QualifiedPath, "/")] = n
		}
	}
	defs := map[string]*tableDefs{}
	get := func(schema, table string) *tableDefs {
		k := schema + "/" + table
		if defs[k] == nil {
			defs[k] = &tableDefs{}
		}
		return defs[k]
	}
	var firstErr error
	fail := func(what string, err error) {
		logging.GetLogger().Sugar().Warnf("Error recording %s: %v", what, err)
		if firstErr == nil {
			firstErr = fmt.Errorf("%s: %w", what, err)
		}
	}

	// constraints. conislocal leaves out a check a partition inherits from its parent, and conparentid = 0 leaves out the
	// primary key, unique or foreign key a partition carries as a clone of its parent's.
	{
		var args []interface{}
		q := `SELECT n.nspname, c.relname, k.conname, k.contype::text, pg_get_constraintdef(k.oid)
		        FROM pg_constraint k
		        JOIN pg_class c ON c.oid = k.conrelid
		        JOIN pg_namespace n ON n.oid = c.relnamespace
		       WHERE k.contype IN ('p', 'u', 'c', 'f', 'x') AND k.conislocal AND k.conparentid = 0 AND c.relkind IN ('r', 'p') AND ` + s.schemaFilter("n.nspname", &args) + `
		       ORDER BY n.nspname, c.relname, k.conname`
		if rows, err := s.neutralQuery(q, args...); err != nil {
			fail("constraints", err)
		} else {
			for rows.Next() {
				var sc, tb, name, typ, def string
				if err := rows.Scan(&sc, &tb, &name, &typ, &def); err != nil {
					fail("constraints", err)
					continue
				}
				d := get(sc, tb)
				d.constraints = append(d.constraints, map[string]interface{}{"name": name, "type": typ, "definition": def})
			}
			if err := rows.Err(); err != nil {
				fail("constraints", err)
			}
			rows.Close()
		}
	}

	// indexes that are not a key: not behind a primary key, unique or exclusion constraint, and not the copy of a
	// partitioned index that a partition carries (that one is attached through pg_inherits).
	{
		var args []interface{}
		q := `SELECT n.nspname, t.relname, ic.relname, pg_get_indexdef(i.indexrelid), i.indisunique, am.amname
		        FROM pg_index i
		        JOIN pg_class ic ON ic.oid = i.indexrelid
		        JOIN pg_class t ON t.oid = i.indrelid
		        JOIN pg_namespace n ON n.oid = t.relnamespace
		        JOIN pg_am am ON am.oid = ic.relam
		       WHERE t.relkind IN ('r', 'p') AND ` + s.schemaFilter("n.nspname", &args) + `
		         AND NOT EXISTS (SELECT 1 FROM pg_constraint k WHERE k.conindid = i.indexrelid AND k.contype IN ('p', 'u', 'x'))
		         AND NOT EXISTS (SELECT 1 FROM pg_inherits h WHERE h.inhrelid = i.indexrelid)
		       ORDER BY n.nspname, t.relname, ic.relname`
		if rows, err := s.neutralQuery(q, args...); err != nil {
			fail("indexes", err)
		} else {
			for rows.Next() {
				var sc, tb, name, def, method string
				var unique bool
				if err := rows.Scan(&sc, &tb, &name, &def, &unique, &method); err != nil {
					fail("indexes", err)
					continue
				}
				d := get(sc, tb)
				d.indexes = append(d.indexes, map[string]interface{}{"name": name, "definition": def, "unique": unique, "method": method})
			}
			if err := rows.Err(); err != nil {
				fail("indexes", err)
			}
			rows.Close()
		}
	}

	// partitioning: a partitioned parent has a key, a partition has a parent and a bound. relkind is restricted to tables
	// because an index that is a partition of a partitioned index also has relispartition set, with no bound.
	{
		var args []interface{}
		q := `SELECT n.nspname, c.relname,
		             CASE WHEN c.relkind = 'p' THEN pg_get_partkeydef(c.oid) ELSE '' END,
		             CASE WHEN c.relispartition THEN COALESCE(pg_get_expr(c.relpartbound, c.oid), '') ELSE '' END,
		             CASE WHEN c.relispartition THEN
		                  (SELECT pn.nspname || '.' || pc.relname FROM pg_inherits h
		                     JOIN pg_class pc ON pc.oid = h.inhparent JOIN pg_namespace pn ON pn.oid = pc.relnamespace
		                    WHERE h.inhrelid = c.oid) ELSE '' END
		        FROM pg_class c
		        JOIN pg_namespace n ON n.oid = c.relnamespace
		       WHERE c.relkind IN ('r', 'p') AND (c.relkind = 'p' OR c.relispartition) AND ` + s.schemaFilter("n.nspname", &args) + `
		       ORDER BY n.nspname, c.relname`
		if rows, err := s.neutralQuery(q, args...); err != nil {
			fail("partitioning", err)
		} else {
			for rows.Next() {
				var sc, tb, key, bound, parent string
				if err := rows.Scan(&sc, &tb, &key, &bound, &parent); err != nil {
					fail("partitioning", err)
					continue
				}
				p := map[string]interface{}{}
				if key != "" {
					p["key"] = key
				}
				if parent != "" {
					p["parent"], p["bound"] = parent, bound
				}
				get(sc, tb).partition = p
			}
			if err := rows.Err(); err != nil {
				fail("partitioning", err)
			}
			rows.Close()
		}
	}

	// triggers a person created. tgparentid <> 0 is a clone made on a partition.
	{
		var args []interface{}
		q := `SELECT n.nspname, c.relname, t.tgname, pg_get_triggerdef(t.oid), fn.nspname || '.' || p.proname
		        FROM pg_trigger t
		        JOIN pg_class c ON c.oid = t.tgrelid
		        JOIN pg_namespace n ON n.oid = c.relnamespace
		        JOIN pg_proc p ON p.oid = t.tgfoid
		        JOIN pg_namespace fn ON fn.oid = p.pronamespace
		       WHERE NOT t.tgisinternal AND t.tgparentid = 0 AND ` + s.schemaFilter("n.nspname", &args) + `
		       ORDER BY n.nspname, c.relname, t.tgname`
		if rows, err := s.neutralQuery(q, args...); err != nil {
			fail("triggers", err)
		} else {
			for rows.Next() {
				var sc, tb, name, def, fn string
				if err := rows.Scan(&sc, &tb, &name, &def, &fn); err != nil {
					fail("triggers", err)
					continue
				}
				d := get(sc, tb)
				d.triggers = append(d.triggers, map[string]interface{}{"name": name, "definition": def, "function": fn})
			}
			if err := rows.Err(); err != nil {
				fail("triggers", err)
			}
			rows.Close()
		}
	}

	// routines, per schema; extension-owned ones are the extension's, not the schema's.
	routines := map[string][]map[string]interface{}{}
	{
		var args []interface{}
		q := `SELECT n.nspname, p.proname, pg_get_function_identity_arguments(p.oid), p.prokind::text, l.lanname, pg_get_functiondef(p.oid)
		        FROM pg_proc p
		        JOIN pg_namespace n ON n.oid = p.pronamespace
		        JOIN pg_language l ON l.oid = p.prolang
		       WHERE p.prokind IN ('f', 'p') AND ` + s.schemaFilter("n.nspname", &args) + `
		         AND NOT EXISTS (SELECT 1 FROM pg_depend d WHERE d.objid = p.oid AND d.classid = 'pg_proc'::regclass AND d.deptype = 'e')
		       ORDER BY n.nspname, p.proname, pg_get_function_identity_arguments(p.oid)`
		if rows, err := s.neutralQuery(q, args...); err != nil {
			fail("routines", err)
		} else {
			for rows.Next() {
				var sc, name, argsText, kind, lang, def string
				if err := rows.Scan(&sc, &name, &argsText, &kind, &lang, &def); err != nil {
					fail("routines", err)
					continue
				}
				routines[sc] = append(routines[sc], map[string]interface{}{"name": name, "arguments": argsText, "kind": kind, "language": lang, "definition": def})
			}
			if err := rows.Err(); err != nil {
				fail("routines", err)
			}
			rows.Close()
		}
	}

	// exact column types. information_schema.data_type says "ARRAY" or "USER-DEFINED" and loses precision on
	// time and interval types; format_type is what DDL needs. generated, identity and collation are recorded only
	// when set, so a compiler that does not model them can refuse rather than guess.
	{
		var args []interface{}
		q := `SELECT n.nspname, c.relname, a.attname, format_type(a.atttypid, a.atttypmod),
		             a.attgenerated::text, a.attidentity::text,
		             CASE WHEN a.attcollation <> 0 AND a.attcollation <> (SELECT typcollation FROM pg_type WHERE oid = a.atttypid)
		                  THEN (SELECT collname FROM pg_collation WHERE oid = a.attcollation) ELSE '' END
		        FROM pg_attribute a
		        JOIN pg_class c ON c.oid = a.attrelid
		        JOIN pg_namespace n ON n.oid = c.relnamespace
		       WHERE a.attnum > 0 AND NOT a.attisdropped AND c.relkind IN ('r', 'p') AND ` + s.schemaFilter("n.nspname", &args)
		if rows, err := s.neutralQuery(q, args...); err != nil {
			fail("column types", err)
		} else {
			for rows.Next() {
				var sc, tb, col, ft, gen, ident, coll string
				if err := rows.Scan(&sc, &tb, &col, &ft, &gen, &ident, &coll); err != nil {
					fail("column types", err)
					continue
				}
				n := s.columnMap[generateID(s.tenantDatasourceId.String(), s.sourceSystem, NODE_TYPE_COLUMN.String(), fmt.Sprintf("/%s/%s/%s", sc, tb, col))]
				if n == nil {
					continue
				}
				// null when unset, so a column that stopped being generated, identity or collated is not still flagged in alpha
				set := map[string]interface{}{"format_type": ft, "generated": nil, "identity": nil, "collation": nil}
				if gen != "" {
					set["generated"] = gen
				}
				if ident != "" {
					set["identity"] = ident
				}
				if coll != "" {
					set["collation"] = coll
				}
				mergeProps(n, set)
			}
			if err := rows.Err(); err != nil {
				fail("column types", err)
			}
			rows.Close()
		}
	}

	// table options that are not the default: recorded so a compiler that does not model them can refuse.
	opts := map[string]map[string]interface{}{}
	{
		var args []interface{}
		q := `SELECT n.nspname, c.relname, c.relpersistence::text, COALESCE(array_to_string(c.reloptions, ','), '')
		        FROM pg_class c
		        JOIN pg_namespace n ON n.oid = c.relnamespace
		       WHERE c.relkind IN ('r', 'p') AND (c.relpersistence <> 'p' OR c.reloptions IS NOT NULL) AND ` + s.schemaFilter("n.nspname", &args)
		if rows, err := s.neutralQuery(q, args...); err != nil {
			fail("table options", err)
		} else {
			for rows.Next() {
				var sc, tb, persistence, options string
				if err := rows.Scan(&sc, &tb, &persistence, &options); err != nil {
					fail("table options", err)
					continue
				}
				m := map[string]interface{}{}
				if persistence != "p" {
					m["persistence"] = persistence
				}
				if options != "" {
					m["options"] = options
				}
				opts[sc+"/"+tb] = m
			}
			if err := rows.Err(); err != nil {
				fail("table options", err)
			}
			rows.Close()
		}
	}

	// extensions the source has installed. A tenant structure creates the ones its definitions need, so the scan has to say
	// which exist and where; plpgsql is always there.
	var extensions []map[string]interface{}
	if rows, err := s.neutralQuery(`SELECT e.extname, e.extversion, n.nspname FROM pg_extension e JOIN pg_namespace n ON n.oid = e.extnamespace WHERE e.extname <> 'plpgsql' ORDER BY e.extname`); err != nil {
		fail("extensions", err)
	} else {
		for rows.Next() {
			var name, version, schema string
			if err := rows.Scan(&name, &version, &schema); err != nil {
				fail("extensions", err)
				continue
			}
			extensions = append(extensions, map[string]interface{}{"name": name, "version": version, "schema": schema})
		}
		if err := rows.Err(); err != nil {
			fail("extensions", err)
		}
		rows.Close()
	}

	// Every table node gets every structural key, empty or null when it has none of that kind. The merge into alpha keeps
	// keys a new scan does not mention, so an index, constraint, trigger or partition that was dropped in the source would
	// otherwise stay recorded in alpha and be deployed to every new tenant.
	for key, n := range tableNodes {
		d := defs[key]
		if d == nil {
			d = &tableDefs{}
		}
		set := map[string]interface{}{
			"constraints": nonNilList(d.constraints), "indexes": nonNilList(d.indexes), "triggers": nonNilList(d.triggers),
			"partition": nil, "persistence": nil, "options": nil,
		}
		if d.partition != nil {
			set["partition"] = d.partition
		}
		if o := opts[key]; o != nil {
			for k, v := range o {
				set[k] = v
			}
		}
		mergeProps(n, set)
	}
	for name, n := range schemaNodes {
		set := map[string]interface{}{"definitions_version": DefinitionsVersion, "definitions_captured": firstErr == nil,
			"definitions_error": nil, "extensions": nonNilList(extensions), "routines": nonNilList(routines[name])}
		if firstErr != nil {
			set["definitions_error"] = firstErr.Error()
		}
		mergeProps(n, set)
	}
	return firstErr
}

// mergeProps adds keys to a node's properties, keeping every key already there.
func mergeProps(n *models.CatalogNode, add map[string]interface{}) {
	if len(add) == 0 {
		return
	}
	var m map[string]interface{}
	if len(n.Properties) > 0 {
		if err := json.Unmarshal(n.Properties, &m); err != nil {
			logging.GetLogger().Sugar().Warnf("Error unmarshaling properties for %s: %v", n.QualifiedPath, err)
			return
		}
	}
	if m == nil {
		m = map[string]interface{}{}
	}
	for k, v := range add {
		m[k] = v
	}
	b, err := json.Marshal(m)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("Error marshaling properties for %s: %v", n.QualifiedPath, err)
		return
	}
	n.Properties = b
}

// nonNilList is l, or an empty list: an empty list is JSON [], which replaces a stale list in alpha, where a missing key
// would not.
func nonNilList(l []map[string]interface{}) []map[string]interface{} {
	if l == nil {
		return []map[string]interface{}{}
	}
	return l
}
