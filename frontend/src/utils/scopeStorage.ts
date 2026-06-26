export type StoredNames = {
  schema?: string
  table?: string
  columns?: string[]
}

const KEYS = {
  schema: 'sem_scope.schema',
  table: 'sem_scope.table',
  columns: 'sem_scope.columns',
  // legacy keys to clean up
  legacySchemaId: 'sem_mapper.scopeSchemaId',
  legacySchema: 'sem_mapper.scopeSchema',
  legacyTableId: 'sem_mapper.scopeTableId',
  legacyTable: 'sem_mapper.scopeTable',
  legacyColumnsIds: 'sem_mapper.scopeColumnsIds',
  legacyColumns: 'sem_mapper.scopeColumns',
}

export function loadNames(): StoredNames {
  try {
    // migrate/cleanup legacy keys if present
    try {
      if (localStorage.getItem(KEYS.legacySchema) !== null) {
        localStorage.removeItem(KEYS.legacySchemaId)
        localStorage.removeItem(KEYS.legacySchema)
        localStorage.removeItem(KEYS.legacyTableId)
        localStorage.removeItem(KEYS.legacyTable)
        localStorage.removeItem(KEYS.legacyColumnsIds)
        localStorage.removeItem(KEYS.legacyColumns)
      }
    } catch {}

    const schema = localStorage.getItem(KEYS.schema) || undefined
    const table = localStorage.getItem(KEYS.table) || undefined
    const colsRaw = localStorage.getItem(KEYS.columns) || undefined
    const columns = colsRaw ? colsRaw.split(',').map(s => s.trim()).filter(Boolean) : undefined
    return { schema, table, columns }
  } catch (err) {
    return {}
  }
}

export function saveNames(names: StoredNames) {
  try {
    if (names.schema !== undefined) localStorage.setItem(KEYS.schema, names.schema || '')
    if (names.table !== undefined) localStorage.setItem(KEYS.table, names.table || '')
    if (names.columns !== undefined) localStorage.setItem(KEYS.columns, (names.columns || []).join(','))
  } catch (err) {
    // ignore
  }
}

export function clearNames() {
  try {
    localStorage.removeItem(KEYS.schema)
    localStorage.removeItem(KEYS.table)
    localStorage.removeItem(KEYS.columns)
  } catch {}
}

export default { loadNames, saveNames, clearNames }
