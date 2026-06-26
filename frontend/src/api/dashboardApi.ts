// Dashboard API Client
// Endpoints for Risk & Compliance Dashboard

const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080/api';

// Types for dashboard data
export interface ComplianceKPIData {
  total_rules: number;
  pass_rate: number;
  hard_breaches: number;
  soft_breaches: number;
  top_failing_rules: Array<{
    rule_code: string;
    failures: number;
  }>;
}

export interface RiskKPIData {
  avg_volatility: number;
  avg_var_95: number;
  avg_var_99: number;
  worst_scenario: {
    scenario_id: string;
    name: string;
    pnl: number;
  };
  top_factors: Array<{
    factor_id: string;
    contribution: number;
  }>;
}

export interface SparklinePoint {
  date: string;
  value: number;
}

export interface SparklinesData {
  pass_rate: SparklinePoint[];
  hard_breaches: SparklinePoint[];
  volatility: SparklinePoint[];
  etl_duration: SparklinePoint[];
}

export interface ETLHealthData {
  last_run: {
    etl_run_id: string;
    status: 'SUCCESS' | 'FAILED' | 'RUNNING';
    duration_ms: number;
    rules_evaluated: number;
    scenarios_evaluated: number;
    wasm_version: string;
  };
}

export interface Alert {
  id?: string;
  type: 'hard_breach' | 'scenario_loss' | 'etl_failure' | 'soft_breach' | 'reg_breach';
  severity: 'error' | 'warning' | 'info';
  title: string;
  description: string;
  timestamp?: string;
  rule_code?: string;
  portfolio_id?: string;
  scenario_id?: string;
  scenario_name?: string;
  pnl?: number;
  metric?: number;
}

export interface AlertsData {
  hard_breaches: Alert[];
  scenario_losses: Alert[];
  etl_failures: Alert[];
  soft_breaches?: Alert[];
  reg_breaches?: Alert[];
}

// Compliance KPIs
export async function fetchComplianceKPIs(
  tenantId: string,
  valuationDate: string
): Promise<ComplianceKPIData> {
  const params = new URLSearchParams({
    tenant_id: tenantId,
    valuation_date: valuationDate,
  });

  const response = await fetch(
    `${API_BASE_URL}/dashboard/compliance?${params}`,
    {
      headers: {
        'Content-Type': 'application/json',
      },
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to fetch compliance KPIs: ${response.statusText}`);
  }

  return response.json();
}

// Risk KPIs
export async function fetchRiskKPIs(
  tenantId: string,
  valuationDate: string
): Promise<RiskKPIData> {
  const params = new URLSearchParams({
    tenant_id: tenantId,
    valuation_date: valuationDate,
  });

  const response = await fetch(
    `${API_BASE_URL}/dashboard/risk?${params}`,
    {
      headers: {
        'Content-Type': 'application/json',
      },
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to fetch risk KPIs: ${response.statusText}`);
  }

  return response.json();
}

// Sparklines (7-day trend data)
export async function fetchSparklines(
  tenantId: string
): Promise<SparklinesData> {
  const params = new URLSearchParams({
    tenant_id: tenantId,
  });

  const response = await fetch(
    `${API_BASE_URL}/dashboard/sparklines?${params}`,
    {
      headers: {
        'Content-Type': 'application/json',
      },
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to fetch sparklines: ${response.statusText}`);
  }

  return response.json();
}

// ETL Health
export async function fetchETLHealth(
  tenantId: string
): Promise<ETLHealthData> {
  const params = new URLSearchParams({
    tenant_id: tenantId,
  });

  const response = await fetch(
    `${API_BASE_URL}/dashboard/etl-health?${params}`,
    {
      headers: {
        'Content-Type': 'application/json',
      },
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to fetch ETL health: ${response.statusText}`);
  }

  return response.json();
}

// Alerts
export async function fetchAlerts(
  tenantId: string,
  valuationDate: string
): Promise<AlertsData> {
  const params = new URLSearchParams({
    tenant_id: tenantId,
    valuation_date: valuationDate,
  });

  const response = await fetch(
    `${API_BASE_URL}/dashboard/alerts?${params}`,
    {
      headers: {
        'Content-Type': 'application/json',
      },
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to fetch alerts: ${response.statusText}`);
  }

  return response.json();
}

// Manual ETL trigger
export async function triggerETLRun(
  tenantId: string
): Promise<{ etl_run_id: string; status: string }> {
  const response = await fetch(
    `${API_BASE_URL}/dashboard/etl/trigger`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ tenant_id: tenantId }),
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to trigger ETL: ${response.statusText}`);
  }

  return response.json();
}
