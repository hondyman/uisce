import React, { useState } from 'react';
import { Box, Paper, Stack, Tab, Tabs, TextField, Typography } from '@mui/material';
import type { CorePageDefinition, LayoutNode, PageLayout } from '../../../types/pageStudio';
import type { ContainerButton, OverlayNodeProps, TabSetNodeProps, TextSpec } from './appModel';
import {
  ActionsEditor, BindingField, ConditionEditor, JsonField, ListEditor, Section, SelectField, TextSpecField, scopePaths,
} from './editors';
import { nodesOf } from '../layoutNodes';

/**
 * Properties of a Drawer, Dialog or TabSet layout node. Opening is a
 * condition over page state (usually a variable an action sets); closing
 * runs actions (usually clearing it). A TabSet's tabs map to its child
 * bodies one to one - adding a tab adds its (empty) body.
 */
export default function ContainerInspector({ node, layout, onLayoutChange, draft }: {
  node: LayoutNode;
  layout: PageLayout;
  onLayoutChange: (updater: (prev: PageLayout) => PageLayout) => void;
  draft: CorePageDefinition;
}) {
  const [view, setView] = useState<'form' | 'json'>('form');
  const paths = scopePaths(draft);
  const vars = (draft.app?.variables ?? []).map((v) => ({ value: v.name, label: v.name }));
  const props = (node.props ?? {}) as Record<string, unknown>;
  const setNode = (next: Partial<LayoutNode>) => onLayoutChange((prev) => ({
    ...prev, nodes: { ...nodesOf(prev), [node.id]: { ...nodesOf(prev)[node.id], ...next } },
  }));
  const setProps = (patch: Record<string, unknown>) => setNode({ props: { ...props, ...patch } });

  let body: React.ReactNode;
  if (node.type === 'TabSet') {
    const p = props as TabSetNodeProps;
    const tabs = p.tabs ?? [];
    const setTabs = (next: NonNullable<TabSetNodeProps['tabs']>) => onLayoutChange((prev) => {
      const current = nodesOf(prev)[node.id];
      const oldChildren = current.children ?? [];
      const nodes = { ...nodesOf(prev) };
      // Keep each surviving tab's body; give new tabs an empty Column.
      const children = next.map((t) => {
        const i = tabs.findIndex((x) => x.id === t.id);
        if (i >= 0 && oldChildren[i]) return oldChildren[i];
        const id = `${node.id}_${t.id}`;
        nodes[id] = nodes[id] ?? { id, type: 'Column', children: [] };
        return id;
      });
      for (const id of oldChildren) if (!children.includes(id)) delete nodes[id];
      nodes[node.id] = { ...current, children, props: { ...current.props, tabs: next } };
      return { ...prev, nodes };
    });
    body = (
      <>
        <Section title="Tabs">
          <ListEditor items={tabs} onChange={setTabs} addLabel="Add tab"
            create={() => ({ id: `tab${Date.now().toString(36).slice(-4)}`, label: 'New tab' })}
            render={(t, up) => (
              <>
                <TextField size="small" label="Id" value={t.id} disabled helperText="The value the tab variable holds" />
                <TextSpecField label="Label" value={t.label} onChange={(v) => up({ ...t, label: v })} paths={paths} />
                <BindingField label="Badge (a count)" value={t.badge} onChange={(v) => up({ ...t, badge: v || undefined })} paths={paths} />
                <ConditionEditor label="Show when" value={t.visibleWhen} onChange={(v) => up({ ...t, visibleWhen: v })} paths={paths} />
              </>
            )} />
        </Section>
        <Section title="State">
          <SelectField label="Active tab variable (optional)" value={p.variable} options={vars} allowEmpty="Local to this tab set"
            onChange={(v) => setProps({ variable: v || undefined })} />
          <Typography variant="caption" color="text.secondary">A variable lets actions switch tabs (and keeps the tab in the URL when the variable does).</Typography>
        </Section>
      </>
    );
  } else {
    const p = props as OverlayNodeProps;
    body = (
      <>
        <Section title={node.type}>
          <TextSpecField label="Title" value={p.title} onChange={(v) => setProps({ title: v })} paths={paths} />
          <TextSpecField label="Subtitle" value={p.subtitle} onChange={(v) => setProps({ subtitle: v || undefined })} paths={paths} />
          {node.type === 'Drawer' ? (
            <Stack direction="row" spacing={1}>
              <SelectField label="Side" value={p.anchor ?? 'right'} options={[{ value: 'right', label: 'Right' }, { value: 'left', label: 'Left' }]} onChange={(v) => setProps({ anchor: v })} />
              <BindingField label="Width (px, or a binding)" value={p.width ?? 720} paths={paths}
                onChange={(v) => setProps({ width: typeof v === 'string' && /^\d+$/.test(v) ? Number(v) : v })} />
            </Stack>
          ) : (
            <SelectField label="Width" value={p.maxWidth ?? 'sm'} options={['xs', 'sm', 'md', 'lg', 'xl'].map((v) => ({ value: v as 'sm', label: v }))} onChange={(v) => setProps({ maxWidth: v })} />
          )}
        </Section>
        <Section title="Opening and closing">
          <ConditionEditor label="Open when" value={p.openWhen} onChange={(v) => setProps({ openWhen: v })} paths={paths} />
          <Typography variant="caption" color="text.secondary">Usually a variable an action sets, e.g. vars.goldenId is not empty.</Typography>
          <ActionsEditor label="On close (usually clears that variable)" value={p.onClose} onChange={(v) => setProps({ onClose: v })} draft={draft} paths={paths} />
        </Section>
        <Section title="Footer buttons">
          <ListEditor items={p.buttons ?? []} onChange={(v) => setProps({ buttons: v.length ? v : undefined })} addLabel="Add button"
            create={(): ContainerButton => ({ label: 'OK' as TextSpec, variant: 'contained', onClick: [] })}
            render={(b, up) => (
              <>
                <TextSpecField label="Label" value={b.label} onChange={(v) => up({ ...b, label: v })} paths={paths} />
                <SelectField label="Style" value={b.variant ?? 'text'} options={['contained', 'outlined', 'text'].map((v) => ({ value: v as 'text', label: v }))} onChange={(v) => up({ ...b, variant: v })} />
                <SelectField label="Colour" value={b.color ?? 'primary'} options={['primary', 'secondary', 'inherit', 'error'].map((v) => ({ value: v as 'primary', label: v }))} onChange={(v) => up({ ...b, color: v })} />
                <ActionsEditor label="On click" value={b.onClick} onChange={(v) => up({ ...b, onClick: v ?? [] })} draft={draft} paths={paths} />
                <ConditionEditor label="Disabled when" value={b.disabledWhen} onChange={(v) => up({ ...b, disabledWhen: v })} paths={paths} />
                <ConditionEditor label="Show when" value={b.visibleWhen} onChange={(v) => up({ ...b, visibleWhen: v })} paths={paths} />
              </>
            )} />
        </Section>
      </>
    );
  }

  return (
    <Paper elevation={0} sx={{ height: '100%', bgcolor: 'background.paper', overflowY: 'auto' }}>
      <Box sx={{ p: 2 }}>
        <Typography variant="overline" color="text.secondary" fontWeight="bold">{node.type}</Typography>
        <Typography variant="caption" color="text.secondary" component="div" fontFamily="monospace">{node.id}</Typography>
        <Tabs value={view} onChange={(_, v) => setView(v)} sx={{ minHeight: 32, mt: 1, '& .MuiTab-root': { minHeight: 32, py: 0 } }}>
          <Tab value="form" label="Properties" />
          <Tab value="json" label="JSON" />
        </Tabs>
        {view === 'json' ? (
          <Box sx={{ mt: 2 }}>
            <JsonField label={node.type} value={props} minRows={16} onChange={(v) => setNode({ props: (v ?? {}) as Record<string, unknown> })} />
          </Box>
        ) : body}
        {layout.root === node.id && <Typography variant="caption" color="warning.main">This is the tab's root node.</Typography>}
      </Box>
    </Paper>
  );
}
