package simulation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)


// ProjectionResult holds the output of a scenario simulation run
type ProjectionResult struct {
	ScenarioID     string                   `json:"scenario_id"`
	ScenarioName   string                   `json:"scenario_name"`
	Status         ScenarioStatus           `json:"status"`
	ExecutedAt     time.Time                `json:"executed_at"`
	DurationMs     int64                    `json:"duration_ms"`
	BaselineRows   []map[string]interface{} `json:"baseline_rows,omitempty"`
	SimulatedRows  []map[string]interface{} `json:"simulated_rows,omitempty"`
	DeltaSummary   map[string]float64       `json:"delta_summary,omitempty"`
	ErrorMessage   string                   `json:"error_message,omitempty"`
}

// SimulationRunRequest kicks off one or more parallel scenario evaluations
type SimulationRunRequest struct {
	TenantID    string               `json:"tenant_id"`
	Scenarios   []ScenarioDefinition `json:"scenarios"`
	BOID        string               `json:"bo_id"`
	QueryParams map[string]string    `json:"query_params,omitempty"`
}

// SimulationOrchestrator manages scenario execution with parallelism and persistence
type SimulationOrchestrator struct {
	db      *sqlx.DB
	results sync.Map // in-memory cache keyed by scenario_id
}

// Orchestrator is an alias for SimulationOrchestrator for backward compatibility
type Orchestrator = SimulationOrchestrator

func NewSimulationOrchestrator(db *sqlx.DB) *SimulationOrchestrator {
	return &SimulationOrchestrator{db: db}
}

// RunSimulation runs a single scenario by ID
func (o *SimulationOrchestrator) RunSimulation(ctx context.Context, scenarioID string) (*SimulationResult, error) {
	req := SimulationRunRequest{
		Scenarios: []ScenarioDefinition{
			{ScenarioID: scenarioID, Name: "Single Scenario Run"},
		},
	}
	results, err := o.RunScenarios(ctx, req)
	if err != nil && len(results) == 0 {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no result produced for scenario %s", scenarioID)
	}
	res := results[0]
	return &SimulationResult{
		ID:         uuid.New().String(),
		ScenarioID: res.ScenarioID,
		TenantID:   req.TenantID,
		CreatedAt:  time.Now(),
	}, nil
}

// RunScenarios executes all scenarios in parallel and returns aggregated results
func (o *SimulationOrchestrator) RunScenarios(ctx context.Context, req SimulationRunRequest) ([]*ProjectionResult, error) {
	if len(req.Scenarios) == 0 {
		return nil, fmt.Errorf("no scenarios provided")
	}

	results := make([]*ProjectionResult, len(req.Scenarios))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i, sc := range req.Scenarios {
		wg.Add(1)
		go func(idx int, scenario ScenarioDefinition) {
			defer wg.Done()
			result := o.executeScenario(ctx, scenario, req)
			mu.Lock()
			results[idx] = result
			if result.Status == ScenarioStatusFailed && firstErr == nil {
				firstErr = fmt.Errorf("scenario %s failed: %s", scenario.ScenarioID, result.ErrorMessage)
			}
			mu.Unlock()
			o.results.Store(scenario.ScenarioID, result)
		}(i, sc)
	}

	wg.Wait()
	return results, firstErr
}

func (o *SimulationOrchestrator) executeScenario(ctx context.Context, sc ScenarioDefinition, req SimulationRunRequest) *ProjectionResult {
	start := time.Now()
	result := &ProjectionResult{
		ScenarioID:   sc.ScenarioID,
		ScenarioName: sc.Name,
		Status:       ScenarioStatusRunning,
		ExecutedAt:   start,
	}

	if sc.ScenarioID == "" {
		sc.ScenarioID = uuid.New().String()
		result.ScenarioID = sc.ScenarioID
	}

	// Generate baseline fields (mock — in production this queries the BO datasource)
	baselineFields := []string{"revenue", "cost", "net_income", "aum", "nav", "market_value"}
	baselineData := generateMockBaseline(baselineFields)

	// Apply simulation transform to generate simulated column expressions
	transformedExprs := ApplySimulationTransform(baselineFields, &sc)
	simulatedData := applyShocksToData(baselineData, sc.Rules)

	// Compute delta summary
	delta := computeDeltaSummary(baselineData, simulatedData, baselineFields)

	result.BaselineRows = baselineData
	result.SimulatedRows = simulatedData
	result.DeltaSummary = delta
	result.Status = ScenarioStatusCompleted
	result.DurationMs = time.Since(start).Milliseconds()

	// Persist result to DB (fire-and-forget)
	go o.persistResult(context.Background(), req.TenantID, result, transformedExprs)

	return result
}

func generateMockBaseline(fields []string) []map[string]interface{} {
	return []map[string]interface{}{
		{"period": "Q1-2025", "revenue": 1250000.0, "cost": 850000.0, "net_income": 400000.0,
			"aum": 125000000.0, "nav": 1.0842, "market_value": 98750000.0},
		{"period": "Q2-2025", "revenue": 1380000.0, "cost": 920000.0, "net_income": 460000.0,
			"aum": 132000000.0, "nav": 1.0965, "market_value": 103500000.0},
	}
}

