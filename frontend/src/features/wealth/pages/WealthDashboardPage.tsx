import React, { useState, useEffect } from 'react';

interface PortfolioSummary {
  id: string;
  name: string;
  totalValue: number;
  dailyReturn: number;
  ytdReturn: number;
  compliance: 'COMPLIANT' | 'NON_COMPLIANT' | 'PENDING_REVIEW';
}

interface ComplianceAlert {
  id: string;
  clientName: string;
  type: string;
  severity: 'HIGH' | 'MEDIUM' | 'LOW';
  message: string;
  createdAt: string;
}

export const WealthDashboardPage: React.FC = () => {
  const [portfolios, setPortfolios] = useState<PortfolioSummary[]>([]);
  const [complianceAlerts, setComplianceAlerts] = useState<ComplianceAlert[]>([]);
  const [totalAUM, setTotalAUM] = useState(0);
  const [activeClients, setActiveClients] = useState(0);

  useEffect(() => {
    // Mock data fetch - replace with real API calls
    setPortfolios([
      { id: '1', name: 'Growth Portfolio', totalValue: 1250000, dailyReturn: 0.85, ytdReturn: 12.5, compliance: 'COMPLIANT' },
      { id: '2', name: 'Retirement Fund', totalValue: 850000, dailyReturn: -0.32, ytdReturn: 8.2, compliance: 'COMPLIANT' },
      { id: '3', name: 'Aggressive Growth', totalValue: 2500000, dailyReturn: 1.42, ytdReturn: 18.7, compliance: 'PENDING_REVIEW' },
      { id: '4', name: 'Conservative Income', totalValue: 675000, dailyReturn: 0.15, ytdReturn: 4.3, compliance: 'COMPLIANT' },
    ]);

    setComplianceAlerts([
      {
        id: 'alert-1',
        clientName: 'John Doe',
        type: 'KYC Expiring',
        severity: 'MEDIUM',
        message: 'KYC documentation expires in 30 days',
        createdAt: new Date().toISOString(),
      },
      {
        id: 'alert-2',
        clientName: 'Aggressive Growth Portfolio',
        type: 'Concentration Risk',
        severity: 'HIGH',
        message: 'Single position exceeds 25% of portfolio value',
        createdAt: new Date().toISOString(),
      },
    ]);

    setTotalAUM(5275000);
    setActiveClients(42);
  }, []);

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'HIGH': return 'bg-red-100 text-red-800 border-red-500';
      case 'MEDIUM': return 'bg-yellow-100 text-yellow-800 border-yellow-500';
      case 'LOW': return 'bg-blue-100 text-blue-800 border-blue-500';
      default: return 'bg-gray-100 text-gray-800 border-gray-500';
    }
  };

  return (
    <div className="min-h-screen bg-gray-50 p-6">
      <div className="max-w-7xl mx-auto">
        <h1 className="text-3xl font-bold text-gray-900 mb-6">Wealth Management Dashboard</h1>

        {/* KPI Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
          <div className="bg-white rounded-lg shadow p-6">
            <p className="text-sm text-gray-600 mb-2">Total AUM</p>
            <p className="text-3xl font-bold text-gray-900">${(totalAUM / 1000000).toFixed(2)}M</p>
            <p className="text-sm text-green-600 mt-2">↑ 8.5% YTD</p>
          </div>

          <div className="bg-white rounded-lg shadow p-6">
            <p className="text-sm text-gray-600 mb-2">Active Clients</p>
            <p className="text-3xl font-bold text-gray-900">{activeClients}</p>
            <p className="text-sm text-blue-600 mt-2">+3 this month</p>
          </div>

          <div className="bg-white rounded-lg shadow p-6">
            <p className="text-sm text-gray-600 mb-2">Avg Portfolio Return</p>
            <p className="text-3xl font-bold text-gray-900">11.2%</p>
            <p className="text-sm text-gray-500 mt-2">Year-to-date</p>
          </div>

          <div className="bg-white rounded-lg shadow p-6">
            <p className="text-sm text-gray-600 mb-2">Compliance Score</p>
            <p className="text-3xl font-bold text-green-600">98%</p>
            <p className="text-sm text-gray-500 mt-2">2 alerts pending</p>
          </div>
        </div>

        {/* Portfolio Performance Table */}
        <div className="bg-white rounded-lg shadow mb-8">
          <div className="px-6 py-4 border-b border-gray-200">
            <h2 className="text-lg font-semibold text-gray-800">Portfolio Performance</h2>
          </div>
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Portfolio</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Total Value</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Daily Return</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">YTD Return</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Compliance</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {portfolios.map((portfolio) => (
                  <tr key={portfolio.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap font-medium text-gray-900">{portfolio.name}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-gray-700">${portfolio.totalValue.toLocaleString()}</td>
                    <td className={`px-6 py-4 whitespace-nowrap font-semibold ${portfolio.dailyReturn >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                      {portfolio.dailyReturn >= 0 ? '↑' : '↓'} {Math.abs(portfolio.dailyReturn)}%
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-gray-700">{portfolio.ytdReturn}%</td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`px-2 py-1 text-xs font-semibold rounded ${
                        portfolio.compliance === 'COMPLIANT' ? 'bg-green-100 text-green-800' :
                        portfolio.compliance === 'PENDING_REVIEW' ? 'bg-yellow-100 text-yellow-800' :
                        'bg-red-100 text-red-800'
                      }`}>
                        {portfolio.compliance}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Compliance Alerts */}
        <div className="bg-white rounded-lg shadow">
          <div className="px-6 py-4 border-b border-gray-200 flex justify-between items-center">
            <h2 className="text-lg font-semibold text-gray-800">Compliance Alerts</h2>
            <span className="bg-red-100 text-red-800 text-xs px-2 py-1 rounded-full">{complianceAlerts.length} Active</span>
          </div>
          <div className="p-6">
            {complianceAlerts.length === 0 ? (
              <p className="text-gray-500 text-center py-8">No active compliance alerts</p>
            ) : (
              <div className="space-y-4">
                {complianceAlerts.map((alert) => (
                  <div key={alert.id} className={`border-l-4 rounded p-4 ${getSeverityColor(alert.severity)}`}>
                    <div className="flex justify-between items-start">
                      <div>
                        <div className="flex items-center space-x-2 mb-1">
                          <span className="font-semibold">{alert.type}</span>
                          <span className="text-xs bg-white/50 px-2 py-0.5 rounded">{alert.severity}</span>
                        </div>
                        <p className="text-sm mb-1">{alert.clientName}</p>
                        <p className="text-sm">{alert.message}</p>
                        <p className="text-xs mt-2 opacity-75">
                          {new Date(alert.createdAt).toLocaleString()}
                        </p>
                      </div>
                      <button className="bg-white text-gray-700 px-3 py-1 rounded text-sm hover:bg-gray-100 border">
                        Review
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
