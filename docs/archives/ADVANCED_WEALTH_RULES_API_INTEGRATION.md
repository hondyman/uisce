# Advanced Wealth Management Rules - External API Integration Guide

This guide provides implementation strategies for integrating external services with the advanced wealth management validation rules.

## Overview

The advanced rules leverage five categories of external integrations:

| API | Rule | Purpose | Provider |
|-----|------|---------|----------|
| ESG Ratings | ESG Compliance (22) | Score holdings against environmental/social/governance criteria | MSCI, Refinitiv, Sustainalytics |
| AML Screening | AML Compliance (29) | Screen transactions for money laundering patterns | World-Check (Refinitiv), Sanctions List |
| Benchmark Data | Performance Benchmarking (28) | Compare portfolio returns to indices | Bloomberg, Refinitiv, Yahoo Finance |
| AI Risk Models | AI Risk Assessment (26) | Calculate Value-at-Risk and stress tests | AWS SageMaker, TensorFlow, PyTorch |
| Market Data | Portfolio Drift (24), Margin (23) | Real-time pricing and liquidity | Market Data Vendors |

## 1. ESG Compliance API Integration (Rule 22)

### MSCI ESG Ratings API

**Endpoint**: `https://api.msci.com/esg-ratings`

**Integration Flow**:

```go
// Client wrapper for MSCI ESG API
type MSCIESGClient struct {
    apiKey  string
    timeout time.Duration
}

func NewMSCIESGClient(apiKey string) *MSCIESGClient {
    return &MSCIESGClient{
        apiKey:  apiKey,
        timeout: 10 * time.Second,
    }
}

// GetESGRating fetches ESG score for a security
func (c *MSCIESGClient) GetESGRating(ctx context.Context, symbol string, isin string) (*ESGRating, error) {
    // Create request
    req, err := http.NewRequestWithContext(ctx, "GET", 
        fmt.Sprintf("https://api.msci.com/esg-ratings?symbol=%s&isin=%s", symbol, isin), nil)
    if err != nil {
        return nil, err
    }
    
    // Add authentication
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
    req.Header.Set("Content-Type", "application/json")
    
    // Execute with timeout
    client := &http.Client{Timeout: c.timeout}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("ESG API call failed: %w", err)
    }
    defer resp.Body.Close()
    
    // Parse response
    var rating ESGRating
    if err := json.NewDecoder(resp.Body).Decode(&rating); err != nil {
        return nil, err
    }
    
    return &rating, nil
}

type ESGRating struct {
    Symbol              string   `json:"symbol"`
    ISIN                string   `json:"isin"`
    CompanyName         string   `json:"company_name"`
    ESGScore            float64  `json:"esg_score"` // 0-10
    EnvironmentScore    float64  `json:"environment_score"`
    SocialScore         float64  `json:"social_score"`
    GovernanceScore     float64  `json:"governance_score"`
    ESGRating           string   `json:"esg_rating"` // AAA, AA, A, BBB, BB, B, CCC
    Controversy         string   `json:"controversy_flag"`
    RestrictedBusiness  []string `json:"restricted_business_activities"`
    LastUpdated         time.Time `json:"last_updated"`
}
```

**Caching Strategy**:

```go
// Cache ESG ratings to minimize API calls
type ESGCache struct {
    ttl time.Duration
    db  *sql.DB
}

func (c *ESGCache) GetESGRating(ctx context.Context, symbol string) (*ESGRating, error) {
    // Check cache first
    var cached string
    err := c.db.QueryRowContext(ctx, 
        "SELECT response_data FROM external_api_cache WHERE api_type = 'esg' AND symbol_or_identifier = ? AND expires_at > NOW()",
        symbol).Scan(&cached)
    
    if err == nil {
        var rating ESGRating
        json.Unmarshal([]byte(cached), &rating)
        return &rating, nil
    }
    
    // If not cached or expired, fetch from API
    rating, err := msciClient.GetESGRating(ctx, symbol, "")
    if err != nil {
        return nil, err
    }
    
    // Store in cache
    data, _ := json.Marshal(rating)
    c.db.ExecContext(ctx,
        "INSERT INTO external_api_cache (api_type, symbol_or_identifier, response_data, expires_at) VALUES (?, ?, ?, DATE_ADD(NOW(), INTERVAL ? HOUR))",
        "esg", symbol, string(data), int(c.ttl.Hours()))
    
    return rating, nil
}
```

