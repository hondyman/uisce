import React, { useState } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar, Radar, RadarChart, PolarGrid, PolarAngleAxis, PolarRadiusAxis, Legend } from 'recharts';
import { TrendingUp, AlertCircle, DollarSign, Activity, CheckCircle, XCircle, HelpCircle } from 'lucide-react';
import { GenerativeTradeExplanation } from '../components/GenerativeTradeExplanation';

// --- Mock Data for Monte Carlo & Factors ---

const monteCarloData = [
  { range: '5th %', value: -1200, fill: '#f87171' },
  { range: 'Median', value: -1800, fill: '#60a5fa' },
  { range: '95th %', value: -2200, fill: '#34d399' },
];

const factorData = [
  { subject: 'Size', A: 0.5, B: 0.5, C: 0.5, fullMark: 1 },
  { subject: 'Value', A: 0.3, B: 0.3, C: 0.3, fullMark: 1 },
  { subject: 'Momentum', A: 0.2, B: 0.2, C: 0.2, fullMark: 1 },
  { subject: 'Quality', A: 0.6, B: 0.6, C: 0.6, fullMark: 1 },
  { subject: 'Volatility', A: 0.4, B: 0.4, C: 0.4, fullMark: 1 },
];

const mockPerformanceData = [
  { date: '2024-01', portfolio: 100, benchmark: 100 },
  { date: '2024-02', portfolio: 102, benchmark: 101.5 },
  { date: '2024-03', portfolio: 101, benchmark: 101.2 },
  { date: '2024-04', portfolio: 104, benchmark: 103 },
  { date: '2024-05', portfolio: 106, benchmark: 104.5 },
];

const mockTrades = [
  { id: 't1', ticker: 'IVV', action: 'SELL', quantity: 50, price: 450.00, reason: 'Reduce Overweight', date: '2024-05-23' },
  { id: 't2', ticker: 'BND', action: 'SELL', quantity: 100, price: 70.00, reason: 'Harvest Loss', date: '2024-05-23' },
  { id: 't3', ticker: 'VOO', action: 'BUY', quantity: 20, price: 400.00, reason: 'Factor Replacement', date: '2024-05-23' },
  { id: 't4', ticker: 'SPY', action: 'BUY', quantity: 15, price: 500.00, reason: 'Factor Replacement', date: '2024-05-23' },
];

