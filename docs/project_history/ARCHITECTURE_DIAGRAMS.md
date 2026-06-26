# Architecture Diagram: AI Portfolio Rebalancer System

## System Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          CLIENT BROWSER                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                    FABRIC BUILDER APPLICATION                         │ │
│  ├───────────────────────────────────────────────────────────────────────┤ │
│  │                                                                       │ │
│  │  Navigation Menu                                                    │ │
│  │  ├─ Entity                                                          │ │
│  │  │  ├─ Scenario Analysis      → /analytics/scenario-analysis       │ │
│  │  │  ├─ Portfolio Rebalancer   → /analytics/rebalancer              │ │
│  │  │  └─ (other options)                                             │ │
│  │  └─ (other menus)                                                  │ │
│  │                                                                       │ │
│  └─────────────────┬─────────────────────────────────────────────────────┘ │
│                    │                                                       │
│                    ├─────────────────────────────────────────┐            │
│                    │                                         │            │
│  ┌─────────────────▼──────────────────┐    ┌────────────────▼──────┐   │
│  │  ScenarioAnalysisPro Component     │    │ AIPortfolioRebalancer │   │
│  │  (React + TypeScript)              │    │ Component             │   │
│  │  ├─ Portfolio Selector             │    │ (React + TypeScript)  │   │
│  │  ├─ Scenario Config Panel          │    │ ├─ SideNav            │   │
│  │  ├─ Results Visualization          │    │ ├─ Stats Cards        │   │
│  │  └─ Analysis History               │    │ ├─ Portfolio Grid     │   │
│  └──────────┬──────────────────────────┘    │ └─ Rebalance Modal    │   │
│             │                               └────────────┬──────────┘   │
│             └───────────────────┬────────────────────────┘               │
│                                 │                                        │
│        Apollo GraphQL Subscriptions & Fetch API                         │
│        setupTenantFetch.ts adds headers/params                          │
│                                 │                                        │
└─────────────────────────────────┼────────────────────────────────────────┘
                                  │
                    ┌─────────────▼─────────────┐
                    │   Tenant Fetch Middleware │
                    │ (setupTenantFetch.ts)     │
                    │ ├─ Add X-Tenant-ID        │
                    │ ├─ Add X-Tenant-Datasrc   │
                    │ └─ Add query params       │
                    └─────────────┬─────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        API GATEWAY (PORT 8080)                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Gin Web Framework + Middleware Stack                                       │
│  ├─ CORS Handler                                                           │
│  ├─ JWT Validation                                                         │
│  ├─ Request Logging                                                        │
│  ├─ Error Recovery                                                         │
│  └─ Rate Limiting                                                          │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │                    ROUTE HANDLERS                                    │  │
│  ├──────────────────────────────────────────────────────────────────────┤  │
│  │                                                                      │  │
│  │  scenario_analysis.go                                              │  │
│  │  ├─ POST /portfolio/:id/scenario                                   │  │
│  │  │  └─→ ExecuteWorkflow("ScenarioAnalysis")                        │  │
│  │  └─ [ABAC: "analyze" on "portfolio"]                              │  │
│  │                                                                      │  │
│  │  rebalancer.go (NEW)                                               │  │
│  │  ├─ POST /portfolio/:id/rebalance                                  │  │
│  │  │  └─→ ExecuteWorkflow("UMAAlpha")                                │  │
│  │  ├─ GET /rebalancer/portfolios                                     │  │
│  │  │  └─→ Fetch portfolio list from DB                              │  │
│  │  ├─ POST /portfolio/:id/propose-rebalance                          │  │
│  │  │  └─→ Return AI proposal                                         │  │
│  │  └─ [ABAC: "rebalance" on "portfolio"]                            │  │
│  │                                                                      │  │
│  │  risk_alpha.go, optimize_alpha.go (other routes)                   │  │
│  └────────────────┬─────────────────────────────────────────────────────┘  │
│                   │                                                        │
│                   └─────────────────┬───────────────────┐                 │
│                                     │                   │                 │
│        Authorization Check          │                   │                 │
│        abac.Evaluate(c, "action",   │                   │                 │
│                      "resource")    │                   │                 │
│                                     │                   │                 │
└─────────────────────────────────────┼───────────────────┼─────────────────┘
                                      │                   │
                          ┌───────────▼──────┐  ┌────────▼──────────┐
                          │ TEMPORAL CLIENT  │  │ GRAPHQL HASURA   │
                          │ (ExecuteWorkflow)│  │ (Data Updates)   │
                          └─────────┬────────┘  └──────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    TEMPORAL WORKFLOW ORCHESTRATION                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐  │
