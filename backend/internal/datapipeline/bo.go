package datapipeline

import (
	"context"
	"fmt"
	"strings"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

// Condition is one BO source filter, in the rule engine's operator
// vocabulary (equals, in, between, contains, before, ...). It is pushed down
// to SQL by vm.CompileConditionSQL.
type Condition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    any    `json:"value,omitempty"`
}

// BOWriteRequest is one enforced bulk write (the /bo/{boKey}/records/bulk
// contract): every row is judged by the rule engine; rejected rows are
// reported and the rest commit.
type BOWriteRequest struct {
	Mode      string
	KeyFields []string
	DryRun    bool
	Records   []map[string]any
}

// BOWriteFailure is one row the write refused. Rules is set when the rule
// engine rejected it.
type BOWriteFailure struct {
	Index int
	Error string
	Rules []string
}

type BOWriteResult struct {
	Written int
	Failed  []BOWriteFailure
}

// BOClient reads and writes business-object records for a tenant through the
// BO records API's service layer - never direct SQL from the pipeline.
type BOClient interface {
	WriteBatch(ctx context.Context, tenantID, boKey string, req BOWriteRequest) (*BOWriteResult, error)
	// ReadPage returns up to limit records after skipping offset, in key order.
	ReadPage(ctx context.Context, tenantID, boKey string, filters []Condition, offset, limit int) ([]map[string]any, error)
}

// boWriteChunk stays under the bulk endpoint's 5000-row cap.
const boWriteChunk = 1000

type boSinkProc struct {
	cfg    BOSinkConfig
	client BOClient
	tenant string
}

func newBOSinkProc(n Node, c BOClient) (Processor, error) {
	var cfg BOSinkConfig
	if err := decodeConfig(n, &cfg); err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("business object writes are not configured for this environment")
	}
	return &boSinkProc{cfg: cfg, client: c}, nil
}

func (p *boSinkProc) Open(_ context.Context, rc *RunContext) error {
	p.tenant = rc.TenantID
	return nil
}
func (p *boSinkProc) Close(context.Context, error) error { return nil }

func (p *boSinkProc) Process(ctx context.Context, rows []Row) (Result, error) {
	var res Result
	for start := 0; start < len(rows); start += boWriteChunk {
		end := start + boWriteChunk
		if end > len(rows) {
			end = len(rows)
		}
		chunk := rows[start:end]
		recs := make([]map[string]any, len(chunk))
		for i, r := range chunk {
			recs[i] = r.Data
		}
		out, err := p.client.WriteBatch(ctx, p.tenant, p.cfg.BOKey, BOWriteRequest{
			Mode: p.cfg.Mode, KeyFields: p.cfg.KeyFields, DryRun: p.cfg.DryRun, Records: recs,
		})
		if err != nil {
			return res, fmt.Errorf("writing to %s: %w", p.cfg.BOKey, err)
		}
		failed := map[int]BOWriteFailure{}
		for _, f := range out.Failed {
			failed[f.Index] = f
		}
		for i, r := range chunk {
			f, bad := failed[i]
			if !bad {
				res.Out = append(res.Out, r)
				continue
			}
			reason := f.Error
			field := ""
			if len(f.Rules) > 0 {
				field = strings.Join(f.Rules, ", ")
				reason = "rejected by rule(s) " + field
			}
			res.Rejected = append(res.Rejected, Reject{Row: r, Field: field, Reason: reason})
		}
	}
	return res, nil
}

// boReadPage is the source page size.
const boReadPage = 1000

type boSource struct {
	cfg    BOSourceConfig
	client BOClient
}

func newBOSource(n Node, c BOClient) (Source, error) {
	var cfg BOSourceConfig
	if err := decodeConfig(n, &cfg); err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("business object reads are not configured for this environment")
	}
	return &boSource{cfg: cfg, client: c}, nil
}

func (s *boSource) Stream(ctx context.Context, rc *RunContext, batchSize int, emit func([]Row) error) error {
	page := boReadPage
	if batchSize > 0 && batchSize < page {
		page = batchSize
	}
	num := 0
	for offset := 0; ; offset += page {
		want := page
		if s.cfg.Limit > 0 && offset+want > s.cfg.Limit {
			want = s.cfg.Limit - offset
		}
		if want <= 0 {
			return nil
		}
		recs, err := s.client.ReadPage(ctx, rc.TenantID, s.cfg.BOKey, s.cfg.Filters, offset, want)
		if err != nil {
			return fmt.Errorf("reading %s: %w", s.cfg.BOKey, err)
		}
		if len(recs) == 0 {
			return nil
		}
		rows := make([]Row, len(recs))
		for i, r := range recs {
			num++
			rows[i] = Row{Num: num, Data: r}
		}
		if err := emit(rows); err != nil {
			return err
		}
		if len(recs) < want {
			return nil
		}
	}
}

// validateConditions checks filters at design time: each needs a field and
// an operator the engine can push down to SQL with the given value.
func validateConditions(cs []Condition) []error {
	var errs []error
	for i, c := range cs {
		if c.Field == "" {
			errs = append(errs, fmt.Errorf("filter %d: field is required", i+1))
			continue
		}
		if _, err := vm.CompileConditionSQL("c", c.Operator, c.Value, func(any) string { return "?" }); err != nil {
			errs = append(errs, fmt.Errorf("filter on %q: %v", c.Field, err))
		}
	}
	return errs
}
