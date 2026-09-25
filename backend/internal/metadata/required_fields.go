package metadata

import (
	"context"
	"fmt"
	"strings"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/models"
)

// requiredField is a BO field every written record must carry: a field
// marked is_required, resolved to its physical column. (binding_requirement
// is a different thing - whether a storage tier must map the field - and
// says nothing about whether a value may be empty.)
type requiredField struct {
	Label  string
	Column string
}

// RequiredFieldsError rejects a write that leaves required fields empty.
// Handlers answer it like a rule rejection (422).
type RequiredFieldsError struct {
	Fields []string `json:"fields"` // "Issuer (issuer_id)"
}

func (e *RequiredFieldsError) Error() string {
	return "missing required field(s): " + strings.Join(e.Fields, ", ")
}

// requiredFields loads the BO's required fields. It fails closed: if the
// definitions cannot be read, the write is refused rather than let through
// unchecked.
func (s *BusinessObjectService) requiredFields(ctx context.Context, bo *models.BusinessObjectDefinition) ([]requiredField, error) {
	var rows []struct {
		Name  string `db:"field_name"`
		Label string `db:"label"`
	}
	if err := s.db.SelectContext(ctx, &rows, `
		SELECT field_name, COALESCE(NULLIF(display_name, ''), field_name) AS label
		FROM public.business_object_fields
		WHERE bo_id::text = $1
		  AND COALESCE(is_required, false)
		ORDER BY display_order NULLS LAST, field_name`, bo.ID); err != nil {
		return nil, fmt.Errorf("loading required fields for business object %s: %w", bo.Key, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	// Semantic field -> physical column; a field with no mapping is its own
	// column name (the same fallback the schema endpoint uses).
	fieldMap, err := analytics.ResolveSemanticFieldMap(ctx, s.db, bo.ID, bo.DriverTableName)
	if err != nil {
		return nil, fmt.Errorf("resolving required field columns for business object %s: %w", bo.Key, err)
	}
	out := make([]requiredField, 0, len(rows))
	for _, r := range rows {
		col := fieldMap[r.Name]
		if col == "" {
			col = r.Name
		}
		out = append(out, requiredField{Label: r.Label, Column: col})
	}
	return out, nil
}

// missingRequired checks the row as written - after database defaults, and
// for an update including the fields it did not touch. A required field
// whose column is not in the written row cannot be judged here (its
// mapping points elsewhere) and is logged, not failed.
func missingRequired(boKey string, fields []requiredField, row map[string]interface{}) *RequiredFieldsError {
	var missing []string
	for _, f := range fields {
		v, ok := row[f.Column]
		if !ok {
			logging.GetLogger().Sugar().Warnf("required field %q of business object %s maps to column %q, which the written row does not have", f.Label, boKey, f.Column)
			continue
		}
		if blank(v) {
			label := f.Label
			if !strings.EqualFold(label, f.Column) {
				label += " (" + f.Column + ")"
			}
			missing = append(missing, label)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return &RequiredFieldsError{Fields: missing}
}

func blank(v interface{}) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(t) == ""
	case []byte:
		return strings.TrimSpace(string(t)) == ""
	}
	return false
}
