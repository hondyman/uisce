package advisor

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestASTNormalizer(t *testing.T) {
	normalizer := NewASTNormalizer(nil)

	expr1, hash1 := normalizer.NormalizeAlgebraicExpression("price * (1 - discount)")
	_, hash2 := normalizer.NormalizeAlgebraicExpression("  price * (1 - discount)  ")

	if hash1 != hash2 {
		t.Errorf("expected hashes to match for equivalent whitespace expressions, got %s vs %s", hash1, hash2)
	}

	if expr1 != "price * (1.0 - discount)" {
		t.Errorf("unexpected normalized expression: %s", expr1)
	}

	ddl, err := normalizer.RecommendMaterializedView(
		context.Background(),
		uuid.New(),
		"a1b2c3d4e5f67890",
		"SELECT customer_id, SUM(order_total) FROM orders GROUP BY customer_id",
		1250,
	)

	if err != nil {
		t.Fatalf("unexpected error generating MV DDL: %v", err)
	}

	if len(ddl) == 0 {
		t.Errorf("expected non-empty DDL")
	}
}
