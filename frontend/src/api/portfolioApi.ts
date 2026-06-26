// Portfolio API Client
// Endpoints for Portfolio detail pages

const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080/api';

// Types
export interface PortfolioOverview {
  portfolio_id: string;
  name: string;
  aum: number;
  currency: string;
  strategy: string;
  benchmark_id: string;
  valuation_date: string;
}

export interface Holding {
  security_id: string;
  name: string;
  weight: number;
  sector?: string;
  country?: string;
  price?: number;
  change_pct?: number;
}

export interface HoldingsSummary {
  top_positions: Holding[];
  sector_weights: Array<{ sector: string; weight: number }>;
  country_weights: Array<{ country: string; weight: number }>;
}

export interface FactorExposure {
  factor_id: string;
  exposure: number;
}

export interface Scenario {
  scenario_id: string;
  name: string;
  pnl: number;
}

export interface RiskSnapshot {
  total_volatility: number;
  var_95: number;
  var_99: number;
  factor_exposures: FactorExposure[];
  worst_scenarios: Scenario[];
}

export interface RuleBreachDetail {
  rule_code: string;
  metric_value: number;
  threshold_value: number;
  severity?: 'hard' | 'soft';
}

export interface ComplianceSnapshot {
  rules_evaluated: number;
  pass_rate: number;
  hard_breaches: RuleBreachDetail[];
  soft_breaches: RuleBreachDetail[];
}

export interface ScenarioResult {
  scenario_id: string;
  name: string;
  pnl: number;
  impact_pct?: number;
}

export interface ScenarioResults {
  results: ScenarioResult[];
}

// Portfolio Overview
export async function fetchPortfolioOverview(
  portfolioId: string,
  valuationDate: string
): Promise<PortfolioOverview> {
  const params = new URLSearchParams({ valuation_date: valuationDate });
  const response = await fetch(
    `${API_BASE_URL}/portfolios/${portfolioId}/overview?${params}`,
    {
      headers: { 'Content-Type': 'application/json' },
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to fetch portfolio overview: ${response.statusText}`);
  }

  return response.json();
}

// Holdings Summary
export async function fetchPortfolioHoldings(
  portfolioId: string,
  valuationDate: string
): Promise<HoldingsSummary> {
  const params = new URLSearchParams({ valuation_date: valuationDate });
  const response = await fetch(
    `${API_BASE_URL}/portfolios/${portfolioId}/holdings?${params}`,
    {
      headers: { 'Content-Type': 'application/json' },
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to fetch holdings: ${response.statusText}`);
  }

  return response.json();
}

// Risk Snapshot
export async function fetchPortfolioRisk(
  portfolioId: string,
  valuationDate: string
): Promise<RiskSnapshot> {
  const params = new URLSearchParams({ valuation_date: valuationDate });
  const response = await fetch(
    `${API_BASE_URL}/portfolios/${portfolioId}/risk?${params}`,
    {
      headers: { 'Content-Type': 'application/json' },
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to fetch risk snapshot: ${response.statusText}`);
  }

  return response.json();
}

// Compliance Snapshot
export async function fetchPortfolioCompliance(
  portfolioId: string,
  valuationDate: string
): Promise<ComplianceSnapshot> {
  const params = new URLSearchParams({ valuation_date: valuationDate });
  const response = await fetch(
    `${API_BASE_URL}/portfolios/${portfolioId}/compliance?${params}`,
    {
      headers: { 'Content-Type': 'application/json' },
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to fetch compliance snapshot: ${response.statusText}`);
  }

  return response.json();
}

// Scenario Results
export async function fetchPortfolioScenarios(
  portfolioId: string,
  valuationDate: string
): Promise<ScenarioResults> {
  const params = new URLSearchParams({ valuation_date: valuationDate });
  const response = await fetch(
    `${API_BASE_URL}/portfolios/${portfolioId}/scenarios?${params}`,
    {
      headers: { 'Content-Type': 'application/json' },
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to fetch scenarios: ${response.statusText}`);
  }

  return response.json();
}
