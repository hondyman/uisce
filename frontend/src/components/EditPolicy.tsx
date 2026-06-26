import { useEffect, useMemo, useState } from 'react'
import { fetchPolicy, savePolicy, deletePolicy, simulatePolicy } from '../services/policyService'
import {
  Autocomplete,
  TextField,
  Chip,
  Stack,
  Button,
  Alert,
  CircularProgress,
  Divider,
  FormControlLabel,
  Switch,
  Typography,
} from '@mui/material'
import BundleTypeahead from './BundleTypeahead'
import type { AccessControlPolicy } from '../types'
import './EditPolicy.css'
import { useNotification } from '../hooks/useNotification'
import { useConfirm } from '../components/ConfirmProvider'

const AVAILABLE_ROLES = ['viewer', 'editor', 'owner', 'admin']
const AVAILABLE_PERMISSIONS = ['read', 'write', 'download', 'delete', 'admin']

type PolicyFormState = {
  id?: string
  policy_id: string
  scope: string[]
  role: string
  permissions: string[]
  duration_days: number
  requires_certification: boolean
  max_claims_per_user: number
  approval_threshold: number
  renewal_conditions: any
  created_at?: string
  updated_at?: string
}

function defaultPolicy(): PolicyFormState {
  return {
    policy_id: '',
    scope: [],
    role: '',
    permissions: [],
    duration_days: 30,
    requires_certification: false,
    max_claims_per_user: 5,
    approval_threshold: 1,
    renewal_conditions: {
      usage_within_days: 30,
      review_required: false,
    },
  }
}

function toFormState(policy: AccessControlPolicy | null | undefined): PolicyFormState {
  if (!policy) {
    return defaultPolicy()
  }

  let renewal = policy.renewal_conditions as any
  if (typeof renewal === 'string') {
    try {
      renewal = JSON.parse(renewal)
    } catch {
      renewal = {}
    }
  }

  return {
    id: policy.id,
    policy_id: policy.policy_id,
    // If scope is stored as 'domain:<id>' we keep only the id in the form state
    // policy.scope may be 'bundle:<id>' or a comma-separated list. Normalize to array of bundle ids
    scope: (() => {
      if (!policy.scope) return []
      const parts = policy.scope.split(',').map((s) => s.trim()).filter(Boolean)
      return parts.map((s) => s.startsWith('bundle:') ? s.split(':')[1] : (s.startsWith('domain:') ? s.split(':')[1] : s))
    })(),
    role: policy.role,
    permissions: policy.permissions || [],
    duration_days: policy.duration_days,
    requires_certification: policy.requires_certification,
    max_claims_per_user: policy.max_claims_per_user,
    approval_threshold: policy.approval_threshold,
    renewal_conditions: renewal || {},
    created_at: policy.created_at,
    updated_at: policy.updated_at,
  }
}

function buildPayload(state: PolicyFormState): Partial<AccessControlPolicy> {
  return {
    id: state.id,
    policy_id: state.policy_id.trim(),
    // convert selected bundles/domains into canonical scope string(s)
    scope: (() => {
      if (!state.scope || state.scope.length === 0) return ''
      return state.scope.map((s) => `bundle:${String(s).trim()}`).join(',')
    })(),
    role: state.role.trim(),
    permissions: state.permissions || [],
    duration_days: Number(state.duration_days ?? 0),
    requires_certification: Boolean(state.requires_certification),
    max_claims_per_user: Number(state.max_claims_per_user ?? 0),
    approval_threshold: Number(state.approval_threshold ?? 0),
    renewal_conditions: state.renewal_conditions || {},
  }
}

interface EditPolicyProps {
  id?: string
  onSaved?: (policy: AccessControlPolicy) => void
  onCancel?: () => void
  onSimulate?: (policy: AccessControlPolicy) => Promise<void> | void
}

