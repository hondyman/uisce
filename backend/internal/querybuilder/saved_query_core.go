package querybuilder

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hondyman/uisce/backend/internal/corecustom"
)

// savedQueryContent is the canonical document representation of a saved query
// used for diffing, version comparisons, and 3-way merges across core upgrades.
type savedQueryContent struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	BOID         string          `json:"boId"`
	BindingID    string          `json:"bindingId"`
	RelatedBOIDs []string        `json:"relatedBoIds"`
	ChartType    string          `json:"chartType"`
	State        SavedQueryState `json:"state"`
	Tags         []string        `json:"tags"`
}

// normalized ensures slice fields are non-nil empty slices so JSON serialization
// and comparison is deterministic.
func (c savedQueryContent) normalized() savedQueryContent {
	if c.RelatedBOIDs == nil {
		c.RelatedBOIDs = []string{}
	}
	if c.Tags == nil {
		c.Tags = []string{}
	}
	if c.State.Dimensions == nil {
		c.State.Dimensions = []SavedQueryDimension{}
	}
	if c.State.Measures == nil {
		c.State.Measures = []SavedQueryMeasure{}
	}
	if c.State.Filters == nil {
		c.State.Filters = []SavedQueryFilter{}
	}
	if c.State.Parameters == nil {
		c.State.Parameters = []SavedQueryParameter{}
	}
	return c
}

func (c savedQueryContent) doc() any {
	b, _ := json.Marshal(c.normalized())
	v, _ := corecustom.Decode(b)
	return v
}

func contentFromSavedQuery(sq *SavedQuery) savedQueryContent {
	return savedQueryContent{
		Name:         sq.Name,
		Description:  sq.Description,
		BOID:         sq.BOID,
		BindingID:    sq.BindingID,
		RelatedBOIDs: sq.RelatedBOIDs,
		ChartType:    sq.ChartType,
		State:        sq.State,
		Tags:         sq.Tags,
	}.normalized()
}

func contentFromJSON(raw []byte) (savedQueryContent, error) {
	var c savedQueryContent
	if err := json.Unmarshal(raw, &c); err != nil {
		return savedQueryContent{}, err
	}
	return c.normalized(), nil
}

func contentFromDoc(v any) (savedQueryContent, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return savedQueryContent{}, err
	}
	var c savedQueryContent
	if err := json.Unmarshal(b, &c); err != nil {
		return savedQueryContent{}, err
	}
	return c.normalized(), nil
}

// ValidateAdditiveExtension enforces the strict Additive-Only Contract:
// A client tenant extension cannot delete, rename, or modify core dimensions,
// core measures (including aggregations), core base filters, core parameters,
// or core joined Business Objects. The client may only extend the query.
func ValidateAdditiveExtension(base, ext savedQueryContent) error {
	base = base.normalized()
	ext = ext.normalized()

	if base.BOID != ext.BOID {
		return fmt.Errorf("cannot change primary business object from %q to %q in a core extension", base.BOID, ext.BOID)
	}

	// 1. Related BOs: all base related BOs must be present in extension.
	extBOs := make(map[string]bool, len(ext.RelatedBOIDs))
	for _, id := range ext.RelatedBOIDs {
		extBOs[id] = true
	}
	for _, id := range base.RelatedBOIDs {
		if !extBOs[id] {
			return fmt.Errorf("cannot remove core joined business object %q in an extension", id)
		}
	}

	// 2. Dimensions: all base dimensions must be retained identically.
	baseDimByAlias := make(map[string]SavedQueryDimension, len(base.State.Dimensions))
	for _, d := range base.State.Dimensions {
		baseDimByAlias[d.Alias] = d
	}
	extDimByAlias := make(map[string]SavedQueryDimension, len(ext.State.Dimensions))
	for _, d := range ext.State.Dimensions {
		if _, exists := extDimByAlias[d.Alias]; exists {
			return fmt.Errorf("duplicate dimension alias %q in extension", d.Alias)
		}
		extDimByAlias[d.Alias] = d
	}

	for alias, baseDim := range baseDimByAlias {
		extDim, found := extDimByAlias[alias]
		if !found {
			return fmt.Errorf("cannot remove core dimension with alias %q (term %q)", alias, baseDim.TermNodeID)
		}
		if extDim.TermNodeID != baseDim.TermNodeID || extDim.BOID != baseDim.BOID {
			return fmt.Errorf("cannot modify core dimension %q (expected term %q, got %q)", alias, baseDim.TermNodeID, extDim.TermNodeID)
		}
	}

	// 3. Measures: all base measures must be retained identically (same alias, term, aggregation).
	baseMeasByAlias := make(map[string]SavedQueryMeasure, len(base.State.Measures))
	for _, m := range base.State.Measures {
		baseMeasByAlias[m.Alias] = m
	}
	extMeasByAlias := make(map[string]SavedQueryMeasure, len(ext.State.Measures))
	for _, m := range ext.State.Measures {
		if _, exists := extMeasByAlias[m.Alias]; exists {
			return fmt.Errorf("duplicate measure alias %q in extension", m.Alias)
		}
		extMeasByAlias[m.Alias] = m
	}

	for alias, baseMeas := range baseMeasByAlias {
		extMeas, found := extMeasByAlias[alias]
		if !found {
			return fmt.Errorf("cannot remove core measure with alias %q (term %q)", alias, baseMeas.TermNodeID)
		}
		if extMeas.TermNodeID != baseMeas.TermNodeID || extMeas.BOID != baseMeas.BOID {
			return fmt.Errorf("cannot modify core measure %q term (expected %q, got %q)", alias, baseMeas.TermNodeID, extMeas.TermNodeID)
		}
		if extMeas.Aggregation != baseMeas.Aggregation {
			return fmt.Errorf("cannot modify core measure %q aggregation from %q to %q", alias, baseMeas.Aggregation, extMeas.Aggregation)
		}
	}

	// 4. Alias collision check between newly added dimensions and measures
	for alias := range extDimByAlias {
		if _, inMeas := extMeasByAlias[alias]; inMeas {
			return fmt.Errorf("alias collision: %q cannot be both a dimension and a measure", alias)
		}
	}

	// 5. Parameters: all base parameters must be retained with same name, type, and requirement.
	baseParamByName := make(map[string]SavedQueryParameter, len(base.State.Parameters))
	for _, p := range base.State.Parameters {
		baseParamByName[p.Name] = p
	}
	extParamByName := make(map[string]SavedQueryParameter, len(ext.State.Parameters))
	for _, p := range ext.State.Parameters {
		if _, exists := extParamByName[p.Name]; exists {
			return fmt.Errorf("duplicate parameter name %q in extension", p.Name)
		}
		extParamByName[p.Name] = p
	}

	for name, baseP := range baseParamByName {
		extP, found := extParamByName[name]
		if !found {
			return fmt.Errorf("cannot remove core parameter %q", name)
		}
		if extP.Type != baseP.Type {
			return fmt.Errorf("cannot change type of core parameter %q from %q to %q", name, baseP.Type, extP.Type)
		}
		if baseP.Required && !extP.Required {
			return fmt.Errorf("cannot make required core parameter %q optional", name)
		}
	}

	// 6. Filters: all base filters must be retained identically.
	for _, baseF := range base.State.Filters {
		matched := false
		for _, extF := range ext.State.Filters {
			if filterEquals(baseF, extF) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("cannot remove or modify core filter on term %q (operator %q)", baseF.TermNodeID, baseF.Operator)
		}
	}

	return nil
}

