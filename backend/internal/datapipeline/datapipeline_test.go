package datapipeline

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func cfg(v any) json.RawMessage { b, _ := json.Marshal(v); return b }

func factsetSpec() *Spec {
	return &Spec{
		Version: SpecVersion, ErrorPolicy: ErrorPolicySkipAndLog, BatchSize: 2,
		Nodes: []Node{
			{ID: "src", Type: NodeFileSource, Config: cfg(FileSourceConfig{URI: "file:///x.txt", Format: "csv", Delimiter: "|"})},
			{ID: "val", Type: NodeValidate, Config: cfg(ValidateConfig{Required: []string{"fund_name"}, Unique: []string{"fsym_id"}})},
			{ID: "map", Type: NodeMap, Config: cfg(MapConfig{Fields: []FieldMap{
				{From: "fsym_id", To: "product_cd", Transform: "trim"},
				{From: "fund_name", To: "name"},
				{From: "inception_date", To: "inception_date", Transform: "to_date"},
				{From: "aum", To: "aum", Transform: "to_number"},
			}})},
			{ID: "sink", Type: NodeStagingSink, Config: cfg(StagingSinkConfig{Table: "staging.ff_product", SourceCd: "FACTSET", Domain: "PRODUCT"})},
		},
		Edges: []Edge{{"src", "val"}, {"val", "map"}, {"map", "sink"}},
	}
}

func TestSpecValid(t *testing.T) {
	if errs := factsetSpec().Validate(); len(errs) != 0 {
		t.Fatalf("expected valid, got %v", errs)
	}
}

func TestSpecProblems(t *testing.T) {
	s := factsetSpec()
	s.Edges = append(s.Edges, Edge{"sink", "src"}) // cycle + source with input
	s.Nodes[3].Config = cfg(StagingSinkConfig{Table: "public.users", SourceCd: "X", Domain: "Y"})
	errs := s.Validate()
	joined := ""
	for _, e := range errs {
		joined += e.Error() + "\n"
	}
	for _, want := range []string{"table must be staging.", "source cannot have inputs"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in:\n%s", want, joined)
		}
	}
}

func TestSpecCycle(t *testing.T) {
	s := &Spec{Version: SpecVersion, Nodes: []Node{
		{ID: "a", Type: NodeMap, Config: cfg(MapConfig{Fields: []FieldMap{{From: "x", To: "y"}}})},
		{ID: "b", Type: NodeMap, Config: cfg(MapConfig{Fields: []FieldMap{{From: "x", To: "y"}}})},
	}, Edges: []Edge{{"a", "b"}, {"b", "a"}}}
	if _, err := s.TopoOrder(); err == nil {
		t.Fatal("expected cycle error")
	}
}

// --- fakes ---

type fakeSource struct{ rows []Row }

func (f fakeSource) Stream(_ context.Context, _ *RunContext, bs int, emit func([]Row) error) error {
	for i := 0; i < len(f.rows); i += bs {
		if err := emit(f.rows[i:min(i+bs, len(f.rows))]); err != nil {
			return err
		}
	}
	return nil
}

type captureSink struct {
	got            []Row
	opened, closed bool
	closeErr       error
}

func (c *captureSink) Open(context.Context, *RunContext) error { c.opened = true; return nil }
func (c *captureSink) Close(_ context.Context, e error) error {
	c.closed, c.closeErr = true, e
	return nil
}
func (c *captureSink) Process(_ context.Context, rows []Row) (Result, error) {
	c.got = append(c.got, rows...)
	return Result{Out: rows}, nil
}

type fakeFactory struct {
	src  Source
	sink *captureSink
}

func (f fakeFactory) Source(Node) (Source, error) { return f.src, nil }
func (f fakeFactory) Processor(n Node) (Processor, error) {
	switch n.Type {
	case NodeValidate:
		return newValidateProc(n)
	case NodeMap:
		return newMapProc(n)
	}
	return f.sink, nil
}

type memRec struct {
	rejects []Reject
	warns   []Reject
	stats   []NodeStats
}

func (m *memRec) NodeDone(_ context.Context, s NodeStats) { m.stats = append(m.stats, s) }
func (m *memRec) Warned(_ context.Context, r Reject)      { m.warns = append(m.warns, r) }
func (m *memRec) Rejected(_ context.Context, r Reject)    { m.rejects = append(m.rejects, r) }

