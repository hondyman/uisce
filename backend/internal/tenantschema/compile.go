// Package tenantschema compiles the DDL for a new tenant's database from what alpha holds after the gold copy's
// datasource scan (ADR-050). It reads catalog nodes only, never the source database: alpha metadata wins, so
// anything the scan did not record is not deployed, and anything the compiler cannot reproduce faithfully is refused.
//
// Output is deterministic: the same nodes always produce the same SQL, and Plan.Hash identifies it, so a database
// built from it can be recognised, cached and cloned by that hash.
package tenantschema

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/hondyman/uisce/backend/internal/scanner"
	"github.com/hondyman/uisce/backend/models"
)

// Refusals. Each means the scan cannot be deployed as it stands; none of them is retried.
var (
	// ErrScanNotComplete: a schema's scan did not record definitions (or recorded an older shape). Rescan.
	ErrScanNotComplete = errors.New("the scan does not record every definition a deploy needs; rescan the datasource")
	// ErrUnsupported: the scan recorded something this compiler does not model, so deploying would lose it.
	ErrUnsupported = errors.New("the scan records something the compiler cannot reproduce")
	// ErrBadScan: the nodes are inconsistent (a column with no table, a partition of a table that is not there).
	ErrBadScan = errors.New("the scan is inconsistent")
)

// Options is policy, not metadata.
type Options struct {
	// Schemas is the template: the schemas a tenant's copy contains, in the order they are created. Everything else the
	// scan holds is ignored.
	Schemas []string
	// Extensions is the allow-list of extensions a tenant database may create. An extension the source has installed
	// that is not on the list is not created, and if a definition needs it, applying the plan fails inside its
	// transaction and nothing is half-built.
	Extensions []string
}

// DefaultExtensions are the contrib extensions that are trusted (a database owner may create them) and that the gold
// copy's definitions have used.
var DefaultExtensions = []string{"uuid-ossp", "pgcrypto", "pg_trgm", "citext", "ltree", "btree_gin", "btree_gist"}

// Statement is one step of a plan.
type Statement struct {
	Phase  string
	Schema string
	Object string
	SQL    string
}

// Plan is an ordered, deterministic set of statements.
type Plan struct {
	Statements []Statement
	Tables     int
}