│  │ UMAAlpha Workflow (5s timeout)                                      │  │
│  ├─────────────────────────────────────────────────────────────────────┤  │
│  │                                                                     │  │
│  │ Input: RebalancePlan { portfolioId, drift, trades, ... }          │  │
│  │                                                                     │  │
│  │ Activities (in sequence):                                          │  │
│  │ 1. FetchUMAData(portfolioId)                                       │  │
│  │    └─→ Retrieve UMA account with holdings & tax lots              │  │
│  │                                                                     │  │
│  │ 2. AITaxHarvest(umaData, rebalancePlan)                            │  │
│  │    └─→ AI analysis for tax-loss harvesting opportunities          │  │
│  │                                                                     │  │
│  │ 3. ABACCheck(tenantId, "rebalance", "portfolio")                  │  │
│  │    └─→ Final authorization verification                           │  │
│  │                                                                     │  │
│  │ 4. ExecuteTrades(trades, umaData)                                  │  │
│  │    └─→ Execute proposed trades in broker system                   │  │
│  │                                                                     │  │
│  │ 5. HasuraUpdate(results)                                           │  │
│  │    └─→ Update portfolio & execution history in database           │  │
│  │                                                                     │  │
│  │ Output: { success, tradesSummary, newDrift, taxSaved, ... }       │  │
│  │                                                                     │  │
│  └─────────────────────────────────────────────────────────────────────┘  │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐  │
│  │ ScenarioAnalysis Workflow (10s timeout)                             │  │
│  ├─────────────────────────────────────────────────────────────────────┤  │
│  │ 1. FetchPortfolioData(portfolioId)                                  │  │
│  │ 2. ProjectScenario(portfolioData, scenario)                         │  │
│  │ 3. CalculateComparison(baseCase, scenarioCase)                      │  │
│  │ 4. StoreAnalysisResult(results)                                     │  │
│  │ Output: { baseCase, scenarioCase, comparison, insights, ... }      │  │
│  └─────────────────────────────────────────────────────────────────────┘  │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐  │
│  │ TaxHarvest, IndexAlpha, AttributionAlpha Workflows (similar)        │  │
│  └─────────────────────────────────────────────────────────────────────┘  │
│                                                                              │
└──────────┬──────────────────────────────────────────────────────────────────┘
           │
           ├─────────────────────────────────┬─────────────────────┐
           │                                 │                     │
           ▼                                 ▼                     ▼
┌──────────────────────┐   ┌────────────────────────┐   ┌──────────────────┐
│ POSTGRESQL DATABASE  │   │ HASURA GRAPHQL ENGINE  │   │ TEMPORAL SERVER  │
├──────────────────────┤   ├────────────────────────┤   ├──────────────────┤
│                      │   │                        │   │                  │
│ Tables:              │   │ Subscriptions:         │   │ Workflow History │
│ ├─ portfolios        │   │ ├─ portfolio_updates   │   │ Activity Logs    │
│ ├─ rebalance_plans   │   │ ├─ rebalance_status    │   │ Execution Stats  │
│ ├─ rebalance_history │   │ └─ analysis_results    │   │                  │
│ ├─ trades            │   │                        │   │ Task Queue:      │
│ ├─ scenario_results  │   │ Mutations:             │   │ "default"        │
│ ├─ uma_accounts      │   │ ├─ updatePortfolio     │   │                  │
│ └─ holdings          │   │ ├─ storeRebalance      │   │ Workers:         │
│                      │   │ └─ insertTrades        │   │ ├─ Activities    │
│ Indexes:             │   │                        │   │ └─ Workflows     │
│ ├─ portfolio_tenant  │   └────────────────────────┘   └──────────────────┘
│ ├─ drift_monitoring  │
│ └─ history_dates     │
│                      │
│ Data Volume:         │
│ ├─ Portfolios: 10k+  │
│ ├─ Trades: 1M+       │
│ └─ History: 10M+     │
│                      │
└──────────────────────┘

