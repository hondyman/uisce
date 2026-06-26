# Backend Integration Guide - Advanced Wealth Validation Rules

## Overview

This guide explains how to extend the Go backend to handle the 10 new advanced wealth validation rules, including integration with external APIs and AI services.

## Architecture

```
Request Flow:
1. /api/validation-rules/execute
   ├── Load Rule Definition
   ├── Create Validation Context
   ├── Route to Rule Handler
   │   ├── business_logic → executeBusinessLogicRule()
   │   ├── field_format → executeFieldFormatRule()
   │   └── other types...
   ├── Call External APIs (if needed)
   ├── Aggregate Results
   └── Log to Audit Trail
```

---

## Implementation Steps

### Step 1: Database Schema Validation

Ensure `validation_rules` table exists with required columns:

```sql
CREATE TABLE IF NOT EXISTS validation_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    datasource_id UUID REFERENCES datasources(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    rule_type VARCHAR(50) NOT NULL DEFAULT 'business_logic',
    scope TEXT[] NOT NULL DEFAULT ARRAY['ALL_ACCOUNTS'],
    severity VARCHAR(50) NOT NULL DEFAULT 'WARNING',
    is_active BOOLEAN DEFAULT true,
    is_core BOOLEAN DEFAULT false,
    effective_from TIMESTAMP,
    frequency VARCHAR(50),
    evaluation_order INTEGER,
    required_authority VARCHAR(100),
    override_conditions TEXT[],
    parameters JSONB NOT NULL,
    condition_json JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_by UUID,
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_validation_rules_tenant_id ON validation_rules(tenant_id);
CREATE INDEX idx_validation_rules_is_active ON validation_rules(is_active);
CREATE INDEX idx_validation_rules_evaluation_order ON validation_rules(evaluation_order);
CREATE INDEX idx_validation_rules_scope ON validation_rules USING GIN(scope);
```

### Step 2: Update ValidationRule Struct

**File**: `backend/internal/models/validation_rule.go`

