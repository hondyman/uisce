package compliance

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateExtendsCoreRuleProps(t *testing.T) {
	tests := []struct {
		name        string
		props       map[string]interface{}
		expectError bool
	}{
		{
			name: "Valid props",
			props: map[string]interface{}{
				"pinned_core_version": 2,
				"drift_status":        "CURRENT",
				"extended_at":         time.Now().UTC().Format(time.RFC3339),
			},
			expectError: false,
		},
		{
			name: "Invalid version (zero)",
			props: map[string]interface{}{
				"pinned_core_version": 0,
				"drift_status":        "CURRENT",
			},
			expectError: true,
		},
		{
			name: "Invalid version (negative)",
			props: map[string]interface{}{
				"pinned_core_version": -1,
				"drift_status":        "CURRENT",
			},
			expectError: true,
		},
		{
			name: "Invalid drift status",
			props: map[string]interface{}{
				"pinned_core_version": 1,
				"drift_status":        "INVALID_STATE",
			},
			expectError: true,
		},
		{
			name:        "Nil props",
			props:       nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateExtendsCoreRuleProps(tt.props)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateExtendsCoreRuleProps() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestToCatalogNode_BytecodeExcludedFromProperties(t *testing.T) {
	ruleID := uuid.New()
	tenantID := uuid.New()
	coreRuleID := uuid.New()

	rule := ComplianceRuleRecord{
		ID:                  ruleID,
		TenantID:            tenantID,
		CoreRuleID:          &coreRuleID,
		InheritMode:         Extend,
		PinnedCoreVersion:   1,
		DriftStatus:         DriftCurrent,
		RuleCode:            "CONC_LIMIT_5PCT",
		Name:                "Single Issuer 5% Concentration Limit",
		Description:         "Restricts maximum single-issuer exposure to 5% of portfolio NAV",
		RulePhase:           "PRE_TRADE",
		Severity:            "HARD_BLOCK",
		CompiledBytecode:    []byte{0x01, 0x02, 0x03, 0x04, 0x05}, // 5 bytes dummy bytecode
		Priority:            10,
		IsActive:            true,
		CreatedAt:           time.Now().UTC(),
		UpdatedAt:           time.Now().UTC(),
	}

	node := rule.ToCatalogNode()

	if node.ID != ruleID.String() {
		t.Errorf("node.ID mismatch: got %s, want %s", node.ID, ruleID.String())
	}
	if node.NodeType != KindComplianceRule {
		t.Errorf("node.NodeType mismatch: got %s, want %s", node.NodeType, KindComplianceRule)
	}
	if node.QualifiedPath != "compliance.rule/"+tenantID.String()+"/CONC_LIMIT_5PCT" {
		t.Errorf("node.QualifiedPath mismatch: got %s", node.QualifiedPath)
	}

	// Assert bytecode is NOT copied into Properties
	if _, exists := node.Properties["compiled_bytecode"]; exists {
		t.Errorf("CRITICAL LEAK: compiled_bytecode found in node.Properties! Must be kept out of graph metadata.")
	}

	// Assert lightweight flag is present
	if hasBytecode, ok := node.Properties["has_compiled_bytecode"].(bool); !ok || !hasBytecode {
		t.Errorf("Expected has_compiled_bytecode=true in node.Properties")
	}
}
