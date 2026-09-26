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
	p, _ := newRuleCheckProc(Node{ID: "rc", Type: NodeRuleCheck, Config: cfg(RuleCheckConfig{RuleIDs: []string{positive.ID.String()}})}, c)
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
