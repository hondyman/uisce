import React, { useState } from 'react';
import { useQuery, gql } from '@apollo/client';
import { useNotification } from '../hooks/useNotification';
import styles from './AIPortfolioRebalancer.module.css';

const PORTFOLIOS_QUERY = gql`
  query GetPortfolios {
    portfolios {
      id
      clientId
      aum
      drift
      holdings
      status
      lastRebalanced
    }
  }
`;

interface Portfolio {
  id: string;
  clientId: string;
  clientName: string;
  aum: number;
  drift: number;
  holdings: number;
  status: 'high-drift' | 'moderate-drift' | 'healthy';
  lastRebalanced: string;
  taxSaved?: number;
}

interface RebalancePlan {
  portfolioId: string;
  currentDrift: number;
  expectedDrift: number;
  taxSavings: number;
  rationale: string;
  trades: Array<{
    action: 'BUY' | 'SELL';
    symbol: string;
    shares: number;
    value: number;
  }>;
  confidence: number;
}

export const AIPortfolioRebalancer: React.FC = () => {
  const { data: portfoliosData } = useQuery(PORTFOLIOS_QUERY);
  const notification = useNotification();
  const [selectedModal, setSelectedModal] = useState<string | null>(null);
  const [selectedPlan, setSelectedPlan] = useState<RebalancePlan | null>(null);

  const mockPortfolios: Portfolio[] = [
    {
      id: 'port-1',
      clientId: 'client-1',
      clientName: 'James Howlett',
      aum: 2500000,
      drift: 8.5,
      holdings: 42,
      status: 'high-drift',
      lastRebalanced: 'Mar 15, 2024',
      taxSaved: 12000,
    },
    {
      id: 'port-2',
      clientId: 'client-2',
      clientName: 'Jean Grey',
      aum: 1800000,
      drift: 4.2,
      holdings: 28,
      status: 'moderate-drift',
      lastRebalanced: 'Feb 28, 2024',
      taxSaved: 8000,
    },
    {
      id: 'port-3',
      clientId: 'client-3',
      clientName: 'Scott Summers',
      aum: 5100000,
      drift: 0.8,
      holdings: 75,
      status: 'healthy',
      lastRebalanced: 'May 1, 2024',
      taxSaved: 25000,
    },
  ];

  const mockStats = {
    totalAUM: 1200000000,
    aumChange: 1.2,
    avgDrift: 2.1,
    driftChange: -0.3,
    taxSavedYTD: 4500000,
    taxSavedChange: 5.8,
    needsRebalance: 12,
  };

  const getDriftColor = (drift: number) => {
    if (drift >= 6) return '#FF4D4D';
    if (drift >= 3) return '#FFC107';
    return '#00D18F';
  };

  const getDriftClass = (drift: number) => {
    if (drift >= 6) return styles.driftHigh;
    if (drift >= 3) return styles.driftModerate;
    return styles.driftLow;
  };

  const getDriftStatus = (drift: number): Portfolio['status'] => {
    if (drift >= 6) return 'high-drift';
    if (drift >= 3) return 'moderate-drift';
    return 'healthy';
  };

  const getStatusLabel = (status: Portfolio['status']) => {
    if (status === 'high-drift') return { label: 'High Drift', icon: 'warning', color: '#FF4D4D' };
    if (status === 'moderate-drift') return { label: 'Moderate Drift', icon: 'error', color: '#FFC107' };
    return { label: 'Healthy', icon: 'check_circle', color: '#00D18F' };
  };

  const handleRebalance = (portfolio: Portfolio) => {
    const plan: RebalancePlan = {
      portfolioId: portfolio.id,
      currentDrift: portfolio.drift,
      expectedDrift: 0.5,
      taxSavings: portfolio.drift > 6 ? 1200 : portfolio.drift > 3 ? 800 : 200,
      rationale:
        'Rebalancing to reduce overweight exposure in the tech sector, capitalizing on tax-loss harvesting opportunities in underperforming industrial assets while maintaining target risk profile.',
      trades: [
        { action: 'SELL', symbol: 'AAPL', shares: 150, value: 25500 },
        { action: 'BUY', symbol: 'MSFT', shares: 60, value: 24000 },
        { action: 'SELL', symbol: 'TSLA', shares: 50, value: 9000 },
        { action: 'BUY', symbol: 'VTI', shares: 35, value: 8750 },
        { action: 'BUY', symbol: 'JPM', shares: 10, value: 1750 },
      ],
      confidence: 0.95,
    };
    setSelectedPlan(plan);
    setSelectedModal(portfolio.id);
  };

  const handleExecutePlan = async () => {
    if (!selectedPlan) return;
    try {
      const response = await fetch(`/api/portfolio/${selectedPlan.portfolioId}/rebalance`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': localStorage.getItem('selected_tenant') || '',
          'X-Tenant-Datasource-ID': localStorage.getItem('selected_datasource') || '',
        },
        body: JSON.stringify(selectedPlan),
      });
      if (response.ok) {
        notification.success('Rebalance plan executed successfully!');
        setSelectedModal(null);
      }
    } catch (error) {
      console.error('Failed to execute rebalance plan:', error);
    }
  };

  return (
    <div className="flex min-h-screen bg-[#1A1D21] font-sans text-[#EAEAEA]">
      {/* SideNavBar */}
      <aside className="w-64 flex-shrink-0 bg-[#252A31] p-4">
        <div className="flex h-full flex-col justify-between">
          <div className="flex flex-col gap-8">
            <div className="flex items-center gap-2 p-2">
              <span className="text-3xl">✨</span>
              <span className="text-lg font-bold text-white">AI Alpha</span>
            </div>
            <div className="flex flex-col gap-2">
              <a className="flex items-center gap-3 rounded-lg bg-blue-600/20 px-3 py-2 text-white" href="#">
                <span>📊</span>
                <p className="text-sm font-medium">Dashboard</p>
              </a>
              <a className="flex items-center gap-3 rounded-lg px-3 py-2 text-[#A0A0A0] hover:bg-white/5" href="/analytics/rebalancer">
                <span>🔄</span>
                <p className="text-sm font-medium">Rebalancer</p>
              </a>
              <a className="flex items-center gap-3 rounded-lg px-3 py-2 text-[#A0A0A0] hover:bg-white/5" href="#">
                <span>📈</span>
                <p className="text-sm font-medium">Analytics</p>
              </a>
              <a className="flex items-center gap-3 rounded-lg px-3 py-2 text-[#A0A0A0] hover:bg-white/5" href="#">
                <span>📋</span>
                <p className="text-sm font-medium">Reports</p>
              </a>
              <a className="flex items-center gap-3 rounded-lg px-3 py-2 text-[#A0A0A0] hover:bg-white/5" href="#">
                <span>👥</span>
                <p className="text-sm font-medium">Clients</p>
              </a>
            </div>
          </div>
          <div className="flex flex-col gap-1">
            <a className="flex items-center gap-3 rounded-lg px-3 py-2 text-[#A0A0A0] hover:bg-white/5" href="#">
              <span>⚙️</span>
              <p className="text-sm font-medium">Settings</p>
            </a>
            <a className="flex items-center gap-3 rounded-lg px-3 py-2 text-[#A0A0A0] hover:bg-white/5" href="#">
              <span>❓</span>
              <p className="text-sm font-medium">Help</p>
            </a>
            <div className="mt-4 border-t border-white/10 pt-4">
              <div className="flex items-center gap-3">
                <div className="h-10 w-10 rounded-full bg-gradient-to-r from-cyan-500 to-purple-600"></div>
                <div className="flex flex-col">
                  <h1 className="text-sm font-medium text-white">Eleanor Vance</h1>
                  <p className="text-xs text-[#A0A0A0]">Financial Advisor</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 overflow-y-auto p-8">
        <div className="mx-auto max-w-7xl">
          {/* PageHeading */}
          <header className="mb-8">
            <h1 className="bg-gradient-to-r from-cyan-400 to-purple-500 bg-clip-text text-4xl font-black text-transparent">
              AI Portfolio Rebalancing Alpha
            </h1>
            <p className="text-[#A0A0A0] text-base">Monitor and manage client portfolios with AI-powered insights.</p>
          </header>

          {/* Stats */}
          <section className="mb-8 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
            <div className="flex flex-col gap-2 rounded-xl bg-[#252A31] p-6">
              <p className="text-base font-medium text-[#A0A0A0]">Total AUM</p>
              <p className="text-3xl font-bold text-white">${(mockStats.totalAUM / 1000000000).toFixed(1)}B</p>
              <p className="flex items-center text-sm font-medium text-[#00D18F]">
                ↑ +{mockStats.aumChange}%
              </p>
            </div>
            <div className="flex flex-col gap-2 rounded-xl bg-[#252A31] p-6">
              <p className="text-base font-medium text-[#A0A0A0]">Avg Drift</p>
              <p className="text-3xl font-bold text-white">{mockStats.avgDrift}%</p>
              <p className="flex items-center text-sm font-medium text-[#00D18F]">
                ↓ {mockStats.driftChange}%
              </p>
            </div>
            <div className="flex flex-col gap-2 rounded-xl bg-[#252A31] p-6">
              <p className="text-base font-medium text-[#A0A0A0]">Tax Saved YTD</p>
              <p className="text-3xl font-bold text-white">${(mockStats.taxSavedYTD / 1000000).toFixed(1)}M</p>
              <p className="flex items-center text-sm font-medium text-[#00D18F]">
                ↑ +{mockStats.taxSavedChange}%
              </p>
            </div>
            <div className="flex flex-col gap-2 rounded-xl bg-[#252A31] p-6">
              <p className="text-base font-medium text-[#A0A0A0]">Portfolios</p>
              <p className="text-3xl font-bold text-white">{mockPortfolios.length}</p>
              <p className="text-sm font-medium text-[#FFC107]">{mockStats.needsRebalance} need rebalance</p>
            </div>
          </section>

          {/* SectionHeader */}
          <h2 className="mb-4 text-2xl font-bold tracking-tight text-white">Portfolios at Risk</h2>

          {/* Portfolio Cards Grid */}
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2 xl:grid-cols-3">
            {mockPortfolios.map((portfolio) => {
              const statusInfo = getStatusLabel(portfolio.status);
              const isHealthy = portfolio.status === 'healthy';
              return (
                <div
                  key={portfolio.id}
                  className="flex flex-col rounded-xl bg-[#252A31] p-6 shadow-lg transition-all hover:ring-2 hover:ring-purple-500/50"
                >
                  <div className="mb-4 flex items-start justify-between">
                    <div>
                      <p className="text-lg font-bold text-white">Vanguard Growth Portfolio</p>
                      <p className="text-sm text-[#A0A0A0]">Client: {portfolio.clientName}</p>
                    </div>
                    <div
                      className={`${styles.statusInfo} flex items-center gap-1 rounded-full px-2 py-1 text-xs font-medium`}
                    >
                      <span>{statusInfo.icon === 'warning' ? '⚠️' : statusInfo.icon === 'error' ? '⚠️' : '✓'}</span>
                      {statusInfo.label}
                    </div>
                  </div>
                  <div className="mb-4 flex justify-between text-sm">
                    <div>
                      <span className="text-[#A0A0A0]">AUM:</span> ${(portfolio.aum / 1000000).toFixed(1)}M
                    </div>
                    <div>
                      <span className="text-[#A0A0A0]">Holdings:</span> {portfolio.holdings}
                    </div>
                    <div>
                      <span className="text-[#A0A0A0]">Tax Saved:</span> ${(portfolio.taxSaved || 0) / 1000}k
                    </div>
                  </div>
                  <div className="mb-6">
                    <div className="flex items-center justify-between text-sm">
                      <span className="font-medium text-[#A0A0A0]">Drift</span>
                      <span className={`${styles.driftText} font-bold`}>
                        {portfolio.drift}%
                      </span>
                    </div>
                    <div className="mt-1 h-2 w-full rounded-full bg-white/10">
                      <div
                        className={`${styles.driftBar} ${getDriftClass(portfolio.drift)} h-2 rounded-full`}
                        style={{ '--drift-width': `${Math.min(portfolio.drift * 10, 100)}%` } as React.CSSProperties}
                      ></div>
                    </div>
                  </div>
                  <button
                    onClick={() => handleRebalance(portfolio)}
                    disabled={isHealthy}
                    className={`flex w-full items-center justify-center overflow-hidden rounded-lg h-10 px-4 text-sm font-bold transition-transform ${
                      isHealthy
                        ? 'cursor-not-allowed text-[#A0A0A0] bg-white/10'
                        : 'text-black bg-gradient-to-r from-cyan-400 to-purple-500 hover:scale-105'
                    }`}
                  >
                    <span className="mr-2">✨</span>
                    <span className="truncate">{isHealthy ? 'Rebalance Not Needed' : 'AI Alpha Rebalance'}</span>
                  </button>
                </div>
              );
            })}
          </div>
        </div>
      </main>

      {/* Rebalance Modal */}
      {selectedModal && selectedPlan && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm">
          <div className="w-full max-w-4xl rounded-xl bg-[#252A31] shadow-2xl max-h-[90vh] flex flex-col">
            <header className="flex items-center justify-between border-b border-white/10 p-6 flex-shrink-0">
              <h3 className="text-xl font-bold text-white">
                Rebalance Plan for <span className="bg-gradient-to-r from-cyan-400 to-purple-500 bg-clip-text text-transparent">
                  Vanguard Growth Portfolio
                </span>
              </h3>
              <button className="text-[#A0A0A0] hover:text-white" onClick={() => setSelectedModal(null)}>
                ✕
              </button>
            </header>
            <div className="overflow-y-auto p-6 space-y-6">
              {/* Plan Details */}
              <div className="rounded-lg border border-white/10 bg-white/5 p-4">
                <div className="mb-4 grid grid-cols-3 gap-4 text-center">
                  <div>
                    <p className="text-sm text-[#A0A0A0]">Current Drift</p>
                    <p className="text-lg font-bold text-[#FF4D4D]">{selectedPlan.currentDrift}%</p>
                  </div>
                  <div>
                    <p className="text-sm text-[#A0A0A0]">Expected Drift</p>
                    <p className="text-lg font-bold text-[#00D18F]">{selectedPlan.expectedDrift}%</p>
                  </div>
                  <div>
                    <p className="text-sm text-[#A0A0A0]">Est. Tax Savings</p>
                    <p className="text-lg font-bold text-[#00D18F]">${(selectedPlan.taxSavings / 1000).toFixed(1)}k</p>
                  </div>
                </div>
                <div className="mb-4 rounded-lg bg-[#1A1D21] p-4">
                  <p className="mb-2 font-bold text-white">AI Rationale</p>
                  <p className="text-sm text-[#A0A0A0]">{selectedPlan.rationale}</p>
                </div>
                <div>
                  <p className="mb-2 font-bold text-white">Proposed Trades</p>
                  <div className="space-y-2">
                    {selectedPlan.trades.map((trade, idx) => (
                      <div key={idx} className="grid grid-cols-4 items-center gap-4 rounded-md bg-black/20 p-2 text-sm">
                        <div className="font-medium text-white">
                          {trade.action} <span className="text-[#A0A0A0]">{trade.symbol}</span>
                        </div>
                        <div className="text-right text-[#A0A0A0]">{trade.shares} Shares</div>
                        <div className="text-right text-[#A0A0A0]">${(trade.value / 1000).toFixed(1)}k</div>
                        <div
                          className={`${styles.tradeAction} ${trade.action === 'SELL' ? styles.sellAction : styles.buyAction} flex items-center justify-end font-bold`}
                        >
                          {trade.action === 'SELL' ? '➖' : '➕'} {trade.action}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>
            <footer className="flex-shrink-0 border-t border-white/10 p-4 flex justify-end gap-3">
              <button
                className="px-4 py-2 rounded-lg text-sm font-medium text-[#A0A0A0] bg-white/10 hover:bg-white/20"
                onClick={() => setSelectedModal(null)}
              >
                Decline Plan
              </button>
              <button
                className="px-4 py-2 rounded-lg text-sm font-bold text-black bg-gradient-to-r from-cyan-400 to-purple-500 hover:scale-105 transition-transform"
                onClick={handleExecutePlan}
              >
                Execute Plan
              </button>
            </footer>
          </div>
        </div>
      )}
    </div>
  );
};

export default AIPortfolioRebalancer;
