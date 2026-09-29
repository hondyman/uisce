package mastering

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hondyman/uisce/backend/internal/stagingbind"
)

// Record is one source record in canonical form: attributes named as the
// golden (anchor) columns they master, the source's identifiers, and the
// source's own key and as-of time.
type Record struct {
	SourceKey   string            `json:"source_key"`
	AsOf        time.Time         `json:"as_of"`
	Attrs       map[string]any    `json:"attrs"`
	Identifiers map[string]string `json:"identifiers,omitempty"`
	Issues      []Issue           `json:"issues,omitempty"`
}

// Issue is a finding on a record or golden record. ERROR keeps a record out
// of mastering (or a golden version from publishing); WARNING doesn't.
type Issue struct {
	Code      string `json:"code"`
	Severity  string `json:"severity"` // ERROR | WARNING
	Attribute string `json:"attribute,omitempty"`
	RuleID    string `json:"rule_id,omitempty"`
	Message   string `json:"message"`
}

const (
	SevError   = "ERROR"
	SevWarning = "WARNING"
)

// Valid reports a record with no ERROR issue.
func (r *Record) Valid() bool {
	for _, i := range r.Issues {
		if i.Severity == SevError {
			return false
		}
	}
	return true
}

func (r *Record) add(code, sev, attr, msg string) {
	r.Issues = append(r.Issues, Issue{Code: code, Severity: sev, Attribute: attr, Message: msg})
}

// Canonicalizer turns staging rows into records through a staging binding.
type Canonicalizer struct {
	Profile *Profile
	// Binding is the staging binding: BO field or mastering key -> column.
	Binding map[string]string
	// FieldAttr maps a bound field name to the golden column it masters (its
	// MAPS_TO column on the anchor table). A binding may name the field either
	// by BO field ("SecName") or by golden attribute ("security_name"), so
	// both keys are present and both resolve to the golden column.
	FieldAttr map[string]string
	// ColumnTypes are the staging columns' data types (information_schema),
	// so numeric text from the driver becomes a number.
	ColumnTypes map[string]string
}

// Unmastered lists bound BO fields with no golden column: they are checked
// by staging rules but not mastered, and are reported once per run.
func (c *Canonicalizer) Unmastered() []string {
	var out []string
	for k := range c.Binding {
		if stagingbind.IsMasteringKey(k) {
			continue
		}
		if _, ok := c.FieldAttr[k]; !ok {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// Record canonicalizes one staging row. ingestedAt is the as-of time when
// the binding has no @as_of column.
func (c *Canonicalizer) Record(row map[string]any, ingestedAt time.Time) Record {
	rec := Record{Attrs: map[string]any{}, Identifiers: map[string]string{}, AsOf: ingestedAt}

	if col, ok := c.Binding[stagingbind.SourceKey]; ok {
		rec.SourceKey = strings.TrimSpace(text(row[col]))
	}
	if rec.SourceKey == "" {
		rec.add("MISSING_SOURCE_KEY", SevError, "", "the source record has no key (bind @source_key)")
	}
	if col, ok := c.Binding[stagingbind.AsOfKey]; ok {
		if t, ok := asTime(row[col]); ok {
			rec.AsOf = t
		}
	}

	for key, col := range c.Binding {
		if typ, ok := stagingbind.IdentifierType(key); ok {
			if v := strings.ToUpper(strings.TrimSpace(text(row[col]))); v != "" {
				rec.Identifiers[typ] = v
			}
			continue
		}
		if stagingbind.IsMasteringKey(key) {
			continue
		}
		attr, ok := c.FieldAttr[key]
		if !ok {
			continue
		}
		v := normalize(row[col], c.ColumnTypes[col])
		if v == nil {
			continue
		}
		// A foreign key arrives as the source's code; it survives as the
		// internal code (resolved later) under the reference's attribute.
		if ref, isRef := c.Profile.referenceFor(attr); isRef {
			rec.Attrs[ref.Attribute] = text(v)
			continue
		}
		rec.Attrs[attr] = v
	}
	return rec
}

// normalize makes a driver value comparable across sources: trimmed text
// (empty is absent), numbers for numeric columns, dates as YYYY-MM-DD.
func normalize(v any, dataType string) any {
	switch x := v.(type) {
	case nil:
		return nil
	case []byte:
		return normalize(string(x), dataType)
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return nil
		}
		if isNumericType(dataType) {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				return f
			}
		}
		return s
	case time.Time:
		if dataType == "date" || (x.Hour() == 0 && x.Minute() == 0 && x.Second() == 0 && x.Nanosecond() == 0) {
			return x.Format("2006-01-02")
		}
		return x.UTC().Format(time.RFC3339)
	case int64:
		return float64(x)
	case int32:
		return float64(x)
	case int:
		return float64(x)
	case float32:
		return float64(x)
	default:
		return v
	}
}

func isNumericType(t string) bool {
	switch t {
	case "numeric", "integer", "bigint", "smallint", "real", "double precision", "decimal":
		return true
	}
	return false
}

func text(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case []byte:
		return string(x)
	case time.Time:
		return x.Format("2006-01-02")
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return fmt.Sprint(x)
	}
}

func asTime(v any) (time.Time, bool) {
	switch x := v.(type) {
	case time.Time:
		return x, true
	case []byte:
		return asTime(string(x))
	case string:
		for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02"} {
			if t, err := time.Parse(layout, strings.TrimSpace(x)); err == nil {
				return t, true
			}
		}
	}
	return time.Time{}, false
}