```

---

## Data Flow: Portfolio Rebalancing

```
                           USER ACTION
                                │
                    [Select Portfolio + Click Rebalance]
                                │
                                ▼
                    ┌────────────────────────┐
                    │  Generate RebalancePlan │
                    │  {portfolioId,drift,   │
                    │   trades,taxSavings}   │
                    └────────────┬───────────┘
                                │
                                ▼
                    ┌────────────────────────────────┐
                    │ POST /portfolio/:id/rebalance  │
                    │ + Headers + Query Params        │
                    └────────────┬───────────────────┘
                                │
                    ┌───────────▼──────────────┐
                    │  API Gateway Handler     │
                    ├──────────────────────────┤
                    │ 1. ABAC Authorization    │
                    │ 2. Validate RebalancePlan│
                    │ 3. Extract Tenant Scope  │
                    └───────────┬──────────────┘
                                │
                    ┌───────────▼──────────────┐
                    │ Temporal Client Execute  │
                    │ Workflow("UMAAlpha",     │
                    │   portfolioId,           │
                    │   rebalancePlan)         │
                    └───────────┬──────────────┘
                                │
                ┌───────────────┼───────────────┐
                │               │               │
                ▼               ▼               ▼
         ┌──────────┐    ┌──────────┐    ┌──────────┐
         │ Activity │    │ Activity │    │ Activity │
         │ Fetch    │───▶│ AITax    │───▶│ ABAC     │
         │ UMA Data │    │ Harvest  │    │ Check    │
         └──────────┘    └──────────┘    └────┬─────┘
                                              │
                                              ▼
                                         ┌──────────┐
                                         │ Activity │
                                         │ Execute  │
                                         │ Trades   │
                                         └────┬─────┘
                                              │
                                              ▼
                                         ┌──────────┐
                                         │ Activity │
                                         │ Hasura   │
                                         │ Update   │
                                         └────┬─────┘
                                              │
                                              ▼
                                        ┌─────────────┐
                                        │ DB Updated  │
                                        │ + Result    │
                                        │ Returned    │
                                        └──────┬──────┘
                                               │
                                               ▼
                                        ┌─────────────┐
                                        │ Frontend    │
                                        │ Shows       │
                                        │ Success/    │
                                        │ Error       │
                                        └─────────────┘
```

---

## Request Authentication Flow

```
┌──────────────────────────────────────────────────────────────────┐
│                    Frontend Makes Request                         │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  fetch('/api/portfolio/:id/rebalance', {                        │
│    method: 'POST',                                              │
│    headers: {                                                   │
│      'X-Tenant-ID': localStorage.selected_tenant_id,           │
│      'X-Tenant-Datasource-ID': localStorage.selected_datasrc,  │
│      'Authorization': 'Bearer ' + jwtToken                      │
│    },                                                            │
│    body: JSON.stringify(rebalancePlan)                          │
│  })                                                              │
│                                                                  │
│  + Query Parameters:                                            │
│  ?tenant_id=...&datasource_id=...&timestamp=...                │
│                                                                  │
└─────────────────────┬──────────────────────────────────────────┘
                      │
                      ▼
┌──────────────────────────────────────────────────────────────────┐
│                  API Gateway Middleware Chain                     │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. Request Validation                                          │
│     └─ Check Content-Type, Content-Length, etc.                 │
│                                                                  │
│  2. Extract Credentials                                         │
│     ├─ Read JWT from Authorization header                       │
│     ├─ Read X-Tenant-ID header                                  │
│     ├─ Read X-Tenant-Datasource-ID header                       │
│     └─ Read query parameters                                    │
│                                                                  │
│  3. JWT Validation                                              │
│     ├─ Verify signature with JWT_SECRET                         │
│     ├─ Check token expiration                                   │
│     ├─ Decode claims (sub, email, roles)                        │
│     └─ Attach to c.Get("claims")                                │
│                                                                  │
│  4. Tenant Scope Verification                                   │
│     ├─ Verify X-Tenant-ID == query tenant_id                   │
│     ├─ Verify X-Tenant-Datasource-ID == query datasource_id    │
│     ├─ Check user has access to tenant                          │
│     └─ Attach tenant context to request                         │
│                                                                  │
│  5. Route Dispatch                                              │
│     └─ Route to handler: rebalancer.go POST /portfolio/:id/...  │
│                                                                  │
└─────────────────────┬──────────────────────────────────────────┘
                      │
                      ▼
