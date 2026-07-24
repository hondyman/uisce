package policy

import (
	"context"
	"testing"
	"time"
)

func BenchmarkCELEvalNumber_RankingExpression(b *testing.B) {
	e, err := NewCELEvaluator()
	if err != nil {
		b.Fatal(err)
	}

	expr := `portfolio_weight(symbol, holdings) * 10.0 + double(recency(dividend_date))`
	vars := map[string]interface{}{
		"symbol":        "MSFT",
		"dividend_date": time.Now().AddDate(0, 0, -3),
		"holdings": []map[string]interface{}{
			{"symbol": "MSFT", "weight": 0.5},
			{"symbol": "AAPL", "weight": 0.3},
			{"symbol": "TSLA", "weight": 0.2},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := e.EvalNumber(context.Background(), expr, vars)
		if err != nil {
			b.Fatalf("eval error: %v", err)
		}
	}
}

func BenchmarkCELEvalBool_Eligibility(b *testing.B) {
	e, err := NewCELEvaluator()
	if err != nil {
		b.Fatal(err)
	}

	expr := `symbol == "MSFT" && portfolio_weight(symbol, holdings) > 0.1`
	vars := map[string]interface{}{
		"symbol": "MSFT",
		"holdings": []map[string]interface{}{
			{"symbol": "MSFT", "weight": 0.5},
			{"symbol": "AAPL", "weight": 0.5},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := e.EvalBool(context.Background(), expr, vars)
		if err != nil {
			b.Fatalf("eval error: %v", err)
		}
	}
}