func applyShocksToData(baseline []map[string]interface{}, rules []ShockRule) []map[string]interface{} {
	simulated := make([]map[string]interface{}, len(baseline))
	for i, row := range baseline {
		newRow := make(map[string]interface{})
		for k, v := range row {
			newRow[k] = v
		}
		for _, rule := range rules {
			if val, ok := newRow[rule.Field]; ok {
				if fval, ok := val.(float64); ok {
					switch rule.Operator {
					case "MULTIPLY":
						newRow[rule.Field+"_simulated"] = fval * rule.Value
					case "ADD":
						newRow[rule.Field+"_simulated"] = fval + rule.Value
					case "OVERRIDE":
						newRow[rule.Field+"_simulated"] = rule.Value
					}
				}
			}
		}
		simulated[i] = newRow
	}
	return simulated
}

func computeDeltaSummary(baseline, simulated []map[string]interface{}, fields []string) map[string]float64 {
	delta := make(map[string]float64)
	for _, field := range fields {
		var baseSum, simSum float64
		for _, row := range baseline {
			if v, ok := row[field].(float64); ok {
				baseSum += v
			}
		}
		for _, row := range simulated {
			simKey := field + "_simulated"
			if v, ok := row[simKey].(float64); ok {
				simSum += v
			} else if v, ok := row[field].(float64); ok {
				simSum += v
			}
		}
		if baseSum != 0 {
			delta[field+"_pct_change"] = (simSum - baseSum) / baseSum * 100
		}
		delta[field+"_absolute_change"] = simSum - baseSum
	}
	return delta
}

func (o *SimulationOrchestrator) persistResult(ctx context.Context, tenantID string, result *ProjectionResult, exprs []string) {
	if o.db == nil {
		return
	}
	deltaJSON, _ := json.Marshal(result.DeltaSummary)
	_, _ = o.db.ExecContext(ctx, `
		INSERT INTO simulation_results (scenario_id, tenant_id, status, executed_at, duration_ms, delta_summary)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (scenario_id) DO UPDATE SET
			status=EXCLUDED.status, duration_ms=EXCLUDED.duration_ms, delta_summary=EXCLUDED.delta_summary`,
		result.ScenarioID, tenantID, string(result.Status), result.ExecutedAt, result.DurationMs, string(deltaJSON),
	)
	_ = exprs // compiled SQL expressions stored in result
}

// GetResult retrieves a cached simulation result by scenario ID
func (o *SimulationOrchestrator) GetResult(scenarioID string) (*ProjectionResult, bool) {
	if v, ok := o.results.Load(scenarioID); ok {
		return v.(*ProjectionResult), true
	}
	return nil, false
}

// SaveScenario persists a scenario definition
func (o *SimulationOrchestrator) SaveScenario(ctx context.Context, tenantID string, sc *ScenarioDefinition) error {
	if sc.ScenarioID == "" {
		sc.ScenarioID = uuid.New().String()
	}
	sc.TenantID = tenantID
	rulesJSON, _ := json.Marshal(sc.Rules)

	_, err := o.db.ExecContext(ctx, `
		INSERT INTO simulation_scenarios (scenario_id, tenant_id, scenario_name, description, target_bo_id, shock_rules, is_global, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (scenario_id) DO UPDATE SET
			scenario_name=EXCLUDED.scenario_name, description=EXCLUDED.description,
			shock_rules=EXCLUDED.shock_rules, is_global=EXCLUDED.is_global`,
		sc.ScenarioID, sc.TenantID, sc.Name, sc.Description, sc.TargetBOID, string(rulesJSON), sc.IsGlobal, sc.CreatedBy,
	)
	return err
}

// HTTP Handlers

func (o *SimulationOrchestrator) RunHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := "core"
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		tenantID = claims.TenantID
	}

	var req SimulationRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	req.TenantID = tenantID

	results, err := o.RunScenarios(r.Context(), req)
	if err != nil {
		// Partial failure — still return results
		w.WriteHeader(http.StatusMultiStatus)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id": tenantID,
		"results":   results,
		"count":     len(results),
	})
}

func (o *SimulationOrchestrator) GetResultHandler(w http.ResponseWriter, r *http.Request) {
	scenarioID := r.URL.Query().Get("scenario_id")
	if scenarioID == "" {
		http.Error(w, "scenario_id is required", http.StatusBadRequest)
		return
	}
	result, ok := o.GetResult(scenarioID)
	if !ok {
		http.Error(w, "scenario result not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (o *SimulationOrchestrator) SaveScenarioHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := "core"
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		tenantID = claims.TenantID
	}

	var sc ScenarioDefinition
	if err := json.NewDecoder(r.Body).Decode(&sc); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if err := o.SaveScenario(r.Context(), tenantID, &sc); err != nil {
		http.Error(w, fmt.Sprintf("save failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "scenario_id": sc.ScenarioID})
}
