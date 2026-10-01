import React, { useState } from 'react';
import {
  Autocomplete, Box, Button, Chip, Stack, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, TextField, Tooltip, Typography,
} from '@mui/material';
import type { ChipColor, MapFieldSpec } from './appModel';
import { resolve, text, type Scope } from './bindings';
import { useCondition } from './conditions';
import { useAppRuntime } from './AppRuntime';

/**
 * A key -> value mapping edited as tables (the staging binding's field ->
 * column map): rows come from data and are grouped, each picks its value
 * from the field's options, a hint chip marks a value that is still what was
 * suggested, and add controls append prefixed keys (identifiers, price
 * columns) that can be removed again. Empty values are kept as '' - the
 * operation that submits decides what an unbound row means.
 */

interface MapRow { key: string; label: string; caption?: string; group?: string; removable?: boolean }
type Hint = { value?: unknown; label?: string; color?: ChipColor; tooltip?: string };
type Values = Record<string, unknown>;

const blank = (v: unknown) => v === undefined || v === null || v === '';

function GroupAction({ action, scope }: { action: NonNullable<NonNullable<MapFieldSpec['groups']>[number]['action']>; scope: Scope }) {
  const { runActions } = useAppRuntime();
  const disabled = useCondition(action.disabledWhen, scope, false);
  const [busy, setBusy] = useState(false);
  const click = async () => {
    setBusy(true);
    try { await runActions(action.onClick, scope); } finally { setBusy(false); }
  };
  return <Button size="small" disabled={busy || (!!action.disabledWhen && disabled)} onClick={() => void click()}>{text(action.label, scope)}</Button>;
}

function AddControl({ add, present, onAdd, scope, disabled }: {
  add: NonNullable<MapFieldSpec['add']>[number]; present: Set<string>; onAdd: (key: string) => void; scope: Scope; disabled: boolean;
}) {
  const shown = useCondition(add.visibleWhen, scope, true);
  const [typed, setTyped] = useState<string | null>(null);
  if (add.visibleWhen && !shown) return null;
  const normalized = () => {
    const raw = (typed ?? '').trim();
    if (!add.code) return raw;
    const code = raw.toUpperCase().replace(/[^A-Z0-9_]/g, '_');
    return /^[A-Z][A-Z0-9_]{1,39}$/.test(code) ? code : '';
  };
  const commit = () => {
    const code = normalized();
    if (!code) return;
    onAdd(`${add.prefix}${code}`);
    setTyped(null);
  };
  return (
    <Stack direction="row" spacing={1} alignItems="center" sx={{ mt: 1 }}>
      <Autocomplete freeSolo size="small" sx={{ width: 240 }} disabled={disabled}
        options={(add.options ?? []).filter((o) => !present.has(`${add.prefix}${o}`))}
        value={typed} onChange={(_, v) => setTyped(v)} onInputChange={(_, v) => setTyped(v)}
        renderInput={(p) => <TextField {...p} label={text(add.inputLabel, scope)} helperText={add.helperText ? text(add.helperText, scope) : undefined} />} />
      <Button size="small" onClick={commit} disabled={disabled || !typed?.trim()}>{text(add.label, scope)}</Button>
    </Stack>
  );
}

