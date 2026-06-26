// SCENARIO ANALYSIS - CODE EXAMPLES & TEMPLATES

// ============================================================================
// 1. BACKEND - TEMPORAL WORKFLOW
// ============================================================================

// File: backend/temporal/workflows/scenario_analysis.go

package workflows

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
	"your-module/temporal/activities"
)

type ScenarioAnalysisInput struct {
	PortfolioID string `json:"portfolio_id"`
	Scenario    string `json:"scenario"`
}

type BaseCase struct {
	AUM                float64                  `json:"aum"`
	Sharpe             float64                  `json:"sharpe"`
	Risk               float64                  `json:"risk"`
	Status             string                   `json:"status"`
	AssetAllocation    []AssetAllocationItem    `json:"assetAllocation"`
}

type ScenarioCase struct {
	AUM                float64                  `json:"aum"`
	AUMChange          float64                  `json:"aumChange"` // percentage
	Sharpe             float64                  `json:"sharpe"`
	SharpeChange       float64                  `json:"sharpeChange"`
	Risk               float64                  `json:"risk"`
	RiskChange         float64                  `json:"riskChange"`
	Status             string                   `json:"status"`
	AssetAllocation    []AssetAllocationItem    `json:"assetAllocation"`
}

type AssetAllocationItem struct {
	Asset      string  `json:"asset"`
	Percentage float64 `json:"percentage"`
}

type ComparisonMetrics struct {
	AUMDifference      float64 `json:"aumDifference"`
	SharpeDifference   float64 `json:"sharpeDifference"`
	RiskDifference     float64 `json:"riskDifference"`
}

type ScenarioAnalysisResult struct {
	BaseCase      BaseCase           `json:"baseCase"`
	ScenarioCase  ScenarioCase       `json:"scenarioCase"`
	Comparison    ComparisonMetrics  `json:"comparison"`
}

// ScenarioAnalysis is the main workflow
func ScenarioAnalysis(
	ctx workflow.Context,
	input ScenarioAnalysisInput,
) (ScenarioAnalysisResult, error) {
	
	// Set up activity options with retry policy
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    10 * time.Second,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// 1. Fetch current portfolio data
	var portfolio BaseCase
	err := workflow.ExecuteActivity(
		ctx,
		activities.FetchPortfolioData,
		input.PortfolioID,
	).Get(ctx, &portfolio)
	
	if err != nil {
		return ScenarioAnalysisResult{}, fmt.Errorf("failed to fetch portfolio: %w", err)
	}

	// 2. Project scenario using xAI
	var scenarioData ScenarioCase
	err = workflow.ExecuteActivity(
		ctx,
		activities.ProjectScenario,
		input.PortfolioID,
		input.Scenario,
		portfolio,
	).Get(ctx, &scenarioData)
	
	if err != nil {
		return ScenarioAnalysisResult{}, fmt.Errorf("failed to project scenario: %w", err)
	}

	// 3. Calculate comparison metrics
	var comparison ComparisonMetrics
	err = workflow.ExecuteActivity(
		ctx,
		activities.CalculateComparison,
		portfolio,
		scenarioData,
	).Get(ctx, &comparison)
	
	if err != nil {
		return ScenarioAnalysisResult{}, fmt.Errorf("failed to calculate comparison: %w", err)
	}

	// 4. Store result (fire-and-forget)
	workflow.ExecuteActivity(
		ctx,
		activities.StoreAnalysisResult,
		input.PortfolioID,
		input.Scenario,
		ScenarioAnalysisResult{
			BaseCase:     portfolio,
			ScenarioCase: scenarioData,
			Comparison:   comparison,
		},
	)

	return ScenarioAnalysisResult{
		BaseCase:     portfolio,
		ScenarioCase: scenarioData,
		Comparison:   comparison,
	}, nil
}

// ============================================================================
// 2. BACKEND - ACTIVITIES
// ============================================================================

// File: backend/temporal/activities/scenario_activities.go

package activities

