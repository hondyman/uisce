import React, { useMemo, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Alert, Autocomplete, Box, Button, Checkbox, Chip, CircularProgress, Divider, FormControlLabel, IconButton,
  MenuItem, Stack, Table, TableBody, TableCell, TableHead, TableRow, TextField, Tooltip, Typography,
} from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';
import UploadFileIcon from '@mui/icons-material/UploadFile';
import AutoFixHighIcon from '@mui/icons-material/AutoFixHigh';
import PreviewIcon from '@mui/icons-material/Preview';
import {
  boTarget, Column, ColumnType, Condition, FieldMap, NodeConfigs, NodeKind, pipelinesApi, platformApi,
  SpecNode, TargetField,
} from './api';
import { missingRequired } from './fields';

const COLUMN_TYPES: ColumnType[] = ['string', 'int', 'float', 'decimal', 'bool', 'date', 'timestamp'];
const TRANSFORMS = [
  { v: '', l: 'as is' }, { v: 'trim', l: 'trim spaces' }, { v: 'upper', l: 'UPPER CASE' }, { v: 'lower', l: 'lower case' },
  { v: 'to_date', l: 'to date' }, { v: 'to_number', l: 'to number' }, { v: 'lookup', l: 'lookup table' },
];
// The rule engine's operators that can filter at the source (vm.CompileConditionSQL).
const FILTER_OPS: { v: string; l: string; arity: 0 | 1 | 2 | 'list' }[] = [
  { v: 'equals', l: 'equals', arity: 1 }, { v: 'not_equals', l: 'does not equal', arity: 1 },
  { v: 'greater_than', l: '>', arity: 1 }, { v: 'greater_equal', l: '≥', arity: 1 },
  { v: 'less_than', l: '<', arity: 1 }, { v: 'less_equal', l: '≤', arity: 1 },
  { v: 'between', l: 'between', arity: 2 }, { v: 'in', l: 'is one of', arity: 'list' }, { v: 'not_in', l: 'is not one of', arity: 'list' },
  { v: 'contains', l: 'contains', arity: 1 }, { v: 'starts_with', l: 'starts with', arity: 1 }, { v: 'ends_with', l: 'ends with', arity: 1 },
  { v: 'before', l: 'before (date)', arity: 1 }, { v: 'after', l: 'after (date)', arity: 1 },
  { v: 'is_null', l: 'is empty', arity: 0 }, { v: 'is_not_null', l: 'has a value', arity: 0 },
  { v: 'is_true', l: 'is true', arity: 0 }, { v: 'is_false', l: 'is false', arity: 0 },
];

export interface NodeConfigPanelProps {
  node: SpecNode;
  inputFields: Column[];        // fields arriving at this node
  mapTargets?: TargetField[];   // for a map step: fields of the destination it feeds
  mapTargetLabel?: string;
  onChange: (node: SpecNode) => void;
  onDelete: () => void;
}

export function NodeConfigPanel(props: NodeConfigPanelProps) {
  const { node, onChange, onDelete } = props;
  const set = <K extends NodeKind>(patch: Partial<NodeConfigs[K]>) =>
    onChange({ ...node, config: { ...(node.config as object), ...patch } as NodeConfigs[K] });
  const cfg = node.config as never;

  return (
    <Stack spacing={2} sx={{ p: 2 }}>
      <Stack direction="row" spacing={1} alignItems="center">
        <TextField
          size="small" label="Step name" fullWidth value={node.label ?? ''}
          onChange={e => onChange({ ...node, label: e.target.value })}
        />
        <Tooltip title="Remove this step">
          <IconButton onClick={onDelete} aria-label="remove step"><DeleteIcon /></IconButton>
        </Tooltip>
      </Stack>
      <Divider />
      {node.type === 'file_source' && <FileSourceForm cfg={cfg} set={set} />}
      {node.type === 'bo_source' && <BOSourceForm cfg={cfg} set={set} />}
      {node.type === 'validate' && <ValidateForm cfg={cfg} set={set} fields={props.inputFields} />}
      {node.type === 'rule_check' && <RuleCheckForm cfg={cfg} set={set} />}
      {node.type === 'map' && (
        <MapForm cfg={cfg} set={set} fields={props.inputFields} targets={props.mapTargets} targetLabel={props.mapTargetLabel} />
      )}
      {node.type === 'bo_sink' && <BOSinkForm cfg={cfg} set={set} fields={props.inputFields} />}
      {node.type === 'staging_sink' && <StagingSinkForm cfg={cfg} set={set} fields={props.inputFields} />}
      {node.type === 'file_sink' && <FileSinkForm cfg={cfg} set={set} />}
    </Stack>
  );
}

