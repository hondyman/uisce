# Calculation Engine: Visual Reference & Quick Links

## 📍 Where Everything Is Located

### Frontend URLs

| Feature | URL | Menu Path | File |
|---------|-----|-----------|------|
| **Calculations Library** | `/fabric/calculations` | Fabric → Calculations Library | `CalculationsLibraryPage.tsx` |
| **Custom Components** | `/fabric/custom-components` | Fabric → Custom Components | `CustomComponentPage.tsx` |
| **Metrics Console** | `/metrics` | Top Menu → Metrics | `MetricsConsolePage.tsx` |
| **Fabric Menu** | N/A | Fabric (dropdown) | `AppRoutes.tsx` line ~200 |

### Backend APIs

| Endpoint | Method | Purpose | File | Line |
|----------|--------|---------|------|------|
| `/api/calc/run` | POST | Execute calculation | `api.go` | 4618 |
| `/api/calc/vectorized` | POST | Batch calculations | `api.go` | 4636 |
| `/api/custom-components` | GET/POST/PUT/DELETE | Manage components | `custom_components.go` | 51 |
| `/api/custom-components/test-api` | POST | Test API connectivity | `custom_components.go` | 425 |

---

## 🎯 Quick Navigation

### I want to... 

**→ Test a calculation**
1. Go to: `http://localhost:5173/fabric/calculations`
2. Filter: Performance → IRR
3. (Coming soon) Click "Test" button
4. Enter sample data
5. See results

**→ Set up external data source**
1. Go to: `http://localhost:5173/fabric/custom-components`
2. Click "Add Component"
3. Select "API Integration"
4. Enter API endpoint
5. Click "Test API"
6. Reference in calculation

**→ Edit a calculation**
1. Go to: `http://localhost:5173/fabric/calculations`
2. Find calculation card
3. Click "Edit" button
4. Modify formula/config
5. Click "Save Changes"

**→ Create new calculation**
1. Go to: `http://localhost:5173/fabric/calculations`
2. Click "Add Calculation" button
3. Fill in form:
   - Name: `my_calc_id`
   - Title: `My Calculation Title`
   - Formula: `SUM(field1) / COUNT(field2)`
   - Category: `Performance`
   - Subcategory: `IRR`
4. Click "Add to Library"

**→ Run calculation via API**
```bash
curl -X POST http://localhost:8082/api/calc/run \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-id" \
  -H "X-Tenant-Datasource-ID: ds-id" \
  -d '{
    "financial": {
      "type": "xirr",
      "formula": "xirr(flows, dates)"
    }
  }'
```

---

## 📊 Calculation Types & Examples

### By Category

```
Performance
├── Returns (total return, annualized return)
├── Growth (CAGR, growth rate)
├── Valuation (P/E, PB ratio)
└── IRR (IRR, XIRR, MIRR)

Risk
├── Volatility (standard deviation, variance)
├── Drawdown (max drawdown, recovery time)
├── Correlation (asset correlation)
├── Market Risk (beta, VaR)
└── Credit Risk (default probability)

Private Markets
├── Performance (net IRR, gross IRR)
├── Multiples (MOIC, revenue multiple)
├── Cash Flow (DPI, PIC)
├── Liquidity (exit rate, hold time)
└── Valuation (residual value)

Insurance
├── Underwriting (loss ratio, expense ratio)
├── Reserving (reserve adequacy)
├── Solvency (coverage ratio)
└── Profitability (combined ratio)

Banking & Lending
├── Risk (credit spread, LTV)
├── Profitability (net interest margin)
└── Regulatory (capital ratio, leverage ratio)

Quant Finance
├── Market Risk (VaR, CVaR)
├── Derivatives (option pricing)
└── Fixed Income (duration, convexity)

Risk Management
├── Market Risk (delta, gamma)
├── Credit Risk (migration, transition matrix)
└── Operational Risk (loss distribution)

Compliance & Regulatory
├── Banking (Basel III, capital requirements)
├── Insurance (Solvency II, technical provisions)
├── AML/KYC (risk scoring)
└── Market Conduct (conduct risk)

Wealth Management
├── Allocation (strategic allocation)
└── Diversification (Herfindahl index)
```

### Popular Calculations

```
IRR (Internal Rate of Return)
├── Type: xirr (irregular cash flows)
├── Input: cash_flows[], transaction_dates[]
├── Output: percentage (e.g., 12.45%)
└── Use: Investment returns, fund performance

XIRR (Extended IRR)
├── Type: xirr_with_dates
├── Input: amounts, dates (irregular intervals)
├── Output: percentage
└── Use: Real-world investments with irregular timing

MIRR (Modified IRR)
├── Type: mirr
├── Input: cash_flows[], reinvestment_rate, cost_rate
├── Output: percentage (adjusted for reinvestment)
└── Use: More realistic return calculations

Volatility
├── Type: standard_deviation
├── Input: returns[]
├── Output: percentage (e.g., 15.2%)
└── Use: Risk measurement

Sharpe Ratio
├── Type: sharpe_ratio
├── Input: returns[], risk_free_rate
├── Output: decimal (e.g., 0.85)
└── Use: Risk-adjusted performance

VaR (Value at Risk)
├── Type: value_at_risk
├── Input: returns[], confidence_level
├── Output: amount (e.g., -$50,000)
└── Use: Downside risk measurement
```