import (
	"context"
	"encoding/json"
	"fmt"

	"database/sql"
	"github.com/hondyman/semlayer/backend/models"
	"your-module/xai"
)

type PortfolioFetcher interface {
	GetPortfolioByID(ctx context.Context, id string) (*models.Portfolio, error)
}

// FetchPortfolioData retrieves current portfolio data
func FetchPortfolioData(ctx context.Context, portfolioID string) (map[string]any, error) {
	// Implementation: Fetch from database or Hasura
	// Return portfolio with AUM, holdings, current performance
	
	return map[string]any{
		"aum": 1200000.0,
		"sharpe": 1.8,
		"risk": 45.0,
		"status": "Optimized",
		"assetAllocation": []map[string]any{
			{"asset": "Stocks", "percentage": 60},
			{"asset": "Bonds", "percentage": 30},
			{"asset": "Cash", "percentage": 10},
		},
	}, nil
}

// ProjectScenario uses xAI to project portfolio performance
func ProjectScenario(
	ctx context.Context,
	portfolioID string,
	scenario string,
	basePortfolio map[string]any,
) (map[string]any, error) {
	
	// Call xAI API for scenario projection
	aiClient := xai.NewClient()
	
	prompt := fmt.Sprintf(`
		Analyze this portfolio under the scenario: %s
		Portfolio: %v
		
		Provide:
		1. Projected AUM change (%)
		2. New Sharpe ratio
		3. New risk score
		4. Status (Optimized/At Risk)
		5. New asset allocation
		
		Return as JSON.
	`, scenario, basePortfolio)
	
	result, err := aiClient.Complete(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("xAI projection failed: %w", err)
	}
	
	var scenarioData map[string]any
	err = json.Unmarshal([]byte(result), &scenarioData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}
	
	return scenarioData, nil
}

// CalculateComparison computes metrics between base and scenario
func CalculateComparison(
	ctx context.Context,
	baseCase map[string]any,
	scenarioCase map[string]any,
) (map[string]any, error) {
	
	baseAUM := baseCase["aum"].(float64)
	scenarioAUM := scenarioCase["aum"].(float64)
	
	baseSharpe := baseCase["sharpe"].(float64)
	scenarioSharpe := scenarioCase["sharpe"].(float64)
	
	baseRisk := baseCase["risk"].(float64)
	scenarioRisk := scenarioCase["risk"].(float64)
	
	return map[string]any{
		"aumDifference":    scenarioAUM - baseAUM,
		"sharpeDifference": scenarioSharpe - baseSharpe,
		"riskDifference":   scenarioRisk - baseRisk,
	}, nil
}

// StoreAnalysisResult saves analysis to database
func StoreAnalysisResult(
	ctx context.Context,
	portfolioID string,
	scenario string,
	result map[string]any,
) error {
	
	// Store in scenario_analyses table
	// Implementation: Use your ORM or SQL client
	
	return nil
}

// ============================================================================
// 3. BACKEND - API ROUTES
// ============================================================================

// File: backend/internal/api/scenario_routes.go

package api

import (
	"net/http"
	"context"
	"github.com/gin-gonic/gin"
	"go.temporal.io/sdk/client"
	"your-module/abac"
	"your-module/temporal/workflows"
)

