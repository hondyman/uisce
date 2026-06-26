import React, { useState } from 'react';
import { 
  DollarSign, TrendingUp, Receipt, Clock, 
  CheckCircle, AlertCircle, Download
} from 'lucide-react';

// Types
interface FeeCalculation {
  calculation_id: string;
  client_id: string;
  billing_period_start: string;
  billing_period_end: string;
  total_fee: number;
  calculation_status: 'DRAFT' | 'APPROVED' | 'INVOICED' | 'PAID';
  aum_based_fee: number;
  performance_fee: number;
}

export const BillingCenter: React.FC = () => {
  const [calculations, setCalculations] = useState<FeeCalculation[]>([]);
  const [selectedPeriod, setSelectedPeriod] = useState('2024-Q4');

  // Mock data
  const mockCalculations: FeeCalculation[] = [
    {
      calculation_id: '1',
      client_id: 'client-1',
      billing_period_start: '2024-10-01',
      billing_period_end: '2024-12-31',
      total_fee: 12500,
      calculation_status: 'PAID',
      aum_based_fee: 12000,
      performance_fee: 500,
    },
    {
      calculation_id: '2',
      client_id: 'client-2',
      billing_period_start: '2024-10-01',
      billing_period_end: '2024-12-31',
      total_fee: 8750,
      calculation_status: 'INVOICED',
      aum_based_fee: 8750,
      performance_fee: 0,
    },
  ];

  React.useEffect(() => {
    setCalculations(mockCalculations);
  }, [selectedPeriod]);

  const totalBilled = calculations.reduce((sum, c) => sum + c.total_fee, 0);
  const totalCollected = calculations
    .filter(c => c.calculation_status === 'PAID')
    .reduce((sum, c) => sum + c.total_fee, 0);
  const outstandingAR = totalBilled - totalCollected;

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <div className="flex justify-between items-center mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Billing Center</h1>
          <p className="text-gray-500 mt-1">Revenue Management & Fee Calculations</p>
        </div>
        <div className="flex gap-3">
          <select 
            value={selectedPeriod}
            onChange={(e) => setSelectedPeriod(e.target.value)}
            className="px-4 py-2 border border-gray-300 rounded-lg bg-white"
          >
            <option value="2024-Q4">2024 Q4</option>
            <option value="2024-Q3">2024 Q3</option>
            <option value="2024-Q2">2024 Q2</option>
          </select>
          <button className="px-4 py-2 bg-blue-600 text-white rounded-lg flex items-center gap-2 hover:bg-blue-700">
            <Download size={16} /> Export Report
          </button>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
        <MetricCard 
          title="Total Billed" 
          value={`$${(totalBilled / 1000).toFixed(1)}k`} 
          icon={<Receipt className="text-blue-600" />}
          trend="+12.3%"
        />
        <MetricCard 
          title="Total Collected" 
          value={`$${(totalCollected / 1000).toFixed(1)}k`} 
          icon={<CheckCircle className="text-green-600" />}
        />
        <MetricCard 
          title="Outstanding A/R" 
          value={`$${(outstandingAR / 1000).toFixed(1)}k`} 
          icon={<AlertCircle className="text-orange-600" />}
        />
        <MetricCard 
          title="Avg Collection Time" 
          value="8 days" 
          icon={<Clock className="text-purple-600" />}
        />
      </div>

      {/* Fee Breakdown */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
          <h2 className="font-semibold text-lg mb-4">Revenue by Fee Type</h2>
          <div className="space-y-4">
            <RevenueBar label="AUM Fees" amount={20750} total={21250} color="bg-blue-500" />
            <RevenueBar label="Performance Fees" amount={500} total={21250} color="bg-green-500" />
          </div>
        </div>

        <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
          <h2 className="font-semibold text-lg mb-4">Collection Status</h2>
          <div className="space-y-3">
            <StatusCount label="Paid" count={1} color="text-green-600" />
            <StatusCount label="Invoiced" count={1} color="text-blue-600" />
            <StatusCount label="Draft" count={0} color="text-gray-600" />
            <StatusCount label="Overdue" count={0} color="text-red-600" />
          </div>
        </div>
      </div>

      {/* Fee Calculations Table */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
        <div className="p-6 border-b border-gray-200">
          <h2 className="font-semibold text-lg">Fee Calculations</h2>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Client</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Period</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">AUM Fee</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Performance Fee</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Total</th>
                <th className="px-6 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {calculations.map((calc) => (
                <tr key={calc.calculation_id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap font-medium text-gray-900">Client #{calc.client_id.slice(-4)}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {new Date(calc.billing_period_start).toLocaleDateString()} - {new Date(calc.billing_period_end).toLocaleDateString()}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-right text-gray-900">
                    ${calc.aum_based_fee.toLocaleString()}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-right text-gray-500">
                    ${calc.performance_fee.toLocaleString()}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-right font-medium text-gray-900">
                    ${calc.total_fee.toLocaleString()}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-center">
                    <StatusBadge status={calc.calculation_status} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

const MetricCard: React.FC<{ title: string; value: string; icon: React.ReactNode; trend?: string }> = ({ 
  title, value, icon, trend 
}) => (
  <div className="bg-white p-6 rounded-xl shadow-sm border border-gray-200">
    <div className="flex justify-between items-start mb-2">
      <div className="p-2 bg-gray-50 rounded-lg">{icon}</div>
      {trend && <span className="text-xs text-green-600 font-medium">{trend}</span>}
    </div>
    <h3 className="text-gray-500 text-sm font-medium">{title}</h3>
    <div className="text-2xl font-bold text-gray-900 mt-1">{value}</div>
  </div>
);

const RevenueBar: React.FC<{ label: string; amount: number; total: number; color: string }> = ({ 
  label, amount, total, color 
}) => {
  const percentage = (amount / total) * 100;
  return (
    <div>
      <div className="flex justify-between text-sm mb-1">
        <span className="text-gray-700">{label}</span>
        <span className="font-medium text-gray-900">${amount.toLocaleString()}</span>
      </div>
      <div className="w-full bg-gray-200 h-2 rounded-full overflow-hidden">
        <div className={`${color} h-full`} style={{ width: `${percentage}%` }}></div>
      </div>
    </div>
  );
};

const StatusCount: React.FC<{ label: string; count: number; color: string }> = ({ label, count, color }) => (
  <div className="flex justify-between items-center">
    <span className="text-gray-700 text-sm">{label}</span>
    <span className={`${color} font-semibold`}>{count}</span>
  </div>
);

const StatusBadge: React.FC<{ status: string }> = ({ status }) => {
  const colors = {
    DRAFT: 'bg-gray-100 text-gray-700',
    APPROVED: 'bg-blue-100 text-blue-700',
    INVOICED: 'bg-purple-100 text-purple-700',
    PAID: 'bg-green-100 text-green-700',
  };
  return (
    <span className={`px-2 py-1 rounded text-xs font-medium ${colors[status as keyof typeof colors]}`}>
      {status}
    </span>
  );
};

export default BillingCenter;
