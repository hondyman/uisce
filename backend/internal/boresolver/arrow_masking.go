package boresolver

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
)

type MaskingTier string

const (
	MaskingTierRedactFull    MaskingTier = "REDACT_FULL"
	MaskingTierHashSHA256    MaskingTier = "HASH_SHA256"
	MaskingTierNoiseAddition MaskingTier = "NOISE_ADDITION"
	MaskingTierPassthrough   MaskingTier = "PASSTHROUGH"
)

type ColumnMaskingRule struct {
	ColumnName     string      `json:"columnName"`
	SensitivityTag string      `json:"sensitivityTag"` // contains:pii, financial:confidential, quant:market_data, standard:cleared
	MaskingTier    MaskingTier `json:"maskingTier"`
	TargetPersona  string      `json:"targetPersona"`
}

// DetermineMaskingTier resolves the masking strategy based on sensitivity tags and user roles/clearance
func DetermineMaskingTier(sensitivityTag string, userRole string, clearanceLevel string) MaskingTier {
	tag := strings.ToLower(sensitivityTag)
	role := strings.ToLower(userRole)
	clearance := strings.ToUpper(clearanceLevel)

	if clearance == "CONFIDENTIAL" || role == "platform_trader" || role == "compliance_officer" {
		return MaskingTierPassthrough
	}

	switch {
	case strings.Contains(tag, "pii") || strings.Contains(tag, "ssn") || strings.Contains(tag, "email"):
		return MaskingTierRedactFull
	case strings.Contains(tag, "financial") || strings.Contains(tag, "confidential") || strings.Contains(tag, "account"):
		return MaskingTierHashSHA256
	case strings.Contains(tag, "quant") || strings.Contains(tag, "market_data") || strings.Contains(tag, "pricing"):
		return MaskingTierNoiseAddition
	default:
		return MaskingTierPassthrough
	}
}

// ApplyVectorMasking transforms an individual scalar or column value in-memory
func ApplyVectorMasking(val interface{}, tier MaskingTier) interface{} {
	if val == nil {
		return nil
	}

	switch tier {
	case MaskingTierRedactFull:
		return "***REDACTED***"

	case MaskingTierHashSHA256:
		str := fmt.Sprintf("%v", val)
		h := sha256.Sum256([]byte(str))
		return hex.EncodeToString(h[:])

	case MaskingTierNoiseAddition:
		switch v := val.(type) {
		case float64:
			jitter := (rand.Float64()*0.02 - 0.01) * v
			return v + jitter
		case int:
			jitter := float64(v) * (rand.Float64()*0.02 - 0.01)
			return float64(v) + jitter
		default:
			return val
		}

	case MaskingTierPassthrough:
		return val

	default:
		return val
	}
}

// CompileMaskedProjectionSQL generates the compile-time SQL masking expression for pushdown
func CompileMaskedProjectionSQL(columnExpr string, tier MaskingTier) string {
	switch tier {
	case MaskingTierRedactFull:
		return "CAST('***REDACTED***' AS VARCHAR)"
	case MaskingTierHashSHA256:
		return fmt.Sprintf("ENCODE(DIGEST(CAST(%s AS VARCHAR), 'sha256'), 'hex')", columnExpr)
	case MaskingTierNoiseAddition:
		return fmt.Sprintf("(%s + (RANDOM() * 0.02 - 0.01) * %s)", columnExpr, columnExpr)
	case MaskingTierPassthrough:
		return columnExpr
	default:
		return columnExpr
	}
}
