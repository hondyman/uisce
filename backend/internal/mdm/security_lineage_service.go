package mdm

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/goldcopy"
)

// SecurityLineageService traces data from multiple sources to portfolio impact.
type SecurityLineageService struct {
	graphService *analytics.SemanticGraphService
	securityRepo *goldcopy.Repository
}

func NewSecurityLineageService(graph *analytics.SemanticGraphService, repo *goldcopy.Repository) *SecurityLineageService {
	return &SecurityLineageService{
		graphService: graph,
		securityRepo: repo,
	}
}

// SecurityLineageNode represents a node in the lineage tree.
type SecurityLineageNode struct {
	ID         string                `json:"id"`
	Type       string                `json:"type"`
	Name       string                `json:"name"`
	Value      interface{}           `json:"value,omitempty"`
	Confidence float64               `json:"confidence"`
	Source     string                `json:"source,omitempty"`
	Children   []SecurityLineageNode `json:"children,omitempty"`
}

// GetSecurityLineage returns the full lineage for a security.
func (s *SecurityLineageService) GetSecurityLineage(ctx context.Context, tenantID uuid.UUID, securityID string) (*SecurityLineageNode, error) {
	// 1. Fetch security master
	sec, err := s.securityRepo.GetCurrentSecurity(ctx, tenantID, securityID)
	if err != nil {
		return nil, err
	}
	if sec == nil {
		return nil, fmt.Errorf("security %s not found", securityID)
	}

	// 2. Build root node
	root := &SecurityLineageNode{
		ID:         sec.SecurityID,
		Type:       "Security",
		Name:       sec.SecurityName,
		Confidence: float64(sec.ConfidenceScore) / 100.0,
	}

	// 3. Drill down into source systems per field (Lineage)
	for field, source := range sec.SourceSystems {
		child := SecurityLineageNode{
			ID:     field,
			Type:   "Field",
			Name:   field,
			Source: source,
		}
		root.Children = append(root.Children, child)
	}

	return root, nil
}

// GetSecurityImpactAnalysis simulates the effect of security attribute changes on portfolio metrics.
func (s *SecurityLineageService) GetSecurityImpactAnalysis(ctx context.Context, tenantID uuid.UUID, securityID string, changes map[string]interface{}) (map[string]interface{}, error) {
	// This is a placeholder for the actual simulation logic
	// In a real implementation, we would re-run CalculatePortfolioAnalytics with the modified security attributes
	impact := make(map[string]interface{})
	impact["status"] = "simulated"
	impact["simulated_changes"] = changes
	impact["portfolio_impacts"] = []map[string]interface{}{
		{
			"portfolio_id":    uuid.New().String(),
			"nav_change":      2.5,
			"exposure_change": -1.2,
		},
	}
	return impact, nil
}
