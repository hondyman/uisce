// Package mastering is the one entity-mastering engine: canonicalize ->
// match -> survive -> publish, configured per entity (see
// docs/mdm-mastering-blueprint.md). Product is the first entity; a
// security, benchmark or price master is a profile row plus configuration
// (business object, staging bindings, rules, match and survivorship rules),
// not new code.
//
// Two databases: the platform (alpha) holds the business object, its
// fields, validation rules and staging bindings; the tenant's data plane
// (crims) holds staging, the mdm.<prefix>_* tables and the mastering
// tables, all under forced row-level security on app.current_tenant.
package mastering

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Profile is one mastered entity (crims mdm.mastering_entity).
type Profile struct {
	ID               string          `db:"id" json:"id"`
	TenantID         string          `db:"tenant_id" json:"tenant_id"`
	EntityCd         string          `db:"entity_cd" json:"entity_cd"`
	DisplayName      string          `db:"display_name" json:"display_name"`
	BOKey            string          `db:"bo_key" json:"bo_key"`
	TablePrefix      string          `db:"table_prefix" json:"table_prefix"`
	AnchorTable      string          `db:"anchor_table" json:"anchor_table"`
	AnchorCodeColumn string          `db:"anchor_code_column" json:"anchor_code_column"`
	IdentifierTable  *string         `db:"identifier_table" json:"identifier_table,omitempty"`
	IncomingTable    string          `db:"incoming_table" json:"incoming_table"`
	CodePrefix       string          `db:"code_prefix" json:"code_prefix"`
	RawSettings      json.RawMessage `db:"settings" json:"settings"`
	IsActive         bool            `db:"is_active" json:"is_active"`
	// Inherited: the gold copy's profile, read-only for this tenant.
	Inherited bool     `db:"inherited" json:"inherited"`
	Settings  Settings `db:"-" json:"-"`
}

// Settings is the profile's settings JSON.
type Settings struct {
	// NameAttribute is the attribute fuzzy matching compares (default name).
	NameAttribute string `json:"name_attribute"`
	// References resolve coded attributes to the anchor's foreign keys.
	References []Reference `json:"references"`
	// Defaults fill attributes no source supplies when minting a record
	// (e.g. status_cd: LIVE), before references resolve.
	Defaults map[string]any `json:"defaults"`
	// RecordColumns fill required columns of the golden-record table from
	// an attribute (golden_record column -> attribute), e.g.
	// product_golden_record.product_type_cd. UNKNOWN when not supplied.
	RecordColumns map[string]string `json:"record_columns"`
}

// Reference turns a code into a foreign key of the anchor table. A source's
// vendor code is first mapped to the internal code through MapTable (per
// source system), when there is one; the internal code is what survives
// (Attribute) and Column gets its id from RefTable at publish.
//
//	{"column": "product_type_id", "attribute": "product_type_cd",
//	 "ref_table": "mdm.product_type", "ref_code_column": "type_cd",
//	 "map_table": "mdm.product_type_mapping", "map_source_column": "mdm_source_system_id",
//	 "map_vendor_column": "vendor_type_cd", "map_code_column": "internal_type_cd"}
type Reference struct {
	Column          string `json:"column"`
	Attribute       string `json:"attribute"`
	RefTable        string `json:"ref_table"`
	RefCodeColumn   string `json:"ref_code_column"`
	MapTable        string `json:"map_table,omitempty"`
	MapSourceColumn string `json:"map_source_column,omitempty"`
	MapVendorColumn string `json:"map_vendor_column,omitempty"`
	MapCodeColumn   string `json:"map_code_column,omitempty"`
	Required        bool   `json:"required,omitempty"`
}

var (
	qualified = regexp.MustCompile(`^[a-z_][a-z0-9_]*\.[a-z_][a-z0-9_]*$`)
	ident     = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)
)

// decode parses and checks the settings: every table and column the engine
// will put into SQL is validated here, once, and quoted where used.
func (p *Profile) decode() error {
	s := Settings{NameAttribute: "name"}
	if len(p.RawSettings) > 0 {
		if err := json.Unmarshal(p.RawSettings, &s); err != nil {
			return fmt.Errorf("mastering profile %s: settings: %w", p.EntityCd, err)
		}
	}
	if s.NameAttribute == "" {
		s.NameAttribute = "name"
	}
	var bad []string
	check := func(re *regexp.Regexp, names ...string) {
		for _, n := range names {
			if !re.MatchString(n) {
				bad = append(bad, n)
			}
		}
	}
	check(qualified, p.AnchorTable, p.IncomingTable)
	check(ident, p.TablePrefix, p.AnchorCodeColumn, s.NameAttribute)
	if p.IdentifierTable != nil {
		check(qualified, *p.IdentifierTable)
	}
	for col, attr := range s.RecordColumns {
		check(ident, col, attr)
	}
	for _, r := range s.References {
		check(ident, r.Column, r.Attribute, r.RefCodeColumn)
		check(qualified, r.RefTable)
		if r.MapTable != "" {
			check(qualified, r.MapTable)
			check(ident, r.MapSourceColumn, r.MapVendorColumn, r.MapCodeColumn)
		}
	}
	if len(bad) > 0 {
		return fmt.Errorf("mastering profile %s: not valid table or column names: %s", p.EntityCd, strings.Join(bad, ", "))
	}
	p.Settings = s
	return nil
}

// table is mdm.<prefix>_<suffix>, in the anchor's schema.
func (p *Profile) table(suffix string) string {
	schema := strings.SplitN(p.AnchorTable, ".", 2)[0]
	return qi(schema) + "." + qi(p.TablePrefix+"_"+suffix)
}

// keyColumn is the anchor id column in the mdm.<prefix>_* tables.
func (p *Profile) keyColumn() string { return qi(p.TablePrefix + "_id") }

// referenceFor returns the reference whose foreign-key column is col.
func (p *Profile) referenceFor(col string) (Reference, bool) {
	for _, r := range p.Settings.References {
		if r.Column == col {
			return r, true
		}
	}
	return Reference{}, false
}

// qi quotes an identifier or schema-qualified name already validated by
// decode.
func qi(name string) string {
	parts := strings.Split(name, ".")
	for i, s := range parts {
		parts[i] = `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return strings.Join(parts, ".")
}
