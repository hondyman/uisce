package wealth

import (
	"context"
	"encoding/json"

	"github.com/hondyman/semlayer/backend/pkg/meta"
)

// RegisterWealthTransferBusinessObjects registers Family Office and related entities
// as BusinessObjects in the metadata system (Workday-style)
func RegisterWealthTransferBusinessObjects(ctx context.Context, metaService *meta.Service, tenantID string) error {
	// 1. Register Family Office Business Object
	familyOfficeBO := &meta.BusinessObjectDefinition{
		TenantID: tenantID,
		Name:     "Family_Office",
		Storage:  "row", // Use row-based storage for primary data
		Version:  1,
		Status:   "active",
		Fields: []meta.FieldDefinition{
			{
				Name:       "family_name",
				Label:      "Family Name",
				Type:       meta.FieldString,
				IsRequired: true,
			},
			{
				Name:       "total_estimated_networth",
				Label:      "Total Estimated Net Worth",
				Type:       meta.FieldDecimal,
				IsRequired: true,
			},
			{
				Name:   "primary_state",
				Label:  "Primary State",
				Type:   meta.FieldEnum,
				EnumID: stringPtr("us_states"),
			},
			{
				Name:   "estate_plan_status",
				Label:  "Estate Plan Status",
				Type:   meta.FieldEnum,
				EnumID: stringPtr("estate_plan_status"),
			},
			{
				Name:  "governance_structure",
				Label: "Governance Structure",
				Type:  meta.FieldJSON, // JSONB for flexible structure
			},
		},
		Metadata: map[string]any{
			"icon":        "family",
			"category":    "wealth_transfer",
			"audit_trail": true,
			"versioning":  true,
		},
	}

	if err := metaService.CreateBusinessObject(ctx, familyOfficeBO); err != nil {
		return err
	}

	// 2. Register Family Member Business Object
	familyMemberBO := &meta.BusinessObjectDefinition{
		TenantID: tenantID,
		Name:     "Family_Member",
		Storage:  "row",
		Version:  1,
		Status:   "active",
		Fields: []meta.FieldDefinition{
			{
				Name:        "family_id",
				Label:       "Family Office",
				Type:        meta.FieldRef,
				RefObjectID: stringPtr("Family_Office"),
				IsRequired:  true,
			},
			{
				Name:       "legal_first_name",
				Label:      "Legal First Name",
				Type:       meta.FieldString,
				IsRequired: true,
			},
			{
				Name:       "legal_last_name",
				Label:      "Legal Last Name",
				Type:       meta.FieldString,
				IsRequired: true,
			},
			{
				Name:  "date_of_birth",
				Label: "Date of Birth",
				Type:  meta.FieldDate,
			},
			{
				Name:  "generation",
				Label: "Generation",
				Type:  meta.FieldDecimal,
			},
			{
				Name:  "separate_networth",
				Label: "Separate Net Worth",
				Type:  meta.FieldDecimal,
			},
			{
				Name:       "relationship_type",
				Label:      "Relationship Type",
				Type:       meta.FieldEnum,
				EnumID:     stringPtr("relationship_types"),
				IsRequired: false,
			},
		},
		Metadata: map[string]any{
			"icon":              "person",
			"category":          "wealth_transfer",
			"parent_object":     "Family_Office",
			"list_view_default": true,
		},
	}

	if err := metaService.CreateBusinessObject(ctx, familyMemberBO); err != nil {
		return err
	}

	// 3. Register Estate Planning Scenario Business Object
	scenarioBO := &meta.BusinessObjectDefinition{
		TenantID: tenantID,
		Name:     "Estate_Plan_Scenario",
		Storage:  "wide_jsonb", // Use JSONB storage for flexible scenario data
		Version:  1,
		Status:   "active",
		Fields: []meta.FieldDefinition{
			{
				Name:        "family_id",
				Label:       "Family Office",
				Type:        meta.FieldRef,
				RefObjectID: stringPtr("Family_Office"),
				IsRequired:  true,
			},
			{
				Name:       "scenario_name",
				Label:      "Scenario Name",
				Type:       meta.FieldString,
				IsRequired: true,
			},
			{
				Name:       "strategy_type",
				Label:      "Strategy Type",
				Type:       meta.FieldEnum,
				EnumID:     stringPtr("estate_planning_strategies"),
				IsRequired: true,
			},
			{
				Name:  "projected_tax_savings",
				Label: "Projected Tax Savings",
				Type:  meta.FieldDecimal,
			},
			{
				Name:  "complexity_score",
				Label: "Complexity Score (1-10)",
				Type:  meta.FieldDecimal,
			},
			{
				Name:  "overall_score",
				Label: "Overall Score",
				Type:  meta.FieldDecimal,
			},
			{
				Name:  "scenario_details",
				Label: "Scenario Details",
				Type:  meta.FieldJSON, // Full scenario data in JSONB
			},
		},
		Metadata: map[string]any{
			"icon":          "strategy",
			"category":      "wealth_transfer",
			"parent_object": "Family_Office",
			"ai_generated":  true,
		},
	}

	if err := metaService.CreateBusinessObject(ctx, scenarioBO); err != nil {
		return err
	}

	// 4. Register Gift History Business Object
	giftHistoryBO := &meta.BusinessObjectDefinition{
		TenantID: tenantID,
		Name:     "Gift_History",
		Storage:  "row",
		Version:  1,
		Status:   "active",
		Fields: []meta.FieldDefinition{
			{
				Name:        "family_id",
				Label:       "Family Office",
				Type:        meta.FieldRef,
				RefObjectID: stringPtr("Family_Office"),
				IsRequired:  true,
			},
			{
				Name:        "donor_member_id",
				Label:       "Donor",
				Type:        meta.FieldRef,
				RefObjectID: stringPtr("Family_Member"),
				IsRequired:  true,
			},
			{
				Name:  "gift_date",
				Label: "Gift Date",
				Type:  meta.FieldDate,
			},
			{
				Name:  "fair_market_value",
				Label: "Fair Market Value",
				Type:  meta.FieldDecimal,
			},
			{
				Name:   "gift_type",
				Label:  "Gift Type",
				Type:   meta.FieldEnum,
				EnumID: stringPtr("gift_types"),
			},
			{
				Name:  "annual_exclusion_utilized",
				Label: "Annual Exclusion Utilized",
				Type:  meta.FieldDecimal,
			},
			{
				Name:  "lifetime_exemption_utilized",
				Label: "Lifetime Exemption Utilized",
				Type:  meta.FieldDecimal,
			},
		},
		Metadata: map[string]any{
			"icon":          "gift",
			"category":      "wealth_transfer",
			"parent_object": "Family_Office",
			"timeline_view": true,
		},
	}

	if err := metaService.CreateBusinessObject(ctx, giftHistoryBO); err != nil {
		return err
	}

	// 5. Register Trust Entity Business Object
	trustBO := &meta.BusinessObjectDefinition{
		TenantID: tenantID,
		Name:     "Trust_Entity",
		Storage:  "wide_jsonb",
		Version:  1,
		Status:   "active",
		Fields: []meta.FieldDefinition{
			{
				Name:        "family_id",
				Label:       "Family Office",
				Type:        meta.FieldRef,
				RefObjectID: stringPtr("Family_Office"),
				IsRequired:  true,
			},
			{
				Name:       "entity_name",
				Label:      "Entity Name",
				Type:       meta.FieldString,
				IsRequired: true,
			},
			{
				Name:       "entity_type",
				Label:      "Entity Type",
				Type:       meta.FieldEnum,
				EnumID:     stringPtr("trust_types"),
				IsRequired: true,
			},
			{
				Name:  "formation_date",
				Label: "Formation Date",
				Type:  meta.FieldDate,
			},
			{
				Name:   "is_revocable",
				Label:  "Is Revocable",
				Type:   meta.FieldEnum,
				EnumID: stringPtr("boolean"),
			},
			{
				Name:  "current_value",
				Label: "Current Value",
				Type:  meta.FieldDecimal,
			},
			{
				Name:  "trust_terms",
				Label: "Trust Terms",
				Type:  meta.FieldJSON, // Flexible JSONB for trust-specific terms
			},
		},
		Metadata: map[string]any{
			"icon":          "trust",
			"category":      "wealth_transfer",
			"parent_object": "Family_Office",
			"compliance":    true,
		},
	}

	if err := metaService.CreateBusinessObject(ctx, trustBO); err != nil {
		return err
	}

	return nil
}

