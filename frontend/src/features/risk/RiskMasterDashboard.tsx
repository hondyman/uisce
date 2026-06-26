import React, { useState } from 'react';
import { Activity, Gauge, BarChart3, CloudLightning, Filter, Download, ArrowUpRight, ArrowDownRight } from 'lucide-react';

const RiskMasterDashboard: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'factors' | 'exposures' | 'portfolio' | 'scenarios'>('portfolio');

  const navItems = [
    { id: 'portfolio', label: 'Portfolio Risk', icon: Gauge },
    { id: 'factors', label: 'Risk Factors', icon: Activity },
    { id: 'exposures', label: 'Factor Exposures', icon: BarChart3 },
    { id: 'scenarios', label: 'Stress Scenarios', icon: CloudLightning },
  ] as const;

  return (
    <div className="flex flex-col h-full bg-slate-900 text-slate-200 p-6 overflow-hidden">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-white flex items-center gap-2">
            <Gauge className="w-6 h-6 text-amber-500" />
            Risk Engine
          </h1>
          <p className="text-sm text-slate-400 mt-1">
            Factor models, VaR calculation, and stress scenario analysis
          </p>
        </div>
        <div className="flex gap-3">
          <button className="flex items-center gap-2 px-4 py-2 bg-slate-800 hover:bg-slate-700 text-white rounded-md text-sm font-medium transition-colors border border-slate-700">
            <Download className="w-4 h-4" />
            Export Report
          </button>
        </div>
      </div>

      <div className="flex bg-slate-800/50 p-1 rounded-lg border border-slate-700 w-fit mb-6">
        {navItems.map(({ id, label, icon: Icon }) => (
          <button
            key={id}
            onClick={() => setActiveTab(id)}
            className={`flex items-center gap-2 px-6 py-2.5 rounded-md text-sm font-medium transition-all ${
              activeTab === id
                ? 'bg-slate-700 text-white shadow-sm border border-slate-600'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-700/50 border border-transparent'
            }`}
          >
            <Icon className="w-4 h-4" />
            {label}
          </button>
        ))}
      </div>

      <div className="flex-1 bg-slate-800 rounded-lg border border-slate-700 overflow-hidden flex flex-col">
        {/* Mock Data Panels */}
        {activeTab === 'portfolio' && <PortfolioRiskPanel />}
        {activeTab === 'factors' && <RiskFactorsPanel />}
        {activeTab === 'exposures' && <ExposuresPanel />}
        {activeTab === 'scenarios' && <ScenariosPanel />}
      </div>
    </div>
  );
};

const PortfolioRiskPanel = () => (
  <div className="p-6">
    <div className="flex items-center justify-between mb-6">
      <h2 className="text-lg font-semibold text-white">Aggregated Portfolio Risk</h2>
      <select className="bg-slate-900 border border-slate-700 text-slate-200 text-sm rounded-md px-3 py-1.5 focus:ring-amber-500 focus:border-amber-500">
        <option>Global Equity Fund</option>
        <option>US Growth Fund</option>
        <option>Fixed Income Strategy</option>
      </select>
    </div>

    <div className="grid grid-cols-4 gap-4 mb-6">
      <div className="bg-slate-900 border border-slate-700 rounded-lg p-4">
        <div className="text-slate-400 text-sm mb-1">Total Volatility</div>
        <div className="text-2xl font-bold text-white flex items-center gap-2">
          14.2% <ArrowUpRight className="w-4 h-4 text-emerald-400" />
        </div>
      </div>
      <div className="bg-slate-900 border border-slate-700 rounded-lg p-4">
        <div className="text-slate-400 text-sm mb-1">Tracking Error</div>
        <div className="text-2xl font-bold text-white flex items-center gap-2">
          2.1% <ArrowDownRight className="w-4 h-4 text-emerald-400" />
        </div>
      </div>
      <div className="bg-slate-900 border border-slate-700 rounded-lg p-4">
        <div className="text-slate-400 text-sm mb-1">VaR (95%)</div>
        <div className="text-2xl font-bold text-red-400">
          $1.42M
        </div>
      </div>
      <div className="bg-slate-900 border border-slate-700 rounded-lg p-4">
        <div className="text-slate-400 text-sm mb-1">Expected Shortfall</div>
        <div className="text-2xl font-bold text-red-500">
          $2.15M
        </div>
      </div>
    </div>

    <div className="overflow-x-auto border border-slate-700 rounded-lg bg-slate-900">
      <table className="w-full text-left text-sm">
        <thead className="bg-slate-800 text-slate-400">
          <tr>
            <th className="px-4 py-3 font-medium">Factor Name</th>
            <th className="px-4 py-3 font-medium">Marginal Contribution to Risk</th>
            <th className="px-4 py-3 font-medium">% of Total Variance</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-700/50">
          <tr className="hover:bg-slate-800/50">
            <td className="px-4 py-3 text-amber-400">Equity Market</td>
            <td className="px-4 py-3">8.5%</td>
            <td className="px-4 py-3">65.2%</td>
          </tr>
          <tr className="hover:bg-slate-800/50">
            <td className="px-4 py-3 text-amber-400">Size Factor</td>
            <td className="px-4 py-3">2.1%</td>
            <td className="px-4 py-3">15.4%</td>
          </tr>
          {/* Add more mock rows if needed */}
        </tbody>
      </table>
    </div>
  </div>
);

