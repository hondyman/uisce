package scoring

import (
	"context"
	"fmt"
	"time"
)

// Repository defines data access for MDM scoring inputs.
type Repository interface {
	GetAttributeTolerances(ctx context.Context) (map[string]AttributeTolerance, error)
	GetGoldenRecords(ctx context.Context, asOf time.Time) ([]GoldenRecord, error)
	GetVendorCandidates(ctx context.Context, asOf time.Time) ([]VendorCandidate, error)
	GetValueOverrides(ctx context.Context, from time.Time) ([]ValueOverrideRecord, error)
}

// Service coordinates source scoring, displacement modeling, and executive report generation.
type Service struct {
	repo      Repository
	evaluator *Evaluator
}

// NewService creates an MDM scoring service instance.
func NewService(repo Repository) *Service {
	return &Service{
		repo:      repo,
		evaluator: NewEvaluator(),
	}
}

// GetScorecardReport generates the full executive vendor quality and displacement tearsheet.
func (s *Service) GetScorecardReport(ctx context.Context, asOf time.Time, universeSize int) (*VendorScorecardReport, error) {
	tolerances, err := s.repo.GetAttributeTolerances(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tolerances: %w", err)
	}

	goldenRecords, err := s.repo.GetGoldenRecords(ctx, asOf)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch golden records: %w", err)
	}

	candidates, err := s.repo.GetVendorCandidates(ctx, asOf)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch vendor candidates: %w", err)
	}

	overrides, err := s.repo.GetValueOverrides(ctx, asOf.AddDate(0, -3, 0)) // 90 days of overrides
	if err != nil {
		return nil, fmt.Errorf("failed to fetch overrides: %w", err)
	}

	if universeSize <= 0 {
		universeSize = 42000
	}

	matrix := s.evaluator.EvaluateSubstitutionMatrix(tolerances, candidates, goldenRecords, overrides, universeSize)

	// Vendor names & costs
	vendorNames := map[string]string{
		"BBG": "Bloomberg",
		"RFT": "Refinitiv (LSEG)",
		"FDS": "FactSet",
		"ICE": "ICE Data Services",
		"SPG": "S&P Global MI",
	}
	vendorCosts := map[string]float64{
		"BBG": 2140000,
		"RFT": 1180000,
		"FDS": 720000,
		"ICE": 540000,
		"SPG": 610000,
	}

	// Compute vendor composite scores
	vendorScores := make(map[string]float64)
	vendorWeights := make(map[string]float64)
	for _, m := range matrix {
		tol := tolerances[m.AttributeCode]
		w := tol.TierWeight
		if w <= 0 {
			w = 0.3
		}
		vendorScores[m.VendorID] += m.SufficiencyRatePct * w
		vendorWeights[m.VendorID] += w
	}
	compositeQuality := make(map[string]float64)
	for vID, sum := range vendorScores {
		if totalW := vendorWeights[vID]; totalW > 0 {
			compositeQuality[vID] = sum / totalW
		}
	}

	frontier := s.evaluator.ComputeValueForMoneyFrontier(vendorNames, vendorCosts, compositeQuality)

	// Build displacement scenarios for each vendor
	hierarchy := []string{"BBG", "RFT", "FDS", "ICE", "SPG"}
	var scenarios []VendorDisplacementResult
	for _, vID := range hierarchy {
		name := vendorNames[vID]
		cost := vendorCosts[vID]
		disp := s.evaluator.SimulateVendorDisplacement(vID, name, cost, tolerances, hierarchy, candidates, goldenRecords)
		scenarios = append(scenarios, disp)
	}

	totalSpend := 0.0
	for _, c := range vendorCosts {
		totalSpend += c
	}

	report := &VendorScorecardReport{
		AsOfDate:              asOf.Format("2006-01-02"),
		UniverseSize:          universeSize,
		TiersTracked:          3,
		AnnualSpendTotal:      totalSpend,
		SubstitutionMatrix:    matrix,
		FrontierPoints:        frontier,
		DisplacementScenarios: scenarios,
	}

	// Aggregate T1 summary metrics
	var bestAltSum float64
	var t1Count int
	for _, m := range matrix {
		if m.Tier == 1 && m.VendorID != "BBG" {
			if m.SufficiencyRatePct > bestAltSum {
				bestAltSum = m.SufficiencyRatePct
			}
			t1Count++
		}
		if m.Tier == 1 && m.VendorID == "BBG" {
			report.PremiumSoloShareT1 += m.SoloRatePct
		}
	}
	if t1Count > 0 {
		report.BestAltSufficiencyT1 = bestAltSum
	}
	if len(scenarios) > 0 {
		report.DisplacementReadiness = scenarios[0].DisplacementReadiness
	}

	return report, nil
}