func fsRows() []Row {
	mk := func(n int, id, name, date, aum string) Row {
		return Row{Num: n, Data: map[string]any{"fsym_id": id, "fund_name": name, "inception_date": date, "aum": aum}}
	}
	return []Row{
		mk(1, " F1 ", "Vanguard 500", "1976-08-31", "450,000.50"),
		mk(2, "F2", "SPDR", "01/22/1993", "5200"),
		mk(3, "F3", "", "2000-01-01", "1"),          // missing name
		mk(4, "F2", "Dup", "2000-01-01", "1"),       // duplicate id
		mk(5, "F5", "Bad date", "not-a-date", "1"),  // bad date
		mk(6, "F6", "Bad aum", "2000-01-01", "abc"), // bad number
		mk(7, "F7", "Good", "2000-01-01", ""),       // blank aum ok
	}
}

func TestRunEndToEnd(t *testing.T) {
	sink := &captureSink{}
	rec := &memRec{}
	sum, err := Run(context.Background(), factsetSpec(), &RunContext{RunID: "r1", TenantID: "t1"},
		fakeFactory{src: fakeSource{fsRows()}, sink: sink}, rec)
	if err != nil {
		t.Fatal(err)
	}
	if sum.RecordsIn != 7 || sum.RecordsOut != 3 || sum.Errors != 4 {
		t.Fatalf("in=%d out=%d err=%d", sum.RecordsIn, sum.RecordsOut, sum.Errors)
	}
	if !sink.opened || !sink.closed || sink.closeErr != nil {
		t.Fatalf("sink lifecycle: %+v", sink)
	}
	if sink.got[0].Data["product_cd"] != "F1" || sink.got[0].Data["aum"] != 450000.5 || sink.got[0].Data["inception_date"] != "1976-08-31" {
		t.Errorf("row 1 transformed wrong: %v", sink.got[0].Data)
	}
	if sink.got[1].Data["inception_date"] != "1993-01-22" {
		t.Errorf("US date: %v", sink.got[1].Data)
	}
	if sink.got[2].Num != 7 {
		t.Errorf("source row number not preserved: %d", sink.got[2].Num)
	}
	reasons := ""
	for _, r := range rec.rejects {
		reasons += r.Reason + "\n"
	}
	for _, w := range []string{`"fund_name" is required`, "duplicate value", "not a recognised date", "is not a number"} {
		if !strings.Contains(reasons, w) {
			t.Errorf("missing reject reason %q in:\n%s", w, reasons)
		}
	}
	if len(rec.stats) != 4 {
		t.Errorf("want telemetry for 4 nodes, got %d", len(rec.stats))
	}
}

func TestRunFailFast(t *testing.T) {
	s := factsetSpec()
	s.ErrorPolicy = ErrorPolicyFailFast
	sink := &captureSink{}
	_, err := Run(context.Background(), s, &RunContext{RunID: "r"}, fakeFactory{src: fakeSource{fsRows()}, sink: sink}, nil)
	if err == nil {
		t.Fatal("expected fail-fast error")
	}
	if !sink.closed || sink.closeErr == nil {
		t.Errorf("sink must be told the run failed: %+v", sink)
	}
}

func TestRunRejectsInvalidSpec(t *testing.T) {
	if _, err := Run(context.Background(), &Spec{Version: 99}, &RunContext{}, fakeFactory{}, nil); err == nil {
		t.Fatal("expected invalid spec error")
	}
}

func TestFanOutDoesNotShareRows(t *testing.T) {
	a, b := &captureSink{}, &captureSink{}
	s := &Spec{Version: SpecVersion, Nodes: []Node{
		{ID: "src", Type: NodeFileSource, Config: cfg(FileSourceConfig{URI: "u", Format: "csv"})},
		{ID: "m1", Type: NodeMap, Config: cfg(MapConfig{Fields: []FieldMap{{From: "n", To: "n", Transform: "upper"}}})},
		{ID: "k1", Type: NodeFileSink, Config: cfg(FileSinkConfig{URI: "o1", Format: "csv"})},
		{ID: "k2", Type: NodeFileSink, Config: cfg(FileSinkConfig{URI: "o2", Format: "csv"})},
	}, Edges: []Edge{{"src", "m1"}, {"m1", "k1"}, {"m1", "k2"}}}
	f := &multiFactory{sinks: map[string]*captureSink{"k1": a, "k2": b},
		src: fakeSource{[]Row{{Num: 1, Data: map[string]any{"n": "x"}}}}}
	if _, err := Run(context.Background(), s, &RunContext{}, f, nil); err != nil {
		t.Fatal(err)
	}
	if len(a.got) != 1 || len(b.got) != 1 || a.got[0].Data["n"] != "X" {
		t.Fatalf("fan-out: %+v %+v", a.got, b.got)
	}
}

