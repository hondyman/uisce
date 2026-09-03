package datapipeline

import (
	"context"
	"fmt"

	"github.com/hondyman/uisce/backend/internal/boresolver"
)

// CalcDefinition is the JSON-config shape for one calc node in a
// host_runtime_calc pipeline tile's "nodes" array.
type CalcDefinition struct {
	TermKey      string   `json:"term_key"`
	Formula      string   `json:"formula"`
	Dependencies []string `json:"dependencies"`
}

// HostRuntimeCalcTransformer runs boresolver host-runtime calcs (XIRR and
// anything else that can't compile to SQL — see calc_functions.go) against
// records already fetched earlier in the pipeline, and emits one output
// record per entity carrying every calc's result as a column.
//
// This is the fix for a real gap in the calc engine: a pushdown formula
// can't reference a host-runtime calc's output directly, because that
// value never exists as a SQL column (see the tier-propagation note in
// boresolver/calc_compiler.go). Running this tile in a pipeline and
// loading its output into an ordinary table via the existing
// bo_loader/catalog_loader steps MATERIALIZES the result as a normal
// column — which a later pushdown query CAN reference (register it as a
// SemanticTerm/CatalogEdge mapping onto that table, same as any other base
// field). It's also how "official" published calc values should be
// produced on a schedule rather than recomputed per request.
type HostRuntimeCalcTransformer struct {
	Nodes       []*boresolver.CalcNode
	EntityField string // PipelineRecord field that groups input rows into entities
	TenantID    string
}

func (h *HostRuntimeCalcTransformer) Transform(ctx context.Context, input []PipelineRecord) ([]PipelineRecord, []string, error) {
	if h.EntityField == "" {
		return nil, nil, fmt.Errorf("HostRuntimeCalcTransformer: EntityField is required")
	}
	if h.TenantID == "" {
		return nil, nil, fmt.Errorf("HostRuntimeCalcTransformer: TenantID is required")
	}
	if len(h.Nodes) == 0 {
		return input, nil, nil
	}

	rowsByEntity := make(map[string][]boresolver.CalcRow)
	for _, rec := range input {
		entityID := fmt.Sprintf("%v", rec[h.EntityField])
		if entityID == "" || entityID == "<nil>" {
			continue
		}
		row := make(boresolver.CalcRow, len(rec))
		for k, v := range rec {
			row[k] = v
		}
		rowsByEntity[entityID] = append(rowsByEntity[entityID], row)
	}

	executor := &boresolver.HostRuntimeExecutor{
		Rows: &boresolver.InMemoryRowSource{RowsByEntity: rowsByEntity},
	}

	results, err := executor.Execute(ctx, h.TenantID, h.Nodes)
	if err != nil {
		return nil, nil, err
	}

	byEntity := make(map[string]PipelineRecord)
	var errs []string
	for _, r := range results {
		out, ok := byEntity[r.EntityID]
		if !ok {
			out = PipelineRecord{h.EntityField: r.EntityID, "tenant_id": h.TenantID}
			byEntity[r.EntityID] = out
		}
		if r.Err != nil {
			errs = append(errs, fmt.Sprintf("entity %s: %s: %v", r.EntityID, r.TermKey, r.Err))
			continue
		}
		out[r.TermKey] = r.Value
	}

	output := make([]PipelineRecord, 0, len(byEntity))
	for _, rec := range byEntity {
		output = append(output, rec)
	}
	return output, errs, nil
}
