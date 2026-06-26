import React, { useState, useEffect } from 'react';
import {
  LineChart, Line, AreaChart, Area, BarChart, Bar, XAxis, YAxis,
  CartesianGrid, Tooltip, Legend, ResponsiveContainer, ComposedChart, ScatterChart, Scatter
} from 'recharts';
import {
  TrendingUp, TrendingDown, Target, AlertCircle, CheckCircle, DollarSign, Zap, Eye
} from 'lucide-react';

export default function BacktestDashboard() {
  const [backtests, setBacktests] = useState([]);
  const [selectedBacktest, setSelectedBacktest] = useState(null);
  const [simulationData, setSimulationData] = useState([]);
  const [tab, setTab] = useState('overview');

  // Mock backtest data
  useEffect(() => {
    const mockBacktest = {
      id: 'bt-001',
      recommendationId: 'rec-tax-loss-1',
      portfolioId: 'port-123',
      simulationType: 'HISTORICAL',
      startDate: '2024-01-01',
      endDate: '2024-10-30',
      baselineReturn: 0.087,
      recommendationReturn: 0.124,
      alphaGenerated: 0.037,
      betaAdjustedReturn: 0.108,
      sharpeRatioBaseline: 1.12,
      sharpeRatioRecommended: 1.58,
      maxDrawdownBaseline: -0.18,
      maxDrawdownRecommended: -0.12,
      taxSavingsAccumulated: 4250,
      transactionCosts: 150,
      netBenefit: 27850,
      confidence: 0.92,
      createdAt: '2024-10-30T12:00:00Z',
    };

    setBacktests([mockBacktest]);
    setSelectedBacktest(mockBacktest);

    // Generate simulation data
    const simData = [];
    let baselineValue = 1000000;
    let recValue = 1000000;

    for (let i = 0; i < 300; i++) {
      const date = new Date('2024-01-01');
      date.setDate(date.getDate() + i);

      const baselineReturn = (Math.random() - 0.48) * 0.02;
      const recReturn = baselineReturn + (Math.random() * 0.015 - 0.005);

      baselineValue *= 1 + baselineReturn;
      recValue *= 1 + recReturn;

      simData.push({
        date: date.toLocaleDateString(),
        baseline: Math.round(baselineValue),
        recommendation: Math.round(recValue),
        alpha: recValue - baselineValue,
        day: i,
      });
    }
    setSimulationData(simData);
  }, []);

  if (!selectedBacktest) {
    return <div className="p-8 text-center text-gray-400">Loading backtest results...</div>;
  }

  const alphaPositive = selectedBacktest.alphaGenerated > 0;
  const sharpeImprovement = selectedBacktest.sharpeRatioRecommended - selectedBacktest.sharpeRatioBaseline;
  const drawdownImprovement = selectedBacktest.maxDrawdownRecommended - selectedBacktest.maxDrawdownBaseline;

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-900 via-gray-800 to-gray-900 text-white p-6">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-4xl font-bold bg-gradient-to-r from-blue-400 to-cyan-400 bg-clip-text text-transparent">
            Backtest Analysis
          </h1>
          <p className="text-gray-400 mt-2">Historical simulation of recommendation outcomes</p>
        </div>

        {/* Key Metrics Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
          {/* Alpha Generated */}
          <div className="bg-gray-800/50 backdrop-blur border border-gray-700 rounded-lg p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-gray-400 text-sm">Alpha Generated</p>
                <p className={`text-3xl font-bold ${alphaPositive ? 'text-green-400' : 'text-red-400'}`}>
                  {alphaPositive ? '+' : ''}{(selectedBacktest.alphaGenerated * 100).toFixed(2)}%
                </p>
                <p className="text-xs text-gray-500 mt-1">
                  ${Math.round(selectedBacktest.alphaGenerated * 1000000).toLocaleString()}
                </p>
              </div>
              {alphaPositive ? (
                <TrendingUp className="w-10 h-10 text-green-500" />
              ) : (
                <TrendingDown className="w-10 h-10 text-red-500" />
              )}
            </div>
          </div>

          {/* Sharpe Ratio Improvement */}
          <div className="bg-gray-800/50 backdrop-blur border border-gray-700 rounded-lg p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-gray-400 text-sm">Sharpe Ratio</p>
                <p className="text-2xl font-bold">
                  {selectedBacktest.sharpeRatioBaseline.toFixed(2)} → {selectedBacktest.sharpeRatioRecommended.toFixed(2)}
                </p>
                <p className={`text-xs mt-1 ${sharpeImprovement > 0 ? 'text-green-400' : 'text-red-400'}`}>
                  {sharpeImprovement > 0 ? '+' : ''}{sharpeImprovement.toFixed(2)} improvement
                </p>
              </div>
              <Target className="w-10 h-10 text-blue-500" />
            </div>
          </div>

          {/* Max Drawdown Reduction */}
          <div className="bg-gray-800/50 backdrop-blur border border-gray-700 rounded-lg p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-gray-400 text-sm">Max Drawdown</p>
                <p className="text-2xl font-bold">
                  {(selectedBacktest.maxDrawdownBaseline * 100).toFixed(1)}% → {(selectedBacktest.maxDrawdownRecommended * 100).toFixed(1)}%
                </p>
                <p className={`text-xs mt-1 ${drawdownImprovement > 0 ? 'text-green-400' : 'text-red-400'}`}>
                  {drawdownImprovement > 0 ? '+' : ''}{(drawdownImprovement * 100).toFixed(1)}pp improvement
                </p>
              </div>
              <AlertCircle className="w-10 h-10 text-yellow-500" />
            </div>
          </div>

          {/* Net Benefit */}
          <div className="bg-gray-800/50 backdrop-blur border border-gray-700 rounded-lg p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-gray-400 text-sm">Net Benefit</p>
                <p className="text-3xl font-bold text-green-400">
                  ${(selectedBacktest.netBenefit / 1000).toFixed(1)}K
                </p>
                <p className="text-xs text-gray-500 mt-1">
                  Tax savings: ${selectedBacktest.taxSavingsAccumulated.toLocaleString()}
                </p>
              </div>
              <DollarSign className="w-10 h-10 text-green-500" />
            </div>
          </div>
        </div>

        {/* Confidence Score & Details */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-8">
          <div className="lg:col-span-2 bg-gray-800/50 backdrop-blur border border-gray-700 rounded-lg p-6">
            <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
              <Eye className="w-5 h-5" />
              Backtest Confidence
            </h3>
            <div className="mb-4">
              <div className="flex justify-between mb-2">
                <span className="text-sm text-gray-400">Model Confidence Score</span>
                <span className="font-bold text-lg">{(selectedBacktest.confidence * 100).toFixed(0)}%</span>
              </div>
              <div className="w-full bg-gray-700 rounded-full h-3">
                <div
                  className="bg-gradient-to-r from-blue-500 to-cyan-500 h-3 rounded-full"
                  style={{ width: `${selectedBacktest.confidence * 100}%` }}
                />
              </div>
              <p className="text-xs text-gray-400 mt-3">
                Based on {simulationData.length} days of historical data. Higher confidence with more consistent results.
              </p>
            </div>
          </div>

          <div className="bg-gray-800/50 backdrop-blur border border-gray-700 rounded-lg p-6">
            <h3 className="text-lg font-semibold mb-4">Key Metrics</h3>
            <div className="space-y-3">
              <div className="flex justify-between">
                <span className="text-gray-400">Return (Baseline)</span>
                <span className="font-semibold text-yellow-400">
                  {(selectedBacktest.baselineReturn * 100).toFixed(2)}%
                </span>
              </div>
              <div className="flex justify-between border-t border-gray-700 pt-3">
                <span className="text-gray-400">Return (Recommended)</span>
                <span className="font-semibold text-green-400">
                  {(selectedBacktest.recommendationReturn * 100).toFixed(2)}%
                </span>
              </div>
              <div className="flex justify-between border-t border-gray-700 pt-3">
                <span className="text-gray-400">Transaction Costs</span>
                <span className="font-semibold text-red-400">
                  ${selectedBacktest.transactionCosts.toLocaleString()}
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex gap-4 mb-6 border-b border-gray-700 overflow-x-auto">
          {['overview', 'performance', 'analysis', 'monte-carlo'].map(t => (
            <button
              key={t}
              onClick={() => setTab(t)}
              className={`px-4 py-3 capitalize font-semibold transition-colors whitespace-nowrap ${
                tab === t
                  ? 'border-b-2 border-blue-500 text-blue-400'
                  : 'text-gray-400 hover:text-gray-300'
              }`}
            >
              {t === 'monte-carlo' ? 'Monte Carlo' : t}
            </button>
          ))}
        </div>

        {/* Tab Content */}
        <div className="space-y-6">
          {tab === 'overview' && (
            <div className="bg-gray-800/50 backdrop-blur border border-gray-700 rounded-lg p-6">
              <h3 className="text-lg font-semibold mb-4">Cumulative Returns Comparison</h3>
              <ResponsiveContainer width="100%" height={400}>
                <ComposedChart data={simulationData}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#444" />
                  <XAxis dataKey="day" stroke="#999" tick={{ fontSize: 12 }} />
                  <YAxis stroke="#999" />
                  <Tooltip contentStyle={{ backgroundColor: '#1f2937', border: '1px solid #444' }} />
                  <Legend />
                  <Line
                    type="monotone"
                    dataKey="baseline"
                    stroke="#fbbf24"
                    dot={false}
                    isAnimationActive={false}
                    name="Baseline Portfolio"
                  />
                  <Line
                    type="monotone"
                    dataKey="recommendation"
                    stroke="#34d399"
                    dot={false}
                    isAnimationActive={false}
                    strokeWidth={2}
                    name="With Recommendation"
                  />
                </ComposedChart>
              </ResponsiveContainer>
            </div>
          )}

          {tab === 'performance' && (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <div className="bg-gray-800/50 backdrop-blur border border-gray-700 rounded-lg p-6">
                <h3 className="text-lg font-semibold mb-4">Returns Distribution</h3>
                <ResponsiveContainer width="100%" height={300}>
                  <BarChart
                    data={[
                      { period: 'Baseline', return: selectedBacktest.baselineReturn * 100 },
                      { period: 'Recommended', return: selectedBacktest.recommendationReturn * 100 },
                    ]}
                  >
                    <CartesianGrid strokeDasharray="3 3" stroke="#444" />
                    <XAxis dataKey="period" stroke="#999" />
                    <YAxis stroke="#999" />
                    <Tooltip contentStyle={{ backgroundColor: '#1f2937', border: '1px solid #444' }} />
                    <Bar dataKey="return" fill="#60a5fa" />
                  </BarChart>
                </ResponsiveContainer>
              </div>

              <div className="bg-gray-800/50 backdrop-blur border border-gray-700 rounded-lg p-6">
                <h3 className="text-lg font-semibold mb-4">Risk Metrics Comparison</h3>
                <ResponsiveContainer width="100%" height={300}>
                  <BarChart
                    data={[
                      { metric: 'Sharpe', baseline: selectedBacktest.sharpeRatioBaseline, recommended: selectedBacktest.sharpeRatioRecommended },
                      { metric: 'Max DD', baseline: selectedBacktest.maxDrawdownBaseline * 100, recommended: selectedBacktest.maxDrawdownRecommended * 100 },
                    ]}
                  >
                    <CartesianGrid strokeDasharray="3 3" stroke="#444" />
                    <XAxis dataKey="metric" stroke="#999" />
                    <YAxis stroke="#999" />
                    <Tooltip contentStyle={{ backgroundColor: '#1f2937', border: '1px solid #444' }} />
                    <Legend />
                    <Bar dataKey="baseline" fill="#fbbf24" name="Baseline" />
                    <Bar dataKey="recommended" fill="#34d399" name="Recommended" />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </div>
          )}

          {tab === 'analysis' && (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <div className="bg-gray-800/50 backdrop-blur border border-gray-700 rounded-lg p-6">
                <h3 className="text-lg font-semibold mb-4">Alpha Contribution Over Time</h3>
                <ResponsiveContainer width="100%" height={300}>
                  <AreaChart data={simulationData}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#444" />
                    <XAxis dataKey="day" stroke="#999" />
                    <YAxis stroke="#999" />
                    <Tooltip contentStyle={{ backgroundColor: '#1f2937', border: '1px solid #444' }} />
                    <Area
                      type="monotone"
                      dataKey="alpha"
                      fill="#60a5fa"
                      stroke="#3b82f6"
                      isAnimationActive={false}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              </div>

              <div className="bg-gray-800/50 backdrop-blur border border-gray-700 rounded-lg p-6">
                <h3 className="text-lg font-semibold mb-4">Performance Summary</h3>
                <div className="space-y-4">
                  <div className="p-4 bg-green-900/20 border border-green-700 rounded-lg">
                    <div className="flex items-center gap-2 mb-2">
                      <CheckCircle className="w-5 h-5 text-green-400" />
                      <span className="font-semibold text-green-400">Key Wins</span>
                    </div>
                    <ul className="text-sm text-green-300 space-y-1">
                      <li>✓ {(sharpeImprovement * 100).toFixed(0)}% improvement in risk-adjusted returns</li>
                      <li>✓ {(drawdownImprovement * 100).toFixed(1)}pp reduction in maximum drawdown</li>
                      <li>✓ ${selectedBacktest.taxSavingsAccumulated.toLocaleString()} in tax savings</li>
                      <li>✓ Net benefit of ${Math.round(selectedBacktest.netBenefit).toLocaleString()}</li>
                    </ul>
                  </div>

                  <div className="p-4 bg-blue-900/20 border border-blue-700 rounded-lg">
                    <div className="flex items-center gap-2 mb-2">
                      <Zap className="w-5 h-5 text-blue-400" />
                      <span className="font-semibold text-blue-400">Insights</span>
                    </div>
                    <ul className="text-sm text-blue-300 space-y-1">
                      <li>• Recommendation outperforms baseline on {Math.round(simulationData.filter(d => d.recommendation > d.baseline).length / simulationData.length * 100)}% of days</li>
                      <li>• Average daily alpha: ${(simulationData[simulationData.length - 1].alpha / simulationData.length).toFixed(0)}</li>
                      <li>• Risk-adjusted return improved by {sharpeImprovement.toFixed(2)} Sharpe points</li>
                    </ul>
                  </div>
                </div>
              </div>
            </div>
          )}

          {tab === 'monte-carlo' && (
            <div className="bg-gray-800/50 backdrop-blur border border-gray-700 rounded-lg p-6">
              <h3 className="text-lg font-semibold mb-4">Monte Carlo Simulation (1000 paths)</h3>
              <p className="text-gray-400 text-sm mb-4">
                Forward-looking simulation showing probability distribution of potential outcomes.
              </p>
              <ResponsiveContainer width="100%" height={400}>
                <ScatterChart margin={{ top: 20, right: 20, bottom: 20, left: 20 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#444" />
                  <XAxis type="number" dataKey="percentile" name="Percentile" stroke="#999" />
                  <YAxis type="number" dataKey="return" name="Return %" stroke="#999" />
                  <Tooltip contentStyle={{ backgroundColor: '#1f2937', border: '1px solid #444' }} />
                  <Scatter
                    name="Simulated Outcomes"
                    data={Array.from({ length: 1000 }, (_, i) => ({
                      percentile: i,
                      return: (selectedBacktest.recommendationReturn * 100) + (Math.random() - 0.5) * 30,
                    }))}
                    fill="#60a5fa"
                    fillOpacity={0.6}
                  />
                </ScatterChart>
              </ResponsiveContainer>
              <div className="mt-6 grid grid-cols-3 gap-4">
                <div className="p-4 bg-gray-700/30 rounded-lg">
                  <p className="text-gray-400 text-sm">5th Percentile</p>
                  <p className="text-xl font-bold text-red-400">{(selectedBacktest.recommendationReturn * 0.5 * 100).toFixed(2)}%</p>
                </div>
                <div className="p-4 bg-gray-700/30 rounded-lg">
                  <p className="text-gray-400 text-sm">50th Percentile</p>
                  <p className="text-xl font-bold text-blue-400">{(selectedBacktest.recommendationReturn * 100).toFixed(2)}%</p>
                </div>
                <div className="p-4 bg-gray-700/30 rounded-lg">
                  <p className="text-gray-400 text-sm">95th Percentile</p>
                  <p className="text-xl font-bold text-green-400">{(selectedBacktest.recommendationReturn * 1.5 * 100).toFixed(2)}%</p>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Metadata */}
        <div className="mt-8 bg-gray-800/50 backdrop-blur border border-gray-700 rounded-lg p-6">
          <h3 className="text-lg font-semibold mb-4">Backtest Details</h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div>
              <p className="text-gray-400 text-sm">Simulation Type</p>
              <p className="font-semibold">{selectedBacktest.simulationType}</p>
            </div>
            <div>
              <p className="text-gray-400 text-sm">Time Period</p>
              <p className="font-semibold">{simulationData.length} days</p>
            </div>
            <div>
              <p className="text-gray-400 text-sm">Data Quality</p>
              <p className="font-semibold">{(selectedBacktest.confidence * 100).toFixed(0)}% Confidence</p>
            </div>
            <div>
              <p className="text-gray-400 text-sm">Analysis Date</p>
              <p className="font-semibold">{new Date(selectedBacktest.createdAt).toLocaleDateString()}</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
