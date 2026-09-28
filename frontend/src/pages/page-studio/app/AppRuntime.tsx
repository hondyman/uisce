import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useQueries, useQueryClient } from '@tanstack/react-query';
import {
  Alert, Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, LinearProgress, Snackbar, Stack, Typography,
} from '@mui/material';
import { CatalogErrorAlert } from '../../../features/message-catalog/parts';
import { getOperation, missingParams } from '../../../studio-core/operations/registry';
import '../../../studio-core/registerDomains';
import type { Action, FormSpec, PageAppModel } from './appModel';
import { evaluateCondition, useCondition } from './conditions';
import { resolve, resolveAll, text, type Scope } from './bindings';
import { FormFields, requiredFilled } from './formFields';

export interface QueryState {
  data: unknown;
  isLoading: boolean;
  isFetching: boolean;
  error: unknown;
}

interface AppRuntimeValue {
  /** Page scope for bindings and conditions: {vars, queries, route}. */
  scope: Scope;
  mode: 'design' | 'preview';
  setVariable: (name: string, value: unknown) => void;
  /** Runs actions in order; `local` adds row/rowState/event/form/result to the scope. */
  runActions: (actions: Action[] | undefined, local?: Scope) => Promise<void>;
}

const EMPTY_SCOPE: Scope = { vars: {}, queries: {}, route: {} };

const AppRuntimeContext = createContext<AppRuntimeValue>({
  scope: EMPTY_SCOPE,
  mode: 'preview',
  setVariable: () => {},
  runActions: async () => {},
});

export const useAppRuntime = () => useContext(AppRuntimeContext);

const isEmpty = (v: unknown) => v === undefined || v === null || v === '';

class Cancelled extends Error {}

type Pending =
  | { kind: 'form'; spec: FormSpec; scope: Scope; resolve: (v: Record<string, unknown>) => Promise<unknown>; cancel: () => void }
  | { kind: 'confirm'; title: string; text: string; confirmLabel: string; resolve: () => void; cancel: () => void };

/**
 * Provides a page's application model at runtime - the same provider runs
 * in the studio (design canvas + preview) and for viewers (PageBrowser), so
 * what an author sees is what a user gets.
 */
