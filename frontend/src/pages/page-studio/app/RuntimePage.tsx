import React, { useEffect, useState } from 'react';
import { Alert, Badge, Box, Tab, Tabs } from '@mui/material';
import type { PageAppModel, ConditionNode } from './appModel';
import { AppRuntimeProvider, useAppRuntime } from './AppRuntime';
import { evaluateCondition } from './conditions';
import { resolve, text, type Scope } from './bindings';
import RenderLayoutTree from '../RenderLayoutTree';
import PageBody from '../PageBody';

interface RuntimeLayout { root: string; nodes: Record<string, { id: string; type: string; children?: string[]; props?: Record<string, unknown>; style?: Record<string, string> }> }
export interface RuntimeTab { id: string; label: string; layout: RuntimeLayout; badge?: string; visibleWhen?: ConditionNode }

export interface RuntimePageProps {
  name: string;
  slug: string;
  tabs: RuntimeTab[];
  filterBar?: RuntimeLayout;
  components: Record<string, { id: string; type: string; props?: Record<string, unknown>; style?: Record<string, string> }>;
  dataSources: unknown[];
  tenantId: string;
  app?: PageAppModel;
  /** Rendered above the title (e.g. PageBrowser's back link). */
  before?: React.ReactNode;
  /** Page title override (PageBrowser's "New …" create mode). */
  title?: string;
  /** Adds the page's own padding (standalone); off when a frame already pads. */
  padded?: boolean;
}

/** Which tabs show now: tab conditions go through the rule engine like every other page condition. */
export function useVisibleTabs<T extends { id: string; visibleWhen?: ConditionNode }>(tabs: T[], scope: Scope): T[] {
  const [visible, setVisible] = useState<Record<string, boolean>>({});
  const key = JSON.stringify(tabs.map((t) => [t.id, t.visibleWhen ?? null]));
  useEffect(() => {
    let live = true;
    void Promise.all(tabs.filter((t) => t.visibleWhen).map(async (t) => [t.id, await evaluateCondition(t.visibleWhen, scope)] as const))
      .then((pairs) => { if (live) setVisible(Object.fromEntries(pairs)); });
    return () => { live = false; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, scope]);
  return tabs.filter((t) => !t.visibleWhen || visible[t.id]);
}

/** A tab's label with its live count. */
export function TabLabel({ label, badge, scope }: { label: string; badge?: string; scope: Scope }) {
  const n = badge ? Number(resolve(badge, scope)) || 0 : 0;
  const shown = text(label, scope);
  if (!badge) return <>{shown}</>;
  return <Badge color="warning" badgeContent={n} sx={{ pr: n ? 1.5 : 0 }}>{shown}</Badge>;
}

function RuntimeBody(props: RuntimePageProps) {
  const { scope, setVariable } = useAppRuntime();
  const { app, tabs, components, dataSources, tenantId, filterBar } = props;
  const visibleTabs = useVisibleTabs(tabs, scope);
  const [localTab, setLocalTab] = useState<string | null>(null);
  const wanted = app?.tabVariable ? (scope.vars as Record<string, unknown>)[app.tabVariable] as string | null : localTab;
  // A hidden or unknown tab falls back to the first one showing.
  const active = visibleTabs.find((t) => t.id === wanted) ?? visibleTabs[0];
  const choose = (id: string) => (app?.tabVariable ? setVariable(app.tabVariable, id) : setLocalTab(id));

  const content = (
    <>
      {filterBar?.root && (
        <Box sx={{ mb: 2 }}>
          <RenderLayoutTree nodeId={filterBar.root} nodes={filterBar.nodes} components={components} dataSources={dataSources} tenantId={tenantId} />
        </Box>
      )}
      {visibleTabs.length > 1 && active && (
        <Tabs value={active.id} onChange={(_, v) => choose(v)} variant="scrollable" allowScrollButtonsMobile sx={{ mb: 2, borderBottom: 1, borderColor: 'divider' }}>
          {visibleTabs.map((t) => (
            <Tab key={t.id} value={t.id} sx={{ textTransform: app ? undefined : 'none' }} label={<TabLabel label={t.label} badge={t.badge} scope={scope} />} />
          ))}
        </Tabs>
      )}
      {active?.layout?.root ? (
        <RenderLayoutTree nodeId={active.layout.root} nodes={active.layout.nodes} components={components} dataSources={dataSources} tenantId={tenantId} />
      ) : tabs.length === 0 || !tabs[0]?.layout?.root ? (
        <Alert severity="info">This page has no components yet.</Alert>
      ) : null}
    </>
  );

  const pad = props.padded ? (app?.surface?.padding ?? 3) : 0;
  return (
    // Own themed surface, as the hand-built consoles do: the shell canvas is dark whatever the MUI mode.
    <Box sx={{ bgcolor: app ? 'background.default' : undefined, color: 'text.primary', minHeight: '100%' }}>
      <Box sx={{ p: pad, maxWidth: app?.surface?.maxWidth, mx: app?.surface?.maxWidth ? 'auto' : undefined }}>
        {props.before}
        {app?.chrome === 'none' ? content : <PageBody name={props.title ?? props.name} slug={props.slug}>{content}</PageBody>}
      </Box>
    </Box>
  );
}

/** A page as a viewer gets it: app runtime, filter bar, tabs, the active layout. */
export default function RuntimePage(props: RuntimePageProps) {
  return (
    <AppRuntimeProvider app={props.app} mode="preview">
      <RuntimeBody {...props} />
    </AppRuntimeProvider>
  );
}
