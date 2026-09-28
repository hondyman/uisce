import React, { useState } from 'react';
import {
  Box, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle, Drawer, IconButton, Stack, Tab, Tabs, Typography,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import type { ContainerButton, OverlayNodeProps, TabSetNodeProps } from './appModel';
import { resolve, text, type Scope } from './bindings';
import { useCondition } from './conditions';
import { useAppRuntime } from './AppRuntime';
import { TabLabel, useVisibleTabs } from './tabs';

/**
 * Overlay and tab containers: layout nodes that hold any widgets or other
 * containers (a record drawer with its own tabs, grids and forms; a dialog
 * with a form and footer buttons). Open/close and the active tab are page
 * state - conditions evaluated by the rule engine and page variables - so a
 * drawer is opened by an action setting a variable, exactly like the
 * hand-built consoles. `renderChild` renders a child node or widget with
 * the caller's renderer (runtime or design canvas).
 */

export const OVERLAY_TYPES = ['Drawer', 'Dialog'] as const;
export const CONTAINER_TYPES = ['Drawer', 'Dialog', 'TabSet'] as const;
export const isContainerType = (t: string) => (CONTAINER_TYPES as readonly string[]).includes(t);

export const DEFAULT_CONTAINER_PROPS: Record<string, Record<string, unknown>> = {
  Drawer: { title: 'Details', anchor: 'right', width: 720, openWhen: { type: 'condition', field: 'vars.open', operator: 'is_true' }, onClose: [] },
  Dialog: { title: 'Dialog', maxWidth: 'sm', openWhen: { type: 'condition', field: 'vars.open', operator: 'is_true' }, onClose: [], buttons: [] },
  TabSet: { tabs: [{ id: 'tab1', label: 'Tab 1' }, { id: 'tab2', label: 'Tab 2' }] },
};

function FooterButton({ b, scope }: { b: ContainerButton; scope: Scope }) {
  const { runActions } = useAppRuntime();
  const shown = useCondition(b.visibleWhen, scope, true);
  const disabled = useCondition(b.disabledWhen, scope, false);
  if (b.visibleWhen && !shown) return null;
  return (
    <Button variant={b.variant ?? 'text'} color={b.color ?? 'primary'} disabled={!!b.disabledWhen && disabled} onClick={() => void runActions(b.onClick, scope)}>
      {text(b.label, scope)}
    </Button>
  );
}

function Header({ p, scope, onClose }: { p: OverlayNodeProps; scope: Scope; onClose?: () => void }) {
  return (
    <Stack direction="row" alignItems="flex-start" spacing={1}>
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography variant="h6" fontWeight={700} noWrap>{text(p.title, scope)}</Typography>
        {p.subtitle && <Typography variant="body2" color="text.secondary">{text(p.subtitle, scope)}</Typography>}
      </Box>
      {onClose && <IconButton size="small" aria-label="Close" onClick={onClose}><CloseIcon fontSize="small" /></IconButton>}
    </Stack>
  );
}

/** A Drawer or Dialog at runtime. */
export function OverlayContainer({ type, props, children }: { type: 'Drawer' | 'Dialog'; props?: Record<string, unknown>; children: React.ReactNode }) {
  const { scope, runActions } = useAppRuntime();
  const p = (props ?? {}) as OverlayNodeProps;
  // Closed until the condition says otherwise (never flashes open while it evaluates).
  const open = useCondition(p.openWhen, scope, false) && !!p.openWhen;
  const close = () => void runActions(p.onClose, scope);
  const buttons = p.buttons ?? [];
  if (type === 'Dialog') {
    return (
      <Dialog open={open} onClose={close} fullWidth maxWidth={p.maxWidth ?? 'sm'}>
        <DialogTitle component="div"><Header p={p} scope={scope} onClose={close} /></DialogTitle>
        <DialogContent dividers><Stack spacing={2}>{children}</Stack></DialogContent>
        {buttons.length > 0 && <DialogActions>{buttons.map((b, i) => <FooterButton key={i} b={b} scope={scope} />)}</DialogActions>}
      </Dialog>
    );
  }
  return (
    <Drawer anchor={p.anchor ?? 'right'} open={open} onClose={close} PaperProps={{ sx: { width: { xs: '100%', md: Number(resolve(p.width ?? 720, scope)) || 720 }, maxWidth: '96vw', p: 3, bgcolor: 'background.default', transition: 'width 200ms' } }}>
      <Stack spacing={2}>
        <Header p={p} scope={scope} onClose={close} />
        {children}
        {buttons.length > 0 && <Stack direction="row" spacing={1} justifyContent="flex-end">{buttons.map((b, i) => <FooterButton key={i} b={b} scope={scope} />)}</Stack>}
      </Stack>
    </Drawer>
  );
}

/** A TabSet at runtime (and on the design canvas - `design` shows every tab's content switchable). */
export function TabSetContainer({ props, childIds, renderChild }: {
  props?: Record<string, unknown>;
  childIds: string[];
  renderChild: (id: string) => React.ReactNode;
}) {
  const { scope, setVariable } = useAppRuntime();
  const p = (props ?? {}) as TabSetNodeProps;
  const tabs = (p.tabs ?? []).slice(0, Math.max(childIds.length, (p.tabs ?? []).length));
  const [local, setLocal] = useState<string | null>(null);
  const shown = useVisibleTabs(tabs, scope);
  const wanted = p.variable ? ((scope.vars as Record<string, unknown>)?.[p.variable] as string | null) : local;
  const active = shown.find((t) => t.id === wanted) ?? shown[0];
  const choose = (id: string) => (p.variable ? setVariable(p.variable, id) : setLocal(id));
  const idx = active ? tabs.indexOf(active) : -1;
  return (
    <Box sx={{ width: '100%' }}>
      {shown.length > 0 && active && (
        <Tabs value={active.id} onChange={(_, v) => choose(v)} variant="scrollable" allowScrollButtonsMobile sx={{ mb: 2, borderBottom: 1, borderColor: 'divider' }}>
          {shown.map((t) => <Tab key={t.id} value={t.id} label={<TabLabel label={t.label} badge={t.badge as string | undefined} scope={scope} />} />)}
        </Tabs>
      )}
      {idx >= 0 && childIds[idx] ? renderChild(childIds[idx]) : shown.length === 0 && <Chip size="small" label="No tabs" />}
    </Box>
  );
}