### Integration in Rule Execution

```go
func executeESGCompliance(ctx context.Context, rule *Rule, evalCtx interface{}, params interface{}) *ExecutionResult {
    result := &ExecutionResult{
        RuleID:          "esg-compliance-v1",
        Severity:        rule.Severity,
        Timestamp:       time.Now(),
        Details:         make(map[string]interface{}),
        ExternalAPICall: true,
    }
    
    params := rule.ConditionJSON
    minEsgScore := params["minEsgScore"].(float64)
    restrictedSectors := params["restrictedSectors"].([]interface{})
    
    holdings := evalCtx.(map[string]interface{})["holdings"].([]interface{})
    
    esgCache := NewESGCache(24 * time.Hour, db) // 24-hour TTL
    
    violations := []string{}
    avgESGScore := 0.0
    
    for _, holdingRaw := range holdings {
        holding := holdingRaw.(map[string]interface{})
        symbol := holding["symbol"].(string)
        
        // Get ESG rating (cached or fresh)
        rating, err := esgCache.GetESGRating(ctx, symbol)
        if err != nil {
            result.Message = fmt.Sprintf("Failed to fetch ESG data: %v", err)
            result.Status = "WARN"
            return result
        }
        
        // Check ESG score
        if rating.ESGScore < minEsgScore {
            violations = append(violations, fmt.Sprintf("%s ESG score %.1f < %.1f", 
                symbol, rating.ESGScore, minEsgScore))
        }
        
        // Check restricted businesses
        for _, business := range rating.RestrictedBusiness {
            for _, restricted := range restrictedSectors {
                if business == restricted.(string) {
                    violations = append(violations, fmt.Sprintf("%s has restricted business: %s", 
                        symbol, business))
                }
            }
        }
        
        avgESGScore += rating.ESGScore
    }
    
    avgESGScore /= float64(len(holdings))
    result.Details["average_esg_score"] = avgESGScore
    result.Details["violations"] = violations
    
    if len(violations) == 0 {
        result.Status = "PASS"
        result.Message = fmt.Sprintf("All holdings meet ESG criteria (avg score: %.1f)", avgESGScore)
    } else {
        result.Status = "WARN"
        result.Message = fmt.Sprintf("%d ESG violations detected", len(violations))
    }
    
    return result
}
```

## 2. AML Compliance API Integration (Rule 29)

### World-Check API (Refinitiv)

**Endpoint**: `https://api.world-check.com/screen`

**Integration Flow**:

```go
type WorldCheckClient struct {
    apiKey  string
    timeout time.Duration
}

func (c *WorldCheckClient) ScreenTransactions(ctx context.Context, 
    transactions []Transaction) (*AMLScreeningResult, error) {
    
    // Prepare screening request
    req := struct {
        Transactions []Transaction `json:"transactions"`
        GroupID      string        `json:"group_id"`
    }{
        Transactions: transactions,
        GroupID:      "default",
    }
    
    payload, _ := json.Marshal(req)
    httpReq, _ := http.NewRequestWithContext(ctx, "POST", 
        "https://api.world-check.com/screen", bytes.NewReader(payload))
    
    httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
    httpReq.Header.Set("Content-Type", "application/json")
    
    client := &http.Client{Timeout: c.timeout}
    resp, err := client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result AMLScreeningResult
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}

type AMLScreeningResult struct {
    TransactionID      string           `json:"transaction_id"`
    Status             string           `json:"status"` // CLEAR, WARN, MATCH
    Matches            []SanctionsMatch `json:"matches"`
    RiskLevel          string           `json:"risk_level"` // LOW, MEDIUM, HIGH, CRITICAL
    RecommendedAction  string           `json:"recommended_action"`
    SARRequired        bool             `json:"sar_required"`
}

type SanctionsMatch struct {
    Type              string  `json:"type"` // entity, individual
    Name              string  `json:"name"`
    Confidence        float64 `json:"confidence"` // 0-1
    List              string  `json:"list"` // SDN, OFAC, EU_CONSOLIDATED, etc
    MatchedField      string  `json:"matched_field"`
}
```

