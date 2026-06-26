import { useEffect, useState } from 'react'
import { useNotification } from '../hooks/useNotification';
import { useAuthFetch } from '../utils/authFetch';

type Bundle = {
  id: string
  name?: string
  description?: string
  claims?: string[]
  score?: number
  risk?: number
  status?: string
}

export default function BundleRecommendationPanel() {
  const [bundles, setBundles] = useState<Bundle[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const notification = useNotification()

  useEffect(() => { loadProposals() }, [])
  const { authFetch } = useAuthFetch();

  const [guardrailReasons, setGuardrailReasons] = useState<Record<string,string[]>>({})

  async function loadProposals() {
    setLoading(true)
    setError(null)
    try {
  const res = await authFetch('/api/bundles/proposals')
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  // extract payload in a typed-local way to avoid blanket `as any` usages
  const _res = res as unknown as { data?: unknown; json?: (() => Promise<unknown>) | undefined };
  const _json = _res.json ? await _res.json().catch(() => ({})) : {};
  const body = _res.data !== undefined ? _res.data : _json;
      // normalize proposals -> Bundle[]
  const bodyAny = body as Record<string, unknown> | null;
  const proposals = Array.isArray(bodyAny?.proposals) ? (bodyAny?.proposals as unknown[]) : [];
  const list = proposals.map((p: unknown) => {
        const rec = p as Record<string, unknown>;
        let details: any = rec.details ?? {};
        try { details = typeof details === 'string' ? JSON.parse(details) : details } catch(e) {}
        return {
          id: String(rec.id ?? ''),
          name: details.description || details.name || `proposal-${String(rec.id ?? '').slice(0,6)}`,
          description: String(details.description ?? ''),
          claims: Array.isArray(details.claims) ? details.claims : [],
          score: (rec.fitness_score ?? rec.score) as any,
          risk: (rec.risk_score ?? rec.risk) as any,
          status: rec.status as any,
        } as Bundle
      })
      setBundles(list)
    } catch (err: unknown) {
      setError((err as Error)?.message || 'failed to load')
    } finally {
      setLoading(false)
    }
  }

  async function doApprove(id: string) {
    setError(null)
    try {
      const res = await authFetch(`/api/bundles/proposals/${id}/approve`, {
        method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify({approver: 'ui'})
      })
      const _res = res as unknown as { data?: unknown; json?: (() => Promise<unknown>) | undefined };
      const _json = _res.json ? await _res.json().catch(() => ({})) : {};
      const body = _res.data !== undefined ? _res.data : _json;
      if (!res.ok) {
        const bodyErr = (body as Record<string, unknown> | null)?.error;
        throw new Error((typeof bodyErr === 'string' && bodyErr) || `HTTP ${res.status}`)
      }
      // server may return reasons when approval is pending review
      const bodyResp = body as Record<string, unknown> | null;
      if (Array.isArray(bodyResp?.reasons)) {
        setGuardrailReasons(prev => ({...prev, [id]: bodyResp.reasons as string[]}))
      }
      // refresh list to pick up updated status
      await loadProposals()
    } catch (err: unknown) {
      setError((err as Error)?.message || 'approve failed')
    }
  }

  async function doReject(id: string) {
    setError(null)
    try {
      const res = await authFetch(`/api/bundles/proposals/${id}/reject`, {
        method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify({approver: 'ui', reason: 'rejected via UI'})
      })
      const _res = res as unknown as { data?: unknown; json?: (() => Promise<unknown>) | undefined };
      const _json = _res.json ? await _res.json().catch(() => ({})) : {};
      const body = _res.data !== undefined ? _res.data : _json;
      if (!res.ok) {
        const bodyErr = (body as Record<string, unknown> | null)?.error;
        throw new Error((typeof bodyErr === 'string' && bodyErr) || `HTTP ${res.status}`)
      }
      const bodyResp = body as Record<string, unknown> | null;
      if (Array.isArray(bodyResp?.reasons)) {
        setGuardrailReasons(prev => ({...prev, [id]: bodyResp.reasons as string[]}))
      }
      // refresh list after rejection
      await loadProposals()
    } catch (err: unknown) {
      setError((err as Error)?.message || 'reject failed')
    }
  }

  if (loading) return <div>Loading bundle recommendations…</div>
  if (error) return <div className="error">Error: {error}</div>

  return (
    <div className="bundle-panel">
      {bundles.map(b => (
        <div key={b.id} className="bundle-card">
          <h3>{b.name} {b.status && <small>• {b.status}</small>}</h3>
          <p>{b.description}</p>
          <div className="meta">Claims: {b.claims?.length ?? 0} • Score: {Math.round(b.score || 0)}</div>
          <div className="actions">
            <button onClick={() => doApprove(b.id)}>Approve</button>
            <button onClick={() => doReject(b.id)}>Reject</button>
            <button onClick={() => notification.info('View details for ' + b.id)}>View Details</button>
          </div>
          {guardrailReasons[b.id] && (
            <div className="guardrail-reasons">
              <strong>Guardrail reasons:</strong>
              <ul>
                {guardrailReasons[b.id].map((r,i) => <li key={i}>{r}</li>)}
              </ul>
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
