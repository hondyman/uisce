import React from 'react'
import { Autocomplete, TextField, CircularProgress, Box, Chip } from '@mui/material'
import './DomainTypeahead.css'

type Domain = {
  id: string
  name: string
  slug?: string
  level?: number
  parent_id?: string | null
  description?: string | null
  fullPath?: string // Add full path for display
}

function normalizeDomain(raw: any): Domain {
  const descriptionValue = typeof raw?.description === 'string'
    ? raw.description
    : raw?.description?.String ?? ''

  // attempt to pick a flattened path returned from the backend if present,
  // otherwise fall back to constructing one or using the name
  const fullPath = raw?.full_path ?? raw?.fullPath ?? raw?.path ?? (Array.isArray(raw?.hierarchy) ? raw.hierarchy.join(' / ') : raw?.name)

  return {
    id: raw?.id,
    name: raw?.name,
    slug: raw?.slug ?? undefined,
    level: raw?.level ?? undefined,
    parent_id: raw?.parent_id ?? null,
    description: descriptionValue || null,
    fullPath: fullPath || raw?.name,
  }
}

async function searchDomains(q: string, limit = 10) {
  const params = new URLSearchParams()
  params.set('q', q)
  params.set('limit', String(limit))
  const res = await fetch(`/api/data-domains/search?${params.toString()}`, { credentials: 'include' })
  if (!res.ok) throw new Error('search failed')
  const data = await res.json()
  if (!Array.isArray(data)) return []
  return data.map(normalizeDomain)
}

async function fetchDomainById(id: string) {
  const res = await fetch(`/api/data-domains/${id}`, { credentials: 'include' })
  if (!res.ok) throw new Error('fetch failed')
  const data = await res.json()
  return normalizeDomain(data)
}

interface Props {
  value?: string | null
  onChange: (value: string | null) => void
  label?: string
  placeholder?: string
  helperText?: string
  allowClear?: boolean
  onLoadingChange?: (loading: boolean) => void
}

export default function DomainTypeahead({ value, onChange, label = 'Domain', placeholder, helperText, allowClear = true, onLoadingChange }: Props) {
  const [open, setOpen] = React.useState(false)
  const [options, setOptions] = React.useState<Domain[]>([])
  const [loading, setLoading] = React.useState(false)
  const [inputValue, setInputValue] = React.useState('')
  const [selected, setSelected] = React.useState<Domain | null>(null)

  // Ensure the currently selected value is present in the options list so
  // Autocomplete can render the label even before search results arrive.
  React.useEffect(() => {
    if (!value) {
      setSelected(null)
      return
    }
    const existing = options.find((o) => o.id === value)
    if (existing) {
      setSelected(existing)
      return
    }

    let cancelled = false
    ;(async () => {
      try {
        const domain = await fetchDomainById(value)
        if (cancelled) return
        setSelected(domain)
        setOptions((prev) => {
          if (prev.some((o) => o.id === domain.id)) return prev
          return [domain, ...prev]
        })
      } catch {
        if (!cancelled) {
          setSelected(null)
        }
      }
    })()

    return () => {
      cancelled = true
    }
  }, [value, options])

  React.useEffect(() => {
    if (!open) return
    let active = true

    setLoading(true)
    // notify parent that a remote search started
    if (onLoadingChange) onLoadingChange(true)
    searchDomains(inputValue || '', 20)
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
  }, [open, inputValue, value])

  // keep parent informed of loading changes when they occur outside the above effect
  React.useEffect(() => {
    if (onLoadingChange) onLoadingChange(loading)
  }, [loading, onLoadingChange])

  const optionLabel = React.useCallback((domain: Domain) => {
    const levelLabel = domain.level ? `L${domain.level}` : 'L?'
    return `${domain.name} (${levelLabel})`
  }, [])

  // Function to highlight search text
  const highlightText = React.useCallback((text: string, highlight: string) => {
    if (!highlight.trim()) return text
    const regex = new RegExp(`(${highlight.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi')
    const parts = text.split(regex)
    return parts.map((part, index) =>
      regex.test(part) ? (
        <span key={index} className="domain-highlight">{part}</span>
      ) : (
        part
      )
    )
  }, [])

  return (
    <Autocomplete
      open={open}
      onOpen={() => setOpen(true)}
      onClose={() => setOpen(false)}
      options={options}
      value={selected}
      loading={loading}
      inputValue={inputValue}
      disableClearable={!allowClear}
      filterOptions={(x) => x}
      onInputChange={(_, newInput) => {
        setInputValue(newInput)
      }}
      isOptionEqualToValue={(o, v) => o.id === v.id}
      getOptionLabel={optionLabel}
      renderOption={(props, option) => {
        const levelColor = option.level === 1 ? '#BBDEFB' : option.level === 2 ? '#C8E6C9' : option.level === 3 ? '#FFF9C4' : '#E0E0E0'
        return (
          <li {...props} key={option.id} className="domain-option">
            <Box sx={{ display: 'flex', justifyContent: 'space-between', width: '100%', alignItems: 'center' }}>
              <Box sx={{ display: 'flex', flexDirection: 'column', mr: 2, minWidth: 0 }}>
                <span className="domain-option__title">
                  {highlightText(option.name, inputValue)}
                </span>
                <span className="domain-option__meta">
                  {highlightText(option.fullPath || option.name, inputValue)}
                </span>
              </Box>
              <Chip label={option.level ? `L${option.level}` : 'L?'} size="small" sx={{ backgroundColor: levelColor, color: '#000' }} />
            </Box>
          </li>
        )
      }}
      onChange={(_, newValue) => {
        if (!newValue) {
          setSelected(null)
          onChange(null)
        } else {
          setSelected(newValue)
          onChange(newValue.id)
        }
      }}
      renderInput={(params) => (
        <TextField
          {...params}
          label={label}
          placeholder={placeholder}
          helperText={helperText}
          size="small"
          InputProps={{
            ...params.InputProps,
            endAdornment: (
              <>
                {loading ? <CircularProgress color="inherit" size={16} /> : null}
                {params.InputProps.endAdornment}
              </>
            ),
          }}
        />
      )}
      noOptionsText={inputValue ? 'No matching domains' : 'Start typing to search domains'}
      loadingText="Searching domains..."
    />
  )
}