┌──────────────────────────────────────────────────────────────────┐
│              Route Handler: /portfolio/:id/rebalance             │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. Extract Context                                             │
│     ├─ tenantID := c.Query("tenant_id")                         │
│     ├─ datasourceID := c.Query("datasource_id")                 │
│     └─ claims := c.Get("claims")                                │
│                                                                  │
│  2. Parse Request Body                                          │
│     └─ c.ShouldBindJSON(&rebalancePlan)                         │
│                                                                  │
│  3. ABAC Authorization                                          │
│     └─ abac.Evaluate(c, "rebalance", "portfolio")               │
│        ├─ Check user attributes                                 │
│        ├─ Check resource attributes                             │
│        ├─ Apply ABAC policies                                   │
│        └─ Return bool (allowed/denied)                          │
│                                                                  │
│  4. Input Validation                                            │
│     ├─ Validate portfolioId format                              │
│     ├─ Validate drift values (0-100)                            │
│     ├─ Validate trades array structure                          │
│     └─ Sanitize inputs                                          │
│                                                                  │
│  5. Business Logic                                              │
│     ├─ Execute Temporal Workflow                                │
│     ├─ Wait for completion                                      │
│     └─ Return result or error                                   │
│                                                                  │
└────────────────────────────────────────────────────────────────┘
```

---

## Activity Execution Pipeline

```
Temporal UMAAlpha Workflow
         │
         ├─────────────────────────────────────────────────────┐
         │                                                     │
         ▼                                                     │
    Activity 1: FetchUMAData                                  │
    ├─ Input: portfolioId, tenantId                           │
    ├─ Processing:                                            │
    │  1. Query PostgreSQL: SELECT * FROM uma_accounts ...   │
    │  2. Fetch holdings: SELECT * FROM holdings ...         │
    │  3. Calculate metrics: aum, drift, lastRebalance       │
    │  4. Mock: Return realistic test data                   │
    └─ Output: UMAData {id, name, aum, holdings[], ...}     │
         │                                                     │
         ▼                                                     │
    Activity 2: AITaxHarvest                                 │
    ├─ Input: UMAData, rebalancePlan                          │
    ├─ Processing:                                            │
    │  1. Parse current holdings with cost basis              │
    │  2. Identify underperforming positions                  │
    │  3. Calculate tax-loss harvesting opportunities         │
    │  4. Mock: Return harvest strategy                       │
    └─ Output: HarvestPlan {losses, gains, savings}          │
         │                                                     │
         ▼                                                     │
    Activity 3: ABACCheck                                    │
    ├─ Input: tenantId, userId, "rebalance"                  │
    ├─ Processing:                                            │
    │  1. Call abac.Evaluate() with attributes               │
    │  2. Check policies (role-based, attribute-based)        │
    │  3. Verify user can execute trades on portfolio         │
    └─ Output: authorized: true/false                        │
         │                                                     │
         └─ if NOT authorized → Workflow FAIL               
         │
         ├─ if authorized → continue
         │
         ▼
    Activity 4: ExecuteTrades
    ├─ Input: trades[], UMAData, harvestPlan
    ├─ Processing:
    │  1. Connect to broker API (mock or real)
    │  2. Submit orders for SELL positions
    │  3. Submit orders for BUY positions
    │  4. Track execution: filled, partial, rejected
    │  5. Calculate costs: commissions, taxes, slippage
    └─ Output: TradeExecution {executed[], failed[], costs}
         │
         ▼
    Activity 5: HasuraUpdate
    ├─ Input: TradeExecution results, tenantId
    ├─ Processing:
    │  1. Call GraphQL mutation: updatePortfolio
    │  2. Insert records: INSERT INTO rebalance_history
    │  3. Insert records: INSERT INTO trades
    │  4. Update portfolio metrics
    └─ Output: updateSuccess: true/false
         │
         ▼
    Workflow Result
    ├─ Success: {
    │   status: "completed",
    │   tradeCount: 4,
    │   newDrift: 0.8,
    │   taxSaved: 1200,
    │   executionTime: "4.2s"
    │ }
    └─ Error: {
        status: "failed",
        reason: "authorization_denied",
        activity: "ABACCheck",
        timestamp: "2024-05-15T10:30:00Z"
      }
