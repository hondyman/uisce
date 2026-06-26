package pagetests

import (
	"context"
	"fmt"

	"github.com/hondyman/semlayer/backend/internal/pagestudio"
)

type TestType string

const (
	TestTypeLoad        TestType = "load"
	TestTypeInteraction TestType = "interaction"
	TestTypeSLO         TestType = "slo"
	TestTypePII         TestType = "pii"
)

type TestCase struct {
	ID          string   `json:"id"`
	Type        TestType `json:"type"`
	Description string   `json:"description"`
	Assertion   string   `json:"assertion"`
	Target      string   `json:"target,omitempty"`
}

type TestSuite struct {
	PageID string     `json:"page_id"`
	Tests  []TestCase `json:"tests"`
}

type TestGenerator struct{}

func NewTestGenerator() *TestGenerator {
	return &TestGenerator{}
}

func (g *TestGenerator) Generate(ctx context.Context, page *pagestudio.CorePage) (*TestSuite, error) {
	suite := &TestSuite{
		PageID: page.ID.String(),
		Tests:  make([]TestCase, 0),
	}

	// Generate Load Tests
	suite.Tests = append(suite.Tests, TestCase{
		ID:          fmt.Sprintf("load_%s", page.ID.String()),
		Type:        TestTypeLoad,
		Description: fmt.Sprintf("Page '%s' must render in < 1s p95 under 50 concurrent users", page.Name),
		Assertion:   "p95_render_time < 1000ms",
	})

	// Generate Interaction Tests (mock)
	suite.Tests = append(suite.Tests, TestCase{
		ID:          fmt.Sprintf("interaction_%s_row_click", page.ID.String()),
		Type:        TestTypeInteraction,
		Description: "Row click must navigate with correct params",
		Assertion:   "onRowClick triggers navigation",
		Target:      "table_component",
	})

	// Generate SLO Tests
	suite.Tests = append(suite.Tests, TestCase{
		ID:          fmt.Sprintf("slo_%s_modal", page.ID.String()),
		Type:        TestTypeSLO,
		Description: "Modal must load in < 300ms p95",
		Assertion:   "modal_load_time < 300ms",
	})

	// Generate PII Tests (mock)
	suite.Tests = append(suite.Tests, TestCase{
		ID:          fmt.Sprintf("pii_%s_ssn", page.ID.String()),
		Type:        TestTypePII,
		Description: "SSN must not appear in unauthorized components",
		Assertion:   "no_pii_in_component(ssn, table)",
	})

	return suite, nil
}
