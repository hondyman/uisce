import React from 'react'
import { Autocomplete, TextField, CircularProgress } from '@mui/material'

type Bundle = {
  id: string
  name: string
  description?: string | null
}

function normalizeBundle(raw: any): Bundle {
  return {
    id: raw?.id,
    name: raw?.name || raw?.display_name || raw?.title || String(raw?.id),
    description: raw?.description ?? raw?.summary ?? null,
  }
}

async function listAllBundles() {
  const res = await fetch('/api/bundles', { credentials: 'include' })
  if (!res.ok) throw new Error('failed to load bundles')
  const data = await res.json()
  if (!Array.isArray(data)) return []
  return data.map(normalizeBundle)
}

async function fetchBundleById(id: string) {
  const res = await fetch(`/api/bundles/${encodeURIComponent(id)}`, { credentials: 'include' })
  if (!res.ok) throw new Error('fetch failed')
  const data = await res.json()
  return normalizeBundle(data)
}

interface Props {
  value?: string[] | null
  onChange: (value: string[] | null) => void
  label?: string
  placeholder?: string
  helperText?: string
  allowClear?: boolean
  onLoadingChange?: (loading: boolean) => void
}

export default function BundleTypeahead({ value, onChange, label = 'Bundle', placeholder, helperText, allowClear = true, onLoadingChange }: Props) {
  const [open, setOpen] = React.useState(false)
  const [options, setOptions] = React.useState<Bundle[]>([])
  const [loading, setLoading] = React.useState(false)
  const [inputValue, setInputValue] = React.useState('')
  const [selected, setSelected] = React.useState<Bundle[]>([])

  React.useEffect(() => {
    let cancelled = false
    setLoading(true)
    if (onLoadingChange) onLoadingChange(true)
    listAllBundles()
      .then((rows) => {
        if (cancelled) return
        setOptions(rows)
        // hydrate selected values if provided
        if (value && value.length) {
          const selectedRows = rows.filter((r) => value.includes(r.id))
          setSelected(selectedRows)
        }
      })
      .catch(() => {})
      .finally(() => {
        if (!cancelled) setLoading(false)
        if (onLoadingChange) onLoadingChange(false)
      })

    return () => { cancelled = true }
  }, [])

  // when external value changes, ensure selected reflects it
  React.useEffect(() => {
    if (!value || !value.length) {
      setSelected([])
      return
    }
    // Try to find in options; if missing, fetch individually
    const found = options.filter((o) => value.includes(o.id))
    const missing = value.filter((id) => !found.some((f) => f.id === id))
    if (missing.length === 0) {
      setSelected(found)
      return
    }
    let cancelled = false
    Promise.all(missing.map((id) => fetchBundleById(id).catch(() => null)))
      .then((more) => {
        if (cancelled) return
        const combined = [...found, ...more.filter(Boolean) as Bundle[]]
        setSelected(combined)
      })
    return () => { cancelled = true }
  }, [value, options])

  const filterOptions = (items: Bundle[], state: any) => {
    const q = (state.inputValue || '').toLowerCase().trim()
    if (!q) return items
    return items.filter((b) => (b.name || '').toLowerCase().includes(q) || (b.description || '').toLowerCase().includes(q))
  }

  return (
    <Autocomplete
      multiple
      open={open}
      onOpen={() => setOpen(true)}
      onClose={() => setOpen(false)}
      options={options}
      value={selected}
      loading={loading}
      inputValue={inputValue}
      disableClearable={!allowClear}
      filterOptions={filterOptions}
      onInputChange={(_, newInput) => setInputValue(newInput)}
      isOptionEqualToValue={(o, v) => o.id === v.id}
      getOptionLabel={(b) => b.name}
      onChange={(_, newValue) => {
        setSelected(newValue as Bundle[])
        onChange((newValue as Bundle[]).map((n) => n.id))
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
      noOptionsText={inputValue ? 'No matching bundles' : 'Start typing to search bundles'}
      loadingText="Searching bundles..."
    />
  )
}
