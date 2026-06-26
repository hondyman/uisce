package nl_intelligence

import "fmt"

const (
	PromptIntentClassification = `
### ROLE: SEMANTIC INTELLIGENCE CLASSIFIER

You are the orchestration brain of a premium semantic data layer. Your goal is to accurately classify user natural language questions into specific action categories.

### INTENTS:
- **SQL_QUERY**: Analytical questions about metrics, counts, lists, or aggregates (e.g., "how many jobs failed yesterday?").
- **GRAPH_QUERY**: Questions about relationships, lineages, dependencies, or impact analysis (e.g., "what nodes are downstream of the trade-aggregator?").
- **LINEAGE_EXPLANATION**: Requests for a narrative explaining data flow.
- **INCIDENT_EXPLANATION**: Deep forensic requests asking *why* an error occurred or for root-cause analysis.
- **CHANGESET_GENERATION**: Requests for code or configuration proposals to fix a detected problem.
- **COMPLIANCE_ANALYSIS**: Questions regarding data residency, PII leakage, or governance policy violations.
- **FORECASTING**: Predictive analysis regarding failure probabilities or future bottlenecks.
- **API_DESIGN**: High-level requests to architect and build new API endpoints.

### OUTPUT FORMAT:
Return ONLY a valid JSON object:
{
  "intent": "INTENT_NAME",
  "entities": [{"type": "EntityClass", "value": "Name"}],
  "filters": {
    "status": "active/failed/etc",
    "timeRange": {"from": "ISO_DATE", "to": "ISO_DATE"}
  },
  "reasoning_steps": ["Step 1: Analyzed intent...", "Step 2: ..."]
}

### USER QUESTION:
"%s"
`

	PromptCypherGenerator = `
### ROLE: GRAPH INTELLIGENCE ARCHITECT

You generate valid, high-performance Cypher queries for an Apache AGE graph database.

### SCHEMA CONTEXT:
- **META (catalog_node)**: Represents static metadata (SEMANTIC_TERM, BUSINESS_OBJECT, DAG, JOB, API).
- **OPS (ops_node)**: Represents dynamic events (JOB_RUN, DAG_RUN, INCIDENT, CHANGESET_EVENT).
- **RELATIONSHIPS**: HAS_IMPACT_ON, RUNS_JOB, HAS_SEMANTIC_CONTEXT, CAUSES, DEPENDS_ON.

### RULES:
1. **CHAIN OF THOUGHT**: Before the query, think step-by-step in a comment.
2. **TENANT ISOLATION**: Mandatory filter: MATCH (n)-[:HAS_TENANT]->(t {id: $tenantId}).
3. **JSONB ACCESS**: Use properties(n)->>'field_name' for node attributes.
4. **LABELS**: Use :META or :OPS labels to speed up traversal.

### EXECUTION STEPS:
1. Identify the starting nodes (Entities: %v).
2. Apply mandatory filters (Filters: %v, Tenant: %v).
3. Traverse relationships to reach the target information.

### USER QUESTION:
"%s"

### OUTPUT:
Provide the thinking process in a comment, followed by ONLY the Cypher query.
`

	PromptSQLGenerator = `
### ROLE: ANALYTICAL SQL ENGINE

You generate precision SQL for Trino/Postgres, targeting the SemLayer internal metadata schemas.

### TABLES:
- **semantic.api_endpoints**: API definitions and health.
- **semantic.catalog_nodes**: Semantic entities and definitions.
- **ops.job_runs**: Execution logs and performance metrics.
- **ops.incidents**: Forensic anomaly logs.

### RULES:
1. **CHAIN OF THOUGHT**: Think through the table joins and filters in a SQL comment block.
2. **GOVERNANCE**: Strict tenant filtering using: WHERE tenant_id = ANY($tenantScope).
3. **JSONB**: Use Postgres-standard ->> operator for properties access.

### INPUT:
Question: "%s"
Entities: %v
Filters: %v
TenantScope: %v

### OUTPUT:
Provide a step-by-step thinking block in a comment, then the SQL query.
`
)

func BuildIntentPrompt(question string) string {
	return fmt.Sprintf(PromptIntentClassification, question)
}

func BuildCypherPrompt(question string, entities []Entity, filters Filters, tenantScope []string) string {
	tmpl := PromptCypherGenerator
	// Order arguments to match placeholders in the template
	return fmt.Sprintf(tmpl, entities, filters, tenantScope, question)
}

func BuildSQLPrompt(question string, entities []Entity, filters Filters, tenantScope []string) string {
	tmpl := PromptSQLGenerator
	return fmt.Sprintf(tmpl, question, entities, filters, tenantScope)
}

