import { useCallback } from 'react'

function getTenantHeaders() {
  try {
    const t = window.localStorage.getItem('selected_tenant')
    const d = window.localStorage.getItem('selected_datasource')
    const tenant = t ? JSON.parse(t) : null
    const datasource = d ? JSON.parse(d) : null
    const headers: Record<string, string> = { 'Content-Type': 'application/json' }
    if (tenant && tenant.id) headers['X-Tenant-ID'] = tenant.id
    if (datasource && datasource.id) headers['X-Tenant-Datasource-ID'] = datasource.id
    return headers
  } catch (err) {
    return { 'Content-Type': 'application/json' }
  }
}

export function useABAC() {
  // Calls backend /api/abac/evaluate with exponential backoff and tenant headers.
  const evaluate = useCallback(async (action: string, resource: string) => {
    const url = '/api/abac/evaluate'
    const headers = getTenantHeaders()
    const body = JSON.stringify({ action, resource })
    const maxAttempts = 3
    let attempt = 0
    let lastErr: any = null

    while (attempt < maxAttempts) {
      try {
        const res = await fetch(url, { method: 'POST', headers, body })
        // Prefer JSON body { allowed: true/false } if returned
        const contentType = res.headers.get ? res.headers.get('content-type') : null
        if (contentType && contentType.indexOf('application/json') !== -1) {
          try {
            const data = await res.json()
            if (typeof data.allowed === 'boolean') return data.allowed
          } catch (e) {
            // fall through to status-based logic
          }
        }

        // Fallback to status-code based logic
        if (res.status === 200) return true
        if (res.status === 403 || res.status === 401) return false
        if (res.status >= 400 && res.status < 500) return false
        // 5xx -> transient and retry
        lastErr = new Error(`abac evaluate unexpected status ${res.status}`)
      } catch (err) {
        lastErr = err
      }

      attempt += 1
      const backoff = 100 * Math.pow(2, attempt) // 200, 400, ...
      await new Promise((r) => setTimeout(r, backoff))
    }

    // Fallback: if running on localhost during development, return allow so dev isn't blocked.
    try {
      const host = window.location.hostname
      if (host === 'localhost' || host === '127.0.0.1') {
        console.warn('ABAC evaluate failed after retries, allowing in dev. lastErr=', lastErr)
        return true
      }
    } catch (e) {
      // ignore
    }

    console.error('ABAC evaluate failed after retries', lastErr)
    return false
  }, [])

  return { evaluate }
}

