# Advanced Wealth Management Rules - Backend Integration Guide

This guide provides detailed instructions for integrating the new advanced wealth management rules into the backend rule execution engine.

## Overview

The backend must be enhanced to:
1. **Execute New Business Logic Rules** (21-25, 26-28, 30)
2. **Integrate External APIs** (ESG, AML, Bloomberg, SageMaker)
3. **Handle Async Operations** (API calls, ML inference)
4. **Validate Parameters** (type checking, range validation)
5. **Log Execution** (audit trail, debugging)

## Backend Architecture

### Rule Execution Flow

```
Request: /api/validation-rules/{rule_id}/execute
    ↓
Parse Rule Parameters (from JSONB condition_json)
    ↓
Route to Handler (by rule_type and rule_id)
    ↓
Execute Business Logic / Call External API
    ↓
Validate Result against Parameters
    ↓
Return Status (PASS/WARN/BLOCK)
    ↓
Store Audit Log
    ↓
Response: ExecutionResult
```

## Implementation Details

### 1. Update `validation_rules_routes.go`

**Add a new handler for advanced rules**:

```go
package api

import (
    "context"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "time"

    "github.com/your-org/semlayer/backend/internal/services"
)

// ExecutionResult represents the result of a validation rule execution
type ExecutionResult struct {
    RuleID           string                 `json:"rule_id"`
    Status           string                 `json:"status"` // PASS, WARN, BLOCK
    Severity         string                 `json:"severity"`
    Message          string                 `json:"message"`
    Details          map[string]interface{} `json:"details"`
    ExecutionTime    int64                  `json:"execution_time_ms"`
    ExternalAPICall  bool                   `json:"external_api_call"`
    Timestamp        time.Time              `json:"timestamp"`
}

// HandleExecuteAdvancedValidationRule executes advanced wealth management rules
func HandleExecuteAdvancedValidationRule(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    ruleID := chi.URLParam(r, "ruleID")
    
    var req struct {
        AccountID      string      `json:"account_id"`
        AccountType    string      `json:"account_type"`
        Context        interface{} `json:"context"` // Validation context (positions, trades, etc)
        Parameters     interface{} `json:"parameters"` // Override parameters
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    // Retrieve rule from database
    rule, err := services.GetValidationRule(ctx, ruleID)
    if err != nil {
        http.Error(w, fmt.Sprintf("Rule not found: %v", err), http.StatusNotFound)
        return
    }
    
    // Start timing
    startTime := time.Now()
    
    // Execute rule based on ID
    var result *ExecutionResult
    switch ruleID {
    case "tax-optimization-v1":
        result = executeTaxOptimization(ctx, rule, req.Context, req.Parameters)
    case "esg-compliance-v1":
        result = executeESGCompliance(ctx, rule, req.Context, req.Parameters)
    case "margin-compliance-v1":
        result = executeMarginCompliance(ctx, rule, req.Context, req.Parameters)
    case "portfolio-drift-v1":
        result = executePortfolioDrift(ctx, rule, req.Context, req.Parameters)
    case "communication-compliance-v1":
        result = executeCommunicationCompliance(ctx, rule, req.Context, req.Parameters)
    case "ai-risk-assessment-v1":
        result = executeAIRiskAssessment(ctx, rule, req.Context, req.Parameters)
    case "client-engagement-v1":
        result = executeClientEngagement(ctx, rule, req.Context, req.Parameters)
    case "performance-benchmarking-v1":
        result = executePerformanceBenchmarking(ctx, rule, req.Context, req.Parameters)
    case "aml-compliance-v1":
        result = executeAMLCompliance(ctx, rule, req.Context, req.Parameters)
    case "alternative-investments-v1":
        result = executeAlternativeInvestments(ctx, rule, req.Context, req.Parameters)
    default:
        http.Error(w, "Unknown rule", http.StatusBadRequest)
        return
    }
    
    result.ExecutionTime = time.Since(startTime).Milliseconds()
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}
```