func RegisterScenarioRoutes(r *gin.Engine, tc client.Client) {
	
	// POST /api/portfolio/:id/scenario
	r.POST("/api/portfolio/:id/scenario", func(c *gin.Context) {
		portfolioID := c.Param("id")
		tenantID := c.GetHeader("X-Tenant-ID")
		
		// ABAC authorization check
		if !abac.Evaluate(c, "analyze", "portfolio") {
			c.JSON(403, gin.H{"error": "unauthorized"})
			return
		}
		
		// Bind request
		var req struct {
			Scenario string `json:"scenario" binding:"required"`
		}
		
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "invalid request"})
			return
		}
		
		// Execute Temporal workflow
		wo := client.StartWorkflowOptions{
			ID:       "scenario-" + portfolioID + "-" + req.Scenario,
			TaskQueue: "scenario",
		}
		
		run, err := tc.ExecuteWorkflow(
			context.Background(),
			wo,
			"ScenarioAnalysis",
			workflows.ScenarioAnalysisInput{
				PortfolioID: portfolioID,
				Scenario:    req.Scenario,
			},
		)
		
		if err != nil {
			c.JSON(500, gin.H{"error": "workflow execution failed"})
			return
		}
		
		// Wait for result
		var result map[string]any
		err = run.Get(context.Background(), &result)
		
		if err != nil {
			c.JSON(500, gin.H{"error": "workflow failed"})
			return
		}
		
		c.JSON(200, result)
	})
	
	// GET /api/ai/scenario-proposals
	r.GET("/api/ai/scenario-proposals", func(c *gin.Context) {
		portfolioID := c.Query("portfolio_id")
		tenantID := c.GetHeader("X-Tenant-ID")
		
		// Get AI-generated scenarios
		scenarios := getAIScenarios(portfolioID)
		marketData := getMarketSnapshot()
		
		c.JSON(200, gin.H{
			"scenarios":  scenarios,
			"marketData": marketData,
		})
	})
}

func getAIScenarios(portfolioID string) []map[string]any {
	return []map[string]any{
		{
			"id":          "1",
			"title":       "Impending Interest Rate Hike",
			"description": "Central bank action to curb inflation...",
			"confidence":  92,
			"impact":      "High",
			"category":    "Macro",
		},
		{
			"id":          "2",
			"title":       "Geopolitical Tensions in EMEA",
			"description": "Supply chain disruptions expected...",
			"confidence":  78,
			"impact":      "Medium",
			"category":    "Geopolitical",
		},
		{
			"id":          "3",
			"title":       "Consumer Spending Slowdown",
			"description": "Retail data indicates contraction...",
			"confidence":  65,
			"impact":      "Low",
			"category":    "Economic",
		},
	}
}

func getMarketSnapshot() map[string]any {
	return map[string]any{
		"sp500":               4510.50,
		"sp500Change":        0.5,
		"vix":                15.80,
		"vixChange":          -1.2,
		"treasuryYield":      4.25,
		"treasuryYieldChange": 0.02,
	}
}

// ============================================================================
// 4. DATABASE SCHEMA
// ============================================================================

/*
File: backend/migrations/20240101_scenario_analysis.sql

CREATE TABLE scenario_analyses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    scenario_name VARCHAR(255) NOT NULL,
    base_case JSONB NOT NULL,
    scenario_case JSONB NOT NULL,
    comparison JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (portfolio_id) REFERENCES portfolios(id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX idx_scenario_portfolio ON scenario_analyses(portfolio_id);
CREATE INDEX idx_scenario_tenant ON scenario_analyses(tenant_id);
CREATE INDEX idx_scenario_created ON scenario_analyses(created_at DESC);

-- Create view for easy querying
CREATE VIEW scenario_analysis_results AS
SELECT
    sa.id,
    sa.portfolio_id,
    sa.tenant_id,
    sa.scenario_name,
    sa.base_case ->> 'aum'::text as base_aum,
    sa.base_case ->> 'sharpe'::text as base_sharpe,
    sa.scenario_case ->> 'aum'::text as scenario_aum,
    sa.scenario_case ->> 'sharpe'::text as scenario_sharpe,
    sa.comparison ->> 'aumDifference'::text as aum_diff,
    sa.created_at
FROM scenario_analyses sa;
*/

// ============================================================================
// 5. FRONTEND HOOK - CUSTOM LOGIC
// ============================================================================

// File: frontend/src/hooks/useScenarioAnalysis.ts

import { useState, useCallback } from 'react'
import { useApolloClient } from '@apollo/client'

interface UseScenarioAnalysisOptions {
  portfolioId?: string
  onSuccess?: (result: any) => void
  onError?: (error: any) => void
}

