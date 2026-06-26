package policy

import (
	"context"
	"testing"
	"time"
)

func TestCELEvalBool_Eligibility(t *testing.T) {
	e, err := NewCELEvaluator()
	if err != nil {
		t.Fatal(err)
	}

	vars := map[string]interface{}{
		"holdings": []map[string]interface{}{
			{"symbol": "MSFT", "weight": 0.5},
		},
		"symbol": "MSFT",
	}

	// Test simple eligibility
	ok, err := e.EvalBool(context.Background(), `symbol == "MSFT"`, vars)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected true for symbol match")
	}
}

func TestCELEvalNumber_Recency(t *testing.T) {
	e, err := NewCELEvaluator()
	if err != nil {
		t.Fatal(err)
	}

	threeDaysAgo := time.Now().AddDate(0, 0, -3)
	vars := map[string]interface{}{
		"dividend_date": threeDaysAgo,
	}

	// Test recency function
	days, err := e.EvalNumber(context.Background(), `recency(dividend_date)`, vars)
	if err != nil {
		t.Fatal(err)
	}

	if days < 2 || days > 4 {
		t.Fatalf("expected ~3 days, got %.1f", days)
	}
}

func TestCELEvalNumber_PortfolioWeight(t *testing.T) {
	e, err := NewCELEvaluator()
	if err != nil {
		t.Fatal(err)
	}

	vars := map[string]interface{}{
		"symbol": "MSFT",
		"holdings": []map[string]interface{}{
			{"symbol": "MSFT", "weight": 0.5},
			{"symbol": "AAPL", "weight": 0.3},
			{"symbol": "TSLA", "weight": 0.2},
		},
	}

	// Test portfolio_weight function
	weight, err := e.EvalNumber(context.Background(), `portfolio_weight(symbol, holdings)`, vars)
	if err != nil {
		t.Fatal(err)
	}

	expected := 0.5
	if weight < expected-0.01 || weight > expected+0.01 {
		t.Fatalf("expected %.2f, got %.2f", expected, weight)
	}
}

func TestCELEvalNumber_RankingExpression(t *testing.T) {
	e, err := NewCELEvaluator()
	if err != nil {
		t.Fatal(err)
	}

	vars := map[string]interface{}{
		"symbol":        "MSFT",
		"dividend_date": time.Now().AddDate(0, 0, -3),
		"holdings": []map[string]interface{}{
			{"symbol": "MSFT", "weight": 0.5},
			{"symbol": "AAPL", "weight": 0.5},
		},
	}

	// Test complex ranking expression
	score, err := e.EvalNumber(context.Background(), `portfolio_weight(symbol, holdings) * 10.0 + double(recency(dividend_date))`, vars)
	if err != nil {
		t.Fatal(err)
	}

	// 0.5 * 10 + 3 = 8
	if score < 7.5 || score > 8.5 {
		t.Fatalf("expected ~8, got %.2f", score)
	}
}