---

## 🔧 Configuration Quick Reference

### External Service Config Template

```json
{
  "name": "Service Name",
  "type": "api_integration",
  "config": {
    "apiEndpoint": "https://api.example.com/endpoint",
    "method": "POST",
    "headers": {
      "Authorization": "Bearer API_KEY",
      "Content-Type": "application/json"
    },
    "refreshInterval": 300,
    "timeout": 30,
    "cacheStrategy": "short"
  },
  "events": [
    {
      "eventName": "onDataReady",
      "action": "custom",
      "customScript": "window.ServiceData = response.data;"
    }
  ],
  "filters": [
    {
      "field": "ticker",
      "operator": "in"
    }
  ]
}
```

### Calculation Config Template

```json
{
  "name": "calculation_id",
  "title": "Calculation Title",
  "type": "measure",
  "category": "Performance",
  "subcategory": "IRR",
  "description": "Detailed description",
  "sql": "SELECT calculation_formula",
  "financial_calc": {
    "type": "xirr",
    "formula": "xirr(ARRAY_AGG(cash_flow), ARRAY_AGG(date))",
    "arguments": {
      "cash_flows": "column_name",
      "dates": "date_column_name"
    }
  },
  "backendEndpoint": "/api/calc/run",
  "preAggregationTemplate": {
    "name": "pre_agg_template",
    "description": "Performance optimization"
  }
}
```

---

## 📈 Data Flow Diagrams

### Simple Calculation Flow
```
┌────────────────────────────────┐
│ User selects calculation       │
│ e.g., "Investment XIRR"        │
└────────────────┬───────────────┘
                 ↓
┌────────────────────────────────┐
│ Frontend calls                 │
│ POST /api/calc/run             │
│ with formula & data            │
└────────────────┬───────────────┘
                 ↓
┌────────────────────────────────┐
│ Backend receives request       │
│ Parses financial_calc object   │
└────────────────┬───────────────┘
                 ↓
┌────────────────────────────────┐
│ Dispatcher matches calculation │
│ Executes computation logic     │
└────────────────┬───────────────┘
                 ↓
┌────────────────────────────────┐
│ Returns result                 │
│ {value: 0.1245, display: ...}  │
└────────────────┬───────────────┘
                 ↓
┌────────────────────────────────┐
│ Frontend displays result       │
│ Shows formatted percentage     │
└────────────────────────────────┘
```

### Calculation with External Service
```
┌────────────────────────────────┐
│ User runs calculation with     │
│ external service reference     │
└────────────────┬───────────────┘
                 ↓
┌────────────────────────────────┐
│ Frontend calls                 │
│ POST /api/calc/run             │
│ with external_services config  │
└────────────────┬───────────────┘
                 ↓
┌────────────────────────────────┐
│ Backend receives & validates   │
│ Parses request + external svc  │
└────────────────┬───────────────┘
                 ↓
┌────────────────────────────────┐
│ Check cache for external data  │
│ If not found: fetch new        │
└────────────────┬───────────────┘
                 ↓
┌────────────────────────────────┐
│ Call external service API      │
│ e.g., market price service     │
│ Retry if failed (3x)           │
└────────────────┬───────────────┘
                 ↓
┌────────────────────────────────┐
│ Transform & cache response     │
│ TTL based on cache_ttl_seconds │
└────────────────┬───────────────┘
                 ↓
┌────────────────────────────────┐
│ Join external data with calc   │
│ data (e.g., price * quantity)  │
└────────────────┬───────────────┘
                 ↓
┌────────────────────────────────┐
│ Execute calculation with       │
│ enriched data                  │
└────────────────┬───────────────┘
                 ↓
┌────────────────────────────────┐
│ Return result + metadata       │
│ Includes external service info │
│ (response time, cache status)  │
└────────────────┬───────────────┘
                 ↓
┌────────────────────────────────┐
│ Frontend displays result +     │
│ service call metadata          │
└────────────────────────────────┘
```

---

## 🎬 Common Workflows

### Workflow 1: Test IRR Calculation (5 min)
```
1. Go to /fabric/calculations
2. Filter: Performance → IRR
3. Find "Investment XIRR" card
4. Click "Edit" button
5. Review formula
6. Click "Test" (when available)
7. Enter: cash_flows=[100, -50, 75], dates=[date1, date2, date3]
8. See result: 12.45%
```

### Workflow 2: Add Market Price Integration (15 min)
```
1. Go to /fabric/custom-components
2. Click "Add Component"
3. Select "API Integration"
4. Name: "Market Price Service"
5. Endpoint: https://api.example.com/prices
6. Add header: Authorization: Bearer KEY
7. Click "Test API"
8. Verify success response
9. Click "Save Component"
```

