import React, { useState, useEffect } from 'react';
import { 
  Sparkles, DollarSign, TrendingDown, Repeat, 
  CheckCircle, Clock, X, ChevronRight
} from 'lucide-react';
import { fetchAPI } from '../../../api';

// Types
interface TaxOpportunity {
  opportunity_id: string;
  client_id: string;
  opportunity_type: 'TAX_LOSS_HARVEST' | 'ROTH_CONVERSION' | 'CHARITABLE_DONATION' | 'ASSET_LOCATION';
  detected_date: string;
  estimated_tax_savings: number;
  implementation_complexity: 'LOW' | 'MEDIUM' | 'HIGH';
  time_sensitivity: string;
  status: 'IDENTIFIED' | 'PRESENTED_TO_CLIENT' | 'APPROVED' | 'IMPLEMENTED' | 'DECLINED';
  recommended_actions?: any;
  positions_affected?: any;
}

interface TaxDashboardProps {
  clientId: string;
}

export const TaxDashboard: React.FC<TaxDashboardProps> = ({ clientId }) => {
  const [opportunities, setOpportunities] = useState<TaxOpportunity[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedOpp, setSelectedOpp] = useState<TaxOpportunity | null>(null);

  useEffect(() => {
    loadOpportunities();
  }, [clientId]);

  const loadOpportunities = async () => {
    setLoading(true);
    try {
      const data = await fetchAPI<TaxOpportunity[]>(`/taxplan/opportunities/${clientId}`);
      setOpportunities(data || []);
    } catch (error) {
      console.error('Failed to load tax opportunities:', error);
      // Mock data for demonstration
      setOpportunities([
        {
          opportunity_id: '1',
          client_id: clientId,
          opportunity_type: 'TAX_LOSS_HARVEST',
          detected_date: '2024-11-15',
          estimated_tax_savings: 11100,
          implementation_complexity: 'MEDIUM',
          time_sensitivity: 'BEFORE_YEAR_END',
          status: 'IDENTIFIED',
          positions_affected: [{ ticker: 'AAPL', loss: -15000 }, { ticker: 'MSFT', loss: -15000 }]
        },
        {
          opportunity_id: '2',
          client_id: clientId,
          opportunity_type: 'ROTH_CONVERSION',
          detected_date: '2024-11-10',
          estimated_tax_savings: 6500,
          implementation_complexity: 'LOW',
          time_sensitivity: 'BEFORE_YEAR_END',
          status: 'PRESENTED_TO_CLIENT',
        }
      ]);
    } finally {
      setLoading(false);
    }
  };

  const totalSavings = opportunities.reduce((sum, o) => sum + o.estimated_tax_savings, 0);

  if (loading) return <div className="p-8 text-center">Loading tax opportunities...</div>;

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <div className="flex justify-between items-center mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 flex items-center gap-2">
            <Sparkles className="text-yellow-500" />
            Tax Alpha Dashboard
          </h1>
          <p className="text-gray-500 mt-1">Proactive Tax Optimization Opportunities</p>
        </div>
        <button className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700">
          Run Detection
        </button>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
        <SummaryCard 
          title="Total Potential Savings" 
          value={`$${(totalSavings / 1000).toFixed(1)}k`} 
          icon={<DollarSign className="text-green-600" />}
          highlight
        />
        <SummaryCard 
          title="Active Opportunities" 
          value={opportunities.filter(o => o.status !== 'IMPLEMENTED' && o.status !== 'DECLINED').length.toString()} 
          icon={<Sparkles className="text-yellow-600" />}
        />
        <SummaryCard 
          title="Implemented YTD" 
          value={opportunities.filter(o => o.status === 'IMPLEMENTED').length.toString()} 
          icon={<CheckCircle className="text-blue-600" />}
        />
      </div>

      {/* Opportunities List */}
      <div className="space-y-4">
        {opportunities.map((opp) => (
          <OpportunityCard 
            key={opp.opportunity_id} 
            opportunity={opp} 
            onClick={() => setSelectedOpp(opp)}
          />
        ))}
        {opportunities.length === 0 && (
          <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-12 text-center text-gray-500">
            <Sparkles size={48} className="mx-auto mb-4 text-gray-300" />
            <p>No tax optimization opportunities detected.</p>
            <button className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700">
              Run Detection
            </button>
          </div>
        )}
      </div>

      {/* Detail Modal (Simplified) */}
      {selectedOpp && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50" onClick={() => setSelectedOpp(null)}>
          <div className="bg-white rounded-xl max-w-2xl w-full p-6" onClick={(e) => e.stopPropagation()}>
            <div className="flex justify-between items-start mb-4">
              <h2 className="text-xl font-bold">{formatOpportunityType(selectedOpp.opportunity_type)}</h2>
              <button onClick={() => setSelectedOpp(null)} className="text-gray-400 hover:text-gray-600">
                <X size={24} />
              </button>
            </div>
            <div className="space-y-4">
              <div className="flex justify-between">
                <span className="text-gray-500">Estimated Savings:</span>
                <span className="font-bold text-green-600">${selectedOpp.estimated_tax_savings.toLocaleString()}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-500">Complexity:</span>
                <ComplexityBadge complexity={selectedOpp.implementation_complexity} />
              </div>
              <div className="flex justify-between">
                <span className="text-gray-500">Time Sensitivity:</span>
                <span className="text-orange-600 text-sm font-medium">{selectedOpp.time_sensitivity.replace('_', ' ')}</span>
              </div>
            </div>
            <div className="mt-6 flex gap-3">
              <button className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700">
                Present to Client
              </button>
              <button className="flex-1 px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50">
                Dismiss
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

const SummaryCard: React.FC<{ title: string; value: string; icon: React.ReactNode; highlight?: boolean }> = ({ 
  title, value, icon, highlight 
}) => (
  <div className={`bg-white p-6 rounded-xl shadow-sm border ${highlight ? 'border-green-200 bg-green-50' : 'border-gray-200'}`}>
    <div className="flex justify-between items-start mb-2">
      <div className="p-2 bg-white rounded-lg">{icon}</div>
    </div>
    <h3 className="text-gray-500 text-sm font-medium">{title}</h3>
    <div className={`text-2xl font-bold mt-1 ${highlight ? 'text-green-700' : 'text-gray-900'}`}>{value}</div>
  </div>
);

const OpportunityCard: React.FC<{ opportunity: TaxOpportunity; onClick: () => void }> = ({ opportunity, onClick }) => {
  const getIcon = () => {
    switch (opportunity.opportunity_type) {
      case 'TAX_LOSS_HARVEST': return <TrendingDown className="text-red-600" />;
      case 'ROTH_CONVERSION': return <Repeat className="text-purple-600" />;
      default: return <Sparkles className="text-yellow-600" />;
    }
  };

  return (
    <div 
      onClick={onClick}
      className="bg-white rounded-xl shadow-sm border border-gray-200 p-6 hover:border-blue-300 transition-colors cursor-pointer"
    >
      <div className="flex items-start justify-between">
        <div className="flex gap-4">
          <div className="p-3 bg-gray-50 rounded-lg">
            {getIcon()}
          </div>
          <div>
            <h3 className="font-semibold text-lg text-gray-900">{formatOpportunityType(opportunity.opportunity_type)}</h3>
            <p className="text-gray-500 text-sm mt-1">Detected {new Date(opportunity.detected_date).toLocaleDateString()}</p>
            <div className="flex gap-3 mt-3">
              <StatusBadge status={opportunity.status} />
              <ComplexityBadge complexity={opportunity.implementation_complexity} />
            </div>
          </div>
        </div>
        <div className="text-right">
          <div className="text-xs text-gray-500 uppercase">Est. Tax Savings</div>
          <div className="text-2xl font-bold text-green-600">${(opportunity.estimated_tax_savings / 1000).toFixed(1)}k</div>
          <div className="flex items-center gap-1 text-orange-600 text-xs mt-1">
            <Clock size={12} />
            {opportunity.time_sensitivity.replace('_', ' ')}
          </div>
        </div>
      </div>
    </div>
  );
};

const StatusBadge: React.FC<{ status: string }> = ({ status }) => {
  const colors = {
    IDENTIFIED: 'bg-blue-100 text-blue-700',
    PRESENTED_TO_CLIENT: 'bg-purple-100 text-purple-700',
    APPROVED: 'bg-green-100 text-green-700',
    IMPLEMENTED: 'bg-gray-100 text-gray-700',
    DECLINED: 'bg-red-100 text-red-700',
  };
  return (
    <span className={`px-2 py-1 rounded text-xs font-medium ${colors[status as keyof typeof colors]}`}>
      {status.replace('_', ' ')}
    </span>
  );
};

const ComplexityBadge: React.FC<{ complexity: string }> = ({ complexity }) => {
  const colors = {
    LOW: 'bg-green-100 text-green-700',
    MEDIUM: 'bg-yellow-100 text-yellow-700',
    HIGH: 'bg-red-100 text-red-700',
  };
  return (
    <span className={`px-2 py-1 rounded text-xs font-medium ${colors[complexity as keyof typeof colors]}`}>
      {complexity}
    </span>
  );
};

const formatOpportunityType = (type: string) => {
  return type.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
};

export default TaxDashboard;
