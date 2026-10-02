import React, { useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Alert, Box, Tab, Tabs } from '@mui/material';
import type { PageAppModel, ConditionNode } from './appModel';
import { AppRuntimeProvider, useAppRuntime } from './AppRuntime';
import RenderLayoutTree from '../RenderLayoutTree';
import PageBody from '../PageBody';

// root/nodes are optional because a grid-kind (or empty) page layout has neither; RuntimeBody
// below already treats a missing root as "no components yet".
interface RuntimeLayout { root?: string; nodes?: Record<string, { id: string; type: string; children?: string[]; props?: Record<string, unknown>; style?: Record<string, string> }> }
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

export { TabLabel, useVisibleTabs } from './tabs';
import { TabLabel, useVisibleTabs } from './tabs';

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
          <RenderLayoutTree nodeId={filterBar.root} nodes={filterBar.nodes ?? {}} components={components} dataSources={dataSources} tenantId={tenantId} />
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
        <RenderLayoutTree nodeId={active.layout.root} nodes={active.layout.nodes ?? {}} components={components} dataSources={dataSources} tenantId={tenantId} />
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
  // The route's parameters ({{route.id}} on /data/pipelines/:id).
  const params = useParams();
  const paramsKey = JSON.stringify(params);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const route = useMemo(() => params, [paramsKey]);
  return (
    <AppRuntimeProvider app={props.app} mode="preview" route={route}>
      <RuntimeBody {...props} />
    </AppRuntimeProvider>
  );
}