func BuildSimulationPrompt(question string, entities []Entity, filters Filters, tenantScope []string) string {
	tmpl := PromptSimulationGenerator
	return fmt.Sprintf(tmpl, question, entities, filters, tenantScope)
}

const (
	PromptSimulationGenerator = `
### ROLE: FINANCIAL SIMULATION ARCHITECT

You are the simulation brain of a premium asset management platform. 
Your goal is to translate natural language questions into a structured "What-If" simulation plan.

### SIMULATION TYPES:
1. **PORTFOLIO_SIMULATION**: Rebalancing, shifting allocations, adding/removing assets.
2. **POSITION_SIMULATION**: Changing quantities, overriding prices/FX of specific holdings.
3. **MARKET_SIMULATION**: Shocks to rates (parallel/twist), spreads, equity markets, volatility, or FX.
4. **CLIENT_SIMULATION**: Deposits, withdrawals, fee changes, risk profile updates.
5. **STRATEGY_SIMULATION**: Changing target model weights or constraints.
6. **INSTRUMENT_SIMULATION**: Idiosyncratic shocks to a single security (e.g., "What if AAPL drops 20%%?").

### RULES:
1. **Determine scenarioType**: Based on the primary intent.
2. **Identify Primary BO**: E.g., 'portfolio:123', 'position:TSLA'.
3. **Build Deltas**:
   - **POSITION_DELTA**: { "boId": "...", "changes": { "quantityPct": -0.5, "priceOverride": 150 } }
   - **PORTFOLIO_DELTA**: { "boId": "...", "changes": { "rebalance": "TO_TARGET_WEIGHTS", "shiftAllocation": { ... } } }
   - **MARKET_DELTA**: { "boId": "market:rates", "changes": { "parallelShiftBps": 50, "equityShockPct": -0.10 } }
   - **CLIENT_DELTA**: { "boId": "...", "changes": { "deposit": 1000000 } }
4. **Select Metrics**: Always include NAV, VAR_95, VOLATILITY. Add SECTOR_WEIGHTS, LIQUIDITY, ESG, SUITABILITY contextually.
5. **Constraints**: Extract explicit constraints (e.g., "avoid short term gains", "exclude fossil fuels").

### INPUT:
Question: "%s"
Entities: %v
Filters: %v
TenantScope: %v

### OUTPUT FORMAT:
Return ONLY a valid JSON object matching the SimulationPlan schema:

Example 1 (Rebalance):
{
  "scenarioType": "PORTFOLIO_SIMULATION",
  "primaryBoId": "portfolio:123",
  "name": "Rebalance to Target",
  "description": "Rebalance portfolio 123 to target weights",
  "deltas": [
    {
      "deltaType": "PORTFOLIO_DELTA",
      "boId": "portfolio:123",
      "changes": { "rebalance": "TO_TARGET_WEIGHTS" }
    }
  ],
  "metrics": ["NAV", "VAR_95", "SECTOR_WEIGHTS", "LIQUIDITY"],
  "constraints": { "avoidShortTermGains": true },
  "explain": "Rebalance portfolio 123 to its target weights, avoiding short-term capital gains."
}

Example 2 (Market Shock):
{
  "scenarioType": "MARKET_SIMULATION",
  "name": "Rates +50bps, Equities -10%",
  "description": "Simulate rate hike and equity sell-off",
  "deltas": [
    {
      "deltaType": "MARKET_DELTA",
      "boId": "market:global",
      "changes": { "parallelShiftBps": 50, "equityShockPct": -0.10 }
    }
  ],
  "metrics": ["NAV", "VAR_95", "VOLATILITY"],
  "explain": "Simulate a scenario where interest rates rise by 50bps and equity markets fall by 10%%."
}
`

	PromptExplanationGenerator = `
### ROLE: PORTFOLIO ANALYST (EXPLAINER)
You explain simulation results to business users (PMs, Risk Managers).

### INPUTS:
- Scenario Type: %s
- Summary: %s (JSON)
- Compliance Issues: %s (JSON)

### GOAL:
1. Summarize key changes (e.g., "NAV decreased by $1.2M (-1.5%%)").
2. Highlight tradeoffs ("Risk (VaR) reduced by 14%% at the cost of lower return").
3. Mention compliance ("Sector cap constraint is now satisfied").
4. No investment advice. Use professional, neutral tone.

### OUTPUT:
Return a single paragraph of natural language.
`
)

func BuildExplanationPrompt(scenarioType string, summaryJSON string, issuesJSON string) string {
	return fmt.Sprintf(PromptExplanationGenerator, scenarioType, summaryJSON, issuesJSON)
}
