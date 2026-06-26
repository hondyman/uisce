import React, { useState, useEffect } from 'react';
import { Users, TrendingUp, DollarSign, UserCheck, Award, Clock } from 'lucide-react';
import { BarChart, Bar, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';

const WealthClientDashboard: React.FC = () => {
  const [clientMetrics, setClientMetrics] = useState<any>(null);
  const [clientsByRisk, setClientsByRisk] = useState<any[]>([]);
  const [clientGrowth, setClientGrowth] = useState<any[]>([]);
  const [advisorPerformance, setAdvisorPerformance] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchClientData();
  }, []);

  const fetchClientData = async () => {
    try {
      // Client metrics
      const metricsQuery = {
        measures: ['clients.client_count', 'clients.total_aum', 'clients.avg_aum'],
        limit: 1,
      };

      // Clients by risk tolerance
      const riskQuery = {
        measures: ['clients.client_count', 'clients.total_aum'],
        dimensions: ['clients.risk_tolerance'],
      };

      // Client growth over time
      const growthQuery = {
        measures: ['clients.client_count'],
        timeDimensions: [{
          dimension: 'clients.created_at',
          granularity: 'month',
        }],
        limit: 12,
      };

      // Advisor performance
      const advisorQuery = {
        measures: ['advisors.client_count', 'advisors.total_aum'],
        dimensions: ['advisors.advisor_name'],
        order: { 'advisors.total_aum': 'desc' },
        limit: 10,
      };

      const [metricsRes, riskRes, growthRes, advisorRes] = await Promise.all([
        fetch('/api/semantic/query', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': 'default-tenant' },
          body: JSON.stringify(metricsQuery),
        }),
        fetch('/api/semantic/query', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': 'default-tenant' },
          body: JSON.stringify(riskQuery),
        }),
        fetch('/api/semantic/query', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': 'default-tenant' },
          body: JSON.stringify(growthQuery),
        }),
        fetch('/api/semantic/query', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': 'default-tenant' },
          body: JSON.stringify(advisorQuery),
        }),
      ]);

      const metricsData = await metricsRes.json();
      const riskData = await riskRes.json();
      const growthData = await growthRes.json();
      const advisorData = await advisorRes.json();

      setClientMetrics(metricsData.data[0] || {});
      setClientsByRisk(riskData.data || []);
      setClientGrowth(growthData.data || []);
      setAdvisorPerformance(advisorData.data || []);
    } catch (error) {
      console.error('Failed to fetch client data:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen bg-gradient-to-br from-slate-50 via-blue-50 to-indigo-50 dark:from-slate-900 dark:via-slate-800 dark:to-indigo-950">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-blue-50 to-indigo-50 dark:from-slate-900 dark:via-slate-800 dark:to-indigo-950">
      {/* Header */}
      <div className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl border-b border-slate-200 dark:border-slate-700 px-6 py-4">
        <div className="max-w-7xl mx-auto">
          <div className="flex items-center space-x-4">
            <div className="p-3 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-xl">
              <Users className="w-6 h-6 text-white" />
            </div>
            <div>
              <h1 className="text-2xl font-bold bg-gradient-to-r from-blue-600 to-indigo-600 bg-clip-text text-transparent">
                Client Analytics
              </h1>
              <p className="text-sm text-slate-600 dark:text-slate-400">
                Client demographics and advisor performance
              </p>
            </div>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-6 py-8 space-y-6">
        {/* Key Metrics */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <MetricCard
            icon={<Users className="w-5 h-5" />}
            label="Total Clients"
            value={(clientMetrics?.['clients.client_count'] || 0).toLocaleString()}
            color="blue"
          />
          <MetricCard
            icon={<DollarSign className="w-5 h-5" />}
            label="Total AUM"
            value={`$${((clientMetrics?.['clients.total_aum'] || 0) / 1000000).toFixed(1)}M`}
            color="green"
          />
          <MetricCard
            icon={<TrendingUp className="w-5 h-5" />}
            label="Avg AUM per Client"
            value={`$${((clientMetrics?.['clients.avg_aum'] || 0) / 1000).toFixed(0)}K`}
            color="purple"
          />
        </div>

        {/* Charts Row 1 */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Client Growth */}
          <div className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl rounded-2xl border border-slate-200 dark:border-slate-700 p-6">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
              Client Growth
            </h3>
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={clientGrowth}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
                <XAxis dataKey="clients.created_at" stroke="#64748b" fontSize={12} />
                <YAxis stroke="#64748b" fontSize={12} />
                <Tooltip />
                <Line 
                  type="monotone" 
                  dataKey="clients.client_count" 
                  stroke="#3b82f6" 
                  strokeWidth={3}
                  dot={{ fill: '#3b82f6', r: 4 }}
                  name="New Clients"
                />
              </LineChart>
            </ResponsiveContainer>
          </div>

          {/* Clients by Risk Tolerance */}
          <div className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl rounded-2xl border border-slate-200 dark:border-slate-700 p-6">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
              Clients by Risk Tolerance
            </h3>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={clientsByRisk}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
                <XAxis dataKey="clients.risk_tolerance" stroke="#64748b" fontSize={12} />
                <YAxis stroke="#64748b" fontSize={12} />
                <Tooltip />
                <Bar dataKey="clients.client_count" fill="#8b5cf6" radius={[8, 8, 0, 0]} name="Clients" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Advisor Performance */}
        <div className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl rounded-2xl border border-slate-200 dark:border-slate-700 p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
              Top Advisors by AUM
            </h3>
            <Award className="w-5 h-5 text-yellow-500" />
          </div>
          <div className="space-y-3">
            {advisorPerformance.map((advisor, idx) => (
              <div
                key={idx}
                className="flex items-center justify-between p-4 bg-gradient-to-r from-slate-50 to-blue-50 dark:from-slate-800 dark:to-blue-950/20 rounded-xl hover:shadow-md transition-all"
              >
                <div className="flex items-center space-x-4">
                  <div className={`w-12 h-12 rounded-xl flex items-center justify-center text-white font-bold ${
                    idx === 0 ? 'bg-gradient-to-br from-yellow-400 to-yellow-600' :
                    idx === 1 ? 'bg-gradient-to-br from-slate-300 to-slate-500' :
                    idx === 2 ? 'bg-gradient-to-br from-orange-400 to-orange-600' :
                    'bg-gradient-to-br from-blue-500 to-indigo-600'
                  }`}>
                    {idx + 1}
                  </div>
                  <div>
                    <div className="font-semibold text-slate-900 dark:text-white">
                      {advisor['advisors.advisor_name']}
                    </div>
                    <div className="text-sm text-slate-500 dark:text-slate-500">
                      {advisor['advisors.client_count']} clients
                    </div>
                  </div>
                </div>
                <div className="text-right">
                  <div className="text-xl font-bold text-slate-900 dark:text-white">
                    ${(advisor['advisors.total_aum'] / 1000000).toFixed(2)}M
                  </div>
                  <div className="text-sm text-slate-500 dark:text-slate-500">
                    ${(advisor['advisors.total_aum'] / advisor['advisors.client_count'] / 1000).toFixed(0)}K avg
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};

// Metric Card Component
const MetricCard: React.FC<{
  icon: React.ReactNode;
  label: string;
  value: string;
  color: 'blue' | 'green' | 'purple';
}> = ({ icon, label, value, color }) => {
  const colorClasses = {
    blue: 'from-blue-500 to-cyan-600',
    green: 'from-green-500 to-emerald-600',
    purple: 'from-purple-500 to-fuchsia-600',
  };

  return (
    <div className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl rounded-xl p-6 border border-slate-200 dark:border-slate-700 hover:shadow-lg transition-all">
      <div className={`inline-flex p-3 rounded-lg bg-gradient-to-br ${colorClasses[color]} mb-4`}>
        <div className="text-white">{icon}</div>
      </div>
      <div className="text-3xl font-bold text-slate-900 dark:text-white mb-1">{value}</div>
      <div className="text-sm font-medium text-slate-600 dark:text-slate-400">{label}</div>
    </div>
  );
};

export default WealthClientDashboard;