export const AppRuntimeProvider: React.FC<{
  app?: PageAppModel;
  mode?: 'design' | 'preview';
  route?: Record<string, unknown>;
  children: React.ReactNode;
}> = ({ app, mode = 'preview', route, children }) => {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const variables = app?.variables ?? [];
  const queries = app?.queries ?? [];

  const [vars, setVars] = useState<Record<string, unknown>>(() => {
    const init: Record<string, unknown> = {};
    for (const v of variables) {
      const fromUrl = v.url ? searchParams.get(v.name) : null;
      init[v.name] = fromUrl ?? v.default ?? null;
    }
    return init;
  });
  // Variables added in the editor after mount get their defaults.
  useEffect(() => {
    setVars((prev) => {
      let changed = false;
      const next = { ...prev };
      for (const v of variables) if (!(v.name in next)) { next[v.name] = v.default ?? null; changed = true; }
      return changed ? next : prev;
    });
  }, [variables]);

  const urlVars = useMemo(() => variables.filter((v) => v.url).map((v) => v.name), [variables]);
  useEffect(() => {
    if (mode === 'design' || urlVars.length === 0) return;
    const next = new URLSearchParams(searchParams);
    let changed = false;
    for (const n of urlVars) {
      const v = vars[n];
      const s = isEmpty(v) ? null : String(v);
      if (next.get(n) !== s) { changed = true; if (s === null) next.delete(n); else next.set(n, s); }
    }
    if (changed) setSearchParams(next, { replace: true });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [vars, urlVars, mode]);

  const setVariable = useCallback((name: string, value: unknown) => {
    setVars((prev) => (prev[name] === value ? prev : { ...prev, [name]: value }));
  }, []);

  // Queries: params resolve against vars (and earlier queries' data); a
  // query waits while a required param is empty or enabledWhen fails.
  const baseScope = useMemo<Scope>(() => ({ vars, route: route ?? {} }), [vars, route]);
  const [enabled, setEnabled] = useState<Record<string, boolean>>({});
  const prevResults = useRef<Record<string, QueryState>>({});
  const resolved = queries.map((q) => {
    const op = getOperation(q.operation);
    const params = resolveAll(q.params, { ...baseScope, queries: prevResults.current });
    const ready = !!op && missingParams(op, params).length === 0 && (q.enabledWhen ? enabled[q.id] === true : true);
    return { q, op, params, ready };
  });
  const results = useQueries({
    queries: resolved.map(({ q, op, params, ready }) => ({
      queryKey: [op?.domain ?? 'studio', 'studio', q.operation, params],
      queryFn: () => op!.run(params),
      enabled: ready,
      // useQueries does not hand placeholderData the previous key's data, so keep it here.
      placeholderData: q.keepPrevious ? () => prevResults.current[q.id]?.data : undefined,
    })),
  });
  const queryStates = useMemo(() => {
    const out: Record<string, QueryState> = {};
    resolved.forEach(({ q, op }, i) => {
      const r = results[i];
      out[q.id] = {
        data: r?.data,
        isLoading: !!r && r.isLoading,
        isFetching: !!r && r.isFetching,
        error: op ? r?.error : new Error(`Unknown operation ${q.operation}`),
      };
    });
    return out;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [results.map((r) => r.dataUpdatedAt + ':' + r.errorUpdatedAt + ':' + r.fetchStatus + ':' + r.status).join('|'), queries]);
  prevResults.current = queryStates;

  const scope = useMemo<Scope>(() => ({ ...baseScope, queries: queryStates }), [baseScope, queryStates]);

  const enabledKey = JSON.stringify(queries.map((q) => q.enabledWhen ?? null));
  useEffect(() => {
    let live = true;
    const gated = queries.filter((q) => q.enabledWhen);
    if (gated.length === 0) return;
    void Promise.all(gated.map(async (q) => [q.id, await evaluateCondition(q.enabledWhen, scope)] as const)).then((pairs) => {
      if (!live) return;
      setEnabled((prev) => {
        const next = { ...prev };
        let changed = false;
        for (const [id, v] of pairs) if (next[id] !== v) { next[id] = v; changed = true; }
        return changed ? next : prev;
      });
    });
    return () => { live = false; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [scope, enabledKey]);

  // Seed variables from query results (first entity, latest date...).
  useEffect(() => {
    for (const v of variables) {
      if (!v.initFrom || !isEmpty(vars[v.name])) continue;
      const got = resolve(`{{queries.${v.initFrom.query}.data.${v.initFrom.path}}}`, scope);
      if (!isEmpty(got)) setVariable(v.name, got);
    }
  }, [variables, vars, scope, setVariable]);

  // Dialog host for confirm/form actions, and a snackbar for notices.
  const [pending, setPending] = useState<Pending | null>(null);
  const [notice, setNotice] = useState<{ severity: 'success' | 'info' | 'warning' | 'error'; text: string } | null>(null);

  const scopeRef = useRef(scope);
  scopeRef.current = scope;

  const runActions = useCallback(async (actions: Action[] | undefined, local: Scope = {}) => {
    let s: Scope = { ...scopeRef.current, ...local };
    try {
      for (const a of actions ?? []) {
        if (a.kind === 'setVariable') {
          const v = resolve(a.value ?? null, s);
          setVariable(a.name, v === undefined ? null : v);
          s = { ...s, vars: { ...(s.vars as Record<string, unknown>), [a.name]: v } };
        } else if (a.kind === 'navigate') {
          if (mode === 'design') continue;
          navigate(String(resolve(a.to, s)));
        } else if (a.kind === 'notify') {
          setNotice({ severity: a.severity, text: text(a.text, s) });
        } else if (a.kind === 'runOperation') {
          const op = getOperation(a.operation);
          if (!op) throw new Error(`Unknown operation ${a.operation}`);
          const exec = async (form?: Record<string, unknown>) => {
            const sc = form ? { ...s, form } : s;
            const pv = a.progressVariable;
            let result: unknown;
            try {
              result = await op.run(resolveAll(a.params, sc), { progress: (msg) => { if (pv) setVariable(pv, msg); } });
            } finally {
              if (pv) setVariable(pv, null);
            }
            for (const prefix of op.invalidates ?? [[op.domain]]) await qc.invalidateQueries({ queryKey: prefix });
            return { ...sc, result };
          };
          if (a.confirm) {
            const c = a.confirm;
            await new Promise<void>((res, rej) => setPending({
              kind: 'confirm', title: text(c.title, s), text: text(c.text, s), confirmLabel: text(c.confirmLabel ?? 'studioApp.ok', s),
              resolve: () => { setPending(null); res(); }, cancel: () => { setPending(null); rej(new Cancelled()); },
            }));
          }
          if (a.form) {
            const spec = a.form;
            s = await new Promise<Scope>((res, rej) => setPending({
              kind: 'form', spec, scope: s,
              // The dialog stays open on failure and shows why.
              resolve: async (values) => {
                try { const next = await exec(values); setPending(null); res(next); return null; } catch (err) { return err ?? new Error('failed'); }
              },
              cancel: () => { setPending(null); rej(new Cancelled()); },
            }));
          } else {
            s = await exec();
          }
          if (a.successMessage) setNotice({ severity: 'success', text: text(a.successMessage, s) });
          if (a.onSuccess?.length) await runActionsInner(a.onSuccess, s);
        }
      }
    } catch (err) {
      if (err instanceof Cancelled) return;
      setNotice({ severity: 'error', text: err instanceof Error ? err.message : String(err) });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mode, navigate, qc, setVariable]);
  // onSuccess chains keep the scope they were given (result, form, row).
  const runActionsInner = (actions: Action[], s: Scope) => runActionsRef.current(actions, s);
  const runActionsRef = useRef(runActions);
  runActionsRef.current = runActions;

  const value = useMemo(() => ({ scope, mode, setVariable, runActions }), [scope, mode, setVariable, runActions]);

  return (
    <AppRuntimeContext.Provider value={value}>
      {children}
      {pending?.kind === 'confirm' && (
        <Dialog open onClose={pending.cancel} fullWidth maxWidth="xs">
          <DialogTitle>{pending.title}</DialogTitle>
          {pending.text && <DialogContent dividers><Typography variant="body2">{pending.text}</Typography></DialogContent>}
          <DialogActions>
            <Button onClick={pending.cancel}>{text('studioApp.cancel', {})}</Button>
            <Button variant="contained" onClick={pending.resolve}>{pending.confirmLabel}</Button>
          </DialogActions>
        </Dialog>
      )}
      {pending?.kind === 'form' && <FormDialog pending={pending} />}
      <Snackbar open={!!notice} autoHideDuration={4000} onClose={() => setNotice(null)} anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}>
        {notice ? <Alert severity={notice.severity} variant="filled" onClose={() => setNotice(null)}>{notice.text}</Alert> : undefined}
      </Snackbar>
    </AppRuntimeContext.Provider>
  );
};

function FormDialog({ pending }: { pending: Extract<Pending, { kind: 'form' }> }) {
  const { spec, scope } = pending;
  const [values, setValues] = useState<Record<string, unknown>>(() =>
    Object.fromEntries(spec.fields.map((f) => [f.name, resolve(f.default ?? (f.kind === 'switch' ? false : ''), scope) ?? ''])));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const showNotice = useCondition(spec.notice?.visibleWhen, scope, false);
  const ready = requiredFilled(spec.fields, values);
  const submit = async () => {
    setBusy(true);
    setError(null);
    const trimmed = Object.fromEntries(Object.entries(values).map(([k, v]) => [k, typeof v === 'string' ? v.trim() : v]));
    const err = await pending.resolve(trimmed);
    setBusy(false);
    if (err) setError(err);
  };
  return (
    <Dialog open onClose={pending.cancel} fullWidth maxWidth="sm">
      <DialogTitle>{text(spec.title, scope)}</DialogTitle>
      <DialogContent dividers>
        <Stack spacing={2}>
          {spec.intro && <Typography variant="body2">{text(spec.intro, scope)}</Typography>}
          {spec.notice && showNotice && <Alert severity={(['info', 'warning', 'error', 'success'].includes(String(resolve(spec.notice.severity, scope))) ? String(resolve(spec.notice.severity, scope)) : 'info') as 'info'}>{text(spec.notice.text, scope)}</Alert>}
          <FormFields fields={spec.fields} values={values} scope={scope} onChange={(name, v) => setValues((p) => ({ ...p, [name]: v }))} />
          {busy && <LinearProgress />}
          {!!error && <Box><CatalogErrorAlert error={error} /></Box>}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={pending.cancel}>{text('studioApp.cancel', {})}</Button>
        <Button variant="contained" color={spec.submitColor ?? 'primary'} disabled={!ready || busy} onClick={() => void submit()}>
          {text(spec.submitLabel ?? 'studioApp.ok', scope)}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