**Integration in Rule Execution**:

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
    amlEndpoint := params["integrationEndpoint"].(string)
    
    account := evalCtx.(map[string]interface{})
    trades := account["trades"].([]interface{})
    
    // Batch transactions for screening
    flaggedTrades := []Transaction{}
    
    for _, tradeRaw := range trades {
        trade := tradeRaw.(map[string]interface{})
        amount := trade["amount"].(float64)
        
        if amount > transactionThreshold {
            flaggedTrades = append(flaggedTrades, Transaction{
                ID:           trade["id"].(string),
                Amount:       amount,
                Currency:     "USD",
                Type:         "TRADE",
                CounterpartyName: trade["counterparty"].(string),
                Date:         trade["date"].(time.Time),
            })
        }
    }
    
    if len(flaggedTrades) == 0 {
        result.Status = "PASS"
        result.Message = "No transactions above threshold"
        return result
    }
    
    // Screen with World-Check
    wcClient := NewWorldCheckClient(os.Getenv("WORLD_CHECK_API_KEY"))
    screenResult, err := wcClient.ScreenTransactions(ctx, flaggedTrades)
    
    if err != nil {
        result.Status = "WARN"
        result.Message = fmt.Sprintf("AML screening unavailable: %v", err)
        return result
    }
    
    // Process results
    highRiskMatches := []SanctionsMatch{}
    for _, match := range screenResult.Matches {
        if match.Confidence > 0.8 {
            highRiskMatches = append(highRiskMatches, match)
        }
    }
    
    result.Details["flagged_transactions"] = len(flaggedTrades)
    result.Details["matches_found"] = len(screenResult.Matches)
    result.Details["high_confidence_matches"] = len(highRiskMatches)
    result.Details["risk_level"] = screenResult.RiskLevel
    result.Details["sar_required"] = screenResult.SARRequired
    
    if screenResult.RiskLevel == "CRITICAL" || len(highRiskMatches) > 0 {
        result.Status = "BLOCK"
        result.Message = fmt.Sprintf("AML violations detected: %d matches, %s risk", 
            len(screenResult.Matches), screenResult.RiskLevel)
        
        // Flag for Suspicious Activity Report (SAR)
        if screenResult.SARRequired {
            logSARRequirement(account, flaggedTrades, screenResult)
        }
    } else if len(screenResult.Matches) > 0 {
        result.Status = "WARN"
        result.Message = fmt.Sprintf("Potential AML concerns: %d matches detected", 
            len(screenResult.Matches))
    } else {
        result.Status = "PASS"
        result.Message = "AML screening passed"
    }
    
    return result
}
```

## 3. Benchmark Data API Integration (Rule 28)

### Bloomberg API Integration

**Endpoint**: `https://api.bloomberg.com/benchmark-data`

**Integration Flow**:

```go
type BloombergClient struct {
    apiKey  string
    timeout time.Duration
}

func (c *BloombergClient) GetBenchmarkData(ctx context.Context, 
    benchmarkIndex string, startDate, endDate time.Time) (*BenchmarkData, error) {
    
    req, _ := http.NewRequestWithContext(ctx, "GET",
        fmt.Sprintf("https://api.bloomberg.com/benchmark-data?index=%s&from=%s&to=%s",
            benchmarkIndex, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")), nil)
    
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
    
    client := &http.Client{Timeout: c.timeout}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var data BenchmarkData
    json.NewDecoder(resp.Body).Decode(&data)
    return &data, nil
}

type BenchmarkData struct {
    Index          string                `json:"index"`
    StartDate      time.Time             `json:"start_date"`
    EndDate        time.Time             `json:"end_date"`
    StartPrice     float64               `json:"start_price"`
    EndPrice       float64               `json:"end_price"`
    Return         float64               `json:"return"` // As decimal (0.05 = 5%)
    DividendYield  float64               `json:"dividend_yield"`
    HistoricalData []BenchmarkDataPoint  `json:"historical_data"`
}

type BenchmarkDataPoint struct {
    Date  time.Time `json:"date"`
    Price float64   `json:"price"`
}
```

