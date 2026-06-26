# Portfolio What-If Simulation Contract

This document defines the canonical, gold-standard contract for the **Portfolio** Business Object within the What-If Simulation Engine.

## 1. Portfolio BO Definition
A Portfolio is a governed, multi-tenant business object representing a client or strategy’s investment holdings.

### Core Attributes
- `portfolioId`
- `tenantId`
- `name`
- `strategyId`
- `benchmarkId`
- `riskProfile`
- `targetWeights` (Asset Class, Sector, Region, Instrument)
- `constraints` (Min/Max, Liquidity, ESG, Suitability)
- `positions[]` (List of Position BOs)
- `cashBalance`
- `valuationDate`

### Relationships (Graph)
- `Portfolio` -> `Positions`
- `Portfolio` -> `Client`
- `Portfolio` -> `Strategy`
- `Portfolio` -> `Benchmark`
- `Portfolio` -> `Risk Model`
- `Portfolio` -> `Compliance Rules`

## 2. Delta Contract
A Portfolio What-If scenario is expressed as one or more deltas.

### A. POSITION_DELTA
Modify a specific position.
```json
{
  "deltaType": "POSITION_DELTA",
  "boId": "position:TSLA",
  "changes": { "quantityPct": -0.30, "price": 250.00 }
}
```

### B. PORTFOLIO_DELTA
Modify the portfolio structure.
```json
{
  "deltaType": "PORTFOLIO_DELTA",
  "boId": "portfolio:123",
  "changes": {
    "rebalance": "TO_TARGET_WEIGHTS", // or TO_NEW_TARGET, SHIFT_ALLOCATION
    "constraints": { "capSector": { "TECH": 0.20 } }
  }
}
```

### C. MARKET_DELTA
Shock market conditions.
```json
{
  "deltaType": "MARKET_DELTA",
  "boId": "market:rates",
  "changes": { "parallelShiftBps": 50, "equitiesShockPct": -0.10 }
}
```

### D. CLIENT_DELTA
Modify client-level assumptions.
```json
{
  "deltaType": "CLIENT_DELTA",
  "boId": "client:991",
  "changes": { "deposit": 2000000, "riskProfile": "AGGRESSIVE" }
}
```

## 3. Metrics Required
Every portfolio simulation must calculate:

- **NAV & Return**: NAV, P&L, IRR, Yield.
- **Risk**: VaR (95%, 99%), Volatility, Drawdown, Factor Exposures.
- **Structure**: Sector/Region/Asset Weights, Concentration.
- **Liquidity**: Days to Liquidate, % Illiquid.
- **ESG**: ESG Score, Restricted Exposure.
- **Client Fit**: Suitability Score, Risk Profile Alignment.
- **Compliance**: Concentration, Liquidity, ESG, Regulatory Status.

## 4. NL -> Intent Contract
**Intent**: `PORTFOLIO_SIMULATION`

### Schema
```json
{
  "scenarioType": "PORTFOLIO_SIMULATION",
  "portfolioId": "portfolio:123",
  "deltas": [ ... ],
  "metrics": [ "NAV", "VAR_95", "SECTOR_WEIGHTS" ],
  "constraints": { ... },
  "explain": "Rebalance portfolio 123 to target weights."
}
```

## 5. Simulation Engine Pipeline
1.  **Snapshot**: Fetch Baseline (Portfolio, Positions, Market, Risk, Constraints).
2.  **Apply Deltas**: Sandbox State (Adjust Positions, Weights, Market, Client).
3.  **Recalculate**: Call Calculation, Risk, and Compliance Engines.
4.  **Aggregate**: Metrics, Compliance Flags, Narrative.
5.  **Store**: Persist Scenario, Deltas, Results.

## 6. Simulation Result Contract
```json
{
  "scenarioId": "scn-991",
  "portfolioId": "portfolio:123",
  "summary": {
    "navDelta": -125000.50,
    "var95Delta": -350000.00,
    "complianceStatus": "RESOLVED"
  },
  "metrics": [
    { "boId": "portfolio:123", "metricName": "NAV", "deltaValue": -125000.0 }
  ],
  "compliance": {
    "newIssues": [],
    "resolvedIssues": [],
    "changedIssues": []
  },
  "impactedEntities": [ "client:991", "strategy:balanced" ],
  "narrative": "Rebalancing reduces NAV by 1.25% but improves VaR by 14%."
}
```

## 7. Governance Contract
- **Checks**: Concentration, Liquidity, Suitability, ESG, Regulatory.
- **ChangeSet Conversion**:
    - Creation -> Review -> Approval -> Execution.
    - Audit Trail (Who, What, Why).

## 8. Next-Best Actions (AI)
- "Compare to baseline"
- "Compare to another scenario"
- "Convert to ChangeSet"
- "Generate API for this scenario"
- "Run stress test"
