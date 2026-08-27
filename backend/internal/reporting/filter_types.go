package reporting

// FilterOperator represents the type of comparison applied by a filter.
type FilterOperator string

const (
	// Standard comparison operators
	OpEquals       FilterOperator = "equals"
	OpNotEquals    FilterOperator = "not_equals"
	OpGreaterThan  FilterOperator = "greater_than"
	OpLessThan    FilterOperator = "less_than"
	OpGreaterEqual FilterOperator = "greater_equal"
	OpLessEqual   FilterOperator = "less_equal"
	OpBetween     FilterOperator = "between"
	OpNotBetween  FilterOperator = "not_between"
	OpIn          FilterOperator = "in"
	OpNotIn       FilterOperator = "not_in"
	OpIsNull      FilterOperator = "is_null"
	OpIsNotNull   FilterOperator = "is_not_null"
	OpContains    FilterOperator = "contains"
	OpStartsWith  FilterOperator = "starts_with"
	OpEndsWith   FilterOperator = "ends_with"

	// Calendar-aware operators
	OpIsBusinessDay       FilterOperator = "is_business_day"
	OpIsHoliday           FilterOperator = "is_holiday"
	OpNextBusinessDay     FilterOperator = "next_business_day"
	OpPreviousBusinessDay FilterOperator = "previous_business_day"
	OpAddBusinessDays     FilterOperator = "add_business_days"

	// Relative date operators
	OpToday          FilterOperator = "today"
	OpYesterday      FilterOperator = "yesterday"
	OpTomorrow       FilterOperator = "tomorrow"
	OpStartOfWeek    FilterOperator = "start_of_week"
	OpEndOfWeek     FilterOperator = "end_of_week"
	OpStartOfMonth   FilterOperator = "start_of_month"
	OpEndOfMonth    FilterOperator = "end_of_month"
	OpStartOfQuarter FilterOperator = "start_of_quarter"
	OpEndOfQuarter   FilterOperator = "end_of_quarter"
	OpStartOfYear    FilterOperator = "start_of_year"
	OpEndOfYear     FilterOperator = "end_of_year"
	OpLastNDays     FilterOperator = "last_n_days"
	OpLastNBusinessDays FilterOperator = "last_n_business_days"
	OpNextNBusinessDays FilterOperator = "next_n_business_days"

	// Custom offset operators
	OpPrevious FilterOperator = "previous"
	OpNext     FilterOperator = "next"
)

// ValueSourceKind categorizes where a filter's value originates.
type ValueSourceKind string

const (
	ValueSourceConstant        ValueSourceKind = "constant"
	ValueSourceParameter      ValueSourceKind = "parameter"
	ValueSourceFunction       ValueSourceKind = "function"
	ValueSourceTenantDefault  ValueSourceKind = "tenant_default"
	ValueSourceInstanceDefault ValueSourceKind = "instance_default"
	ValueSourceCalendar       ValueSourceKind = "calendar"
)

// ValueSource describes the origin of a filter's value.
type ValueSource struct {
	Kind          ValueSourceKind `json:"kind"`
	Value         string          `json:"value,omitempty"`
	ParameterID   string          `json:"parameterId,omitempty"`
	ParameterName string          `json:"parameterName,omitempty"`
	Expression    string          `json:"expression,omitempty"`
	DefaultKey    string          `json:"defaultKey,omitempty"`
	CalendarCode  string          `json:"calendarCode,omitempty"`
}

// Filter represents a single predicate on a field.
type Filter struct {
	ID              string          `json:"id"`
	Field           string          `json:"field"`
	FieldScope      string          `json:"fieldScope,omitempty"`       // 'root' | 'subtype'
	FieldSubtypeKey string          `json:"fieldSubtypeKey,omitempty"` // subtype key when scope is 'subtype'
	Operator        FilterOperator  `json:"operator"`
	ValueSource     ValueSource     `json:"valueSource"`
	Values          []string       `json:"values,omitempty"` // used by 'in', 'between', 'last_n_days', etc.
	Enabled         bool           `json:"enabled"`
}

// FilterGroup is a collection of filters combined with AND or OR.
type FilterGroup struct {
	ID         string   `json:"id"`
	Combinator string   `json:"combinator"` // 'AND' | 'OR'
	Filters    []Filter `json:"filters"`
}

// FilterModel is the complete filter definition for a report.
type FilterModel struct {
	Groups          []FilterGroup `json:"groups"`
	GroupCombinator string        `json:"groupCombinator"` // 'AND' | 'OR' — how groups combine with each other
}

// TenantDefaults carries a tenant's configured default values.
type TenantDefaults struct {
	DefaultCalendarCode string `json:"defaultCalendarCode"`
	DefaultFiscalYear   int    `json:"defaultFiscalYear"`
	DefaultRegion       string `json:"defaultRegion"`
}

// TenantCalendar describes a calendar available to a tenant.
type TenantCalendar struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}
