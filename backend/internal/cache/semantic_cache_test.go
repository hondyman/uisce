package cache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type dummyAST struct {
	BO      string   `json:"bo"`
	Metrics []string `json:"metrics"`
}

func TestSemanticCacheASTHashing(t *testing.T) {
	ast1 := dummyAST{BO: "Customer", Metrics: []string{"total_sales"}}
	ast2 := dummyAST{BO: "Customer", Metrics: []string{"total_sales"}}
	ast3 := dummyAST{BO: "Customer", Metrics: []string{"total_orders"}}

	hash1, err1 := ComputeASTHash(ast1)
	hash2, err2 := ComputeASTHash(ast2)
	hash3, err3 := ComputeASTHash(ast3)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)

	assert.Equal(t, hash1, hash2, "Identical ASTs must produce identical hashes")
	assert.NotEqual(t, hash1, hash3, "Different ASTs must produce different hashes")
}

func TestSemanticCacheNilSafe(t *testing.T) {
	var cache *SemanticCache
	var out map[string]interface{}
	found, err := cache.Get(context.Background(), "dummy_hash", &out)
	assert.NoError(t, err)
	assert.False(t, found)

	err = cache.Set(context.Background(), "dummy_hash", "data")
	assert.NoError(t, err)
}
