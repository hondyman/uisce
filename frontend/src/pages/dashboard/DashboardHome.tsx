import React, { useState, useCallback } from 'react';
import { useDashboardData } from '../../hooks/useDashboardData';
import { useDashboardContext } from '../../contexts/DashboardContext';
import {
  ComplianceKPIs,
  RiskKPIs,
} from './KPIComponents';
import { SparklinesGrid } from './SparklineComponents';
import { ETLHealth, AlertsPanel } from './OperationsComponents';
import {
  ConsoleLayout,
  ConsoleBreadcrumbs,
  ConsoleHeader,
  ConsoleGrid,
  ConsoleTopNav,
  ConsoleStatusBar,
} from './LayoutComponents';

interface TenantSelectorProps {
  value: string | null;
  onChange: (tenantId: string) => void;
  isLoading?: boolean;
}

const TenantSelector: React.FC<TenantSelectorProps> = ({ value, onChange, isLoading }) => {
  return (
    <div className="flex items-center gap-2 bg-slate-100 dark:bg-slate-800 rounded-lg px-3 py-1.5 border border-slate-200 dark:border-slate-700">
      <span className="text-xs font-semibold text-slate-600 dark:text-slate-400 uppercase">
        Tenant
      </span>
      <select
        value={value || ''}
        onChange={(e) => onChange(e.target.value)}
        disabled={isLoading}
        className="bg-transparent border-none text-sm font-medium text-slate-900 dark:text-white cursor-pointer focus:outline-none"
      >
        <option value="">Select Tenant...</option>
        <option value="acme-asset-mgmt">Acme Asset Management</option>
        <option value="global-wealth">Global Wealth Partners</option>
        <option value="institutional-inv">Institutional Investors LLC</option>
      </select>
    </div>
  );
};

interface DateSelectorProps {
  value: string;
  onChange: (date: string) => void;
}

const DateSelector: React.FC<DateSelectorProps> = ({ value, onChange }) => {
  return (
    <div className="flex items-center gap-2 bg-slate-100 dark:bg-slate-800 rounded-lg px-3 py-1.5 border border-slate-200 dark:border-slate-700">
      <span className="text-xs font-semibold text-slate-600 dark:text-slate-400 uppercase">
        Date
      </span>
      <input
        type="date"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="bg-transparent border-none text-sm font-medium text-slate-900 dark:text-white cursor-pointer focus:outline-none"
      />
    </div>
  );
};