export function useScenarioAnalysis(options: UseScenarioAnalysisOptions = {}) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<any>(null)
  const [result, setResult] = useState<any>(null)
  const apolloClient = useApolloClient()

  const runAnalysis = useCallback(
    async (portfolioId: string, scenario: string) => {
      setLoading(true)
      setError(null)

      try {
        const response = await fetch(`/api/portfolio/${portfolioId}/scenario`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': localStorage.getItem('selected_tenant') || '',
          },
          body: JSON.stringify({ scenario }),
        })

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`)
        }

        const data = await response.json()
        setResult(data)
        options.onSuccess?.(data)

        // Invalidate Apollo cache to refetch portfolios
        apolloClient.cache.evict({ broadcast: false })

        return data
      } catch (err) {
        setError(err)
        options.onError?.(err)
        throw err
      } finally {
        setLoading(false)
      }
    },
    [apolloClient, options],
  )

  const fetchAIScenarios = useCallback(async (portfolioId: string) => {
    try {
      const response = await fetch(`/api/ai/scenario-proposals?portfolio_id=${portfolioId}`, {
        headers: {
          'X-Tenant-ID': localStorage.getItem('selected_tenant') || '',
        },
      })

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`)
      }

      return await response.json()
    } catch (err) {
      setError(err)
      throw err
    }
  }, [])

  return {
    loading,
    error,
    result,
    runAnalysis,
    fetchAIScenarios,
  }
}

// ============================================================================
// 6. FRONTEND USAGE EXAMPLE
// ============================================================================

/*
// In your component:

import ScenarioAnalysisPro from '@/components/ScenarioAnalysisPro'
import { useScenarioAnalysis } from '@/hooks/useScenarioAnalysis'

export function ScenarioPage() {
  const { runAnalysis, loading } = useScenarioAnalysis({
    onSuccess: (result) => {
      console.log('Analysis complete:', result)
      // Show success notification
    },
    onError: (error) => {
      console.error('Analysis failed:', error)
      // Show error notification
    },
  })

  return (
    <div>
      <ScenarioAnalysisPro />
    </div>
  )
}
*/

// ============================================================================
// 7. GRAPHQL SCHEMA ADDITIONS
// ============================================================================

/*
# File: backend/graph/schema.graphql

type Portfolio {
  id: ID!
  aum: Float!
  sharpe: Float!
  risk: Float!
  status: String!
  assetAllocation: [AssetAllocation!]!
}

type AssetAllocation {
  asset: String!
  percentage: Float!
}

type ScenarioAnalysisResult {
  baseCase: ScenarioCase!
  scenarioCase: ScenarioCase!
  comparison: ComparisonMetrics!
}

type ScenarioCase {
  aum: Float!
  sharpe: Float!
  risk: Float!
  status: String!
  assetAllocation: [AssetAllocation!]!
}

type ComparisonMetrics {
  aumDifference: Float!
  sharpeDifference: Float!
  riskDifference: Float!
}

type Subscription {
  portfolios: [Portfolio!]!
  scenarioAnalysis(portfolioId: ID!): ScenarioAnalysisResult!
}
*/

// ============================================================================
// 8. TESTING EXAMPLE
// ============================================================================

/*
// File: frontend/src/components/__tests__/ScenarioAnalysisPro.test.tsx

import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MockedProvider } from '@apollo/client/testing'
import ScenarioAnalysisPro from '../ScenarioAnalysisPro'

describe('ScenarioAnalysisPro', () => {
  it('renders configuration and results panels', () => {
    render(
      <MockedProvider>
        <ScenarioAnalysisPro />
      </MockedProvider>
    )
    
    expect(screen.getByText('Scenario Analysis')).toBeInTheDocument()
    expect(screen.getByLabelText('Select Portfolio')).toBeInTheDocument()
    expect(screen.getByLabelText('Select Scenario')).toBeInTheDocument()
  })

  it('disables Run Analysis button until selections made', () => {
    render(
      <MockedProvider>
        <ScenarioAnalysisPro />
      </MockedProvider>
    )
    
    const button = screen.getByText('Run Analysis')
    expect(button).toBeDisabled()
  })

  it('executes analysis when button clicked', async () => {
    // Test implementation
  })
})
*/
