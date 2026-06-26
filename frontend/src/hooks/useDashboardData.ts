import { useQuery, UseQueryResult } from 'react-query';
import {
  fetchComplianceKPIs,
  fetchRiskKPIs,
  fetchSparklines,
  fetchETLHealth,
  fetchAlerts,
  ComplianceKPIData,
  RiskKPIData,
  SparklinesData,
  ETLHealthData,
  AlertsData,
} from '../api/dashboardApi';

const STALE_TIME = 60000; // 1 minute
const CACHE_TIME = 5 * 60000; // 5 minutes

/**
 * Hook for fetching Compliance KPIs
 * Provides real-time compliance metrics including pass rates, breaches, and failing rules
 */
export function useComplianceKPIs(
  tenantId: string | null,
  valuationDate: string
): UseQueryResult<ComplianceKPIData, Error> {
  return useQuery(
    ['complianceKPIs', tenantId, valuationDate],
    () => fetchComplianceKPIs(tenantId!, valuationDate),
    {
      enabled: !!tenantId,
      staleTime: STALE_TIME,
      cacheTime: CACHE_TIME,
      refetchInterval: 60000, // Refetch every minute
      refetchOnWindowFocus: true,
      retry: 2,
    }
  );
}

/**
 * Hook for fetching Risk KPIs
 * Provides portfolio risk posture metrics including volatility, VaR, and scenario analysis
 */
export function useRiskKPIs(
  tenantId: string | null,
  valuationDate: string
): UseQueryResult<RiskKPIData, Error> {
  return useQuery(
    ['riskKPIs', tenantId, valuationDate],
    () => fetchRiskKPIs(tenantId!, valuationDate),
    {
      enabled: !!tenantId,
      staleTime: STALE_TIME,
      cacheTime: CACHE_TIME,
      refetchInterval: 60000, // Refetch every minute
      refetchOnWindowFocus: true,
      retry: 2,
    }
  );
}

/**
 * Hook for fetching Sparkline data
 * Provides 7-day trend data for pass rate, breaches, volatility, and ETL duration
 */
export function useSparklines(
  tenantId: string | null
): UseQueryResult<SparklinesData, Error> {
  return useQuery(
    ['sparklines', tenantId],
    () => fetchSparklines(tenantId!),
    {
      enabled: !!tenantId,
      staleTime: 5 * 60000, // 5 minutes (less frequent than KPIs)
      cacheTime: 15 * 60000, // 15 minutes
      refetchInterval: 300000, // Refetch every 5 minutes
      refetchOnWindowFocus: false,
      retry: 2,
    }
  );
}

/**
 * Hook for fetching ETL Health
 * Provides operational heartbeat: last run status, duration, WASM version
 */
export function useETLHealth(
  tenantId: string | null
): UseQueryResult<ETLHealthData, Error> {
  return useQuery(
    ['etlHealth', tenantId],
    () => fetchETLHealth(tenantId!),
    {
      enabled: !!tenantId,
      staleTime: 30000, // 30 seconds (more frequent updates)
      cacheTime: 2 * 60000, // 2 minutes
      refetchInterval: 30000, // Refetch every 30 seconds
      refetchOnWindowFocus: true,
      retry: 1,
    }
  );
}

/**
 * Hook for fetching Alerts
 * Provides real-time alerts for hard/soft breaches, scenarios, and ETL failures
 */
export function useAlerts(
  tenantId: string | null,
  valuationDate: string
): UseQueryResult<AlertsData, Error> {
  return useQuery(
    ['alerts', tenantId, valuationDate],
    () => fetchAlerts(tenantId!, valuationDate),
    {
      enabled: !!tenantId,
      staleTime: 30000, // 30 seconds
      cacheTime: 2 * 60000, // 2 minutes
      refetchInterval: 30000, // Refetch every 30 seconds
      refetchOnWindowFocus: true,
      retry: 2,
    }
  );
}

/**
 * Combined hook for all dashboard data
 * Useful for the main dashboard view that needs all data at once
 */
export function useDashboardData(
  tenantId: string | null,
  valuationDate: string
) {
  const compliance = useComplianceKPIs(tenantId, valuationDate);
  const risk = useRiskKPIs(tenantId, valuationDate);
  const sparklines = useSparklines(tenantId);
  const etl = useETLHealth(tenantId);
  const alerts = useAlerts(tenantId, valuationDate);

  const isLoading =
    compliance.isLoading ||
    risk.isLoading ||
    sparklines.isLoading ||
    etl.isLoading ||
    alerts.isLoading;

  const isError =
    compliance.isError ||
    risk.isError ||
    sparklines.isError ||
    etl.isError ||
    alerts.isError;

  const error =
    compliance.error ||
    risk.error ||
    sparklines.error ||
    etl.error ||
    alerts.error;

  return {
    compliance,
    risk,
    sparklines,
    etl,
    alerts,
    isLoading,
    isError,
    error,
  };
}