### Workflow 3: Use Market Prices in Calculation (10 min)
```
1. Go to /fabric/calculations
2. Click "Edit" on calculation
3. Modify formula to reference prices
4. Add external_services config
5. Map: tickers → investments.stock_ticker
6. Click "Save Changes"
7. Run test with sample data
```

### Workflow 4: Deploy to Production (2 hours)
```
1. Test all calculations locally
2. Test external service integrations
3. Verify caching strategy
4. Configure rate limiting
5. Set up monitoring alerts
6. Deploy backend (docker compose up)
7. Deploy frontend (npm run build)
8. Verify all endpoints working
9. Monitor logs for errors
```

---

## 🔐 Security Checklist

- [ ] API keys stored in environment variables
- [ ] HTTPS enforced for external APIs
- [ ] Rate limiting configured
- [ ] Input validation on all endpoints
- [ ] SQL injection prevention (parameterized queries)
- [ ] CORS headers configured
- [ ] Tenant isolation enforced
- [ ] Audit logging enabled
- [ ] Error messages don't expose sensitive info
- [ ] API response timeout set

---

## 📊 Monitoring Metrics

### Key Metrics to Track

```
Calculation Execution
├── Execution time (ms)
├── Error rate (%)
├── Cache hit rate (%)
└── Result accuracy (%)

External Service Calls
├── Response time (ms)
├── Availability (%)
├── Cache hit rate (%)
└── Retry rate (%)

System Health
├── Memory usage
├── Database connections
├── API rate limit usage
└── Backend latency (p50, p95, p99)
```

### Alert Thresholds

```
⚠️  Warning if:
├── Execution time > 5000ms
├── Error rate > 1%
├── Cache hit rate < 50%
├── External service response > 2000ms

🔴 Critical if:
├── Execution time > 30000ms
├── Error rate > 5%
├── Service unavailable
├── Memory usage > 80%
```

---

## 📚 Related Documentation

| Topic | Document | Link |
|-------|----------|------|
| Architecture | CALC_ENGINE_EXTENSIONS_GUIDE.md | See Part 1-2 |
| Implementation | CALC_TEST_BUTTON_IMPLEMENTATION.md | Full file |
| Integration | EXTERNAL_SERVICE_INTEGRATION_GUIDE.md | Full file |
| Reference | CALC_ENGINE_README.md | Overview |
| Quick Links | This file | You are here |

---

## 🎯 Success Metrics

### Functional Success
- [ ] Calculations execute correctly
- [ ] External services return data
- [ ] Results are cached appropriately
- [ ] Errors are handled gracefully

### Performance Success
- [ ] Calculation completes < 1s (simple)
- [ ] Calculation completes < 5s (complex)
- [ ] External service calls cached
- [ ] Batch calculations parallelized

### User Experience Success
- [ ] Test button visible and working
- [ ] Results displayed clearly
- [ ] Errors have helpful messages
- [ ] Documentation is accessible

---

## 🚀 Getting Started

### 5-Minute Setup
```bash
# 1. Navigate to Calculations Library
open http://localhost:5173/fabric/calculations

# 2. Find a calculation (e.g., IRR)
# 3. Click "Edit" to view formula
# 4. Close (changes not saved)
# 5. Done! You've explored a calculation
```

### 30-Minute Deep Dive
```bash
# 1. Add Custom Component for external data
open http://localhost:5173/fabric/custom-components

# 2. Create API Integration
# 3. Fill in external service URL
# 4. Test connectivity
# 5. Save component

# 6. Go back to Calculations Library
# 7. Create new calculation
# 8. Reference external component
# 9. Test calculation
```

### 2-Hour Full Implementation
```bash
# 1. Read: CALC_ENGINE_EXTENSIONS_GUIDE.md
# 2. Read: CALC_TEST_BUTTON_IMPLEMENTATION.md
# 3. Implement test button (1 hour)
# 4. Test implementation (30 min)
# 5. Deploy and verify (30 min)
```

---

## 🎓 Learning Resources

### Recommended Reading Order
1. Start: CALC_ENGINE_README.md (this provides overview)
2. Then: CALC_ENGINE_EXTENSIONS_GUIDE.md (understand architecture)
3. Then: EXTERNAL_SERVICE_INTEGRATION_GUIDE.md (real workflow)
4. Then: CALC_TEST_BUTTON_IMPLEMENTATION.md (code details)

### External Resources
- Financial calculations: https://en.wikipedia.org/wiki/Internal_rate_of_return
- API integration patterns: https://restfulapi.net/
- Performance optimization: https://12factor.net/

---

**Quick Links:**
- 🏠 [Home](./)
- 📖 [Full Documentation](./CALC_ENGINE_README.md)
- 🔧 [Architecture Guide](./CALC_ENGINE_EXTENSIONS_GUIDE.md)
- 💻 [Implementation Guide](./CALC_TEST_BUTTON_IMPLEMENTATION.md)
- 🌐 [Integration Guide](./EXTERNAL_SERVICE_INTEGRATION_GUIDE.md)

