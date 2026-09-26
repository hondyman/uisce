package datapipeline

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type fakeCatalog struct{}

func (fakeCatalog) BusinessObjects(context.Context) ([]BOInfo, error) {
	return []BOInfo{{Key: "fund", Label: "Fund"}, {Key: "account", Label: "Account"}}, nil
}
func (fakeCatalog) BOFields(_ context.Context, k string) ([]TargetField, error) {
	if k == "fund" {
		return []TargetField{{Name: "fsym_id", Required: true}, {Name: "fund_name"}, {Name: "aum"}}, nil
	}
	return []TargetField{{Name: "account_no"}}, nil
}
func (fakeCatalog) Rules(_ context.Context, k string) ([]RuleInfo, error) {
	if k == "fund" {
		return []RuleInfo{{ID: "11111111-1111-1111-1111-111111111111", Name: "AUM positive", Severity: "BLOCK", Fields: []string{"aum"}}}, nil
	}
	return nil, nil
}
func (fakeCatalog) Files(context.Context) ([]string, error) { return []string{"uploads/fs.txt"}, nil }
func (fakeCatalog) StagingTables(context.Context) ([]StagingTable, error) {
	return []StagingTable{{Table: "staging.ff_fund", Columns: []TargetField{{Name: "fsym_id"}, {Name: "aum"}}}}, nil
}

const goodSpec = `{"version":1,"nodes":[
 {"id":"file","type":"file_source","label":"FactSet file","config":{"uri":"uploads/fs.txt","format":"csv","delimiter":"|","columns":[{"name":"FSYM_ID","type":"string"},{"name":"AUM","type":"decimal"}]}},
 {"id":"map","type":"map","config":{"fields":[{"from":"FSYM_ID","to":"fsym_id"},{"from":"AUM","to":"aum"}]}},
 {"id":"rules","type":"rule_check","config":{"bo_key":"fund","rule_ids":["11111111-1111-1111-1111-111111111111"]}},
 {"id":"out","type":"bo_sink","config":{"bo_key":"fund","mode":"upsert","key_fields":["fsym_id"]}}],
 "edges":[{"from":"file","to":"map"},{"from":"map","to":"rules"},{"from":"rules","to":"out"}]}`

func specOf(t *testing.T, s string) *Spec {
	var sp Spec
	if err := json.Unmarshal([]byte(s), &sp); err != nil {
		t.Fatal(err)
	}
	return &sp
}

func TestCheckGroundsEveryReference(t *testing.T) {
	if issues := Check(context.Background(), fakeCatalog{}, specOf(t, goodSpec)); len(issues) != 0 {
		t.Fatalf("grounded spec flagged: %+v", issues)
	}
	bad := strings.NewReplacer(
		`"uploads/fs.txt"`, `"uploads/missing.csv"`,
		`"to":"aum"`, `"to":"assets"`,
		`"rule_ids":["11111111-1111-1111-1111-111111111111"]`, `"rule_ids":["22222222-2222-2222-2222-222222222222"]`,
		`"key_fields":["fsym_id"]`, `"key_fields":["isin"]`,
	).Replace(goodSpec)
	got := map[string]bool{}
	for _, i := range Check(context.Background(), fakeCatalog{}, specOf(t, bad)) {
		got[i.NodeID+": "+i.Message] = true
	}
	for _, want := range []string{
		`file: file "uploads/missing.csv" has not been uploaded`,
		`map: fund has no field "assets" (mapped from AUM)`,
		`rules: rule "22222222-2222-2222-2222-222222222222" is not an active rule of fund`,
		`out: fund has no field "isin" to match on`,
	} {
		if !got[want] {
			t.Errorf("missing issue %q in %v", want, got)
		}
	}
	unknownBO := strings.Replace(goodSpec, `"type":"bo_sink","config":{"bo_key":"fund"`, `"type":"bo_sink","config":{"bo_key":"funds"`, 1)
	found := false
	for _, i := range Check(context.Background(), fakeCatalog{}, specOf(t, unknownBO)) {
		found = found || (i.NodeID == "out" && i.Message == `there is no business object "funds"`)
	}
	if !found {
		t.Error("an invented business object must be caught")
	}
}

