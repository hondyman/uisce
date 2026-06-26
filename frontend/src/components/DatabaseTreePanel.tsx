import { useEffect, useState, useCallback, useMemo } from 'react'
import { devError } from '../utils/devLogger';
import { useScope } from '../contexts/ScopeContext'
import { Box, CircularProgress, Paper, Typography, List, ListItem, ListItemIcon, ListItemText, Collapse, IconButton, Alert, Checkbox, Tooltip } from '@mui/material'
import { Database, Link, Unlink } from 'lucide-react'
import FolderIcon from '@mui/icons-material/Folder'
import TableChartIcon from '@mui/icons-material/TableChart'
import ViewColumnIcon from '@mui/icons-material/ViewColumn'
import AccountTreeIcon from '@mui/icons-material/AccountTree'
import RefreshIcon from '@mui/icons-material/Refresh'
import ExpandLess from '@mui/icons-material/ExpandLess'
import ExpandMore from '@mui/icons-material/ExpandMore'
import { hasTenantScope } from '../utils/tenantScope'

type CatalogNode = {
  id: string
  node_name: string
  qualified_path?: string
  catalog_type?: string
  parent_id?: string
  properties?: any
}

interface DatabaseTreePanelProps {
  title?: string
  subtitle?: string
  showColumnSelection?: boolean
  showSchemaSelection?: boolean
  showTableSelection?: boolean
  showColumns?: boolean
  onSelectionChange?: (selectedSchemas: string[], selectedTables: string[], selectedColumns: string[]) => void
  initialSelectedSchemas?: string[]
  initialSelectedTables?: string[]
  initialSelectedColumns?: string[]
  height?: string
  // Filter props
  // Filter props
  mappedFilter?: Set<string>
  setMappedFilter?: (filter: Set<any>) => void
  mappingCounts?: { all: number; mapped: number; unmapped: number; pending: number }
  mappings?: any[]
}

