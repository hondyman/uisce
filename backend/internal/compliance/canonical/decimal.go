package canonical

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// FixedScale is the standard scale (6 decimal places) for financial compliance metrics.
const FixedScale int32 = 6

// ErrDecimalScaleExceeded is returned when a decimal value has more than 6 fractional decimal places.
var ErrDecimalScaleExceeded = errors.New("canonical: decimal exceeds maximum allowed scale of 6 decimal places")

// FormatDecimal6 formats a decimal into a fixed 6-decimal canonical string (e.g. 123.4 -> "123.400000").
// If the decimal has non-zero digits beyond 6 decimal places, it strictly returns ErrDecimalScaleExceeded
// rather than silently rounding, to preserve cross-system determinism.
func FormatDecimal6(d decimal.Decimal) (string, error) {
	// Check if there are non-zero fractional digits beyond 6 decimal places
	if d.Exponent() < -FixedScale {
		truncated := d.Truncate(FixedScale)
		if !d.Equal(truncated) {
			return "", fmt.Errorf("%w: value %s has scale %d", ErrDecimalScaleExceeded, d.String(), -d.Exponent())
		}
	}
	return d.StringFixed(FixedScale), nil
}

// ParseDecimal6 parses a string into a decimal and validates that it does not exceed 6 decimal places.
func ParseDecimal6(s string) (decimal.Decimal, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, err
	}
	if _, err := FormatDecimal6(d); err != nil {
		return decimal.Zero, err
	}
	return d, nil
}

// CanonicalDecimalMap transforms a map of string->any to ensure all decimal or float fields
// are strictly validated against the 6-decimal limit and converted to fixed 6-decimal strings.
func CanonicalDecimalMap(m map[string]interface{}) (map[string]interface{}, error) {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		norm, err := normalizeValue(v)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", k, err)
		}
		out[k] = norm
	}
	return out, nil
}

func normalizeValue(v interface{}) (interface{}, error) {
	switch val := v.(type) {
	case decimal.Decimal:
		return FormatDecimal6(val)
	case *decimal.Decimal:
		if val == nil {
			return nil, nil
		}
		return FormatDecimal6(*val)
	case float64:
		d := decimal.NewFromFloat(val)
		return FormatDecimal6(d)
	case float32:
		d := decimal.NewFromFloat(float64(val))
		return FormatDecimal6(d)
	case map[string]interface{}:
		return CanonicalDecimalMap(val)
	case []interface{}:
		res := make([]interface{}, len(val))
		for i, item := range val {
			norm, err := normalizeValue(item)
			if err != nil {
				return nil, err
			}
			res[i] = norm
		}
		return res, nil
	default:
		return val, nil
	}
}

// EvaluationHashInput contains the fields needed to compute a deterministic EvaluationHash under schema v2.
type EvaluationHashInput struct {
	LineageID       uuid.UUID              `json:"lineageId"`
	TenantID        uuid.UUID              `json:"tenantId"`
	RuleID          uuid.UUID              `json:"ruleId"`
	RuleVersion     int                    `json:"ruleVersion"`
	RuleContentHash string                 `json:"ruleContentHash"`
	ActionTaken     string                 `json:"actionTaken"`
	Passed          bool                   `json:"passed"`
	InputParams     map[string]interface{} `json:"inputParams"`
	MetricSnapshots map[string]interface{} `json:"metricSnapshots"`
}

// ComputeEvaluationHash generates an RFC 8785 canonical hash of the evaluation event under schema v2.
// EvaluationHash = SHA256( "v2|" || LineageID || "|" || TenantID || "|" || RuleID || "|" || RuleVersion || "|" || RuleContentHash || "|" || ActionTaken || "|" || Passed || "|" || JCS(InputParams) || "|" || JCS(MetricSnapshots) )
func ComputeEvaluationHash(input EvaluationHashInput) (string, error) {
	normInputs, err := CanonicalDecimalMap(input.InputParams)
	if err != nil {
		return "", fmt.Errorf("invalid inputParams: %w", err)
	}
	normMetrics, err := CanonicalDecimalMap(input.MetricSnapshots)
	if err != nil {
		return "", fmt.Errorf("invalid metricSnapshots: %w", err)
	}

	canonicalInputBytes, err := Marshal(normInputs)
	if err != nil {
		return "", fmt.Errorf("failed to canonicalize inputParams: %w", err)
	}

	canonicalMetricBytes, err := Marshal(normMetrics)
	if err != nil {
		return "", fmt.Errorf("failed to canonicalize metricSnapshots: %w", err)
	}

	h := sha256.New()
	h.Write([]byte("v2|"))
	h.Write([]byte(input.LineageID.String()))
	h.Write([]byte("|"))
	h.Write([]byte(input.TenantID.String()))
	h.Write([]byte("|"))
	h.Write([]byte(input.RuleID.String()))
	h.Write([]byte("|"))
	h.Write([]byte(fmt.Sprintf("%d", input.RuleVersion)))
	h.Write([]byte("|"))
	h.Write([]byte(input.RuleContentHash))
	h.Write([]byte("|"))
	h.Write([]byte(input.ActionTaken))
	h.Write([]byte("|"))
	if input.Passed {
		h.Write([]byte("true"))
	} else {
		h.Write([]byte("false"))
	}
	h.Write([]byte("|"))
	h.Write(canonicalInputBytes)
	h.Write([]byte("|"))
	h.Write(canonicalMetricBytes)

	return hex.EncodeToString(h.Sum(nil)), nil
}

// ComputeRuleContentHash calculates the canonical SHA-256 hash of a rule version's logic, thresholds, and citation.
func ComputeRuleContentHash(astCondition, parameterThresholds map[string]interface{}, citation string) (string, error) {
	normAST, err := CanonicalDecimalMap(astCondition)
	if err != nil {
		return "", fmt.Errorf("canonicalize AST: %w", err)
	}
	normParams, err := CanonicalDecimalMap(parameterThresholds)
	if err != nil {
		return "", fmt.Errorf("canonicalize params: %w", err)
	}

	astBytes, err := Marshal(normAST)
	if err != nil {
		return "", fmt.Errorf("marshal AST: %w", err)
	}
	paramBytes, err := Marshal(normParams)
	if err != nil {
		return "", fmt.Errorf("marshal params: %w", err)
	}

	h := sha256.New()
	h.Write([]byte("v1|"))
	h.Write(astBytes)
	h.Write([]byte("|"))
	h.Write(paramBytes)
	h.Write([]byte("|"))
	h.Write([]byte(citation))

	return hex.EncodeToString(h.Sum(nil)), nil
}

// ComputeBytecodeHash calculates the SHA-256 digest of compiled bytecode.
func ComputeBytecodeHash(bytecode []byte) string {
	if len(bytecode) == 0 {
		h := sha256.Sum256([]byte{})
		return hex.EncodeToString(h[:])
	}
	h := sha256.Sum256(bytecode)
	return hex.EncodeToString(h[:])
}
