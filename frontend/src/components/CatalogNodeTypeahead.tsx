import React from 'react'
import { devError } from '../utils/devLogger';
import { Autocomplete, TextField, CircularProgress, Box } from '@mui/material'
import './CatalogNodeTypeahead.css'

type CatalogNode = {
  id: string
  node_name: string
  qualified_path: string
  catalog_type: string
  parent_id?: string
  properties?: Record<string, unknown>
  created_at: string
}

async function searchCatalogNodes(nodeType: string, q: string, limit = 10, parentId?: string) {
  const params = new URLSearchParams()
  params.set('q', q)
  params.set('limit', String(limit))
  params.set('type', nodeType)
  if (parentId) params.set('parent_id', parentId)
  
  const url = `/api/catalog/nodes?${params.toString()}`
  // Searching for catalog nodes
  
  try {
    const res = await fetch(url, { credentials: 'include' })
    if (!res.ok) {
      const text = await res.text()
      devError(`[CatalogNodeTypeahead] Search failed: ${res.status} ${res.statusText}`, text)
      throw new Error(`search failed: ${res.status} ${text.substring(0, 100)}`)
    }
    const data = await res.json()
    // Got search results
    if (!Array.isArray(data)) return []
    return data.map(normalizeCatalogNode)
  } catch (error) {
    devError('[CatalogNodeTypeahead] Search error:', error)
    throw error
  }
}

async function fetchCatalogNodeById(id: string, parentId?: string) {
  // Search by id via q param and parent filter if provided
  const nodes = await searchCatalogNodes('', id, 5, parentId)
  return nodes.find(node => node.id === id) || null
}

function normalizeCatalogNode(raw: unknown): CatalogNode {
  const r = (raw as Record<string, unknown>) || {}
  return {
    id: typeof r.id === 'string' ? (r.id as string) : '',
    node_name: typeof r.node_name === 'string' ? (r.node_name as string) : '',
    qualified_path: typeof r.qualified_path === 'string' ? (r.qualified_path as string) : '',
    catalog_type: typeof r.catalog_type === 'string' ? (r.catalog_type as string) : '',
    parent_id: typeof r.parent_id === 'string' ? (r.parent_id as string) : undefined,
    properties: (r.properties as Record<string, unknown>) || undefined,
    created_at: typeof r.created_at === 'string' ? (r.created_at as string) : '',
  }
}

interface Props {
  nodeType: string
  value?: string | string[] | null
  onChange: (value: string | string[] | null) => void
  onSelect?: (node: CatalogNode | CatalogNode[] | null) => void
  parentId?: string
  multiple?: boolean
  label?: string
  placeholder?: string
  helperText?: string
  onLoadingChange?: (loading: boolean) => void
}