export const AdvisorDashboard: React.FC = () => {
  const [selectedTrade, setSelectedTrade] = useState<string | null>(null);

  return (
    <div className="p-6 max-w-7xl mx-auto bg-white min-h-screen">
      <header className="mb-8 flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-slate-900">Portfolio Command Center</h1>
          <p className="text-slate-500">Household: Smith Family Trust (HH-8821)</p>
        </div>
        <div className="flex gap-2">
            <button className="px-4 py-2 bg-green-600 text-white rounded-lg font-medium hover:bg-green-700 flex items-center gap-2">
                <CheckCircle className="w-4 h-4" /> Approve Proposal
            </button>
            <button className="px-4 py-2 bg-red-100 text-red-700 rounded-lg font-medium hover:bg-red-200 flex items-center gap-2">
                <XCircle className="w-4 h-4" /> Reject
            </button>
        </div>
      </header>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
        <div className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm">
          <div className="flex justify-between items-start">
            <div>
              <p className="text-sm font-medium text-slate-500">Total AUM</p>
              <h3 className="text-2xl font-bold text-slate-900">$2,450,000</h3>
            </div>
            <div className="p-2 bg-green-100 rounded-lg">
              <DollarSign className="w-5 h-5 text-green-600" />
            </div>
          </div>
        </div>

        <div className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm">
          <div className="flex justify-between items-start">
            <div>
              <p className="text-sm font-medium text-slate-500">Est. Tax Benefit</p>
              <h3 className="text-2xl font-bold text-green-600">$1,800</h3>
            </div>
            <div className="p-2 bg-blue-100 rounded-lg">
              <TrendingUp className="w-5 h-5 text-blue-600" />
            </div>
          </div>
          <p className="text-xs text-slate-400 mt-2">Median from Monte Carlo</p>
        </div>

        <div className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm">
          <div className="flex justify-between items-start">
            <div>
              <p className="text-sm font-medium text-slate-500">Tracking Error</p>
              <h3 className="text-2xl font-bold text-slate-900">1.30%</h3>
            </div>
            <div className="p-2 bg-purple-100 rounded-lg">
              <Activity className="w-5 h-5 text-purple-600" />
            </div>
          </div>
          <p className="text-xs text-slate-400 mt-2">Reduced from 2.0%</p>
        </div>

        <div className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm">
          <div className="flex justify-between items-start">
            <div>
              <p className="text-sm font-medium text-slate-500">Confidence</p>
              <h3 className="text-2xl font-bold text-slate-900">95%</h3>
            </div>
            <div className="p-2 bg-orange-100 rounded-lg">
              <CheckCircle className="w-5 h-5 text-orange-600" />
            </div>
          </div>
          <p className="text-xs text-slate-400 mt-2">Based on 1000 runs</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8 mb-8">
        {/* Monte Carlo Chart */}
        <div className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm">
            <h3 className="text-lg font-semibold mb-4">Tax Impact Distribution (Monte Carlo)</h3>
            <div className="h-64">
                <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={monteCarloData}>
                        <CartesianGrid strokeDasharray="3 3" vertical={false} />
                        <XAxis dataKey="range" />
                        <YAxis />
                        <Tooltip formatter={(value) => `$${value}`} />
                        <Bar dataKey="value" fill="#8884d8" />
                    </BarChart>
                </ResponsiveContainer>
            </div>
            <p className="text-xs text-center text-slate-500 mt-2">80% Confidence Interval: -$1,500 to -$2,000</p>
        </div>

        {/* Factor Radar Chart */}
        <div className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm">
            <h3 className="text-lg font-semibold mb-4">Factor Similarity</h3>
            <div className="h-64">
                <ResponsiveContainer width="100%" height="100%">
                    <RadarChart cx="50%" cy="50%" outerRadius="80%" data={factorData}>
                        <PolarGrid />
                        <PolarAngleAxis dataKey="subject" />
                        <PolarRadiusAxis angle={30} domain={[0, 1]} />
                        <Radar name="IVV (Target)" dataKey="A" stroke="#2563eb" fill="#2563eb" fillOpacity={0.3} />
                        <Radar name="VOO (Rep)" dataKey="B" stroke="#f97316" fill="#f97316" fillOpacity={0.3} />
                        <Radar name="SPY (Rep)" dataKey="C" stroke="#10b981" fill="#10b981" fillOpacity={0.3} />
                        <Legend />
                    </RadarChart>
                </ResponsiveContainer>
            </div>
        </div>

        {/* Performance Chart */}
        <div className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm">
          <h3 className="text-lg font-semibold mb-4">Performance vs Benchmark</h3>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={mockPerformanceData}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="date" />
                <YAxis domain={['auto', 'auto']} />
                <Tooltip />
                <Line type="monotone" dataKey="portfolio" stroke="#2563eb" strokeWidth={2} name="Portfolio" />
                <Line type="monotone" dataKey="benchmark" stroke="#94a3b8" strokeWidth={2} strokeDasharray="5 5" name="S&P 500" />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>
        {/* Values & Constraints Panel */}
        <div className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm">
            <h3 className="text-lg font-semibold mb-4">Values & Constraints</h3>
            <div className="space-y-4">
                <div>
                    <div className="flex justify-between text-sm mb-1">
                        <span className="text-slate-600">Tax Budget Used</span>
                        <span className="font-medium text-slate-900">$12,500 / $20,000</span>
                    </div>
                    <div className="w-full bg-slate-100 rounded-full h-2">
                        <div className="bg-blue-600 h-2 rounded-full" style={{ width: '62.5%' }}></div>
                    </div>
                </div>
                
                <div className="pt-4 border-t border-slate-100">
                    <h4 className="text-sm font-medium text-slate-700 mb-2">Active Screens</h4>
                    <div className="flex flex-wrap gap-2">
                        <span className="px-2 py-1 bg-red-50 text-red-700 text-xs rounded-md border border-red-100 flex items-center gap-1">
                            <XCircle className="w-3 h-3" /> No Tobacco
                        </span>
                        <span className="px-2 py-1 bg-red-50 text-red-700 text-xs rounded-md border border-red-100 flex items-center gap-1">
                            <XCircle className="w-3 h-3" /> No Weapons
                        </span>
                        <span className="px-2 py-1 bg-green-50 text-green-700 text-xs rounded-md border border-green-100 flex items-center gap-1">
                            <CheckCircle className="w-3 h-3" /> Low Carbon
                        </span>
                    </div>
                </div>

                <div className="pt-4 border-t border-slate-100">
                    <div className="flex items-start gap-2">
                        <HelpCircle className="w-4 h-4 text-slate-400 mt-0.5" />
                        <p className="text-xs text-slate-500">
                            Constraints are applied before optimization. Tax budget limits realized gains across all tax lots.
                        </p>
                    </div>
                </div>
            </div>
        </div>

      </div>

      {/* Recent Activity Feed */}
      <div className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm">
          <h3 className="text-lg font-semibold mb-4">Proposed Executions</h3>
          <div className="space-y-4">
            {mockTrades.map((trade) => (
              <div 
                key={trade.id} 
                className={`p-4 rounded-lg border cursor-pointer transition-colors ${selectedTrade === trade.id ? 'border-blue-500 bg-blue-50' : 'border-slate-100 hover:bg-slate-50'}`}
                onClick={() => setSelectedTrade(trade.id === selectedTrade ? null : trade.id)}
              >
                <div className="flex justify-between items-center mb-2">
                  <div className="flex items-center gap-2">
                    <span className={`px-2 py-1 rounded text-xs font-bold ${trade.action === 'BUY' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>
                      {trade.action}
                    </span>
                    <span className="font-semibold">{trade.ticker}</span>
                  </div>
                  <span className="text-sm text-slate-500">{trade.date}</span>
                </div>
                <div className="flex justify-between text-sm text-slate-600 mb-2">
                  <span>{trade.quantity} shares @ ${trade.price}</span>
                  <span className="font-medium">${(trade.quantity * trade.price).toLocaleString()}</span>
                </div>
                
                {selectedTrade === trade.id && (
                  <div className="mt-3 pt-3 border-t border-slate-200">
                    <GenerativeTradeExplanation 
                      tradeId={trade.id} 
                      ticker={trade.ticker} 
                      action={trade.action as 'BUY' | 'SELL'} 
                      reason={trade.reason} 
                    />
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
    </div>
  );
};

export default AdvisorDashboard;
