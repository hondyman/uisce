# Portfolio Management System - Quick Reference Card

## 🚀 Quick Start (5 minutes)

### 1. Database Setup
```bash
psql < portfolio-management/database/portfolio_management_schema.sql
```

### 2. Start Backend
```bash
cd portfolio-management/backend
go mod download
go run ./cmd/main.go
```

### 3. Test Health
```bash
curl http://localhost:8081/health
```

## 📚 Core API Endpoints

### Portfolio Management
```
POST   /api/portfolios               Create portfolio
GET    /api/holdings                 List holdings
GET    /api/portfolio-risk-metrics   Get risk metrics
```

### Recommendations
```
POST   /api/recommendations          Create recommendation
GET    /api/recommendation-status    Get details
PATCH  /api/recommendation-status    Update status
```

### Backtesting
```
POST   /api/backtest/run             Execute backtest
GET    /api/backtest/results         Get history
POST   /api/backtest/compare         Compare strategies
```

## 🔧 Code Structure

```
portfolio-management/
├── backend/cmd/main.go              HTTP handlers & routes
├── backend/internal/backtest/
│   ├── models.go                   Domain types
│   └── service.go                  Business logic
└── database/
    └── portfolio_management_schema.sql   Database
```

## 📊 Key Models

### Portfolio
```go
type Portfolio struct {
    ID          uuid.UUID
    UserID      uuid.UUID
    Name        string
    TotalValue  float64
    Holdings    []Holding
    Metadata    json.RawMessage
}
```

### Recommendation
```go
type Recommendation struct {
    ID                uuid.UUID
    PortfolioID       uuid.UUID
    Type              string    // rebalance, tactical, strategic
    Status            string    // draft, proposed, accepted
    TargetAllocations []TargetAllocation
    Rationale         string
}
```

### BacktestResult
```go
type BacktestResult struct {
    ID                     uuid.UUID
    BaselineReturn         float64
    RecommendationReturn   float64
    AlphaGenerated         float64
    SharpeRatioBaseline    float64
    SharpeRatioRecommended float64
    NetBenefit             float64
}
```

## 📈 Example: Complete Flow

### 1. Create Portfolio
```bash
curl -X POST http://localhost:8081/api/portfolios \
  -H "X-User-ID: user-123" \
  -d '{"name":"My Portfolio","currency":"USD","holdings":[...]}'
```
Returns: `Portfolio` with ID

### 2. Create Recommendation
```bash
curl -X POST "http://localhost:8081/api/recommendations?portfolio_id=PORT_ID" \
  -H "X-User-ID: user-123" \
  -d '{"title":"...","type":"rebalance","target_allocations":[...]}'
```
Returns: `Recommendation` with ID

### 3. Run Backtest
```bash
curl -X POST http://localhost:8081/api/backtest/run \
  -d '{"portfolio_id":"PORT_ID","recommendation_id":"REC_ID","start_date":"2023-01-01T00:00:00Z","end_date":"2024-01-01T00:00:00Z"}'
```
Returns: `BacktestResult` with metrics

### 4. Get Results
```bash
curl "http://localhost:8081/api/backtest/results?portfolio_id=PORT_ID&limit=10"
```
Returns: List of `BacktestResult[]`

### 5. Compare Recommendations
```bash
curl -X POST http://localhost:8081/api/backtest/compare \
  -d '{"portfolio_id":"PORT_ID","recommendation_id_1":"REC1","recommendation_id_2":"REC2"}'
```
Returns: `ComparisonResult` with winner

## 🗄️ Database Tables

| Table | Purpose |
|-------|---------|
| `portfolios` | Portfolio records |
| `holdings` | Individual investments |
| `recommendations` | Investment strategies |
| `backtest_results` | Simulation outcomes |
| `historical_prices` | Price history |
| `portfolio_risk_metrics` | Risk analysis |
| `rebalancing_plans` | Rebalancing proposals |

## 🔐 Authentication

All endpoints require headers:
```
X-User-ID: {user_id}           // Required for portfolio ops
X-Tenant-ID: {tenant_id}       // For multi-tenant support
X-Tenant-Datasource-ID: {ds_id} // For data scoping
```

## 📊 Key Metrics Explained

| Metric | Formula | Interpretation |
|--------|---------|-----------------|
| **Alpha** | Rec Return - Baseline Return | Excess return from change |
| **Sharpe** | (Return - RiskFree) / Volatility | Risk-adjusted return |
| **Drawdown** | Max Decline | Worst peak-to-trough loss |
| **Net Benefit** | Alpha + Tax Savings - Costs | Total value add |

## 🐛 Debugging

### Check service health
```bash
curl http://localhost:8081/health
```

### View recent backtests
```bash
curl "http://localhost:8081/api/backtest/results?portfolio_id=PORTFOLIO_ID"
```

### Get detailed backtest
```bash
curl "http://localhost:8081/api/backtest-detail?id=BACKTEST_ID"
```

## 📝 Common Tasks

### Create sample portfolio
```bash
cat << 'EOF' | curl -X POST http://localhost:8081/api/portfolios \
  -H "X-User-ID: user-123" \
  -d @-
{
  "name": "Sample Portfolio",
  "currency": "USD",
  "holdings": [
    {"symbol": "AAPL", "name": "Apple", "asset_class": "equity", "quantity": 100, "average_cost": 150},
    {"symbol": "MSFT", "name": "Microsoft", "asset_class": "equity", "quantity": 50, "average_cost": 300}
  ]
}
EOF
```

### Update recommendation to accepted
```bash
curl -X PATCH "http://localhost:8081/api/recommendation-status?id=REC_ID" \
  -d '{"status":"accepted","notes":"Client approved"}'
```

## 🚨 Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| 400 Bad Request | Missing fields | Check required fields in body |
| 401 Unauthorized | Missing X-User-ID | Add X-User-ID header |
| 404 Not Found | Resource doesn't exist | Verify IDs are correct |
| 500 Internal Error | Database issue | Check database connection |

## 📚 Full Documentation

- **Complete Guide**: `portfolio-management/PORTFOLIO_MANAGEMENT_COMPLETE.md`
- **Executive Summary**: `/PORTFOLIO_MANAGEMENT_SYSTEM_COMPLETE.md`
- **API Reference**: See endpoint list above
- **Database Schema**: `portfolio-management/database/portfolio_management_schema.sql`

## ✅ Verification Checklist

- [ ] Database created and schema applied
- [ ] Backend compiles without errors
- [ ] Service starts on port 8081
- [ ] `/health` endpoint returns 200
- [ ] Can create portfolio
- [ ] Can create recommendation
- [ ] Backtest execution works
- [ ] Results are returned correctly
- [ ] Comparison engine functioning

## 🎯 Next Steps

1. **Build Frontend Components**
   - PortfolioDashboard.tsx
   - RecommendationReviewUI.tsx
   - RiskAnalyticsDashboard.tsx

2. **Load Historical Data**
   - Import 2+ years of prices
   - Test backtest simulations

3. **Integration Testing**
   - End-to-end flows
   - Performance benchmarking
   - Error scenarios

4. **Production Deployment**
   - Environment setup
   - Database migration
   - Monitoring & alerting

---

**Version**: 1.0.0  
**Last Updated**: October 30, 2025  
**Status**: Production Ready
