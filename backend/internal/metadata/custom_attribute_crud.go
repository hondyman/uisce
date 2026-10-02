package metadata

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/attribute"
	"github.com/hondyman/uisce/backend/internal/logging"
)

// dehydrateCustomAttributesForBO packs semantic/custom field keys on a write
// payload into custom_attributes JSONB using attribute_def projections.
func (s *BusinessObjectService) dehydrateCustomAttributesForBO(
	ctx context.Context,
	tenantID string,
	driverTable string,
	record map[string]any,
) map[string]any {
	if record == nil || s.db == nil || driverTable == "" {
		return record
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return record
	}
	tableRef := normalizeDriverTableRef(driverTable)
	projections, err := attribute.LoadProjectionsForTable(ctx, s.db, tid, tableRef)
	if err != nil || len(projections) == 0 {
		return record
	}
	return attribute.DehydrateRecord(record, projections)
}

// hydrateCustomAttributesForBO expands custom_attributes onto logical field names.
func (s *BusinessObjectService) hydrateCustomAttributesForBO(
	ctx context.Context,
	tenantID string,
	driverTable string,
	record map[string]any,
) map[string]any {
	if record == nil || s.db == nil || driverTable == "" {
		return record
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return record
	}
	tableRef := normalizeDriverTableRef(driverTable)
	projections, err := attribute.LoadProjectionsForTable(ctx, s.db, tid, tableRef)
	if err != nil {
		logging.GetLogger().Sugar().Debugf("custom attribute hydrate skipped: %v", err)
		return record
	}
	if len(projections) == 0 {
		// Still normalize jsonb bytes → map when present
		if raw, ok := record["custom_attributes"]; ok {
			switch v := raw.(type) {
			case []byte:
				var m map[string]any
				if json.Unmarshal(v, &m) == nil {
					record["custom_attributes"] = m
				}
			case string:
				var m map[string]any
				if json.Unmarshal([]byte(v), &m) == nil {
					record["custom_attributes"] = m
				}
			}
		}
		return record
	}
	return attribute.HydrateRecord(record, projections)
}

func (s *BusinessObjectService) hydrateCustomAttributeRows(
	ctx context.Context,
	tenantID string,
	driverTable string,
	rows []map[string]any,
) {
	for i := range rows {
		rows[i] = s.hydrateCustomAttributesForBO(ctx, tenantID, driverTable, rows[i])
	}
}

func normalizeDriverTableRef(drivingTable string) string {
	// "/mdm/product" → "mdm.product"; "mdm.product" unchanged; "product" → "public.product"
	t := drivingTable
	if len(t) > 0 && t[0] == '/' {
		t = t[1:]
		parts := splitPath(t)
		if len(parts) >= 2 {
			return parts[0] + "." + parts[1]
		}
		if len(parts) == 1 {
			return "public." + parts[0]
		}
	}
	if !containsDot(t) {
		return "public." + t
	}
	return t
}

func splitPath(s string) []string {
	var parts []string
	cur := ""
	for _, r := range s {
		if r == '/' {
			if cur != "" {
				parts = append(parts, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		parts = append(parts, cur)
	}
	return parts
}

func containsDot(s string) bool {
	for _, r := range s {
		if r == '.' {
			return true
		}
	}
	return false
}
