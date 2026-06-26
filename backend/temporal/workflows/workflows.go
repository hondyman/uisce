package workflows

import (
	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/temporal/activities"
	"go.temporal.io/sdk/workflow"
)

// ScenarioAnalysisWorkflow executes portfolio scenario analysis with AI optimization
func ScenarioAnalysisWorkflow(ctx workflow.Context, portfolioID string, scenarioType string) (map[string]interface{}, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})

	// 1. Fetch portfolio data
	var portfolioData map[string]interface{}
	err := workflow.ExecuteActivity(ctx, activities.FetchPortfolioData, portfolioID).Get(ctx, &portfolioData)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch portfolio data: %w", err)
	}

	// 2. Project scenario
	var scenarioResult map[string]interface{}
	err = workflow.ExecuteActivity(ctx, "ProjectScenario", portfolioData, scenarioType).Get(ctx, &scenarioResult)
	if err != nil {
		return nil, fmt.Errorf("failed to project scenario: %w", err)
	}

	// 3. Calculate comparison
	var comparison map[string]interface{}
	err = workflow.ExecuteActivity(ctx, activities.CalculateComparison, portfolioData, scenarioResult).Get(ctx, &comparison)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate comparison: %w", err)
	}

	// 4. Store result
	result := map[string]interface{}{
		"baseCase":     portfolioData,
		"scenarioCase": scenarioResult,
		"comparison":   comparison,
		"executedAt":   time.Now(),
		"scenarioType": scenarioType,
	}

	err = workflow.ExecuteActivity(ctx, activities.HasuraUpdate, result).Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to store result: %w", err)
	}

	return result, nil
}

// UMAAlpha executes the killer UMA rebalance workflow with AI tax harvesting
func UMAAlpha(ctx workflow.Context, umaID string) (map[string]interface{}, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
	})

	// 1. Fetch UMA data
	var umaData map[string]interface{}
	err := workflow.ExecuteActivity(ctx, activities.FetchUMAData, umaID).Get(ctx, &umaData)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch UMA data: %w", err)
	}

	// 2. AI Tax Harvest Analysis
	var harvestPlan map[string]interface{}
	err = workflow.ExecuteActivity(ctx, activities.AITaxHarvest, umaData).Get(ctx, &harvestPlan)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze tax harvest: %w", err)
	}

	// 3. ABAC Check
	var allowed bool
	err = workflow.ExecuteActivity(ctx, activities.ABACCheck, "rebalance", "uma", umaID).Get(ctx, &allowed)
	if err != nil {
		return nil, fmt.Errorf("ABAC check failed: %w", err)
	}
	if !allowed {
		return nil, fmt.Errorf("ABAC denied rebalance for UMA %s", umaID)
	}

	// 4. Execute trades
	var trades map[string]interface{}
	err = workflow.ExecuteActivity(ctx, activities.ExecuteTrades, harvestPlan).Get(ctx, &trades)
	if err != nil {
		return nil, fmt.Errorf("failed to execute trades: %w", err)
	}

	// 5. Update Hasura
	result := map[string]interface{}{
		"umaID":       umaID,
		"status":      "alpha_rebalanced",
		"harvestPlan": harvestPlan,
		"trades":      trades,
		"taxSaved":    harvestPlan["saved"],
		"executedAt":  time.Now(),
	}

	err = workflow.ExecuteActivity(ctx, activities.HasuraUpdate, result).Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to update Hasura: %w", err)
	}

	return result, nil
}

