# Calculation Engine: Complete Reference

## 📋 Documentation Index

This folder contains comprehensive guides for working with the Calculation Engine in Semlayer:

### 1. **CALC_ENGINE_EXTENSIONS_GUIDE.md** 
   - **What it covers:** Architecture overview, testing, external services integration
   - **Best for:** Understanding the big picture
   - **Read time:** 15 minutes
   - **Topics:**
     - Testing calculations (current + enhancements)
     - Custom functions & transformations
     - External service architecture
     - Practical IRR example with market prices
     - Implementation roadmap

### 2. **CALC_TEST_BUTTON_IMPLEMENTATION.md**
   - **What it covers:** Step-by-step code implementation
   - **Best for:** Developers who want to add test functionality
   - **Read time:** 10 minutes
   - **Topics:**
     - Add "Test" button to calculation cards
     - Create test dialog UI
     - Handle test execution and results
     - File locations and line numbers
     - Testing checklist

### 3. **EXTERNAL_SERVICE_INTEGRATION_GUIDE.md**
   - **What it covers:** Complete workflow for external service integration
   - **Best for:** DevOps/integration engineers
   - **Read time:** 20 minutes
   - **Topics:**
     - Step-by-step Custom Components setup
     - API configuration examples
     - Test workflows
     - Common external services (stocks, risk, fx)
     - Monitoring and troubleshooting
     - Security best practices

---

## 🚀 Quick Start: 3 Scenarios

### Scenario 1: I want to test a calculation directly
→ **Read:** CALC_TEST_BUTTON_IMPLEMENTATION.md

**Time:** 2 hours to implement
**Complexity:** Easy
**Code changes:** 1 file (CalculationsLibraryPage.tsx)

### Scenario 2: I need to fetch current stock prices for IRR
→ **Read:** EXTERNAL_SERVICE_INTEGRATION_GUIDE.md (Part 1-3)

**Time:** 30 minutes to setup
**Complexity:** Medium
**UI changes:** Custom Components page

### Scenario 3: I want to add a new calculation type with external data
→ **Read:** All 3 guides in this order

**Time:** 1-2 days end-to-end
**Complexity:** Hard
**Code changes:** Multiple files

---

## 🔄 Recommended Implementation Order

### Phase 1: Foundation (Week 1)
- ✅ **Done:** Calculations Library functional
- ✅ **Done:** Custom Components framework in place
- 📋 **TODO:** Add Test button to calculation cards
- 📋 **TODO:** Create example external service integration

### Phase 2: Integration (Week 2)
- 📋 **TODO:** Set up market price service integration
- 📋 **TODO:** Add caching layer for external calls
- 📋 **TODO:** Implement error handling & retries
- 📋 **TODO:** Add monitoring & logging

### Phase 3: Advanced (Week 3+)
- 📋 **TODO:** Build dedicated risk factors API
- 📋 **TODO:** Implement real-time data streaming (WebSocket)
- 📋 **TODO:** Create calculation validation framework
- 📋 **TODO:** Add performance optimization tools

---

## 🎯 Key Concepts

### Calculation Types

| Type | Example | Uses |
|------|---------|------|
| **Atomic** | SUM, AVG, COUNT | Basic aggregations |
| **Derived** | P/E Ratio = Price / EPS | Ratio calculations |
| **Complex** | IRR, Variance, Covariance | Financial metrics |
| **With External** | IRR + Market Prices | Real-time enhanced |

### Integration Layers

```
┌─────────────────────────────┐
│   React Components (UI)      │
│   - Calculations Library     │
│   - Custom Components        │
└──────────────┬──────────────┘
               │
┌──────────────▼──────────────┐
│   Frontend API Clients       │
│   - /api/calc/run           │
│   - /api/custom-components  │
└──────────────┬──────────────┘
               │
┌──────────────▼──────────────┐
│   Backend Services           │
│   - Dispatcher               │
│   - External Service Calls   │
│   - Data Transformation      │
│   - Caching Layer            │
└──────────────┬──────────────┘
               │
┌──────────────▼──────────────┐
│   External Services          │
│   - Stock Prices API         │
│   - Risk Factors API         │
│   - Exchange Rates API       │
└─────────────────────────────┘
```

