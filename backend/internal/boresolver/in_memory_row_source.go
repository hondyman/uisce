package boresolver

import "context"

// InMemoryRowSource is a RowSource (see host_runtime_executor.go) backed by
// rows already materialized in memory — e.g. records a datapipeline step
// already fetched from a source earlier in its DAG, rather than issuing a
// fresh SQL query. termKeys is accepted for RowSource interface
// compatibility but ignored: the caller is expected to have already
// selected the right columns before constructing this.
type InMemoryRowSource struct {
	RowsByEntity map[string][]CalcRow
}

func (s *InMemoryRowSource) FetchRows(ctx context.Context, tenantID string, termKeys []string) (map[string][]CalcRow, error) {
	return s.RowsByEntity, nil
}