// SQL is the plan as one script, to be applied in a single transaction.
func (p *Plan) SQL() string {
	var b strings.Builder
	for _, s := range p.Statements {
		b.WriteString(s.SQL)
		if !strings.HasSuffix(s.SQL, "\n") {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// Hash identifies the plan. Two plans with the same hash build the same structure.
func (p *Plan) Hash() string {
	h := sha256.Sum256([]byte(p.SQL()))
	return hex.EncodeToString(h[:])
}

type column struct {
	name, formatType, def string
	notNull               bool
	ordinal               int
}

type table struct {
	schema, name string
	cols         []column
	props        map[string]interface{}
	kids         []*table // partitions
	parent       string   // schema.table of the partitioned parent, if this is a partition
	bound        string
	key          string // partition key, if this table is partitioned
}

func qident(s string) string           { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }
func qname(schema, name string) string { return qident(schema) + "." + qident(name) }

func propsOf(n *models.CatalogNode) (map[string]interface{}, error) {
	var m map[string]interface{}
	if len(n.Properties) == 0 {
		return map[string]interface{}{}, nil
	}
	if err := json.Unmarshal(n.Properties, &m); err != nil {
		return nil, fmt.Errorf("%w: properties of %s are not JSON", ErrBadScan, n.QualifiedPath)
	}
	if m == nil {
		m = map[string]interface{}{}
	}
	return m, nil
}

func list(m map[string]interface{}, key string) []map[string]interface{} {
	var out []map[string]interface{}
	if l, ok := m[key].([]interface{}); ok {
		for _, e := range l {
			if em, ok := e.(map[string]interface{}); ok {
				out = append(out, em)
			}
		}
	}
	return out
}

func str(m map[string]interface{}, key string) string {
	s, _ := m[key].(string)
	return s
}

var onOnly = regexp.MustCompile(`^(CREATE (?:UNIQUE )?INDEX \S+ ON) ONLY `)

// Compile builds the plan for the schemas in opts.Schemas from nodes (the catalog nodes of the gold copy's datasource).
func Compile(nodes []*models.CatalogNode, opts Options) (*Plan, error) {
	if len(opts.Schemas) == 0 {
		return nil, errors.New("tenantschema: the template names no schemas")
	}
	inTemplate := map[string]int{}
	for i, s := range opts.Schemas {
		inTemplate[s] = i
	}
	allowExt := map[string]bool{}
	exts := opts.Extensions
	if exts == nil {
		exts = DefaultExtensions
	}
	for _, e := range exts {
		allowExt[e] = true
	}

	// Two passes: a table or column is trusted only if it came from the same scan as its schema node. The merge of a scan into
	// alpha never deletes, so a column that has since vanished from the source is still there, still active, and carries the id
	// of an older scan; it is ignored here, not deployed.
	schemaProps := map[string]map[string]interface{}{}
	for _, n := range nodes {
		if n.NodeTypeID != scanner.NODE_TYPE_SCHEMA {
			continue
		}
		if _, ok := inTemplate[n.NodeName]; !ok {
			continue
		}
		p, err := propsOf(n)
		if err != nil {
			return nil, err
		}
		schemaProps[n.NodeName] = p
	}
	tables := map[string]*table{}
	colsByTable := map[string][]*models.CatalogNode{}
	for _, n := range nodes {
		switch n.NodeTypeID {
		case scanner.NODE_TYPE_TABLE:
			p, err := propsOf(n)
			if err != nil {
				return nil, err
			}
			sc := str(p, "schema")
			sp, ok := schemaProps[sc]
			if !ok || str(p, "scan_id") == "" || str(p, "scan_id") != str(sp, "scan_id") {
				continue
			}
			tables[sc+"."+n.NodeName] = &table{schema: sc, name: n.NodeName, props: p}
		case scanner.NODE_TYPE_COLUMN:
			parts := strings.Split(strings.TrimPrefix(n.QualifiedPath, "/"), "/")
			if len(parts) != 3 {
				continue
			}
			sp, ok := schemaProps[parts[0]]
			if !ok {
				continue
			}
			p, err := propsOf(n)
			if err != nil {
				return nil, err
			}
			if str(p, "scan_id") == "" || str(p, "scan_id") != str(sp, "scan_id") {
				continue
			}
			colsByTable[parts[0]+"."+parts[1]] = append(colsByTable[parts[0]+"."+parts[1]], n)
		}
	}

	// 1. Every schema in the template was scanned, and its scan recorded everything. Fail closed on the first doubt.
	var incomplete []string
	for _, s := range opts.Schemas {
		p, ok := schemaProps[s]
		if !ok {
			incomplete = append(incomplete, s+" (not in the scan)")
			continue
		}
		if p["definitions_captured"] != true || p["definitions_version"] != float64(scanner.DefinitionsVersion) || str(p, "scan_id") == "" {
			incomplete = append(incomplete, s)
		}
	}
	if len(incomplete) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrScanNotComplete, strings.Join(incomplete, ", "))
	}

	// 2. Columns, refusing what is not modelled.
	for key, t := range tables {
		if v := str(t.props, "persistence"); v != "" {
			return nil, fmt.Errorf("%w: %s is not a plain logged table (persistence %q)", ErrUnsupported, key, v)
		}
		if v := str(t.props, "options"); v != "" {
			return nil, fmt.Errorf("%w: %s has storage options (%s)", ErrUnsupported, key, v)
		}
		for _, cn := range colsByTable[key] {
			p, err := propsOf(cn)
			if err != nil {
				return nil, err
			}
			if p["is_physical_column"] != true {
				continue
			}
			for _, k := range []string{"generated", "identity", "collation"} {
				if v := str(p, k); v != "" {
					return nil, fmt.Errorf("%w: column %s has %s=%q", ErrUnsupported, cn.QualifiedPath, k, v)
				}
			}
			ft := str(p, "format_type")
			if ft == "" {
				return nil, fmt.Errorf("%w: column %s has no recorded type", ErrScanNotComplete, cn.QualifiedPath)
			}
			ord := 0
			if f, ok := p["ordinal_position"].(float64); ok {
				ord = int(f)
			}
			nullable, _ := p["is_nullable"].(bool)
			t.cols = append(t.cols, column{name: cn.NodeName, formatType: ft, def: str(p, "default_value"), notNull: !nullable, ordinal: ord})
		}
		sort.Slice(t.cols, func(i, j int) bool { return t.cols[i].ordinal < t.cols[j].ordinal })
		if pp, ok := t.props["partition"].(map[string]interface{}); ok {
			t.key = str(pp, "key")
			t.parent, t.bound = str(pp, "parent"), str(pp, "bound")
		}
		if t.parent == "" && len(t.cols) == 0 {
			return nil, fmt.Errorf("%w: table %s has no columns", ErrBadScan, key)
		}
	}
	for key, t := range tables {
		if t.parent == "" {
			continue
		}
		par, ok := tables[t.parent]
		if !ok {
			return nil, fmt.Errorf("%w: %s is a partition of %s, which is not in the scan", ErrBadScan, key, t.parent)
		}
		par.kids = append(par.kids, t)
	}

	// deterministic order: template schema order, then name
	var all []*table
	for _, t := range tables {
		all = append(all, t)
	}
	sort.Slice(all, func(i, j int) bool {
		if inTemplate[all[i].schema] != inTemplate[all[j].schema] {
			return inTemplate[all[i].schema] < inTemplate[all[j].schema]
		}
		return all[i].name < all[j].name
	})

	plan := &Plan{Tables: len(all)}
	add := func(phase, schema, object, sql string) {
		plan.Statements = append(plan.Statements, Statement{Phase: phase, Schema: schema, Object: object, SQL: sql})
	}

	// 3. Preamble, extensions, schemas.
	// The plan runs with an EMPTY search_path (SET LOCAL: it ends with the transaction the plan runs in). Every definition the
	// scan recorded is schema-qualified, so nothing here depends on the path; a definition that is NOT qualified (a scan by a
	// session that could see the schema once recorded `REFERENCES party(id)`) therefore fails, loudly and whole, instead of
	// resolving to whichever table of that name the path happens to find.
	add("prelude", "", "", "-- Generated by tenantschema.Compile from alpha's catalog scan (ADR-050). Do not edit.\nSET LOCAL search_path = '';\nSET check_function_bodies = off;\n")
	extSeen := map[string]bool{}
	for _, s := range opts.Schemas {
		for _, e := range list(schemaProps[s], "extensions") {
			name := str(e, "name")
			if !allowExt[name] || extSeen[name] {
				continue
			}
			extSeen[name] = true
			add("extension", str(e, "schema"), name, fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s WITH SCHEMA %s;\n", qident(name), qident(str(e, "schema"))))
		}
	}
	for _, s := range opts.Schemas {
		add("schema", s, s, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;\n", qident(s)))
	}

	// 4. Routines (bodies are not checked at creation; see the prelude).
	for _, s := range opts.Schemas {
		for _, r := range list(schemaProps[s], "routines") {
			add("routine", s, str(r, "name")+"("+str(r, "arguments")+")", strings.TrimRight(str(r, "definition"), "\n")+";\n")
		}
	}

	// 5. Tables: columns only, parents and plain tables first, then partitions, shallowest first.
	var emitTable func(t *table)
	emitTable = func(t *table) {
		var b strings.Builder
		if t.parent != "" {
			ps, pn, _ := strings.Cut(t.parent, ".")
			fmt.Fprintf(&b, "CREATE TABLE %s PARTITION OF %s %s", qname(t.schema, t.name), qname(ps, pn), t.bound)
			if t.key != "" {
				fmt.Fprintf(&b, " PARTITION BY %s", t.key)
			}
			b.WriteString(";\n")
		} else {
			fmt.Fprintf(&b, "CREATE TABLE %s (\n", qname(t.schema, t.name))
			for i, c := range t.cols {
				fmt.Fprintf(&b, "    %s %s", qident(c.name), c.formatType)
				if c.def != "" {
					fmt.Fprintf(&b, " DEFAULT %s", c.def)
				}
				if c.notNull {
					b.WriteString(" NOT NULL")
				}
				if i < len(t.cols)-1 {
					b.WriteString(",")
				}
				b.WriteString("\n")
			}
			b.WriteString(")")
			if t.key != "" {
				fmt.Fprintf(&b, " PARTITION BY %s", t.key)
			}
			b.WriteString(";\n")
		}
		add("table", t.schema, t.name, b.String())
		sort.Slice(t.kids, func(i, j int) bool { return t.kids[i].name < t.kids[j].name })
		for _, k := range t.kids {
			emitTable(k)
		}
	}
	for _, t := range all {
		if t.parent == "" {
			emitTable(t)
		}
	}

	// 6. Constraints, in the server's own words: keys first (a foreign key needs its target's key), then checks, then
	// foreign keys last, so the order never depends on which table refers to which.
	for _, phase := range []struct{ name, types string }{{"key", "pux"}, {"check", "c"}, {"foreign key", "f"}} {
		for _, t := range all {
			for _, c := range list(t.props, "constraints") {
				if !strings.Contains(phase.types, str(c, "type")) {
					continue
				}
				add(phase.name, t.schema, str(c, "name"), fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s %s;\n", qname(t.schema, t.name), qident(str(c, "name")), str(c, "definition")))
			}
		}
	}

	// 7. Indexes. A partitioned parent's index is recorded "ON ONLY"; created that way it would be invalid and its
	// partitions would have no index, so it is created on the whole tree.
	for _, t := range all {
		for _, ix := range list(t.props, "indexes") {
			def := str(ix, "definition")
			if t.key != "" {
				def = onOnly.ReplaceAllString(def, "$1 ")
			}
			add("index", t.schema, str(ix, "name"), def+";\n")
		}
	}

	// 8. Triggers last: they call routines and fire on tables that must already exist.
	for _, t := range all {
		for _, tr := range list(t.props, "triggers") {
			add("trigger", t.schema, str(tr, "name"), str(tr, "definition")+";\n")
		}
	}
	add("epilogue", "", "", "RESET check_function_bodies;\n")
	return plan, nil
}
