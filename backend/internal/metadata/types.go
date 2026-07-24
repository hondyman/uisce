package metadata

import "time"

// AttributeType defines the supported data types for attributes
type AttributeType string

const (
	AttrString  AttributeType = "string"
	AttrNumber  AttributeType = "number"
	AttrDecimal AttributeType = "decimal"
	AttrDate    AttributeType = "date"
	AttrBoolean AttributeType = "boolean"
	AttrRef     AttributeType = "ref"
)

// Version represents the semantic version of a metadata object
type Version struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch"`
}

// MetaBase contains common fields for all metadata objects
type MetaBase struct {
	ID        string     `json:"id"`
	TenantID  string     `json:"tenant_id,omitempty"` // empty for core
	Name      string     `json:"name"`
	Version   Version    `json:"version"`
	Status    string     `json:"status"` // draft, active, deprecated
	ValidFrom time.Time  `json:"valid_from"`
	ValidTo   *time.Time `json:"valid_to,omitempty"`
}

// BOAttribute defines a field on a Business Object
type BOAttribute struct {
	Name        string        `json:"name"`
	Type        AttributeType `json:"type"`
	Required    bool          `json:"required"`
	Default     *string       `json:"default,omitempty"`
	Validation  *string       `json:"validation,omitempty"`   // CEL or Rego expression
	ComputeExpr *string       `json:"compute_expr,omitempty"` // e.g., "market_value=quantity*price"
}

// BORelationship defines a relationship to another Business Object
type BORelationship struct {
	Name        string `json:"name"`
	TargetBO    string `json:"target_bo"`
	Cardinality string `json:"cardinality"` // one, many
	OnDelete    string `json:"on_delete"`   // restrict, cascade
}

// BusinessObject defines the schema of a domain entity
type BusinessObject struct {
	Meta       MetaBase         `json:"meta"`
	Attributes []BOAttribute    `json:"attributes"`
	Rels       []BORelationship `json:"rels"`
	Lifecycle  []string         `json:"lifecycle"` // list of valid states
	Policies   []string         `json:"policies"`  // entitlement refs
}

// Transition defines a valid state change in a Business Process
type Transition struct {
	From      string `json:"from"`
	To        string `json:"to"`
	GuardExpr string `json:"guard_expr,omitempty"` // CEL/Rego
	ActionRef string `json:"action_ref,omitempty"` // handler name
	SLA       string `json:"sla,omitempty"`        // ISO 8601 Duration e.g. "PT2H"
}

// BusinessProcess defines a workflow definition
type BusinessProcess struct {
	Meta        MetaBase          `json:"meta"`
	States      []string          `json:"states"`
	Transitions []Transition      `json:"transitions"`
	Bindings    map[string]string `json:"bindings"` // state->viewID
}

// ViewField defines a field in a UI View
type ViewField struct {
	Label       string  `json:"label"`
	Attr        string  `json:"attr"`
	Component   string  `json:"component"` // "Text", "Number", "Select", "Date", "Table"
	ReadOnly    bool    `json:"read_only"`
	Required    bool    `json:"required"`
	VisibleExpr *string `json:"visible_expr,omitempty"` // show/hide rule
	HelpText    *string `json:"help_text,omitempty"`
}

// UIView defines a UI screen layout
type UIView struct {
	Meta       MetaBase          `json:"meta"`
	Type       string            `json:"type"` // "Form","Table","Dashboard"
	Sections   [][]ViewField     `json:"sections"`
	DataSource string            `json:"data_source"` // BO id
	Actions    []string          `json:"actions"`     // action handlers
	Theme      map[string]string `json:"theme"`       // tokens
}

// MetricDef defines a governed metric
type MetricDef struct {
	Meta    MetaBase `json:"meta"`
	Formula string   `json:"formula"` // e.g., "NetPnL = Realized + Unrealized - Fees - Costs"
	Grain   []string `json:"grain"`   // "desk_id","trade_date"
	Filters []string `json:"filters"`
	Routing string   `json:"routing"` // "hot","cold","hybrid"
	Lineage []string `json:"lineage"` // source tables or BO attributes
}