### Data Flow: Simple Calculation
```
User Input
    ↓
Calculation Config
    ↓
Backend /api/calc/run
    ↓
SQL Execution
    ↓
Result Formatting
    ↓
User Display
```

### Data Flow: With External Service
```
User Input
    ↓
Calculation Config + External Service Config
    ↓
Backend /api/calc/run
    ↓
Fetch External Data (with retry/cache)
    ↓
Transform & Join Data
    ↓
SQL Execution
    ↓
Result Formatting (+ Service Metadata)
    ↓
User Display
```

---

## 📂 File Structure

```
frontend/src/
├── features/fabric/pages/
│   └── CalculationsLibraryPage.tsx (4-5 hours to add test button)
├── components/
│   ├── CustomComponentManager/
│   │   └── CustomComponentManager.tsx (already has UI)
│   └── UnifiedSemanticBuilder/
│       └── financialCalculations.ts (calculation definitions)

backend/internal/
├── api/
│   ├── api.go (line 4618: runPreview function)
│   └── custom_components.go (API integration endpoints)
├── services/
│   └── calculation_dispatcher.go (NEW: for external services)
└── handlers/
    └── custom_component_handler.go (already implemented)
```

---

## 🔧 Configuration Examples

### Market Price Integration
```json
{
  "name": "Stock Prices",
  "type": "api_integration",
  "config": {
    "apiEndpoint": "https://api.example.com/prices",
    "method": "POST",
    "headers": {"Authorization": "Bearer KEY"},
    "refreshInterval": 300,
    "cacheStrategy": "short"
  }
}
```

### Risk Factors Integration
```json
{
  "name": "Risk Factors",
  "type": "api_integration",
  "config": {
    "apiEndpoint": "https://risk-api.company.com/factors",
    "method": "POST",
    "headers": {"Authorization": "Bearer KEY"},
    "refreshInterval": 600,
    "cacheStrategy": "long"
  }
}
```

### Internal Transformation
```json
{
  "name": "Data Transformer",
  "type": "custom_code",
  "config": {
    "jsCode": "return data.map(d => ({...d, adjusted: d.value * 1.1}))"
  }
}
```

---

## 🌐 API Endpoints Reference

### Calculation Endpoints
```
POST /api/calc/run
  - Execute single calculation
  - Input: FinancialCalc
  - Response: { result: {...} }

POST /api/calc/vectorized
  - Execute batch calculations
  - Input: { metrics, entities }
  - Response: { results: [...], batch_info: {...} }
```

### Custom Components Endpoints
```
GET  /api/custom-components
POST /api/custom-components
GET  /api/custom-components/{id}
PUT  /api/custom-components/{id}
DELETE /api/custom-components/{id}
POST /api/custom-components/test-api
GET  /api/custom-components/export
POST /api/custom-components/import
```

### External Service Endpoints (To Implement)
```
POST /api/calc/run-with-context
  - Execute with external services
  - Input: CalculationRequest + ExternalServiceCall[]
  
GET  /api/calc/test
  - Validate calculation
  - Input: CalculationOption
  
POST /api/services/cache/invalidate
  - Clear service cache
  - Input: { service_name, ttl_seconds }
```

---

## 🎓 Learning Paths

### Path 1: Frontend Developer
**Goal:** Add test functionality to UI
**Time:** 2-3 hours
**Read:** CALC_TEST_BUTTON_IMPLEMENTATION.md
**Skills:** React, TypeScript, MUI, API calls
**Deliverable:** Test button + test dialog

### Path 2: Backend Developer
**Goal:** Support external services in calculations
**Time:** 4-6 hours
**Read:** CALC_ENGINE_EXTENSIONS_GUIDE.md (Part 3-4)
**Skills:** Go, HTTP, data transformation, caching
**Deliverable:** Enhanced /api/calc/run endpoint

