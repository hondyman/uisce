package testplans

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type TestType string

const (
	TestTypeUnit        TestType = "unit"
	TestTypeIntegration TestType = "integration"
	TestTypeRegression  TestType = "regression"
	TestTypePerformance TestType = "performance"
	TestTypeCompliance  TestType = "compliance"
)

type TestCase struct {
	ID          uuid.UUID `json:"id"`
	Type        TestType  `json:"type"`
	Description string    `json:"description"`
	Priority    string    `json:"priority"` // critical, high, medium, low
}

type TestPlan struct {
	ChangeSetID uuid.UUID  `json:"changeset_id"`
	Tests       []TestCase `json:"tests"`
	TotalTests  int        `json:"total_tests"`
}

type TestPlanGenerator struct{}

func NewTestPlanGenerator() *TestPlanGenerator {
	return &TestPlanGenerator{}
}

func (g *TestPlanGenerator) Generate(ctx context.Context, changesetID uuid.UUID) (*TestPlan, error) {
	// Mock: Generate test plan
	// Real: Analyze changeset diff, lineage, SLOs, data policies

	tests := []TestCase{
		{
			ID:          uuid.New(),
			Type:        TestTypeUnit,
			Description: "Validate Positions API returns market_value_usd field",
			Priority:    "critical",
		},
		{
			ID:          uuid.New(),
			Type:        TestTypeUnit,
			Description: "Validate market_value_usd calculation logic",
			Priority:    "critical",
		},
		{
			ID:          uuid.New(),
			Type:        TestTypeIntegration,
			Description: "Validate Positions Dashboard displays market_value_usd KPI",
			Priority:    "high",
		},
		{
			ID:          uuid.New(),
			Type:        TestTypePerformance,
			Description: "Validate positions_dashboard renders in < 300ms p95",
			Priority:    "high",
		},
		{
			ID:          uuid.New(),
			Type:        TestTypeCompliance,
			Description: "Validate masking rules for tenant-123",
			Priority:    "critical",
		},
		{
			ID:          uuid.New(),
			Type:        TestTypeCompliance,
			Description: "Validate no residency violations for EU tenants",
			Priority:    "critical",
		},
		{
			ID:          uuid.New(),
			Type:        TestTypeRegression,
			Description: "Validate semantic drift tests for BO Positions",
			Priority:    "medium",
		},
		{
			ID:          uuid.New(),
			Type:        TestTypeRegression,
			Description: "Validate no breaking changes in positions_api",
			Priority:    "critical",
		},
	}

	plan := &TestPlan{
		ChangeSetID: changesetID,
		Tests:       tests,
		TotalTests:  len(tests),
	}

	return plan, nil
}

func (g *TestPlanGenerator) ExportForAuditors(ctx context.Context, plan *TestPlan) string {
	// Mock: Export test plan for auditors
	// Real: Generate formatted document
	output := "Test Plan for ChangeSet " + plan.ChangeSetID.String() + "\n\n"
	output += "Total Tests: " + fmt.Sprintf("%d", plan.TotalTests) + "\n\n"

	for _, test := range plan.Tests {
		output += fmt.Sprintf("- [%s] %s (%s)\n", test.Type, test.Description, test.Priority)
	}

	return output
}
