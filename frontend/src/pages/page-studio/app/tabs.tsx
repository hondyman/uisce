import React, { useEffect, useState } from 'react';
import { Badge } from '@mui/material';
import type { ConditionNode, TextSpec } from './appModel';
import { resolve, text, type Scope } from './bindings';
import { evaluateCondition } from './conditions';

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
export function TabLabel({ label, badge, scope }: { label: TextSpec; badge?: string; scope: Scope }) {
  const n = badge ? Number(resolve(badge, scope)) || 0 : 0;
  const shown = text(label, scope);
  if (!badge) return <>{shown}</>;
  return <Badge color="warning" badgeContent={n} sx={{ pr: n ? 1.5 : 0 }}>{shown}</Badge>;
}

