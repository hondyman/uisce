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
	// Kind: RECORD (one golden record per entity, mastered record at a
	// time) or TIMESERIES (one golden value per entity x series x date,
	// mastered set-based per valuation date - the price master).
	Kind string `db:"kind" json:"kind"`
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
	// FieldGroups puts attributes in the source-priority field groups of
	// mdm.<prefix>_source_priority (name -> NAME); an attribute in no group
	// is ranked by DefaultFieldGroup. This ranking is every attribute's
	// default when it has no survivorship rule.
	FieldGroups       map[string]string `json:"field_groups"`
	DefaultFieldGroup string            `json:"default_field_group"`

	// BOBinding names the business object's binding over the anchor, when
	// the anchor isn't the BO's default binding (e.g. the Security BO: OMS
	// instrument by default, "Security Master Binding" for the master). Its
	// field bindings map BO fields to anchor columns.
	BOBinding string `json:"bo_binding,omitempty"`

	// EntityIDColumn is the anchor column holding the golden record's
	// stable id - what golden records, identifiers and the cross-reference
	// point at (default: id). A bitemporal anchor needs one that survives
	// versions (e.g. master_id), since each version is a new row.
	EntityIDColumn string `json:"entity_id_column"`
	// Versioning of the anchor: "in_place" (default: rows are updated) or
	// "bitemporal" (a change closes the current row - valid_to - and
	// inserts the new version; only rows with valid_to null are current).
	Versioning string `json:"versioning"`
	// Identifiers describes the identifier table when its columns differ
	// from the default (<prefix>_id, id_type, id_value, source code,
	// effective_to, is_primary).
	Identifiers *IdentifierSpec `json:"identifiers,omitempty"`
	// Derived fills a required anchor column when minting from the first
	// available of: id:<TYPE> (the record's identifier), @code (the minted
	// code) or an attribute name. E.g. primary_identifier: [id:ISIN, id:FIGI, @code].
	Derived map[string][]string `json:"derived,omitempty"`
	// Required attributes a record must have to mint a golden record
	// (a clear exception instead of a database error).
	Required []string `json:"required,omitempty"`
	// MergedValues are set on the anchor of a record merged into another
	// (e.g. status: MERGED) - besides merged_into_id when the anchor has it.
	MergedValues map[string]any `json:"merged_values,omitempty"`

	// Series configures a TIMESERIES profile.
	Series *SeriesSettings `json:"series,omitempty"`
}

// Kinds of profile.
const (
	KindRecord     = "RECORD"
	KindTimeSeries = "TIMESERIES"
)

// SeriesSettings configure a time-series (price) profile: what its values
// are observed on and how a vendor row becomes observations.
type SeriesSettings struct {
	// EntityType is the price entity type observations are for (SECURITY).
	EntityType string `json:"entity_type"`
	// Instrument is the record profile whose golden records observations
	// resolve to, through its identifier table (SECURITY).
	Instrument string `json:"instrument"`
	// IdentifierOrder: which identifier resolves a quote when a vendor row
	// carries several (the first that resolves wins).
	IdentifierOrder []string `json:"identifier_order"`
	// BO fields a binding maps: the valuation date (required), the
	// currency, and for vendor files with one row per price the value and
	// its price type (else value:<TYPE> keys, one column per price type).
	DateField     string `json:"date_field"`
	CurrencyField string `json:"currency_field"`
	ValueField    string `json:"value_field"`
	TypeField     string `json:"type_field"`
	// ObservationType recorded on each observation (EOD).
	ObservationType string         `json:"observation_type"`
	Controls        SeriesControls `json:"controls"`
}

// SeriesControls are the price-validation controls.
type SeriesControls struct {
	// DayOverDay: what an error or critical move against the prior date's
	// golden value does - FLAG (publish, with an exception) or HOLD (the
	// price waits for a steward). Warnings are always flagged.
	DayOverDay struct {
		Error    string `json:"error"`
		Critical string `json:"critical"`
	} `json:"day_over_day"`
	// CrossSource records a variance event for each source that disagrees
	// with the golden value beyond the warning threshold.
	CrossSource bool `json:"cross_source"`
	// CurrencyMismatch: EXCLUDE a quote in another currency than the
	// instrument's from survivorship (flagged), or FLAG it only.
	CurrencyMismatch string `json:"currency_mismatch"`
	// RelatedTypes are price types that should agree (e.g. LAST and
	// OFFICIAL_CLOSE; BID, MID and ASK): a price only one source quotes is
	// checked against the golden prices of its related types for the same
	// instrument and date - beyond the error threshold it is flagged, beyond
	// critical held. Default: [[LAST, OFFICIAL_CLOSE], [BID, MID, ASK]].
	RelatedTypes [][]string `json:"related_types,omitempty"`
}

