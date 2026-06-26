import { IconButton, Tooltip } from '@mui/material'
import { X } from 'lucide-react'
import { useScope } from '../contexts/ScopeContext'
import { clearNames } from '../utils/scopeStorage'

export default function ScopeBadge() {
  const {
    schemaIds, setSchemaIds, 
    tableIds, setTableIds, 
    columnIds, setColumnIds, 
  } = useScope()

  const hasAny = !!(schemaIds.length > 0 || tableIds.length > 0 || (columnIds && columnIds.length > 0))

  const handleClear = () => {
    setSchemaIds([])
    setTableIds([])
    setColumnIds([])
    try { clearNames() } catch (e) {}
  }

  return (
    <div className="scope-badge" role="region" aria-label="Current scope">
      {hasAny && (
        <IconButton size="small" onClick={handleClear} aria-label="Clear scope">
          <X width={14} height={14} />
        </IconButton>
      )}
    </div>
  )
}
