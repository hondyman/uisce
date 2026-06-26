import React, { useMemo } from 'react';
import { useSubscription, useMutation, gql } from '@apollo/client';
import { format, formatDistance } from 'date-fns';
import { AlertCircle, TrendingUp, DollarSign, Calendar, CheckCircle, AlertTriangle } from 'lucide-react';

// =============================================================================
// GRAPHQL SUBSCRIPTIONS & MUTATIONS
// =============================================================================

const PORTFOLIO_EXPOSURE_SUBSCRIPTION = gql`
  subscription PortfolioExposure($tenantId: uuid!) {
    v_portfolio_exposure_summary(where: { tenant_id: { _eq: $tenantId } }) {
      commitment_id
      fund_name
      strategy_type
      fund_status
      commitment_amount
      paid_in_capital
      distributed_capital
      current_nav
      unfunded_commitment
      current_tvpi
      current_dpi
      current_irr_pct
      projected_calls_12m
      projected_distributions_12m
    }
  }
`;

const LIQUIDITY_NEEDS_SUBSCRIPTION = gql`
  subscription LiquidityNeeds($tenantId: uuid!) {
    v_liquidity_needs_projection(
      where: { tenant_id: { _eq: $tenantId } }
      order_by: [{ month: asc }]
    ) {
      month
      scenario
      total_calls
      total_distributions
      net_needs
      max_probable_calls_95th
    }
  }
`;

const CASH_FLOW_FORECASTS_SUBSCRIPTION = gql`
  subscription CashFlowForecasts($tenantId: uuid!) {
    cash_flow_forecasts(
      where: { tenant_id: { _eq: $tenantId }, scenario: { _eq: "base_case" } }
      order_by: [{ forecast_date: asc }]
    ) {
      commitment_id
      forecast_date
      scenario
      projected_calls
      projected_distributions
      projected_tvpi
      projected_irr_pct
      p5_percentile
      p95_percentile
    }
  }
`;

const RECONCILIATION_STATUS_SUBSCRIPTION = gql`
  subscription ReconciliationStatus($tenantId: uuid!) {
    v_reconciliation_status(where: { tenant_id: { _eq: $tenantId } }) {
      reconciliation_period
      reconciliation_rate_pct
      reconciled_count
      exceptions
      total_variance
    }
  }
`;

const TRIGGER_NAVIGATOR_FORECAST = gql`
  mutation TriggerForecast($commitmentId: uuid!, $tenantId: uuid!) {
    executeBusinessProcess(
      processId: "navigator_v1"
      input: {
        commitment_id: $commitmentId
        tenant_id: $tenantId
      }
    ) {
      workflow_id
      started_at
    }
  }
`;

// =============================================================================
// MAIN NAVIGATOR DASHBOARD COMPONENT
// =============================================================================

interface NavigatorDashboardProps {
  tenantId: string;
  currentCashBalance?: number;
}

