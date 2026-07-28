package pgwire

import (
	"context"
	"testing"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockSQLGenerator struct{}

func (m *mockSQLGenerator) GenerateFromSemanticRequest(req boresolver.SemanticSQLGenerationRequest) (string, []interface{}, error) {
	return "SELECT id, total_amount FROM orders WHERE tenant_id = $1 AND region = $2 LIMIT 100", []interface{}{req.TenantID, "US"}, nil
}

type mockSecurityInterceptor struct{}

func (m *mockSecurityInterceptor) InterceptSemanticRequest(ctx context.Context, tenantID, userID string, req *boresolver.SemanticSQLGenerationRequest) error {
	// Auto-inject tenant context into request
	req.TenantID = tenantID
	return nil
}

func TestPGWireQueryTranslation(t *testing.T) {
	sql := `SELECT id, total_amount FROM "Customer" WHERE region = 'US' LIMIT 50`
	req, err := TranslateSimpleQuery(sql, "tenant_123")

	assert.NoError(t, err)
	assert.Equal(t, "Customer", req.Datasource)
	assert.Equal(t, "tenant_123", req.TenantID)
	assert.Len(t, req.Select, 2)
	assert.Equal(t, "id", req.Select[0].Term)
	assert.Equal(t, "total_amount", req.Select[1].Term)
	assert.Len(t, req.Filters, 1)
	assert.Equal(t, "region", req.Filters[0].Term)
	assert.Equal(t, "US", req.Filters[0].Value)
}

func TestPGWireASTConvergence(t *testing.T) {
	srv := NewPGWireServer(Config{
		Addr:                ":5433",
		DefaultTenantID:     "gold_copy",
		SQLGenerator:        &mockSQLGenerator{},
		SecurityInterceptor: &mockSecurityInterceptor{},
		Logger:              zap.NewNop(),
	})

	handler := &SessionHandler{
		server:   srv,
		tenantID: "gold_copy",
		userID:   "user_test",
	}

	req, err := TranslateSimpleQuery(`SELECT id, total_amount FROM "orders"`, "gold_copy")
	assert.NoError(t, err)

	compiledSQL, args, err := handler.ExecuteSemanticAST(context.Background(), *req)
	assert.NoError(t, err)
	assert.Contains(t, compiledSQL, "SELECT id, total_amount FROM orders")
	assert.Equal(t, []interface{}{"gold_copy", "US"}, args)
}
