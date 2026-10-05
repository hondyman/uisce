package datapipeline

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type cubeMatStub struct {
	got []CubeMaterializeStartRequest
	err error
	res *CubeMaterializeResult
}

func (c *cubeMatStub) MaterializeCube(_ context.Context, r CubeMaterializeStartRequest) (*CubeMaterializeResult, error) {
	c.got = append(c.got, r)
	if c.res != nil {
		out := *c.res
		return &out, c.err
	}
	return &CubeMaterializeResult{
		CubeID:      r.CubeID,
		Started:     1,
		Grains:      1,
		WorkflowIDs: []string{"wf-1"},
		Summary:     "started=1 already_running=0 grains=1",
	}, c.err
}

type cubeMatFactory struct {
	src    Source
	cube   *cubeMatStub
	sink   Processor
	hasSrc bool
}

func (f cubeMatFactory) Source(Node) (Source, error) {
	if !f.hasSrc {
		return nil, errors.New("no source")
	}
	return f.src, nil
}
func (f cubeMatFactory) Processor(Node) (Processor, error) {
	if f.sink != nil {
		return f.sink, nil
	}
	return nil, errors.New("no processor")
}
func (f cubeMatFactory) CubeMaterializer() CubeMaterializer {
	if f.cube == nil {
		return nil
	}
	return f.cube
}

func actionOnlyCubeSpec() *Spec {
	return &Spec{Version: SpecVersion, Nodes: []Node{
		{ID: "mat", Type: NodeCubeMaterialize, Config: cfg(CubeMaterializeConfig{CubeID: "cube-1", Force: true})},
	}}
}

func chainedCubeSpec() *Spec {
	return &Spec{Version: SpecVersion, Nodes: []Node{
		{ID: "src", Type: NodeFileSource, Config: cfg(FileSourceConfig{URI: "u", Format: "csv"})},
		{ID: "load", Type: NodeStagingSink, Config: cfg(StagingSinkConfig{Table: "staging.bbg_price", SourceCd: "BLOOMBERG", Domain: "PRICE"})},
		{ID: "mat", Type: NodeCubeMaterialize, Config: cfg(CubeMaterializeConfig{
			CubeID: "cube-1",
			Grain:  []string{"account_id", "day"},
			FederationKeySamples: []CubeMaterializeKeySample{{
				LeftAlias: "pos", RightAlias: "acct", LeftKeys: 100, RightKeys: 100, Matched: 99,
			}},
		})},
	}, Edges: []Edge{{"src", "load"}, {"load", "mat"}}}
}

func TestCubeMaterializeValidateActionOnly(t *testing.T) {
	s := actionOnlyCubeSpec()
	if errs := s.Validate(); len(errs) != 0 {
		t.Fatalf("action-only cube_materialize must validate: %v", errs)
	}
	s.Nodes[0].Config = cfg(CubeMaterializeConfig{})
	if errs := s.Validate(); len(errs) == 0 || !strings.Contains(errors.Join(errs...).Error(), "cube_id is required") {
		t.Errorf("validation: %v", errs)
	}
}

func TestCubeMaterializeValidateRejectsOutputs(t *testing.T) {
	s := &Spec{Version: SpecVersion, Nodes: []Node{
		{ID: "mat", Type: NodeCubeMaterialize, Config: cfg(CubeMaterializeConfig{CubeID: "c1"})},
		{ID: "sink", Type: NodeFileSink, Config: cfg(FileSinkConfig{URI: "o", Format: "csv"})},
	}, Edges: []Edge{{"mat", "sink"}}}
	if errs := s.Validate(); len(errs) == 0 || !strings.Contains(errors.Join(errs...).Error(), "cannot have outputs") {
		t.Errorf("validation: %v", errs)
	}
}