### 2. Implement Rule Handlers

#### Tax Optimization Rule (21)

```go
func executeTaxOptimization(ctx context.Context, rule *Rule, evalCtx interface{}, params interface{}) *ExecutionResult {
    result := &ExecutionResult{
        RuleID:    "tax-optimization-v1",
        Severity:  rule.Severity,
        Timestamp: time.Now(),
        Details:   make(map[string]interface{}),
    }
    
    // Parse parameters
    params := rule.ConditionJSON
    maxTaxableGain := params["maxTaxableGainPercentage"].(float64)
    washSaleWindow := int(params["washSaleWindowDays"].(float64))
    
    // Extract trades from context
    trades := evalCtx.(map[string]interface{})["trades"].([]interface{})
    
    // Calculate realized gains
    totalGain := 0.0
    gainCount := 0
    washSaleViolations := 0
    
    for _, tradeRaw := range trades {
        trade := tradeRaw.(map[string]interface{})
        
        // Check for taxable gains
        if gain, ok := trade["realizedGain"].(float64); ok && gain > 0 {
            totalGain += gain
            gainCount++
        }
        
        // Check for wash-sale violations
        if hasWashSaleViolation(ctx, trade, washSaleWindow) {
            washSaleViolations++
        }
    }
    
    avgGainPercentage := 0.0
    if gainCount > 0 {
        avgGainPercentage = totalGain / float64(gainCount)
    }
    
    result.Details["total_taxable_gain"] = totalGain
    result.Details["average_gain_percentage"] = avgGainPercentage
    result.Details["wash_sale_violations"] = washSaleViolations
    
    // Determine pass/fail
    if washSaleViolations > 0 {
        result.Status = "BLOCK"
        result.Message = fmt.Sprintf("Wash-sale rule violations detected: %d trades", washSaleViolations)
        return result
    }
    
    if avgGainPercentage > maxTaxableGain {
        result.Status = "WARN"
        result.Message = fmt.Sprintf("Avg gain (%.2f%%) exceeds threshold (%.2f%%)", 
            avgGainPercentage*100, maxTaxableGain*100)
    } else {
        result.Status = "PASS"
        result.Message = "Trades comply with tax optimization rules"
    }
    
    return result
}

// Helper to detect wash-sale violations
func hasWashSaleViolation(ctx context.Context, trade map[string]interface{}, windowDays int) bool {
    symbol := trade["symbol"].(string)
    saleDate := trade["date"].(time.Time)
    
    // Query for repurchases within window
    repurchases, err := services.GetRepurchases(ctx, symbol, saleDate.AddDate(0, 0, -windowDays), saleDate.AddDate(0, 0, windowDays))
    
    return err == nil && len(repurchases) > 0
}
```

#### ESG Compliance Rule (22)

