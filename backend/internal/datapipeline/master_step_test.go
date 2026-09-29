package datapipeline

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// stagingStub claims a load run like the staging sink and records when it
// committed (Close).
type stagingStub struct {
	captureSink
	id        string
	committed *bool
}

func (s *stagingStub) Open(_ context.Context, rc *RunContext) error {
	if !rc.DryRun {
		rc.Loads[s.id] = "load-1"
	}
	return nil
}
func (s *stagingStub) Close(context.Context, error) error { *s.committed = true; return nil }

type masterStub struct {
	committed *bool
	got       []MasterRequest
	err       error
	afterSink bool
	result    *MasterResult // overrides the default success result
}

func (m *masterStub) MasterLoad(_ context.Context, r MasterRequest) (*MasterResult, error) {
	m.got = append(m.got, r)
	m.afterSink = *m.committed
	if m.result != nil {
		return m.result, m.err
	}
	return &MasterResult{Entity: r.Entity, RunID: "mr-1", Status: "COMPLETED", Records: 2, Published: 2}, m.err
}

type masterFactory struct {
	src     Source
	staging *stagingStub
	master  *masterStub
}

func (f masterFactory) Source(Node) (Source, error)       { return f.src, nil }
func (f masterFactory) Processor(Node) (Processor, error) { return f.staging, nil }
func (f masterFactory) Masterer() Masterer {
	if f.master == nil {
		return nil
	}
	return f.master
}

func masterSpec() *Spec {
	return &Spec{Version: SpecVersion, Nodes: []Node{
		{ID: "src", Type: NodeFileSource, Config: cfg(FileSourceConfig{URI: "u", Format: "csv"})},
		{ID: "load", Type: NodeStagingSink, Config: cfg(StagingSinkConfig{Table: "staging.bbg_price", SourceCd: "BLOOMBERG", Domain: "PRICE"})},
		{ID: "master", Type: NodeMaster, Config: cfg(MasterConfig{Entity: "Price"})},
	}, Edges: []Edge{{"src", "load"}, {"load", "master"}}}
}

func newMasterFactory(masterErr error) (masterFactory, *masterStub) {
	committed := false
	m := &masterStub{committed: &committed, err: masterErr}
	return masterFactory{src: fakeSource{[]Row{{Num: 1, Data: map[string]any{"a": 1}}, {Num: 2, Data: map[string]any{"a": 2}}}},
		staging: &stagingStub{id: "load", committed: &committed}, master: m}, m
}

func TestMasterStepRunsAfterTheLoadCommits(t *testing.T) {
	f, m := newMasterFactory(nil)
	sum, err := Run(context.Background(), masterSpec(), &RunContext{RunID: "run-1", TenantID: "t1"}, f, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.got) != 1 || !m.afterSink {
		t.Fatalf("mastered %d time(s), after the commit: %v", len(m.got), m.afterSink)
	}
	if r := m.got[0]; r.Entity != "price" || r.StagingTable != "staging.bbg_price" || r.LoadRunID != "load-1" || r.PipelineRunID != "run-1" || r.TenantID != "t1" {
		t.Errorf("request: %+v", r)
	}
	if len(sum.Mastering) != 1 || sum.Mastering[0].RunID != "mr-1" || sum.Mastering[0].NodeID != "master" || sum.Mastering[0].LoadRunID != "load-1" {
		t.Errorf("summary: %+v", sum.Mastering)
	}
	// The staging sink is still the sink: its rows count as the run's output.
	if sum.RecordsOut != 2 {
		t.Errorf("records out: %d", sum.RecordsOut)
	}
}

func TestMasterStepSkippedInPreview(t *testing.T) {
	f, m := newMasterFactory(nil)
	sum, err := Run(context.Background(), masterSpec(), &RunContext{DryRun: true}, f, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.got) != 0 || len(sum.Mastering) != 0 {
		t.Errorf("a preview mastered: %+v", m.got)
	}
	for _, s := range sum.Nodes {
		if s.NodeID == "master" && s.Status != "SKIPPED" {
			t.Errorf("master step status %q in a preview", s.Status)
		}
	}
}

func TestMasterStepFailureKeepsTheLoad(t *testing.T) {
	f, _ := newMasterFactory(errors.New("no profile"))
	_, err := Run(context.Background(), masterSpec(), &RunContext{RunID: "r"}, f, nil)
	if err == nil || !strings.Contains(err.Error(), "load is committed") {
		t.Fatalf("error: %v", err)
	}
}

func TestMasterStepNeedsMastering(t *testing.T) {
	f, _ := newMasterFactory(nil)
	f.master = nil
	if _, err := Run(context.Background(), masterSpec(), &RunContext{RunID: "r"}, f, nil); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("error: %v", err)
	}
}

func TestMasterStepFailsWhenEveryRecordIsRejected(t *testing.T) {
	// The engine reports a run that rejected everything as PARTIAL, not as an
	// error, so it used to finish COMPLETED with zero published. That is how a
	// broken field mapping stayed invisible.
	f, _ := newMasterFactory(nil)
	f.master.result = &MasterResult{Entity: "security", RunID: "mr-1", Status: "PARTIAL",
		Records: 12, Valid: 0, Invalid: 12, Published: 0}
	sum, err := Run(context.Background(), masterSpec(), &RunContext{RunID: "r"}, f, nil)
	if err == nil || !strings.Contains(err.Error(), "rejected all 12 records") {
		t.Fatalf("a run that published nothing must fail the node: %v", err)
	}
	for _, s := range sum.Nodes {
		if s.NodeID == "master" && s.Status != "FAILED" {
			t.Errorf("master step status %q, want FAILED", s.Status)
		}
	}
}

func TestMasterStepToleratesSomeRejectedRecords(t *testing.T) {
	// Partial rejection is ordinary data quality and must stay COMPLETED.
	f, _ := newMasterFactory(nil)
	f.master.result = &MasterResult{Entity: "security", RunID: "mr-1", Status: "PARTIAL",
		Records: 12, Valid: 9, Invalid: 3, Published: 9, Exceptions: 3}
	sum, err := Run(context.Background(), masterSpec(), &RunContext{RunID: "r"}, f, nil)
	if err != nil {
		t.Fatalf("some rejected records must not fail the run: %v", err)
	}
	for _, s := range sum.Nodes {
		if s.NodeID == "master" && s.Status != "COMPLETED" {
			t.Errorf("master step status %q, want COMPLETED", s.Status)
		}
	}
}

func TestMasterStepMustFollowAStagingLoad(t *testing.T) {
	s := masterSpec()
	s.Nodes[1] = Node{ID: "load", Type: NodeFileSink, Config: cfg(FileSinkConfig{URI: "o", Format: "csv"})}
	if errs := s.Validate(); len(errs) == 0 || !strings.Contains(errors.Join(errs...).Error(), "must follow a staging load") {
		t.Errorf("validation: %v", errs)
	}
	s = masterSpec()
	s.Nodes[2].Config = cfg(MasterConfig{})
	if errs := s.Validate(); len(errs) == 0 || !strings.Contains(errors.Join(errs...).Error(), "entity is required") {
		t.Errorf("validation: %v", errs)
	}
}
