import React, { useState, useEffect } from 'react';
import { DollarSign, TrendingUp, Users, Calendar, CheckCircle, AlertCircle, Settings } from 'lucide-react';

interface FeeSchedule {
  scheduleId: string;
  scheduleName: string;
  feeType: string;
  aumTiers: Array<{ threshold: number; rate: number }>;
  performanceFeeRate?: number;
  highWaterMark?: number;
  isActive: boolean;
}

interface FeeCalculation {
  calculationId: string;
  clientName: string;
  period: string;
  aumBaseFee: number;
  performanceFee: number;
  totalFee: number;
  status: string;
  calculatedAt: string;
}

export const FeeBillingCenter: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'overview' | 'schedules' | 'calculations'>('overview');
  const [schedules, setSchedules] = useState<FeeSchedule[]>([]);
  const [calculations, setCalculations] = useState<FeeCalculation[]>([]);
  const [stats, setStats] = useState({
    totalRevenue: 0,
    pendingApprovals: 0,
    activeClients: 0,
    avgFee: 0,
  });

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      const [schedulesRes, calculationsRes] = await Promise.all([
        fetch('/api/fee-billing/schedules'),
        fetch('/api/fee-billing/calculations'),
      ]);

      setSchedules(await schedulesRes.json());
      const calcs = await calculationsRes.json();
      setCalculations(calcs);

      // Calculate stats
      const totalRev = calcs.reduce((sum: number, c: FeeCalculation) => sum + c.totalFee, 0);
      const pending = calcs.filter((c: FeeCalculation) => c.status === 'PENDING').length;

      setStats({
        totalRevenue: totalRev,
        pendingApprovals: pending,
        activeClients: calcs.length,
        avgFee: totalRev / (calcs.length || 1),
      });
    } catch (error) {
      console.error('Failed to fetch fee billing data:', error);
    }
  };

  return (
    <div className="max-w-7xl mx-auto p-6">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Fee Billing Center</h1>
          <p className="text-gray-600 mt-1">Manage fee schedules, calculations, and revenue recognition</p>
        </div>
        <button className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors">
          <Settings className="w-5 h-5 inline mr-2" />
          Configure Billing
        </button>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
        <StatCard
          icon={<DollarSign />}
          label="Total Revenue YTD"
          value={`$${(stats.totalRevenue / 1000).toFixed(0)}K`}
          change="+12.5%"
          positive
        />
        <StatCard
          icon={<AlertCircle />}
          label="Pending Approvals"
          value={stats.pendingApprovals.toString()}
          change="Requires action"
          positive={false}
        />
        <StatCard
          icon={<Users />}
          label="Active Clients"
          value={stats.activeClients.toString()}
          change="+8 this month"
          positive
        />
        <StatCard
          icon={<TrendingUp />}
          label="Avg Fee per Client"
          value={`$${(stats.avgFee / 1000).toFixed(1)}K`}
          change="+5.2%"
          positive
        />
      </div>

      {/* Tabs */}
      <div className="flex gap-4 border-b border-gray-200 mb-6">
        <TabButton
          label="Overview"
          active={activeTab === 'overview'}
          onClick={() => setActiveTab('overview')}
        />
        <TabButton
          label="Fee Schedules"
          active={activeTab === 'schedules'}
          onClick={() => setActiveTab('schedules')}
        />
        <TabButton
          label="Calculations"
          active={activeTab === 'calculations'}
          onClick={() => setActiveTab('calculations')}
        />
      </div>

      {/* Tab Content */}
      {activeTab === 'overview' && <OverviewTab calculations={calculations} />}
      {activeTab === 'schedules' && <SchedulesTab schedules={schedules} onUpdate={fetchData} />}
      {activeTab === 'calculations' && <CalculationsTab calculations={calculations} onUpdate={fetchData} />}
    </div>
  );
};

const StatCard: React.FC<{ icon: React.ReactNode; label: string; value: string; change: string; positive: boolean }> = ({
  icon,
  label,
  value,
  change,
  positive,
}) => (
  <div className="bg-white rounded-xl border border-gray-200 p-6">
    <div className="flex items-start justify-between mb-4">
      <div className="p-3 bg-indigo-100 rounded-lg">
        <div className="text-indigo-600">{icon}</div>
      </div>
      <span className={`text-sm font-medium ${positive ? 'text-green-600' : 'text-orange-600'}`}>
        {change}
      </span>
    </div>
    <p className="text-3xl font-bold text-gray-900">{value}</p>
    <p className="text-sm text-gray-600 mt-1">{label}</p>
  </div>
);

const TabButton: React.FC<{ label: string; active: boolean; onClick: () => void }> = ({ label, active, onClick }) => (
  <button
    onClick={onClick}
    className={`px-4 py-3 border-b-2 transition-colors font-medium ${
      active ? 'border-indigo-600 text-indigo-600' : 'border-transparent text-gray-600 hover:text-gray-900'
    }`}
  >
    {label}
  </button>
);