// TaxHarvest executes the AI-powered tax optimization workflow for UMA accounts
func TaxHarvest(ctx workflow.Context, umaID string) (map[string]interface{}, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 60 * time.Second,
	})

	// 1. Fetch UMA data
	var umaData map[string]interface{}
	err := workflow.ExecuteActivity(ctx, activities.FetchUMAData, umaID).Get(ctx, &umaData)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch UMA data: %w", err)
	}

	// 2. AI Tax Harvest Analysis
	var harvest map[string]interface{}
	err = workflow.ExecuteActivity(ctx, activities.AITaxHarvest, umaData).Get(ctx, &harvest)
	if err != nil {
		return nil, fmt.Errorf("AI tax harvest failed: %w", err)
	}

	// 3. ABAC Check
	var allowed bool
	err = workflow.ExecuteActivity(ctx, activities.ABACCheck, "harvest", "uma", umaID).Get(ctx, &allowed)
	if err != nil {
		return nil, fmt.Errorf("ABAC check failed: %w", err)
	}
	if !allowed {
		return nil, fmt.Errorf("ABAC denied tax harvest for UMA %s", umaID)
	}

	// 4. Execute tax harvest
	var executed map[string]interface{}
	err = workflow.ExecuteActivity(ctx, activities.ExecuteHarvest, harvest).Get(ctx, &executed)
	if err != nil {
		return nil, fmt.Errorf("tax harvest execution failed: %w", err)
	}

	// 5. Update Hasura
	result := map[string]interface{}{
		"umaID":        umaID,
		"status":       "tax_optimized",
		"taxSaved":     harvest["saved"],
		"lotsSelected": harvest["lots"],
		"executedAt":   time.Now(),
	}

	err = workflow.ExecuteActivity(ctx, activities.HasuraUpdate, result).Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("Hasura update failed: %w", err)
	}

	return result, nil
}

// IndexAlpha executes the AI-powered direct indexing optimization workflow
func IndexAlpha(ctx workflow.Context, indexID string) (map[string]interface{}, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
	})

	// 1. Fetch index data
	var indexData map[string]interface{}
	err := workflow.ExecuteActivity(ctx, activities.FetchIndexData, indexID).Get(ctx, &indexData)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch index data: %w", err)
	}

	// 2. AI Index Optimization Analysis
	var optimization map[string]interface{}
	err = workflow.ExecuteActivity(ctx, activities.AIIndexOptimize, indexData).Get(ctx, &optimization)
	if err != nil {
		return nil, fmt.Errorf("AI index optimization failed: %w", err)
	}

	// 3. ABAC Check
	var allowed bool
	err = workflow.ExecuteActivity(ctx, activities.ABACCheck, "optimize", "index", indexID).Get(ctx, &allowed)
	if err != nil {
		return nil, fmt.Errorf("ABAC check failed: %w", err)
	}
	if !allowed {
		return nil, fmt.Errorf("ABAC denied index optimization for %s", indexID)
	}

	// 4. Execute optimization trades
	var trades map[string]interface{}
	err = workflow.ExecuteActivity(ctx, activities.ExecuteTrades, optimization).Get(ctx, &trades)
	if err != nil {
		return nil, fmt.Errorf("index optimization execution failed: %w", err)
	}

	// 5. Update Hasura
	result := map[string]interface{}{
		"indexID":    indexID,
		"status":     "alpha_optimized",
		"drift":      optimization["drift"],
		"taxSaved":   optimization["saved"],
		"esgScore":   optimization["esg_score"],
		"trades":     trades,
		"executedAt": time.Now(),
	}

	err = workflow.ExecuteActivity(ctx, activities.HasuraUpdate, result).Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("Hasura update failed: %w", err)
	}

	return result, nil
}

// AttributionAlpha executes the killer performance attribution workflow with AI
func AttributionAlpha(ctx workflow.Context, portfolioID string) (map[string]interface{}, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})

	// 1. Fetch portfolio data
	var portfolioData map[string]interface{}
	err := workflow.ExecuteActivity(ctx, activities.FetchPortfolioData, portfolioID).Get(ctx, &portfolioData)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch portfolio data: %w", err)
	}

	// 2. AI Attribution Analysis
	var attribution map[string]interface{}
	err = workflow.ExecuteActivity(ctx, activities.AIAttribution, portfolioData).Get(ctx, &attribution)
	if err != nil {
		return nil, fmt.Errorf("AI attribution failed: %w", err)
	}

	// 3. ABAC Check
	var allowed bool
	err = workflow.ExecuteActivity(ctx, activities.ABACCheck, "attribute", "portfolio", portfolioID).Get(ctx, &allowed)
	if err != nil {
		return nil, fmt.Errorf("ABAC check failed: %w", err)
	}
	if !allowed {
		return nil, fmt.Errorf("ABAC denied attribution for portfolio %s", portfolioID)
	}

	// 4. Store result
	result := map[string]interface{}{
		"portfolioID": portfolioID,
		"attribution": attribution,
		"alpha":       attribution["alpha"],
		"sector":      attribution["sector"],
		"executedAt":  time.Now(),
	}

	err = workflow.ExecuteActivity(ctx, activities.HasuraUpdate, result).Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to store result: %w", err)
	}

	return result, nil
}