// Rules read business object fields. Rows reaching a rule check in another
// shape - here mapped to staging columns - would be rejected wholesale at run
// time, so the editor says so up front; and rules of one object in front of a
// load into another are flagged.
func TestCheckRuleFieldsReachTheRuleCheck(t *testing.T) {
	staging := `{"version":1,"nodes":[
	 {"id":"file","type":"file_source","config":{"uri":"uploads/fs.txt","format":"csv","columns":[{"name":"FSYM_ID","type":"string"},{"name":"AUM","type":"decimal"}]}},
	 {"id":"map","type":"map","config":{"fields":[{"from":"FSYM_ID","to":"fsym_id"},{"from":"AUM","to":"aum_amt"}]}},
	 {"id":"rules","type":"rule_check","config":{"bo_key":"fund","rule_ids":["11111111-1111-1111-1111-111111111111"]}},
	 {"id":"out","type":"staging_sink","config":{"table":"staging.ff_fund","source_cd":"FACTSET","domain":"PRODUCT"}}],
	 "edges":[{"from":"file","to":"map"},{"from":"map","to":"rules"},{"from":"rules","to":"out"}]}`
	want := `rules: rule "AUM positive" reads fund field(s) aum, which the rows here don't have - map them to fund's fields before this step`
	got := map[string]bool{}
	for _, i := range Check(context.Background(), fakeCatalog{}, specOf(t, staging)) {
		got[i.NodeID+": "+i.Message] = true
	}
	if !got[want] {
		t.Errorf("missing %q in %v", want, got)
	}

	// Straight from the file, rows carry the file's columns (AUM, not aum).
	direct := `{"version":1,"nodes":[
	 {"id":"file","type":"file_source","config":{"uri":"uploads/fs.txt","format":"csv","columns":[{"name":"AUM","type":"decimal"}]}},
	 {"id":"rules","type":"rule_check","config":{"bo_key":"fund","rule_ids":["11111111-1111-1111-1111-111111111111"]}}],
	 "edges":[{"from":"file","to":"rules"}]}`
	found := false
	for _, i := range Check(context.Background(), fakeCatalog{}, specOf(t, direct)) {
		found = found || (i.NodeID == "rules" && strings.Contains(i.Message, "reads fund field(s) aum"))
	}
	if !found {
		t.Error("file columns reaching a rule check must be checked against the rule's fields")
	}

	// fund rules in front of an account load.
	mismatch := strings.Replace(goodSpec, `"type":"bo_sink","config":{"bo_key":"fund","mode":"upsert","key_fields":["fsym_id"]}`,
		`"type":"bo_sink","config":{"bo_key":"account"}`, 1)
	found = false
	for _, i := range Check(context.Background(), fakeCatalog{}, specOf(t, mismatch)) {
		found = found || (i.NodeID == "rules" && i.Message == "these are fund rules, but the rows are loaded into account")
	}
	if !found {
		t.Error("rules of one business object in front of a load into another must be flagged")
	}
}

// scriptedLLM returns answers in order and records the prompts.
type scriptedLLM struct {
	answers []string
	prompts []string
}

func (s *scriptedLLM) call(_ context.Context, p string) (string, error) {
	s.prompts = append(s.prompts, p)
	a := s.answers[0]
	if len(s.answers) > 1 {
		s.answers = s.answers[1:]
	}
	return a, nil
}

func TestAssistantCorrectsItselfFromGroundingProblems(t *testing.T) {
	invented := strings.Replace(goodSpec, `"type":"bo_sink","config":{"bo_key":"fund"`, `"type":"bo_sink","config":{"bo_key":"funds"`, 1)
	llm := &scriptedLLM{answers: []string{
		"```json\n" + `{"reply":"Built it","changes":["added steps"],"spec":` + invented + "}\n```",
		`{"reply":"Built it","changes":["added steps"],"spec":` + goodSpec + `}`,
	}}
	a := &Assistant{LLM: llm.call}
	out, err := a.Respond(context.Background(), fakeCatalog{}, AssistRequest{Message: "Load the FactSet file into Fund, applying the fund rules"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Attempts != 2 || len(out.Issues) != 0 || out.Spec == nil {
		t.Fatalf("got %+v", out)
	}
	if !strings.Contains(llm.prompts[1], `there is no business object "funds"`) {
		t.Error("the second attempt must be told what was wrong")
	}
	// Context: the relevant BO's fields and rules, not every BO's.
	if !strings.Contains(llm.prompts[0], `"fund":[{"name":"fsym_id"`) || strings.Contains(llm.prompts[0], `"account_no"`) {
		t.Errorf("context should carry fund's fields only")
	}
	if !strings.Contains(llm.prompts[0], "AUM positive") || !strings.Contains(llm.prompts[0], "uploads/fs.txt") {
		t.Error("context must include rules and uploaded files")
	}
	for _, n := range out.Spec.Nodes {
		if n.Position == nil {
			t.Errorf("node %s needs a canvas position", n.ID)
		}
	}
}

func TestAssistantReturnsBestEffortWithRemainingIssues(t *testing.T) {
	invented := strings.Replace(goodSpec, `"type":"bo_sink","config":{"bo_key":"fund"`, `"type":"bo_sink","config":{"bo_key":"funds"`, 1)
	llm := &scriptedLLM{answers: []string{`{"reply":"x","spec":` + invented + `}`}}
	out, err := (&Assistant{LLM: llm.call, MaxAttempts: 2}).Respond(context.Background(), fakeCatalog{}, AssistRequest{Message: "go"})
	if err != nil || len(out.Issues) == 0 || len(llm.prompts) != 2 {
		t.Fatalf("out=%+v err=%v prompts=%d", out, err, len(llm.prompts))
	}
}

func TestAssistantAnswersQuestionsWithoutChanges(t *testing.T) {
	llm := &scriptedLLM{answers: []string{"not json at all", `{"reply":"Row 3 failed because AUM was blank.","spec":null}`}}
	out, err := (&Assistant{LLM: llm.call}).Respond(context.Background(), fakeCatalog{}, AssistRequest{
		Message: "why did row 3 fail?", Spec: *specOf(t, goodSpec), PreviewRejects: []string{"row 3: AUM is required"},
	})
	if err != nil || out.Spec != nil || !strings.Contains(out.Reply, "AUM") {
		t.Fatalf("out=%+v err=%v", out, err)
	}
	if !strings.Contains(llm.prompts[0], "row 3: AUM is required") || !strings.Contains(llm.prompts[1], "not valid JSON") {
		t.Error("preview rejects and the JSON retry hint must reach the model")
	}
}

func TestKeepPositionsOnEdit(t *testing.T) {
	prev := specOf(t, goodSpec)
	prev.Nodes[0].Position = &Position{X: 10, Y: 20}
	next := specOf(t, goodSpec)
	keepPositions(next, prev)
	if next.Nodes[0].Position.X != 10 || next.Nodes[1].Position == nil {
		t.Fatalf("positions: %+v %+v", next.Nodes[0].Position, next.Nodes[1].Position)
	}
}
