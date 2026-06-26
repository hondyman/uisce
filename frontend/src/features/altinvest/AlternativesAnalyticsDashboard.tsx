import React, { useState, useEffect } from 'react';
import { TrendingUp, Target, DollarSign, LineChart as LineChartIcon, Award, AlertTriangle } from 'lucide-react';
import { LineChart, Line, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, ScatterChart, Scatter } from 'recharts';

interface AlternativeInvestment {
  id: string;
  fundName: string;
  assetClass: string;
  vintageYear: number;
  commitmentAmount: number;
  fundedAmount: number;
  unfundedCommitment: number;
  currentNav: number;
  performanceMetrics: {
    irr: number;
    moic: number;
    tvpi: number;
    dpi: number;
    rvpi: number;
    pme_ks?: number;
    vintage_quartile?: number;
  };
}

interface PMEBenchmark {
  investmentId: string;
  fundName: string;
  pmeKaplanSchoar: number;
  pmeDirectAlpha: number;
  benchmarkIndex: string;
  peerMedianIRR: number;
  peerTopQuartileIRR: number;
  fundPercentileRank: number;
}

export const AlternativesAnalyticsDashboard: React.FC = () => {
  const [investments, setInvestments] = useState<AlternativeInvestment[]>([]);
  const [benchmarks, setBenchmarks] = useState<PMEBenchmark[]>([]);
  const [selectedView, setSelectedView] = useState<'performance' | 'pme' | 'cashflow'>('performance');

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      const [invRes, benchRes] = await Promise.all([
        fetch('/api/alternatives/investments'),
        fetch('/api/alternatives/benchmarks'),
      ]);

      setInvestments(await invRes.json());
      setBenchmarks(await benchRes.json());
    } catch (error) {
      console.error('Failed to fetch alternatives data:', error);
    }
  };

  const formatCurrency = (value: number) => {
    if (value >= 1000000) return `$${(value / 1000000).toFixed(1)}M`;
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(value);
  };

  const formatPercent = (value: number) => {
    return `${(value * 100).toFixed(1)}%`;
  };

  // Aggregate stats
  const totalCommitment = investments.reduce((sum, inv) => sum + inv.commitmentAmount, 0);
  const totalFunded = investments.reduce((sum, inv) => sum + inv.fundedAmount, 0);
  const totalUnfunded = investments.reduce((sum, inv) => sum + inv.unfundedCommitment, 0);
  const totalNav = investments.reduce((sum, inv) => sum + inv.currentNav, 0);
  const avgIRR = investments.reduce((sum, inv) => sum + (inv.performanceMetrics.irr || 0), 0) / (investments.length || 1);

  // PME comparison data
  const pmeData = benchmarks.map((b) => ({
    name: b.fundName.substring(0, 20),
    pme: b.pmeKaplanSchoar,
    benchmark: 1.0,
  }));

  // Vintage year analysis
  const vintageData = investments.reduce((acc: any, inv) => {
    const year = inv.vintageYear;
    if (!acc[year]) {
      acc[year] = { year, avgIRR: 0, count: 0, totalNAV: 0 };
    }
    acc[year].avgIRR += inv.performanceMetrics.irr || 0;
    acc[year].count += 1;
    acc[year].totalNAV += inv.currentNav;
    return acc;
  }, {});

  const vintageChartData = Object.values(vintageData).map((v: any) => ({
    year: v.year,
    avgIRR: (v.avgIRR / v.count) * 100,
    totalNAV: v.totalNAV,
  })).sort((a: any, b: any) => a.year - b.year);

  return (
    <div className="max-w-7xl mx-auto p-6">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900">Alternative Investments Analytics</h1>
        <p className="text-gray-600 mt-1">Performance benchmarking and portfolio analytics</p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-5 gap-6 mb-8">
        <div className="bg-gradient-to-br from-indigo-500 to-purple-600 text-white rounded-xl p-6">
          <div className="flex items-center gap-3 mb-2">
            <DollarSign className="w-8 h-8 opacity-80" />
          </div>
          <p className="text-3xl font-bold">{formatCurrency(totalCommitment)}</p>
          <p className="text-indigo-100 text-sm mt-1">Total Commitments</p>
        </div>

        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <div className="flex items-center gap-3 mb-2">
            <div className="p-2 bg-green-100 rounded-lg">
              <Target className="w-6 h-6 text-green-600" />
            </div>
          </div>
          <p className="text-3xl font-bold text-gray-900">{formatCurrency(totalNav)}</p>
          <p className="text-sm text-gray-600 mt-1">Current NAV</p>
        </div>

        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <div className="flex items-center gap-3 mb-2">
            <div className="p-2 bg-orange-100 rounded-lg">
              <AlertTriangle className="w-6 h-6 text-orange-600" />
            </div>
          </div>
          <p className="text-3xl font-bold text-gray-900">{formatCurrency(totalUnfunded)}</p>
          <p className="text-sm text-gray-600 mt-1">Unfunded</p>
        </div>

        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <div className="flex items-center gap-3 mb-2">
            <div className="p-2 bg-blue-100 rounded-lg">
              <TrendingUp className="w-6 h-6 text-blue-600" />
            </div>
          </div>
          <p className="text-3xl font-bold text-gray-900">{formatPercent(avgIRR)}</p>
          <p className="text-sm text-gray-600 mt-1">Avg IRR</p>
        </div>

        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <div className="flex items-center gap-3 mb-2">
            <div className="p-2 bg-purple-100 rounded-lg">
              <Award className="w-6 h-6 text-purple-600" />
            </div>
          </div>
          <p className="text-3xl font-bold text-gray-900">{investments.length}</p>
          <p className="text-sm text-gray-600 mt-1">Total Funds</p>
        </div>
      </div>

      {/* View Selector */}
      <div className="flex gap-4 border-b border-gray-200 mb-6">
        {(['performance', 'pme', 'cashflow'] as const).map((view) => (
          <button
            key={view}
            onClick={() => setSelectedView(view)}
            className={`px-4 py-3 border-b-2 transition-colors font-medium capitalize ${
              selectedView === view
                ? 'border-indigo-600 text-indigo-600'
                : 'border-transparent text-gray-600 hover:text-gray-900'
            }`}
          >
            {view === 'pme' ? 'PME Benchmarking' : view}
          </button>
        ))}
      </div>

      {/* Performance View */}
      {selectedView === 'performance' && (
        <div className="space-y-6">
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Vintage Year Performance</h3>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={vintageChartData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="year" />
                <YAxis yAxisId="left" label={{ value: 'IRR (%)', angle: -90, position: 'insideLeft' }} />
                <YAxis yAxisId="right" orientation="right" label={{ value: 'NAV ($M)', angle: 90, position: 'insideRight' }} />
                <Tooltip formatter={(value: number, name: string) => 
                  name === 'avgIRR' ? `${value.toFixed(1)}%` : formatCurrency(value)
                } />
                <Legend />
                <Bar yAxisId="left" dataKey="avgIRR" fill="#6366f1" name="Avg IRR" />
                <Bar yAxisId="right" dataKey="totalNAV" fill="#10b981" name="Total NAV" />
              </BarChart>
            </ResponsiveContainer>
          </div>

          <div className="bg-white rounded-xl border border-gray-200">
            <div className="p-6 border-b border-gray-200">
              <h3 className="text-lg font-semibold text-gray-900">Investment Performance</h3>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-gray-50 border-b border-gray-200">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Fund</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Asset Class</th>
                    <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">IRR</th>
                    <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">MOIC</th>
                    <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">TVPI</th>
                    <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">DPI</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Quartile</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200">
                  {investments.map((inv) => {
                    const quartile = inv.performanceMetrics.vintage_quartile || 0;
                    const quartileColor = quartile === 1 ? 'text-green-600 bg-green-100' :
                                         quartile === 2 ? 'text-blue-600 bg-blue-100' :
                                         'text-gray-600 bg-gray-100';

                    return (
                      <tr key={inv.id} className="hover:bg-gray-50">
                        <td className="px-6 py-4 font-medium text-gray-900">{inv.fundName}</td>
                        <td className="px-6 py-4 text-sm text-gray-600 capitalize">{inv.assetClass.replace('_', ' ')}</td>
                        <td className="px-6 py-4 text-right font-semibold">{formatPercent(inv.performanceMetrics.irr || 0)}</td>
                        <td className="px-6 py-4 text-right">{(inv.performanceMetrics.moic || 0).toFixed(2)}x</td>
                        <td className="px-6 py-4 text-right">{(inv.performanceMetrics.tvpi || 0).toFixed(2)}</td>
                        <td className="px-6 py-4 text-right">{(inv.performanceMetrics.dpi || 0).toFixed(2)}</td>
                        <td className="px-6 py-4">
                          {quartile > 0 && (
                            <span className={`px-2 py-1 rounded-full text-xs font-medium ${quartileColor}`}>
                              Q{quartile}
                            </span>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      {/* PME Benchmarking View */}
      {selectedView === 'pme' && (
        <div className="space-y-6">
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Public Market Equivalent (PME) vs. S&P 500</h3>
            <p className="text-sm text-gray-600 mb-4">PME {'>'} 1.0 indicates outperformance vs. public markets</p>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={pmeData} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis type="number" domain={[0, 'dataMax + 0.5']} />
                <YAxis dataKey="name" type="category" width={150} />
                <Tooltip />
                <Legend />
                <Bar dataKey="pme" fill="#6366f1" name="PME Ratio" />
                <Bar dataKey="benchmark" fill="#94a3b8" name="Market Benchmark" />
              </BarChart>
            </ResponsiveContainer>
          </div>

          <div className="bg-white rounded-xl border border-gray-200">
            <div className="p-6 border-b border-gray-200">
              <h3 className="text-lg font-semibold text-gray-900">PME Analysis & Peer Comparison</h3>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-gray-50 border-b border-gray-200">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Fund</th>
                    <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">PME K-S</th>
                    <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Direct Alpha</th>
                    <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Peer Median</th>
                    <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Top Quartile</th>
                    <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Percentile</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200">
                  {benchmarks.map((bench) => {
                    const outperforming = bench.pmeKaplanSchoar > 1.0;
                    const topPerformer = bench.fundPercentileRank >= 75;

                    return (
                      <tr key={bench.investmentId} className="hover:bg-gray-50">
                        <td className="px-6 py-4 font-medium text-gray-900">{bench.fundName}</td>
                        <td className={`px-6 py-4 text-right font-bold ${outperforming ? 'text-green-600' : 'text-red-600'}`}>
                          {bench.pmeKaplanSchoar.toFixed(2)}
                        </td>
                        <td className={`px-6 py-4 text-right ${bench.pmeDirectAlpha >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                          {formatCurrency(bench.pmeDirectAlpha)}
                        </td>
                        <td className="px-6 py-4 text-right">{formatPercent(bench.peerMedianIRR)}</td>
                        <td className="px-6 py-4 text-right">{formatPercent(bench.peerTopQuartileIRR)}</td>
                        <td className="px-6 py-4 text-right">
                          <span className={`px-2 py-1 rounded-full text-xs font-medium ${
                            topPerformer ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-700'
                          }`}>
                            {bench.fundPercentileRank}th
                          </span>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
