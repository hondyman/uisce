package analytics

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockGraphProvider for testing
type MockGraphProvider struct {
	mock.Mock
}

func (m *MockGraphProvider) GetNodeByID(nodeID uuid.UUID) (*SemanticNode, error) {
	args := m.Called(nodeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SemanticNode), args.Error(1)
}

func (m *MockGraphProvider) GetEdgesByType(sourceNodeID uuid.UUID, edgeType EdgeType) ([]SemanticEdge, error) {
	args := m.Called(sourceNodeID, edgeType)
	return args.Get(0).([]SemanticEdge), args.Error(1)
}

func (m *MockGraphProvider) GetOutgoingEdges(nodeID uuid.UUID) ([]SemanticEdge, error) {
	args := m.Called(nodeID)
	return args.Get(0).([]SemanticEdge), args.Error(1)
}

func TestHashGenerator_ComputeHash_Determinism(t *testing.T) {
	mockProvider := new(MockGraphProvider)
	hashGen := NewHashGenerator(mockProvider)
	datasourceID := uuid.New()
	boID := uuid.New()

	// 1. Setup BO Node
	boNode := &SemanticNode{
		ID:       boID,
		NodeName: "SalesOrder",
		Properties: map[string]interface{}{
			"domain":            "Sales",
			"driving_table":     "orders",
			"governance_status": "APPROVED",
		},
	}

	// 2. Setup Term Node
	termID := uuid.New()
	termNode := &SemanticNode{
		ID:       termID,
		NodeName: "TotalAmount",
		Properties: map[string]interface{}{
			"data_type":         "DECIMAL",
			"category":          "Measure",
			"governance_status": "APPROVED",
		},
		Config: map[string]interface{}{
			"physical_mappings": []interface{}{
				map[string]interface{}{
					"table":      "orders",
					"column":     "total_amt",
					"priority":   1.0,
					"is_default": true,
				},
			},
		},
	}

	// 3. Mock Expectations
	mockProvider.On("GetNodeByID", boID).Return(boNode, nil)
	mockProvider.On("GetEdgesByType", boID, EdgeTypeBOHasTerm).Return([]SemanticEdge{
		{TargetNodeID: termID},
	}, nil)
	mockProvider.On("GetNodeByID", termID).Return(termNode, nil)
	// No calcs for this simple test
	mockProvider.On("GetEdgesByType", boID, EdgeTypeBOHasCalc).Return([]SemanticEdge{}, nil)

	// Execute
	hash1, err := hashGen.ComputeHash(boID, datasourceID)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash1)

	// Execute again - should be same
	hash2, err := hashGen.ComputeHash(boID, datasourceID)
	assert.NoError(t, err)
	assert.Equal(t, hash1, hash2, "Hash must be deterministic")
}

func TestHashGenerator_ComputeHash_Sensitivity(t *testing.T) {
	mockProvider := new(MockGraphProvider)
	hashGen := NewHashGenerator(mockProvider)
	datasourceID := uuid.New()
	boID := uuid.New()

	// 1. Setup BO Node (Base)
	boNode := &SemanticNode{
		ID:       boID,
		NodeName: "SalesOrder",
		Properties: map[string]interface{}{
			"domain":            "Sales",
			"driving_table":     "orders",
			"governance_status": "APPROVED",
		},
	}

	// 2. Setup Term Node (Base)
	termID := uuid.New()
	termNode := &SemanticNode{
		ID:       termID,
		NodeName: "TotalAmount",
		Properties: map[string]interface{}{
			"data_type":         "DECIMAL",
			"category":          "Measure",
			"governance_status": "APPROVED",
		},
		Config: map[string]interface{}{
			"physical_mappings": []interface{}{
				map[string]interface{}{
					"table":      "orders",
					"column":     "total_amt",
					"priority":   1.0,
					"is_default": true,
				},
			},
		},
	}

	// Mock Expectations for Run 1
	mockProvider.On("GetNodeByID", boID).Return(boNode, nil).Times(2) // Run 1 & 2
	mockProvider.On("GetEdgesByType", boID, EdgeTypeBOHasTerm).Return([]SemanticEdge{
		{TargetNodeID: termID},
	}, nil).Times(2) // Run 1 & 2

	// Term Node Lookup - returns the SAME object pointer in mock usually,
	// but we want simulate a change. We can use .Return(func... or return different objects)

	// Approach: Run hash1 with one provider setup, run hash2 with another provider setup?
	// Or just reset expectations?
	// Easiest is to simulate behavior within the return.

	// Run 1: Normal term
	mockProvider.On("GetNodeByID", termID).Return(termNode, nil).Once()
	mockProvider.On("GetEdgesByType", boID, EdgeTypeBOHasCalc).Return([]SemanticEdge{}, nil).Times(2)

	hash1, err := hashGen.ComputeHash(boID, datasourceID)
	assert.NoError(t, err)

	// Run 2: Modified term (e.g., column mapping change)
	modifiedTermNode := &SemanticNode{
		ID:       termID,
		NodeName: "TotalAmount",
		Properties: map[string]interface{}{
			"data_type":         "DECIMAL",
			"category":          "Measure",
			"governance_status": "APPROVED",
		},
		Config: map[string]interface{}{
			"physical_mappings": []interface{}{
				map[string]interface{}{
					"table":      "orders",
					"column":     "total_amount_v2", // Changed column
					"priority":   1.0,
					"is_default": true,
				},
			},
		},
	}
	mockProvider.On("GetNodeByID", termID).Return(modifiedTermNode, nil).Once()

	hash2, err := hashGen.ComputeHash(boID, datasourceID)
	assert.NoError(t, err)

	assert.NotEqual(t, hash1, hash2, "Hash must change when term mapping changes")
}
