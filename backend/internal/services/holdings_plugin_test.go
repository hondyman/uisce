package services

import (
	"testing"

	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/stretchr/testify/assert"
)

type MockRepo struct{}

func (m *MockRepo) GetTerm(id string) (*models.SemanticTerm, error) {
	return &models.SemanticTerm{NodeName: "holding.market_value_resolved"}, nil
}

func (m *MockRepo) GetTermsByTable(tableName string) ([]*models.SemanticTerm, error) {
	return nil, nil
}

func TestHoldingsPlugin_Support(t *testing.T) {
	plugin := &HoldingsPlugin{}

	tests := []struct {
		name     string
		termName string
		want     bool
	}{
		{"Supported", "holding.market_value_resolved", true},
		{"Supported Specific", "holding.market_value_sod", true},
		{"Unsupported", "financial.irr", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			term := &models.SemanticTerm{NodeName: tt.termName}
			assert.Equal(t, tt.want, plugin.Supports(term))
		})
	}
}

func TestDetectHoldingAnomalies(t *testing.T) {
	// Directly test the helper function

	tests := []struct {
		name             string
		rows             []models.ExplainRow
		wantAnomalyCount int
		wantType         string
	}{
		{
			name: "No Anomalies",
			rows: []models.ExplainRow{
				{Key: map[string]interface{}{"id": "1", "holding_type": "SOD"}},
				{Key: map[string]interface{}{"id": "2", "holding_type": "EOD"}},
			},
			wantAnomalyCount: 0,
		},
		{
			name: "Missing Holding Type",
			rows: []models.ExplainRow{
				{Key: map[string]interface{}{"id": "1", "holding_type": ""}},
			},
			wantAnomalyCount: 1,
			wantType:         "MISSING_HOLDING_TYPE",
		},
		{
			name: "Duplicate IDs (Anomaly if logic enforced)",
			rows: []models.ExplainRow{
				{Key: map[string]interface{}{"id": "1", "holding_type": "SOD"}},
				{Key: map[string]interface{}{"id": "1", "holding_type": "EOD"}},
			},
			wantAnomalyCount: 1,
			wantType:         "DUPLICATE_HOLDING_ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anomalies := detectHoldingAnomalies(tt.rows)
			assert.Len(t, anomalies, tt.wantAnomalyCount)
			if tt.wantAnomalyCount > 0 {
				assert.Equal(t, tt.wantType, anomalies[0].Type)
			}
		})
	}
}

func TestHoldingsPlugin_Resolve(t *testing.T) {
	plugin := &HoldingsPlugin{Repo: &MockRepo{}}

	term := &models.SemanticTerm{NodeName: "holding.market_value_resolved"}
	res, err := plugin.Resolve(term, "entity-123")

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.NotEmpty(t, res.Path)
	// Check that rows are generated (mock data)
	assert.NotEmpty(t, res.Rows)
}
