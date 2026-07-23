package simulation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Orchestrator manages the execution of simulation workflows
type Orchestrator struct {
	service          Service
	riskEngine       RiskEngine
	calcEngine       CalculationEngine
	rebalanceEngine  *RebalanceEngine
	complianceEngine ComplianceEngine
	logger           *zap.Logger
}

// NewOrchestrator creates a new simulation orchestrator
func NewOrchestrator(svc Service, risk RiskEngine, calc CalculationEngine, rebal *RebalanceEngine, comp ComplianceEngine, logger *zap.Logger) *Orchestrator {
	return &Orchestrator{
		service:          svc,
		riskEngine:       risk,
		calcEngine:       calc,
		rebalanceEngine:  rebal,
		complianceEngine: comp,
		logger:           logger,
	}
}

// RunSimulation executes a scenario end-to-end
// In a full production system, this would trigger a Temporal workflow.
// Here we execute the steps synchronously for the MVP.
func (o *Orchestrator) RunSimulation(ctx context.Context, scenarioID string) (*SimulationResult, error) {
	o.logger.Info("starting simulation run", zap.String("scenarioID", scenarioID))

	// 1. Create Run Record
	runID := uuid.NewString()
	// In a real impl, we'd persist this run record as RUNNING

	// 2. Load Scenario & Deltas
	scenario, err := o.service.GetScenario(ctx, scenarioID)
	if err != nil {
		return nil, fmt.Errorf("load scenario: %w", err)
	}
	if scenario == nil {
		return nil, fmt.Errorf("scenario not found: %s", scenarioID)
	}

	deltas, err := o.service.GetDeltas(ctx, scenarioID)
	if err != nil {
		return nil, fmt.Errorf("load deltas: %w", err)
	}

	// 3. Resolve Current State (Mocked)
	// We would normally fetch current positions for the tenant/portfolio
	currentPositions := map[string]float64{
		"TSLA": 1000.0,
		"AAPL": 500.0,
		"MSFT": 200.0,
		"USD":  1_000_000.0,
		"EUR":  50_000.0,
	}

	// 4. Pre-process Deltas (Expand Rebalance Rules & Extract Shocks)
	var expandedDeltas []*SimulationDelta
	marketShock := &MarketShock{}

	for _, d := range deltas {
		if d.DeltaType == DeltaTypeRebalance {
			// Parse Rule
			var rule RebalanceRule
			if err := json.Unmarshal(d.Changes, &rule); err != nil {
				o.logger.Error("failed to parse rebalance rule", zap.Error(err))
				continue
			}
			// Current prices mock
			mockPrices := map[string]float64{
				"TSLA": 250.0, "AAPL": 150.0, "MSFT": 300.0,
				"GOOGL": 2800.0, "AMZN": 3400.0,
			}

			generated, err := o.rebalanceEngine.GenerateDeltas(ctx, currentPositions, mockPrices, &rule)
			if err != nil {
				o.logger.Error("failed to generate rebalance deltas", zap.Error(err))
				continue
			}
			expandedDeltas = append(expandedDeltas, generated...)
		} else if d.DeltaType == DeltaTypeMarket {
			// Parse Market Shock
			var shock MarketShock
			if err := json.Unmarshal(d.Changes, &shock); err == nil {
				marketShock.ParallelShiftBps += shock.ParallelShiftBps
				marketShock.EquityShockPct += shock.EquityShockPct
				marketShock.VolShockPct += shock.VolShockPct
				marketShock.FXShockPct += shock.FXShockPct
			}
			expandedDeltas = append(expandedDeltas, d)
		} else {
			expandedDeltas = append(expandedDeltas, d)
		}
	}

	// 5. Apply Deltas to create Simulated State
	simulatedPositions := o.applyDeltas(currentPositions, expandedDeltas)

	// 5. Calculate Metrics (Baseline vs Simulated)
	baselineMetrics, err := o.calcEngine.ComputeMetrics(ctx, MetricRequest{
		TenantID:  scenario.TenantID,
		AsOf:      scenario.BaseAsOf,
		Positions: currentPositions,
	})
	if err != nil {
		return nil, fmt.Errorf("calc baseline: %w", err)
	}

	simulatedMetricsResponse, err := o.calcEngine.ComputeMetrics(ctx, MetricRequest{
		TenantID:  scenario.TenantID,
		AsOf:      scenario.BaseAsOf,
		Positions: simulatedPositions,
		Shocks:    marketShock,
	})
	if err != nil {
		return nil, fmt.Errorf("calc simulated: %w", err)
	}

	// 6. Calculate Risk
	// Convert map to slice for Risk Engine
	var riskPositions []PositionInput
	for k, v := range simulatedPositions {
		riskPositions = append(riskPositions, PositionInput{AssetID: k, Quantity: v})
	}

	riskResponse, err := o.riskEngine.ComputeRisk(ctx, RiskRequest{
		TenantID:    scenario.TenantID,
		PortfolioID: "portfolio:default", // Mock
		HorizonDays: 1,
		AsOf:        scenario.BaseAsOf,
		Positions:   riskPositions,
		MarketData:  MarketSnapshot{ScenarioDate: scenario.BaseAsOf},
		Shocks:      marketShock,
	})
	if err != nil {
		return nil, fmt.Errorf("calc risk: %w", err)
	}

	// 7. Check Compliance
	complianceRes, err := o.complianceEngine.CheckCompliance(ctx, ComplianceRequest{
		TenantID:  scenario.TenantID,
		AsOf:      scenario.BaseAsOf,
		Positions: simulatedPositions,
	})
	if err != nil {
		return nil, fmt.Errorf("check compliance: %w", err)
	}

	// 8. Aggregate Results
	resultID := uuid.NewString()
	metrics := o.mergeMetrics(resultID, baselineMetrics.Metrics, simulatedMetricsResponse.Metrics, riskResponse.Metrics)

	// Add Compliance Metrics
	for _, m := range complianceRes.Metrics {
		m.ID = uuid.NewString()
		m.ResultID = resultID
		m.BaselineValue = 0
		m.DeltaValue = m.SimulatedValue
		metrics = append(metrics, m)
	}

	// Summary calculation (e.g. Total NAV delta)
	summary := map[string]float64{
		"nav_delta": 0.0,
		"var_delta": 0.0,
	}
	for _, m := range metrics {
		if m.MetricName == "NAV" {
			summary["nav_delta"] = m.DeltaValue
		}
	}
	summaryJSON, _ := json.Marshal(summary)

	compJSON, _ := json.Marshal(complianceRes)

	res := &SimulationResult{
		ID:                resultID,
		RunID:             runID,
		ScenarioID:        scenario.ID,
		TenantID:          scenario.TenantID,
		Summary:           summaryJSON,
		ComplianceSummary: compJSON,
		ImpactedEntities:  []string{"portfolio:default"}, // Mock
		CreatedAt:         time.Now().UTC(),
	}

	o.logger.Info("simulation completed", zap.String("runID", runID))
	return res, nil
}

