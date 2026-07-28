package simulation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

type ScenarioService struct {
	db *sqlx.DB
}

func NewScenarioService(db *sqlx.DB) *ScenarioService {
	return &ScenarioService{db: db}
}

func (s *ScenarioService) ListScenarios(ctx context.Context, tenantID string) ([]SimulationScenario, error) {
	if s.db == nil {
		// Mock fallback scenarios if DB is uninitialized
		return []SimulationScenario{
			{
				ScenarioID:  "scen-rate-hike-150",
				TenantID:    tenantID,
				Name:        "Q3 Interest Rate Shock (+150bps)",
				Description: "Applies a +150bps discount yield curve shock to fixed income and cash balances",
				TargetBOID:  "customers",
				Rules: []ShockRule{
					{Field: "balance", Operator: "MULTIPLY", Value: 0.92},
					{Field: "market_value", Operator: "MULTIPLY", Value: 0.88},
				},
				IsGlobal:  true,
				CreatedBy: "system",
			},
			{
				ScenarioID:  "scen-tech-bull-20",
				TenantID:    tenantID,
				Name:        "Tech Rally (+20% Growth Projection)",
				Description: "Simulates 20% equity expansion across tech portfolio holdings",
				TargetBOID:  "customers",
				Rules: []ShockRule{
					{Field: "balance", Operator: "MULTIPLY", Value: 1.20},
					{Field: "market_value", Operator: "MULTIPLY", Value: 1.20},
				},
				IsGlobal:  true,
				CreatedBy: "system",
			},
		}, nil
	}

	query := `SELECT scenario_id, tenant_id, scenario_name, description, target_bo_id, shock_rules, is_global, created_by FROM public.simulation_scenarios WHERE tenant_id = $1 OR is_global = true`
	rows, err := s.db.QueryxContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scenarios []SimulationScenario
	for rows.Next() {
		var sc SimulationScenario
		var shockJSON []byte
		if err := rows.Scan(&sc.ScenarioID, &sc.TenantID, &sc.Name, &sc.Description, &sc.TargetBOID, &shockJSON, &sc.IsGlobal, &sc.CreatedBy); err != nil {
			continue
		}
		json.Unmarshal(shockJSON, &sc.Rules)
		scenarios = append(scenarios, sc)
	}

	return scenarios, nil
}

func (s *ScenarioService) CreateScenario(ctx context.Context, sc SimulationScenario) error {
	if sc.ScenarioID == "" {
		sc.ScenarioID = uuid.New().String()
	}

	if s.db != nil {
		shockB, _ := json.Marshal(sc.Rules)
		query := `
			INSERT INTO public.simulation_scenarios (scenario_id, tenant_id, scenario_name, description, target_bo_id, shock_rules, is_global, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		_, err := s.db.ExecContext(ctx, query, sc.ScenarioID, sc.TenantID, sc.Name, sc.Description, sc.TargetBOID, shockB, sc.IsGlobal, sc.CreatedBy)
		return err
	}
	return nil
}

// HTTP Handlers

func (s *ScenarioService) ListScenariosHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := "core"
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		tenantID = claims.TenantID
	}

	scenarios, err := s.ListScenarios(r.Context(), tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch scenarios: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"scenarios": scenarios})
}

func (s *ScenarioService) CreateScenarioHandler(w http.ResponseWriter, r *http.Request) {
	var sc SimulationScenario
	if err := json.NewDecoder(r.Body).Decode(&sc); err != nil {
		http.Error(w, "Invalid scenario payload", http.StatusBadRequest)
		return
	}

	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		sc.TenantID = claims.TenantID
	}
	if sc.TenantID == "" {
		sc.TenantID = "core"
	}

	if err := s.CreateScenario(r.Context(), sc); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create scenario: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "scenario": sc})
}