func filterEquals(a, b SavedQueryFilter) bool {
	if a.TermNodeID != b.TermNodeID || a.Operator != b.Operator || a.ParamRef != b.ParamRef || a.BOID != b.BOID {
		return false
	}
	return reflect.DeepEqual(canonicalizeFilterValue(a.Value), canonicalizeFilterValue(b.Value))
}

func canonicalizeFilterValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	// Handle numbers decoded from JSON
	switch val := v.(type) {
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return i
		}
		if f, err := val.Float64(); err == nil {
			return f
		}
		return val.String()
	case float64:
		if float64(int64(val)) == val {
			return int64(val)
		}
		return val
	case []interface{}:
		res := make([]interface{}, len(val))
		for i, elem := range val {
			res[i] = canonicalizeFilterValue(elem)
		}
		return res
	default:
		return v
	}
}

// savedQueryGrouper groups fine-grained JSON diffs into intuitive units for
// the corecustom.Compare and corecustom.Merge 3-way merge review UI.
func savedQueryGrouper(p corecustom.Path, docs ...any) (string, string, string) {
	if len(p) == 0 {
		return "query", "query", "Query"
	}
	top := p[0].Value
	switch top {
	case "name", "description", "chartType":
		return "query:details", "query", "Query name and presentation"
	case "relatedBoIds":
		if len(p) > 1 && p[1].Kind == corecustom.SegMember {
			return "relatedBo:" + p[1].Value, "relatedBo", "Joined Business Object " + p[1].Value
		}
		return "relatedBoIds", "relatedBo", "Joined Business Objects"
	case "tags":
		return "tags", "tags", "Tags"
	case "state":
		if len(p) < 2 {
			return "state", "query", "Query state"
		}
		sub := p[1].Value
		switch sub {
		case "dimensions":
			if len(p) > 2 && p[2].Kind == corecustom.SegElem {
				alias := p[2].Value
				return "dimension:" + alias, "dimension", "Dimension “" + alias + "”"
			}
			return "dimensions", "dimension", "Dimensions"
		case "measures":
			if len(p) > 2 && p[2].Kind == corecustom.SegElem {
				alias := p[2].Value
				return "measure:" + alias, "measure", "Measure “" + alias + "”"
			}
			return "measures", "measure", "Measures"
		case "filters":
			if len(p) > 2 && p[2].Kind == corecustom.SegElem {
				term := p[2].Value
				return "filter:" + term, "filter", "Filter on " + term
			}
			return "filters", "filter", "Filters"
		case "parameters":
			if len(p) > 2 && p[2].Kind == corecustom.SegElem {
				name := p[2].Value
				return "parameter:" + name, "parameter", "Parameter “" + name + "”"
			}
			return "parameters", "parameter", "Parameters"
		case "sorts":
			return "sorts", "sort", "Sort order"
		case "limit":
			return "limit", "limit", "Row limit"
		}
	}
	return p.String(), "other", p.String()
}