export function MapField({ spec, value, onChange, disabled, options, scope }: {
  spec: MapFieldSpec; value: unknown; onChange: (v: unknown) => void; disabled: boolean; options: { value: string; label: string }[]; scope: Scope;
}) {
  const values = (value && typeof value === 'object' && !Array.isArray(value) ? value : {}) as Values;
  const listed = (resolve(spec.rows, scope) ?? []) as MapRow[];
  const hints = (spec.hints !== undefined ? resolve(spec.hints, scope) : undefined) as Record<string, Hint> | undefined;
  const adds = spec.add ?? [];
  const known = new Set((Array.isArray(listed) ? listed : []).map((r) => r.key));
  // Keys the user added (or a saved mapping carries) that no row lists.
  const extra: MapRow[] = Object.keys(values).filter((k) => !known.has(k)).sort().flatMap((k) => {
    const a = adds.find((x) => k.startsWith(x.prefix));
    if (!a) return [];
    const type = k.slice(a.prefix.length);
    return [{ key: k, caption: k, group: a.group, removable: true, label: a.rowLabel ? text(a.rowLabel, { ...scope, item: { type, key: k } }) : k }];
  });
  const rows = [...(Array.isArray(listed) ? listed : []), ...extra];
  const groups = spec.groups?.length ? spec.groups : [{ id: '', headers: true }];
  const inGroup = (g: string) => rows.filter((r) => (r.group ?? groups[0].id) === g);
  const set = (key: string, v: unknown) => onChange({ ...values, [key]: v ?? '' });
  const remove = (key: string) => {
    const next = { ...values };
    delete next[key];
    onChange(next);
  };
  const present = new Set(Object.keys(values));
  const valueOptions = options.map((o) => o.value);
  const labelOf = new Map(options.map((o) => [o.value, o.label]));
  const placeholder = spec.placeholder ? text(spec.placeholder, scope) : undefined;

  return (
    <Stack spacing={2}>
      {groups.map((g) => {
        const gr = inGroup(g.id);
        const local: Scope = { ...scope, map: { bound: gr.filter((r) => !blank(values[r.key])).length, total: gr.filter((r) => !r.removable).length } };
        const table = (
          <Table size="small" stickyHeader={g.headers}>
            {g.headers && (
              <TableHead>
                <TableRow>
                  <TableCell>{text(spec.keyHeader ?? '', local)}</TableCell>
                  <TableCell sx={{ width: '45%' }}>{text(spec.valueHeader ?? '', local)}</TableCell>
                </TableRow>
              </TableHead>
            )}
            <TableBody>
              {gr.map((r) => {
                const v = values[r.key];
                const hint = hints?.[r.key];
                const add = r.removable ? adds.find((a) => r.key.startsWith(a.prefix)) : undefined;
                return (
                  <TableRow key={r.key} hover>
                    <TableCell>
                      <Typography variant="body2" fontWeight={500}>{r.label}</Typography>
                      {r.caption && <Typography variant="caption" color="text.secondary" fontFamily="monospace">{r.caption}</Typography>}
                    </TableCell>
                    <TableCell sx={{ width: '45%' }}>
                      <Stack direction="row" spacing={1} alignItems="center">
                        <Autocomplete size="small" sx={{ flex: 1 }} disabled={disabled} options={valueOptions}
                          getOptionLabel={(o) => labelOf.get(o) ?? o}
                          value={blank(v) ? null : String(v)} onChange={(_, n) => set(r.key, n)}
                          renderInput={(p) => <TextField {...p} placeholder={placeholder} inputProps={{ ...p.inputProps, 'aria-label': r.label }} />} />
                        {hint && !blank(v) && String(hint.value) === String(v) && (
                          <Tooltip title={hint.tooltip ?? ''}>
                            <Chip size="small" variant="outlined" color={hint.color ?? 'default'} label={hint.label ?? ''} />
                          </Tooltip>
                        )}
                        {add && (
                          <Button size="small" color="inherit" disabled={disabled} onClick={() => remove(r.key)}
                            aria-label={`${add.removeLabel ? text(add.removeLabel, local) : 'Remove'} ${r.label}`}>
                            {add.removeLabel ? text(add.removeLabel, local) : 'Remove'}
                          </Button>
                        )}
                      </Stack>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        );
        return (
          <Box key={g.id || 'rows'}>
            {(g.title || g.action) && (
              <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 1 }}>
                <Typography variant="subtitle2" sx={{ flex: 1 }}>{g.title ? text(g.title, local) : ''}</Typography>
                {g.action && <GroupAction action={g.action} scope={local} />}
              </Stack>
            )}
            {g.help && <Typography variant="caption" color="text.secondary" component="p" sx={{ mb: 1 }}>{text(g.help, local)}</Typography>}
            {g.headers
              ? <TableContainer sx={{ maxHeight: spec.maxHeight ?? 380, border: 1, borderColor: 'divider', borderRadius: 1 }}>{table}</TableContainer>
              : <Box sx={{ border: 1, borderColor: 'divider', borderRadius: 1 }}>{table}</Box>}
            {adds.filter((a) => (a.group ?? groups[0].id) === g.id).map((a) => (
              <AddControl key={a.prefix} add={a} present={present} scope={local} disabled={disabled} onAdd={(k) => { if (!(k in values)) set(k, ''); }} />
            ))}
          </Box>
        );
      })}
    </Stack>
  );
}
