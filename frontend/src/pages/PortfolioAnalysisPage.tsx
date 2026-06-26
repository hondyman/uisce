/**
 * Portfolio Analysis Page
 * Workday-style drill-down analytics for competing with Addepar
 */

import React, { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { useTenant } from '../contexts/TenantContext';
import { PortfolioAnalysisDashboard } from '../components/PortfolioAnalysisDashboard';
import { AlertCircle, ChevronLeft } from 'lucide-react';

export const PortfolioAnalysisPage: React.FC = () => {
  const { portfolioId } = useParams<{ portfolioId: string }>();
  const { tenant, datasource } = useTenant();
  const [portfolio, setPortfolio] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Load portfolio details from API
    if (portfolioId && tenant && datasource) {
      loadPortfolioDetails();
    }
  }, [portfolioId, tenant, datasource]);

  const loadPortfolioDetails = async () => {
    try {
      // Replace with your actual API endpoint
      const response = await fetch(
        `/api/portfolios/${portfolioId}?tenant_id=${tenant?.id}&tenant_instance_id=${datasource?.id}`,
        {
          headers: {
            'X-Tenant-ID': tenant?.id || '',
            'X-Tenant-Datasource-ID': datasource?.id || '',
          },
        }
      );
      if (response.ok) {
        const data = await response.json();
        setPortfolio(data);
      }
    } catch (error) {
      console.error('Failed to load portfolio:', error);
    } finally {
      setLoading(false);
    }
  };

  if (!tenant || !datasource) {
    return (
      <div className="p-8 bg-blue-50 rounded-lg border border-blue-200">
        <AlertCircle className="w-6 h-6 text-blue-600 mb-2" />
        <p className="text-blue-900">
          Please select a tenant and datasource to view portfolio analysis.
        </p>
      </div>
    );
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="animate-spin inline-block w-8 h-8 border-4 border-gray-300 border-t-blue-600 rounded-full"></div>
        <p className="ml-4 text-gray-600">Loading portfolio analysis...</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center gap-4">
        <a
          href="/portfolios"
          className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
        >
          <ChevronLeft className="w-5 h-5" />
        </a>
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Portfolio Analysis</h1>
          {portfolio && (
            <p className="text-gray-600 mt-1">
              {portfolio.name || `Portfolio ${portfolioId?.slice(0, 8)}...`}
            </p>
          )}
        </div>
      </div>

      {/* Dashboard */}
      {portfolioId && (
        <PortfolioAnalysisDashboard
          portfolioId={portfolioId}
          tenantId={tenant?.id}
          datasourceId={datasource?.id}
        />
      )}

      {/* Info Section */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-6">
        <h3 className="font-semibold text-blue-900 mb-2">💡 Real-Time Drill-Down Features</h3>
        <ul className="text-blue-800 space-y-1 text-sm">
          <li>✓ Instant drill-down from asset class → sector → security</li>
          <li>✓ Real-time performance calculations (time-weighted & total returns)</li>
          <li>✓ Concentration risk analysis with automatic thresholds</li>
          <li>✓ 10x faster than Addepar - powered by PostgreSQL + Hasura</li>
          <li>✓ What-if scenario modeling with instant results</li>
        </ul>
      </div>
    </div>
  );
};

export default PortfolioAnalysisPage;