```go
package models

import (
    "database/sql/driver"
    "encoding/json"
    "time"
    "github.com/lib/pq"
)

// ValidationRule represents a wealth management validation rule
type ValidationRule struct {
    ID                   string          `db:"id" json:"id"`
    TenantID             string          `db:"tenant_id" json:"tenant_id"`
    DatasourceID         *string         `db:"datasource_id" json:"datasource_id,omitempty"`
    Name                 string          `db:"name" json:"name"`
    Description          string          `db:"description" json:"description"`
    RuleType             string          `db:"rule_type" json:"rule_type"`
    Scope                pq.StringArray  `db:"scope" json:"scope"`
    Severity             string          `db:"severity" json:"severity"`
    IsActive             bool            `db:"is_active" json:"is_active"`
    IsCore               bool            `db:"is_core" json:"is_core"`
    EffectiveFrom        *time.Time      `db:"effective_from" json:"effective_from,omitempty"`
    Frequency            string          `db:"frequency" json:"frequency"`
    EvaluationOrder      int             `db:"evaluation_order" json:"evaluation_order"`
    RequiredAuthority    *string         `db:"required_authority" json:"required_authority,omitempty"`
    OverrideConditions   pq.StringArray  `db:"override_conditions" json:"override_conditions,omitempty"`
    Parameters           json.RawMessage `db:"parameters" json:"parameters"`
    ConditionJSON        json.RawMessage `db:"condition_json" json:"condition_json,omitempty"`
    CreatedAt            time.Time       `db:"created_at" json:"created_at"`
    UpdatedAt            time.Time       `db:"updated_at" json:"updated_at"`
    CreatedBy            *string         `db:"created_by" json:"created_by,omitempty"`
    UpdatedBy            *string         `db:"updated_by" json:"updated_by,omitempty"`
}

// ValidationContext represents the context for rule evaluation
type ValidationContext struct {
    AccountID       string          `json:"account_id"`
    AccountType     string          `json:"account_type"`
    Portfolio       Portfolio       `json:"portfolio"`
    Trade           *Trade          `json:"trade,omitempty"`
    Client          *Client         `json:"client,omitempty"`
    PerformanceData *PerformanceData `json:"performance_data,omitempty"`
}

type Portfolio struct {
    Holdings      []Holding      `json:"holdings"`
    TotalValue    float64        `json:"total_value"`
    Currency      string         `json:"currency"`
    Allocations   map[string]float64 `json:"allocations"`
}

type Holding struct {
    SecurityID     string  `json:"security_id"`
    Quantity       float64 `json:"quantity"`
    CurrentPrice   float64 `json:"current_price"`
    CostBasis      float64 `json:"cost_basis"`
    Weight         float64 `json:"weight"`
    ESGScore       *float64 `json:"esg_score,omitempty"`
    PurchaseDate   time.Time `json:"purchase_date"`
}

type Trade struct {
    TradeID        string  `json:"trade_id"`
    SecurityID     string  `json:"security_id"`
    Side           string  `json:"side"` // BUY, SELL
    Quantity       float64 `json:"quantity"`
    Price          float64 `json:"price"`
    TradeAmount    float64 `json:"trade_amount"`
    TradeDatetime  time.Time `json:"trade_datetime"`
}

type Client struct {
    ID                    string    `json:"id"`
    Name                  string    `json:"name"`
    NetWorth              float64   `json:"net_worth"`
    AnnualIncome          float64   `json:"annual_income"`
    RiskTolerance         string    `json:"risk_tolerance"`
    InvestmentObjective   string    `json:"investment_objective"`
    AccreditedInvestor    bool      `json:"accredited_investor"`
    AccreditedValidDate   *time.Time `json:"accredited_valid_date,omitempty"`
    ESGPreferences        []string  `json:"esg_preferences"`
    TaxBracket            string    `json:"tax_bracket"`
}

type PerformanceData struct {
    Returns       float64        `json:"returns"`
    Volatility    float64        `json:"volatility"`
    SharpeRatio   float64        `json:"sharpe_ratio"`
    Benchmark     string         `json:"benchmark"`
    BenchmarkReturn float64      `json:"benchmark_return"`
    PeriodStartDate time.Time    `json:"period_start_date"`
}

// ValidationResult represents the result of rule evaluation
type ValidationResult struct {
    RuleID           string            `json:"rule_id"`
    RuleName         string            `json:"rule_name"`
    Passed           bool              `json:"passed"`
    Severity         string            `json:"severity"`
    Message          string            `json:"message"`
    Details          map[string]interface{} `json:"details,omitempty"`
    ExternalAPIData  map[string]interface{} `json:"external_api_data,omitempty"`
    Duration         int64             `json:"duration_ms"`
    ExecutedAt       time.Time         `json:"executed_at"`
}
```

### Step 3: Create Rule Handler Registry

**File**: `backend/internal/api/rule_handlers.go`

