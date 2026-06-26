import React, { useEffect, useState } from 'react'
import { Layout, Card, Button, Spinner, Badge } from '../components/Layout'
import { apiClient } from '../api/client'
import { Prediction, Explainability } from '../types'
import { BarChart, Bar, LineChart, Line, ScatterChart, Scatter, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'

export function PredictionsPage() {
  const tenantId = localStorage.getItem('tenant_id') || 'default'
  const [predictions, setPredictions] = useState<Prediction[]>([])
  const [selectedChainId, setSelectedChainId] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filterRisk, setFilterRisk] = useState<'all' | 'critical' | 'high' | 'medium' | 'low'>('all')

  const selectedPrediction = predictions.find(p => p.chain_id === selectedChainId)

  const fetchPredictions = async () => {
    try {
      setLoading(true)
      setError(null)

      // Mock predictions data
      const mockPredictions: Prediction[] = [
        {
          id: 'pred-1',
          chain_id: 'chain-1',
          region: 'us-east-1',
          tenant_id: tenantId,
          failure_probability: 0.15,
          confidence: 0.88,
          risk_level: 'low',
          predicted_at: new Date().toISOString(),
          horizon_hours: 24,
          top_risk_factors: [
            { name: 'Health Score', contribution: 0.3, current_value: 0.94, threshold: 0.85, direction: 'stable' }
          ],
          model_version: '2.1.0'
        },
        {
          id: 'pred-2',
          chain_id: 'chain-2',
          region: 'eu-west-1',
          tenant_id: tenantId,
          failure_probability: 0.72,
          confidence: 0.91,
          risk_level: 'critical',
          predicted_at: new Date(Date.now() - 3600000).toISOString(),
          horizon_hours: 6,
          top_risk_factors: [
            { name: 'Active Conflicts', contribution: 0.4, current_value: 12, threshold: 5, direction: 'increasing' },
            { name: 'P99 Latency', contribution: 0.28, current_value: 950, threshold: 500, direction: 'increasing' }
          ],
          model_version: '2.1.0'
        },
        {
          id: 'pred-3',
          chain_id: 'chain-3',
          region: 'apac-1',
          tenant_id: tenantId,
          failure_probability: 0.45,
          confidence: 0.85,
          risk_level: 'high',
          predicted_at: new Date(Date.now() - 7200000).toISOString(),
          horizon_hours: 1,
          top_risk_factors: [
            { name: 'Error Rate', contribution: 0.35, current_value: 0.08, threshold: 0.05, direction: 'increasing' }
          ],
          model_version: '2.1.0'
        }
      ]

      setPredictions(mockPredictions)
      if (mockPredictions.length > 0) {
        setSelectedChainId(mockPredictions[0].chain_id)
      }
    } catch (err) {
      console.error('Failed to fetch predictions:', err)
      setError(err instanceof Error ? err.message : 'Failed to load predictions')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchPredictions()
  }, [])

  const filteredPredictions = predictions.filter(p => {
    if (filterRisk === 'all') return true
    return p.risk_level === filterRisk
  })

  if (loading) return <Spinner />

  return (
    <Layout sidebar={<PredictionsSidebar />} header={<PredictionsHeader />}>
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Predictions List */}
        <div className="lg:col-span-1 space-y-4">
          <Card className="p-4">
            <h3 className="font-bold mb-4">Filter by Risk</h3>
            <div className="space-y-2">
              {(['all', 'critical', 'high', 'medium', 'low'] as const).map(level => (
                <button
                  key={level}
                  onClick={() => setFilterRisk(level)}
                  className={`w-full px-3 py-2 rounded text-left text-sm transition-colors ${
                    filterRisk === level
                      ? 'bg-blue-600 text-white'
                      : 'hover:bg-slate-100'
                  }`}
                >
                  {level.charAt(0).toUpperCase() + level.slice(1)} ({predictions.filter(p => p.risk_level === level).length})
                </button>
              ))}
            </div>
          </Card>

          <div className="space-y-3">
            {filteredPredictions.map(pred => (
              <PredictionCard
                key={pred.id}
                prediction={pred}
                isSelected={selectedChainId === pred.chain_id}
                onSelect={() => setSelectedChainId(pred.chain_id)}
              />
            ))}
          </div>
        </div>

        {/* Details & Explainability */}
        {selectedPrediction ? (
          <div className="lg:col-span-2 space-y-6">
            <PredictionDetails prediction={selectedPrediction} />
            {selectedPrediction.explainability && (
              <ExplainabilityView explainability={selectedPrediction.explainability} />
            )}
          </div>
        ) : (
          <Card className="lg:col-span-2 text-center py-12">
            <p className="text-slate-500">Select a prediction to view details</p>
          </Card>
        )}
      </div>
    </Layout>
  )
}