```go
func executeESGCompliance(ctx context.Context, rule *Rule, evalCtx interface{}, params interface{}) *ExecutionResult {
    result := &ExecutionResult{
        RuleID:          "esg-compliance-v1",
        Severity:        rule.Severity,
        Timestamp:       time.Now(),
        Details:         make(map[string]interface{}),
        ExternalAPICall: true,
    }
    
    // Parse parameters
    params := rule.ConditionJSON
    minEsgScore := params["minEsgScore"].(float64)
    restrictedSectors := params["restrictedSectors"].([]interface{})
    esgEndpoint := params["integrationEndpoint"].(string)
    
    // Extract holdings from context
    holdings := evalCtx.(map[string]interface{})["holdings"].([]interface{})
    
    lowESGHoldings := []map[string]interface{}{}
    restrictedHoldings := []map[string]interface{}{}
    portfolioESGScore := 0.0
    holdingCount := 0
    
    for _, holdingRaw := range holdings {
        holding := holdingRaw.(map[string]interface{})
        symbol := holding["symbol"].(string)
        sector := holding["sector"].(string)
        
        // Check restricted sectors
        for _, restricted := range restrictedSectors {
            if sector == restricted.(string) {
                restrictedHoldings = append(restrictedHoldings, holding)
            }
        }
        
        // Fetch ESG score from external API
        esgScore, err := fetchESGScore(ctx, symbol, esgEndpoint)
        if err != nil {
            result.Status = "WARN"
            result.Message = fmt.Sprintf("Failed to fetch ESG data for %s: %v", symbol, err)
            return result
        }
        
        if esgScore < minEsgScore {
            holding["esg_score"] = esgScore
            lowESGHoldings = append(lowESGHoldings, holding)
        }
        
        portfolioESGScore += esgScore
        holdingCount++
    }
    
    if holdingCount > 0 {
        portfolioESGScore /= float64(holdingCount)
    }
    
    result.Details["portfolio_esg_score"] = portfolioESGScore
    result.Details["low_esg_holdings"] = len(lowESGHoldings)
    result.Details["restricted_holdings"] = len(restrictedHoldings)
    
    if len(restrictedHoldings) > 0 {
        result.Status = "WARN"
        result.Message = fmt.Sprintf("Portfolio contains %d holdings from restricted sectors", len(restrictedHoldings))
    } else if len(lowESGHoldings) > 0 {
        result.Status = "WARN"
        result.Message = fmt.Sprintf("Portfolio contains %d holdings below ESG threshold", len(lowESGHoldings))
    } else {
        result.Status = "PASS"
        result.Message = fmt.Sprintf("Portfolio ESG score: %.2f (above %.2f threshold)", portfolioESGScore, minEsgScore)
    }
    
    return result
}

// Helper to fetch ESG score from MSCI or similar service
func fetchESGScore(ctx context.Context, symbol string, endpoint string) (float64, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", 
        fmt.Sprintf("%s?symbol=%s", endpoint, symbol), nil)
    if err != nil {
        return 0, err
    }
    
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()
    
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return 0, err
    }
    
    var data struct {
        Score float64 `json:"esg_score"`
    }
    if err := json.Unmarshal(body, &data); err != nil {
        return 0, err
    }
    
    return data.Score, nil
}
```

#### Regulatory Margin Compliance Rule (23)

```go
func executeMarginCompliance(ctx context.Context, rule *Rule, evalCtx interface{}, params interface{}) *ExecutionResult {
    result := &ExecutionResult{
        RuleID:    "margin-compliance-v1",
        Severity:  rule.Severity,
        Timestamp: time.Now(),
        Details:   make(map[string]interface{}),
    }
    
    params := rule.ConditionJSON
    initialMarginLimit := params["initialMarginLimit"].(float64)
    maintenanceMarginLimit := params["maintenanceMarginLimit"].(float64)
    maxLoanValue := params["maxLoanValue"].(float64)
    marginCallThreshold := params["marginCallThreshold"].(float64)
    
    account := evalCtx.(map[string]interface{})
    equity := account["equity"].(float64)
    loanValue := account["loan_value"].(float64)
    positions := account["positions"].([]interface{})
    
    // Calculate total portfolio value
    portfolioValue := 0.0
    for _, posRaw := range positions {
        pos := posRaw.(map[string]interface{})
        portfolioValue += pos["market_value"].(float64)
    }
    
    // Calculate margin ratios
    initialMarginUsed := loanValue / portfolioValue
    maintenanceMarginCurrent := equity / portfolioValue
    
    result.Details["portfolio_value"] = portfolioValue
    result.Details["loan_value"] = loanValue
    result.Details["equity"] = equity
    result.Details["initial_margin_used"] = initialMarginUsed
    result.Details["maintenance_margin_current"] = maintenanceMarginCurrent
    
    violations := []string{}
    
    // Check violations
    if initialMarginUsed > initialMarginLimit {
        violations = append(violations, fmt.Sprintf("Initial margin exceeded: %.2f%% > %.2f%%", 
            initialMarginUsed*100, initialMarginLimit*100))
    }
    
    if maintenanceMarginCurrent < maintenanceMarginLimit {
        violations = append(violations, fmt.Sprintf("Maintenance margin below limit: %.2f%% < %.2f%%", 
            maintenanceMarginCurrent*100, maintenanceMarginLimit*100))
    }
    
    if loanValue > maxLoanValue {
        violations = append(violations, fmt.Sprintf("Loan value exceeds max: $%.0f > $%.0f", 
            loanValue, maxLoanValue))
    }
    
    if maintenanceMarginCurrent < marginCallThreshold {
        violations = append(violations, fmt.Sprintf("Margin call triggered: equity %.2f%% < %.2f%%", 
            maintenanceMarginCurrent*100, marginCallThreshold*100))
    }
    
    if len(violations) > 0 {
        result.Status = "BLOCK"
        result.Message = strings.Join(violations, "; ")
        result.Details["violations"] = violations
    } else {
        result.Status = "PASS"
        result.Message = "Account margin compliance verified"
    }
    
    return result
}
```

