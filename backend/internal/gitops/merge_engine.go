package gitops

import (
	"github.com/google/uuid"
)

type FieldDefinition struct {
	FieldKey    string `json:"field_key"`
	DataType    string `json:"data_type"`
	Description string `json:"description"`
	IsRequired  bool   `json:"is_required"`
}

type BusinessObjectOverlay struct {
	BOKey    string            `json:"bo_key"`
	TenantID uuid.UUID         `json:"tenant_id"`
	Version  string            `json:"version"`
	Fields   []FieldDefinition `json:"fields"`
}

type MergeConflict struct {
	FieldKey      string `json:"field_key"`
	BaseValue     string `json:"base_value"`
	UpstreamValue string `json:"upstream_value"`
	TenantValue   string `json:"tenant_value"`
}

type MergeResult struct {
	MergedBO    *BusinessObjectOverlay `json:"merged_bo"`
	Conflicts   []MergeConflict        `json:"conflicts"`
	HasConflict bool                   `json:"has_conflict"`
}

type OverlayMergeEngine struct{}

func NewOverlayMergeEngine() *OverlayMergeEngine {
	return &OverlayMergeEngine{}
}

func (e *OverlayMergeEngine) ThreeWayMerge(
	ancestor *BusinessObjectOverlay,
	upstream *BusinessObjectOverlay,
	tenant *BusinessObjectOverlay,
) (*MergeResult, error) {
	result := &MergeResult{
		MergedBO: &BusinessObjectOverlay{
			BOKey:    upstream.BOKey,
			TenantID: tenant.TenantID,
			Version:  upstream.Version,
			Fields:   make([]FieldDefinition, 0),
		},
		Conflicts:   make([]MergeConflict, 0),
		HasConflict: false,
	}

	ancestorMap := indexFields(ancestor.Fields)
	upstreamMap := indexFields(upstream.Fields)
	tenantMap := indexFields(tenant.Fields)

	allKeys := make(map[string]bool)
	for k := range ancestorMap {
		allKeys[k] = true
	}
	for k := range upstreamMap {
		allKeys[k] = true
	}
	for k := range tenantMap {
		allKeys[k] = true
	}

	for key := range allKeys {
		baseField, inBase := ancestorMap[key]
		upField, inUpstream := upstreamMap[key]
		tenField, inTenant := tenantMap[key]

		switch {
		case !inBase && !inUpstream && inTenant:
			result.MergedBO.Fields = append(result.MergedBO.Fields, tenField)

		case !inBase && inUpstream && !inTenant:
			result.MergedBO.Fields = append(result.MergedBO.Fields, upField)

		case inBase && !inUpstream && inTenant:
			if tenField.DataType == baseField.DataType {
				// Tenant kept ancestor field; upstream deleted it — deletion wins, skip
			} else {
				// Tenant modified field that upstream deleted — conflict
				result.HasConflict = true
				result.Conflicts = append(result.Conflicts, MergeConflict{
					FieldKey:      key,
					BaseValue:     baseField.DataType,
					UpstreamValue: "DELETED",
					TenantValue:   tenField.DataType,
				})
			}

		case inBase && inUpstream && !inTenant:
			// Tenant deleted the field; upstream and ancestor preserved it — tenant deletion wins
			_ = baseField

		case inBase && !inUpstream && !inTenant:
			// All three removed the field — skip (no action needed)

		case !inBase && inUpstream && inTenant:
			if upField.DataType == tenField.DataType {
				result.MergedBO.Fields = append(result.MergedBO.Fields, tenField)
			} else {
				result.HasConflict = true
				result.Conflicts = append(result.Conflicts, MergeConflict{
					FieldKey:      key,
					BaseValue:     "N/A",
					UpstreamValue: upField.DataType,
					TenantValue:   tenField.DataType,
				})
			}

		case inBase && inUpstream && inTenant:
			upModified := upField.DataType != baseField.DataType
			tenModified := tenField.DataType != baseField.DataType

			if !upModified && !tenModified {
				result.MergedBO.Fields = append(result.MergedBO.Fields, baseField)
			} else if upModified && !tenModified {
				result.MergedBO.Fields = append(result.MergedBO.Fields, upField)
			} else if !upModified && tenModified {
				result.MergedBO.Fields = append(result.MergedBO.Fields, tenField)
			} else {
				if upField.DataType == tenField.DataType {
					result.MergedBO.Fields = append(result.MergedBO.Fields, tenField)
				} else {
					result.HasConflict = true
					result.Conflicts = append(result.Conflicts, MergeConflict{
						FieldKey:      key,
						BaseValue:     baseField.DataType,
						UpstreamValue: upField.DataType,
						TenantValue:   tenField.DataType,
					})
				}
			}
		}
	}

	return result, nil
}

func indexFields(fields []FieldDefinition) map[string]FieldDefinition {
	m := make(map[string]FieldDefinition)
	for _, f := range fields {
		m[f.FieldKey] = f
	}
	return m
}