func TestCubeMaterializeActionOnlyRuns(t *testing.T) {
	stub := &cubeMatStub{}
	f := cubeMatFactory{cube: stub}
	sum, err := Run(context.Background(), actionOnlyCubeSpec(), &RunContext{RunID: "run-1", TenantID: "t1"}, f, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(stub.got) != 1 {
		t.Fatalf("started %d time(s)", len(stub.got))
	}
	r := stub.got[0]
	if r.TenantID != "t1" || r.CubeID != "cube-1" || !r.Force || r.PipelineRunID != "run-1" {
		t.Errorf("request: %+v", r)
	}
	if len(sum.CubeMaterialize) != 1 || sum.CubeMaterialize[0].NodeID != "mat" || sum.CubeMaterialize[0].Started != 1 {
		t.Errorf("summary: %+v", sum.CubeMaterialize)
	}
	for _, s := range sum.Nodes {
		if s.NodeID == "mat" && s.Status != "COMPLETED" {
			t.Errorf("status %q", s.Status)
		}
	}
}

func TestCubeMaterializeChainedAfterLoad(t *testing.T) {
	committed := false
	stub := &cubeMatStub{}
	f := cubeMatFactory{
		hasSrc: true,
		src:    fakeSource{[]Row{{Num: 1, Data: map[string]any{"a": 1}}}},
		sink:   &stagingStub{id: "load", committed: &committed},
		cube:   stub,
	}
	sum, err := Run(context.Background(), chainedCubeSpec(), &RunContext{RunID: "run-2", TenantID: "t1"}, f, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !committed {
		t.Fatal("staging must commit before cube materialize")
	}
	if len(stub.got) != 1 {
		t.Fatalf("started %d time(s)", len(stub.got))
	}
	r := stub.got[0]
	if len(r.Grain) != 2 || r.Grain[0] != "account_id" || len(r.FederationKeySamples) != 1 {
		t.Errorf("request: %+v", r)
	}
	if sum.RecordsOut != 1 {
		t.Errorf("records out: %d", sum.RecordsOut)
	}
}

func TestCubeMaterializeSkippedInPreview(t *testing.T) {
	stub := &cubeMatStub{}
	f := cubeMatFactory{cube: stub}
	sum, err := Run(context.Background(), actionOnlyCubeSpec(), &RunContext{DryRun: true}, f, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(stub.got) != 0 || len(sum.CubeMaterialize) != 0 {
		t.Errorf("preview must not materialize: %+v", stub.got)
	}
	for _, s := range sum.Nodes {
		if s.NodeID == "mat" && s.Status != "SKIPPED" {
			t.Errorf("status %q in preview", s.Status)
		}
	}
}

func TestCubeMaterializeNeedsDependency(t *testing.T) {
	f := cubeMatFactory{}
	if _, err := Run(context.Background(), actionOnlyCubeSpec(), &RunContext{RunID: "r"}, f, nil); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("error: %v", err)
	}
}

func TestCubeMaterializeFailureSurfaces(t *testing.T) {
	stub := &cubeMatStub{err: errors.New("starrocks down")}
	f := cubeMatFactory{cube: stub}
	_, err := Run(context.Background(), actionOnlyCubeSpec(), &RunContext{RunID: "r"}, f, nil)
	if err == nil || !strings.Contains(err.Error(), "cube materialize failed") {
		t.Fatalf("error: %v", err)
	}
}

func TestPaletteCubeMaterializeAvailability(t *testing.T) {
	off := Palette(Deps{})
	var found *NodeType
	for i := range off {
		if off[i].Type == NodeCubeMaterialize {
			found = &off[i]
			break
		}
	}
	if found == nil {
		t.Fatal("cube_materialize missing from palette")
	}
	if found.Available {
		t.Error("should be unavailable without CubeMaterialize dep")
	}
	on := Palette(Deps{CubeMaterialize: &cubeMatStub{}})
	for _, n := range on {
		if n.Type == NodeCubeMaterialize && !n.Available {
			t.Error("should be available when CubeMaterialize is set")
		}
	}
}
