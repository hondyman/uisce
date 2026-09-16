import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import type { PresentationRule } from '../../types/pageStudio';
import { evaluateExpressionTextWasm } from '../../rules/wasmRuntime';
import {
  applyActions,
  isPersistentEvent,
  isTruthy,
  targetKey,
  type OverlayMap,
  type PresentationOverlay,
} from './presentationEvents';

const SAMPLE_CONTEXT: Record<string, unknown> = {
  Status: 'Open',
  TargetQuantity: 100,
  LimitPrice: 10,
};

interface PresentationRuntimeValue {
  overlays: OverlayMap;
  setRecord: (record: Record<string, unknown> | null) => void;
  setEditingField: (termKey: string | null, value?: unknown) => void;
  overlayFor: (id: string, fieldName?: string) => PresentationOverlay | undefined;
  usingSampleData: boolean;
}

const PresentationRuntimeContext = createContext<PresentationRuntimeValue>({
  overlays: {},
  setRecord: () => {},
  setEditingField: () => {},
  overlayFor: () => undefined,
  usingSampleData: true,
});

const evaluateRules = async (
  rules: PresentationRule[],
  ctx: Record<string, unknown>,
): Promise<OverlayMap> => {
  let map: OverlayMap = {};
  for (const rule of rules) {
    let ok = true;
    if (rule.when.trim()) {
      try {
        const { result } = await evaluateExpressionTextWasm(rule.when, ctx);
        ok = isTruthy(result);
      } catch {
        ok = false;
      }
    }
    if (ok) map = applyActions(rule.actions, map);
  }
  return map;
};

export const PresentationProvider: React.FC<{
  rules: PresentationRule[];
  children: React.ReactNode;
}> = ({ rules, children }) => {
  const [record, setRecordState] = useState<Record<string, unknown> | null>(null);
  const [editing, setEditing] = useState<{ termKey: string; value: unknown } | null>(null);
  const [persistent, setPersistent] = useState<OverlayMap>({});
  const [editOverlay, setEditOverlay] = useState<OverlayMap>({});

  const usingSampleData = !record;
  const baseCtx = record || SAMPLE_CONTEXT;

  useEffect(() => {
    let cancelled = false;
    const persistentRules = rules.filter((r) => isPersistentEvent(r.event));
    if (persistentRules.length === 0) {
      setPersistent({});
      return;
    }
    const timer = setTimeout(() => {
      void evaluateRules(persistentRules, baseCtx).then((map) => {
        if (!cancelled) setPersistent(map);
      });
    }, 150);
    return () => { cancelled = true; clearTimeout(timer); };
  }, [rules, baseCtx]);

  useEffect(() => {
    let cancelled = false;
    if (!editing) {
      setEditOverlay({});
      return;
    }
    const editRules = rules.filter(
      (r) => r.event === 'fieldEdit' && (!r.source?.termKey || r.source.termKey === editing.termKey),
    );
    if (editRules.length === 0) {
      setEditOverlay({});
      return;
    }
    const ctx = { ...baseCtx, [editing.termKey]: editing.value };
    void evaluateRules(editRules, ctx).then((map) => {
      if (!cancelled) setEditOverlay(map);
    });
    return () => { cancelled = true; };
  }, [rules, editing, baseCtx]);

  const merged = useMemo(() => {
    const next: OverlayMap = { ...persistent };
    for (const [k, v] of Object.entries(editOverlay)) {
      next[k] = { ...next[k], ...v, style: { ...next[k]?.style, ...v.style } };
    }
    return next;
  }, [persistent, editOverlay]);

  const setRecord = useCallback((next: Record<string, unknown> | null) => {
    setRecordState(next);
  }, []);

  const setEditingField = useCallback((termKey: string | null, value?: unknown) => {
    setEditing(termKey ? { termKey, value } : null);
  }, []);

  const overlayFor = useCallback((id: string, fieldName?: string): PresentationOverlay | undefined => {
    if (fieldName) return merged[targetKey({ kind: 'field', id, fieldName })] || merged[id];
    return merged[id];
  }, [merged]);

  const value = useMemo(
    () => ({ overlays: merged, setRecord, setEditingField, overlayFor, usingSampleData }),
    [merged, setRecord, setEditingField, overlayFor, usingSampleData],
  );

  return (
    <PresentationRuntimeContext.Provider value={value}>
      {children}
    </PresentationRuntimeContext.Provider>
  );
};

export const usePresentationRuntime = () => useContext(PresentationRuntimeContext);

export const usePresentationOverlay = (id: string, fieldName?: string): PresentationOverlay | undefined => {
  const { overlayFor } = usePresentationRuntime();
  return overlayFor(id, fieldName);
};