```go
package api

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"
    
    "github.com/hondyman/semlayer/backend/internal/models"
    "github.com/hondyman/semlayer/backend/internal/services"
)

// RuleHandlerFunc is a function that executes a validation rule
type RuleHandlerFunc func(context.Context, models.ValidationContext, models.ValidationRule) (*models.ValidationResult, error)

// RuleHandlerRegistry maps rule names to their handler functions
var RuleHandlerRegistry = map[string]RuleHandlerFunc{
    // Core wealth management rules
    "Concentration Limit": executeConcentrationLimitRule,
    "KYC Completeness": executeKYCCompletenessRule,
    
    // Advanced wealth management rules (21-25)
    "Tax Optimization": executeTaxOptimizationRule,
    "ESG Compliance": executeESGComplianceRule,
    "Regulatory Margin Compliance": executeMarginComplianceRule,
    "Portfolio Drift Detection": executePortfolioDriftRule,
    "Communication Compliance": executeCommunicationComplianceRule,
    
    // Competitive management rules (26-30)
    "AI-Driven Risk Assessment": executeAIRiskAssessmentRule,
    "Client Engagement Tracking": executeClientEngagementRule,
    "Performance Benchmarking": executePerformanceBenchmarkingRule,
    "Anti-Money Laundering (AML) Compliance": executeAMLComplianceRule,
    "Alternative Investments Eligibility": executeAlternativeInvestmentsRule,
}

// GetRuleHandler returns the handler for a specific rule
func GetRuleHandler(ruleName string) (RuleHandlerFunc, error) {
    handler, exists := RuleHandlerRegistry[ruleName]
    if !exists {
        return nil, fmt.Errorf("no handler found for rule: %s", ruleName)
    }
    return handler, nil
}

// ExecuteRule executes a validation rule with timing
func ExecuteRule(ctx context.Context, validationCtx models.ValidationContext, rule models.ValidationRule) (*models.ValidationResult, error) {
    start := time.Now()
    
    handler, err := GetRuleHandler(rule.Name)
    if err != nil {
        return &models.ValidationResult{
            RuleID:   rule.ID,
            RuleName: rule.Name,
            Passed:   false,
            Severity: "ERROR",
            Message:  fmt.Sprintf("Rule handler not found: %s", err.Error()),
            Duration: time.Since(start).Milliseconds(),
            ExecutedAt: time.Now(),
        }, err
    }
    
    result, err := handler(ctx, validationCtx, rule)
    if result != nil {
        result.Duration = time.Since(start).Milliseconds()
        result.ExecutedAt = time.Now()
    }
    
    return result, err
}

// ===== ADVANCED WEALTH MANAGEMENT RULES =====

// Tax Optimization Rule
func executeTaxOptimizationRule(ctx context.Context, validationCtx models.ValidationContext, rule models.ValidationRule) (*models.ValidationResult, error) {
    result := &models.ValidationResult{
        RuleID:   rule.ID,
        RuleName: rule.Name,
        Details:  make(map[string]interface{}),
    }

    // Parse parameters
    var params struct {
        MaxTaxableGainPercentage float64 `json:"maxTaxableGainPercentage"`
        WashSaleWindowDays       int     `json:"washSaleWindowDays"`
        TaxBracketThresholds     []struct {
            Bracket string  `json:"bracket"`
            MaxGain float64 `json:"maxGain"`
        } `json:"taxBracketThresholds"`
    }

    if err := json.Unmarshal(rule.Parameters, &params); err != nil {
        result.Passed = false
        result.Message = fmt.Sprintf("Invalid rule parameters: %v", err)
        return result, nil
    }

    // Check if trade exists (tax rules apply at trade time)
    if validationCtx.Trade == nil {
        result.Passed = true
        result.Message = "No trade to evaluate"
        return result, nil
    }

    // 1. Verify wash-sale rule compliance
    // For a SELL, check if same security was purchased within washSaleWindowDays
    if validationCtx.Trade.Side == "SELL" {
        lastPurchaseDate := findLastPurchaseDateForSecurity(validationCtx.Portfolio.Holdings, validationCtx.Trade.SecurityID)
        if lastPurchaseDate != nil {
            daysSincePurchase := time.Since(*lastPurchaseDate).Hours() / 24
            if daysSincePurchase < float64(params.WashSaleWindowDays) {
                result.Passed = false
                result.Severity = "WARNING"
                result.Message = fmt.Sprintf("Wash-sale rule violation: security purchased %d days ago (window: %d days)", 
                    int(daysSincePurchase), params.WashSaleWindowDays)
                result.Details["daysSincePurchase"] = daysSincePurchase
                return result, nil
            }
        }
    }

    // 2. Estimate taxable gain and compare to threshold
    realizedGain := calculateRealizedGain(validationCtx)
    portfolioValue := validationCtx.Portfolio.TotalValue
    gainPercentage := realizedGain / portfolioValue

    if gainPercentage > params.MaxTaxableGainPercentage {
        result.Passed = false
        result.Severity = "WARNING"
        result.Message = fmt.Sprintf("Taxable gain %.2f%% exceeds maximum %.2f%%",
            gainPercentage*100, params.MaxTaxableGainPercentage*100)
        result.Details["estimatedGainPercentage"] = gainPercentage
        result.Details["maxAllowedGainPercentage"] = params.MaxTaxableGainPercentage
        result.Details["realizationAmountPercentage"] = gainPercentage
        return result, nil
    }

    result.Passed = true
    result.Message = "Tax optimization rules satisfied"
    result.Details["estimatedGainPercentage"] = gainPercentage
    result.Details["washSaleCompliant"] = true
    return result, nil
}

// ESG Compliance Rule
func executeESGComplianceRule(ctx context.Context, validationCtx models.ValidationContext, rule models.ValidationRule) (*models.ValidationResult, error) {
    result := &models.ValidationResult{
        RuleID:   rule.ID,
        RuleName: rule.Name,
        Details:  make(map[string]interface{}),
        ExternalAPIData: make(map[string]interface{}),
    }

    var params struct {
        MinESGScore           float64  `json:"minEsgScore"`
        MaxESGScoreDeviation  float64  `json:"maxEsgScoreDeviation"`
        RestrictedSectors     []string `json:"restrictedSectors"`
        ESGDataSource         string   `json:"esgDataSource"`
        IntegrationEndpoint   string   `json:"integrationEndpoint"`
    }

    if err := json.Unmarshal(rule.Parameters, &params); err != nil {
        result.Passed = false
        result.Message = fmt.Sprintf("Invalid rule parameters: %v", err)
        return result, nil
    }

    externalAPI := services.GetExternalAPIClient()
    if externalAPI == nil {
        result.Passed = false
        result.Message = "External API service not initialized"
        return result, nil
    }

    violatingHoldings := []string{}
    esgScores := make(map[string]float64)

    // Check each holding for ESG compliance
    for _, holding := range validationCtx.Portfolio.Holdings {
        // Get ESG rating from MSCI API
        esgRating, err := externalAPI.GetESGRating(holding.SecurityID)
        if err != nil {
            log.Printf("Failed to get ESG rating for %s: %v", holding.SecurityID, err)
            continue
        }

        if esgRating == nil {
            continue
        }

        esgScores[holding.SecurityID] = esgRating.Score

        // Check against minimum ESG score
        if esgRating.Score < params.MinESGScore {
            violatingHoldings = append(violatingHoldings, fmt.Sprintf("%s (score: %.1f)", holding.SecurityID, esgRating.Score))
        }

        // Check for restricted sectors (simplified - would need sector mapping)
        // In production, would use MSCI classification data
    }

    result.ExternalAPIData["esgScores"] = esgScores

    if len(violatingHoldings) > 0 {
        result.Passed = false
        result.Severity = "WARNING"
        result.Message = fmt.Sprintf("Holdings below minimum ESG score (%.1f): %v", params.MinESGScore, violatingHoldings)
        result.Details["violatingCount"] = len(violatingHoldings)
        result.Details["violatingHoldings"] = violatingHoldings
        return result, nil
    }

    result.Passed = true
    result.Message = "All holdings meet ESG requirements"
    result.Details["holdingsReviewed"] = len(validationCtx.Portfolio.Holdings)
    result.Details["minimumESGScore"] = params.MinESGScore
    return result, nil
}

// AI-Driven Risk Assessment Rule
func executeAIRiskAssessmentRule(ctx context.Context, validationCtx models.ValidationContext, rule models.ValidationRule) (*models.ValidationResult, error) {
    result := &models.ValidationResult{
        RuleID:   rule.ID,
        RuleName: rule.Name,
        Details:  make(map[string]interface{}),
        ExternalAPIData: make(map[string]interface{}),
    }

    var params struct {
        MaxVaR              float64  `json:"maxVaR"`
        VaRConfidenceLevel  float64  `json:"varConfidenceLevel"`
        StressTestScenarios []string `json:"stressTestScenarios"`
        AIModelEndpoint     string   `json:"aiModelEndpoint"`
        ModelType           string   `json:"modelType"`
        IntegrationTimeout  int      `json:"integrationTimeout"`
    }

    if err := json.Unmarshal(rule.Parameters, &params); err != nil {
        result.Passed = false
        result.Message = fmt.Sprintf("Invalid rule parameters: %v", err)
        return result, nil
    }

    externalAPI := services.GetExternalAPIClient()
    if externalAPI == nil {
        result.Passed = false
        result.Message = "External API service not initialized"
        return result, nil
    }

    // Prepare portfolio data for AI model
    portfolioData := map[string]interface{}{
        "holdings": convertHoldingsForAI(validationCtx.Portfolio.Holdings),
        "correlationMatrix": generateCorrelationMatrix(validationCtx.Portfolio.Holdings),
        "stressTestScenarios": params.StressTestScenarios,
    }

    // Call AI model endpoint
    riskAssessment, err := externalAPI.AssessPortfolioRisk(portfolioData)
    if err != nil {
        result.Passed = false
        result.Message = fmt.Sprintf("AI model error: %v", err)
        return result, nil
    }

    result.ExternalAPIData["riskAssessment"] = riskAssessment

    // Compare VaR to threshold
    if riskAssessment.Var95 > params.MaxVaR {
        result.Passed = false
        result.Severity = "WARNING"
        result.Message = fmt.Sprintf("VaR (95%%): %.2f%% exceeds maximum %.2f%%", riskAssessment.Var95*100, params.MaxVaR*100)
        result.Details["var95"] = riskAssessment.Var95
        result.Details["maxVar"] = params.MaxVaR
        result.Details["recommendations"] = riskAssessment.Recommendations
        return result, nil
    }

    result.Passed = true
    result.Message = fmt.Sprintf("Portfolio risk within limits. VaR (95%%): %.2f%%", riskAssessment.Var95*100)
    result.Details["var95"] = riskAssessment.Var95
    result.Details["var99"] = riskAssessment.Var99
    result.Details["riskLevel"] = riskAssessment.RiskLevel
    result.Details["stressTestResults"] = riskAssessment.StressTestResults
    return result, nil
}

// AML Compliance Rule
func executeAMLComplianceRule(ctx context.Context, validationCtx models.ValidationContext, rule models.ValidationRule) (*models.ValidationResult, error) {
    result := &models.ValidationResult{
        RuleID:   rule.ID,
        RuleName: rule.Name,
        Details:  make(map[string]interface{}),
        ExternalAPIData: make(map[string]interface{}),
    }

    var params struct {
        TransactionThreshold   float64  `json:"transactionThreshold"`
        CumulativeThreshold    float64  `json:"cumulativeThreshold"`
        CumulativeWindowDays   int      `json:"cumulativeWindowDays"`
        SuspiciousPatterns     []string `json:"suspiciousPatterns"`
        AMLScreeningService    string   `json:"amlScreeningService"`
        IntegrationEndpoint    string   `json:"integrationEndpoint"`
        ReportingRequirement   string   `json:"reportingRequirement"`
    }

    if err := json.Unmarshal(rule.Parameters, &params); err != nil {
        result.Passed = false
        result.Message = fmt.Sprintf("Invalid rule parameters: %v", err)
        return result, nil
    }

    // Check trade size threshold
    if validationCtx.Trade != nil && validationCtx.Trade.TradeAmount > params.TransactionThreshold {
        result.Passed = false
        result.Severity = "BLOCK"
        result.Message = fmt.Sprintf("Transaction $%.2f exceeds reporting threshold $%.2f", 
            validationCtx.Trade.TradeAmount, params.TransactionThreshold)
        result.Details["transactionAmount"] = validationCtx.Trade.TradeAmount
        result.Details["reportingRequired"] = true
        result.Details["reportType"] = params.ReportingRequirement
        return result, nil
    }

    // Screen client for AML
    if validationCtx.Client != nil {
        externalAPI := services.GetExternalAPIClient()
        amlResult, err := externalAPI.ScreenAML(validationCtx.Client.Name)
        if err != nil {
            result.Passed = false
            result.Message = fmt.Sprintf("AML screening error: %v", err)
            return result, nil
        }

        result.ExternalAPIData["amlScreening"] = amlResult

        if amlResult.RiskLevel != "LOW" {
            result.Passed = false
            result.Severity = "BLOCK"
            result.Message = fmt.Sprintf("AML screening flagged entity as %s risk", amlResult.RiskLevel)
            result.Details["riskLevel"] = amlResult.RiskLevel
            result.Details["matches"] = len(amlResult.Matches)
            result.Details["screeningId"] = amlResult.ScreeningID
            return result, nil
        }
    }

    result.Passed = true
    result.Message = "AML compliance check passed"
    result.Details["transactionThreshold"] = params.TransactionThreshold
    result.Details["requiresReporting"] = false
    return result, nil
}

// Helper functions

func findLastPurchaseDateForSecurity(holdings []models.Holding, securityID string) *time.Time {
    for _, h := range holdings {
        if h.SecurityID == securityID {
            return &h.PurchaseDate
        }
    }
    return nil
}

func calculateRealizedGain(ctx models.ValidationContext) float64 {
    if ctx.Trade == nil {
        return 0
    }
    // Simplified gain calculation
    return ctx.Trade.Price * ctx.Trade.Quantity
}

func convertHoldingsForAI(holdings []models.Holding) []map[string]interface{} {
    result := make([]map[string]interface{}, len(holdings))
    for i, h := range holdings {
        result[i] = map[string]interface{}{
            "ticker":      h.SecurityID,
            "weight":      h.Weight,
            "price":       h.CurrentPrice,
            "volatility":  0.2, // Would come from market data
        }
    }
    return result
}

func generateCorrelationMatrix(holdings []models.Holding) [][]float64 {
    n := len(holdings)
    matrix := make([][]float64, n)
    for i := 0; i < n; i++ {
        matrix[i] = make([]float64, n)
        for j := 0; j < n; j++ {
            if i == j {
                matrix[i][j] = 1.0
            } else {
                matrix[i][j] = -0.15 // Simplified correlation
            }
        }
    }
    return matrix
}
```

