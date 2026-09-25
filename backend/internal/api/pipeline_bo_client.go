package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/uisce/backend/internal/datapipeline"
	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
	"github.com/lib/pq"
)

// pipelineBOClient gives the data pipeline the /bo/{boKey}/records API's
// behaviour in-process: writes are the enforced bulk write (every row judged
// by the rule engine), reads are tenant-scoped like HandleListBORecords.
type pipelineBOClient struct{ h *BOCRUDHandler }

// NewPipelineBOClient adapts the BO records API for datapipeline.
func NewPipelineBOClient(h *BOCRUDHandler) datapipeline.BOClient { return &pipelineBOClient{h: h} }

func (c *pipelineBOClient) WriteBatch(ctx context.Context, tenantID, boKey string, req datapipeline.BOWriteRequest) (*datapipeline.BOWriteResult, error) {
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant id: %w", err)
	}
	recs := make([]map[string]interface{}, len(req.Records))
	for i, r := range req.Records {
		recs[i] = r
	}
	resp, err := c.h.bulkWrite(ctx, tid, boKey, "", boBulkRequest{Mode: req.Mode, KeyFields: req.KeyFields, DryRun: req.DryRun, Records: recs})
	if err != nil {
		return nil, err
	}
	out := &datapipeline.BOWriteResult{Written: resp.Written}
	for _, f := range resp.Failed {
		out.Failed = append(out.Failed, datapipeline.BOWriteFailure{Index: f.Index, Error: f.Error, Rules: f.Rules})
	}
	return out, nil
}

func (c *pipelineBOClient) ReadPage(ctx context.Context, tenantID, boKey string, filters []datapipeline.Condition, offset, limit int) ([]map[string]any, error) {
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant id: %w", err)
	}
	db, q, args, err := c.h.buildPipelineRead(ctx, tid, boKey, filters, offset, limit)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryxContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("reading records: %w", err)
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		item := map[string]interface{}{}
		if err := rows.MapScan(item); err != nil {
			return nil, err
		}
		cleanScanResult(item)
		out = append(out, item)
	}
	return out, rows.Err()
}

// buildPipelineRead builds a tenant-scoped page query. Filter fields must be
// real columns of the driving table; operators and values are the rule
// engine's (vm.CompileConditionSQL), always bound, never interpolated.
func (h *BOCRUDHandler) buildPipelineRead(ctx context.Context, tenantID uuid.UUID, boKey string, filters []datapipeline.Condition, offset, limit int) (*sqlx.DB, string, []interface{}, error) {
	boMeta, err := h.resolveBOMetadata(ctx, boKey, tenantID)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed resolving BO contract: %w", err)
	}
	cols, err := h.resolveWritableColumns(ctx, boMeta.RecordsDB, boMeta.DrivingTable)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed resolving table schema: %w", err)
	}
	var where []string
	var args []interface{}
	param := func(v interface{}) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	if cols["tenant_id"] {
		where = append(where, "tenant_id = "+param(tenantID))
	}
	for _, f := range filters {
		if !cols[f.Field] {
			return nil, "", nil, fmt.Errorf("unknown field '%s' on %s", f.Field, boKey)
		}
		pred, err := vm.CompileConditionSQL(pq.QuoteIdentifier(f.Field), f.Operator, f.Value, param)
		if err != nil {
			return nil, "", nil, fmt.Errorf("filter on %q: %w", f.Field, err)
		}
		where = append(where, pred)
	}
	whereSQL := "TRUE"
	if len(where) > 0 {
		whereSQL = strings.Join(where, " AND ")
	}
	lim, off := param(limit), param(offset)
	return boMeta.RecordsDB, fmt.Sprintf(`SELECT * FROM %s WHERE %s ORDER BY %s ASC LIMIT %s OFFSET %s`,
		boMeta.DrivingTable, whereSQL, pq.QuoteIdentifier(boMeta.KeyColumn), lim, off), args, nil
}
