import type { AccessControlPolicy } from '../types'
import resolveApiUrl from '../utils/resolveApiUrl';

const API_BASE = '/api/policies'

function normalizePolicyPayload(payload: Partial<AccessControlPolicy>): Partial<AccessControlPolicy> {
  const out: Partial<AccessControlPolicy> = { ...payload }

  if (Array.isArray(out.permissions)) {
    out.permissions = out.permissions.map((p) => String(p).trim()).filter(Boolean)
  } else if (out.permissions) {
    out.permissions = [String(out.permissions)].filter(Boolean)
  } else {
    out.permissions = []
  }

  if (out.duration_days !== undefined) {
    const parsed = Number(out.duration_days)
    if (Number.isNaN(parsed)) {
      throw new Error('duration_days must be a number')
    }
    out.duration_days = parsed
  }

  if (out.max_claims_per_user !== undefined) {
    const parsed = Number(out.max_claims_per_user)
    if (Number.isNaN(parsed)) {
      throw new Error('max_claims_per_user must be a number')
    }
    out.max_claims_per_user = parsed
  }

  if (out.approval_threshold !== undefined) {
    const parsed = Number(out.approval_threshold)
    if (Number.isNaN(parsed)) {
      throw new Error('approval_threshold must be a number')
    }
    out.approval_threshold = parsed
  }

  const renewal = out.renewal_conditions
  if (typeof renewal === 'string') {
    try {
      out.renewal_conditions = JSON.parse(renewal)
    } catch (err) {
      throw new Error('Renewal conditions must be valid JSON')
    }
  }

  return out
}

export async function fetchPolicy(id: string): Promise<AccessControlPolicy> {
  const res = await fetch(resolveApiUrl(`/api/policies/${encodeURIComponent(id)}`), {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
  })
  if (!res.ok) throw new Error(`Failed to fetch policy: ${res.status}`)
  return res.json()
}

export async function listPolicies(): Promise<AccessControlPolicy[]> {
  const res = await fetch(resolveApiUrl(API_BASE), {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
  })
  if (!res.ok) throw new Error(`Failed to list policies: ${res.status}`)
  return res.json()
}

export async function savePolicy(payload: Partial<AccessControlPolicy>) {
  const bodyObj = normalizePolicyPayload(payload)

  const res = await fetch(resolveApiUrl(API_BASE), {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(bodyObj),
  })
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`Save failed: ${res.status} ${txt}`)
  }
  return res.json()
}

export async function simulatePolicy(payload: Partial<AccessControlPolicy>) {
  const res = await fetch(resolveApiUrl('/api/policies/simulate'), {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`Simulation failed: ${res.status} ${txt}`)
  }
  return res.json()
}

export async function deletePolicy(id: string) {
  const res = await fetch(resolveApiUrl(`/api/policies/${encodeURIComponent(id)}`), {
    method: 'DELETE',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
  })
  if (!res.ok) {
    const txt = await res.text()
    throw new Error(`Delete failed: ${res.status} ${txt}`)
  }
  return res.json()
}
