package datapipeline

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/models"
)

type fakeRules struct {
	byID  map[uuid.UUID]*models.ValidationRuleDescriptor
	calls int
}

func (f *fakeRules) GetByIDForTenant(_ context.Context, tenant string, id uuid.UUID) (*models.ValidationRuleDescriptor, error) {
	f.calls++
	d, ok := f.byID[id]
	if !ok || (d.TenantID != tenant && d.TenantID != "gold") {
		return nil, errors.New("validation rule not found")
	}
	return d, nil
}

func rule(tenant, name, sev, ast string) *models.ValidationRuleDescriptor {
	return &models.ValidationRuleDescriptor{ID: uuid.New(), TenantID: tenant, Name: name, Severity: sev, IsActive: true, RuleAST: json.RawMessage(ast)}
}

func TestCatalogRuleChecker(t *testing.T) {
	positive := rule("t1", "AUM positive", "BLOCK", `{"type":"condition","field":"aum","operator":">","value":0}`)
	named := rule("gold", "Has name", "WARN", `{"type":"condition","field":"name","operator":"is_not_empty"}`)
	broken := rule("t1", "Bad regex", "WARN", `{"type":"condition","field":"name","operator":"matches_regex","value":"(["}`)
	other := rule("t2", "Other tenant", "BLOCK", `{"type":"condition","field":"aum","operator":">","value":0}`)
	src := &fakeRules{byID: map[uuid.UUID]*models.ValidationRuleDescriptor{}}
	for _, r := range []*models.ValidationRuleDescriptor{positive, named, broken, other} {
		src.byID[r.ID] = r
	}
	c := &CatalogRuleChecker{Rules: src}
	ctx := context.Background()
	ids := []string{positive.ID.String(), named.ID.String()}

	fails, err := c.Check(ctx, "t1", ids, map[string]any{"aum": 10.0, "name": "Fund A"})
	if err != nil || len(fails) != 0 {
		t.Fatalf("clean row: %v %v", fails, err)
	}
	fails, _ = c.Check(ctx, "t1", ids, map[string]any{"aum": -1.0, "name": ""})
	if len(fails) != 2 || fails[0].Severity != "BLOCK" || fails[1].Severity != "WARN" {
		t.Fatalf("failing row: %+v", fails)
	}
	if src.calls != 2 {
		t.Errorf("rules should load once per run, loaded %d times", src.calls)
	}

	// Each run starts with a fresh cache: an edited rule applies next run.
	positive.RuleAST = json.RawMessage(`{"type":"condition","field":"aum","operator":">","value":100}`)
	p, _ := newRuleCheckProc(Node{ID: "rc", Type: NodeRuleCheck, Config: cfg(RuleCheckConfig{RuleIDs: []string{positive.ID.String()}})}, c, nil)
	_ = p.Open(ctx, &RunContext{TenantID: "t1"})
	res, _ := p.Process(ctx, []Row{{Num: 1, Data: map[string]any{"aum": 50.0}}})
	if len(res.Rejected) != 1 {
		t.Error("a new run must see the edited rule")
	}
	if len(c.cache) != 1 {
		t.Error("the shared checker's cache must not grow from runs")
	}

	// An unevaluable rule blocks, whatever its declared severity.
	fails, _ = c.Check(ctx, "t1", []string{broken.ID.String()}, map[string]any{"name": "x"})
	if len(fails) != 1 || fails[0].Severity != "BLOCK" || !strings.Contains(fails[0].Message, "could not be evaluated") {
		t.Fatalf("broken rule: %+v", fails)
	}

	// Another tenant's rule is invisible.
	if _, err := c.Check(ctx, "t1", []string{other.ID.String()}, map[string]any{}); err == nil {
		t.Error("another tenant's rule must not load")
	}
	if _, err := c.Check(ctx, "t1", []string{"not-a-uuid"}, map[string]any{}); err == nil {
		t.Error("bad id must error")
	}
	inactive := rule("t1", "Off", "BLOCK", `{"type":"condition","field":"a","operator":"is_null"}`)
	inactive.IsActive = false
	src.byID[inactive.ID] = inactive
	if _, err := c.Check(ctx, "t1", []string{inactive.ID.String()}, map[string]any{}); err == nil {
		t.Error("inactive rule must error")
	}
}

