package models

import "time"

// AccessRule represents a security rule that binds LDAP groups to Business Objects.
type AccessRule struct {
	RuleID           string       `json:"ruleId" db:"id"`
	TenantID         string       `json:"tenantId" db:"tenant_id"`
	BusinessObjectID string       `json:"businessObjectId" db:"business_object_id"`
	GroupDn          string       `json:"groupDn" db:"group_dn"`
	AccessLevel      string       `json:"accessLevel" db:"access_level"`    // NONE, READ, WRITE
	Status           string       `json:"status" db:"status"`               // DRAFT, REVIEW, APPROVED, DEPRECATED
	RowFilterDsl     string       `json:"rowFilterDsl" db:"row_filter_dsl"` // DSL expression for row filtering
	ColumnMasks      []ColumnMask `json:"columnMasks" db:"column_masks"`    // JSON array in DB
	AppliesToApis    *bool        `json:"appliesToApis" db:"applies_to_apis"`
	AppliesToBi      *bool        `json:"appliesToBi" db:"applies_to_bi"`
	AppliesToAi      *bool        `json:"appliesToAi" db:"applies_to_ai"`
	CreatedBy        string       `json:"createdBy" db:"created_by"`
	CreatedAt        time.Time    `json:"createdAt" db:"created_at"`
	UpdatedBy        string       `json:"updatedBy" db:"updated_by"`
	UpdatedAt        time.Time    `json:"updatedAt" db:"updated_at"`
	Version          int          `json:"version" db:"version"`
	Description      string       `json:"description" db:"description"`
}

// ColumnMask represents field-level masking for a semantic term.
type ColumnMask struct {
	SemanticTermID string `json:"semanticTermId"`
	MaskType       string `json:"maskType"` // HIDE, MASK, NONE
}
