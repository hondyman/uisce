package eval

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type SyntheticTestCase struct {
	TestCaseID     uuid.UUID              `json:"test_case_id"`
	PromptText     string                 `json:"prompt_text"`
	ExpectedAST    string                 `json:"expected_ast"`
	Parameters     map[string]interface{} `json:"parameters"`
	ExpectedResult float64                `json:"expected_result"`
}

type TestMatrixGenerator struct {
	db *sql.DB
}

func NewTestMatrixGenerator(db *sql.DB) *TestMatrixGenerator {
	return &TestMatrixGenerator{db: db}
}

// Generate500TestSuite expands combinations of dimensions, measures, time grains, and filters
func (g *TestMatrixGenerator) Generate500TestSuite(
	ctx context.Context,
	boName string,
	dimensions []string,
	measures []string,
	horizons []string,
	regions []string,
) []SyntheticTestCase {
	cases := make([]SyntheticTestCase, 0, 500)

	templatePrompts := []string{
		"What was the %s by %s for %s accounts in %s?",
		"Calculate %s grouped by %s over %s in %s",
		"Show %s breakdown across %s for %s during %s",
		"Analyze %s and %s by %s over %s in %s",
	}

	for _, m := range measures {
		for _, d := range dimensions {
			for _, h := range horizons {
				for _, r := range regions {
					if len(cases) >= 500 {
						break
					}

					prompt := fmt.Sprintf(templatePrompts[len(cases)%len(templatePrompts)], m, d, r, h)
					ast := fmt.Sprintf(`{"bo":"%s","dimensions":["%s"],"measures":[{"term":"%s","agg":"SUM"}],"filters":[{"term":"region","op":"EQ","val":"%s"}]}`,
						boName, d, m, r)

					cases = append(cases, SyntheticTestCase{
						TestCaseID:  uuid.New(),
						PromptText:  prompt,
						ExpectedAST: ast,
						Parameters: map[string]interface{}{
							"dimension": d,
							"measure":   m,
							"horizon":   h,
							"region":    r,
						},
						ExpectedResult: 1425000.50,
					})
				}
			}
		}
	}

	return cases
}