// RegisterWealthTransferEnums registers enums used by wealth transfer
func RegisterWealthTransferEnums(ctx context.Context, metaService *meta.Service, tenantID string) error {
	enums := []meta.EnumDefinition{
		{
			TenantID: tenantID,
			Name:     "estate_plan_status",
			Values: []meta.EnumValue{
				{Value: "NO_PLAN", Label: "No Plan"},
				{Value: "DRAFT", Label: "Draft"},
				{Value: "UNDER_REVIEW", Label: "Under Review"},
				{Value: "APPROVED", Label: "Approved"},
				{Value: "IMPLEMENTING", Label: "Implementing"},
				{Value: "ACTIVE", Label: "Active"},
				{Value: "NEEDS_UPDATE", Label: "Needs Update"},
			},
		},
		{
			TenantID: tenantID,
			Name:     "estate_planning_strategies",
			Values: []meta.EnumValue{
				{Value: "ANNUAL_GIFTING", Label: "Annual Exclusion Gifting"},
				{Value: "SLAT", Label: "Spousal Lifetime Access Trust"},
				{Value: "GRAT", Label: "Grantor Retained Annuity Trust"},
				{Value: "DYNASTY_TRUST", Label: "Dynasty Trust"},
				{Value: "ILIT", Label: "Irrevocable Life Insurance Trust"},
				{Value: "CRT", Label: "Charitable Remainder Trust"},
				{Value: "QTIP", Label: "QTIP Trust"},
			},
		},
		{
			TenantID: tenantID,
			Name:     "trust_types",
			Values: []meta.EnumValue{
				{Value: "SLAT", Label: "SLAT"},
				{Value: "GRAT", Label: "GRAT"},
				{Value: "QPRT", Label: "QPRT"},
				{Value: "ILIT", Label: "ILIT"},
				{Value: "DYNASTY_TRUST", Label: "Dynasty Trust"},
				{Value: "CRT", Label: "Charitable Remainder Trust"},
				{Value: "CLT", Label: "Charitable Lead Trust"},
				{Value: "QTIP", Label: "QTIP Trust"},
				{Value: "GST", Label: "Generation-Skipping Trust"},
			},
		},
		{
			TenantID: tenantID,
			Name:     "gift_types",
			Values: []meta.EnumValue{
				{Value: "ANNUAL_EXCLUSION", Label: "Annual Exclusion"},
				{Value: "LIFETIME_EXEMPTION", Label: "Lifetime Exemption"},
				{Value: "CHARITABLE", Label: "Charitable"},
				{Value: "QUALIFIED_TUITION", Label: "Qualified Tuition/Medical"},
			},
		},
		{
			TenantID: tenantID,
			Name:     "relationship_types",
			Values: []meta.EnumValue{
				{Value: "SPOUSE", Label: "Spouse"},
				{Value: "CHILD", Label: "Child"},
				{Value: "GRANDCHILD", Label: "Grandchild"},
				{Value: "SIBLING", Label: "Sibling"},
				{Value: "PARENT", Label: "Parent"},
			},
		},
	}

	for _, enum := range enums {
		enumJSON, _ := json.Marshal(enum)
		_ = enumJSON // TODO: Call metaService.CreateEnum when available
	}

	return nil
}

func stringPtr(s string) *string {
	return &s
}