**Integration in Rule Execution**:

```go
func executePerformanceBenchmarking(ctx context.Context, rule *Rule, evalCtx interface{}, params interface{}) *ExecutionResult {
    result := &ExecutionResult{
        RuleID:          "performance-benchmarking-v1",
        Severity:        rule.Severity,
        Timestamp:       time.Now(),
        Details:         make(map[string]interface{}),
        ExternalAPICall: true,
    }
    
    params := rule.ConditionJSON
    benchmarkIndex := params["benchmarkIndex"].(string)
    minPerformanceDelta := params["minPerformanceDelta"].(float64)
    evaluationMonths := int(params["evaluationPeriodMonths"].(float64))
    
    account := evalCtx.(map[string]interface{})
    positions := account["positions"].([]interface{})
    
    // Calculate portfolio return
    portfolioReturn := calculatePortfolioReturn(positions, evaluationMonths)
    
    // Get benchmark data
    bbgClient := NewBloombergClient(os.Getenv("BLOOMBERG_API_KEY"))
    now := time.Now()
    benchmarkData, err := bbgClient.GetBenchmarkData(ctx, benchmarkIndex,
        now.AddDate(0, -evaluationMonths, 0), now)
    
    if err != nil {
        result.Status = "WARN"
        result.Message = fmt.Sprintf("Benchmark data unavailable: %v", err)
        return result
    }
    
    // Compare returns
    performanceDelta := portfolioReturn - benchmarkData.Return
    
    result.Details["portfolio_return"] = portfolioReturn
    result.Details["benchmark_return"] = benchmarkData.Return
    result.Details["performance_delta"] = performanceDelta
    result.Details["benchmark_index"] = benchmarkIndex
    result.Details["evaluation_period_months"] = evaluationMonths
    
    if performanceDelta < minPerformanceDelta {
        result.Status = "WARN"
        result.Message = fmt.Sprintf("Underperforming benchmark by %.2f%%", 
            (minPerformanceDelta-performanceDelta)*100)
    } else {
        result.Status = "PASS"
        alpha := performanceDelta
        result.Message = fmt.Sprintf("Outperforming benchmark by %.2f%% (alpha: %.2f%%)", 
            (performanceDelta-minPerformanceDelta)*100, alpha*100)
    }
    
    return result
}
```

## 4. AI Risk Assessment API Integration (Rule 26)

### AWS SageMaker Integration

**Setup**:

```bash
# Create SageMaker Jupyter notebook for VAR model
aws sagemaker create-notebook-instance \
    --notebook-instance-name wealth-var-model \
    --instance-type ml.t3.medium \
    --role-arn arn:aws:iam::YOUR_ACCOUNT:role/SageMaker

# Deploy model as endpoint
aws sagemaker create-endpoint \
    --endpoint-name wealth-risk-assessment \
    --endpoint-config-name wealth-risk-assessment-config
```

**Model Code (Python)**:

```python
# Lambda function for risk assessment
import json
import boto3
import numpy as np
from scipy.stats import norm

def lambda_handler(event, context):
    """
    Calculate Value-at-Risk and stress test scenarios
    """
    positions = event['positions']
    confidence = event['confidence']  # e.g., 0.95
    
    # Convert positions to returns array
    returns = []
    for pos in positions:
        # Historical return calculation
        historical_return = pos.get('ytd_return', 0)
        returns.append(historical_return)
    
    # Calculate VAR
    returns_array = np.array(returns)
    mean_return = np.mean(returns_array)
    std_return = np.std(returns_array)
    
    # VaR = mean - z_score * std
    z_score = norm.ppf(1 - confidence)
    var = mean_return - (z_score * std_return)
    
    # Stress test scenarios
    scenarios = event.get('scenarios', ['market_crash_10', 'interest_rate_spike'])
    stress_results = {}
    
    for scenario in scenarios:
        if scenario == 'market_crash_10':
            stress_loss = np.sum(returns_array) * -0.10
        elif scenario == 'interest_rate_spike':
            stress_loss = np.sum([r * -0.05 for r in returns_array if 'BOND' in pos.get('asset_class', '')])
        else:
            stress_loss = 0
            
        stress_results[scenario] = stress_loss
    
    return {
        'var': abs(var),
        'stress_test_loss': stress_results,
        'risk_score': (abs(var) + np.mean(list(stress_results.values()))) / 2,
        'recommendations': generate_recommendations(var, stress_results)
    }

def generate_recommendations(var, stress_results):
    recommendations = []
    if var > 0.05:
        recommendations.append("Consider reducing position concentration")
    if max(stress_results.values()) > 0.15:
        recommendations.append("Portfolio sensitive to interest rate changes - consider duration adjustment")
    return recommendations
```

