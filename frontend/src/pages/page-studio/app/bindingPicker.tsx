import React, { useMemo, useState } from 'react';
import {
  Box, Chip, Collapse, IconButton, InputAdornment, List, ListItemButton, Popover, Stack, TextField, Tooltip, Typography,
} from '@mui/material';
import DataObjectIcon from '@mui/icons-material/DataObject';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import SearchIcon from '@mui/icons-material/Search';
import { useAppRuntime } from './AppRuntime';
import type { Scope } from './bindings';

/**
 * The binding picker: browse what a setting can read instead of typing
 * {{paths}} by hand. It shows the page's live state - variables, route
 * parameters, each query's real result (so the data's actual shape, not a
 * guess) - plus the names that exist only where the setting is used (row,
 * form, item...). Picking an entry hands its path to the field.
 */

/** One browsable entry. */
export interface ScopeEntry {
  path: string;
  /** The last segment, as shown in the tree. */
  name: string;
  /** text, number, yes/no, list (n), object, empty. */
  type: string;
  /** A short look at the value. */
  preview: string;
  /** Whether it has entries of its own. */
  expandable: boolean;
}

const typeOf = (v: unknown): string => {
  if (v === null || v === undefined) return 'empty';
  if (Array.isArray(v)) return `list (${v.length})`;
  if (typeof v === 'object') return 'object';
  if (typeof v === 'boolean') return 'yes/no';
  return typeof v === 'number' ? 'number' : 'text';
};
const previewOf = (v: unknown): string => {
  if (v === null || v === undefined) return '';
  if (Array.isArray(v)) return v.length ? `first: ${previewOf(v[0])}` : '';
  if (typeof v === 'object') return Object.keys(v as object).slice(0, 4).join(', ');
  const s = String(v);
  return s.length > 40 ? `${s.slice(0, 40)}…` : s;
};

/** The entries directly under a value: an object's keys, or a list's first item (as index 0) and its length. */
export function entriesUnder(value: unknown, base: string): ScopeEntry[] {
  const entry = (name: string, v: unknown): ScopeEntry => ({
    path: base ? `${base}.${name}` : name, name, type: typeOf(v), preview: previewOf(v),
    expandable: !!v && typeof v === 'object' && (Array.isArray(v) ? v.length > 0 : Object.keys(v as object).length > 0),
  });
  if (Array.isArray(value)) return value.length ? [entry('length', value.length), entry('0', value[0])] : [entry('length', 0)];
  if (value && typeof value === 'object') return Object.entries(value as Record<string, unknown>).map(([k, v]) => entry(k, v));
  return [];
}

const valueAt = (root: unknown, path: string): unknown =>
  path.split('.').reduce<unknown>((v, k) => (v === null || v === undefined ? undefined : (v as Record<string, unknown>)[k]), root);

/** Every path down to a depth, for search (capped, so a huge result cannot freeze the designer). */
export function flattenScope(root: unknown, maxDepth = 7, cap = 600): ScopeEntry[] {
  const out: ScopeEntry[] = [];
  const visit = (value: unknown, base: string, depth: number) => {
    for (const e of entriesUnder(value, base)) {
      if (out.length >= cap) return;
      out.push(e);
      if (e.expandable && depth < maxDepth) visit(valueAt(root, e.path), e.path, depth + 1);
    }
  };
  visit(root, '', 1);
  return out;
}

/** What the picker browses: the page's live state, trimmed to what bindings read. */
export function browsableScope(scope: Scope): Record<string, unknown> {
  const queries = Object.fromEntries(Object.entries((scope.queries ?? {}) as Record<string, { data?: unknown; isLoading?: boolean; error?: unknown }>).map(([id, q]) => [
    id, { data: q?.data ?? null, isLoading: !!q?.isLoading, error: q?.error ? String((q.error as Error).message ?? q.error) : null },
  ]));
  return { vars: scope.vars ?? {}, queries, route: scope.route ?? {} };
}

function Row({ e, depth, open, onToggle, onPick }: { e: ScopeEntry; depth: number; open: boolean; onToggle: () => void; onPick: (path: string) => void }) {
  return (
    <ListItemButton dense onClick={() => onPick(e.path)} sx={{ pl: 1 + depth * 1.75, py: 0.25 }} aria-label={`Use ${e.path}`}>
      <Box sx={{ width: 24, display: 'flex' }}>
        {e.expandable && (
          <IconButton size="small" aria-label={open ? `Collapse ${e.path}` : `Expand ${e.path}`} onClick={(ev) => { ev.stopPropagation(); onToggle(); }}>
            {open ? <ExpandMoreIcon fontSize="inherit" /> : <ChevronRightIcon fontSize="inherit" />}
          </IconButton>
        )}
      </Box>
      <Typography variant="body2" fontFamily="monospace" sx={{ mr: 1 }}>{e.name}</Typography>
      <Chip size="small" variant="outlined" label={e.type} sx={{ height: 18, fontSize: 10, mr: 1 }} />
      <Typography variant="caption" color="text.secondary" noWrap sx={{ flex: 1, minWidth: 0 }}>{e.preview}</Typography>
    </ListItemButton>
  );
}