export default function EditPolicy({ id, onSaved, onCancel, onSimulate }: EditPolicyProps) {
  const [policy, setPolicy] = useState<PolicyFormState>(() => defaultPolicy())
  const [renewalText, setRenewalText] = useState<string>(() => JSON.stringify(defaultPolicy().renewal_conditions, null, 2))
  const [jsonError, setJsonError] = useState<string | null>(null)
  const [loading, setLoading] = useState<boolean>(false)
  const [saving, setSaving] = useState<boolean>(false)
  const [simulating, setSimulating] = useState<boolean>(false)
  const [error, setError] = useState<string | null>(null)
  const notification = useNotification()
  const confirm = useConfirm()
  const isEdit = useMemo(() => Boolean(id), [id])

  useEffect(() => {
    let cancelled = false

    async function load() {
      if (!id) {
        const blank = defaultPolicy()
        if (!cancelled) {
          setPolicy(blank)
          setRenewalText(JSON.stringify(blank.renewal_conditions, null, 2))
          setJsonError(null)
          setError(null)
        }
        return
      }

      try {
        setLoading(true)
        setError(null)
        const fetched = await fetchPolicy(id)
        if (cancelled) return
        const next = toFormState(fetched)
        setPolicy(next)
        setRenewalText(JSON.stringify(next.renewal_conditions ?? {}, null, 2))
        setJsonError(null)
      } catch (err: any) {
        if (cancelled) return
        setError(err?.message || 'Failed to load policy')
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    load()
    return () => {
      cancelled = true
    }
  }, [id])

  function updateField(field: keyof PolicyFormState, value: any) {
    setPolicy((prev) => ({ ...prev, [field]: value }))
  }

  function handleRenewalChange(value: string) {
    setRenewalText(value)
    if (!value.trim()) {
      updateField('renewal_conditions', {})
      setJsonError(null)
      return
    }
    try {
      const parsed = JSON.parse(value)
      updateField('renewal_conditions', parsed)
      setJsonError(null)
    } catch (err) {
      setJsonError('Renewal conditions must be valid JSON')
    }
  }

  function validate(state: PolicyFormState): string | null {
    if (!state.policy_id.trim()) return 'Policy ID is required'
      if (!state.scope || state.scope.length === 0) return 'Scope is required'
    if (!state.role.trim()) return 'Role is required'
    if (!state.permissions || state.permissions.length === 0) return 'Select at least one permission'
    if (jsonError) return jsonError
    if (Number(state.duration_days) < 0) return 'Duration must be >= 0'
    if (Number(state.max_claims_per_user) < 0) return 'Max claims per user must be >= 0'
    if (Number(state.approval_threshold) < 0) return 'Approval threshold must be >= 0'
    return null
  }

  async function handleSave() {
    const validation = validate(policy)
    if (validation) {
      setError(validation)
      return
    }

    try {
      setSaving(true)
      setError(null)
      const payload = buildPayload(policy)
      const response = await savePolicy(payload)
      const saved = (response?.policy || response) as AccessControlPolicy
      setPolicy(toFormState(saved))
      setRenewalText(JSON.stringify(saved.renewal_conditions ?? {}, null, 2))
      setJsonError(null)
      if (onSaved) onSaved(saved)
    } catch (err: any) {
      setError(err?.message || 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  async function handleSimulate() {
    const validation = validate(policy)
    if (validation) {
      setError(validation)
      return
    }
    const payload = buildPayload(policy) as AccessControlPolicy
    try {
      setSimulating(true)
      setError(null)
      if (onSimulate) {
        await onSimulate(payload)
      } else {
        const result = await simulatePolicy(payload)
        notification.info('Simulation result: ' + JSON.stringify(result, null, 2))
      }
    } catch (err: any) {
      setError(err?.message || 'Simulation failed')
    } finally {
      setSimulating(false)
    }
  }

  async function handleDelete() {
    if (!policy.id) return
    if (!(await confirm({ title: 'Delete policy', description: 'Delete this policy? This action cannot be undone.' }))) return
    try {
      setSaving(true)
      await deletePolicy(policy.id)
      if (onSaved) {
        const payload = buildPayload(policy)
        onSaved(payload as AccessControlPolicy)
      }
      if (onCancel) onCancel()
    } catch (err: any) {
      setError(err?.message || 'Delete failed')
    } finally {
      setSaving(false)
    }
  }

  const headerTitle = isEdit ? 'Edit Policy' : 'Create Policy'

  return (
    <div className="edit-policy">
      <Stack spacing={2} divider={<Divider />}
        sx={{ opacity: loading ? 0.6 : 1, pointerEvents: loading ? 'none' : 'auto' }}>
        <Stack direction="row" justifyContent="space-between" alignItems="center">
          <Typography variant="h6">{headerTitle}</Typography>
          {loading && <CircularProgress size={20} />}
        </Stack>

        {error && <Alert severity="error">{error}</Alert>}
        {jsonError && !error && <Alert severity="warning">{jsonError}</Alert>}

        <Stack spacing={2}>
          <TextField
            label="Policy ID"
            size="small"
            value={policy.policy_id}
            onChange={(e) => updateField('policy_id', e.target.value)}
            required
          />
          <BundleTypeahead
            value={policy.scope || null}
            onChange={(v) => updateField('scope', v ?? [])}
            label="Assign to bundles"
            helperText="Select one or more data bundles to which this policy applies"
            allowClear={true}
          />
          <Autocomplete
            options={AVAILABLE_ROLES}
            value={policy.role}
            onChange={(_, value) => updateField('role', value ?? '')}
            freeSolo
            renderInput={(params) => <TextField {...params} label="Role" size="small" required />}
          />

          <div className="ep-row ep-row-col">
            <Typography variant="subtitle2">Permissions</Typography>
            <Autocomplete
              multiple
              freeSolo
              options={AVAILABLE_PERMISSIONS}
              value={policy.permissions || []}
              onChange={(_, value) => updateField('permissions', value)}
              renderTags={(value: readonly string[], getTagProps) =>
                value.map((option: string, index: number) => {
                  const tagProps = getTagProps({ index } as any) as any
                  const { key, ...rest } = tagProps || {}
                  return <Chip key={key ?? `${option}-${index}`} size="small" variant="outlined" label={option} {...rest} />
                })
              }
              renderInput={(params) => <TextField {...params} placeholder="Search or add permissions" size="small" />}
            />
          </div>

          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
            <TextField
              label="Duration (days)"
              type="number"
              size="small"
              value={policy.duration_days}
              onChange={(e) => updateField('duration_days', Number(e.target.value))}
              inputProps={{ min: 0 }}
            />
            <TextField
              label="Max Claims per User"
              type="number"
              size="small"
              value={policy.max_claims_per_user}
              onChange={(e) => updateField('max_claims_per_user', Number(e.target.value))}
              inputProps={{ min: 0 }}
            />
            <TextField
              label="Approval Threshold"
              type="number"
              size="small"
              value={policy.approval_threshold}
              onChange={(e) => updateField('approval_threshold', Number(e.target.value))}
              inputProps={{ min: 0 }}
            />
          </Stack>

          <FormControlLabel
            control={
              <Switch
                checked={Boolean(policy.requires_certification)}
                onChange={(_, checked) => updateField('requires_certification', checked)}
              />
            }
            label="Requires certification"
          />

          <TextField
            label="Renewal Conditions (JSON)"
            multiline
            minRows={4}
            value={renewalText}
            onChange={(e) => handleRenewalChange(e.target.value)}
            size="small"
          />

          {(policy.created_at || policy.updated_at) && (
            <Stack spacing={1}>
              {policy.created_at && <Typography variant="body2" color="text.secondary">Created: {new Date(policy.created_at).toLocaleString()}</Typography>}
              {policy.updated_at && <Typography variant="body2" color="text.secondary">Updated: {new Date(policy.updated_at).toLocaleString()}</Typography>}
            </Stack>
          )}
        </Stack>

        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} className="ep-actions">
          <Button variant="contained" onClick={handleSave} disabled={saving || simulating}>
            {saving ? 'Saving…' : 'Save'}
          </Button>
          <Button variant="outlined" onClick={handleSimulate} disabled={saving || simulating}>
            {simulating ? 'Simulating…' : 'Simulate'}
          </Button>
          {isEdit && (
            <Button variant="outlined" color="error" onClick={handleDelete} disabled={saving || simulating}>
              Delete
            </Button>
          )}
          <Button variant="text" onClick={onCancel} disabled={saving || simulating}>
            Cancel
          </Button>
        </Stack>
      </Stack>
    </div>
  )
}
