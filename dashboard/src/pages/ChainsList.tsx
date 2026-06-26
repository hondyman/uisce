import React, { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Layout, Card, Button, Spinner, ErrorMessage, Badge } from '../components/Layout'
import { Chain, ChainHealthReport } from '../types'
import { apiClient } from '../api/client'

export function ChainsList() {
  const navigate = useNavigate()
  const tenantId = localStorage.getItem('tenant_id') || 'default'
  const [chains, setChains] = useState<(Chain & { health?: ChainHealthReport })[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [searchTerm, setSearchTerm] = useState('')
  const [filterStatus, setFilterStatus] = useState<'all' | 'healthy' | 'at-risk' | 'failed'>('all')
  const [sortBy, setSortBy] = useState<'name' | 'status' | 'updated'>('name')

  const fetchChains = async () => {
    try {
      setLoading(true)
      setError(null)

      // Mock chains data
      const mockChains: (Chain & { health?: ChainHealthReport })[] = [
        {
          id: 'chain-1',
          name: 'Payment Processing Chain',
          tenant_id: tenantId,
          region: 'us-east-1',
          is_active: true,
          created_at: new Date(Date.now() - 90 * 24 * 3600000).toISOString(),
          updated_at: new Date(Date.now() - 2 * 3600000).toISOString(),
          health: {
            id: 'health-1',
            chain_id: 'chain-1',
            region: 'us-east-1',
            is_healthy: true,
            health_score: 0.94,
            last_check_at: new Date(Date.now() - 300000).toISOString(),
            active_conflicts: 0,
            resolved_conflicts: 45,
            p99_latency_ms: 240
          }
        },
        {
          id: 'chain-2',
          name: 'Settlement Chain',
          tenant_id: tenantId,
          region: 'eu-west-1',
          is_active: true,
          created_at: new Date(Date.now() - 60 * 24 * 3600000).toISOString(),
          updated_at: new Date(Date.now() - 1 * 3600000).toISOString(),
          health: {
            id: 'health-2',
            chain_id: 'chain-2',
            region: 'eu-west-1',
            is_healthy: true,
            health_score: 0.87,
            last_check_at: new Date(Date.now() - 180000).toISOString(),
            active_conflicts: 2,
            resolved_conflicts: 23,
            p99_latency_ms: 320
          }
        },
        {
          id: 'chain-3',
          name: 'Risk Monitoring Chain',
          tenant_id: tenantId,
          region: 'apac-1',
          is_active: true,
          created_at: new Date(Date.now() - 45 * 24 * 3600000).toISOString(),
          updated_at: new Date().toISOString(),
          health: {
            id: 'health-3',
            chain_id: 'chain-3',
            region: 'apac-1',
            is_healthy: false,
            health_score: 0.62,
            last_check_at: new Date().toISOString(),
            active_conflicts: 8,
            resolved_conflicts: 15,
            p99_latency_ms: 850
          }
        },
        {
          id: 'chain-4',
          name: 'Compliance Audit Chain',
          tenant_id: tenantId,
          region: 'us-west-1',
          is_active: true,
          created_at: new Date(Date.now() - 30 * 24 * 3600000).toISOString(),
          updated_at: new Date(Date.now() - 24 * 3600000).toISOString(),
          health: {
            id: 'health-4',
            chain_id: 'chain-4',
            region: 'us-west-1',
            is_healthy: false,
            health_score: 0.71,
            last_check_at: new Date(Date.now() - 24 * 3600000).toISOString(),
            active_conflicts: 5,
            resolved_conflicts: 32,
            p99_latency_ms: 540
          }
        }
      ]

      setChains(mockChains)
    } catch (err) {
      console.error('Failed to fetch chains:', err)
      setError(err instanceof Error ? err.message : 'Failed to load chains')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchChains()
  }, [])

  const getChainStatus = (health?: ChainHealthReport): 'healthy' | 'at-risk' | 'failed' => {
    if (!health) return 'healthy'
    if (health.is_healthy && health.health_score >= 0.9) return 'healthy'
    if (health.health_score < 0.7) return 'failed'
    return 'at-risk'
  }

  let filtered = chains.filter(chain => {
    const matchesSearch = 
      chain.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      chain.id.toLowerCase().includes(searchTerm.toLowerCase())

    const status = getChainStatus(chain.health)
    const matchesFilter = 
      filterStatus === 'all' || 
      status === filterStatus

    return matchesSearch && matchesFilter
  })

  // Sort
  filtered = filtered.sort((a, b) => {
    switch (sortBy) {
      case 'status':
        const statusOrder = { healthy: 0, 'at-risk': 1, failed: 2 }
        const aStatus = getChainStatus(a.health)
        const bStatus = getChainStatus(b.health)
        return statusOrder[aStatus] - statusOrder[bStatus]
      case 'updated':
        return new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
      case 'name':
      default:
        return a.name.localeCompare(b.name)
    }
  })

  if (loading) return <Spinner />

  return (
    <Layout sidebar={<ChainsSidebar />} header={<ChainsHeader />}>
      <div className="space-y-6">
        {/* Search & Filter */}
        <Card className="p-4 space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <input
              type="text"
              placeholder="Search chains..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="px-4 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            />

            <select
              value={filterStatus}
              onChange={(e) => setFilterStatus(e.target.value as any)}
              className="px-4 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="all">All Statuses</option>
              <option value="healthy">Healthy</option>
              <option value="at-risk">At Risk</option>
              <option value="failed">Failed</option>
            </select>

            <select
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value as any)}
              className="px-4 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="name">Sort by Name</option>
              <option value="status">Sort by Status</option>
              <option value="updated">Sort by Updated</option>
            </select>
          </div>
        </Card>

        {/* Chains Grid */}
        {error ? (
          <ErrorMessage message={error} />
        ) : filtered.length > 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {filtered.map(chain => (
              <ChainCard key={chain.id} chain={chain} onSelect={() => navigate(`/chains/${chain.id}`)} />
            ))}
          </div>
        ) : (
          <Card className="text-center py-12">
            <p className="text-slate-500">No chains found</p>
          </Card>
        )}
      </div>
    </Layout>
  )
}