**Go Integration**:

```go
type SageMakerClient struct {
    endpointName string
    sagemaker    *sagemaker.Client
}

func (c *SageMakerClient) InvokeRiskModel(ctx context.Context, 
    positions []interface{}, confidence float64) (*RiskAssessmentResult, error) {
    
    // Prepare payload
    payload := map[string]interface{}{
        "positions":  positions,
        "confidence": confidence,
        "scenarios":  []string{"market_crash_10", "interest_rate_spike"},
    }
    
    payloadJSON, _ := json.Marshal(payload)
    
    // Invoke SageMaker endpoint
    resp, err := c.sagemaker.InvokeEndpoint(ctx, &sagemaker.InvokeEndpointInput{
        EndpointName:   aws.String(c.endpointName),
        ContentType:    aws.String("application/json"),
        Body:           payloadJSON,
    })
    
    if err != nil {
        return nil, err
    }
    
    var result RiskAssessmentResult
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}

type RiskAssessmentResult struct {
    VaR            float64            `json:"var"`
    StressTestLoss map[string]float64 `json:"stress_test_loss"`
    RiskScore      float64            `json:"risk_score"`
    Recommendations []string           `json:"recommendations"`
}
```

## 5. API Retry & Error Handling

### Retry Strategy

```go
type RetryPolicy struct {
    MaxRetries      int
    InitialDelay    time.Duration
    BackoffMultiplier float64
}

func (p *RetryPolicy) ExecuteWithRetry(ctx context.Context, 
    fn func(context.Context) error) error {
    
    var lastErr error
    delay := p.InitialDelay
    
    for attempt := 0; attempt < p.MaxRetries; attempt++ {
        err := fn(ctx)
        if err == nil {
            return nil
        }
        
        lastErr = err
        
        // Check if error is retryable
        if !isRetryable(err) {
            return err
        }
        
        // Exponential backoff
        select {
        case <-time.After(delay):
            delay = time.Duration(float64(delay) * p.BackoffMultiplier)
        case <-ctx.Done():
            return ctx.Err()
        }
    }
    
    return lastErr
}

func isRetryable(err error) bool {
    // Retry on network errors, timeouts
    if os.IsTimeout(err) {
        return true
    }
    
    // Don't retry on 4xx errors (except 429)
    if apiErr, ok := err.(*APIError); ok {
        return apiErr.StatusCode >= 500 || apiErr.StatusCode == 429
    }
    
    return true
}
```

## Monitoring & Observability

### Logging

```go
type RuleExecutionLog struct {
    RuleID            string                 `json:"rule_id"`
    AccountID         string                 `json:"account_id"`
    Status            string                 `json:"status"`
    ExecutionTime     int64                  `json:"execution_time_ms"`
    ExternalAPICall   bool                   `json:"external_api_call"`
    APIName           string                 `json:"api_name,omitempty"`
    APIResponseTime   int64                  `json:"api_response_time_ms,omitempty"`
    ErrorMessage      string                 `json:"error_message,omitempty"`
    Timestamp         time.Time              `json:"timestamp"`
}

// Log to structured logging system (e.g., CloudWatch, ELK)
func logRuleExecution(log *RuleExecutionLog) {
    logger.WithFields(log).Info("Rule execution completed")
}
```

### Alerts

Configure alerts for:
- API downtime (>5% failure rate)
- Slow responses (>10s)
- SAR triggers (AML rule blocks)
- ESG score changes >2 points
