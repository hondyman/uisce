import React, { useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Alert, Autocomplete, Box, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle, FormControlLabel, IconButton,
  LinearProgress, MenuItem, Stack, Switch, TextField, Typography,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';
import { CatalogErrorAlert } from '../message-catalog/parts';
import { type ConfigColumn, type ConfigKind, type ConfigRow, masteringConfigApi } from './api';

export type EditorMode = 'new' | 'edit' | 'override';

interface Props {
  kind: ConfigKind;
  entity?: string;
  mode: EditorMode;
  /** The row being changed or overridden (omit for a new row). */
  row?: ConfigRow | null;
  onClose: () => void;
  onProposed: () => void;
}

const label = (name: string) => name.replace(/_cd$/, '').replace(/_/g, ' ').replace(/^./, (c) => c.toUpperCase());

/** A JSON list of plain values (match keys) - edited as chips. */
const isStringList = (v: unknown) => Array.isArray(v) && v.every((x) => typeof x !== 'object');
/** A JSON list of objects (fuzzy keys) - edited as rows with the objects' own fields. */
const isObjectList = (v: unknown) => Array.isArray(v) && v.length > 0 && v.every((x) => x && typeof x === 'object' && !Array.isArray(x));

function ObjectListField({ value, onChange }: { value: Record<string, unknown>[]; onChange: (v: Record<string, unknown>[]) => void }) {
  const keys = Array.from(new Set(value.flatMap((o) => Object.keys(o))));
  const set = (i: number, k: string, raw: string) => {
    const next = value.map((o) => ({ ...o }));
    const n = Number(raw);
    next[i][k] = raw !== '' && !Number.isNaN(n) && typeof value[i]?.[k] === 'number' ? n : raw;
    onChange(next);
  };
  return (
    <Stack spacing={1}>
      {value.map((o, i) => (
        <Stack key={i} direction="row" spacing={1} alignItems="center">
          {keys.map((k) => (
            <TextField key={k} size="small" label={label(k)} value={o[k] === undefined ? '' : String(o[k])} onChange={(e) => set(i, k, e.target.value)} />
          ))}
          <IconButton size="small" aria-label="Remove" onClick={() => onChange(value.filter((_, j) => j !== i))}><DeleteIcon fontSize="small" /></IconButton>
        </Stack>
      ))}
      <Box>
        <Button size="small" startIcon={<AddIcon />} onClick={() => onChange([...value, Object.fromEntries(keys.map((k) => [k, typeof value[0]?.[k] === 'number' ? 0 : '']))])}>
          Add
        </Button>
      </Box>
    </Stack>
  );
}

/**
 * Proposes a configuration change: a new row, a change to one of the tenant's
 * own rows, or an override of a gold-copy row (the tenant's own row with the
 * same key). The form comes from the table's columns, so the same editor
 * serves the vendor registry, every entity's source hierarchy and its match
 * rules. Nothing changes until a second administrator approves.
 */
export default function ConfigRowEditor({ kind, entity, mode, row, onClose, onProposed }: Props) {
  const table = useQuery({ queryKey: ['mdmcfg', 'editor', kind, entity], queryFn: () => masteringConfigApi.table(kind, entity) });
  const sources = useQuery({ queryKey: ['mdmcfg', 'editor', 'source_system'], queryFn: () => masteringConfigApi.table('source_system') });
  const [values, setValues] = useState<Record<string, unknown>>({});
  const [reason, setReason] = useState('');
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);

  const columns = useMemo(() => table.data?.table.columns ?? [], [table.data]);
  useEffect(() => {
    setValues(row ? { ...row.values } : {});
  }, [row]);

  const sourceCodes = useMemo(
    () => Array.from(new Set((sources.data?.table.rows ?? []).map((r) => String(r.values.code ?? '')))).filter(Boolean).sort(),
    [sources.data],
  );

  const set = (name: string, v: unknown) => setValues((prev) => ({ ...prev, [name]: v }));

  const field = (c: ConfigColumn) => {
    const v = values[c.name];
    // An override keeps the key (that is what makes it replace the gold row).
    const lockedKey = mode === 'override' && c.key;
    const common = { fullWidth: true, size: 'small' as const, label: `${label(c.name)}${c.required ? ' *' : ''}`, disabled: lockedKey };
    switch (c.type) {
      case 'source':
        return (
          <TextField select {...common} value={v ?? ''} onChange={(e) => set(c.name, e.target.value)}>
            {sourceCodes.map((s) => <MenuItem key={s} value={s}>{s}</MenuItem>)}
          </TextField>
        );
      case 'boolean':
        return (
          <FormControlLabel control={<Switch checked={v === true} onChange={(e) => set(c.name, e.target.checked)} disabled={lockedKey} />}
            label={label(c.name)} />
        );
      case 'integer':
      case 'number':
        return (
          <TextField {...common} type="number" value={v ?? ''} inputProps={{ step: c.type === 'integer' ? 1 : 'any' }}
            onChange={(e) => set(c.name, e.target.value === '' ? null : Number(e.target.value))} />
        );
      case 'json':
      case 'list':
        if (isObjectList(v)) {
          return (
            <Box>
              <Typography variant="caption" color="text.secondary">{common.label}</Typography>
              <ObjectListField value={v as Record<string, unknown>[]} onChange={(n) => set(c.name, n)} />
            </Box>
          );
        }
        if (v === undefined || v === null || isStringList(v)) {
          return (
            <Autocomplete multiple freeSolo options={[]} value={((v as unknown[]) ?? []).map(String)} disabled={lockedKey}
              onChange={(_, n) => set(c.name, n)}
              renderTags={(vals, getTagProps) => vals.map((x, i) => <Chip size="small" label={x} {...getTagProps({ index: i })} key={x} />)}
              renderInput={(p) => <TextField {...p} size="small" label={common.label} helperText="Type a value and press Enter" />} />
          );
        }
        return (
          <TextField {...common} multiline minRows={2} value={typeof v === 'string' ? v : JSON.stringify(v, null, 2)}
            onChange={(e) => { try { set(c.name, JSON.parse(e.target.value)); } catch { set(c.name, e.target.value); } }} />
        );
      default:
        return <TextField {...common} value={v ?? ''} onChange={(e) => set(c.name, e.target.value)} />;
    }
  };

  const submit = async () => {
    setBusy(true);
    setError(null);
    try {
      // Only what changed for an edit; everything for a new row or an override.
      const changed = mode === 'edit' && row
        ? Object.fromEntries(Object.entries(values).filter(([k, v]) => JSON.stringify(v) !== JSON.stringify(row.values[k])))
        : Object.fromEntries(Object.entries(values).filter(([, v]) => v !== undefined && v !== ''));
      await masteringConfigApi.propose({
        kind, entity, action: 'upsert', target_id: mode === 'edit' ? row?.id : undefined, values: changed, reason,
      });
      onProposed();
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };

  const title = { new: 'Propose a new row', edit: 'Propose a change', override: 'Override for your environment' }[mode];
  return (
    <Dialog open onClose={() => !busy && onClose()} maxWidth="sm" fullWidth>
      <DialogTitle>{title}</DialogTitle>
      <DialogContent dividers>
        {mode === 'override' && (
          <Alert severity="info" sx={{ mb: 2 }}>
            This row comes from the gold copy. Your version replaces it in your environment only; the gold copy is unchanged.
          </Alert>
        )}
        {(table.isLoading || sources.isLoading) && <LinearProgress />}
        {table.error && <CatalogErrorAlert error={table.error} />}
        <Stack spacing={2} sx={{ mt: 1 }}>
          {columns.map((c) => <Box key={c.name}>{field(c)}</Box>)}
          <TextField fullWidth size="small" multiline minRows={2} label="Reason (shown to the approver)" value={reason} onChange={(e) => setReason(e.target.value)} />
        </Stack>
        {error != null && <Box sx={{ mt: 2 }}><CatalogErrorAlert error={error} /></Box>}
      </DialogContent>
      <DialogActions>
        <Typography variant="caption" color="text.secondary" sx={{ flex: 1, pl: 2 }}>Nothing changes until another administrator approves.</Typography>
        <Button onClick={onClose} disabled={busy}>Cancel</Button>
        <Button variant="contained" onClick={submit} disabled={busy || !columns.length}>Send for approval</Button>
      </DialogActions>
    </Dialog>
  );
}