export default function CatalogNodeTypeahead({ nodeType, value, onChange, onSelect, parentId, multiple, label, placeholder, helperText, onLoadingChange }: Props) {
  const [open, setOpen] = React.useState(false)
  const [options, setOptions] = React.useState<CatalogNode[]>([])
  const [loading, setLoading] = React.useState(false)
  const [inputValue, setInputValue] = React.useState('')
  const [selected, setSelected] = React.useState<CatalogNode | CatalogNode[] | null>(() => (multiple ? [] : null))

  // Clear options when parentId changes to avoid showing stale results
  React.useEffect(() => {
    setOptions([])
    setSelected(multiple ? [] : null)
  }, [parentId, multiple])

  // Ensure the currently selected value is present in the options list so
  // Autocomplete can render the label even before search results arrive.
  React.useEffect(() => {
    if (!value) {
      setSelected(null)
      return
    }
    const isMultiple = Array.isArray(value)
    // If caller passed an empty array for a multiple typeahead, ensure selected is []
    if (isMultiple && (value as string[]).length === 0) {
      setSelected([])
      return
    }
    if (!isMultiple) {
      const existing = options.find((o) => o.id === value)
      if (existing) {
        setSelected(existing)
        return
      }
    }

    let cancelled = false
    ;(async () => {
      try {
        if (isMultiple) {
          const fetched: CatalogNode[] = []
          for (const id of value as string[]) {
            const node = await fetchCatalogNodeById(id, parentId)
            if (node) fetched.push(node)
          }
          if (cancelled) return
          if (fetched.length > 0) {
            setSelected(fetched)
            setOptions((prev) => {
              const merged = [...fetched, ...prev]
              const seen = new Set<string>()
              return merged.filter((n) => (seen.has(n.id) ? false : seen.add(n.id)))
            })
          }
        } else {
          const node = await fetchCatalogNodeById(value as string, parentId)
          if (cancelled) return
          if (node) {
            setSelected(node)
            setOptions((prev) => {
              if (prev.some((o) => o.id === node.id)) return prev
              return [node, ...prev]
            })
          }
        }
      } catch {
        if (!cancelled) {
          setSelected(null)
        }
      }
    })()

    return () => {
      cancelled = true
    }
  }, [value, options, parentId])

  React.useEffect(() => {
    if (!open) return
    let active = true

    setLoading(true)
    // notify parent that a remote search started
    if (onLoadingChange) onLoadingChange(true)
    searchCatalogNodes(nodeType, inputValue || '', 20, parentId)
      .then((rows) => {
        if (!active) return
        setOptions((prev) => {
          if (!value) return rows
          const selectedInRows = rows.some((row) => row.id === value)
          if (selectedInRows) return rows
          const selectedOption = prev.find((row) => row.id === value)
          return selectedOption ? [selectedOption, ...rows] : rows
        })
      })
      .catch(() => {
        if (active) setOptions((prev) => prev)
      })
      .finally(() => {
        if (active) setLoading(false)
        if (onLoadingChange) onLoadingChange(false)
      })

    return () => {
      active = false
    }
  }, [open, inputValue, value, nodeType, parentId])

  // keep parent informed of loading changes when they occur outside the above effect
  React.useEffect(() => {
    if (onLoadingChange) onLoadingChange(loading)
  }, [loading, onLoadingChange])

  const optionLabel = React.useCallback((node: CatalogNode) => {
    return node.node_name || node.qualified_path
  }, [])

  // Function to highlight search text
  const highlightText = React.useCallback((text: string, highlight: string) => {
    if (!highlight.trim()) return text
    const regex = new RegExp(`(${highlight.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi')
    const parts = text.split(regex)
    return (
      <span>
        {parts.map((part, index) =>
          regex.test(part) ? (
            <Box component="span" key={`${part}-${index}`} sx={{ fontWeight: 'bold' }}>
              {part}
            </Box>
          ) : (
            <span key={`${part}-${index}`}>{part}</span>
          )
        )}
      </span>
    )
  }, [])

  return (
  <Autocomplete<CatalogNode, boolean, boolean, false>
      id={`catalog-node-${nodeType}-typeahead`}
      open={open}
      onOpen={() => setOpen(true)}
      onClose={() => setOpen(false)}
      options={options}
      value={selected}
      multiple={!!multiple}
      onChange={(_, newValue: string | CatalogNode | (string | CatalogNode)[] | null) => {
        // newValue can be a CatalogNode, an array of CatalogNode/string in multiple mode,
        // or a string if freeSolo is enabled elsewhere. We guard and handle CatalogNode shapes.
        if (typeof newValue === 'string') {
          // unexpected string value - clear selection and forward as-is
          setSelected(null)
          onChange(newValue || null)
          if (onSelect) onSelect(null)
          return
        }

        setSelected(newValue as CatalogNode | CatalogNode[] | null)
        if (Array.isArray(newValue)) {
          const arr = newValue.filter((v): v is CatalogNode => typeof v !== 'string')
          onChange(arr.map((n) => n.id))
          if (onSelect) onSelect(arr)
        } else {
          const node = newValue && typeof newValue === 'object' ? (newValue as CatalogNode) : null
          onChange(node ? node.id : null)
          if (onSelect) onSelect(node)
        }
      }}
      onInputChange={(_, newInputValue) => {
        setInputValue(newInputValue)
      }}
  getOptionLabel={(option) => (typeof option === 'string' ? option : optionLabel(option))}
      isOptionEqualToValue={(option, value) => {
        // value may be a CatalogNode or an array (multiple mode) or null
        if (!value) return false
        if (Array.isArray(value)) return false
        return option.id === value.id
      }}
      loading={loading}
      clearOnEscape
      selectOnFocus
      handleHomeEndKeys
      renderInput={(params) => (
        <TextField
          {...params}
          label={label}
          placeholder={placeholder}
          helperText={helperText}
          InputProps={{
            ...params.InputProps,
            endAdornment: (
              <>
                {loading ? <CircularProgress color="inherit" size={20} /> : null}
                {params.InputProps.endAdornment}
              </>
            ),
          }}
        />
      )}
      renderOption={(props, option) => {
        const { key, ...optionProps } = props
        return (
          <li key={key} {...optionProps}>
            <Box sx={{ display: 'flex', flexDirection: 'column' }}>
              <Box sx={{ fontWeight: 'medium' }}>
                {highlightText(option.node_name, inputValue)}
              </Box>
              {option.qualified_path && option.qualified_path !== option.node_name && (
                <Box sx={{ fontSize: '0.875rem', color: 'text.secondary' }}>
                  {highlightText(option.qualified_path, inputValue)}
                </Box>
              )}
            </Box>
          </li>
        )
      }}
    />
  )
}