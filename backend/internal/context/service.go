package context

import (
	"github.com/google/uuid"
)

type Service struct {
	// In a real implementation, this would hold references to other services/repos
	// profileRepo ProfileRepository
	// portfolioSvc PortfolioService
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) GetContext(tenantID, clientID string) (*ClientContext, error) {
	// Mock implementation for Phase 1 Skeleton
	// In reality, this would fetch from DBs and other services

	return &ClientContext{
		ClientID: clientID,
		TenantID: tenantID,
		Profile: ClientProfile{
			Name:      "John Doe",
			TaxStatus: "taxable",
			RiskScore: 7,
		},
		Portfolio: PortfolioSummary{
			TotalValue:        1250000.00,
			UnrealizedLoss:    -15000.00,
			UnrealizedLossPct: -0.012,
			DriftPct:          0.04,
			CashBalance:       50000.00,
		},
		RecentSignals: []Signal{
			{
				ID:   uuid.New().String(),
				Type: "LOGIN",
			},
		},
		Compliance: ComplianceStatus{
			IsRestricted: false,
		},
	}, nil
}