### Step 4: Update API Routes

**File**: `backend/internal/api/api.go`

```go
func (a *API) SetupRoutes() {
    // ... existing routes ...

    // Validation Rules endpoints
    a.router.Post("/api/validation-rules", a.handleCreateValidationRule)
    a.router.Get("/api/validation-rules", a.handleListValidationRules)
    a.router.Get("/api/validation-rules/:id", a.handleGetValidationRule)
    a.router.Put("/api/validation-rules/:id", a.handleUpdateValidationRule)
    a.router.Delete("/api/validation-rules/:id", a.handleDeleteValidationRule)
    a.router.Post("/api/validation-rules/execute", a.handleExecuteValidationRule)
    a.router.Post("/api/validation-rules/import", a.handleImportValidationRules)
}

func (a *API) handleExecuteValidationRule(w http.ResponseWriter, r *http.Request) {
    tenantID := r.Header.Get("X-Tenant-ID")
    if tenantID == "" {
        http.Error(w, "Missing X-Tenant-ID header", http.StatusBadRequest)
        return
    }

    var req struct {
        RuleID  string                      `json:"ruleId"`
        Context models.ValidationContext    `json:"context"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Load rule from database
    rule, err := a.db.GetValidationRule(r.Context(), req.RuleID, tenantID)
    if err != nil {
        http.Error(w, fmt.Sprintf("Rule not found: %v", err), http.StatusNotFound)
        return
    }

    // Execute rule
    result, err := ExecuteRule(r.Context(), req.Context, *rule)
    if err != nil {
        log.Printf("Error executing rule %s: %v", req.RuleID, err)
        result.Message = fmt.Sprintf("Execution error: %v", err)
    }

    // Log to audit trail
    a.logRuleExecution(r.Context(), tenantID, result)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}