const RiskFactorsPanel = () => (
  <div className="p-6">
    <div className="flex items-center justify-between mb-4">
      <h2 className="text-lg font-semibold text-white">Systematic Risk Factors</h2>
      <button className="flex items-center gap-2 px-3 py-1.5 bg-slate-700 hover:bg-slate-600 rounded text-sm text-white">
        <Filter className="w-4 h-4" /> Filter
      </button>
    </div>
    <div className="overflow-x-auto border border-slate-700 rounded-lg bg-slate-900">
      <table className="w-full text-left text-sm">
        <thead className="bg-slate-800 text-slate-400">
          <tr>
            <th className="px-4 py-3 font-medium">Code</th>
            <th className="px-4 py-3 font-medium">Name</th>
            <th className="px-4 py-3 font-medium">Category</th>
            <th className="px-4 py-3 font-medium">Type</th>
            <th className="px-4 py-3 font-medium">Unit</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-700/50">
          <tr className="hover:bg-slate-800/50">
            <td className="px-4 py-3 font-mono text-amber-400">EQUITY_MKT</td>
            <td className="px-4 py-3 text-white">Equity Market</td>
            <td className="px-4 py-3">EQUITY</td>
            <td className="px-4 py-3">SYSTEMATIC</td>
            <td className="px-4 py-3">%</td>
          </tr>
          <tr className="hover:bg-slate-800/50">
            <td className="px-4 py-3 font-mono text-amber-400">CREDIT_SPREAD</td>
            <td className="px-4 py-3 text-white">Credit Spread</td>
            <td className="px-4 py-3">FIXED_INCOME</td>
            <td className="px-4 py-3">SYSTEMATIC</td>
            <td className="px-4 py-3">BP</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
);

const ExposuresPanel = () => (
  <div className="p-6 flex items-center justify-center text-slate-400 h-64">
    Select a security to view fundamental and statistical factor exposures.
  </div>
);

const ScenariosPanel = () => (
  <div className="p-6">
    <div className="flex items-center justify-between mb-4">
      <h2 className="text-lg font-semibold text-white">Stress Scenarios & Results</h2>
    </div>
    <div className="overflow-x-auto border border-slate-700 rounded-lg bg-slate-900">
      <table className="w-full text-left text-sm">
        <thead className="bg-slate-800 text-slate-400">
          <tr>
            <th className="px-4 py-3 font-medium">Scenario Name</th>
            <th className="px-4 py-3 font-medium">Type</th>
            <th className="px-4 py-3 font-medium">Portfolio Impact (Est.)</th>
            <th className="px-4 py-3 font-medium">Status</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-700/50">
          <tr className="hover:bg-slate-800/50">
            <td className="px-4 py-3 font-medium text-white">2008 Financial Crisis</td>
            <td className="px-4 py-3">HISTORICAL</td>
            <td className="px-4 py-3 text-red-400 font-semibold">-18.4%</td>
            <td className="px-4 py-3"><span className="text-emerald-400 text-xs">ACTIVE</span></td>
          </tr>
          <tr className="hover:bg-slate-800/50">
            <td className="px-4 py-3 font-medium text-white">+200bps Rate Shock</td>
            <td className="px-4 py-3">HYPOTHETICAL</td>
            <td className="px-4 py-3 text-amber-400 font-semibold">-4.1%</td>
            <td className="px-4 py-3"><span className="text-emerald-400 text-xs">ACTIVE</span></td>
          </tr>
          <tr className="hover:bg-slate-800/50">
            <td className="px-4 py-3 font-medium text-white">COVID-19 March 2020</td>
            <td className="px-4 py-3">HISTORICAL</td>
            <td className="px-4 py-3 text-red-500 font-bold">-22.1%</td>
            <td className="px-4 py-3"><span className="text-emerald-400 text-xs">ACTIVE</span></td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
);

export default RiskMasterDashboard;