#### AI-Driven Risk Assessment Rule (26)

```go
func executeAIRiskAssessment(ctx context.Context, rule *Rule, evalCtx interface{}, params interface{}) *ExecutionResult {
    result := &ExecutionResult{
        RuleID:          "ai-risk-assessment-v1",
        Severity:        rule.Severity,
        Timestamp:       time.Now(),
        Details:         make(map[string]interface{}),
        ExternalAPICall: true,
    }
    
    params := rule.ConditionJSON
    maxVaR := params["maxVaR"].(float64)
    varConfidence := params["varConfidenceLevel"].(float64)
    stressScenarios := params["stressTestScenarios"].([]interface{})
    aiEndpoint := params["aiModelEndpoint"].(string)
    modelType := params["modelType"].(string)
    
    account := evalCtx.(map[string]interface{})
    positions := account["positions"].([]interface{})
    
    // Prepare request for AI model
    aiRequest := map[string]interface{}{
        "positions":       positions,
        "confidence":      varConfidence,
        "scenarios":       stressScenarios,
        "model_type":      modelType,
    }
    
    // Call AI model endpoint
    payload, _ := json.Marshal(aiRequest)
    req, err := http.NewRequestWithContext(ctx, "POST", aiEndpoint, 
        bytes.NewReader(payload))
    if err != nil {
        result.Status = "WARN"
        result.Message = fmt.Sprintf("Failed to create AI request: %v", err)
        return result
    }
    
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        result.Status = "WARN"
        result.Message = fmt.Sprintf("Failed to call AI model: %v", err)
        return result
    }
    defer resp.Body.Close()
    
    // Parse AI response
    var aiResult struct {
        VaR             float64                `json:"var"`
        StressTestLoss  map[string]float64     `json:"stress_test_loss"`
        RiskScore       float64                `json:"risk_score"`
        Recommendations []string               `json:"recommendations"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&aiResult); err != nil {
        result.Status = "WARN"
        result.Message = fmt.Sprintf("Failed to parse AI response: %v", err)
        return result
    }
    
    result.Details["var_estimate"] = aiResult.VaR
    result.Details["stress_test_loss"] = aiResult.StressTestLoss
    result.Details["risk_score"] = aiResult.RiskScore
    result.Details["recommendations"] = aiResult.Recommendations
    
    if aiResult.VaR > maxVaR {
        result.Status = "WARN"
        result.Message = fmt.Sprintf("Portfolio VaR (%.2f%%) exceeds threshold (%.2f%%)", 
            aiResult.VaR*100, maxVaR*100)
    } else {
        result.Status = "PASS"
        result.Message = fmt.Sprintf("Portfolio risk acceptable. VaR: %.2f%%", aiResult.VaR*100)
    }
    
    return result
}
```

#### AML Compliance Rule (29)

```go
func executeAMLCompliance(ctx context.Context, rule *Rule, evalCtx interface{}, params interface{}) *ExecutionResult {
    result := &ExecutionResult{
        RuleID:          "aml-compliance-v1",
        Severity:        rule.Severity,
        Timestamp:       time.Now(),
        Details:         make(map[string]interface{}),
        ExternalAPICall: true,
    }
    
    params := rule.ConditionJSON
    transactionThreshold := params["transactionThreshold"].(float64)
    cumulativeThreshold := params["cumulativeThreshold"].(float64)
    windowDays := int(params["cumulativeWindowDays"].(float64))
    suspiciousPatterns := params["suspiciousPatterns"].([]interface{})
    amlEndpoint := params["integrationEndpoint"].(string)
    
    account := evalCtx.(map[string]interface{})
    trades := account["trades"].([]interface{})
    
    flaggedTrades := []map[string]interface{}{}
    cumulativeVolume := 0.0
    patternMatches := 0
    
    for _, tradeRaw := range trades {
        trade := tradeRaw.(map[string]interface{})
        amount := trade["amount"].(float64)
        tradeDate := trade["date"].(time.Time)
        
        // Check single transaction threshold
        if amount > transactionThreshold {
            flaggedTrades = append(flaggedTrades, trade)
            continue
        }
        
        // Check cumulative threshold
        if isWithinWindow(tradeDate, time.Now(), windowDays) {
            cumulativeVolume += amount
        }
        
        // Check for suspicious patterns
        for _, patternRaw := range suspiciousPatterns {
            pattern := patternRaw.(string)
            if matchesPattern(trade, pattern) {
                patternMatches++
                flaggedTrades = append(flaggedTrades, trade)
            }
        }
    }
    
    result.Details["flagged_trades"] = len(flaggedTrades)
    result.Details["cumulative_volume"] = cumulativeVolume
    result.Details["pattern_matches"] = patternMatches
    
    // External AML screening
    if len(flaggedTrades) > 0 {
        screeningResult, err := screenWithAML(ctx, flaggedTrades, amlEndpoint)
        if err != nil {
            result.Status = "WARN"
            result.Message = fmt.Sprintf("AML screening failed: %v", err)
        } else {
            result.Details["aml_screening_result"] = screeningResult
            if screeningResult.IsSuspicious {
                result.Status = "BLOCK"
                result.Message = fmt.Sprintf("AML violations detected: %s", screeningResult.Reason)
                return result
            }
        }
    }
    
    if cumulativeVolume > cumulativeThreshold || len(flaggedTrades) > 0 {
        result.Status = "WARN"
        result.Message = fmt.Sprintf("Suspicious activity detected: %d flagged trades, $%.0f cumulative", 
            len(flaggedTrades), cumulativeVolume)
    } else {
        result.Status = "PASS"
        result.Message = "No AML violations detected"
    }
    
    return result
}