type multiFactory struct {
	src   Source
	sinks map[string]*captureSink
}

func (f *multiFactory) Source(Node) (Source, error) { return f.src, nil }
func (f *multiFactory) Processor(n Node) (Processor, error) {
	if n.Type == NodeMap {
		return newMapProc(n)
	}
	return f.sinks[n.ID], nil
}

func TestSuggestMapping(t *testing.T) {
	src := []Column{{Name: "fund_name", Type: "string"}, {Name: "isin", Type: "string"},
		{Name: "inception_date", Type: "string"}, {Name: "aum", Type: "string"}, {Name: "junk", Type: "string"}}
	tgt := []TargetField{{Name: "name", Label: "Fund Name", Type: "string"}, {Name: "isin", Type: "string"},
		{Name: "inceptionDate", Type: "date"}, {Name: "aum_amount", Type: "decimal"}}
	got := map[string]Suggestion{}
	for _, s := range SuggestMapping(src, tgt) {
		got[s.From] = s
	}
	if got["fund_name"].To != "name" || got["isin"].To != "isin" || got["isin"].Confidence < 0.99 {
		t.Errorf("bad suggestions: %+v", got)
	}
	if got["inception_date"].To != "inceptionDate" || got["inception_date"].Transform != "to_date" {
		t.Errorf("date suggestion: %+v", got["inception_date"])
	}
	if got["aum"].To != "aum_amount" || got["aum"].Transform != "to_number" {
		t.Errorf("aum suggestion: %+v", got["aum"])
	}
	if _, ok := got["junk"]; ok {
		t.Errorf("junk should not match: %+v", got["junk"])
	}
}

type fakeChecker struct{}

func (fakeChecker) Check(_ context.Context, _ string, _ []string, d map[string]any) ([]RuleFailure, error) {
	switch d["aum"] {
	case "block":
		return []RuleFailure{{RuleID: "r1", RuleName: "AUM positive", Severity: "BLOCK", Message: "aum must be > 0"}}, nil
	case "warn":
		return []RuleFailure{{RuleID: "r2", RuleName: "AUM round", Severity: "WARN", Message: "looks rounded"}}, nil
	}
	return nil, nil
}

func TestRuleCheckNode(t *testing.T) {
	n := Node{ID: "rc", Type: NodeRuleCheck, Config: cfg(RuleCheckConfig{RuleIDs: []string{"r1", "r2"}})}
	p, err := newRuleCheckProc(n, fakeChecker{})
	if err != nil {
		t.Fatal(err)
	}
	_ = p.Open(context.Background(), &RunContext{TenantID: "t"})
	res, _ := p.Process(context.Background(), []Row{
		{Num: 1, Data: map[string]any{"aum": "ok"}},
		{Num: 2, Data: map[string]any{"aum": "block"}},
		{Num: 3, Data: map[string]any{"aum": "warn"}},
	})
	if len(res.Out) != 2 || len(res.Rejected) != 1 || len(res.Warnings) != 1 {
		t.Fatalf("out=%d rej=%d warn=%d", len(res.Out), len(res.Rejected), len(res.Warnings))
	}
	if !strings.Contains(res.Rejected[0].Reason, `rule "AUM positive" failed`) {
		t.Errorf("reason: %s", res.Rejected[0].Reason)
	}
	if _, err := newRuleCheckProc(n, nil); err == nil {
		t.Error("nil checker must error")
	}
	bad := &Spec{Version: SpecVersion, Nodes: []Node{{ID: "x", Type: NodeRuleCheck, Config: cfg(RuleCheckConfig{})}}}
	if errs := validateNodeConfig(&bad.Nodes[0]); len(errs) == 0 {
		t.Error("empty rule list must be invalid")
	}
}