```

---

## Database Schema (Relevant Tables)

```
Portfolios Table
├─ id: UUID (PK)
├─ tenant_id: UUID (FK)
├─ client_id: UUID
├─ name: string
├─ aum: decimal
├─ drift_percentage: decimal
├─ last_rebalanced: timestamp
├─ status: enum (high-drift, moderate-drift, healthy)
└─ created_at: timestamp

Rebalance_History Table
├─ id: UUID (PK)
├─ portfolio_id: UUID (FK)
├─ tenant_id: UUID (FK)
├─ rebalance_date: timestamp
├─ old_drift: decimal
├─ new_drift: decimal
├─ tax_saved: decimal
├─ trade_count: integer
├─ execution_time_ms: integer
└─ status: enum (pending, completed, failed)

Trades Table
├─ id: UUID (PK)
├─ portfolio_id: UUID (FK)
├─ rebalance_history_id: UUID (FK)
├─ action: enum (BUY, SELL)
├─ symbol: string
├─ shares: decimal
├─ price: decimal
├─ total_value: decimal
├─ executed_at: timestamp
└─ status: enum (pending, filled, partial, rejected)

UMA_Accounts Table
├─ id: UUID (PK)
├─ portfolio_id: UUID (FK)
├─ account_name: string
├─ account_number: string
├─ aum: decimal
├─ cash: decimal
└─ last_sync: timestamp

Holdings Table
├─ id: UUID (PK)
├─ uma_account_id: UUID (FK)
├─ symbol: string
├─ shares: decimal
├─ cost_basis: decimal
├─ current_value: decimal
├─ unrealized_gain_loss: decimal
└─ holding_period: integer (days)
```

---

## Component Hierarchy

```
Frontend App (React)
│
├─ AppRoutes (Router + Layout)
│  ├─ MainNavigation (TopNav + SideNav)
│  │  ├─ EntityMenu (dropdown)
│  │  │  ├─ Scenario Analysis link
│  │  │  ├─ Portfolio Rebalancer link
│  │  │  └─ (other menu items)
│  │  ├─ FabricMenu (dropdown)
│  │  ├─ AdminMenu (dropdown)
│  │  └─ CoreMenu (dropdown)
│  │
│  ├─ Route: /analytics/scenario-analysis
│  │  └─ ScenarioAnalysisPro
│  │     ├─ PortfolioSelector
│  │     ├─ ScenarioConfigPanel
│  │     ├─ ResultsDisplay
│  │     │  ├─ BaseCase (Gauge components)
│  │     │  ├─ ScenarioCase (Gauge components)
│  │     │  └─ Comparison Metrics
│  │     └─ AnalysisHistorySidebar
│  │
│  ├─ Route: /analytics/rebalancer
│  │  └─ AIPortfolioRebalancer
│  │     ├─ SideNav
│  │     ├─ StatsCards (AUM, Drift, TaxSaved)
│  │     ├─ PortfolioGrid
│  │     │  └─ PortfolioCard[] (drift + rebalance button)
│  │     └─ RebalanceModal (conditional)
│  │        ├─ PlanDetails
│  │        ├─ TradesList
│  │        └─ ExecuteButton
│  │
│  └─ ProtectedRoute (auth wrapper)
│
├─ TenantContext
│  ├─ selected_tenant
│  ├─ selected_product
│  └─ selected_datasource
│
├─ Apollo Client (GraphQL)
│  ├─ Subscriptions (real-time updates)
│  ├─ Queries (portfolio data)
│  └─ Mutations (updates)
│
└─ Middleware: setupTenantFetch.ts
   └─ Patches window.fetch for tenant scoping
```

---

**Document Version**: 1.0
**Last Updated**: May 2024
**Status**: Complete ✅