func screenWithAML(ctx context.Context, trades []map[string]interface{}, endpoint string) (
    map[string]interface{}, error) {
    payload, _ := json.Marshal(trades)
    resp, err := http.Post(endpoint, "application/json", bytes.NewReader(payload))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    return result, nil
}

func isWithinWindow(tradeDate time.Time, now time.Time, days int) bool {
    return tradeDate.After(now.AddDate(0, 0, -days))
}

func matchesPattern(trade map[string]interface{}, pattern string) bool {
    // Pattern matching logic
    switch pattern {
    case "rapid_transfers":
        // Check if multiple trades within short time
        return false // Implementation pending
    case "high_frequency_small_amounts":
        // Check for many small trades
        return false
    case "round_number_trades":
        amount := trade["amount"].(float64)
        return int64(amount)%10000 == 0
    }
    return false
}
```

### 3. Database Schema Updates

Add columns for tracking external API calls and execution results:

```sql
-- Add audit logging for rule executions
CREATE TABLE validation_rule_executions (
    id BIGSERIAL PRIMARY KEY,
    rule_id VARCHAR(255) NOT NULL,
    account_id VARCHAR(255) NOT NULL,
    execution_status VARCHAR(50), -- PASS, WARN, BLOCK
    execution_message TEXT,
    parameters JSONB,
    execution_details JSONB,
    external_api_called BOOLEAN,
    execution_time_ms BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (rule_id) REFERENCES validation_rules(id),
    INDEX idx_rule_id (rule_id),
    INDEX idx_account_id (account_id),
    INDEX idx_created_at (created_at)
);