### Path 3: DevOps/Integration Engineer
**Goal:** Set up external service integrations
**Time:** 30 mins - 2 hours per service
**Read:** EXTERNAL_SERVICE_INTEGRATION_GUIDE.md
**Skills:** API configuration, testing, monitoring
**Deliverable:** Working custom components

### Path 4: Full Stack
**Goal:** Complete implementation
**Time:** 1 week
**Read:** All 3 guides
**Skills:** Full stack development
**Deliverable:** Complete calculation engine with external services

---

## ✅ Testing Checklist

### Before Deployment
- [ ] Calculation formulas validated
- [ ] External services connectivity tested
- [ ] Error handling working
- [ ] Cache invalidation working
- [ ] Rate limiting configured
- [ ] Security headers set
- [ ] API keys secured
- [ ] Logging enabled
- [ ] Monitoring alerts set up
- [ ] Load testing completed

### Testing Scenarios
- [ ] Basic arithmetic calculations
- [ ] Complex financial calculations (IRR, MIRR)
- [ ] Calculations with external data
- [ ] Failed external service recovery
- [ ] Cache hit/miss verification
- [ ] Rate limit handling
- [ ] Concurrent calculation execution
- [ ] Large batch processing

---

## 🐛 Common Issues & Solutions

| Issue | Root Cause | Fix |
|-------|-----------|-----|
| "API unreachable" | Wrong endpoint URL | Verify Custom Component config |
| "401 Unauthorized" | Invalid API key | Check authorization header |
| "Calculation timeout" | Slow external service | Increase timeout, add caching |
| "Out of memory" | Large dataset | Use batch processing |
| "Cache stale" | TTL too high | Lower refresh interval |
| "Test button not showing" | Code not deployed | Rebuild frontend |

---

## 📞 Support & Resources

### Documentation Files
- `CALC_ENGINE_EXTENSIONS_GUIDE.md` - Architecture & concepts
- `CALC_TEST_BUTTON_IMPLEMENTATION.md` - Code implementation
- `EXTERNAL_SERVICE_INTEGRATION_GUIDE.md` - Integration workflow

### Code Files
- `frontend/src/features/fabric/pages/CalculationsLibraryPage.tsx` - Main UI
- `backend/internal/api/api.go` - API server
- `backend/internal/api/custom_components.go` - Component management

### External References
- Alpha Vantage: https://www.alphavantage.co/
- Open Exchange Rates: https://openexchangerates.org/
- IEX Cloud: https://iexcloud.io/

---

## 🎉 Success Criteria

✅ **Phase 1 Complete When:**
- [ ] Test button visible on calculation cards
- [ ] Sample calculation can be tested from UI
- [ ] Results display correctly

✅ **Phase 2 Complete When:**
- [ ] Custom component can fetch external data
- [ ] Data is cached appropriately
- [ ] Errors are handled gracefully

✅ **Phase 3 Complete When:**
- [ ] Calculation uses external data automatically
- [ ] Results include service metadata
- [ ] Performance is optimized

---

## 🚢 Deployment Checklist

Before going to production:

1. **Security**
   - [ ] API keys rotated
   - [ ] HTTPS enforced
   - [ ] Rate limits set
   - [ ] Input validation on

2. **Performance**
   - [ ] Caching enabled
   - [ ] Database indexes created
   - [ ] Queries optimized
   - [ ] Load testing passed

3. **Monitoring**
   - [ ] Metrics exported
   - [ ] Alerts configured
   - [ ] Logs centralized
   - [ ] Dashboards created

4. **Documentation**
   - [ ] API documented
   - [ ] Configuration guide written
   - [ ] Troubleshooting guide prepared
   - [ ] Team trained

---

**Generated:** November 2, 2024
**Last Updated:** November 2, 2024
**Status:** Complete - Ready for Implementation