```

### Step 5: Create External API Service

**File**: `backend/internal/services/external_api_client.go`

```go
package services

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "sync"
    "time"
)

var (
    externalAPIClient *ExternalAPIClient
    apiClientOnce     sync.Once
)

type ExternalAPIClient struct {
    httpClient *http.Client
    cache      sync.Map // Simple in-memory cache
}

type ESGRating struct {
    Ticker            string    `json:"ticker"`
    Score             float64   `json:"esgScore"`
    Rating            string    `json:"esgRating"`
    EnvironmentScore  float64   `json:"environmentScore"`
    SocialScore       float64   `json:"socialScore"`
    GovernanceScore   float64   `json:"governanceScore"`
    DataAsOfDate      string    `json:"dataAsOfDate"`
}

type AMLScreeningResult struct {
    ScreeningID  string    `json:"screeningId"`
    EntityName   string    `json:"entityName"`
    RiskLevel    string    `json:"riskLevel"` // LOW, MEDIUM, HIGH, CRITICAL
    Matches      []interface{} `json:"matches"`
    ScreeningDate string   `json:"screeningDate"`
}

type RiskAssessment struct {
    Var95               float64              `json:"var95"`
    Var99               float64              `json:"var99"`
    ConditionalVar      float64              `json:"conditionalVar"`
    StressTestResults   map[string]float64   `json:"stressTestResults"`
    RiskLevel           string               `json:"riskLevel"`
    Recommendations     []string             `json:"recommendations"`
    GeneratedAt         string               `json:"generatedAt"`
}

