import React, { useEffect, useState } from 'react'
import { Layout, Card, MetricCard, Button, Spinner, ErrorMessage } from '../components/Layout'
import { SLAComplianceTrend, ChainHealthReport, ChainPrediction } from '../types'
import { apiClient } from '../api/client'
import { useWebSocket } from '../hooks/useWebSocket'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts'

export function DashboardHome() {
  const tenantId = localStorage.getItem('tenant_id') || 'default'
  const [slaData, setSlaData] = useState<SLAComplianceTrend[]>([])
  const [healthReports, setHealthReports] = useState<ChainHealthReport[]>([])
  const [predictions, setPredictions] = useState<ChainPrediction[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const { isConnected } = useWebSocket({
    tenantId,
    regions: ['us-east-1', 'eu-west-1', 'apac-1'],
    onEvent: (event) => {
      console.log('Real-time event:', event)
      // Update UI based on event type
      if (event.type === 'daily_sla_refreshed') {
        refetchData()
      }
    }
  })

  const refetchData = async () => {
    try {
      setLoading(true)
      setError(null)

      const [slaRes, healthRes, predictRes] = await Promise.all([
        apiClient.getSLAComplianceTrends(tenantId, 30),
        apiClient.getChainHealth('chain-1'), // Example chain
        apiClient.getChainPredictions(tenantId)
      ])

      setSlaData(slaRes)
      setHealthReports(healthRes ? [healthRes] : [])
      setPredictions(predictRes)
    } catch (err) {
      console.error('Failed to fetch dashboard data:', err)
      setError(err instanceof Error ? err.message : 'Failed to load data')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    refetchData()
  }, [tenantId])

  if (loading) return <Spinner />
  if (error) return <ErrorMessage message={error} />

  const avgSLA = slaData.length > 0 
    ? (slaData.reduce((sum, d) => sum + d.compliance_score, 0) / slaData.length).toFixed(1)
    : '0'

  const highRiskChains = predictions.filter(p => p.failure_prob >= 0.7).length
  const healthyChains = healthReports.filter(h => h.is_healthy).length

  const chartData = slaData.slice(-14).map(d => ({
    date: new Date(d.reported_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
    compliance: d.compliance_score,
    p99: d.percentile_99
  }))

  const healthDistribution = [
    { name: 'Healthy', value: healthyChains, color: '#10b981' },
    { name: 'At Risk', value: highRiskChains, color: '#f59e0b' },
    { name: 'Failed', value: healthReports.length - healthyChains - highRiskChains, color: '#ef4444' }
  ]

  return (
    <Layout
      sidebar={<Sidebar tenantId={tenantId} />}
      header={<Header isConnected={isConnected} />}
    >
      <div className="space-y-8">
        {/* KPI Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <MetricCard
            label="Average SLA Compliance"
            value={`${avgSLA}%`}
            change={Number(avgSLA) > 95 ? 5 : -3}
            trend={Number(avgSLA) > 95 ? 'up' : 'down'}
            color={Number(avgSLA) > 95 ? 'success' : 'warning'}
          />
          <MetricCard
            label="Healthy Chains"
            value={healthyChains}
            change={10}
            trend="up"
            color="success"
          />
          <MetricCard
            label="High Risk Chains"
            value={highRiskChains}
            change={-5}
            trend="down"
            color="danger"
          />
          <MetricCard
            label="Total Predictions"
            value={predictions.length}
            change={25}
            trend="up"
            color="info"
          />
        </div>

        {/* Charts */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* SLA Trend */}
          <Card title="SLA Compliance Trend (14 days)" className="lg:col-span-2">
            {chartData.length > 0 ? (
              <ResponsiveContainer width="100%" height={300}>
                <LineChart data={chartData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="date" />
                  <YAxis />
                  <Tooltip />
                  <Legend />
                  <Line type="monotone" dataKey="compliance" stroke="#3b82f6" name="Compliance (%)" />
                  <Line type="monotone" dataKey="p99" stroke="#f59e0b" name="P99 Latency (ms)" strokeWidth={2} />
                </LineChart>
              </ResponsiveContainer>
            ) : (
              <p className="text-slate-500">No data available</p>
            )}
          </Card>

          {/* Health Distribution */}
          <Card title="Chain Health Distribution">
            {healthDistribution.length > 0 ? (
              <ResponsiveContainer width="100%" height={300}>
                <PieChart>
                  <Pie
                    data={healthDistribution}
                    cx="50%"
                    cy="50%"
                    labelLine={false}
                    label={({ name, value }) => `${name}: ${value}`}
                    outerRadius={80}
                    fill="#8884d8"
                    dataKey="value"
                  >
                    {healthDistribution.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip />
                </PieChart>
              </ResponsiveContainer>
            ) : (
              <p className="text-slate-500">No data available</p>
            )}
          </Card>
        </div>

        {/* Recent Events */}
        <Card title="High-Risk Chains" className="space-y-4">
          {predictions.filter(p => p.failure_prob >= 0.7).length > 0 ? (
            <div className="space-y-3">
              {predictions.slice(0, 5).map(pred => (
                <div key={pred.id} className="flex items-center justify-between p-4 bg-red-50 rounded-lg border border-red-200">
                  <div>
                    <p className="font-medium text-slate-900">{pred.chain_id}</p>
                    <p className="text-sm text-slate-600">{pred.region}</p>
                  </div>
                  <div className="text-right">
                    <p className="text-red-600 font-bold">{(pred.failure_prob * 100).toFixed(1)}% risk</p>
                    <p className="text-xs text-slate-600">{pred.recommended_action}</p>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-slate-500">All chains are operating normally</p>
          )}
        </Card>
      </div>
    </Layout>
  )
}

function Sidebar({ tenantId }: { tenantId: string }) {
  return (
    <nav className="p-6 space-y-4">
      <a href="/" className="block px-4 py-2 rounded-lg bg-blue-600 text-white">
        Dashboard
      </a>
      <a href="/chains" className="block px-4 py-2 rounded-lg hover:bg-brand-light">
        Chains
      </a>
      <a href="/feed" className="block px-4 py-2 rounded-lg hover:bg-brand-light">
        Live Feed
      </a>
      <a href="/reports" className="block px-4 py-2 rounded-lg hover:bg-brand-light">
        Reports
      </a>
      <hr className="border-brand-light my-4" />
      <div className="px-4 py-2 text-xs text-gray-400">
        <p>Tenant: {tenantId}</p>
      </div>
    </nav>
  )
}

function Header({ isConnected }: { isConnected: boolean }) {
  return (
    <div className="px-8 py-4 flex items-center justify-between">
      <h1 className="text-2xl font-bold text-slate-900">SemLayer Analytics</h1>
      <div className="flex items-center gap-4">
        <div className={`flex items-center gap-2 px-3 py-1 rounded-lg ${isConnected ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-100 text-slate-600'}`}>
          <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-emerald-600' : 'bg-slate-400'}`}></div>
          <span className="text-sm font-medium">{isConnected ? 'Connected' : 'Disconnected'}</span>
        </div>
      </div>
    </div>
  )
}