interface ChainCardProps {
  chain: Chain & { health?: ChainHealthReport }
  onSelect: () => void
}

function ChainCard({ chain, onSelect }: ChainCardProps) {
  const health = chain.health

  const getStatusColor = (status: 'healthy' | 'at-risk' | 'failed') => {
    switch (status) {
      case 'healthy':
        return { bg: 'bg-emerald-50', border: 'border-emerald-200', text: 'text-emerald-700', badge: 'emerald' }
      case 'at-risk':
        return { bg: 'bg-amber-50', border: 'border-amber-200', text: 'text-amber-700', badge: 'warning' }
      case 'failed':
        return { bg: 'bg-red-50', border: 'border-red-200', text: 'text-red-700', badge: 'danger' }
    }
  }

  const status = health
    ? (health.is_healthy && health.health_score >= 0.9 ? 'healthy' : health.health_score < 0.7 ? 'failed' : 'at-risk')
    : 'healthy'

  const colors = getStatusColor(status)
  const latencyStatus = health ? (health.p99_latency_ms > 500 ? 'warning' : 'success') : 'success'

  return (
    <Card
      className={`cursor-pointer transition-all hover:shadow-lg ${colors.bg} border-2 ${colors.border}`}
      onClick={onSelect}
    >
      <div className="space-y-4">
        {/* Header */}
        <div className="flex items-start justify-between">
          <div>
            <h3 className="text-lg font-bold text-slate-900">{chain.name}</h3>
            <p className="text-sm text-slate-600">{chain.id}</p>
          </div>
          <Badge status={status as any} />
        </div>

        {/* Region & Active */}
        <div className="flex items-center gap-4 text-sm">
          <span className="px-2 py-1 bg-white bg-opacity-60 rounded text-slate-700">
            📍 {chain.region}
          </span>
          {chain.is_active && (
            <span className="px-2 py-1 bg-emerald-100 rounded text-emerald-700 font-medium">
              ✓ Active
            </span>
          )}
        </div>

        {/* Health Metrics */}
        {health && (
          <div className="space-y-2 pt-2 border-t border-current border-opacity-10">
            <div className="flex items-center justify-between">
              <span className="text-sm text-slate-700">Health Score</span>
              <span className={`text-lg font-bold ${colors.text}`}>
                {(health.health_score * 100).toFixed(0)}%
              </span>
            </div>

            <div className="flex items-center justify-between">
              <span className="text-sm text-slate-700">Active Conflicts</span>
              <span className={`font-bold ${health.active_conflicts > 5 ? 'text-red-600' : 'text-slate-700'}`}>
                {health.active_conflicts}
              </span>
            </div>

            <div className="flex items-center justify-between">
              <span className="text-sm text-slate-700">P99 Latency</span>
              <span className={`font-bold ${latencyStatus === 'warning' ? 'text-amber-600' : 'text-emerald-600'}`}>
                {health.p99_latency_ms}ms
              </span>
            </div>
          </div>
        )}

        {/* Last Check */}
        {health && (
          <div className="text-xs text-slate-600">
            Last checked: {new Date(health.last_check_at).toLocaleTimeString()}
          </div>
        )}

        {/* Action Buttons */}
        <div className="flex gap-2 pt-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={(e) => {
              e.stopPropagation()
              onSelect()
            }}
          >
            View Details
          </Button>
        </div>
      </div>
    </Card>
  )
}

function ChainsSidebar() {
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

function ChainsHeader() {
  return (
    <div className="px-8 py-4 flex items-center justify-between border-b border-slate-200">
      <h1 className="text-2xl font-bold text-slate-900">Trading Chains</h1>
    </div>
  )
}