// A row that is not in the rule's terms - here staging columns (isin_cd)
// where the rule reads the business object field Isin - is rejected with the
// missing field named, whatever the rule's severity, never evaluated: a
// missing field would otherwise read as an ordinary pass or fail.
func TestCatalogRuleCheckerMissingField(t *testing.T) {
	warnOnly := rule("t1", "ISIN present", "WARN", `{"type":"group","operator":"OR","conditions":[
		{"type":"condition","field":"Isin","operator":"is_null"},
		{"type":"condition","field":"Isin","operator":"matches_regex","value":"^[A-Z]{2}[A-Z0-9]{9}[0-9]$"}]}`)
	nested := rule("t1", "Issuer active", "BLOCK", `{"type":"condition","field":"issuer.status","operator":"=","value":"A"}`)
	src := &fakeRules{byID: map[uuid.UUID]*models.ValidationRuleDescriptor{warnOnly.ID: warnOnly, nested.ID: nested}}
	c := &CatalogRuleChecker{Rules: src}
	ctx := context.Background()

	fails, err := c.Check(ctx, "t1", []string{warnOnly.ID.String()}, map[string]any{"isin_cd": "US0378331005"})
	if err != nil || len(fails) != 1 {
		t.Fatalf("staging-shaped row: %+v %v", fails, err)
	}
	if fails[0].Severity != models.ValidationRuleSeverityBlock || !strings.Contains(fails[0].Message, "Isin") {
		t.Errorf("want a BLOCK naming Isin, got %+v", fails[0])
	}

	// The same rule on a row in its terms evaluates normally.
	if fails, _ := c.Check(ctx, "t1", []string{warnOnly.ID.String()}, map[string]any{"Isin": "US0378331005"}); len(fails) != 0 {
		t.Errorf("row in the rule's terms: %+v", fails)
	}

	// Nested paths resolve through related data, not the row: not flagged here.
	fails, _ = c.Check(ctx, "t1", []string{nested.ID.String()}, map[string]any{})
	for _, f := range fails {
		if strings.Contains(f.Message, "does not have") {
			t.Errorf("nested path flagged as missing: %+v", f)
		}
	}
}

type fakeBindings map[string]map[string]string // bo|table -> field -> column

func (f fakeBindings) StagingFields(_ context.Context, _, bo, table string) (map[string]string, error) {
	return f[bo+"|"+table], nil
}

// In front of a staging load the rows are read through the table's binding:
// the rule reads the object's field (aum), the row carries the staging column
// (aum_amt), and what flows on to the load is the row as it was.
func TestRuleCheckReadsStagingRowsThroughTheBinding(t *testing.T) {
	positive := rule("t1", "AUM positive", "BLOCK", `{"type":"condition","field":"aum","operator":">","value":0}`)
	checker := &CatalogRuleChecker{Rules: &fakeRules{byID: map[uuid.UUID]*models.ValidationRuleDescriptor{positive.ID: positive}}}
	spec := &Spec{Nodes: []Node{
		{ID: "rc", Type: NodeRuleCheck, Config: cfg(RuleCheckConfig{BOKey: "fund", RuleIDs: []string{positive.ID.String()}})},
		{ID: "out", Type: NodeStagingSink, Config: cfg(StagingSinkConfig{Table: "staging.ff_fund", SourceCd: "FS", Domain: "PRODUCT"})},
	}, Edges: []Edge{{From: "rc", To: "out"}}}
	ctx := context.Background()

	bindings := fakeBindings{"fund|staging.ff_fund": {"aum": "aum_amt"}}
	p, err := newRuleCheckProc(spec.Nodes[0], checker, bindings)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Open(ctx, &RunContext{TenantID: "t1", Spec: spec}); err != nil {
		t.Fatal(err)
	}
	res, err := p.Process(ctx, []Row{
		{Num: 1, Data: map[string]any{"aum_amt": 5.0}},
		{Num: 2, Data: map[string]any{"aum_amt": -1.0}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Out) != 1 || len(res.Rejected) != 1 || res.Rejected[0].Row.Num != 2 {
		t.Fatalf("out=%d rejected=%+v", len(res.Out), res.Rejected)
	}
	if _, aliased := res.Out[0].Data["aum"]; aliased {
		t.Error("the row passed on to the load must keep its own columns only")
	}

	// No approved binding: the run does not start.
	p, _ = newRuleCheckProc(spec.Nodes[0], checker, fakeBindings{})
	if err := p.Open(ctx, &RunContext{TenantID: "t1", Spec: spec}); err == nil || !strings.Contains(err.Error(), "no approved binding") {
		t.Errorf("want a no-binding error, got %v", err)
	}
	// Nor without the bindings service.
	p, _ = newRuleCheckProc(spec.Nodes[0], checker, nil)
	if err := p.Open(ctx, &RunContext{TenantID: "t1", Spec: spec}); err == nil {
		t.Error("a staging rule check without bindings must not run")
	}
}