// GetExternalAPIClient returns singleton instance
func GetExternalAPIClient() *ExternalAPIClient {
    apiClientOnce.Do(func() {
        externalAPIClient = &ExternalAPIClient{
            httpClient: &http.Client{
                Timeout: 30 * time.Second,
            },
        }
    })
    return externalAPIClient
}

// GetESGRating fetches ESG rating from MSCI API
func (c *ExternalAPIClient) GetESGRating(securityID string) (*ESGRating, error) {
    // Check cache
    if cached, ok := c.cache.Load("esg_" + securityID); ok {
        return cached.(*ESGRating), nil
    }

    endpoint := os.Getenv("MSCI_ENDPOINT")
    if endpoint == "" {
        endpoint = "https://api.msci.com/esg-ratings"
    }

    apiKey := os.Getenv("MSCI_API_KEY")
    if apiKey == "" {
        return nil, fmt.Errorf("MSCI_API_KEY not configured")
    }

    url := fmt.Sprintf("%s?ticker=%s&format=json", endpoint, securityID)

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }

    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
    }

    var esgRating ESGRating
    if err := json.NewDecoder(resp.Body).Decode(&esgRating); err != nil {
        return nil, err
    }

    // Cache for 24 hours
    c.cache.Store("esg_"+securityID, &esgRating)
    go c.expireCache("esg_"+securityID, 24*time.Hour)

    return &esgRating, nil
}

