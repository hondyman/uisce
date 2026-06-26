import React, { useState, useEffect } from 'react';
import { TrendingUp, TrendingDown, DollarSign, AlertCircle, Shield, Zap, Activity } from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts';

interface CryptoHolding {
  holdingId: string;
  assetSymbol: string;
  assetName: string;
  quantity: number;
  currentPriceUsd: number;
  currentValueUsd: number;
  totalCostBasis: number;
  unrealizedGainLoss: number;
  isStaked: boolean;
  stakingYieldApy?: number;
}

interface PortfolioSummary {
  totalCryptoValueUsd: number;
  allocationPct: number;
  unrealizedGainLoss: number;
  totalCostBasis: number;
  change24hPct: number;
  uniqueAssets: number;
}

interface TaxLossOpportunity {
  holdingId: string;
  assetSymbol: string;
  quantity: number;
  unrealizedLoss: number;
  estimatedTaxSavings: number;
  replacementSuggestion: string;
}

const COLORS = ['#6366f1', '#8b5cf6', '#ec4899', '#f59e0b', '#10b981'];

export const CryptoDashboard: React.FC = () => {
  const [holdings, setHoldings] = useState<CryptoHolding[]>([]);
  const [summary, setSummary] = useState<PortfolioSummary | null>(null);
  const [taxOpportunities, setTaxOpportunities] = useState<TaxLossOpportunity[]>([]);
  const [priceHistory, setPriceHistory] = useState<any[]>([]);
  const [selectedAsset, setSelectedAsset] = useState<string>('BTC');

  useEffect(() => {
    fetchData();
    // Set up real-time price updates every 5 seconds
    const interval = setInterval(fetchData, 5000);
    return () => clearInterval(interval);
  }, []);

  const fetchData = async () => {
    try {
      const clientId = 'current-client-id'; // Get from auth context
      
      const [holdingsRes, summaryRes, taxRes, historyRes] = await Promise.all([
        fetch(`/api/crypto/clients/${clientId}/holdings`),
        fetch(`/api/crypto/clients/${clientId}/portfolio`),
        fetch(`/api/crypto/clients/${clientId}/tax-loss-opportunities`),
        fetch(`/api/crypto/prices/${selectedAsset}/history`),
      ]);

      setHoldings(await holdingsRes.json());
      setSummary(await summaryRes.json());
      setTaxOpportunities(await taxRes.json());
      setPriceHistory(await historyRes.json());
    } catch (error) {
      console.error('Failed to fetch crypto data:', error);
    }
  };

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(value);
  };

  const formatPercent = (value: number) => {
    return `${(value * 100).toFixed(2)}%`;
  };

  if (!summary) {
    return <div className="p-6">Loading...</div>;
  }

  const pieData = holdings.map((h) => ({
    name: h.assetSymbol,
    value: h.currentValueUsd,
  }));

  return (
    <div className="max-w-7xl mx-auto p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Crypto Portfolio</h1>
          <p className="text-gray-600 mt-1">Institutional-grade digital asset management</p>
        </div>
        <div className="flex items-center gap-2 px-4 py-2 bg-green-50 border border-green-200 rounded-lg">
          <Shield className="w-5 h-5 text-green-600" />
          <span className="text-sm font-medium text-green-700">Qualified Custody</span>
        </div>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
        <div className="bg-gradient-to-br from-indigo-500 to-purple-600 text-white rounded-xl p-6">
          <div className="flex items-center justify-between mb-2">
            <DollarSign className="w-8 h-8 opacity-80" />
            <div className={`flex items-center gap-1 text-sm ${
              summary.change24hPct >= 0 ? 'text-green-200' : 'text-red-200'
            }`}>
              {summary.change24hPct >= 0 ? <TrendingUp className="w-4 h-4" /> : <TrendingDown className="w-4 h-4" />}
              {formatPercent(summary.change24hPct)} 24h
            </div>
          </div>
          <p className="text-3xl font-bold">{formatCurrency(summary.totalCryptoValueUsd)}</p>
          <p className="text-indigo-100 text-sm mt-1">Total Crypto Value</p>
        </div>

        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <div className="flex items-center gap-3 mb-2">
            <div className="p-2 bg-blue-100 rounded-lg">
              <Activity className="w-6 h-6 text-blue-600" />
            </div>
          </div>
          <p className="text-3xl font-bold text-gray-900">{formatPercent(summary.allocationPct / 100)}</p>
          <p className="text-sm text-gray-600 mt-1">Portfolio Allocation</p>
        </div>

        <div className={`bg-white rounded-xl border-2 p-6 ${
          summary.unrealizedGainLoss >= 0 ? 'border-green-300' : 'border-red-300'
        }`}>
          <div className="flex items-center gap-3 mb-2">
            <div className={`p-2 rounded-lg ${
              summary.unrealizedGainLoss >= 0 ? 'bg-green-100' : 'bg-red-100'
            }`}>
              {summary.unrealizedGainLoss >= 0 ? (
                <TrendingUp className="w-6 h-6 text-green-600" />
              ) : (
                <TrendingDown className="w-6 h-6 text-red-600" />
              )}
            </div>
          </div>
          <p className={`text-3xl font-bold ${
            summary.unrealizedGainLoss >= 0 ? 'text-green-600' : 'text-red-600'
          }`}>
            {formatCurrency(Math.abs(summary.unrealizedGainLoss))}
          </p>
          <p className="text-sm text-gray-600 mt-1">
            {summary.unrealizedGainLoss >= 0 ? 'Unrealized Gains' : 'Unrealized Losses'}
          </p>
        </div>

        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <div className="flex items-center gap-3 mb-2">
            <div className="p-2 bg-purple-100 rounded-lg">
              <Zap className="w-6 h-6 text-purple-600" />
            </div>
          </div>
          <p className="text-3xl font-bold text-gray-900">{summary.uniqueAssets}</p>
          <p className="text-sm text-gray-600 mt-1">Unique Assets</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
        {/* Price Chart */}
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-900">Price History</h3>
            <div className="flex gap-2">
              {['BTC', 'ETH', 'SOL'].map((asset) => (
                <button
                  key={asset}
                  onClick={() => setSelectedAsset(asset)}
                  className={`px-3 py-1 rounded-lg text-sm font-medium transition-colors ${
                    selectedAsset === asset
                      ? 'bg-indigo-600 text-white'
                      : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                  }`}
                >
                  {asset}
                </button>
              ))}
            </div>
          </div>
          <ResponsiveContainer width="100%" height={250}>
            <LineChart data={priceHistory}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="timestamp" />
              <YAxis />
              <Tooltip />
              <Line type="monotone" dataKey="priceUsd" stroke="#6366f1" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>

        {/* Asset Allocation Pie Chart */}
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Asset Allocation</h3>
          <ResponsiveContainer width="100%" height={250}>
            <PieChart>
              <Pie
                data={pieData}
                cx="50%"
                cy="50%"
                labelLine={false}
                label={(entry) => `${entry.name} ${((entry.value / summary.totalCryptoValueUsd) * 100).toFixed(1)}%`}
                outerRadius={80}
                fill="#8884d8"
                dataKey="value"
              >
                {pieData.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                ))}
              </Pie>
              <Tooltip formatter={(value: number) => formatCurrency(value)} />
            </PieChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Holdings Table */}
      <div className="bg-white rounded-xl border border-gray-200 mb-8">
        <div className="p-6 border-b border-gray-200">
          <h3 className="text-lg font-semibold text-gray-900">Your Holdings</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Asset</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Quantity</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Price</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Value</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Cost Basis</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Gain/Loss</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {holdings.map((holding) => {
                const gainLossPct = (holding.unrealizedGainLoss / holding.totalCostBasis) * 100;
                const isGain = holding.unrealizedGainLoss >= 0;

                return (
                  <tr key={holding.holdingId} className="hover:bg-gray-50">
                    <td className="px-6 py-4">
                      <div>
                        <p className="font-medium text-gray-900">{holding.assetSymbol}</p>
                        <p className="text-sm text-gray-600">{holding.assetName}</p>
                      </div>
                    </td>
                    <td className="px-6 py-4 text-right">{holding.quantity.toFixed(8)}</td>
                    <td className="px-6 py-4 text-right">{formatCurrency(holding.currentPriceUsd)}</td>
                    <td className="px-6 py-4 text-right font-semibold">{formatCurrency(holding.currentValueUsd)}</td>
                    <td className="px-6 py-4 text-right">{formatCurrency(holding.totalCostBasis)}</td>
                    <td className={`px-6 py-4 text-right font-semibold ${isGain ? 'text-green-600' : 'text-red-600'}`}>
                      {isGain ? '+' : ''}{formatCurrency(holding.unrealizedGainLoss)}
                      <span className="text-sm ml-1">({gainLossPct.toFixed(2)}%)</span>
                    </td>
                    <td className="px-6 py-4">
                      {holding.isStaked && (
                        <span className="px-2 py-1 bg-purple-100 text-purple-700 rounded-full text-xs font-medium">
                          Staked {holding.stakingYieldApy ? `${(holding.stakingYieldApy * 100).toFixed(2)}% APY` : ''}
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

      {/* Tax Loss Harvesting Opportunities */}
      {taxOpportunities.length > 0 && (
        <div className="bg-gradient-to-br from-orange-50 to-red-50 border-2 border-orange-200 rounded-xl p-6">
          <div className="flex items-start gap-4 mb-4">
            <AlertCircle className="w-6 h-6 text-orange-600 flex-shrink-0 mt-1" />
            <div className="flex-1">
              <h3 className="text-lg font-semibold text-gray-900 mb-2">Tax-Loss Harvesting Opportunities</h3>
              <p className="text-sm text-gray-700 mb-4">
                We've identified opportunities to reduce your tax bill by harvesting losses (no wash-sale rules for crypto!)
              </p>
              <div className="space-y-3">
                {taxOpportunities.map((opp) => (
                  <div key={opp.holdingId} className="bg-white rounded-lg p-4 border border-orange-100">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="font-semibold text-gray-900">{opp.assetSymbol}</p>
                        <p className="text-sm text-gray-600">Loss: {formatCurrency(Math.abs(opp.unrealizedLoss))}</p>
                        <p className="text-sm text-green-600 font-medium">
                          Tax Savings: {formatCurrency(opp.estimatedTaxSavings)}
                        </p>
                      </div>
                      <div className="text-right">
                        <p className="text-sm text-gray-600">Suggested replacement:</p>
                        <p className="font-medium text-indigo-600">{opp.replacementSuggestion}</p>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default CryptoDashboard;
