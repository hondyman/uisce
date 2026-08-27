package ai

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestCompileConstrainedSQL_CFGEnforcement(t *testing.T) {
	// Unit test for CFG error handling logic
	generator := &CFGSQLGenerator{db: nil}

	// 1. Assert nil tenantID fails Rule 7
	_, err := generator.CompileConstrainedSQL(context.Background(), uuid.Nil, uuid.New(), "account", []string{"region"}, []string{"aum"}, nil)
	if err == nil {
		t.Fatalf("expected error for nil tenantID, got nil")
	}

	// 2. Assert CFGValidationError message structure
	valErr := &CFGValidationError{
		ViolatedRule:  "DIMENSION_WHITELIST",
		TokenOffender: "unauthorized_column",
	}
	expectedMsg := "CFG Constraint Violation [DIMENSION_WHITELIST]: Unauthorized token 'unauthorized_column'"
	if valErr.Error() != expectedMsg {
		t.Errorf("expected %q, got %q", expectedMsg, valErr.Error())
	}
}
