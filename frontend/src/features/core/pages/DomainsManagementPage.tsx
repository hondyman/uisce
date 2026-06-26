import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Box,
  Typography,
  Button,
  Paper,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
  IconButton, Alert, LinearProgress, Tabs, Tab
} from '@mui/material'
import EditIcon from '@mui/icons-material/Edit'
import DeleteIcon from '@mui/icons-material/Delete'
import AddIcon from '@mui/icons-material/Add'
import AbbreviationManager from '../../../components/AbbreviationManager'

type Domain = {
  id?: string
  name: string
  slug?: string
  parent_id?: string | null
  level?: number
  description?: string
}

function normalizeDomain(raw: any): Domain {
  if (!raw) {
    return {
      id: undefined,
      name: '',
      slug: undefined,
      parent_id: null,
      level: undefined,
      description: '',
    }
  }

  const descriptionValue = typeof raw.description === 'string'
    ? raw.description
    : raw?.description?.String ?? ''

  return {
    id: raw.id,
    name: typeof raw.name === 'string' ? raw.name : String(raw.name ?? ''),
    slug: typeof raw.slug === 'string' ? raw.slug : undefined,
    parent_id: raw.parent_id ?? null,
    level: typeof raw.level === 'number' ? raw.level : undefined,
    description: descriptionValue || '',
  }
}

export default function DomainsManagementPage() {
  const navigate = useNavigate()
  const [domains, setDomains] = useState<Domain[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [banner, setBanner] = useState<{ severity: 'success' | 'error'; message: string } | null>(null)
  const [activeTab, setActiveTab] = useState(0)

  useEffect(() => {
    fetchList()
  }, [])
  async function fetchList() {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch('/api/data-domains', { credentials: 'include' })
      if (!res.ok) {
        throw new Error('Failed to load domains')
      }
      const json = await res.json()
      const rows: Domain[] = Array.isArray(json) ? json.map(normalizeDomain) : []
      setDomains(rows)
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      setError(msg || 'Failed to load domains')
    } finally {
      setLoading(false)
    }
  }

  function handleCreate() {
    setError(null)
    navigate('/core/domains/new')
  }

  function handleEdit(d: Domain) {
    setError(null)
    if (d?.id) navigate(`/core/domains/${d.id}`)
  }

  async function handleDelete(id?: string) {
    if (!id) return
    if (!confirm('Delete domain?')) return
    try {
      const res = await fetch(`/api/data-domains/${id}`, { method: 'DELETE', credentials: 'include' })
      if (!res.ok) {
        throw new Error('Delete failed')
      }
      setBanner({ severity: 'success', message: 'Domain deleted' })
      await fetchList()
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      setBanner({ severity: 'error', message: msg || 'Failed to delete domain' })
    }
  }

  const domainMap = new Map<string, Domain>()
  domains.forEach((d) => {
    if (d.id) {
      domainMap.set(d.id, d)
    }
  })

  const hasDomains = domains.length > 0

  const parentName = (parentId?: string | null) => {
    if (!parentId) return '—'
    const parent = domainMap.get(parentId)
    return parent?.name || parentId
  }

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}>
        <Tabs value={activeTab} onChange={(_, newValue) => setActiveTab(newValue)}>
          <Tab label="Data Domains" />
          <Tab label="Abbreviations" />
        </Tabs>
      </Box>

      {activeTab === 0 && (
        <>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2, flexWrap: 'wrap', gap: 2 }}>
            <Typography variant="h5">Core Domains</Typography>
            <Button variant="contained" startIcon={<AddIcon />} onClick={handleCreate}>New Domain</Button>
          </Box>

          {banner && (
            <Box sx={{ mb: 2 }}>
              <Alert severity={banner.severity} onClose={() => setBanner(null)}>{banner.message}</Alert>
            </Box>
          )}

          {error && (
            <Box sx={{ mb: 2 }}>
              <Alert severity="error" onClose={() => setError(null)}>{error}</Alert>
            </Box>
          )}

          <Paper sx={{ overflowX: 'auto' }}>
            {loading && <LinearProgress />}
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Name</TableCell>
                  <TableCell>Level</TableCell>
                  <TableCell>Parent</TableCell>
                  <TableCell>Description</TableCell>
                  <TableCell>Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {domains.map((d: Domain) => (
                  <TableRow key={d.id} hover sx={{ cursor: 'pointer' }} onClick={() => navigate(`/core/domains/${d.id}`)}>
                    <TableCell>{d.name}</TableCell>
                    <TableCell>{d.level}</TableCell>
                    <TableCell>{parentName(d.parent_id)}</TableCell>
                    <TableCell>{d.description}</TableCell>
                    <TableCell>
                      <IconButton onClick={(e) => { e.stopPropagation(); handleEdit(d); }}><EditIcon fontSize="small" /></IconButton>
                      <IconButton onClick={(e) => { e.stopPropagation(); handleDelete(d.id); }}><DeleteIcon fontSize="small" /></IconButton>
                    </TableCell>
                  </TableRow>
                ))}
                {!loading && !hasDomains && (
                  <TableRow>
                    <TableCell colSpan={6} align="center" sx={{ py: 3, color: 'text.secondary' }}>
                      No domains yet. Use "New Domain" to add your first domain.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </Paper>
        </>
      )}

      {activeTab === 1 && (
        <AbbreviationManager />
      )}
    </Box>
  )
}