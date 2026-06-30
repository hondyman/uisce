package boresolver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type ClassificationRule struct {
	Tag      string `json:"tag"`
	MaskType string `json:"mask_type"`
}

type PolicyMaskingPayload struct {
	Classifications []ClassificationRule `json:"classifications"`
}

type AIGraphSecurityInterceptor struct {
	TenantPool *sql.DB // Connection pool locked into tenant RLS
	SystemPool *sql.DB // Bypasses RLS to read system blueprints safely
}

// ResolveGraphGovernanceContext traces lineage down to the Business Term node to find classification tags
func (in *AIGraphSecurityInterceptor) ResolveGraphGovernanceContext(ctx context.Context, physicalColumnPath string) (string, error) {
	// Traces Physical -> Business -> Semantic lineage path 
	query := `
		SELECT 
			bus_node.properties AS business_properties
		FROM public.catalog_node phys_node
		INNER JOIN public.catalog_edge e1 ON e1.source_node_id = phys_node.id AND e1.is_active = true
		INNER JOIN public.catalog_node bus_node ON e1.target_node_id = bus_node.id AND bus_node.is_active = true
		WHERE phys_node.qualified_path = $1 
		  AND phys_node.is_active = true
		LIMIT 1;
	`

	var rawBusProps []byte

	// Invariant Check 1: Query tenant local database space first
	err := in.TenantPool.QueryRowContext(ctx, query, physicalColumnPath).Scan(&rawBusProps)
	if err == sql.ErrNoRows {
		// Fallback Invariant 2: Pull global Gold Copy definitions if no tenant override exists
		err = in.SystemPool.QueryRowContext(ctx, query, physicalColumnPath).Scan(&rawBusProps)
		if err != nil {
			// Zero-Tolerance Default: Redact field fully if metadata connections are broken
			return "REDACT_FULL", nil
		}
	} else if err != nil {
		return "", fmt.Errorf("metadata graph traversal exception: %w", err)
	}

	var busProps map[string]interface{}
	if err := json.Unmarshal(rawBusProps, &busProps); err != nil {
		return "", err
	}

	classification, ok := busProps["classification"].(string)
	if !ok || classification == "" {
		return "NONE", nil
	}

	return classification, nil
}

// EvaluateEffectiveMaskingType cross-references active classifications against profile-level ABAC rule dictionaries
func (in *AIGraphSecurityInterceptor) EvaluateEffectiveMaskingType(ctx context.Context, targetProfile string, userTenantID uuid.UUID, classificationTag string) string {
	if classificationTag == "NONE" {
		return "NONE"
	}

	// Fetch both global and tenant custom overrides ordered by priority
	query := `
		SELECT masking_rules 
		FROM security.abac_policies
		WHERE target_profile = $1 
		  AND (tenant_id IS NULL OR tenant_id = $2)
		  AND is_active = true
		ORDER BY tenant_id FETCH FIRST ROW ONLY; -- Custom tenant rows apply last to overwrite settings
	`

	var rawRules []byte
	err := in.TenantPool.QueryRowContext(ctx, query, targetProfile, userTenantID).Scan(&rawRules)
	if err != nil {
		return "NONE" // Return baseline clean status if no matching profile constraints exist
	}

	var payload PolicyMaskingPayload
	if err := json.Unmarshal(rawRules, &payload); err != nil {
		return "NONE"
	}

	// Match classification strings sequentially
	for _, rule := range payload.Classifications {
		if rule.Tag == classificationTag {
			return rule.MaskType
		}
	}

	return "NONE"
}

// MutateSQLSelectExpression transforms projected column queries into encapsulated dialect-safe operations
func (in *AIGraphSecurityInterceptor) MutateSQLSelectExpression(tableAlias string, columnName string, maskType string) string {
	qualifiedColumn := fmt.Sprintf("%s.%s", tableAlias, columnName)

	switch strings.ToUpper(maskType) {
	case "REDACT_FULL":
		return fmt.Sprintf("('[REDACTED]') AS %s", columnName)
	case "REDACT_LAST_FOUR":
		return fmt.Sprintf("(CONCAT('******', RIGHT(CAST(%s AS VARCHAR), 4))) AS %s", qualifiedColumn, columnName)
	case "HASH_SHA256":
		return fmt.Sprintf("(SHA256(CAST(%s AS VARCHAR))) AS %s", qualifiedColumn, columnName)
	default:
		return fmt.Sprintf("%s AS %s", qualifiedColumn, columnName)
	}
}
