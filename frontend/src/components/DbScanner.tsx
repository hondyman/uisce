import { useEffect, useState, useCallback } from 'react'
import { devError } from '../utils/devLogger';
import { useScope } from '../contexts/ScopeContext'
import { Box, Button, CircularProgress, Paper, Typography, Table, TableHead, TableRow, TableCell, TableBody, Alert, IconButton, List, ListItem, ListItemIcon, ListItemText, Collapse, Checkbox, Chip, Tooltip } from '@mui/material'
import ProfessionalSearchInput from './common/ProfessionalSearchInput'
import FolderIcon from '@mui/icons-material/Folder'
import TableChartIcon from '@mui/icons-material/TableChart'
import AccountTreeIcon from '@mui/icons-material/AccountTree'
import RefreshIcon from '@mui/icons-material/Refresh'
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline'
import ExpandLess from '@mui/icons-material/ExpandLess'
import ExpandMore from '@mui/icons-material/ExpandMore'
import { useTenant } from '../contexts/TenantContext'

type DbScannerProps = {
  refreshMappings?: () => Promise<void>
  onProfile?: (schemaId: string | null, tableName: string | null, tableNames?: string[]) => void
  registerRunScan?: (fn: () => void) => void
  searchTerm?: string
}
import { hasTenantScope } from '../utils/tenantScope'

type CatalogNode = {
  id: string
  node_name: string
  qualified_path?: string
  catalog_type?: string
  parent_id?: string
  properties?: any
}