-- Cache external API responses to reduce calls
CREATE TABLE external_api_cache (
    id BIGSERIAL PRIMARY KEY,
    api_type VARCHAR(50), -- esg, aml, benchmark, etc
    symbol_or_identifier VARCHAR(255),
    response_data JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    INDEX idx_api_type (api_type),
    INDEX idx_symbol (symbol_or_identifier),
    INDEX idx_expires (expires_at)
);
```

## Configuration

### Environment Variables

```bash
# ESG/AML/Benchmark API Configuration
MSCI_API_KEY=your_msci_api_key
MSCI_API_ENDPOINT=https://api.msci.com/esg-ratings

WORLD_CHECK_API_KEY=your_world_check_key
WORLD_CHECK_ENDPOINT=https://api.world-check.com/screen

BLOOMBERG_API_KEY=your_bloomberg_key
BLOOMBERG_ENDPOINT=https://api.bloomberg.com/benchmark-data

# AI Model Configuration
SAGEMAKER_ROLE_ARN=arn:aws:iam::YOUR_ACCOUNT:role/SageMaker
SAGEMAKER_ENDPOINT=https://api.sagemaker.us-east-1.amazonaws.com

# Rule Execution
RULE_EXECUTION_TIMEOUT_MS=30000
EXTERNAL_API_RETRY_COUNT=3
EXTERNAL_API_RETRY_DELAY_MS=1000
```

### Configuration File

Update `config.yaml`:

```yaml
validation:
  rules:
    execution_timeout_ms: 30000
    external_api_retry_count: 3
    cache_responses: true
    cache_ttl_hours: 24
    
  external_services:
    esg:
      provider: msci
      endpoint: ${MSCI_API_ENDPOINT}
      api_key: ${MSCI_API_KEY}
      timeout_ms: 10000
      
    aml:
      provider: world_check
      endpoint: ${WORLD_CHECK_ENDPOINT}
      api_key: ${WORLD_CHECK_API_KEY}
      timeout_ms: 15000
      
    benchmark:
      provider: bloomberg
      endpoint: ${BLOOMBERG_ENDPOINT}
      api_key: ${BLOOMBERG_API_KEY}
      timeout_ms: 10000
      
    ai_risk:
      endpoint: ${SAGEMAKER_ENDPOINT}
      model_type: tensorflow_var
      timeout_ms: 30000
```

## Testing

### Unit Test Example (Go)

```go
package api

import (
    "context"
    "testing"
    "time"
)

func TestExecuteTaxOptimization(t *testing.T) {
    rule := &Rule{
        ID: "tax-optimization-v1",
        ConditionJSON: map[string]interface{}{
            "maxTaxableGainPercentage": 0.15,
            "washSaleWindowDays":       30,
        },
    }
    
    evalCtx := map[string]interface{}{
        "trades": []interface{}{
            map[string]interface{}{
                "symbol":       "AAPL",
                "realizedGain": 5000.0,
                "date":         time.Now(),
            },
        },
    }
    
    result := executeTaxOptimization(context.Background(), rule, evalCtx, nil)
    
    if result.Status != "PASS" && result.Status != "WARN" {
        t.Errorf("Unexpected status: %s", result.Status)
    }
}
```

## Deployment Checklist

- [ ] Update Go dependencies (HTTP client, JSON parsing)
- [ ] Add database migrations for audit tables and caching
- [ ] Configure environment variables for external APIs
- [ ] Implement parameter validation in handlers
- [ ] Add error handling and retry logic
- [ ] Set up monitoring/alerting for API failures
- [ ] Create integration tests for each rule
- [ ] Load test external API calls
- [ ] Document API key rotation procedures
- [ ] Set up audit logging