export const NavigatorDashboard: React.FC<NavigatorDashboardProps> = ({
  tenantId,
  currentCashBalance = 0,
}) => {
  const { data: exposureData, loading: exposureLoading } = useSubscription(
    PORTFOLIO_EXPOSURE_SUBSCRIPTION,
    { variables: { tenantId } }
  );

  const { data: liquidityData, loading: liquidityLoading } = useSubscription(
    LIQUIDITY_NEEDS_SUBSCRIPTION,
    { variables: { tenantId } }
  );

  const { data: forecastData, loading: forecastDataLoading } = useSubscription(
    CASH_FLOW_FORECASTS_SUBSCRIPTION,
    { variables: { tenantId } }
  );

  const { data: reconciliationData, loading: reconciliationLoading } = useSubscription(
    RECONCILIATION_STATUS_SUBSCRIPTION,
    { variables: { tenantId } }
  );

  const [triggerForecast, { loading: triggerForecastLoading }] = useMutation(TRIGGER_NAVIGATOR_FORECAST);

  // =========================================================================
  // COMPUTED METRICS
  // =========================================================================

  const metrics = useMemo(() => {
    if (!exposureData?.v_portfolio_exposure_summary) return null;

    const funds = exposureData.v_portfolio_exposure_summary;

    const totalCommitment = funds.reduce((sum: number, f: any) => sum + (f.commitment_amount || 0), 0 as number);
    const totalPICC = funds.reduce((sum: number, f: any) => sum + (f.paid_in_capital || 0), 0 as number);
    const totalDCC = funds.reduce((sum: number, f: any) => sum + (f.distributed_capital || 0), 0 as number);
    const totalNAV = funds.reduce((sum: number, f: any) => sum + (f.current_nav || 0), 0 as number);
    const totalUnfunded = funds.reduce((sum: number, f: any) => sum + (f.unfunded_commitment || 0), 0 as number);
    const portfolioTVPI = totalPICC > 0 ? (totalNAV + totalDCC) / totalPICC : 0;
    const totalCalls12m = funds.reduce((sum: number, f: any) => sum + (f.projected_calls_12m || 0), 0 as number);
    const activeAlerts = funds.filter((f: any) => f.projected_calls_12m > 2000000).length;

    return {
      totalCommitment,
      totalPICC,
      totalDCC,
      totalNAV,
      totalUnfunded,
      portfolioTVPI,
      totalCalls12m,
      fundCount: funds.length,
      activeAlerts,
      liquidityGap: Math.max(0, totalCalls12m - currentCashBalance),
    };
  }, [exposureData, currentCashBalance]);

  const liquidityMetrics = useMemo(() => {
    if (!liquidityData?.v_liquidity_needs_projection) return null;

    const data = liquidityData.v_liquidity_needs_projection;
    const baseCase = data.filter((d: any) => d.scenario === 'base_case');
    const nextMonth = baseCase[0];
    const next12m = baseCase.slice(0, 12);

    const maxProb95 = Math.max(
      ...baseCase.map((d: any) => d.max_probable_calls_95th || 0)
    );

    return {
      nextMonthCalls: nextMonth?.total_calls || 0,
      nextMonthDist: nextMonth?.total_distributions || 0,
      netNeeds12m: next12m.reduce((sum: number, d: any) => sum + (d.net_needs || 0), 0),
      mpc: maxProb95, // Maximum Probable Call (95th percentile)
    };
  }, [liquidityData]);

  const reconciliationMetrics = useMemo(() => {
    if (!reconciliationData?.v_reconciliation_status) return null;

    const latest = reconciliationData.v_reconciliation_status[0];
    return {
      reconciliationRate: latest?.reconciliation_rate_pct || 0,
      reconciled: latest?.reconciled_count || 0,
      exceptions: latest?.exceptions || 0,
      totalVariance: latest?.total_variance || 0,
    };
  }, [reconciliationData]);

  // =========================================================================
  // EVENT HANDLERS
  // =========================================================================

  const handleTriggerForecast = async (commitmentId: string) => {
    try {
      await triggerForecast({
        variables: { commitmentId, tenantId },
      });
    } catch (error) {
      console.error('Failed to trigger forecast:', error);
    }
  };

  // =========================================================================
  // RENDER
  // =========================================================================

  if (!metrics || !liquidityMetrics || !reconciliationMetrics) {
    return <div className="p-8 text-center text-gray-500">Loading Navigator data...</div>;
  }

  return (
    <div className="space-y-6 p-6 bg-gray-50">
      {/* PAGE HEADER */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900">Navigator: Capital Management</h1>
        <p className="mt-1 text-gray-600">PE fund forecasting, reconciliation, and exposure tracking</p>
      </div>

      {/* ALERT SECTION */}
      {metrics.liquidityGap > 0 && (
        <div className="rounded-lg border-2 border-orange-200 bg-orange-50 p-4">
          <div className="flex gap-3">
            <AlertCircle className="h-5 w-5 flex-shrink-0 text-orange-600 mt-0.5" />
            <div>
              <h3 className="font-semibold text-orange-900">Liquidity Gap Detected</h3>
              <p className="mt-1 text-sm text-orange-800">
                Projected capital calls over next 12 months (${formatCurrency(metrics.totalCalls12m)})
                exceed available cash by ${formatCurrency(metrics.liquidityGap)}.
              </p>
              <p className="mt-2 text-sm font-medium text-orange-900">
                Maximum Probable Call (95th %ile): ${formatCurrency(liquidityMetrics.mpc)}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* KEY METRICS GRID */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          icon={<DollarSign className="h-5 w-5 text-blue-600" />}
          label="Total Commitment"
          value={formatCurrency(metrics.totalCommitment)}
          subtext={`${metrics.fundCount} funds`}
        />
        <MetricCard
          icon={<TrendingUp className="h-5 w-5 text-green-600" />}
          label="Portfolio TVPI"
          value={metrics.portfolioTVPI.toFixed(2)}
          subtext={`NAV: ${formatCurrency(metrics.totalNAV)}`}
        />
        <MetricCard
          icon={<Calendar className="h-5 w-5 text-purple-600" />}
          label="Projected Calls (12M)"
          value={formatCurrency(metrics.totalCalls12m)}
          subtext={`Unfunded: ${formatCurrency(metrics.totalUnfunded)}`}
        />
        <MetricCard
          icon={<AlertTriangle className="h-5 w-5 text-red-600" />}
          label="Liquidity Status"
          value={metrics.liquidityGap > 0 ? 'Gap: ' + formatCurrency(metrics.liquidityGap) : 'Healthy'}
          subtext={`Current cash: ${formatCurrency(currentCashBalance)}`}
        />
      </div>

      {/* PORTFOLIO SECTION */}
      <div className="rounded-lg border border-gray-200 bg-white shadow-sm">
        <div className="border-b border-gray-200 px-6 py-4">
          <h2 className="text-lg font-semibold text-gray-900">Fund Exposure Summary</h2>
        </div>
        <div className="overflow-x-auto">
          <FundExposureTable
            funds={exposureData?.v_portfolio_exposure_summary || []}
            onTriggerForecast={handleTriggerForecast}
            forecastLoading={triggerForecastLoading}
          />
        </div>
      </div>

      {/* CASH FLOW FORECAST SECTION */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* 12-Month Outlook */}
        <div className="rounded-lg border border-gray-200 bg-white shadow-sm">
          <div className="border-b border-gray-200 px-6 py-4">
            <h3 className="font-semibold text-gray-900">12-Month Liquidity Outlook</h3>
          </div>
          <div className="p-6">
            <LiquidityTimeline liquidityData={liquidityData?.v_liquidity_needs_projection || []} />
          </div>
        </div>

        {/* Reconciliation Status */}
        <div className="rounded-lg border border-gray-200 bg-white shadow-sm">
          <div className="border-b border-gray-200 px-6 py-4">
            <h3 className="font-semibold text-gray-900">Reconciliation Status</h3>
          </div>
          <div className="p-6">
            <ReconciliationStatus metrics={reconciliationMetrics} />
          </div>
        </div>
      </div>

      {/* FORECAST DETAILS */}
      <div className="rounded-lg border border-gray-200 bg-white shadow-sm">
        <div className="border-b border-gray-200 px-6 py-4">
          <h3 className="font-semibold text-gray-900">Cash Flow Forecast Detail</h3>
        </div>
        <div className="overflow-x-auto">
          <ForecastDetailTable forecasts={forecastData?.cash_flow_forecasts || []} />
        </div>
      </div>
    </div>
  );
};

// =============================================================================
// SUBCOMPONENTS
// =============================================================================

interface MetricCardProps {
  icon: React.ReactNode;
  label: string;
  value: string;
  subtext: string;
}

const MetricCard: React.FC<MetricCardProps> = ({ icon, label, value, subtext }) => (
  <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm hover:shadow-md transition">
    <div className="flex items-start justify-between">
      <div>
        <p className="text-sm font-medium text-gray-600">{label}</p>
        <p className="mt-2 text-2xl font-bold text-gray-900">{value}</p>
        <p className="mt-1 text-xs text-gray-500">{subtext}</p>
      </div>
      <div className="rounded-lg bg-gray-50 p-2">{icon}</div>
    </div>
  </div>
);

interface FundExposureTableProps {
  funds: any[];
  onTriggerForecast: (id: string) => void;
  forecastLoading: boolean;
}

const FundExposureTable: React.FC<FundExposureTableProps> = ({ funds, onTriggerForecast, forecastLoading }) => (
  <table className="w-full">
    <thead className="bg-gray-50">
      <tr>
        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Fund Name</th>
        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Strategy</th>
        <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Commitment</th>
        <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">PICC</th>
        <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">NAV</th>
        <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">TVPI</th>
        <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">12M Calls</th>
        <th className="px-6 py-3 text-center text-xs font-medium text-gray-500 uppercase">Action</th>
      </tr>
    </thead>
    <tbody className="divide-y divide-gray-200">
      {funds.map((fund) => (
        <tr key={fund.commitment_id} className="hover:bg-gray-50">
          <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
            {fund.fund_name}
          </td>
          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
            <span className="inline-block rounded-full bg-blue-100 px-2 py-1 text-xs font-medium text-blue-700">
              {fund.strategy_type.replace(/_/g, ' ')}
            </span>
          </td>
          <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium text-gray-900">
            {formatCurrency(fund.commitment_amount)}
          </td>
          <td className="px-6 py-4 whitespace-nowrap text-right text-sm text-gray-600">
            {formatCurrency(fund.paid_in_capital)}
          </td>
          <td className="px-6 py-4 whitespace-nowrap text-right text-sm text-gray-600">
            {formatCurrency(fund.current_nav)}
          </td>
          <td className={`px-6 py-4 whitespace-nowrap text-right text-sm font-semibold ${
            fund.current_tvpi > 1.5 ? 'text-green-600' : fund.current_tvpi > 1.0 ? 'text-blue-600' : 'text-orange-600'
          }`}>
            {fund.current_tvpi?.toFixed(2)}x
          </td>
          <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium text-gray-900">
            {formatCurrency(fund.projected_calls_12m)}
          </td>
          <td className="px-6 py-4 whitespace-nowrap text-center">
            <button
              onClick={() => onTriggerForecast(fund.commitment_id)}
              disabled={forecastLoading}
              className="inline-flex items-center gap-1 rounded bg-indigo-600 px-3 py-1 text-xs font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {forecastLoading ? 'Running...' : 'Forecast'}
            </button>
          </td>
        </tr>
      ))}
    </tbody>
  </table>
);

interface LiquidityTimelineProps {
  liquidityData: any[];
}

const LiquidityTimeline: React.FC<LiquidityTimelineProps> = ({ liquidityData }) => {
  const next12 = liquidityData.slice(0, 12);
  const maxValue = Math.max(...next12.map((d) => d.max_probable_calls_95th || 0), 1);

  return (
    <div className="space-y-3">
      {next12.map((month, idx) => (
        <div key={idx} className="space-y-1">
          <div className="flex items-center justify-between text-sm">
            <span className="font-medium text-gray-700">
              {format(new Date(month.month), 'MMM yyyy')}
            </span>
            <span className="text-gray-900 font-semibold">
              {formatCurrency(month.total_calls || 0)}
            </span>
          </div>
          <div className="h-2 rounded-full bg-gray-200">
            <div
              className="h-2 rounded-full bg-blue-500"
              style={{ width: `${((month.total_calls || 0) / maxValue) * 100}%` } as React.CSSProperties}
            />
          </div>
          <div className="text-xs text-gray-500">
            95% confidence: {formatCurrency(month.max_probable_calls_95th || 0)}
          </div>
        </div>
      ))}
    </div>
  );
};

interface ReconciliationStatusProps {
  metrics: any;
}

const ReconciliationStatus: React.FC<ReconciliationStatusProps> = ({ metrics }) => (
  <div className="space-y-4">
    <div className="flex items-center justify-between">
      <span className="text-sm font-medium text-gray-600">Reconciliation Rate</span>
      <span className="text-2xl font-bold text-green-600">{metrics.reconciliationRate.toFixed(1)}%</span>
    </div>
    <div className="grid grid-cols-2 gap-4">
      <div className="rounded-lg bg-green-50 p-3">
        <p className="text-xs font-medium text-gray-600">Reconciled</p>
        <p className="mt-1 text-xl font-bold text-green-600">{metrics.reconciled}</p>
      </div>
      <div className="rounded-lg bg-red-50 p-3">
        <p className="text-xs font-medium text-gray-600">Exceptions</p>
        <p className="mt-1 text-xl font-bold text-red-600">{metrics.exceptions}</p>
      </div>
    </div>
    <div>
      <p className="text-xs font-medium text-gray-600">Total Variance</p>
      <p className="mt-1 text-sm font-semibold text-gray-900">
        {formatCurrency(metrics.totalVariance)}
      </p>
    </div>
  </div>
);

interface ForecastDetailTableProps {
  forecasts: any[];
}

const ForecastDetailTable: React.FC<ForecastDetailTableProps> = ({ forecasts }) => {
  const grouped = forecasts.slice(0, 12); // Next 12 months

  return (
    <table className="w-full">
      <thead className="bg-gray-50">
        <tr>
          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Date</th>
          <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Proj Calls</th>
          <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Proj Dists</th>
          <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Proj TVPI</th>
          <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">p5 (5th %ile)</th>
          <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">p95 (95th %ile)</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-gray-200">
        {grouped.map((f) => (
          <tr key={f.forecast_date} className="hover:bg-gray-50">
            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
              {format(new Date(f.forecast_date), 'MMM dd, yyyy')}
            </td>
            <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium text-gray-900">
              {formatCurrency(f.projected_calls)}
            </td>
            <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium text-gray-900">
              {formatCurrency(f.projected_distributions)}
            </td>
            <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium text-gray-900">
              {f.projected_tvpi?.toFixed(2)}x
            </td>
            <td className="px-6 py-4 whitespace-nowrap text-right text-sm text-gray-600">
              {formatCurrency(f.p5_percentile)}
            </td>
            <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-semibold text-orange-600">
              {formatCurrency(f.p95_percentile)}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
};

// =============================================================================
// UTILITIES
// =============================================================================

function formatCurrency(value: number | null | undefined): string {
  if (!value && value !== 0) return '$0';
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value);
}

export default NavigatorDashboard;