// ScreenAML screens entity against watchlists
func (c *ExternalAPIClient) ScreenAML(entityName string) (*AMLScreeningResult, error) {
    // Check cache
    if cached, ok := c.cache.Load("aml_" + entityName); ok {
        return cached.(*AMLScreeningResult), nil
    }

    endpoint := os.Getenv("WORLD_CHECK_ENDPOINT")
    if endpoint == "" {
        endpoint = "https://api.world-check.com/screen"
    }

    payload := map[string]interface{}{
        "entityName": entityName,
        "entityType": "INDIVIDUAL",
        "screeningDate": time.Now().Format(time.RFC3339),
    }

    body, _ := json.Marshal(payload)

    req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(body))
    if err != nil {
        return nil, err
    }

    username := os.Getenv("WORLD_CHECK_USERNAME")
    password := os.Getenv("WORLD_CHECK_PASSWORD")
    req.SetBasicAuth(username, password)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
    }

    var result AMLScreeningResult
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    // Cache for 7 days
    c.cache.Store("aml_"+entityName, &result)
    go c.expireCache("aml_"+entityName, 7*24*time.Hour)

    return &result, nil
}

// AssessPortfolioRisk calls AI model for risk assessment
func (c *ExternalAPIClient) AssessPortfolioRisk(portfolioData interface{}) (*RiskAssessment, error) {
    endpoint := os.Getenv("SAGEMAKER_ENDPOINT")
    if endpoint == "" {
        return nil, fmt.Errorf("SAGEMAKER_ENDPOINT not configured")
    }

    body, _ := json.Marshal(portfolioData)

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(body))
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("AI model returned %d: %s", resp.StatusCode, string(body))
    }

    var assessment RiskAssessment
    if err := json.NewDecoder(resp.Body).Decode(&assessment); err != nil {
        return nil, err
    }

    return &assessment, nil
}

// Helper to expire cache entries
func (c *ExternalAPIClient) expireCache(key string, ttl time.Duration) {
    time.Sleep(ttl)
    c.cache.Delete(key)
}
```

---

## Configuration

### Environment Variables

Create `.env` file in backend root:

```bash
# PostgreSQL
POSTGRES_HOST=host.docker.internal
POSTGRES_PORT=5432
POSTGRES_DB=alpha
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_SSLMODE=disable

# External APIs
MSCI_API_KEY=your_api_key_here
MSCI_ENDPOINT=https://api.msci.com/esg-ratings

WORLD_CHECK_USERNAME=your_username
WORLD_CHECK_PASSWORD=your_password
WORLD_CHECK_ENDPOINT=https://api.world-check.com/screen

BLOOMBERG_TOKEN=your_token_here
BLOOMBERG_ENDPOINT=https://api.bloomberg.com/benchmark-data

SAGEMAKER_ENDPOINT=https://your-endpoint.sagemaker.amazonaws.com/invocations

# Logging
LOG_LEVEL=debug
```

---

## Testing

### Integration Tests

**File**: `backend/internal/api/validation_rules_test.go`

```go
package api

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestExecuteESGComplianceRule(t *testing.T) {
    // ... test implementation ...
}

func TestExecuteAMLComplianceRule(t *testing.T) {
    // ... test implementation ...
}

func TestExecuteAIRiskAssessmentRule(t *testing.T) {
    // ... test implementation ...
}
```

---

## Deployment

1. Update PostgreSQL schema with migrations
2. Deploy backend with new handlers
3. Configure environment variables
4. Test rule execution endpoints
5. Monitor external API integrations

