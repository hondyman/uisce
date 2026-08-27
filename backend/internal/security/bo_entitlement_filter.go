package security

import (
	"github.com/hondyman/uisce/backend/internal/models"
)

type FilteredBusinessObject struct {
	BO             *models.BusinessObjectDefinition
	FieldsVisible  []*models.FieldDefinition
	FieldsHidden   []string
	FieldsMasked   map[string]string
	HiddenSubtypes []string
}

type EntitlementsSummary struct {
	TotalBO         int `json:"totalBO"`
	VisibleBO       int `json:"visibleBO"`
	HiddenBO        int `json:"hiddenBO"`
	TotalFields     int `json:"totalFields"`
	VisibleFields   int `json:"visibleFields"`
	HiddenFields    int `json:"hiddenFields"`
	MaskedFields    int `json:"maskedFields"`
	HiddenSubtypes  int `json:"hiddenSubtypes"`
}

func ApplyBOEntitlements(
	bos []*models.BusinessObjectDefinition,
	fieldsPerBO map[string][]*models.FieldDefinition,
	entitlements *EntitlementsResult,
	isGlobalAdmin bool,
) ([]*FilteredBusinessObject, *EntitlementsSummary) {
	summary := &EntitlementsSummary{
		TotalBO: len(bos),
	}

	if isGlobalAdmin {
		result := make([]*FilteredBusinessObject, 0, len(bos))
		for _, bo := range bos {
			fbo := &FilteredBusinessObject{
				BO:             bo,
				FieldsVisible:   fieldsPerBO[bo.ID],
				FieldsHidden:   []string{},
				FieldsMasked:   map[string]string{},
				HiddenSubtypes: []string{},
			}
			if fieldsPerBO[bo.ID] != nil {
				summary.TotalFields += len(fieldsPerBO[bo.ID])
				summary.VisibleFields += len(fieldsPerBO[bo.ID])
			}
			result = append(result, fbo)
		}
		summary.VisibleBO = len(bos)
		summary.HiddenBO = 0
		return result, summary
	}

	if entitlements == nil {
		entitlements = &EntitlementsResult{
			Entitlements:    map[EntitlementKey]string{},
			MaskingPatterns: map[EntitlementKey]string{},
			HiddenBOs:       map[string]struct{}{},
			MaskedFields:    map[EntitlementKey]struct{}{},
		}
	}

	result := make([]*FilteredBusinessObject, 0, len(bos))

	for _, bo := range bos {
		isExplicitlyDenied := false

		if _, denied := entitlements.HiddenBOs[bo.ID]; denied {
			isExplicitlyDenied = true
		}

		if isExplicitlyDenied {
			summary.HiddenBO++
			continue
		}

		fields := fieldsPerBO[bo.ID]
		if fields == nil {
			fields = []*models.FieldDefinition{}
		}

		visibleFields := make([]*models.FieldDefinition, 0, len(fields))
		hiddenFields := []string{}
		maskedFields := map[string]string{}

		for _, field := range fields {
			summary.TotalFields++
			fieldKey := EntitlementKey{ResourceID: bo.ID, FieldName: field.Name}

			if field.Name == "" {
				fieldKey.FieldName = field.Key
			}

			perm := entitlements.Entitlements[fieldKey]

			if perm == "none" {
				hiddenFields = append(hiddenFields, field.Name)
				summary.HiddenFields++
				continue
			}

			if perm == "mask" || perm == "read" || perm == "write" {
				if _, isMasked := entitlements.MaskedFields[fieldKey]; isMasked {
					fieldCopy := *field
					fieldCopy.Masked = true
					if pattern, ok := entitlements.MaskingPatterns[fieldKey]; ok {
						fieldCopy.MaskingPattern = pattern
					}
					visibleFields = append(visibleFields, &fieldCopy)
					maskedFields[field.Name] = entitlements.MaskingPatterns[fieldKey]
					summary.MaskedFields++
				} else {
					visibleFields = append(visibleFields, field)
				}
				summary.VisibleFields++
			} else {
				visibleFields = append(visibleFields, field)
				summary.VisibleFields++
			}
		}

		hiddenSubtypes := []string{}
		for subtypeKey := range bo.Subtypes {
			stResourceID := bo.ID + "/" + subtypeKey
			_, isExplicitlyDenied := entitlements.HiddenBOs[stResourceID]
			stPerm := entitlements.Entitlements[EntitlementKey{ResourceID: stResourceID, FieldName: "*"}]
			if isExplicitlyDenied || stPerm == "none" {
				hiddenSubtypes = append(hiddenSubtypes, subtypeKey)
				summary.HiddenSubtypes++
			}
		}

		result = append(result, &FilteredBusinessObject{
			BO:             bo,
			FieldsVisible:  visibleFields,
			FieldsHidden:   hiddenFields,
			FieldsMasked:   maskedFields,
			HiddenSubtypes: hiddenSubtypes,
		})
		summary.VisibleBO++
	}

	return result, summary
}

func EntitlementsSummaryHeader(summary *EntitlementsSummary) string {
	if summary == nil {
		return "hidden_bos=0,hidden_fields=0,masked_fields=0"
	}
	return formatEntitlementsSummary(summary)
}

func formatEntitlementsSummary(s *EntitlementsSummary) string {
	return "hidden_bos=" + itoa(s.HiddenBO) +
		",hidden_fields=" + itoa(s.HiddenFields) +
		",masked_fields=" + itoa(s.MaskedFields) +
		",hidden_subtypes=" + itoa(s.HiddenSubtypes)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	result := ""
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	for i > 0 {
		result = string(rune('0'+i%10)) + result
		i /= 10
	}
	if neg {
		result = "-" + result
	}
	return result
}
