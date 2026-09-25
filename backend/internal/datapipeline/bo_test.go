package datapipeline

import (
	"context"
	"strings"
	"testing"
)

type fakeBO struct {
	writes [][]map[string]any
	store  []map[string]any
	reads  [][2]int
}

func (f *fakeBO) WriteBatch(_ context.Context, _, _ string, req BOWriteRequest) (*BOWriteResult, error) {
	f.writes = append(f.writes, req.Records)
	res := &BOWriteResult{}
	for i, r := range req.Records {
		if r["aum"] == -1 {
			res.Failed = append(res.Failed, BOWriteFailure{Index: i, Error: "rejected", Rules: []string{"AUM positive"}})
			continue
		}
		res.Written++
	}
	return res, nil
}

func (f *fakeBO) ReadPage(_ context.Context, _, _ string, _ []Condition, offset, limit int) ([]map[string]any, error) {
	f.reads = append(f.reads, [2]int{offset, limit})
	if offset >= len(f.store) {
		return nil, nil
	}
	end := offset + limit
	if end > len(f.store) {
		end = len(f.store)
	}
	return f.store[offset:end], nil
}

func TestBOSinkChunksAndAttributesRuleRejections(t *testing.T) {
	bo := &fakeBO{}
	p, err := newBOSinkProc(Node{ID: "s", Type: NodeBOSink, Config: cfg(BOSinkConfig{BOKey: "fund"})}, bo)
	if err != nil {
		t.Fatal(err)
	}
	_ = p.Open(context.Background(), &RunContext{TenantID: "t"})
	rows := make([]Row, boWriteChunk+5)
	for i := range rows {
		rows[i] = Row{Num: i + 1, Data: map[string]any{"aum": i}}
	}
	rows[boWriteChunk+2].Data["aum"] = -1
	res, err := p.Process(context.Background(), rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(bo.writes) != 2 || len(bo.writes[1]) != 5 {
		t.Fatalf("expected 2 chunks, got %d", len(bo.writes))
	}
	if len(res.Out) != len(rows)-1 || len(res.Rejected) != 1 {
		t.Fatalf("out=%d rejected=%d", len(res.Out), len(res.Rejected))
	}
	rj := res.Rejected[0]
	if rj.Row.Num != boWriteChunk+3 || !strings.Contains(rj.Reason, "AUM positive") {
		t.Errorf("reject attributed to wrong row or rule: %+v", rj)
	}
	if _, err := newBOSinkProc(Node{ID: "s", Type: NodeBOSink, Config: cfg(BOSinkConfig{BOKey: "fund"})}, nil); err == nil {
		t.Error("nil client must error")
	}
}

func TestBOSourcePagesToLimit(t *testing.T) {
	bo := &fakeBO{}
	for i := 0; i < 25; i++ {
		bo.store = append(bo.store, map[string]any{"i": i})
	}
	src, _ := newBOSource(Node{ID: "b", Type: NodeBOSource, Config: cfg(BOSourceConfig{BOKey: "fund", Limit: 22})}, bo)
	var got []Row
	err := src.Stream(context.Background(), &RunContext{TenantID: "t"}, 10, func(rs []Row, _ []Reject) error {
		got = append(got, rs...)
		return nil
	})
	if err != nil || len(got) != 22 || got[21].Num != 22 {
		t.Fatalf("got %d rows, err %v", len(got), err)
	}
	if bo.reads[2] != [2]int{20, 2} {
		t.Errorf("last page should request only the remaining 2 rows: %v", bo.reads)
	}
}

func TestBOSourceFiltersValidatedAtDesignTime(t *testing.T) {
	good := Node{ID: "b", Type: NodeBOSource, Config: cfg(BOSourceConfig{BOKey: "fund", Filters: []Condition{
		{Field: "status", Operator: "in", Value: []any{"open", "closed"}},
		{Field: "inception", Operator: "before", Value: "2020-01-01"},
	}})}
	if errs := validateNodeConfig(&good); len(errs) != 0 {
		t.Fatalf("valid filters rejected: %v", errs)
	}
	bad := Node{ID: "b", Type: NodeBOSource, Config: cfg(BOSourceConfig{BOKey: "fund", Filters: []Condition{
		{Field: "", Operator: "equals", Value: 1},
		{Field: "aum", Operator: "length_equals", Value: 3}, // VM-only: no SQL pushdown
		{Field: "aum", Operator: ">", Value: []any{1, 2}},
	}})}
	if errs := validateNodeConfig(&bad); len(errs) != 3 {
		t.Fatalf("want 3 filter errors, got %v", errs)
	}
}

var _ Factory = Deps{}

// End to end through Run: a BO source feeding a rule check and a BO sink.
func TestRunBOToBO(t *testing.T) {
	src := &fakeBO{}
	for i := 0; i < 5; i++ {
		src.store = append(src.store, map[string]any{"aum": i})
	}
	spec := &Spec{Version: SpecVersion, Nodes: []Node{
		{ID: "in", Type: NodeBOSource, Config: cfg(BOSourceConfig{BOKey: "fund"})},
		{ID: "rc", Type: NodeRuleCheck, Config: cfg(RuleCheckConfig{RuleIDs: []string{"r"}})},
		{ID: "out", Type: NodeBOSink, Config: cfg(BOSinkConfig{BOKey: "fund_copy"})},
	}, Edges: []Edge{{From: "in", To: "rc"}, {From: "rc", To: "out"}}}
	sum, err := Run(context.Background(), spec, &RunContext{TenantID: "t", RunID: "r1"},
		Deps{BO: src, Rules: blockOdd{}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sum.RecordsIn != 5 || sum.RecordsOut != 3 || sum.Errors != 2 {
		t.Fatalf("summary %+v", sum)
	}
	if got := len(src.writes[0]); got != 3 {
		t.Errorf("sink should receive the 3 rows that passed, got %d", got)
	}
}

type blockOdd struct{}

func (blockOdd) Check(_ context.Context, _ string, _ []string, d map[string]any) ([]RuleFailure, error) {
	if d["aum"].(int)%2 == 1 {
		return []RuleFailure{{RuleName: "even", Severity: "BLOCK", Message: "odd"}}, nil
	}
	return nil, nil
}

func TestPreviewLimitsSamplesAndWritesNothing(t *testing.T) {
	src := &fakeBO{}
	for i := 0; i < 50; i++ {
		src.store = append(src.store, map[string]any{"aum": i})
	}
	spec := &Spec{Version: SpecVersion, Nodes: []Node{
		{ID: "in", Type: NodeBOSource, Config: cfg(BOSourceConfig{BOKey: "fund"})},
		{ID: "rc", Type: NodeRuleCheck, Config: cfg(RuleCheckConfig{RuleIDs: []string{"r"}})},
		{ID: "out", Type: NodeBOSink, Config: cfg(BOSinkConfig{BOKey: "fund_copy"})},
	}, Edges: []Edge{{From: "in", To: "rc"}, {From: "rc", To: "out"}}, BatchSize: 4}
	dry := &dryBO{fakeBO: src}
	sum, err := Run(context.Background(), spec, &RunContext{TenantID: "t", MaxRows: 10, SampleRows: 3, DryRun: true},
		Deps{BO: dry, Rules: blockOdd{}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sum.RecordsIn != 10 || sum.Errors != 5 {
		t.Fatalf("preview should stop at 10 rows: %+v", sum)
	}
	if len(sum.Samples["in"]) != 3 || len(sum.Samples["rc"]) != 3 || sum.Samples["rc"][1].Data["aum"] != 2 {
		t.Fatalf("samples: %+v", sum.Samples)
	}
	if !dry.allDry {
		t.Error("preview BO writes must be dry runs")
	}
}

type dryBO struct {
	*fakeBO
	allDry bool
}

func (d *dryBO) WriteBatch(ctx context.Context, t, k string, req BOWriteRequest) (*BOWriteResult, error) {
	d.allDry = req.DryRun
	return d.fakeBO.WriteBatch(ctx, t, k, req)
}
