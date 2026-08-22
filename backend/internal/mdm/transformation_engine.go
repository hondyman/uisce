package mdm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type FieldTransformRule struct {
	SourceFieldName    string  `db:"source_field_name"`
	TargetColumnName   string  `db:"target_column_name"`
	TransformationType string  `db:"transformation_type"`
	TransformationExpr string  `db:"transformation_expr"`
	TargetDataType     string  `db:"target_data_type"`
	NullFallbackValue  *string `db:"null_fallback_value"`
}

type TransformationEngine struct {
	db *sqlx.DB
}

func NewTransformationEngine(db *sqlx.DB) *TransformationEngine {
	return &TransformationEngine{db: db}
}

// TransformRecord maps canonical Gold attributes into target-specific schema representations
func (e *TransformationEngine) TransformRecord(
	ctx context.Context,
	tenantID, bindingID uuid.UUID,
	goldAttributes map[string]interface{},
) (map[string]interface{}, string, error) {
	var rules []FieldTransformRule
	if e.db != nil {
		query := `
			SELECT source_field_name, target_column_name, transformation_type, 
			       COALESCE(transformation_expr, '') AS transformation_expr,
			       target_data_type, null_fallback_value
			FROM mdm_pipeline.field_transformation_rules
			WHERE tenant_id = $1 AND binding_id = $2;
		`
		err := e.db.SelectContext(ctx, &rules, query, tenantID, bindingID)
		if err != nil {
			return nil, "", fmt.Errorf("failed fetching field transform rules: %w", err)
		}
	}

	transformed := make(map[string]interface{})

	if len(rules) == 0 {
		for k, v := range goldAttributes {
			transformed[k] = v
		}
	} else {
		for _, r := range rules {
			rawVal, exists := goldAttributes[r.SourceFieldName]
			if !exists || rawVal == nil {
				if r.NullFallbackValue != nil {
					transformed[r.TargetColumnName] = *r.NullFallbackValue
				} else {
					transformed[r.TargetColumnName] = nil
				}
				continue
			}

			switch r.TransformationType {
			case "DIRECT":
				transformed[r.TargetColumnName] = rawVal

			case "EXPRESSION":
				transformed[r.TargetColumnName] = evaluateStringTransform(r.TransformationExpr, fmt.Sprintf("%v", rawVal))

			case "CODE_TRANSLATION":
				targetCode, transErr := e.resolveCodeTranslation(ctx, tenantID, r.TransformationExpr, fmt.Sprintf("%v", rawVal))
				if transErr != nil {
					transformed[r.TargetColumnName] = rawVal
				} else {
					transformed[r.TargetColumnName] = targetCode
				}

			default:
				transformed[r.TargetColumnName] = rawVal
			}
		}
	}

	payloadBytes, _ := json.Marshal(transformed)
	hash := sha256.Sum256(payloadBytes)
	checksum := hex.EncodeToString(hash[:])

	return transformed, checksum, nil
}

func (e *TransformationEngine) resolveCodeTranslation(
	ctx context.Context,
	tenantID uuid.UUID,
	dictionaryName, sourceCode string,
) (string, error) {
	if e.db == nil {
		return sourceCode, nil
	}
	var targetCode string
	query := `
		SELECT target_code 
		FROM mdm_pipeline.code_translation_dictionaries 
		WHERE tenant_id = $1 AND dictionary_name = $2 AND source_code = $3;
	`
	err := e.db.GetContext(ctx, &targetCode, query, tenantID, dictionaryName, sourceCode)
	if err != nil {
		return "", err
	}
	return targetCode, nil
}

func evaluateStringTransform(expr, val string) string {
	clean := strings.TrimSpace(val)
	if strings.HasPrefix(expr, "UPPER") {
		return strings.ToUpper(clean)
	}
	if strings.HasPrefix(expr, "LOWER") {
		return strings.ToLower(clean)
	}
	return clean
}
