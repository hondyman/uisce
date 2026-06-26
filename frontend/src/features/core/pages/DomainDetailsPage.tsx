import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Box,
  Typography,
  Button,
  Paper,
  TextField,
  Alert,
  CircularProgress,
  Tabs,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tooltip,
  Link,
} from '@mui/material'
import DomainTypeahead from '../../../components/DomainTypeahead'
import { listPolicies } from '../../../services/policyService'
import type { AccessControlPolicy } from '../../../types'

type Domain = {
  id?: string
  name: string
  slug?: string
  parent_id?: string | null
  level?: number
  description?: string
}

function slugify(value: string) {
  return value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9\s-]/g, '')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')
}

export default function DomainDetailsPage() {
  const { id } = useParams()
  const navigate = useNavigate()

  const [domain, setDomain] = useState<Domain | null>(null)
  const [domains, setDomains] = useState<Domain[]>([])
  const [policies, setPolicies] = useState<AccessControlPolicy[]>([])
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [tabIndex, setTabIndex] = useState<number>(0)

  useEffect(() => {
    fetchAll()
    fetchPolicies()
  }, [id])

  async function fetchPolicies() {
    try {
      const allPolicies = await listPolicies()
      setPolicies(allPolicies)
    } catch (e: any) {
      // ignore errors for now, policies are optional
    }
  }

  useEffect(() => {
    if (!id || id === 'new') {
      setDomain({ name: '', parent_id: null, level: 1, description: '' })
      return
    }
    // load single domain
    setLoading(true)
    fetch(`/api/data-domains/${id}`, { credentials: 'include' })
      .then((r) => {
        if (!r.ok) throw new Error('failed to load')
        return r.json()
      })
      .then((d) => {
        setDomain({
          id: d.id,
          name: d.name || '',
          slug: typeof d.slug === 'string' ? d.slug : d?.slug?.String ?? undefined,
          parent_id: d.parent_id ?? null,
          level: typeof d.level === 'number' ? d.level : undefined,
          description: typeof d.description === 'string' ? d.description : d?.description?.String ?? '',
        })
      })
      .catch((e) => setError(e?.message || String(e)))
      .finally(() => setLoading(false))
  }, [id])

  async function fetchAll() {
    try {
      const res = await fetch('/api/data-domains', { credentials: 'include' })
      if (!res.ok) throw new Error('Failed to fetch domains')
      const json = await res.json()
      setDomains(Array.isArray(json) ? json.map((r: any) => ({
        id: r.id,
        name: r.name,
        slug: typeof r.slug === 'string' ? r.slug : r?.slug?.String ?? undefined,
        parent_id: r.parent_id ?? null,
        level: typeof r.level === 'number' ? r.level : undefined,
        description: typeof r.description === 'string' ? r.description : r?.description?.String ?? '',
      })) : [])
    } catch (e: any) {
      setError(e?.message || String(e))
    }
  }

  const domainMap = useMemo(() => {
    const m = new Map<string, Domain>()
    domains.forEach((d) => { if (d.id) m.set(d.id, d) })
    return m
  }, [domains])

  // compute parent chain
  const parentChain = useMemo(() => {
    if (!domain) return []
    const chain: Domain[] = []
    let curr = domain
    while (curr?.parent_id) {
      const p = domainMap.get(curr.parent_id)
      if (!p) break
      chain.unshift(p)
      curr = p
    }
    return chain
  }, [domain, domainMap])

  // compute children domains
  const childrenDomains = useMemo(() => {
    if (!domain?.id) return []
    return domains.filter((d) => d.parent_id === domain.id)
  }, [domains, domain?.id])

  // filter policies associated with this domain and inherited from parents
  const associatedPolicies = useMemo(() => {
    if (!domain?.id) return { direct: [], inherited: [] }
    const direct = policies.filter((p) => p.scope === `domain:${domain.id}`)
    const inherited: Array<AccessControlPolicy & { inheritedFrom: string }> = []
    parentChain.forEach((parent) => {
      if (parent.id) {
        const parentPolicies = policies.filter((p) => p.scope === `domain:${parent.id}`)
        parentPolicies.forEach((p) => {
          inherited.push({ ...p, inheritedFrom: parent.name })
        })
      }
    })
    return { direct, inherited }
  }, [policies, domain?.id, parentChain])

  useEffect(() => {
    // when parent_id changes, auto compute level
    if (!domain) return
    if (!domain.parent_id) {
      setDomain((d) => d ? { ...d, level: 1 } : d)
      return
    }
    const p = domainMap.get(domain.parent_id)
    const newLevel = p?.level ? Math.min((p.level || 1) + 1, 3) : 2
    setDomain((d) => d ? { ...d, level: newLevel } : d)
  }, [domain?.parent_id, domainMap])

  function handleChange<K extends keyof Domain>(key: K, val: Domain[K]) {
    setDomain((d) => (d ? { ...d, [key]: val } : d))
  }

  function handleTabChange(_: React.SyntheticEvent, newValue: number) {
    setTabIndex(newValue)
  }

  async function handleSave() {
    if (!domain) return
    if (!domain.name?.trim()) {
      setError('Name is required')
      return
    }
    setSaving(true)
    setError(null)
    const generatedSlug = (domain.slug || (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function' ? crypto.randomUUID() : slugify(domain.name || '')))
    const payload = {
      ...domain,
      // For new domains, slug must be a UUID. If editing and slug present, preserve it.
      slug: (domain.id ? (domain.slug || generatedSlug) : generatedSlug).toString().trim(),
      parent_id: domain.parent_id || null,
      level: Math.min(Math.max(Number(domain.level ?? 1), 1), 3),
      // send description as sql.NullString-like shape so backend can decode it
      description: {
        String: domain.description || '',
        Valid: Boolean(domain.description && domain.description.length > 0),
      },
    }

    try {
      const url = domain.id ? `/api/data-domains/${domain.id}` : '/api/data-domains'
      const method = domain.id ? 'PUT' : 'POST'
      const res = await fetch(url, {
        method,
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!res.ok) {
        const text = await res.text()
        throw new Error(text || 'Save failed')
      }
      // go back to list
      navigate('/core/domains')
    } catch (e: any) {
      setError(e?.message || String(e))
    } finally {
      setSaving(false)
    }
  }

  if (loading || !domain) {
    return (
      <Box sx={{ p: 3 }}>
        <CircularProgress />
      </Box>
    )
  }

  return (
    <Box sx={{ p: 3 }}>
      <Paper sx={{ p: 0, mb: 2 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', px: 3, py: 2 }}>
          <Typography variant="h5">{domain.id ? 'Edit Domain' : 'Create Domain'}</Typography>
          <Box>
            <Button onClick={() => navigate('/core/domains')} sx={{ mr: 1 }}>Back</Button>
            <Button variant="contained" onClick={handleSave} disabled={saving}>{saving ? 'Saving...' : 'Save'}</Button>
          </Box>
        </Box>

        <Tabs value={tabIndex} onChange={handleTabChange} aria-label="Domain tabs" sx={{ borderTop: 1, borderColor: 'divider' }}>
          <Tab label="Properties" />
          <Tab label="Data" />
          <Tab label="Policy" />
        </Tabs>

        <Box sx={{ p: 3 }}>
          {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

          {tabIndex === 0 && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <TextField label="Name" size="small" value={domain.name} onChange={(e) => handleChange('name', e.target.value)} />
              <TextField label="Slug (auto-generated)" size="small" value={domain.slug || slugify(domain.name || '')} onChange={(e) => handleChange('slug', e.target.value)} helperText="Editable slug. If blank it will be generated from the name." />
              <DomainTypeahead
                value={domain.parent_id ?? null}
                onChange={(v) => handleChange('parent_id', v ?? null)}
                label="Parent domain"
                placeholder="Search domains to nest under"
                helperText="Optional. Choose a parent to nest under an existing domain"
                allowClear
              />
              <TextField label="Level" type="number" size="small" value={domain.level || 1} onChange={(e) => handleChange('level', Number(e.target.value))} inputProps={{ min: 1, max: 3 }} helperText="Hierarchy depth: 1 (root) to 3 (leaf)" />
              <TextField label="Description" size="small" multiline minRows={3} value={domain.description || ''} onChange={(e) => handleChange('description', e.target.value)} />

              <Box>
                <Typography variant="subtitle2">Hierarchy</Typography>
                <Box sx={{ display: 'flex', gap: 1, mt: 1, flexWrap: 'wrap', alignItems: 'center' }}>
                  {parentChain.map((p, i) => (
                    <Tooltip key={p.id || i} title={`Level ${p.level} • ${p.description || 'No description'}`}>
                      <Link
                        component="button"
                        variant="body2"
                        onClick={() => navigate(`/core/domains/${p.id}`)}
                        sx={{ fontWeight: 500, textDecoration: 'underline' }}
                      >
                        {p.name} /
                      </Link>
                    </Tooltip>
                  ))}
                  <Tooltip title={`Level ${domain.level} • ${domain.description || 'No description'}`}>
                    <Typography sx={{ fontWeight: 700, color: 'primary.main' }}>{domain.name || '(current)'}</Typography>
                  </Tooltip>
                </Box>
                {childrenDomains.length > 0 && (
                  <Box sx={{ mt: 1 }}>
                    <Typography variant="caption" sx={{ fontWeight: 500 }}>Children:</Typography>
                    <Box sx={{ display: 'flex', gap: 1, mt: 0.5, flexWrap: 'wrap' }}>
                      {childrenDomains.map((child) => (
                        <Tooltip key={child.id} title={`Level ${child.level} • ${child.description || 'No description'}`}>
                          <Link
                            component="button"
                            variant="caption"
                            onClick={() => navigate(`/core/domains/${child.id}`)}
                            sx={{ textDecoration: 'underline' }}
                          >
                            {child.name}
                          </Link>
                        </Tooltip>
                      ))}
                    </Box>
                  </Box>
                )}
                <Box sx={{ mt: 1 }}>
                  <Typography variant="caption">Selected level: {domain.level}</Typography>
                </Box>
              </Box>
            </Box>
          )}

          {tabIndex === 1 && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <Typography variant="body2" color="text.secondary">Data tab: additional data and metrics related to the domain can be shown here.</Typography>
            </Box>
          )}

          {tabIndex === 2 && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <Typography variant="body2" color="text.secondary">
                Policies associated with this domain ({associatedPolicies.direct.length + associatedPolicies.inherited.length}):
              </Typography>
              {(associatedPolicies.direct.length > 0 || associatedPolicies.inherited.length > 0) ? (
                <TableContainer component={Paper}>
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>Policy ID</TableCell>
                        <TableCell>Role</TableCell>
                        <TableCell>Permissions</TableCell>
                        <TableCell>Duration (days)</TableCell>
                        <TableCell>Type</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {associatedPolicies.direct.map((policy) => (
                        <TableRow key={policy.id}>
                          <TableCell>{policy.policy_id}</TableCell>
                          <TableCell>{policy.role}</TableCell>
                          <TableCell>{Array.isArray(policy.permissions) ? policy.permissions.join(', ') : policy.permissions}</TableCell>
                          <TableCell>{policy.duration_days}</TableCell>
                          <TableCell><Typography variant="body2" color="primary">Direct</Typography></TableCell>
                        </TableRow>
                      ))}
                      {associatedPolicies.inherited.map((policy) => (
                        <TableRow key={`${policy.id}-inherited`}>
                          <TableCell>{policy.policy_id}</TableCell>
                          <TableCell>{policy.role}</TableCell>
                          <TableCell>{Array.isArray(policy.permissions) ? policy.permissions.join(', ') : policy.permissions}</TableCell>
                          <TableCell>{policy.duration_days}</TableCell>
                          <TableCell><Typography variant="body2" color="text.secondary">Inherited from {policy.inheritedFrom}</Typography></TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              ) : (
                <Typography variant="body2" color="text.secondary">
                  No policies associated with this domain.
                </Typography>
              )}
            </Box>
          )}
        </Box>
      </Paper>
    </Box>
  )
}