export default function DbScanner({ refreshMappings: _refreshMappings, onProfile, registerRunScan, searchTerm }: DbScannerProps) {
  const { tenant, datasource } = useTenant()
  const makeApiUrl = (path: string) => {
    // During test runs we prefer returning the relative path so the test's fetch
    // mock can intercept it without attempting real network connections.
    try {
      if ((typeof process !== 'undefined' && process.env && process.env.NODE_ENV === 'test') || import.meta.env.MODE === 'test') return path
      const base = (typeof window !== 'undefined' && (window as any).location && (window as any).location.origin) ? (window as any).location.origin : 'http://localhost'
      return new URL(path, base).toString()
    } catch (e) {
      return path
    }
  }
  const [schemas, setSchemas] = useState<CatalogNode[]>([])
  const [expanded, setExpanded] = useState<string[]>([])
  const [tablesBySchema, setTablesBySchema] = useState<Record<string, CatalogNode[]>>({})
  const [columnsByTable, setColumnsByTable] = useState<Record<string, CatalogNode[]>>({})
  const [profiledColumns, setProfiledColumns] = useState<Record<string, boolean>>({})
  // pagination state per table
  const [columnsPageByTable, setColumnsPageByTable] = useState<Record<string, { offset: number; limit: number; hasMore: boolean }>>({})
  const [columnsLoadingByTable, setColumnsLoadingByTable] = useState<Record<string, boolean>>({})
  const [selectedForScan, setSelectedForScan] = useState<Record<string, boolean>>({})
  const [selectedTable, setSelectedTable] = useState<CatalogNode | null>(null)
  const [selectedTablesForProfile, setSelectedTablesForProfile] = useState<string[]>([])
  const [columnFilter, setColumnFilter] = useState<string>('')
  const [loading, setLoading] = useState(false)
  const [scanRunning, setScanRunning] = useState(false)
  const [scanMessage, setScanMessage] = useState<string | null>(null)

  const canUseScope = hasTenantScope() && !!datasource && !!tenant

  const fetchSchemas = useCallback(async () => {
    setLoading(true)
    try {
  const res = await fetch(makeApiUrl('/api/catalog/nodes?type=schema&limit=500'), { credentials: 'include' })
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      setSchemas(Array.isArray(data) ? data : [])
    } catch (err: any) {
      devError('[DbScanner] fetchSchemas error', err)
      setSchemas([])
    } finally {
      setLoading(false)
    }
  }, [])

  const fetchTables = useCallback(async (schemaId: string) => {
    if (tablesBySchema[schemaId]) return
    try {
  const res = await fetch(makeApiUrl(`/api/catalog/nodes?type=table&parent_id=${encodeURIComponent(schemaId)}&limit=500`), { credentials: 'include' })
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      setTablesBySchema(prev => ({ ...prev, [schemaId]: Array.isArray(data) ? data : [] }))
    } catch (err) {
      devError('[DbScanner] fetchTables error', err)
      setTablesBySchema(prev => ({ ...prev, [schemaId]: [] }))
    }
  }, [tablesBySchema])

  // Fetch columns lazily with pagination. Keeps previous columns for the table and appends new ones.
  const fetchColumns = useCallback(async (tableId: string, opts?: { offset?: number; limit?: number; reset?: boolean; q?: string }) => {
    const limit = opts?.limit ?? 200
    const q = opts?.q ?? columnFilter
    const offset = opts?.offset ?? (opts?.reset ? 0 : (columnsPageByTable[tableId]?.offset ?? 0))
    // if no more items and not resetting, skip
    if (!opts?.reset && columnsPageByTable[tableId] && !columnsPageByTable[tableId].hasMore) return

    try {
      setColumnsLoadingByTable(prev => ({ ...prev, [tableId]: true }))
      const qParam = q ? `&q=${encodeURIComponent(String(q))}` : ''
  const res = await fetch(makeApiUrl(`/api/catalog/nodes?type=column&parent_id=${encodeURIComponent(tableId)}&limit=${limit}&offset=${offset}${qParam}`), { credentials: 'include' })
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      const items: CatalogNode[] = Array.isArray(data) ? data : []

      setColumnsByTable(prev => ({
        ...prev,
        [tableId]: opts?.reset ? items : [...(prev[tableId] || []), ...items]
      }))

      setColumnsPageByTable(prev => ({
        ...prev,
        [tableId]: {
          offset: offset + items.length,
          limit,
          hasMore: items.length === limit
        }
      }))
    } catch (err) {
      devError('[DbScanner] fetchColumns error', err)
      setColumnsByTable(prev => ({ ...prev, [tableId]: prev[tableId] || [] }))
      setColumnsPageByTable(prev => ({ ...prev, [tableId]: { offset: prev[tableId]?.offset ?? 0, limit: prev[tableId]?.limit ?? 200, hasMore: false } }))
    } finally {
      setColumnsLoadingByTable(prev => ({ ...prev, [tableId]: false }))
    }
  }, [columnsPageByTable, columnFilter])

  // when the columnFilter changes, reload first page for the currently selected table
  // Note: we intentionally do NOT include `selectedTable` in the dependency array to avoid
  // triggering a duplicate fetch when a table is selected (handleTableSelect already
  // performs the initial load). This effect should only run when the filter itself
  // changes while a table is selected.
  useEffect(() => {
    if (selectedTable) {
      const query = searchTerm || columnFilter
      // reset stored columns and pages for this table and fetch first page with q
      setColumnsByTable(prev => ({ ...prev, [selectedTable.id]: [] }))
      setColumnsPageByTable(prev => ({ ...prev, [selectedTable.id]: { offset: 0, limit: prev[selectedTable.id]?.limit ?? 200, hasMore: true } }))
      fetchColumns(selectedTable.id, { offset: 0, limit: columnsPageByTable[selectedTable.id]?.limit ?? 200, reset: true, q: query })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [columnFilter, searchTerm])

  useEffect(() => {
    if (canUseScope) fetchSchemas()
  }, [canUseScope, fetchSchemas])

  const scope = useScope()

  const applySelectionToMapper = () => {
    if (!selectedTable) return
    // find parent schema id by searching tablesBySchema
    let parentSchemaId: string | undefined
    let parentSchemaName: string | undefined
    for (const s of schemas) {
      const tables = tablesBySchema[s.id] || []
      if (tables.find(t => t.id === selectedTable.id)) {
        parentSchemaId = s.id
        parentSchemaName = s.node_name
        break
      }
    }

    try {
      if (parentSchemaId) {
        scope.setSchemaIds([parentSchemaId])
        scope.setSchemaNames([parentSchemaName || ''])
      }
      scope.setTableIds([selectedTable.id])
      scope.setTableNames([selectedTable.node_name])
      // Optionally refresh mappings if caller passed refreshMappings
      if (typeof _refreshMappings === 'function') _refreshMappings()
    } catch (e) {
      devError('[DbScanner] applySelectionToMapper error', e)
    }
  }
  const handleTableSelect = async (table: CatalogNode) => {
    setSelectedTable(table)
    // reset pagination and load first page
    await fetchColumns(table.id, { offset: 0, limit: 200, reset: true })
    // find parent schema name for this table (used for profiling lookup)
    let parentSchemaName: string | undefined
    for (const s of schemas) {
      const tables = tablesBySchema[s.id] || []
      if (tables.find(t => t.id === table.id)) {
        parentSchemaName = s.node_name
        break
      }
    }
    if (parentSchemaName) {
      // fetch profiler results for this table to determine which columns are profiled
      fetchProfileForTable(parentSchemaName, table.node_name)
    }
    // Do not auto-apply: selection just shows details. User can click "Apply to Semantic Mapper" to apply.
    // Still, update selectedTable which is used in the details pane.
  }

  const fetchProfileForTable = async (schemaName: string, tableName: string) => {
    try {
  const res = await fetch(makeApiUrl(`/api/profiler/results?schema=${encodeURIComponent(schemaName)}&table=${encodeURIComponent(tableName)}&limit=500`), { credentials: 'include' })
      if (!res.ok) return
      const data = await res.json().catch(() => null)
      if (!data || !Array.isArray(data.profiles)) return
      const map: Record<string, boolean> = {}
      data.profiles.forEach((p: any) => { if (p.ColumnName) map[p.ColumnName] = true })
      setProfiledColumns(map)
    } catch (err) {
      // ignore
    }
  }

  // profile the currently selected table (call back to parent)
  const handleRunProfile = () => {
    if (!selectedTable) return
    // find parent schema id
    let parentSchemaName: string | null = null
    for (const s of schemas) {
      const tables = tablesBySchema[s.id] || []
      if (tables.find(t => t.id === selectedTable.id)) {
        parentSchemaName = s.node_name
        break
      }
    }
    try {
      if (typeof onProfile === 'function') onProfile(parentSchemaName, selectedTable.node_name)
    } catch (e) {
      // swallow
    }
  }

  // keyboard shortcut: press 'p' to profile selected table
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'p' || e.key === 'P') {
        if (selectedTable) {
          e.preventDefault()
          handleRunProfile()
        }
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [selectedTable, schemas, tablesBySchema])

  // (applySelectionToMapper removed) Selection now only shows details; applying selection
  // to the Semantic Mapper is handled elsewhere when needed.

  const runScan = async () => {
    if (!datasource) return
    const selectedSchemaNames = schemas.filter(s => selectedForScan[s.id]).map(s => s.node_name)
    setScanRunning(true)
    setScanMessage(null)
    try {
      const body: any = { tenant_instance_id: datasource.id }
      if (selectedSchemaNames.length > 0) body.schema_names = selectedSchemaNames
  const res = await fetch(makeApiUrl('/api/catalog/scan'), { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body), credentials: 'include' })
      const data = await res.json().catch(() => null)
      if (!res.ok) {
        setScanMessage(`Scan failed: ${res.status} ${res.statusText} ${data?.error || ''}`)
      } else {
        if (data && data.results) {
          const succeeded = (data.results.filter ? data.results.filter((r: any) => r.success).length : 0)
          setScanMessage(`${data.message || 'Scan completed'} (${succeeded} succeeded, ${data.results.length - succeeded} failed)`)
        } else if (data && data.message) {
          setScanMessage(data.message)
        } else {
          setScanMessage('Scan completed')
        }
        // refresh schemas/tables
        await fetchSchemas()
        setTablesBySchema({})
        setColumnsByTable({})
      }
    } catch (err: any) {
      devError('[DbScanner] runScan error', err)
      setScanMessage(String(err.message || err))
    } finally {
      setScanRunning(false)
    }
  }

  // expose runScan to parent if requested
  useEffect(() => {
    try {
      if (typeof registerRunScan === 'function') {
        registerRunScan(runScan)
      }
    } catch (e) {}
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <Box sx={{ display: 'flex', gap: 3 }}>
      {!canUseScope && (
        <Alert severity="warning">Select a tenant and datasource to enable DB scanning</Alert>
      )}

      <Box sx={{ width: '360px', flexShrink: 0 }}>
        <Paper sx={{ p: 2, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
          <AccountTreeIcon />
          <Box>
            <Typography variant="h6">Database</Typography>
            <Typography variant="body2" color="text.secondary">Schemas and tables</Typography>
          </Box>
          <Box sx={{ flex: 1 }} />
          <IconButton size="small" onClick={() => { setTablesBySchema({}); setColumnsByTable({}); fetchSchemas() }}><RefreshIcon /></IconButton>
        </Paper>

        <Paper sx={{ maxHeight: '72vh', overflow: 'auto', p: 1 }}>
          {loading ? (
            <Box sx={{ p: 3, textAlign: 'center' }}><CircularProgress size={28} /></Box>
          ) : schemas.length === 0 ? (
            <Box sx={{ p: 2 }}><Typography>No schemas found</Typography></Box>
          ) : (
            <List>
              {schemas.map(schema => {
                const isExpanded = expanded.includes(schema.id)
                return (
                  <div key={schema.id}>
                    <ListItem
                      secondaryAction={isExpanded ? <ExpandLess /> : <ExpandMore />}
                      button
                      onClick={async () => {
                        // ensure tables are loaded, then toggle expand
                        if (!tablesBySchema[schema.id]) await fetchTables(schema.id)
                        setExpanded(prev => prev.includes(schema.id) ? prev.filter(id => id !== schema.id) : [...prev, schema.id])
                      }}
                      onKeyDown={async (e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault()
                          if (!tablesBySchema[schema.id]) await fetchTables(schema.id)
                          setExpanded(prev => prev.includes(schema.id) ? prev.filter(id => id !== schema.id) : [...prev, schema.id])
                        }
                      }}
                      data-testid={`schema-${schema.node_name}`}
                    >
                      <ListItemIcon>
                        <Checkbox
                          edge="start"
                          size="small"
                          checked={!!selectedForScan[schema.id]}
                          onClick={(e) => { e.stopPropagation(); setSelectedForScan(prev => ({ ...prev, [schema.id]: !prev[schema.id] })); }}
                        />
                      </ListItemIcon>
                      <ListItemIcon><FolderIcon fontSize="small" /></ListItemIcon>
                      <ListItemText primary={schema.node_name} />
                    </ListItem>

                    <Collapse in={isExpanded} timeout="auto" unmountOnExit>
                      <List component="div" disablePadding>
                          {(tablesBySchema[schema.id] || []).map(table => (
                          <ListItem key={table.id} button sx={{ pl: 4 }} onClick={() => handleTableSelect(table)} onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); handleTableSelect(table) } }} data-testid={`table-${table.node_name}`}>
                            <ListItemIcon>
                              <Checkbox
                                edge="start"
                                size="small"
                                checked={selectedTablesForProfile.includes(table.node_name)}
                                onClick={(e) => { e.stopPropagation(); setSelectedTablesForProfile(prev => prev.includes(table.node_name) ? prev.filter(x => x !== table.node_name) : [...prev, table.node_name]) }}
                                inputProps={{ 'aria-label': `profile-select-${table.node_name}` }}
                              />
                            </ListItemIcon>
                            <ListItemIcon><TableChartIcon fontSize="small" /></ListItemIcon>
                            <ListItemText primary={table.node_name} />
                          </ListItem>
                        ))}
                      </List>
                    </Collapse>
                  </div>
                )
              })}
            </List>
          )}
        </Paper>



        {scanMessage && <Typography variant="body2" sx={{ mt: 1 }}>{scanMessage}</Typography>}
      </Box>

      <Box sx={{ flex: 1 }}>
        <Paper sx={{ p: 2, mb: 2, display: 'flex', alignItems: 'center' }}>
          <TableChartIcon sx={{ mr: 1 }} />
          <Box>
            <Typography variant="h6">Table Details</Typography>
            <Typography variant="body2" color="text.secondary">Select a table to view its columns and properties.</Typography>
          </Box>
          <Box sx={{ flex: 1 }} />
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button variant="contained" size="small" onClick={applySelectionToMapper} disabled={!selectedTable} data-testid="apply-to-mapper">Apply to Semantic Mapper</Button>
            <Button variant="outlined" size="small" onClick={handleRunProfile} disabled={!selectedTable} data-testid="run-profile">Run Data Profile</Button>
            <Button variant="outlined" size="small" onClick={() => { setSelectedForScan({}); setTablesBySchema({}); setColumnsByTable({}); fetchSchemas() }}>Reload</Button>
          </Box>
        </Paper>

        <Paper sx={{ p: 2, minHeight: '50vh' }}>
          {!selectedTable ? (
            <Typography variant="body2" color="text.secondary">No table selected</Typography>
          ) : (
            <>
              <Box sx={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between' }}>
                <Box>
                  <Typography variant="subtitle1" sx={{ mb: 0 }}>{selectedTable.node_name}</Typography>
                  <Typography variant="caption" color="text.secondary">{selectedTable.qualified_path}</Typography>
                </Box>
              </Box>

              <Box sx={{ mt: 2 }}>
                {/* Search is now in global header */}
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Column</TableCell>
                      <TableCell>Type</TableCell>
                      <TableCell>Nullable</TableCell>
                      <TableCell>Extras</TableCell>
                    </TableRow>
                  </TableHead>
                      <TableBody>
                        {(
                          (columnsByTable[selectedTable.id] || [])
                            .filter(c => {
                              if (!columnFilter) return true
                              const q = columnFilter.toLowerCase()
                              return String(c.node_name || '').toLowerCase().includes(q) || String(parseColumnType(c.properties) || '').toLowerCase().includes(q) || JSON.stringify(c.properties || '').toLowerCase().includes(q)
                            })
                            .map(col => (
                          <TableRow key={col.id} data-testid={`column-${col.node_name}`}>
                            <TableCell sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                              {col.node_name}
                              {profiledColumns[col.node_name] ? (
                                <Tooltip title={`Profile exists for ${col.node_name}`} arrow>
                                  <CheckCircleOutlineIcon color="success" sx={{ fontSize: 18, ml: 1 }} data-testid={`profile-indicator-${col.node_name}`} aria-label={`profile-exists-${col.node_name}`} />
                                </Tooltip>
                              ) : null}
                            </TableCell>
                            <TableCell>
                              <Chip label={String(parseColumnType(col.properties) || '-')} size="small" data-testid={`type-${col.node_name}`} />
                            </TableCell>
                            <TableCell data-testid={`nullable-${col.node_name}`}>{formatNullable(col.properties)}</TableCell>
                            <TableCell>
                              {col.properties ? (
                                <Box component="pre" sx={{ margin: 0, whiteSpace: 'pre-wrap', fontSize: '0.85rem' }} data-testid={`props-${col.node_name}`}>{JSON.stringify(col.properties, null, 2)}</Box>
                              ) : '-'}
                            </TableCell>
                          </TableRow>
                        ))
                        )}
                      </TableBody>
                </Table>
                {/* Load more control for lazy pagination */}
                <Box sx={{ mt: 1, display: 'flex', justifyContent: 'center', alignItems: 'center', gap: 1 }}>
                  {selectedTable && columnsPageByTable[selectedTable.id]?.hasMore && (
                    <Button size="small" onClick={() => fetchColumns(selectedTable.id, { offset: columnsPageByTable[selectedTable.id]?.offset ?? 0, limit: columnsPageByTable[selectedTable.id]?.limit ?? 200 })} disabled={!!columnsLoadingByTable[selectedTable.id]} data-testid="load-more">
                      {columnsLoadingByTable[selectedTable.id] ? 'Loading...' : 'Load more'}
                    </Button>
                  )}
                </Box>
              </Box>
            </>
          )}
        </Paper>
      </Box>
    </Box>
  )
}

// Helpers to normalize/format column type and nullable info
function parseColumnType(properties: any) {
  if (!properties) return '-'
  // common property keys observed in catalog nodes
  if (properties.data_type) return String(properties.data_type)
  if (properties.type) return String(properties.type)
  if (properties.column_type) return String(properties.column_type)
  if (properties.udt_name) return String(properties.udt_name)
  // fallback: try to infer from properties object
  if (properties.format) return String(properties.format)
  if (typeof properties === 'string') return properties
  return '-'
}

function formatNullable(properties: any) {
  if (!properties) return '-'
  if (properties.is_nullable !== undefined) return String(properties.is_nullable)
  if (properties.nullable !== undefined) return String(properties.nullable)
  return '-'
}
