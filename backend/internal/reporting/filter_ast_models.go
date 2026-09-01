package reporting

import (
	"time"

	"github.com/google/uuid"
)

type FilterNodeType string

const (
	NodeGroup      FilterNodeType = "GROUP"
	NodeComparison FilterNodeType = "COMPARISON"
	NodeFunction   FilterNodeType = "FUNCTION"
	NodeParameter  FilterNodeType = "PARAMETER"
	NodeLiteral    FilterNodeType = "LITERAL"
	NodeField      FilterNodeType = "FIELD"
)

type CombinatorType string

const (
	CombinatorAnd CombinatorType = "AND"
	CombinatorOr  CombinatorType = "OR"
)

type FilterComparisonOp string

const (
	CmpOpEquals              FilterComparisonOp = "EQUALS"
	CmpOpNotEquals           FilterComparisonOp = "NOT_EQUALS"
	CmpOpGreaterThan         FilterComparisonOp = "GREATER_THAN"
	CmpOpGreaterThanOrEquals FilterComparisonOp = "GREATER_THAN_OR_EQUALS"
	CmpOpLessThan            FilterComparisonOp = "LESS_THAN"
	CmpOpLessThanOrEquals    FilterComparisonOp = "LESS_THAN_OR_EQUALS"
	CmpOpBetween             FilterComparisonOp = "BETWEEN"
	CmpOpIn                  FilterComparisonOp = "IN"
	CmpOpNotIn               FilterComparisonOp = "NOT_IN"
	CmpOpContains            FilterComparisonOp = "CONTAINS"
	CmpOpStartsWith          FilterComparisonOp = "STARTS_WITH"
	CmpOpEndsWith            FilterComparisonOp = "ENDS_WITH"
	CmpOpIsNull              FilterComparisonOp = "IS_NULL"
	CmpOpIsNotNull           FilterComparisonOp = "IS_NOT_NULL"
	CmpOpMatchesRegex        FilterComparisonOp = "MATCHES_REGEX"
)

// Legacy alias definitions if needed
const (
	OpGreaterThanOrEquals = CmpOpGreaterThanOrEquals
	OpLessThanOrEquals    = CmpOpLessThanOrEquals
	OpMatchesRegex        = CmpOpMatchesRegex
)

type FilterExpressionNode struct {
	Type       FilterNodeType         `json:"type"`
	Combinator CombinatorType         `json:"combinator,omitempty"` // For GROUP
	Children   []FilterExpressionNode `json:"children,omitempty"`   // For GROUP

	// Comparison details
	LeftExpression  *FilterExpressionNode  `json:"leftExpression,omitempty"`
	Operator        FilterComparisonOp     `json:"operator,omitempty"`
	RightExpression *FilterExpressionNode  `json:"rightExpression,omitempty"`
	RightValues     []FilterExpressionNode `json:"rightValues,omitempty"` // For BETWEEN, IN

	// Leaf properties
	FieldKey     string                 `json:"fieldKey,omitempty"`     // e.g. "account_bk"
	FunctionName string                 `json:"functionName,omitempty"` // e.g. "SUBSTR", "UPPER", "PREVIOUS_BUSINESS_DAY"
	FunctionArgs []FilterExpressionNode `json:"functionArgs,omitempty"`
	ParamKey     string                 `json:"paramKey,omitempty"`     // e.g. "@AsOfDate", "@Session.TenantID"
	LiteralValue interface{}            `json:"literalValue,omitempty"`
	DataType     string                 `json:"dataType,omitempty"`
	IsEnabled    bool                   `json:"isEnabled"`
}

type FilterExecutionContext struct {
	TenantID         uuid.UUID              `json:"tenantId"`
	Parameters       map[string]interface{} `json:"parameters"`
	SessionVariables map[string]interface{} `json:"sessionVariables"` // e.g. "TenantID", "UserID", "DeskList"
	ExecutionTime    time.Time              `json:"executionTime"`
	ExchangeCalendar string                 `json:"exchangeCalendar"` // e.g. "NYSE"
}
