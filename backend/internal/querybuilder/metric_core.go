package querybuilder

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hondyman/uisce/backend/internal/corecustom"
)

// metricContent is the canonical document representation of a metric definition
// used for diffing, version comparisons, and 3-way merges across core upgrades.
type metricContent struct {
	Name           string             `json:"name"`
	Description    string             `json:"description"`
	BOID           string             `json:"boId"`
	Expression     MetricExpression   `json:"expression"`
	GrainAllowlist []string           `json:"grainAllowlist"`
	FormatConfig   MetricFormatConfig `json:"formatConfig"`
	Variables      []MetricVariable   `json:"variables"`
	Tags           []string           `json:"tags"`
}

func (c metricContent) normalized() metricContent {
	if c.GrainAllowlist == nil {
		c.GrainAllowlist = []string{}
	}
	if c.Variables == nil {
		c.Variables = []MetricVariable{}
	}
	if c.Tags == nil {
		c.Tags = []string{}
	}
	if c.Expression.BaseMetricIDs == nil {
		c.Expression.BaseMetricIDs = []string{}
	}
	return c
}

func (c metricContent) doc() any {
	b, _ := json.Marshal(c.normalized())
	v, _ := corecustom.Decode(b)
	return v
}

func contentFromMetric(m *MetricDefinition) metricContent {
	return metricContent{
		Name:           m.Name,
		Description:    m.Description,
		BOID:           m.BOID,
		Expression:     m.Expression,
		GrainAllowlist: m.GrainAllowlist,
		FormatConfig:   m.FormatConfig,
		Variables:      m.Variables,
		Tags:           m.Tags,
	}.normalized()
}

func contentFromMetricJSON(raw []byte) (metricContent, error) {
	var c metricContent
	if err := json.Unmarshal(raw, &c); err != nil {
		return metricContent{}, err
	}
	return c.normalized(), nil
}

func contentFromMetricDoc(v any) (metricContent, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return metricContent{}, err
	}
	var c metricContent
	if err := json.Unmarshal(b, &c); err != nil {
		return metricContent{}, err
	}
	return c.normalized(), nil
}

// ValidateAdditiveMetricExtension enforces the Additive-Only Contract for Core Metrics:
// A client tenant extension cannot modify or delete core base formulas, aggregation functions,
// or bound term IDs. The client may only extend the grain allowlist, add variables, or add custom tags/formatting.
func ValidateAdditiveMetricExtension(base, ext metricContent) error {
	base = base.normalized()
	ext = ext.normalized()

	if base.BOID != ext.BOID {
		return fmt.Errorf("cannot change primary business object from %q to %q in a core metric extension", base.BOID, ext.BOID)
	}

	// 1. Immutable Expression: kind, fn, termNodeId, and formula cannot be modified by client extension
	if strings.ToLower(strings.TrimSpace(base.Expression.Kind)) != strings.ToLower(strings.TrimSpace(ext.Expression.Kind)) {
		return fmt.Errorf("cannot change core metric kind from %q to %q", base.Expression.Kind, ext.Expression.Kind)
	}
	if strings.ToLower(strings.TrimSpace(base.Expression.Fn)) != strings.ToLower(strings.TrimSpace(ext.Expression.Fn)) {
		return fmt.Errorf("cannot change core metric aggregation function from %q to %q", base.Expression.Fn, ext.Expression.Fn)
	}
	if base.Expression.TermNodeID != ext.Expression.TermNodeID {
		return fmt.Errorf("cannot change core metric underlying term ID from %q to %q", base.Expression.TermNodeID, ext.Expression.TermNodeID)
	}
	if normalizeFormulaForHash(base.Expression.Formula) != normalizeFormulaForHash(ext.Expression.Formula) {
		return fmt.Errorf("cannot modify core metric formula %q", base.Expression.Formula)
	}

	// 2. Grain Allowlist: all base grains must be present in extension (additive extension only)
	extGrains := make(map[string]bool, len(ext.GrainAllowlist))
	for _, g := range ext.GrainAllowlist {
		extGrains[strings.ToLower(strings.TrimSpace(g))] = true
	}
	for _, baseG := range base.GrainAllowlist {
		normBaseG := strings.ToLower(strings.TrimSpace(baseG))
		if !extGrains[normBaseG] {
			return fmt.Errorf("cannot remove core grain %q from metric grain allowlist", baseG)
		}
	}

	// 3. Variables: all base variables must be present in extension
	extVars := make(map[string]MetricVariable, len(ext.Variables))
	for _, v := range ext.Variables {
		extVars[v.Name] = v
	}
	for _, baseV := range base.Variables {
		extV, exists := extVars[baseV.Name]
		if !exists {
			return fmt.Errorf("cannot remove core variable %q from metric", baseV.Name)
		}
		if extV.Type != baseV.Type {
			return fmt.Errorf("cannot change type of core variable %q from %q to %q", baseV.Name, baseV.Type, extV.Type)
		}
	}

	return nil
}

// metricGrouper groups fine-grained JSON diffs into intuitive units for the corecustom 3-way merge UI.
func metricGrouper(p corecustom.Path, docs ...any) (string, string, string) {
	if len(p) == 0 {
		return "metric", "metric", "Metric"
	}
	top := p[0].Value
	switch top {
	case "name", "description":
		return "metric:details", "metric", "Metric name and description"
	case "expression":
		return "metric:expression", "expression", "Metric formula and calculation rules"
	case "grainAllowlist":
		if len(p) > 1 && p[1].Kind == corecustom.SegMember {
			return "grain:" + p[1].Value, "grain", "Allowed grain " + p[1].Value
		}
		return "grainAllowlist", "grain", "Grain allowlist"
	case "variables":
		if len(p) > 1 && p[1].Kind == corecustom.SegMember {
			return "variable:" + p[1].Value, "variable", "Variable " + p[1].Value
		}
		return "variables", "variable", "Bound variables"
	case "formatConfig":
		return "formatConfig", "format", "Metric formatting"
	case "tags":
		return "tags", "tags", "Tags"
	}
	return p.String(), "other", p.String()
}