const OverviewTab: React.FC<{ calculations: FeeCalculation[] }> = ({ calculations }) => {
  const recentCalcs = calculations.slice(0, 5);

  return (
    <div className="space-y-6">
      <div className="bg-white rounded-xl border border-gray-200 p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Recent Calculations</h3>
        <div className="space-y-3">
          {recentCalcs.map((calc) => (
            <div key={calc.calculationId} className="flex items-center justify-between p-4 bg-gray-50 rounded-lg">
              <div>
                <p className="font-medium text-gray-900">{calc.clientName}</p>
                <p className="text-sm text-gray-600">{calc.period}</p>
              </div>
              <div className="text-right">
                <p className="text-lg font-bold text-gray-900">${(calc.totalFee / 1000).toFixed(1)}K</p>
                <StatusBadge status={calc.status} />
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

const SchedulesTab: React.FC<{ schedules: FeeSchedule[]; onUpdate: () => void }> = ({ schedules, onUpdate }) => {
  return (
    <div className="space-y-4">
      <div className="flex justify-end mb-4">
        <button className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors">
          Create New Schedule
        </button>
      </div>

      {schedules.map((schedule) => (
        <div key={schedule.scheduleId} className="bg-white rounded-xl border border-gray-200 p-6">
          <div className="flex items-start justify-between mb-4">
            <div>
              <h4 className="text-lg font-semibold text-gray-900">{schedule.scheduleName}</h4>
              <p className="text-sm text-gray-600">{schedule.feeType}</p>
            </div>
            {schedule.isActive && (
              <span className="px-3 py-1 bg-green-100 text-green-700 rounded-full text-sm font-medium">
                Active
              </span>
            )}
          </div>

          <div className="grid grid-cols-3 gap-4">
            {schedule.aumTiers.map((tier, idx) => (
              <div key={idx} className="bg-gray-50 p-3 rounded-lg">
                <p className="text-xs text-gray-600">Tier {idx + 1}</p>
                <p className="font-semibold text-gray-900">
                  ${(tier.threshold / 1000000).toFixed(1)}M+ → {(tier.rate * 100).toFixed(2)}%
                </p>
              </div>
            ))}
          </div>

          {schedule.performanceFeeRate && (
            <div className="mt-4 pt-4 border-t border-gray-200">
              <p className="text-sm text-gray-700">
                <strong>Performance Fee:</strong> {(schedule.performanceFeeRate * 100).toFixed(0)}% above high water mark
              </p>
            </div>
          )}
        </div>
      ))}
    </div>
  );
};

const CalculationsTab: React.FC<{ calculations: FeeCalculation[]; onUpdate: () => void }> = ({ calculations, onUpdate }) => {
  const approveCalculation = async (calcId: string) => {
    try {
      await fetch(`/api/fee-billing/calculations/${calcId}/approve`, { method: 'POST' });
      onUpdate();
    } catch (error) {
      console.error('Failed to approve calculation:', error);
    }
  };

  return (
    <div className="bg-white rounded-xl border border-gray-200">
      <table className="w-full">
        <thead className="bg-gray-50 border-b border-gray-200">
          <tr>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Client</th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Period</th>
            <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">AUM Fee</th>
            <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Performance</th>
            <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Total</th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
            <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-200">
          {calculations.map((calc) => (
            <tr key={calc.calculationId} className="hover:bg-gray-50">
              <td className="px-6 py-4 font-medium text-gray-900">{calc.clientName}</td>
              <td className="px-6 py-4 text-sm text-gray-600">{calc.period}</td>
              <td className="px-6 py-4 text-right text-sm">${(calc.aumBaseFee / 1000).toFixed(1)}K</td>
              <td className="px-6 py-4 text-right text-sm">${(calc.performanceFee / 1000).toFixed(1)}K</td>
              <td className="px-6 py-4 text-right font-semibold">${(calc.totalFee / 1000).toFixed(1)}K</td>
              <td className="px-6 py-4">
                <StatusBadge status={calc.status} />
              </td>
              <td className="px-6 py-4 text-right">
                {calc.status === 'PENDING' && (
                  <button
                    onClick={() => approveCalculation(calc.calculationId)}
                    className="text-indigo-600 hover:text-indigo-700 font-medium text-sm"
                  >
                    Approve
                  </button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

const StatusBadge: React.FC<{ status: string }> = ({ status }) => {
  const config = {
    PENDING: { bg: 'bg-yellow-100', text: 'text-yellow-700', label: 'Pending' },
    APPROVED: { bg: 'bg-green-100', text: 'text-green-700', label: 'Approved' },
    BILLED: { bg: 'bg-blue-100', text: 'text-blue-700', label: 'Billed' },
  }[status] || { bg: 'bg-gray-100', text: 'text-gray-700', label: status };

  return (
    <span className={`px-2 py-1 rounded-full text-xs font-medium ${config.bg} ${config.text}`}>
      {config.label}
    </span>
  );
};
