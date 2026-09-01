package compliance

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type WizardRuleDefinition struct {
	TenantID            uuid.UUID `json:"tenantId"`
	RuleName            string    `json:"ruleName"`
	TestType            string    `json:"testType"`
	NumeratorMetric     string    `json:"numeratorMetric"`
	DenominatorMetric   string    `json:"denominatorMetric"`
	EvaluateForEach     bool      `json:"evaluateForEach"`
	GroupByDimension    string    `json:"groupByDimension"`
	WhereExpression     string    `json:"whereExpression"`
	AlertThresholdAbove *float64  `json:"alertThresholdAbove"`
	AlertThresholdBelow *float64  `json:"alertThresholdBelow"`
	WarnThresholdAbove  *float64  `json:"warnThresholdAbove"`
	WarnThresholdBelow  *float64  `json:"warnThresholdBelow"`
	ReopenTolerance     float64   `json:"reopenTolerance"`
}

type CompiledComplianceAST struct {
	RuleCode       string          `json:"ruleCode"`
	Category       string          `json:"category"`
	NumeratorAST   json.RawMessage `json:"numeratorAst"`
	DenominatorAST json.RawMessage `json:"denominatorAst"`
	WhereFilterAST json.RawMessage `json:"whereFilterAst"`
	GoogleCELCode  string          `json:"googleCelCode"`
	Operator       string          `json:"operator"`
	ThresholdVal   float64         `json:"thresholdVal"`
	Tolerance      float64         `json:"tolerance"`
}

type ComplianceRuleCompiler struct{}

func NewComplianceRuleCompiler() *ComplianceRuleCompiler {
	return &ComplianceRuleCompiler{}
}

// CompileWizardToAST compiles CRIMS wizard parameters into Google CEL & FastRecord AST
func (c *ComplianceRuleCompiler) CompileWizardToAST(def WizardRuleDefinition) (*CompiledComplianceAST, error) {
	ruleCode := fmt.Sprintf("RULE_%s", def.GroupByDimension)
	operator := "<="
	var threshold float64

	if def.AlertThresholdAbove != nil {
		operator = "<="
		threshold = *def.AlertThresholdAbove
	} else if def.AlertThresholdBelow != nil {
		operator = ">="
		threshold = *def.AlertThresholdBelow
	}

	numAST, _ := json.Marshal(map[string]interface{}{
		"metric": def.NumeratorMetric,
		"agg":    "SUM",
	})

	denAST, _ := json.Marshal(map[string]interface{}{
		"metric": def.DenominatorMetric,
	})

	whereAST, _ := json.Marshal(map[string]interface{}{
		"raw_clause": def.WhereExpression,
		"parsed_op":  "NOT_EQUAL",
		"target":     "USGOV",
	})

	celExpr := fmt.Sprintf(
		"filter(holdings, h, %s).sum(h.market_value) / account.%s %s %.4f",
		def.WhereExpression, def.DenominatorMetric, operator, threshold/100.0,
	)

	return &CompiledComplianceAST{
		RuleCode:       ruleCode,
		Category:       "CONCENTRATION",
		NumeratorAST:   numAST,
		DenominatorAST: denAST,
		WhereFilterAST: whereAST,
		GoogleCELCode:  celExpr,
		Operator:       operator,
		ThresholdVal:   threshold,
		Tolerance:      def.ReopenTolerance,
	}, nil
}
