import React from 'react';
import { Alert, Box, Chip, Stack, TextField, Tooltip, Typography } from '@mui/material';
import type { CorePageDefinition, PageTab } from '../../../types/pageStudio';
import { getOperation, listOperations } from '../../../studio-core/operations/registry';
import { listDomainComponents } from '../../../studio-core/components/registry';
import type { PageAppModel, PageQuery, PageVariable } from './appModel';
import { BindingField, ConditionEditor, JsonField, ListEditor, Section, SelectField, SwitchField, scopePaths } from './editors';

const parseDefault = (s: string): unknown => {
  if (s === '') return undefined;
  try { return JSON.parse(s); } catch { return s; }
};

/**
 * The page's application model: its state (variables), its data (queries
 * against registered operations), its tabs' state and counts, and its
 * chrome. What the hand-built consoles keep in useState/useQuery, a studio
 * page declares here.
 */
export default function AppModelPanel({ draft, setDraft }: { draft: CorePageDefinition; setDraft: React.Dispatch<React.SetStateAction<CorePageDefinition>> }) {
  const app: PageAppModel = draft.app ?? {};
  const setApp = (patch: Partial<PageAppModel>) => setDraft((prev) => ({ ...prev, app: { ...(prev.app ?? {}), ...patch } }));
  const paths = scopePaths(draft);
  const tabs: PageTab[] = draft.tabs ?? [];
  const setTab = (id: string, patch: Partial<PageTab>) => setDraft((prev) => ({ ...prev, tabs: (prev.tabs ?? []).map((t) => (t.id === id ? { ...t, ...patch } : t)) }));

  return (
    <Box sx={{ p: 2 }}>
      {!draft.app && (
        <Alert severity="info" sx={{ mb: 1 }}>
          Give this page state, data from registered operations, and actions - what a hand-built console does with code.
        </Alert>
      )}

      <Section title="Variables">
        <ListEditor<PageVariable> items={app.variables ?? []} onChange={(v) => setApp({ variables: v })} addLabel="Add variable"
          create={() => ({ name: `var${(app.variables?.length ?? 0) + 1}` })}
          render={(v, set) => (
            <>
              <TextField size="small" label="Name" value={v.name} onChange={(e) => set({ ...v, name: e.target.value.replace(/[^\w]/g, '') })} />
              <TextField size="small" label="Default (JSON or text)" value={v.default === undefined ? '' : typeof v.default === 'string' ? v.default : JSON.stringify(v.default)}
                onChange={(e) => set({ ...v, default: parseDefault(e.target.value) })} />
              <SwitchField label="Keep in the URL (linkable)" checked={!!v.url} onChange={(x) => set({ ...v, url: x || undefined })} />
              <Stack direction="row" spacing={1}>
                <SelectField label="Seed from query" value={v.initFrom?.query} allowEmpty="No"
                  options={(app.queries ?? []).map((q) => ({ value: q.id, label: q.id }))}
                  onChange={(q) => set({ ...v, initFrom: q ? { query: q, path: v.initFrom?.path ?? '0.id' } : undefined })} />
                {v.initFrom && <TextField size="small" label="Path" value={v.initFrom.path} onChange={(e) => set({ ...v, initFrom: { ...v.initFrom!, path: e.target.value } })} />}
              </Stack>
            </>
          )} />
      </Section>

      <Section title="Queries">
        <ListEditor<PageQuery> items={app.queries ?? []} onChange={(v) => setApp({ queries: v })} addLabel="Add query"
          create={() => ({ id: `query${(app.queries?.length ?? 0) + 1}`, operation: '' })}
          render={(q, set) => {
            const op = getOperation(q.operation);
            return (
              <>
                <TextField size="small" label="Id ({{queries.<id>.data}})" value={q.id} onChange={(e) => set({ ...q, id: e.target.value.replace(/[^\w]/g, '') })} />
                <SelectField label="Operation" value={q.operation} options={listOperations('query').map((o) => ({ value: o.id, label: `${o.label} (${o.id})` }))}
                  onChange={(v) => set({ ...q, operation: v ?? '' })} />
                {op?.description && <Typography variant="caption" color="text.secondary">{op.description}</Typography>}
                {op?.params.map((p) => (
                  <BindingField key={p.name} label={`${p.label ?? p.name}${p.required ? ' *' : ''}`} value={q.params?.[p.name]} paths={paths}
                    helperText={p.required ? 'Waits until this has a value' : p.description} onChange={(v) => set({ ...q, params: { ...q.params, [p.name]: v } })} />
                ))}
                <SwitchField label="Keep previous result while filtering" checked={!!q.keepPrevious} onChange={(x) => set({ ...q, keepPrevious: x || undefined })} />
                <ConditionEditor label="Run only when" value={q.enabledWhen} onChange={(v) => set({ ...q, enabledWhen: v })} paths={paths} />
              </>
            );
          }} />
      </Section>

      <Section title="Tabs">
        <SelectField label="Tab state variable (lets actions switch tabs)" value={app.tabVariable} allowEmpty="None"
          options={(app.variables ?? []).map((v) => ({ value: v.name, label: v.name }))} onChange={(v) => setApp({ tabVariable: v })} />
        {tabs.length === 0 && <Typography variant="caption" color="text.secondary">Add a second tab on the canvas to give tabs counts and conditions.</Typography>}
        {tabs.map((t) => (
          <Box key={t.id} sx={{ border: 1, borderColor: 'divider', borderRadius: 1, p: 1.25 }}>
            <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 1 }}>
              <Typography variant="body2" fontWeight={600}>{t.label}</Typography>
              <Tooltip title="Actions set the tab variable to this id"><Chip size="small" variant="outlined" label={t.id} /></Tooltip>
            </Stack>
            <Stack spacing={1}>
              <BindingField label="Count badge" value={t.badge} onChange={(v) => setTab(t.id, { badge: (v as string) || undefined })} paths={paths} />
              <ConditionEditor label="Show tab when" value={t.visibleWhen} onChange={(v) => setTab(t.id, { visibleWhen: v })} paths={paths} />
            </Stack>
          </Box>
        ))}
      </Section>

      <Section title="Page">
        <SwitchField label="Own header (hide title and /slug)" checked={app.chrome === 'none'} onChange={(v) => setApp({ chrome: v ? 'none' : undefined })} />
        <TextField size="small" type="number" label="Max content width (px)" value={app.surface?.maxWidth ?? ''}
          onChange={(e) => setApp({ surface: { ...app.surface, maxWidth: e.target.value ? Number(e.target.value) : undefined } })} />
      </Section>

      <Section title="Available operations">
        {listOperations().map((o) => (
          <Tooltip key={o.id} title={o.description ?? ''} placement="right">
            <Box>
              <Typography variant="body2" fontFamily="monospace">{o.id}</Typography>
              <Typography variant="caption" color="text.secondary">{o.kind} · {o.label}</Typography>
            </Box>
          </Tooltip>
        ))}
      </Section>
      <Section title="Domain components">
        {listDomainComponents().map((d) => (
          <Tooltip key={d.id} title={d.description ?? ''} placement="right">
            <Box>
              <Typography variant="body2" fontFamily="monospace">{d.id}</Typography>
              <Typography variant="caption" color="text.secondary">{d.label}{d.overlay ? ' · overlay' : ''}</Typography>
            </Box>
          </Tooltip>
        ))}
      </Section>
      <Section title="Whole model (JSON)">
        <JsonField label="app" value={draft.app ?? null} minRows={6} onChange={(v) => setDraft((prev) => ({ ...prev, app: (v || undefined) as PageAppModel | undefined }))} />
      </Section>
    </Box>
  );
}