interface PredictionCardProps {
  prediction: Prediction
  isSelected: boolean
  onSelect: () => void
}

function PredictionCard({ prediction, isSelected, onSelect }: PredictionCardProps) {
  const colors = {
    critical: { bg: 'bg-red-50', border: 'border-red-300', badge: 'danger' },
    high: { bg: 'bg-amber-50', border: 'border-amber-300', badge: 'warning' },
    medium: { bg: 'bg-yellow-50', border: 'border-yellow-300', badge: 'warning' },
    low: { bg: 'bg-emerald-50', border: 'border-emerald-300', badge: 'success' }
  }

  const style = colors[prediction.risk_level as keyof typeof colors] || colors.low
  const failurePercent = (prediction.failure_probability * 100).toFixed(0)

  return (
    <Card
      className={`p-4 cursor-pointer transition-all border-2 ${style.border} ${isSelected ? 'ring-2 ring-blue-500' : ''} ${style.bg}`}
      onClick={onSelect}
    >
      <div className="space-y-3">
        <div className="flex items-start justify-between">
          <div>
            <p className="font-bold text-slate-900">{prediction.chain_id}</p>
            <p className="text-xs text-slate-600">{prediction.region}</p>
          </div>
          <Badge status={prediction.risk_level as any} />
        </div>

        <div className="bg-white bg-opacity-60 rounded p-2">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium">Failure Risk</span>
            <span className={`text-lg font-bold ${prediction.failure_probability > 0.7 ? 'text-red-600' : 'text-slate-700'}`}>
              {failurePercent}%
            </span>
          </div>
          <div className="w-full bg-slate-200 rounded-full h-2">
            <div
              className={`h-2 rounded-full ${prediction.failure_probability > 0.7 ? 'bg-red-600' : prediction.failure_probability > 0.4 ? 'bg-amber-600' : 'bg-emerald-600'}`}
              style={{ width: `${prediction.failure_probability * 100}%` }}
            ></div>
          </div>
        </div>

        <div className="text-xs text-slate-600">
          <p>Confidence: {(prediction.confidence * 100).toFixed(0)}%</p>
          <p>Horizon: {prediction.horizon_hours}h</p>
        </div>
      </div>
    </Card>
  )
}

interface PredictionDetailsProps {
  prediction: Prediction
}

