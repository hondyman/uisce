package eval

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestSyntheticEvalPipeline(t *testing.T) {
	generator := NewTestMatrixGenerator(nil)
	evaluator := NewRegressionEvaluator(nil)

	testCases := generator.Generate500TestSuite(
		context.Background(),
		"wealth.portfolio",
		[]string{"country", "currency", "sector"},
		[]string{"market_value", "xirr", "yield"},
		[]string{"1Y", "3Y", "YTD"},
		[]string{"USCAN", "EMEA", "APAC"},
	)

	if len(testCases) != 81 {
		t.Errorf("expected 81 test cases for 3x3x3x3 permutations, got %d", len(testCases))
	}

	result, err := evaluator.RunEvaluationSuite(
		context.Background(),
		uuid.New(),
		uuid.New(),
		"e4b10294819a4c22bade4f4d439ac84c",
		42,
		testCases,
		0.000001,
	)

	if err != nil {
		t.Fatalf("unexpected error running eval suite: %v", err)
	}

	if result.Status != "PASSED" {
		t.Errorf("expected evaluation to PASS, got %s", result.Status)
	}

	if len(result.MerklePassport) == 0 {
		t.Errorf("expected Merkle audit passport to be generated")
	}
}
