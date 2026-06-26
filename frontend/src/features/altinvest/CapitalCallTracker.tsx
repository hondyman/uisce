import React, { useState, useEffect } from 'react';
import { Calendar, AlertTriangle, CheckCircle2, Clock, DollarSign, TrendingUp } from 'lucide-react';
import { fetchAPI } from '../../api';

// Types
interface UpcomingCapitalCall {
  call_id: string;
  investment_id: string;
  fund_name: string;
  notice_date: string;
  due_date: string;
  amount_requested: number;
  amount_funded: number;
  status: 'PENDING' | 'FUNDED' | 'PARTIALLY_FUNDED' | 'OVERDUE' | 'CANCELLED';
  liquidity_check_passed: boolean | null;
  days_until_due: number;
}

interface CapitalCallTrackerProps {
  clientId?: string;
}

export const CapitalCallTracker: React.FC<CapitalCallTrackerProps> = ({ clientId }) => {
  const [calls, setCalls] = useState<UpcomingCapitalCall[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadCapitalCalls();
  }, [clientId]);

  const loadCapitalCalls = async () => {
    setLoading(true);
    try {
      const url = clientId 
        ? `/alternative-investments/capital-calls/upcoming?client_id=${clientId}`
        : '/alternative-investments/capital-calls/upcoming';
      const data = await fetchAPI<UpcomingCapitalCall[]>(url);
      setCalls(data);
    } catch (error) {
      console.error('Failed to load capital calls:', error);
    } finally {
      setLoading(false);
    }
  };

  const getUrgencyColor = (daysUntilDue: number) => {
    if (daysUntilDue < 0) return 'text-red-600 bg-red-50';
    if (daysUntilDue <= 3) return 'text-orange-600 bg-orange-50';
    if (daysUntilDue <= 7) return 'text-yellow-600 bg-yellow-50';
    return 'text-blue-600 bg-blue-50';
  };

  const getLiquidityIcon = (passed: boolean | null) => {
    if (passed === null) return <Clock size={16} className="text-gray-400" />;
    if (passed) return <CheckCircle2 size={16} className="text-green-500" />;
    return <AlertTriangle size={16} className="text-red-500" />;
  };

  if (loading) return <div className="p-8 text-center">Loading capital calls...</div>;

  return (
    <div className="space-y-6">
      {/* Urgent Calls Alert */}
      {calls.some(c => c.days_until_due <= 3) && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <div className="flex items-center gap-3">
            <AlertTriangle className="text-red-600" size={24} />
            <div>
              <h3 className="font-semibold text-red-900">Urgent Capital Calls</h3>
              <p className="text-sm text-red-700">
                {calls.filter(c => c.days_until_due <= 3).length} capital call(s) due within 3 days
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Capital Calls Timeline */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-200">
        <div className="p-6 border-b border-gray-200">
          <h2 className="font-semibold text-lg flex items-center gap-2">
            <Calendar size={20} />
            Upcoming Capital Calls
          </h2>
        </div>
        
        <div className="divide-y divide-gray-200">
          {calls.length === 0 ? (
            <div className="p-8 text-center text-gray-500">
              No upcoming capital calls
            </div>
          ) : (
            calls.map((call) => (
              <div key={call.call_id} className="p-6 hover:bg-gray-50 transition-colors">
                <div className="flex justify-between items-start">
                  <div className="flex-1">
                    <div className="flex items-center gap-3 mb-2">
                      <h3 className="font-semibold text-gray-900">{call.fund_name}</h3>
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${getUrgencyColor(call.days_until_due)}`}>
                        {call.days_until_due < 0 
                          ? `${-call.days_until_due} days overdue` 
                          : `${call.days_until_due} days until due`}
                      </span>
                      <span className="px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-700">
                        {call.status.replace('_', ' ')}
                      </span>
                    </div>
                    
                    <div className="grid grid-cols-3 gap-4 mt-3">
                      <div>
                        <p className="text-xs text-gray-500">Requested</p>
                        <p className="font-semibold text-gray-900">${(call.amount_requested / 1000).toFixed(0)}k</p>
                      </div>
                      <div>
                        <p className="text-xs text-gray-500">Funded</p>
                        <p className="font-semibold text-gray-900">${(call.amount_funded / 1000).toFixed(0)}k</p>
                      </div>
                      <div>
                        <p className="text-xs text-gray-500">Balance Due</p>
                        <p className="font-semibold text-blue-600">
                          ${((call.amount_requested - call.amount_funded) / 1000).toFixed(0)}k
                        </p>
                      </div>
                    </div>

                    <div className="flex items-center gap-4 mt-3 text-sm">
                      <div className="flex items-center gap-2">
                        <Calendar size={14} className="text-gray-400" />
                        <span className="text-gray-600">Due: {new Date(call.due_date).toLocaleDateString()}</span>
                      </div>
                      <div className="flex items-center gap-2">
                        {getLiquidityIcon(call.liquidity_check_passed)}
                        <span className="text-gray-600">
                          {call.liquidity_check_passed === null 
                            ? 'Liquidity check pending'
                            : call.liquidity_check_passed
                            ? 'Sufficient liquidity'
                            : 'Liquidity shortage'}
                        </span>
                      </div>
                    </div>
                  </div>

                  <button className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700">
                    Fund Call
                  </button>
                </div>

                {/* Funding Progress Bar */}
                <div className="mt-4">
                  <div className="w-full bg-gray-200 rounded-full h-2">
                    <div 
                      className="bg-blue-600 h-2 rounded-full transition-all"
                      style={{ width: `${(call.amount_funded / call.amount_requested) * 100}%` }}
                    />
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Summary Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <StatCard 
          title="Total Due" 
          value={`$${(calls.reduce((sum, c) => sum + (c.amount_requested - c.amount_funded), 0) / 1000000).toFixed(2)}M`}
          icon={<DollarSign className="text-blue-600" />}
        />
        <StatCard 
          title="Calls Pending" 
          value={calls.filter(c => c.status === 'PENDING').length.toString()}
          icon={<Clock className="text-orange-600" />}
        />
        <StatCard 
          title="Liquidity Issues" 
          value={calls.filter(c => c.liquidity_check_passed === false).length.toString()}
          icon={<AlertTriangle className="text-red-600" />}
        />
      </div>
    </div>
  );
};

const StatCard: React.FC<{ title: string; value: string; icon: React.ReactNode }> = ({ title, value, icon }) => (
  <div className="bg-white p-4 rounded-lg border border-gray-200">
    <div className="flex items-center justify-between">
      <div>
        <p className="text-sm text-gray-500">{title}</p>
        <p className="text-2xl font-bold text-gray-900 mt-1">{value}</p>
      </div>
      <div className="p-3 bg-gray-50 rounded-lg">{icon}</div>
    </div>
  </div>
);

export default CapitalCallTracker;
