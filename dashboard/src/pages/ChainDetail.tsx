import React, { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Layout, Card, MetricCard, Button, Spinner, ErrorMessage, Badge } from '../components/Layout'
import { ChainHealthReport, ChainPrediction, Incident } from '../types'
import { apiClient } from '../api/client'
import { LineChart, Line, AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, ScatterChart, Scatter } from 'recharts'

export function ChainDetail() {
  const navigate = useNavigate()
  const { chainId = 'chain-1' } = useParams()
  const tenantId = localStorage.getItem('tenant_id') || 'default'

  const [health, setHealth] = useState<ChainHealthReport | null>(null)
  const [prediction, setPrediction] = useState<ChainPrediction | null>(null)
  const [incidents, setIncidents] = useState<Incident[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchData = async () => {
    try {
      setLoading(true)
      setError(null)

      const [healthRes, predRes] = await Promise.all([
        apiClient.getChainHealth(chainId),
        apiClient.getChainPredictions(tenantId),
      ])

      setHealth(healthRes)
      const matchingPred = predRes.find(p => p.chain_id === chainId)
      setPrediction(matchingPred || null)

      // Mock incidents data for demo
      setIncidents([
        {
          id: 'inc-1',
          chain_id: chainId,
          region: 'us-east-1',
          incident_type: 'CONFLICT',
          severity_score: 0.85,
          is_resolved: false,
          detected_at: new Date(Date.now() - 3600000).toISOString(),
          resolved_at: null,
          resolution_steps: ['Check data replication', 'Verify consensus'],
          root_cause: 'Network latency detected'
        }
      ])
    } catch (err) {
      console.error('Failed to fetch chain data:', err)
      setError(err instanceof Error ? err.message : 'Failed to load chain data')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [chainId])

  if (loading) return <Spinner />
  if (error) return <ErrorMessage message={error} />
  if (!health) return <ErrorMessage message="Chain not found" />

  const healthScore = health.health_score * 100
  const failureProb = prediction?.failure_prob || 0
  const timelineData = generateTimelineData(new Date())

  return (
    <Layout sidebar={<ChainSidebar />} header={<ChainHeader chainId={chainId} onBack={() => navigate('/chains')} />}>
      <div className="space-y-8">
        {/* Top Actions */}
        <div className="flex gap-3">
          <Button variant="primary" onClick={() => navigate(`/chains/${chainId}/edit`)}>
            Configure Chain
          </Button>
          <Button variant="secondary" onClick={() => apiClient.executeChainAction(tenantId, chainId, 'restart')}>
            Restart Chain
          </Button>
          {prediction && prediction.failure_prob > 0.7 && (
            <Button variant="danger" onClick={() => apiClient.executeChainAction(tenantId, chainId, 'failover')}>
              Trigger Failover
            </Button>
          )}
        </div>

        {/* KPI Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <MetricCard
            label="Health Score"
            value={`${healthScore.toFixed(1)}%`}
            change={health.health_score > 0.9 ? 5 : -10}
            trend={health.health_score > 0.9 ? 'up' : 'down'}
            color={health.health_score > 0.9 ? 'success' : 'danger'}
          />
          <MetricCard
            label="Failure Probability"
            value={`${(failureProb * 100).toFixed(1)}%`}
            change={failureProb > 0.7 ? 15 : -5}
            trend={failureProb > 0.7 ? 'up' : 'down'}
            color={failureProb > 0.7 ? 'danger' : 'success'}
          />
          <MetricCard
            label="Active Incidents"
            value={incidents.filter(i => !i.is_resolved).length}
            change={0}
            trend="neutral"
            color="warning"
          />
          <MetricCard
            label="Resolved (24h)"
            value={(health.resolved_conflicts || 0).toString()}
            change={8}
            trend="up"
            color="success"
          />
        </div>

        {/* Health Timeline */}
        <Card title="Health Score Timeline (48 hours)">
          {timelineData.length > 0 ? (
            <ResponsiveContainer width="100%" height={300}>
              <AreaChart data={timelineData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" />
                <YAxis />
                <Tooltip />
                <Area type="monotone" dataKey="score" stroke="#10b981" fill="#d1fae5" />
                <Line type="monotone" dataKey="incidents" stroke="#ef4444" strokeWidth={2} />
              </AreaChart>
            </ResponsiveContainer>
          ) : (
            <p className="text-slate-500">No timeline data</p>
          )}
        </Card>

        {/* Prediction & Recommendation */}
        {prediction && (
          <Card title="AI Prediction & Recommendation">
            <div className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                  <p className="text-sm font-medium text-slate-600 mb-2">Failure Probability</p>
                  <div className="w-full bg-slate-200 rounded-full h-3">
                    <div
                      className={`h-3 rounded-full ${failureProb > 0.7 ? 'bg-red-600' : failureProb > 0.4 ? 'bg-yellow-600' : 'bg-green-600'}`}
                      style={{ width: `${failureProb * 100}%` }}
                    ></div>
                  </div>
                  <p className="mt-2 text-2xl font-bold text-slate-900">{(failureProb * 100).toFixed(1)}%</p>
                </div>
                <div>
                  <p className="text-sm font-medium text-slate-600 mb-2">Confidence</p>
                  <div className="w-full bg-slate-200 rounded-full h-3">
                    <div className="h-3 bg-blue-600 rounded-full" style={{ width: '92%' }}></div>
                  </div>
                  <p className="mt-2 text-2xl font-bold text-slate-900">92%</p>
                </div>
              </div>
              <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                <p className="text-sm font-medium text-blue-900 mb-2">Recommended Action</p>
                <p className="text-slate-700">{prediction.recommended_action}</p>
              </div>
            </div>
          </Card>
        )}

        {/* Recent Incidents */}
        <Card title={`Recent Incidents (${incidents.length})`}>
          {incidents.length > 0 ? (
            <div className="space-y-3">
              {incidents.map(incident => (
                <div key={incident.id} className="p-4 border border-slate-200 rounded-lg hover:bg-slate-50">
                  <div className="flex items-start justify-between mb-2">
                    <div>
                      <p className="font-medium text-slate-900">{incident.incident_type}</p>
                      <p className="text-xs text-slate-500 mt-1">{incident.id}</p>
                    </div>
                    <div className="flex gap-2">
                      <Badge status={incident.is_resolved ? 'resolved' : 'active'} />
                      <span className={`text-xs font-medium ${incident.severity_score > 0.7 ? 'text-red-600' : 'text-yellow-600'}`}>
                        {(incident.severity_score * 100).toFixed(0)}% severity
                      </span>
                    </div>
                  </div>
                  <p className="text-sm text-slate-600 mb-2">{incident.root_cause}</p>
                  <p className="text-xs text-slate-500">
                    Detected: {new Date(incident.detected_at).toLocaleString()}
                  </p>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-slate-500">No incidents reported</p>
          )}
        </Card>
      </div>
    </Layout>
  )
}

function generateTimelineData(baseDate: Date) {
  const data = []
  for (let i = 48; i >= 0; i -= 4) {
    const date = new Date(baseDate.getTime() - i * 3600000)
    data.push({
      time: date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' }),
      score: 85 + Math.random() * 12 - 5,
      incidents: Math.floor(Math.random() * 3)
    })
  }
  return data
}

function ChainSidebar() {
  return (
    <nav className="p-6 space-y-4">
      <a href="/" className="block px-4 py-2 rounded-lg hover:bg-brand-light">
        Dashboard
      </a>
      <a href="/chains" className="block px-4 py-2 rounded-lg bg-blue-600 text-white">
        Chains
      </a>
      <a href="/feed" className="block px-4 py-2 rounded-lg hover:bg-brand-light">
        Live Feed
      </a>
      <a href="/reports" className="block px-4 py-2 rounded-lg hover:bg-brand-light">
        Reports
      </a>
    </nav>
  )
}

function ChainHeader({ chainId, onBack }: { chainId: string; onBack: () => void }) {
  return (
    <div className="px-8 py-4 flex items-center justify-between border-b border-slate-200">
      <div className="flex items-center gap-4">
        <button onClick={onBack} className="text-blue-600 hover:text-blue-700 font-medium">
          ← Back
        </button>
        <h1 className="text-2xl font-bold text-slate-900">{chainId}</h1>
      </div>
    </div>
  )
}
