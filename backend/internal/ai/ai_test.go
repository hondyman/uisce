package ai

import (
	"context"
	"testing"

	"github.com/hondyman/semlayer/backend/internal/rules"
	"github.com/hondyman/semlayer/backend/internal/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mocks

type MockMetadataRepository struct {
	mock.Mock
}

func (m *MockMetadataRepository) GetEntityDefinition(ctx context.Context, tenantID, entity string) (map[string]interface{}, []string, error) {
	args := m.Called(ctx, tenantID, entity)
	return args.Get(0).(map[string]interface{}), args.Get(1).([]string), args.Error(2)
}

type MockRuleRepository struct {
	mock.Mock
	// Embed SQLRepo or interface? Interface is better but we need to satisfy all methods.
	// For simplicity in this test, we only implement what SuggestService *might* use if expanded
	// but currently it relies on metaRepo + LLM + existing RuleRepo lists (stubbed in service).
	// We'll skip deep mocking of RuleRepo if not actively called in current MVP code.
	rules.RuleRepository
}

type MockSampleEntityRepository struct {
	mock.Mock
}

func (m *MockSampleEntityRepository) SampleEntities(ctx context.Context, tenantID, entityName string, sampleSize int, filter map[string]interface{}) ([]map[string]interface{}, error) {
	args := m.Called(ctx, tenantID, entityName, sampleSize, filter)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

// Tests

// TestValidateStarlarkSnippet removed (Starlark deprecated)

func TestSuggestService_SuggestRule(t *testing.T) {
	mockLLM := &MockLLMClient{}
	mockMeta := new(MockMetadataRepository)
	mockRuleRepo := new(MockRuleRepository)
	mockSampleRepo := new(MockSampleEntityRepository)

	// Engine setup for Preflight Test
	// Starlark compilers removed. Using simplified RuleEngine.
	engine := rules.NewRuleEngine(nil)

	testSvc := validation.NewTestService(mockSampleRepo, engine)

	svc := NewSuggestService(mockLLM, mockRuleRepo, mockMeta, testSvc)
	ctx := context.Background()

	// Mock Data
	mockSampleRepo.On("SampleEntities", ctx, "tenant-1", "payment", 20, mock.Anything).Return(
		[]map[string]interface{}{{"amount": 500}, {"amount": 2000}}, nil,
	)

	// In MVP, SuggestService stub for Step 1 (meta load) is commented out, so we skip mocking metaRepo call

	req := SuggestRequest{
		TenantID: "tenant-1",
		Entity:   "payment",
		Intent:   "Limit amount",
	}

	resp, err := svc.SuggestRule(ctx, req)
	require.NoError(t, err)

	assert.Equal(t, "Limit amount to 1000", resp.Description)
	assert.Equal(t, "error", resp.Severity)
	// assert.Contains(t, resp.StarlarkSrc, "amount") // Removed
	assert.True(t, resp.RuntimeOK, "Runtime checks failed")
	assert.Equal(t, 0.5, resp.TestFailureRate) // 1 fail out of 2 (2000 >= 1000)
}

func TestUpgradeAssistService_SuggestUpgrade(t *testing.T) {
	mockLLM := &MockLLMClient{} // returns static JSON
	svc := NewUpgradeAssistService(mockLLM)
	ctx := context.Background()

	// The MockLLMClient returns a `starlarkSrc` field by default which UpgradeAssist parses as `newExtensionSrc`?
	// Wait, MockLLMClient is hardcoded. We need a flexible mock for this test.

	flexibleMock := &FlexibleMockLLM{
		Response: `{"newExtensionSrc": "def tenant_ok(ctx): return True", "summary": "Updated logic"}`,
	}
	svc.llm = flexibleMock

	req := UpgradeAssistRequest{
		TenantID:   "t1",
		OldCore:    rules.CoreValidationRule{RuleKey: "k1", Version: 1},
		NewCore:    rules.CoreValidationRule{RuleKey: "k1", Version: 2},
		TenantRule: rules.TenantValidationRule{InheritMode: rules.Extend},
	}

	resp, err := svc.SuggestUpgrade(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "Updated logic", resp.Summary)
}

// Flexible Mock

type FlexibleMockLLM struct {
	Response string
}

func (m *FlexibleMockLLM) Complete(ctx context.Context, prompt string) (string, error) {
	return m.Response, nil
}
