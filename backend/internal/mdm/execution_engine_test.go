package mdm

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

type fakeGraph struct {
	nodes map[uuid.UUID]*analytics.SemanticNode
	edges map[uuid.UUID][]analytics.SemanticEdge
}

func (g *fakeGraph) GetNodeByID(id uuid.UUID) (*analytics.SemanticNode, error) {
	return g.nodes[id], nil
}
func (g *fakeGraph) GetOutgoingEdges(id uuid.UUID) ([]analytics.SemanticEdge, error) {
	return g.edges[id], nil
}

func (g *fakeGraph) add(name string, props, cfg map[string]interface{}) uuid.UUID {
	id := uuid.New()
	g.nodes[id] = &analytics.SemanticNode{ID: id, NodeName: name, Properties: props, Config: cfg}
	return id
}
func (g *fakeGraph) dep(from, to uuid.UUID, edgeType string) {
	g.edges[from] = append(g.edges[from], analytics.SemanticEdge{SourceNodeID: from, TargetNodeID: to, EdgeType: analytics.EdgeType(edgeType)})
}

func newGraph() *fakeGraph {
	return &fakeGraph{nodes: map[uuid.UUID]*analytics.SemanticNode{}, edges: map[uuid.UUID][]analytics.SemanticEdge{}}
}

// NAV -> PositionValue (calc) -> Quantity, Price (context leaves), plus a
// Liabilities leaf on NAV itself. Exercises every expression source.
func TestExecuteCalculation_RecursiveWithVM(t *testing.T) {
	ast, err := vm.ParseExpression("PositionValue - Liabilities")
	if err != nil {
		t.Fatal(err)
	}
	var astJSON map[string]interface{}
	b, _ := json.Marshal(ast)
	_ = json.Unmarshal(b, &astJSON)

	for _, tc := range []struct {
		name   string
		navCfg map[string]interface{}
		navPrp map[string]interface{}
	}{
		{"config.rule_ast", map[string]interface{}{"rule_ast": astJSON}, nil},
		{"config.expression", map[string]interface{}{"expression": "PositionValue - Liabilities"}, nil},
		{"legacy properties.expression with assignment", nil, map[string]interface{}{"expression": "NAV = PositionValue - Liabilities", "engine": "wasm"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newGraph()
			qty := g.add("Quantity", nil, nil)
			price := g.add("Price", nil, nil)
			liab := g.add("Liabilities", nil, nil)
			pos := g.add("PositionValue", nil, map[string]interface{}{"expression": "Quantity * Price"})
			nav := g.add("NetAssetValue", tc.navPrp, tc.navCfg)
			g.dep(pos, qty, "calc_depends_on_term")
			g.dep(pos, price, "calc_depends_on_term")
			g.dep(nav, pos, "calc_depends_on_calc")
			g.dep(nav, liab, "calc_depends_on_term")

			e := &ExecutionEngine{graphService: g}
			res, trace, err := e.ExecuteCalculation(context.Background(), nav,
				map[string]interface{}{"Quantity": 10, "Price": 15.5, "Liabilities": 5.0})
			if err != nil {
				t.Fatalf("err: %v (trace %+v)", err, trace)
			}
			if res != 150.0 {
				t.Fatalf("NAV = %v, want 150 (10*15.5 - 5)", res)
			}
			if trace.Dependencies["PositionValue"].Output != 155.0 {
				t.Errorf("PositionValue trace = %+v", trace.Dependencies["PositionValue"])
			}
			if _, ok := trace.Dependencies["PositionValue"].Dependencies["Quantity"]; !ok {
				t.Errorf("nested trace missing Quantity: %+v", trace.Dependencies["PositionValue"])
			}
		})
	}
}

// The seeded NAV term (migrations/019) is "NAV = sum(PositionValue)".
func TestExecuteCalculation_SeededNAVExpression(t *testing.T) {
	g := newGraph()
	pv := g.add("PositionValue", nil, nil)
	nav := g.add("NetAssetValue", map[string]interface{}{"expression": "NAV = sum(PositionValue)", "engine": "wasm"}, nil)
	g.dep(nav, pv, "calc_depends_on_term")
	res, _, err := (&ExecutionEngine{graphService: g}).ExecuteCalculation(context.Background(), nav, map[string]interface{}{"PositionValue": 42.0})
	if err != nil || res != 42.0 {
		t.Fatalf("res=%v err=%v", res, err)
	}
}

// The replaced engine returned 0.0 for anything it did not understand.
// Every one of these must now be an error naming the term.
func TestExecuteCalculation_FailsLoudNeverZero(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		build      func(g *fakeGraph) uuid.UUID
	}{
		{"no expression", `term "Orphan"`, func(g *fakeGraph) uuid.UUID {
			return g.add("Orphan", map[string]interface{}{"engine": "mock"}, nil)
		}},
		{"unparsable expression", `term "Bad"`, func(g *fakeGraph) uuid.UUID {
			return g.add("Bad", nil, map[string]interface{}{"expression": "Quantity * * Price"})
		}},
		{"leaf missing from context", `term "Price"`, func(g *fakeGraph) uuid.UUID {
			price := g.add("Price", nil, nil)
			pos := g.add("PositionValue", nil, map[string]interface{}{"expression": "Price * 2"})
			g.dep(pos, price, "calc_depends_on_term")
			return pos
		}},
		{"unknown field in expression", `term "Typo"`, func(g *fakeGraph) uuid.UUID {
			return g.add("Typo", nil, map[string]interface{}{"expression": "Quantty * 2"})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newGraph()
			id := tc.build(g)
			res, _, err := (&ExecutionEngine{graphService: g}).ExecuteCalculation(context.Background(), id, map[string]interface{}{"Quantity": 1.0})
			if err == nil {
				t.Fatalf("got %v, want an error", res)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not name %s", err, tc.want)
			}
		})
	}
}

func TestStripAssignment(t *testing.T) {
	for in, want := range map[string]string{
		"NAV = sum(PositionValue)": "sum(PositionValue)",
		"a == b":                   "a == b",
		"Quantity * Price":         "Quantity * Price",
		"x >= 3":                   "x >= 3",
		"2 = 3":                    "2 = 3",
		"":                         "",
	} {
		if got := stripAssignment(in); got != want {
			t.Errorf("stripAssignment(%q) = %q, want %q", in, got, want)
		}
	}
}