function Tree({ root, base, depth, onPick, startOpen }: { root: unknown; base: string; depth: number; onPick: (path: string) => void; startOpen?: boolean }) {
  const [open, setOpen] = useState<Record<string, boolean>>({});
  const entries = entriesUnder(base ? valueAt(root, base) : root, base);
  return (
    <>
      {entries.map((e) => {
        const isOpen = open[e.path] ?? (!!startOpen && depth === 0);
        return (
          <React.Fragment key={e.path}>
            <Row e={e} depth={depth} open={isOpen} onToggle={() => setOpen((o) => ({ ...o, [e.path]: !isOpen }))} onPick={onPick} />
            {e.expandable && (
              <Collapse in={isOpen} unmountOnExit><Tree root={root} base={e.path} depth={depth + 1} onPick={onPick} /></Collapse>
            )}
          </React.Fragment>
        );
      })}
    </>
  );
}

/** The browse button and its panel. `contextPaths` are names that exist only where the setting is used (row.x, form, item...). */
export function BindingPicker({ onPick, contextPaths = [], label = 'Browse data' }: { onPick: (path: string) => void; contextPaths?: string[]; label?: string }) {
  const { scope } = useAppRuntime();
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const [q, setQ] = useState('');
  const root = useMemo(() => browsableScope(scope), [scope]);
  // Names the page state does not hold: they exist per row, per form, per item.
  const context = useMemo(() => contextPaths.filter((p) => !/^(vars|queries|route)(\.|$)/.test(p)), [contextPaths]);
  const hits = useMemo(() => {
    const words = q.trim().toLowerCase();
    if (!words) return null;
    return [
      ...context.filter((p) => p.toLowerCase().includes(words)).map((p): ScopeEntry => ({ path: p, name: p, type: 'here', preview: '', expandable: false })),
      ...flattenScope(root).filter((e) => e.path.toLowerCase().includes(words)).map((e) => ({ ...e, name: e.path })),
    ].slice(0, 80);
  }, [q, root, context]);
  const pick = (path: string) => { onPick(path); setAnchor(null); setQ(''); };

  return (
    <>
      <Tooltip title={label}>
        <IconButton size="small" aria-label={label} onClick={(e) => setAnchor(e.currentTarget)}><DataObjectIcon fontSize="small" /></IconButton>
      </Tooltip>
      <Popover open={!!anchor} anchorEl={anchor} onClose={() => setAnchor(null)} anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }} slotProps={{ paper: { sx: { width: 440, maxHeight: 460, display: 'flex', flexDirection: 'column' } } }}>
        <Box sx={{ p: 1, borderBottom: 1, borderColor: 'divider' }}>
          <TextField size="small" fullWidth placeholder="Search variables and query data" value={q} onChange={(e) => setQ(e.target.value)}
            inputProps={{ 'aria-label': 'Search data' }}
            InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon fontSize="small" /></InputAdornment> }} />
        </Box>
        <Box sx={{ overflowY: 'auto', flex: 1 }}>
          {hits ? (
            <List dense disablePadding>
              {hits.map((e) => <Row key={e.path} e={e} depth={0} open={false} onToggle={() => {}} onPick={pick} />)}
              {hits.length === 0 && <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>Nothing matches.</Typography>}
            </List>
          ) : (
            <List dense disablePadding>
              {context.length > 0 && (
                <>
                  <Typography variant="overline" color="text.secondary" sx={{ px: 1.5 }}>Available here</Typography>
                  {context.map((p) => <Row key={p} e={{ path: p, name: p, type: 'here', preview: '', expandable: false }} depth={0} open={false} onToggle={() => {}} onPick={pick} />)}
                </>
              )}
              <Typography variant="overline" color="text.secondary" sx={{ px: 1.5 }}>Page data (live)</Typography>
              <Tree root={root} base="" depth={0} onPick={pick} startOpen />
            </List>
          )}
        </Box>
        <Stack sx={{ px: 1.5, py: 0.75, borderTop: 1, borderColor: 'divider' }}>
          <Typography variant="caption" color="text.secondary">Values are what the page holds right now. A list shows its first item as 0.</Typography>
        </Stack>
      </Popover>
    </>
  );
}
