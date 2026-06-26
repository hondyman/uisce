import React, { createContext, useContext, useEffect, useState } from 'react'
import { loadNames, saveNames } from '../utils/scopeStorage'

type ScopeContextValue = {
  schemaIds: string[]
  setSchemaIds: (ids: string[]) => void
  schemaNames: string[]
  setSchemaNames: (names: string[]) => void
  tableIds: string[]
  setTableIds: (ids: string[]) => void
  tableNames: string[]
  setTableNames: (names: string[]) => void
  columnIds: string[]
  setColumnIds: (ids: string[]) => void
  columnNames: string[]
  setColumnNames: (names: string[]) => void
  // backward-compatible singular setters used by some tests
  setSchemaName: (name: string) => void
  setTableName: (name: string) => void
  setSchemaId: (id: string) => void
  setTableId: (id: string) => void
}

const ScopeContext = createContext<ScopeContextValue | null>(null)

export const useScope = () => {
  const ctx = useContext(ScopeContext)
  if (!ctx) throw new Error('useScope must be used within ScopeProvider')
  return ctx
}

export const ScopeProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [schemaIds, setSchemaIds] = useState<string[]>([])
  const [schemaNames, setSchemaNames] = useState<string[]>([])
  const [tableIds, setTableIds] = useState<string[]>([])
  const [tableNames, setTableNames] = useState<string[]>([])
  const [columnIds, setColumnIds] = useState<string[]>([])
  const [columnNames, setColumnNames] = useState<string[]>([])

  // initialize names from localStorage
  useEffect(() => {
    const names = loadNames()
    if (names.schema) setSchemaNames([names.schema])
    if (names.table) setTableNames([names.table])
    if (names.columns) setColumnNames(names.columns)
  }, [])

  // persist names when they change
  useEffect(() => {
    saveNames({ 
      schema: schemaNames.length > 0 ? schemaNames[0] : undefined, 
      table: tableNames.length > 0 ? tableNames[0] : undefined, 
      columns: columnNames 
    })
  }, [schemaNames, tableNames, columnNames])

  // keep ids in the URL only (privacy)
  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    if (schemaIds && schemaIds.length > 0) params.set('schema_ids', schemaIds.join(','))
    else params.delete('schema_ids')
    if (tableIds && tableIds.length > 0) params.set('table_ids', tableIds.join(','))
    else params.delete('table_ids')
    if (columnIds && columnIds.length > 0) params.set('column_ids', columnIds.join(','))
    else params.delete('column_ids')
    const newUrl = `${window.location.pathname}?${params.toString()}`
    window.history.replaceState({}, '', newUrl)
  }, [schemaIds, tableIds, columnIds])

  return (
    <ScopeContext.Provider value={{ 
      schemaIds, 
      setSchemaIds, 
      schemaNames, 
      setSchemaNames, 
      tableIds, 
      setTableIds, 
      tableNames, 
      setTableNames, 
      columnIds, 
      setColumnIds, 
      columnNames,
      setColumnNames,
      // backward-compatible singular setters used by a few tests
      setSchemaName: (n: string) => setSchemaNames(n ? [n] : []),
      setTableName: (n: string) => setTableNames(n ? [n] : []),
      setSchemaId: (id: string) => setSchemaIds(id ? [id] : []),
      setTableId: (id: string) => setTableIds(id ? [id] : [])
    }}>
      {children}
    </ScopeContext.Provider>
  )
}

export default ScopeContext
