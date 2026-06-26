import React, { useState, useEffect } from 'react';
import { 
  PieChart, TrendingUp, DollarSign, Calendar, AlertCircle, 
  ArrowUpRight, ArrowDownRight, FileText
} from 'lucide-react';
import { fetchAPI } from '../../../api';

// Types
interface AlternativeInvestment {
  investment_id: string;
  fund_name: string;
  investment_type: string;
  vintage_year: number;
  total_commitment_amount: number;
  unfunded_commitment: number;
  total_capital_called: number;
  total_distributions: number;
  current_nav: number;
  irr_since_inception?: number;
  tvpi?: number;
  dpi?: number;
}

interface AltInvestDashboardProps {
  clientId: string;
}

export const AltInvestDashboard: React.FC<AltInvestDashboardProps> = ({ clientId }) => {
  const [investments, setInvestments] = useState<AlternativeInvestment[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadInvestments();
  }, [clientId]);

  const loadInvestments = async () => {
    setLoading(true);
    try {
      const data = await fetchAPI<AlternativeInvestment[]>(`/altinvest/client/${clientId}`);
      setInvestments(data);
    } catch (error) {
      console.error('Failed to load investments:', error);
    } finally {
      setLoading(false);
    }
  };

  const totalCommitment = investments.reduce((sum, i) => sum + i.total_commitment_amount, 0);
  const totalNAV = investments.reduce((sum, i) => sum + i.current_nav, 0);
  const totalUnfunded = investments.reduce((sum, i) => sum + i.unfunded_commitment, 0);

  if (loading) return <div className="p-8 text-center">Loading alternative investments...</div>;

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <div className="flex justify-between items-center mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Alternative Investments</h1>
          <p className="text-gray-500 mt-1">Private Equity, Hedge Funds, and Real Estate Portfolio</p>
        </div>
        <button className="px-4 py-2 bg-blue-600 text-white rounded-lg flex items-center gap-2 hover:bg-blue-700">
          <FileText size={16} /> Upload K-1 / Statement
        </button>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
        <SummaryCard 
          title="Total Commitment" 
          value={`$${(totalCommitment / 1000000).toFixed(1)}M`} 
          icon={<DollarSign className="text-blue-600" />}
        />
        <SummaryCard 
          title="Current NAV" 
          value={`$${(totalNAV / 1000000).toFixed(1)}M`} 
          icon={<PieChart className="text-purple-600" />}
        />
        <SummaryCard 
          title="Unfunded Commitment" 
          value={`$${(totalUnfunded / 1000000).toFixed(1)}M`} 
          icon={<AlertCircle className="text-orange-600" />}
          subtext="Potential Capital Calls"
        />
        <SummaryCard 
          title="Portfolio IRR" 
          value="14.2%" 
          icon={<TrendingUp className="text-green-600" />}
          subtext="Since Inception"
        />
      </div>

      {/* Investment List */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
        <div className="p-6 border-b border-gray-200">
          <h2 className="font-semibold text-lg">Holdings</h2>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Fund Name</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Type</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Vintage</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Commitment</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Paid-In</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">NAV</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">IRR</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">TVPI</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {investments.map((inv) => (
                <tr key={inv.investment_id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap font-medium text-gray-900">{inv.fund_name}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    <span className="px-2 py-1 bg-gray-100 rounded text-xs font-medium">
                      {inv.investment_type.replace('_', ' ')}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{inv.vintage_year}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-right text-gray-900">
                    ${(inv.total_commitment_amount / 1000).toFixed(0)}k
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-right text-gray-500">
                    ${(inv.total_capital_called / 1000).toFixed(0)}k
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-right font-medium text-gray-900">
                    ${(inv.current_nav / 1000).toFixed(0)}k
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-right text-green-600 font-medium">
                    {inv.irr_since_inception ? `${inv.irr_since_inception}%` : '-'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-right text-gray-500">
                    {inv.tvpi ? `${inv.tvpi}x` : '-'}
                  </td>
                </tr>
              ))}
              {investments.length === 0 && (
                <tr>
                  <td colSpan={8} className="px-6 py-12 text-center text-gray-500">
                    No alternative investments found.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

const SummaryCard: React.FC<{ title: string; value: string; icon: React.ReactNode; subtext?: string }> = ({ 
  title, value, icon, subtext 
}) => (
  <div className="bg-white p-6 rounded-xl shadow-sm border border-gray-200">
    <div className="flex justify-between items-start mb-2">
      <div className="p-2 bg-gray-50 rounded-lg">{icon}</div>
    </div>
    <h3 className="text-gray-500 text-sm font-medium">{title}</h3>
    <div className="text-2xl font-bold text-gray-900 mt-1">{value}</div>
    {subtext && <div className="text-xs text-gray-400 mt-1">{subtext}</div>}
  </div>
);

export default AltInvestDashboard;