func (c SeriesControls) related(priceType string) []string {
	groups := c.RelatedTypes
	if groups == nil {
		groups = [][]string{{"LAST", "OFFICIAL_CLOSE"}, {"BID", "MID", "ASK"}}
	}
	for _, g := range groups {
		for _, t := range g {
			if t == priceType {
				out := make([]string, 0, len(g)-1)
				for _, o := range g {
					if o != priceType {
						out = append(out, o)
					}
				}
				return out
			}
		}
	}
	return nil
}

func (p *Profile) timeSeries() bool { return p.Kind == KindTimeSeries }

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

// IdentifierSpec is an identifier table's layout.
type IdentifierSpec struct {
	KeyColumn    string `json:"key_column"`    // the golden id column
	TypeColumn   string `json:"type_column"`   // e.g. id_type
	ValueColumn  string `json:"value_column"`  // e.g. id_value
	SourceColumn string `json:"source_column"` // who reported it
	SourceIsID   bool   `json:"source_is_id"`  // the source column holds the source system id (else its code)
	// Active: an end-date column (active while null) or a flag column
	// (active while true).
	ActiveColumn  string `json:"active_column"`
	ActiveIsFlag  bool   `json:"active_is_flag"`
	PrimaryColumn string `json:"primary_column,omitempty"`
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
	if s.EntityIDColumn != "" {
		check(ident, s.EntityIDColumn)
	}
	if s.Versioning != "" && s.Versioning != "in_place" && s.Versioning != "bitemporal" {
		bad = append(bad, "versioning "+s.Versioning)
	}
	if s.Identifiers != nil {
		i := s.Identifiers
		check(ident, i.KeyColumn, i.TypeColumn, i.ValueColumn, i.SourceColumn, i.ActiveColumn)
		if i.PrimaryColumn != "" {
			check(ident, i.PrimaryColumn)
		}
	}
	for col := range s.Derived {
		check(ident, col)
	}
	for col := range s.MergedValues {
		check(ident, col)
	}
	for _, r := range s.References {
		check(ident, r.Column, r.Attribute, r.RefCodeColumn)
		check(qualified, r.RefTable)
		if r.MapTable != "" {
			check(qualified, r.MapTable)
			check(ident, r.MapSourceColumn, r.MapVendorColumn, r.MapCodeColumn)
		}
	}
	if p.Kind == "" {
		p.Kind = KindRecord
	}
	if p.Kind == KindTimeSeries {
		if s.Series == nil {
			return fmt.Errorf("mastering profile %s: a time-series profile needs settings.series", p.EntityCd)
		}
		sr := s.Series
		if sr.EntityType == "" {
			sr.EntityType = "SECURITY"
		}
		if sr.DateField == "" {
			sr.DateField = "ValuationDate"
		}
		if sr.ObservationType == "" {
			sr.ObservationType = "EOD"
		}
		if sr.Instrument == "" {
			bad = append(bad, "series.instrument (empty)")
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

// plainTable is table() unquoted, for to_regclass (names are validated).
func (p *Profile) plainTable(suffix string) string {
	return strings.SplitN(p.AnchorTable, ".", 2)[0] + "." + p.TablePrefix + "_" + suffix
}

// entityCol is the anchor column holding the golden record's stable id.
func (p *Profile) entityCol() string {
	if p.Settings.EntityIDColumn != "" {
		return p.Settings.EntityIDColumn
	}
	return "id"
}

func (p *Profile) bitemporal() bool { return p.Settings.Versioning == "bitemporal" }

// current is the condition for an anchor row being the current version
// (always true for an in-place anchor), for alias ("" = unqualified).
func (p *Profile) current(alias string) string {
	if !p.bitemporal() {
		return "true"
	}
	if alias != "" {
		alias += "."
	}
	return alias + `"valid_to" IS NULL`
}

// ident is the identifier table's layout (the Product layout by default).
func (p *Profile) ident() IdentifierSpec {
	if s := p.Settings.Identifiers; s != nil {
		return *s
	}
	return IdentifierSpec{KeyColumn: p.TablePrefix + "_id", TypeColumn: "id_type", ValueColumn: "id_value",
		SourceColumn: "source", ActiveColumn: "effective_to", PrimaryColumn: "is_primary"}
}

// identActive is the condition for an identifier row being in force.
func (i IdentifierSpec) identActive(alias string) string {
	if alias != "" {
		alias += "."
	}
	if i.ActiveIsFlag {
		return alias + qi(i.ActiveColumn)
	}
	return alias + qi(i.ActiveColumn) + " IS NULL"
}

// identRetire ends an identifier row.
func (i IdentifierSpec) identRetire() string {
	if i.ActiveIsFlag {
		return qi(i.ActiveColumn) + " = false"
	}
	return qi(i.ActiveColumn) + " = CURRENT_DATE"
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