func (o *Orchestrator) applyDeltas(current map[string]float64, deltas []*SimulationDelta) map[string]float64 {
	simulated := make(map[string]float64)
	for k, v := range current {
		simulated[k] = v
	}

	for _, d := range deltas {
		// Mock logic for parsing JSON changes
		var changes map[string]interface{}
		if err := json.Unmarshal(d.Changes, &changes); err != nil {
			o.logger.Warn("failed to parse delta changes", zap.String("deltaID", d.ID))
			continue
		}

		// Handle Position Delta
		if d.DeltaType == DeltaTypePosition {
			symbol := d.BOID // Assuming BOID is symbol for this MVP
			if val, ok := changes["quantityPct"]; ok {
				if pct, ok := val.(float64); ok {
					original := simulated[symbol]
					simulated[symbol] = original * (1 + pct)
				}
			}
			if val, ok := changes["quantity"]; ok {
				if qty, ok := val.(float64); ok {
					simulated[symbol] += qty
				}
			}
		}
		// ... handle other delta types
	}
	return simulated
}

func (o *Orchestrator) mergeMetrics(resultID string, baseline, simulated, risk []SimulationMetric) []SimulationMetric {
	var merged []SimulationMetric
	baselineMap := make(map[string]float64)

	for _, m := range baseline {
		baselineMap[m.MetricName] = m.SimulatedValue
	}

	// Process Calculation Metrics
	for _, m := range simulated {
		baseVal := baselineMap[m.MetricName]
		m.ID = uuid.NewString()
		m.ResultID = resultID
		m.BaselineValue = baseVal
		m.DeltaValue = m.SimulatedValue - baseVal
		merged = append(merged, m)
	}

	// Process Risk Metrics (assuming 0 baseline if not calculated for baseline in this simplified flow)
	// In production, we'd run risk for baseline too
	for _, m := range risk {
		m.ID = uuid.NewString()
		m.ResultID = resultID
		m.BaselineValue = 0 // Placeholder
		m.DeltaValue = m.SimulatedValue
		merged = append(merged, m)
	}

	return merged
}
