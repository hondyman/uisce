package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/internal/calcengine"
)

// GLBalanceResolver is the domain port for historical general-ledger balance
// lookups. Downstream calculation engines (e.g., the Fee Accrual DSL evaluator)
// must use this port instead of issuing raw SELECT statements, ensuring every
// balance query respects the hot/cold watermark seam and tenant isolation.
type GLBalanceResolver interface {
	ResolveBalanceQuery(ctx context.Context, req GLBalanceRequest) (string, []interface{}, error)
}

// GLBalanceRequest parameterizes a balance lookup across the hot/cold seam.
type GLBalanceRequest struct {
	TenantID      string
	DatasourceID  string
	TableName     string
	DateColumn    string        // Defaults to "as_of_date"
	AsOfDate      time.Time     // Inclusive as-of date for the balance snapshot
	SelectColumns string        // Defaults to "*"
	WhereClause   string        // Optional extra filter (parenthesized automatically)
	WhereArgs     []interface{} // Args for placeholders inside WhereClause
	Limit         int           // Optional row limit
	Offset        int           // Optional row offset
}

type glBalanceResolver struct {
	dim *calcengine.DataIntegrityManager
}

// NewGLBalanceResolver creates a domain port backed by the DataIntegrityManager.
func NewGLBalanceResolver(dim *calcengine.DataIntegrityManager) GLBalanceResolver {
	return &glBalanceResolver{dim: dim}
}

// ResolveBalanceQuery builds a UNION-safe, tenant-scoped balance query that
// spans the operational hot tier and the historical cold tier.
func (r *glBalanceResolver) ResolveBalanceQuery(ctx context.Context, req GLBalanceRequest) (string, []interface{}, error) {
	if req.TenantID == "" {
		return "", nil, fmt.Errorf("tenant_id is required for GL balance resolution")
	}
	if req.TableName == "" {
		return "", nil, fmt.Errorf("table_name is required for GL balance resolution")
	}
	if req.DateColumn == "" {
		req.DateColumn = "as_of_date"
	}
	if req.SelectColumns == "" {
		req.SelectColumns = "*"
	}

	q := &calcengine.TierQuery{
		TableName:     req.TableName,
		TenantID:      req.TenantID,
		DatasourceID:  req.DatasourceID,
		DateColumn:    req.DateColumn,
		EndDate:       &req.AsOfDate,
		Mode:          calcengine.UnionSafe,
		SelectColumns: req.SelectColumns,
		WhereClause:   req.WhereClause,
		WhereArgs:     req.WhereArgs,
		Limit:         req.Limit,
		Offset:        req.Offset,
	}

	return r.dim.BuildSafeQuery(ctx, q)
}
