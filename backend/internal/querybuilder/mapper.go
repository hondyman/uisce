package querybuilder

import (
	"fmt"
	"strings"

	"github.com/hondyman/uisce/backend/internal/boresolver"
)

// mapQueryDefToSemanticRequest translates the frontend QueryDef contract into
// the existing SemanticSQLGenerationRequest consumed by BOSQLGenerator.
func mapQueryDefToSemanticRequest(
	qd *boresolver.QueryDef,
	boDef *boresolver.BODefinition,
	binding *boresolver.BOBinding,
) (*boresolver.SemanticSQLGenerationRequest, error) {
	if boDef == nil {
		return nil, fmt.Errorf("BO definition is nil")
	}

	req := &boresolver.SemanticSQLGenerationRequest{
		Datasource: boDef.DrivingTable,
		Limit:      qd.Query.Limit,
	}

	for _, dim := range qd.Query.Dimensions {
		field, err := resolveTermToField(boDef, dim.TermNodeID)
		if err != nil {
			return nil, err
		}
		req.Select = append(req.Select, boresolver.SemanticField{
			Term:  field.Name,
			Label: dim.Alias,
		})
	}

	for _, measure := range qd.Query.Measures {
		field, err := resolveTermToField(boDef, measure.TermNodeID)
		if err != nil {
			return nil, err
		}
		// The current generator does not have a first-class aggregation concept in
		// SemanticField. We encode the aggregation into the alias as a hint and rely
		// on the generator selecting the column. When the generator gains aggregation
		// support, this mapping should be updated.
		label := measure.Alias
		if measure.Aggregation != "" && strings.ToUpper(measure.Aggregation) != "NONE" {
			label = fmt.Sprintf("%s(%s)", strings.ToUpper(measure.Aggregation), label)
		}
		req.Select = append(req.Select, boresolver.SemanticField{
			Term:  field.Name,
			Label: label,
		})
	}

	for _, filter := range qd.Query.Filters {
		field, err := resolveTermToField(boDef, filter.TermNodeID)
		if err != nil {
			return nil, err
		}
		op := filter.Operator
		if op == "" {
			op = "="
		}
		req.Filters = append(req.Filters, boresolver.SemanticFilter{
			Term:  field.Name,
			Op:    op,
			Value: filter.Value,
		})
	}

	// binding is accepted for forward compatibility.
	_ = binding
	return req, nil
}

func resolveTermToField(boDef *boresolver.BODefinition, termNodeID string) (*boresolver.BOField, error) {
	termNodeID = strings.TrimSpace(termNodeID)
	if termNodeID == "" {
		return nil, fmt.Errorf("empty termNodeId")
	}

	for i := range boDef.Fields {
		f := &boDef.Fields[i]
		if strings.EqualFold(f.ID, termNodeID) ||
			strings.EqualFold(f.Name, termNodeID) ||
			strings.EqualFold(f.DisplayName, termNodeID) ||
			strings.EqualFold(f.SemanticTermID, termNodeID) {
			return f, nil
		}
	}

	return nil, fmt.Errorf("termNodeId %q not found in business object %s", termNodeID, boDef.ID)
}