export const DashboardHome: React.FC = () => {
  const { selectedTenant, valuationDate, selectTenant, setValuationDate } =
    useDashboardContext();
  const [showAllAlerts, setShowAllAlerts] = useState(false);

  const dashboard = useDashboardData(
    selectedTenant?.id || null,
    valuationDate
  );

  const handleTenantChange = useCallback(
    (tenantId: string) => {
      if (tenantId) {
        selectTenant({
          id: tenantId,
          name: tenantId
            .split('-')
            .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
            .join(' '),
        });
      }
    },
    [selectTenant]
  );

  const handleViewAllAlerts = useCallback(() => {
    setShowAllAlerts(true);
    // In production, this would navigate to a dedicated alerts page
    console.log('View all alerts clicked');
  }, []);

  const handleTriggerETL = useCallback(() => {
    console.log('Trigger ETL clicked for tenant:', selectedTenant?.id);
    // In production, call the ETL trigger API
  }, [selectedTenant?.id]);

  const handleViewLogs = useCallback(() => {
    console.log('View logs clicked');
    // In production, navigate to logs page or open modal
  }, []);

  return (
    <>
      {/* Top Navigation */}
      <ConsoleTopNav
        title="Risk & Compliance Cockpit"
        logo={
          <div className="p-2 bg-blue-600 dark:bg-blue-700 rounded-lg flex items-center justify-center">
            <span className="text-white font-bold text-lg">🛡️</span>
          </div>
        }
        navItems={[
          { label: 'Console', href: '/dashboard', active: true },
          { label: 'Portfolios', href: '/portfolios' },
          { label: 'Reports', href: '/reports' },
        ]}
        rightContent={
          <div className="flex items-center gap-3">
            <TenantSelector
              value={selectedTenant?.id || null}
              onChange={handleTenantChange}
              isLoading={dashboard.isLoading}
            />
            <DateSelector value={valuationDate} onChange={setValuationDate} />
            <div className="h-8 w-[1px] bg-slate-200 dark:border-slate-800 mx-2" />
            <button className="p-2 rounded-full hover:bg-slate-100 dark:hover:bg-slate-800">
              🔔
            </button>
            <div className="h-8 w-8 rounded-full bg-blue-600/20 flex items-center justify-center border border-blue-600/40">
              <span className="text-xs font-bold text-blue-600 dark:text-blue-400">JD</span>
            </div>
          </div>
        }
      />

      {/* Main Content */}
      <ConsoleLayout maxWidth="2xl">
        {/* Breadcrumbs */}
        <ConsoleBreadcrumbs
          items={[
            { label: 'Console', href: '/dashboard' },
            { label: 'Dashboard', active: true },
          ]}
        />

        {!selectedTenant ? (
          <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-xl p-8 text-center">
            <p className="text-blue-900 dark:text-blue-100 font-semibold mb-2">
              👋 Welcome to the Risk & Compliance Cockpit
            </p>
            <p className="text-blue-700 dark:text-blue-200 text-sm">
              Please select a tenant from the dropdown above to view dashboard metrics
            </p>
          </div>
        ) : (
          <>
            {/* Error State */}
            {dashboard.isError && (
              <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-xl p-4 mb-6">
                <p className="text-red-700 dark:text-red-300 text-sm font-medium">
                  {dashboard.error?.message ||
                    'Failed to load dashboard data. Please try again.'}
                </p>
              </div>
            )}

            {/* Row 1: KPI Cards */}
            <ConsoleGrid columns={2} gap="lg">
              <ComplianceKPIs
                data={dashboard.compliance.data}
                isLoading={dashboard.compliance.isLoading}
                error={dashboard.compliance.error}
              />
              <RiskKPIs
                data={dashboard.risk.data}
                isLoading={dashboard.risk.isLoading}
                error={dashboard.risk.error}
              />
            </ConsoleGrid>

            {/* Row 2: Sparklines */}
            <div className="mt-6">
              <SparklinesGrid
                data={dashboard.sparklines.data}
                isLoading={dashboard.sparklines.isLoading}
                error={dashboard.sparklines.error}
              />
            </div>

            {/* Row 3: Operations & Alerts */}
            <ConsoleGrid columns={2} gap="lg">
              <div className="lg:col-span-1">
                <ETLHealth
                  data={dashboard.etl.data}
                  isLoading={dashboard.etl.isLoading}
                  error={dashboard.etl.error}
                  onTriggerRun={handleTriggerETL}
                  onViewLogs={handleViewLogs}
                />
              </div>
              <div className="lg:col-span-1">
                <AlertsPanel
                  data={dashboard.alerts.data}
                  isLoading={dashboard.alerts.isLoading}
                  error={dashboard.alerts.error}
                  onViewAll={handleViewAllAlerts}
                />
              </div>
            </ConsoleGrid>

            {/* Bottom Spacing for Status Bar */}
            <div className="h-16" />
          </>
        )}
      </ConsoleLayout>

      {/* Status Bar Footer */}
      {selectedTenant && (
        <ConsoleStatusBar
          items={[
            { label: 'System Status:', value: '🟢 Online' },
            { label: 'Node Cluster:', value: 'US-EAST-1' },
            { label: 'API Latency:', value: '12ms' },
          ]}
          rightItems={[
            { label: '© 2024 Risk Compliance Console', value: '' },
            { label: 'Mode:', value: 'PROD' },
          ]}
        />
      )}
    </>
  );
};

export default DashboardHome;