type FormProps<K extends NodeKind> = { cfg: NodeConfigs[K]; set: (p: Partial<NodeConfigs[K]>) => void };

function Hint({ children }: { children: React.ReactNode }) {
  return <Typography variant="body2" color="text.secondary">{children}</Typography>;
}

// --- sources -----------------------------------------------------------------

function FileSourceForm({ cfg, set }: FormProps<'file_source'>) {
  const files = useQuery({ queryKey: ['dp-files'], queryFn: () => pipelinesApi.files() });
  const inputRef = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [sample, setSample] = useState<Record<string, unknown>[] | null>(null);
  const [rowCount, setRowCount] = useState<number | undefined>();

  const upload = async (f: File) => {
    setBusy('Uploading…'); setErr(null);
    try {
      const r = await pipelinesApi.upload(f);
      await files.refetch();
      // A new version of the same file keeps its column contract; a different file starts over.
      set(r.uri === cfg.uri ? { uri: r.uri } : { uri: r.uri, format: guessFormat(r.uri), columns: [] });
    } catch (e) { setErr(String((e as Error).message)); } finally { setBusy(null); }
  };

  const profile = async () => {
    setBusy('Reading the file…'); setErr(null);
    try {
      const p = await pipelinesApi.profile({
        uri: cfg.uri, format: cfg.format, delimiter: cfg.delimiter, has_header: cfg.has_header ?? true, count_rows: true,
      });
      setSample(p.sample); setRowCount(p.row_count);
      // Keep what the analyst already tightened; add newly seen columns.
      const known = new Map((cfg.columns ?? []).map(c => [c.name, c]));
      set({ format: p.format as never, columns: p.columns.map(c => known.get(c.name) ?? c) });
    } catch (e) { setErr(String((e as Error).message)); } finally { setBusy(null); }
  };

  const cols = cfg.columns ?? [];
  const setCol = (i: number, patch: Partial<Column>) => set({ columns: cols.map((c, j) => (j === i ? { ...c, ...patch } : c)) });

  return (
    <Stack spacing={2}>
      <Hint>Pick an uploaded file (or upload one), then press <b>Read file</b> to see its columns. Every row is checked against the columns below.</Hint>
      <Stack direction="row" spacing={1}>
        <Autocomplete
          fullWidth size="small" options={(files.data ?? []).map(f => f.path)} value={cfg.uri || null}
          loading={files.isLoading}
          onChange={(_, v) => set({ uri: v ?? '', format: guessFormat(v ?? ''), columns: [] })}
          renderInput={p => <TextField {...p} label="File" error={files.isError} helperText={files.isError ? 'Could not list files' : undefined} />}
        />
        <input ref={inputRef} type="file" hidden onChange={e => e.target.files?.[0] && upload(e.target.files[0])} />
        <Tooltip title="Upload a file"><Button variant="outlined" onClick={() => inputRef.current?.click()}><UploadFileIcon /></Button></Tooltip>
      </Stack>
      <Stack direction="row" spacing={1}>
        <TextField select size="small" label="Format" value={cfg.format ?? 'csv'} onChange={e => set({ format: e.target.value as never })} sx={{ minWidth: 110 }}>
          {['csv', 'json', 'parquet'].map(f => <MenuItem key={f} value={f}>{f}</MenuItem>)}
        </TextField>
        {cfg.format === 'csv' && (
          <>
            <TextField select size="small" label="Separator" value={cfg.delimiter ?? ','} onChange={e => set({ delimiter: e.target.value })} sx={{ minWidth: 120 }}>
              <MenuItem value=",">comma ,</MenuItem><MenuItem value="|">pipe |</MenuItem>
              <MenuItem value=";">semicolon ;</MenuItem><MenuItem value="tab">tab</MenuItem>
            </TextField>
            <FormControlLabel control={<Checkbox checked={cfg.has_header ?? true} onChange={e => set({ has_header: e.target.checked })} />} label="Header row" />
          </>
        )}
      </Stack>
      <Button variant="contained" startIcon={busy ? <CircularProgress size={16} /> : <PreviewIcon />} disabled={!cfg.uri || !!busy} onClick={profile}>
        {busy ?? 'Read file'}
      </Button>
      {err && <Alert severity="error">{err}</Alert>}
      {rowCount !== undefined && <Alert severity="info">{rowCount.toLocaleString()} rows in this file.</Alert>}
      {cols.length > 0 && (
        <Box>
          <Typography variant="subtitle2" gutterBottom>Columns ({cols.length})</Typography>
          <Table size="small">
            <TableHead><TableRow><TableCell>Column</TableCell><TableCell>Type</TableCell><TableCell>Required</TableCell></TableRow></TableHead>
            <TableBody>
              {cols.map((c, i) => (
                <TableRow key={c.name}>
                  <TableCell>
                    <Typography variant="body2" fontFamily="monospace">{c.name}</Typography>
                    {sample?.[0] && <Typography variant="caption" color="text.secondary" noWrap>e.g. {String(sample[0][c.name] ?? '—')}</Typography>}
                  </TableCell>
                  <TableCell>
                    <TextField select size="small" variant="standard" value={c.type} onChange={e => setCol(i, { type: e.target.value as ColumnType })}>
                      {COLUMN_TYPES.map(t => <MenuItem key={t} value={t}>{t}</MenuItem>)}
                    </TextField>
                  </TableCell>
                  <TableCell>
                    <Checkbox size="small" checked={c.nullable === false} onChange={e => setCol(i, { nullable: e.target.checked ? false : undefined })} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Box>
      )}
    </Stack>
  );
}

function guessFormat(uri: string): 'csv' | 'json' | 'parquet' {
  const u = uri.toLowerCase();
  if (/\.(json|ndjson|jsonl)$/.test(u)) return 'json';
  if (/\.(parquet|pq)$/.test(u)) return 'parquet';
  return 'csv';
}

function useBOs() {
  return useQuery({ queryKey: ['dp-bos'], queryFn: platformApi.businessObjects, staleTime: 5 * 60_000 });
}

export function useBOFields(boKey?: string) {
  return useQuery({
    queryKey: ['dp-bo-schema', boKey], enabled: !!boKey, staleTime: 5 * 60_000,
    queryFn: async () => (await platformApi.boSchema(boKey!)).fields ?? [],
  });
}

function BOPicker({ value, onChange, label = 'Business object' }: { value?: string; onChange: (v: string) => void; label?: string }) {
  const bos = useBOs();
  const opts = bos.data ?? [];
  return (
    <Autocomplete
      size="small" options={opts} loading={bos.isLoading}
      value={opts.find(b => b.name === value) ?? null}
      getOptionLabel={b => b.display_name || b.name}
      isOptionEqualToValue={(a, b) => a.name === b.name}
      onChange={(_, b) => onChange(b?.name ?? '')}
      renderOption={(p, b) => (
        <li {...p} key={b.id}>
          <Box><Typography variant="body2">{b.display_name || b.name}</Typography>
            {b.description && <Typography variant="caption" color="text.secondary">{b.description}</Typography>}</Box>
        </li>
      )}
      renderInput={p => <TextField {...p} label={label} error={bos.isError} helperText={bos.isError ? 'Could not load business objects' : undefined} />}
    />
  );
}

function BOSourceForm({ cfg, set }: FormProps<'bo_source'>) {
  const fields = useBOFields(cfg.bo_key);
  const filters = cfg.filters ?? [];
  const setF = (i: number, patch: Partial<Condition>) => set({ filters: filters.map((f, j) => (j === i ? { ...f, ...patch } : f)) });
  return (
    <Stack spacing={2}>
      <Hint>Read records from a business object. Only your tenant's records are read.</Hint>
      <BOPicker value={cfg.bo_key} onChange={v => set({ bo_key: v, filters: [] })} />
      {cfg.bo_key && (
        <Box>
          <Typography variant="subtitle2" gutterBottom>Only records where…</Typography>
          <Stack spacing={1}>
            {filters.map((f, i) => {
              const op = FILTER_OPS.find(o => o.v === f.operator);
              return (
                <Stack key={i} direction="row" spacing={1} alignItems="center">
                  <TextField select size="small" label="Field" value={f.field} onChange={e => setF(i, { field: e.target.value })} sx={{ minWidth: 140 }}>
                    {(fields.data ?? []).map(fl => { const t = boTarget(fl); return <MenuItem key={t.name} value={t.name}>{t.label}</MenuItem>; })}
                  </TextField>
                  <TextField select size="small" label="Condition" value={f.operator} onChange={e => setF(i, { operator: e.target.value, value: undefined })} sx={{ minWidth: 130 }}>
                    {FILTER_OPS.map(o => <MenuItem key={o.v} value={o.v}>{o.l}</MenuItem>)}
                  </TextField>
                  <ConditionValue arity={op?.arity ?? 1} value={f.value} onChange={v => setF(i, { value: v })} />
                  <IconButton size="small" onClick={() => set({ filters: filters.filter((_, j) => j !== i) })}><DeleteIcon fontSize="small" /></IconButton>
                </Stack>
              );
            })}
            <Button size="small" startIcon={<AddIcon />} onClick={() => set({ filters: [...filters, { field: '', operator: 'equals' }] })} sx={{ alignSelf: 'start' }}>
              Add condition
            </Button>
          </Stack>
        </Box>
      )}
      <TextField size="small" type="number" label="Read at most (optional)" value={cfg.limit ?? ''}
        onChange={e => set({ limit: e.target.value ? Number(e.target.value) : undefined })} />
    </Stack>
  );
}

function ConditionValue({ arity, value, onChange }: { arity: 0 | 1 | 2 | 'list'; value: unknown; onChange: (v: unknown) => void }) {
  if (arity === 0) return <Box sx={{ flex: 1 }} />;
  if (arity === 2) {
    const [lo, hi] = Array.isArray(value) ? value : ['', ''];
    return (
      <Stack direction="row" spacing={0.5} sx={{ flex: 1 }}>
        <TextField size="small" label="from" value={lo ?? ''} onChange={e => onChange([e.target.value, hi])} />
        <TextField size="small" label="to" value={hi ?? ''} onChange={e => onChange([lo, e.target.value])} />
      </Stack>
    );
  }
  if (arity === 'list') {
    return (
      <Autocomplete multiple freeSolo size="small" sx={{ flex: 1 }} options={[] as string[]}
        value={Array.isArray(value) ? (value as string[]) : []} onChange={(_, v) => onChange(v)}
        renderInput={p => <TextField {...p} label="values" placeholder="type, then Enter" />} />
    );
  }
  return <TextField size="small" label="value" sx={{ flex: 1 }} value={(value as string) ?? ''} onChange={e => onChange(e.target.value)} />;
}

// --- steps -------------------------------------------------------------------

function FieldMulti({ label, fields, value, onChange }: { label: string; fields: Column[]; value: string[]; onChange: (v: string[]) => void }) {
  return (
    <Autocomplete multiple size="small" options={fields.map(f => f.name)} value={value} onChange={(_, v) => onChange(v)}
      renderInput={p => <TextField {...p} label={label} />} />
  );
}

function ValidateForm({ cfg, set, fields }: FormProps<'validate'> & { fields: Column[] }) {
  return (
    <Stack spacing={2}>
      <Hint>Rows missing a required value, or repeating a key already seen in this run, are rejected with the reason.</Hint>
      <FieldMulti label="Must have a value" fields={fields} value={cfg.required ?? []} onChange={v => set({ required: v })} />
      <FieldMulti label="Must be unique together" fields={fields} value={cfg.unique ?? []} onChange={v => set({ unique: v })} />
      {fields.length === 0 && <Alert severity="info">Connect this step to a source to pick fields.</Alert>}
    </Stack>
  );
}

function RuleCheckForm({ cfg, set }: FormProps<'rule_check'>) {
  const rules = useQuery({
    queryKey: ['dp-rules', cfg.bo_key], enabled: !!cfg.bo_key,
    queryFn: () => platformApi.rules(cfg.bo_key!),
  });
  const active = (rules.data ?? []).filter(r => r.is_active);
  const picked = new Set(cfg.rule_ids ?? []);
  return (
    <Stack spacing={2}>
      <Hint>Apply validation rules from the rules catalog - the same rules the business object enforces. <b>Block</b> rules reject the row; <b>Warn</b> rules keep it and record a warning.</Hint>
      <BOPicker label="Rules of business object" value={cfg.bo_key} onChange={v => set({ bo_key: v })} />
      {rules.isLoading && <CircularProgress size={20} />}
      {cfg.bo_key && !rules.isLoading && active.length === 0 && <Alert severity="info">This business object has no active rules yet.</Alert>}
      {active.map(r => (
        <FormControlLabel key={r.id} sx={{ alignItems: 'flex-start' }}
          control={<Checkbox checked={picked.has(r.id)} onChange={e => set({
            rule_ids: e.target.checked ? [...picked, r.id] : [...picked].filter(x => x !== r.id),
          })} />}
          label={
            <Box>
              <Stack direction="row" spacing={1} alignItems="center">
                <Typography variant="body2">{r.name}</Typography>
                <Chip size="small" label={r.severity === 'BLOCK' ? 'Block' : 'Warn'} color={r.severity === 'BLOCK' ? 'error' : 'warning'} variant="outlined" />
                {r.origin === 'core' && <Chip size="small" label="core" />}
              </Stack>
              {r.description && <Typography variant="caption" color="text.secondary">{r.description}</Typography>}
            </Box>
          }
        />
      ))}
      <Button size="small" href="/core/validation-rules/editor" target="_blank" sx={{ alignSelf: 'start' }}>Create a rule</Button>
    </Stack>
  );
}

function MapForm({ cfg, set, fields, targets, targetLabel }:
  FormProps<'map'> & { fields: Column[]; targets?: TargetField[]; targetLabel?: string }) {
  const [suggesting, setSuggesting] = useState(false);
  const [note, setNote] = useState<string | null>(null);
  const rows = cfg.fields ?? [];
  const setRow = (i: number, patch: Partial<FieldMap>) => set({ fields: rows.map((r, j) => (j === i ? { ...r, ...patch } : r)) });
  const missing = useMemo(() => (targets ? missingRequired(rows.map(r => r.to), targets) : []), [rows, targets]);

  const suggest = async () => {
    if (!targets) return;
    setSuggesting(true); setNote(null);
    try {
      const s = await pipelinesApi.suggestMapping(fields, targets);
      const mappedFrom = new Set(rows.map(r => r.from));
      const add = s.filter(x => !mappedFrom.has(x.from) && x.confidence >= 0.5)
        .map(x => ({ from: x.from, to: x.to, transform: x.transform || undefined }));
      set({ fields: [...rows, ...add] });
      setNote(add.length ? `Added ${add.length} suggested mappings - check them before running.` : 'No new matches found.');
    } catch (e) { setNote(String((e as Error).message)); } finally { setSuggesting(false); }
  };

  return (
    <Stack spacing={2}>
      <Hint>Choose which incoming field fills each {targetLabel ? <b>{targetLabel}</b> : 'output'} field.</Hint>
      {targets && (
        <Button variant="outlined" startIcon={suggesting ? <CircularProgress size={16} /> : <AutoFixHighIcon />} disabled={suggesting || !fields.length} onClick={suggest}>
          Suggest mappings
        </Button>
      )}
      {note && <Alert severity="info" onClose={() => setNote(null)}>{note}</Alert>}
      {missing.length > 0 && <Alert severity="warning">Required and not mapped yet: {missing.map(m => m.label || m.name).join(', ')}</Alert>}
      <Table size="small">
        <TableHead><TableRow><TableCell>From</TableCell><TableCell>To</TableCell><TableCell>Transform</TableCell><TableCell /></TableRow></TableHead>
        <TableBody>
          {rows.map((r, i) => (
            <TableRow key={i}>
              <TableCell sx={{ minWidth: 120 }}>
                <TextField select size="small" variant="standard" fullWidth value={r.from} onChange={e => setRow(i, { from: e.target.value })}>
                  {fields.map(f => <MenuItem key={f.name} value={f.name}>{f.name}</MenuItem>)}
                  {r.from && !fields.some(f => f.name === r.from) && <MenuItem value={r.from}>{r.from} (missing)</MenuItem>}
                </TextField>
              </TableCell>
              <TableCell sx={{ minWidth: 120 }}>
                {targets ? (
                  <TextField select size="small" variant="standard" fullWidth value={r.to} onChange={e => setRow(i, { to: e.target.value })}>
                    {targets.map(t => <MenuItem key={t.name} value={t.name}>{t.label || t.name}{t.required ? ' *' : ''}</MenuItem>)}
                  </TextField>
                ) : (
                  <TextField size="small" variant="standard" fullWidth value={r.to} onChange={e => setRow(i, { to: e.target.value })} />
                )}
              </TableCell>
              <TableCell>
                <TextField select size="small" variant="standard" value={r.transform ?? ''} onChange={e => setRow(i, { transform: e.target.value || undefined })}>
                  {TRANSFORMS.map(t => <MenuItem key={t.v} value={t.v}>{t.l}</MenuItem>)}
                </TextField>
                {r.transform === 'lookup' && (
                  <TextField size="small" variant="standard" placeholder="A=Alpha, B=Beta" sx={{ mt: 0.5 }}
                    value={Object.entries(r.lookup ?? {}).map(([k, v]) => `${k}=${v}`).join(', ')}
                    onChange={e => setRow(i, { lookup: parseLookup(e.target.value) })} />
                )}
              </TableCell>
              <TableCell padding="none">
                <IconButton size="small" onClick={() => set({ fields: rows.filter((_, j) => j !== i) })}><DeleteIcon fontSize="small" /></IconButton>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <Button size="small" startIcon={<AddIcon />} sx={{ alignSelf: 'start' }} onClick={() => set({ fields: [...rows, { from: '', to: '' }] })}>Add mapping</Button>
      <FormControlLabel control={<Checkbox checked={!!cfg.keep_unmapped} onChange={e => set({ keep_unmapped: e.target.checked })} />} label="Also pass through fields I didn't map" />
    </Stack>
  );
}

function parseLookup(s: string): Record<string, string> {
  const out: Record<string, string> = {};
  for (const part of s.split(',')) {
    const [k, ...v] = part.split('=');
    if (k?.trim()) out[k.trim()] = v.join('=').trim();
  }
  return out;
}

// --- destinations ------------------------------------------------------------

function BOSinkForm({ cfg, set, fields }: FormProps<'bo_sink'> & { fields: Column[] }) {
  const bofields = useBOFields(cfg.bo_key);
  const targets = (bofields.data ?? []).map(boTarget);
  const incoming = new Set(fields.map(f => f.name));
  const unknown = fields.filter(f => targets.length && !targets.some(t => t.name === f.name));
  const missing = missingRequired([...incoming], targets);
  return (
    <Stack spacing={2}>
      <Hint>Write each row as a business object record. Every record goes through the object's validation rules; rejected records are reported with the rule that stopped them.</Hint>
      <BOPicker value={cfg.bo_key} onChange={v => set({ bo_key: v, key_fields: [] })} />
      <TextField select size="small" label="When the record already exists" value={cfg.mode ?? 'create'} onChange={e => set({ mode: e.target.value as never })}>
        <MenuItem value="create">always create a new record</MenuItem>
        <MenuItem value="upsert">update it (match on key fields)</MenuItem>
      </TextField>
      {cfg.mode === 'upsert' && (
        <Autocomplete multiple size="small" options={targets.map(t => t.name)} value={cfg.key_fields ?? []} onChange={(_, v) => set({ key_fields: v })}
          renderInput={p => <TextField {...p} label="Match records on" />} />
      )}
      <FormControlLabel control={<Checkbox checked={!!cfg.dry_run} onChange={e => set({ dry_run: e.target.checked })} />}
        label="Rehearsal: check every record against the rules, but save nothing" />
      {unknown.length > 0 && <Alert severity="warning">Not fields of this object (add a Map step): {unknown.map(u => u.name).join(', ')}</Alert>}
      {missing.length > 0 && <Alert severity="warning">Required fields no step provides: {missing.map(m => m.label || m.name).join(', ')}</Alert>}
    </Stack>
  );
}

function StagingSinkForm({ cfg, set, fields }: FormProps<'staging_sink'> & { fields: Column[] }) {
  const tables = useQuery({ queryKey: ['dp-staging'], queryFn: pipelinesApi.stagingTables, retry: false });
  const table = tables.data?.find(t => t.table === cfg.table);
  const cols = cfg.columns ?? {};
  return (
    <Stack spacing={2}>
      <Hint>Bulk-load rows into a staging table, as one tracked load. Re-running the same load reference does nothing; a failed load is cleared and retried.</Hint>
      {tables.isError && <Alert severity="warning">{String((tables.error as Error).message)}</Alert>}
      <TextField select size="small" label="Staging table" value={cfg.table ?? ''} onChange={e => set({ table: e.target.value, columns: {} })}>
        {(tables.data ?? []).map(t => <MenuItem key={t.table} value={t.table}>{t.table}</MenuItem>)}
      </TextField>
      <Stack direction="row" spacing={1}>
        <TextField size="small" label="Source system" placeholder="FACTSET" value={cfg.source_cd ?? ''} onChange={e => set({ source_cd: e.target.value.toUpperCase() })} />
        <TextField size="small" label="Domain" placeholder="PRODUCT" value={cfg.domain ?? ''} onChange={e => set({ domain: e.target.value.toUpperCase() })} />
      </Stack>
      <TextField size="small" label="Load reference (optional)" helperText="e.g. the file date. Blank: every run is a new load."
        value={cfg.run_ref ?? ''} onChange={e => set({ run_ref: e.target.value || undefined })} />
      {table && (
        <Box>
          <Typography variant="subtitle2" gutterBottom>Which field fills each column</Typography>
          <Table size="small">
            <TableBody>
              {table.columns.map(c => {
                const from = Object.entries(cols).find(([, col]) => col === c.name)?.[0] ?? '';
                return (
                  <TableRow key={c.name}>
                    <TableCell><Typography variant="body2" fontFamily="monospace">{c.name}{c.required ? ' *' : ''}</Typography></TableCell>
                    <TableCell>
                      <TextField select size="small" variant="standard" fullWidth value={from} onChange={e => {
                        const next = Object.fromEntries(Object.entries(cols).filter(([, col]) => col !== c.name));
                        if (e.target.value) next[e.target.value] = c.name;
                        set({ columns: next });
                      }}>
                        <MenuItem value=""><em>not loaded</em></MenuItem>
                        {fields.map(f => <MenuItem key={f.name} value={f.name}>{f.name}</MenuItem>)}
                      </TextField>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </Box>
      )}
    </Stack>
  );
}

function FileSinkForm({ cfg, set }: FormProps<'file_sink'>) {
  return (
    <Stack spacing={2}>
      <Hint>Export the rows to a file. It appears only when the whole run succeeds.</Hint>
      <TextField size="small" label="File name" placeholder="exports/funds.csv" value={cfg.uri ?? ''}
        onChange={e => set({ uri: e.target.value, format: guessFormat(e.target.value) })} />
      <Stack direction="row" spacing={1}>
        <TextField select size="small" label="Format" value={cfg.format ?? 'csv'} onChange={e => set({ format: e.target.value as never })} sx={{ minWidth: 110 }}>
          {['csv', 'json', 'parquet'].map(f => <MenuItem key={f} value={f}>{f}</MenuItem>)}
        </TextField>
        {cfg.format === 'csv' && (
          <TextField select size="small" label="Separator" value={cfg.delimiter ?? ','} onChange={e => set({ delimiter: e.target.value })} sx={{ minWidth: 120 }}>
            <MenuItem value=",">comma ,</MenuItem><MenuItem value="|">pipe |</MenuItem><MenuItem value=";">semicolon ;</MenuItem>
          </TextField>
        )}
      </Stack>
    </Stack>
  );
}