export default function DatabaseTreePanel({
  title = "Database",
  subtitle = "Schemas, tables, and columns",
  showColumnSelection = false,
  showSchemaSelection: _showSchemaSelection = true,
  showTableSelection: _showTableSelection = true,
  showColumns = true,
  onSelectionChange,
  initialSelectedSchemas = [],
  initialSelectedTables = [],
  initialSelectedColumns = [],
  height = '72vh',
  mappedFilter,
  setMappedFilter,
  mappingCounts,
  mappings = []
}: DatabaseTreePanelProps) {
  const [schemas, setSchemas] = useState<CatalogNode[]>([])
  const [expandedSchemas, setExpandedSchemas] = useState<string[]>([])
  const [expandedTables, setExpandedTables] = useState<string[]>([])
  const [tablesBySchema, setTablesBySchema] = useState<Record<string, CatalogNode[]>>({})
  const [columnsByTable, setColumnsByTable] = useState<Record<string, CatalogNode[]>>({})
  const [selectedSchemas, setSelectedSchemas] = useState<string[]>(initialSelectedSchemas)
  const [selectedTables, setSelectedTables] = useState<string[]>(initialSelectedTables)
  const [selectedColumns, setSelectedColumns] = useState<string[]>(initialSelectedColumns)
  const [loading, setLoading] = useState(false)

  const canUseScope = hasTenantScope()
  const scope = useScope()

  // Calculate match counts per schema and table
  const matchCounts = useMemo(() => {
    const counts: Record<string, { schema: number; tables: Record<string, number> }> = {}
    
    mappings.forEach((mapping: any) => {
      if (mapping.ignored) return
      
      const schemaName = mapping.database_column?.schema || ''
      const tableName = mapping.database_column?.table || ''
      
      if (!counts[schemaName]) {
        counts[schemaName] = { schema: 0, tables: {} }
      }
      
      counts[schemaName].schema++
      
      if (!counts[schemaName].tables[tableName]) {
        counts[schemaName].tables[tableName] = 0
      }
      
      counts[schemaName].tables[tableName]++
    })
    
    return counts
  }, [mappings])

  const makeApiUrl = (path: string) => {
    try {
      if ((typeof process !== 'undefined' && process.env && process.env.NODE_ENV === 'test') || import.meta.env.MODE === 'test') return path
      const base = (typeof window !== 'undefined' && (window as any).location && (window as any).location.origin) ? (window as any).location.origin : 'http://localhost'
      return new URL(path, base).toString()
    } catch (e) {
      return path
    }
  }

  const fetchSchemas = useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetch(makeApiUrl('/api/catalog/nodes?type=schema&limit=500'), { credentials: 'include' })
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      setSchemas(Array.isArray(data) ? data : [])
    } catch (err: any) {
      devError('[DatabaseTreePanel] fetchSchemas error', err)
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
      devError('[DatabaseTreePanel] fetchTables error', err)
      setTablesBySchema(prev => ({ ...prev, [schemaId]: [] }))
    }
  }, [tablesBySchema])

  const fetchColumns = useCallback(async (tableId: string) => {
    if (columnsByTable[tableId]) return
    try {
      const res = await fetch(makeApiUrl(`/api/catalog/nodes?type=column&parent_id=${encodeURIComponent(tableId)}&limit=500`), { credentials: 'include' })
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      setColumnsByTable(prev => ({ ...prev, [tableId]: Array.isArray(data) ? data : [] }))
    } catch (err) {
      devError('[DatabaseTreePanel] fetchColumns error', err)
      setColumnsByTable(prev => ({ ...prev, [tableId]: [] }))
    }
  }, [columnsByTable])

  useEffect(() => {
    if (canUseScope) {
      fetchSchemas()
    }
  }, [canUseScope, fetchSchemas])

  // Update scope context when selections change
  useEffect(() => {
    if (selectedSchemas.length > 0 || selectedTables.length > 0 || selectedColumns.length > 0) {
      // Find schema IDs and names
      const schemaIds: string[] = []
      const schemaNames: string[] = []
      const tableIds: string[] = []
      const tableNames: string[] = []
      const columnIds: string[] = []
      const columnNames: string[] = []

      schemas.forEach(schema => {
        if (selectedSchemas.includes(schema.node_name)) {
          schemaIds.push(schema.id)
          schemaNames.push(schema.node_name)
        }
      })

      Object.entries(tablesBySchema).forEach(([, tables]) => {
        tables.forEach(table => {
          if (selectedTables.includes(table.node_name)) {
            tableIds.push(table.id)
            tableNames.push(table.node_name)
          }
        })
      })

      Object.entries(columnsByTable).forEach(([, columns]) => {
        columns.forEach(column => {
          if (selectedColumns.includes(column.node_name)) {
            columnIds.push(column.id)
            columnNames.push(column.node_name)
          }
        })
      })

      scope.setSchemaIds(schemaIds)
      scope.setSchemaNames(schemaNames)
      scope.setTableIds(tableIds)
      scope.setTableNames(tableNames)
      scope.setColumnIds(columnIds)
      scope.setColumnNames(columnNames)

      if (onSelectionChange) {
        onSelectionChange(selectedSchemas, selectedTables, selectedColumns)
      }
    }
  }, [selectedSchemas, selectedTables, selectedColumns, schemas, tablesBySchema, columnsByTable, scope, onSelectionChange])

  const toggleSchema = async (schemaName: string) => {
    const schema = schemas.find(s => s.node_name === schemaName)
    if (!schema) return

    if (!tablesBySchema[schema.id]) {
      await fetchTables(schema.id)
    }

    setExpandedSchemas(prev =>
      prev.includes(schemaName) ? prev.filter(name => name !== schemaName) : [...prev, schemaName]
    )
  }

  const toggleTable = async (tableId: string) => {
    if (!columnsByTable[tableId]) {
      await fetchColumns(tableId)
    }

    setExpandedTables(prev =>
      prev.includes(tableId) ? prev.filter(id => id !== tableId) : [...prev, tableId]
    )
  }

  const toggleSchemaSelection = (schemaName: string) => {
    // When clicking on a schema, set it as the selected schema (single selection)
    setSelectedSchemas([schemaName])
    setSelectedTables([]) // Clear table selection when selecting a schema
    setSelectedColumns([])
  }

  const toggleTableSelection = (tableName: string) => {
    // When clicking on a table, set it as the selected table (single selection)
    setSelectedTables([tableName])
    setSelectedSchemas([]) // Clear schema selection when selecting a table
    setSelectedColumns([])
  }

  const toggleColumnSelection = (columnName: string) => {
    setSelectedColumns(prev =>
      prev.includes(columnName) ? prev.filter(name => name !== columnName) : [...prev, columnName]
    )
  }

  if (!canUseScope) {
    return (
      <Box sx={{ width: '360px', flexShrink: 0 }}>
        <Alert severity="warning">Select a tenant and datasource to enable database browsing</Alert>
      </Box>
    )
  }

  return (
    <Box sx={{ width: '360px', flexShrink: 0 }}>
      <Paper sx={{ p: 2, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
        <AccountTreeIcon />
        <Box>
          <Typography variant="h6">{title}</Typography>
          <Typography variant="body2" color="text.secondary">{subtitle}</Typography>
        </Box>
        <Box sx={{ flex: 1 }} />
        <IconButton size="small" onClick={() => { setTablesBySchema({}); setColumnsByTable({}); fetchSchemas() }}>
          <RefreshIcon />
        </IconButton>
      </Paper>



      <Paper sx={{ maxHeight: height, overflow: 'auto', p: 1 }}>
        {loading ? (
          <Box sx={{ p: 3, textAlign: 'center' }}><CircularProgress size={28} /></Box>
        ) : schemas.length === 0 ? (
          <Box sx={{ p: 2 }}><Typography>No schemas found</Typography></Box>
        ) : (
          <List>
            {schemas.map(schema => {
              const isExpanded = expandedSchemas.includes(schema.node_name)
              const tables = tablesBySchema[schema.id] || []
              return (
                <div key={schema.id}>
                  <ListItem
                    secondaryAction={
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Typography variant="caption" color="text.secondary">
                          {matchCounts[schema.node_name]?.schema || 0} matches
                        </Typography>
                        {isExpanded ? <ExpandLess /> : <ExpandMore />}
                      </Box>
                    }
                    button
                    onClick={() => {
                      toggleSchema(schema.node_name)
                      toggleSchemaSelection(schema.node_name)
                    }}
                    data-testid={`schema-${schema.node_name}`}
                    sx={{
                      bgcolor: selectedSchemas.includes(schema.node_name) ? 'action.selected' : 'transparent',
                      '&:hover': { bgcolor: 'action.hover' }
                    }}
                  >
                    <ListItemIcon><FolderIcon fontSize="small" /></ListItemIcon>
                    <ListItemText primary={schema.node_name} />
                  </ListItem>

                  <Collapse in={isExpanded} timeout="auto" unmountOnExit>
                    <List component="div" disablePadding>
                      {tables.map(table => {
                        const isTableExpanded = expandedTables.includes(table.id)
                        const columns = columnsByTable[table.id] || []
                        return (
                          <div key={table.id}>
                            <ListItem
                              button
                              sx={{ pl: 4, bgcolor: selectedTables.includes(table.node_name) ? 'action.selected' : 'transparent', '&:hover': { bgcolor: 'action.hover' } }}
                              secondaryAction={
                                showColumns && columns.length > 0 ? (
                                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                                    <Typography variant="caption" color="text.secondary">
                                      {matchCounts[schemas.find(s => s.id === table.parent_id)?.node_name || '']?.tables[table.node_name] || 0} matches
                                    </Typography>
                                    {isTableExpanded ? <ExpandLess /> : <ExpandMore />}
                                  </Box>
                                ) : (
                                  <Typography variant="caption" color="text.secondary">
                                    {matchCounts[schemas.find(s => s.id === table.parent_id)?.node_name || '']?.tables[table.node_name] || 0} matches
                                  </Typography>
                                )
                              }
                              onClick={() => showColumns ? toggleTable(table.id) : toggleTableSelection(table.node_name)}
                              data-testid={`table-${table.node_name}`}
                            >
                              <ListItemIcon><TableChartIcon fontSize="small" /></ListItemIcon>
                              <ListItemText primary={table.node_name} />
                            </ListItem>

                            {showColumns && (
                              <Collapse in={isTableExpanded} timeout="auto" unmountOnExit>
                                <List component="div" disablePadding>
                                  {columns.map(column => (
                                    <ListItem key={column.id} sx={{ pl: 6 }} data-testid={`column-${column.node_name}`}>
                                      {showColumnSelection && (
                                        <ListItemIcon>
                                          <Checkbox
                                            edge="start"
                                            size="small"
                                            checked={selectedColumns.includes(column.node_name)}
                                            onClick={(e) => { e.stopPropagation(); toggleColumnSelection(column.node_name); }}
                                            inputProps={{ 'aria-label': `select-column-${column.node_name}` }}
                                          />
                                        </ListItemIcon>
                                      )}
                                      <ListItemIcon><ViewColumnIcon fontSize="small" /></ListItemIcon>
                                      <ListItemText primary={column.node_name} />
                                    </ListItem>
                                  ))}
                                </List>
                              </Collapse>
                            )}
                          </div>
                        )
                      })}
                    </List>
                  </Collapse>
                </div>
              )
            })}
          </List>
        )}
      </Paper>
    </Box>
  )
}