function PredictionDetails({ prediction }: PredictionDetailsProps) {
  return (
    <Card title="Prediction Details" className="space-y-4">
      <div className="grid grid-cols-2 gap-4">
        <div>
          <p className="text-xs text-slate-600 mb-1">Chain ID</p>
          <p className="font-mono font-bold">{prediction.chain_id}</p>
        </div>
        <div>
          <p className="text-xs text-slate-600 mb-1">Region</p>
          <p className="font-mono font-bold">{prediction.region}</p>
        </div>
        <div>
          <p className="text-xs text-slate-600 mb-1">Failure Probability</p>
          <p className={`font-bold ${prediction.failure_probability > 0.7 ? 'text-red-600' : 'text-slate-700'}`}>
            {(prediction.failure_probability * 100).toFixed(1)}%
          </p>
        </div>
        <div>
          <p className="text-xs text-slate-600 mb-1">Model Confidence</p>
          <p className="font-bold">{(prediction.confidence * 100).toFixed(0)}%</p>
        </div>
        <div className="col-span-2">
          <p className="text-xs text-slate-600 mb-2">Risk Factors</p>
          <div className="space-y-2">
            {prediction.top_risk_factors?.slice(0, 4).map((factor, idx) => (
              <div key={idx} className="flex items-center justify-between text-sm">
                <span>{factor.name}</span>
                <span className="text-slate-600">{(factor.contribution * 100).toFixed(0)}% contrib.</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </Card>
  )
}

interface ExplainabilityViewProps {
  explainability: Explainability
}

function ExplainabilityView({ explainability }: ExplainabilityViewProps) {
  const chartData = Object.entries(explainability.feature_importance || {})
    .map(([name, value]) => ({
      name,
      importance: value
    }))
    .sort((a, b) => b.importance - a.importance)
    .slice(0, 8)

  const localContributions = (explainability.local_contributions || [])
    .slice(0, 5)
    .map(c => ({
      name: c.feature,
      shap: c.shap_value,
      abs_shap: c.abs_shap_value
    }))

  return (
    <>
      {/* Feature Importance */}
      <Card title="Feature Importance (SHAP)" className="h-96">
        {chartData.length > 0 ? (
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" angle={-45} textAnchor="end" height={80} />
              <YAxis />
              <Tooltip />
              <Bar dataKey="importance" fill="#3b82f6" />
            </BarChart>
          </ResponsiveContainer>
        ) : (
          <p className="text-slate-500">No feature importance data</p>
        )}
      </Card>

      {/* SHAP Contributions */}
      <Card title="Local Feature Contributions" className="space-y-3">
        {localContributions.map((contrib, idx) => (
          <div key={idx} className="space-y-1">
            <div className="flex items-center justify-between">
              <span className="font-medium">{contrib.name}</span>
              <span className={`font-bold ${contrib.shap > 0 ? 'text-red-600' : 'text-emerald-600'}`}>
                {contrib.shap > 0 ? '+' : ''}{contrib.shap.toFixed(3)}
              </span>
            </div>
            <div className="flex h-2 bg-slate-200 rounded">
              <div
                className={`h-2 rounded ${contrib.shap > 0 ? 'bg-red-400' : 'bg-emerald-400'}`}
                style={{ width: `${Math.abs(contrib.shap) * 100}%` }}
              ></div>
            </div>
          </div>
        ))}
      </Card>

      {/* Metadata */}
      <Card className="p-4 text-xs text-slate-600">
        <div className="grid grid-cols-2 gap-4">
          <div>
            <p className="font-medium">Base Value</p>
            <p>{explainability.base_value?.toFixed(3)}</p>
          </div>
          <div>
            <p className="font-medium">Explanation Type</p>
            <p>{explainability.explanation_type}</p>
          </div>
          <div>
            <p className="font-medium">Computation Time</p>
            <p>{explainability.computation_time_ms?.toFixed(1)}ms</p>
          </div>
        </div>
      </Card>
    </>
  )
}

function PredictionsSidebar() {
  return (
    <nav className="p-6 space-y-4">
      <a href="/" className="block px-4 py-2 rounded-lg hover:bg-brand-light">
        Dashboard
      </a>
      <a href="/chains" className="block px-4 py-2 rounded-lg hover:bg-brand-light">
        Chains
      </a>
      <a href="/predictions" className="block px-4 py-2 rounded-lg bg-blue-600 text-white">
        Predictions
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

function PredictionsHeader() {
  return (
    <div className="px-8 py-4 flex items-center justify-between border-b border-slate-200">
      <h1 className="text-2xl font-bold text-slate-900">AI Predictions & Explainability</h1>
    </div>
  )
}
