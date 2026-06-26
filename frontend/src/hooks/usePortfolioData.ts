import { useQuery, UseQueryResult } from 'react-query';
import {
  fetchPortfolioOverview,
  fetchPortfolioHoldings,
  fetchPortfolioRisk,
  fetchPortfolioCompliance,
  fetchPortfolioScenarios,
  PortfolioOverview,
  HoldingsSummary,
  RiskSnapshot,
  ComplianceSnapshot,
  ScenarioResults,
} from '../api/portfolioApi';

const STALE_TIME = 60000; // 1 minute
const CACHE_TIME = 5 * 60000; // 5 minutes

/**
 * Hook for fetching portfolio overview
 */
export function usePortfolioOverview(
  portfolioId: string | null,
  valuationDate: string
): UseQueryResult<PortfolioOverview, Error> {
  return useQuery(
    ['portfolioOverview', portfolioId, valuationDate],
    () => fetchPortfolioOverview(portfolioId!, valuationDate),
    {
      enabled: !!portfolioId,
      staleTime: STALE_TIME,
      cacheTime: CACHE_TIME,
      retry: 2,
    }
  );
}

/**
 * Hook for fetching portfolio holdings
 */
export function usePortfolioHoldings(
  portfolioId: string | null,
  valuationDate: string
): UseQueryResult<HoldingsSummary, Error> {
  return useQuery(
    ['portfolioHoldings', portfolioId, valuationDate],
    () => fetchPortfolioHoldings(portfolioId!, valuationDate),
    {
      enabled: !!portfolioId,
      staleTime: STALE_TIME,
      cacheTime: CACHE_TIME,
      retry: 2,
    }
  );
}

/**
 * Hook for fetching portfolio risk metrics
 */
export function usePortfolioRisk(
  portfolioId: string | null,
  valuationDate: string
): UseQueryResult<RiskSnapshot, Error> {
  return useQuery(
    ['portfolioRisk', portfolioId, valuationDate],
    () => fetchPortfolioRisk(portfolioId!, valuationDate),
    {
      enabled: !!portfolioId,
      staleTime: STALE_TIME,
      cacheTime: CACHE_TIME,
      retry: 2,
    }
  );
}

/**
 * Hook for fetching portfolio compliance
 */
export function usePortfolioCompliance(
  portfolioId: string | null,
  valuationDate: string
): UseQueryResult<ComplianceSnapshot, Error> {
  return useQuery(
    ['portfolioCompliance', portfolioId, valuationDate],
    () => fetchPortfolioCompliance(portfolioId!, valuationDate),
    {
      enabled: !!portfolioId,
      staleTime: STALE_TIME,
      cacheTime: CACHE_TIME,
      retry: 2,
    }
  );
}

/**
 * Hook for fetching scenario results
 */
export function usePortfolioScenarios(
  portfolioId: string | null,
  valuationDate: string
): UseQueryResult<ScenarioResults, Error> {
  return useQuery(
    ['portfolioScenarios', portfolioId, valuationDate],
    () => fetchPortfolioScenarios(portfolioId!, valuationDate),
    {
      enabled: !!portfolioId,
      staleTime: STALE_TIME,
      cacheTime: CACHE_TIME,
      retry: 2,
    }
  );
}

/**
 * Combined hook for all portfolio data
 */
export function usePortfolioData(
  portfolioId: string | null,
  valuationDate: string
) {
  const overview = usePortfolioOverview(portfolioId, valuationDate);
  const holdings = usePortfolioHoldings(portfolioId, valuationDate);
  const risk = usePortfolioRisk(portfolioId, valuationDate);
  const compliance = usePortfolioCompliance(portfolioId, valuationDate);
  const scenarios = usePortfolioScenarios(portfolioId, valuationDate);

  const isLoading =
    overview.isLoading ||
    holdings.isLoading ||
    risk.isLoading ||
    compliance.isLoading ||
    scenarios.isLoading;

  const isError =
    overview.isError ||
    holdings.isError ||
    risk.isError ||
    compliance.isError ||
    scenarios.isError;

  const error =
    overview.error ||
    holdings.error ||
    risk.error ||
    compliance.error ||
    scenarios.error;

  return {
    overview,
    holdings,
    risk,
    compliance,
    scenarios,
    isLoading,
    isError,
    error,
  };
}
