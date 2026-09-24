/**
 * useBusinessObjectSelector - the one control for "pick a primary Business
 * Object + binding, then add/remove relevant related Business Objects and
 * their fields" - the exact selection flow Query Builder, Page Studio's
 * Data Binding panel, and the BO Binding wizard each re-implemented with
 * their own copy of this state machine before this module existed.
 *
 * Headless by design: this hook owns the data (BO list, bindings,
 * relationships, per-object field terms, selection) and the mutations, but
 * renders nothing, so it fits Query Builder's table-of-checkboxes UI,
 * Page Studio's draggable field-chip UI, and any future consumer without
 * forcing one visual shape. See BusinessObjectSelectorControl.tsx for the
 * ready-made presentational component built on top of it.
 */
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  listBusinessObjects, fetchBORelationships, fetchBusinessObjectBindings, fetchBOTerms,
  type BusinessObjectOption, type BORelationship, type SemanticTermView,
} from './businessObjectApi';

export interface SelectedObject {
  /** The Business Object id. */
  boId: string;
  boName: string;
  bindingId: string;
  /** true for the primary object; related objects can be removed, the
   * primary object cannot (removing it means starting a new selection). */
  isPrimary: boolean;
  terms: SemanticTermView[];
  termsLoading: boolean;
  termsError: string | null;
}

export interface UseBusinessObjectSelectorOptions {
  /** Hydrate from an existing selection (editing a saved query/report/page
   * binding) instead of starting empty. Only read on mount / when the
   * underlying id changes - pass a stable key via `hydrationKey` if the
   * object itself is recreated on every render. */
  initial?: {
    boId: string;
    bindingId: string;
    relatedBoIds?: string[];
  };
  /** Re-run hydration when this changes (e.g. the saved query's id) -
   * `initial` itself is not deep-compared. */
  hydrationKey?: string;
  /** Selected term ids to hydrate per BO, keyed by boId (primary included).
   * Re-applied whenever a BO's terms finish loading. */
  initialSelectedTermIds?: Record<string, string[]>;
  /** Once true, the primary object + binding can never change again
   * (default true - "the primary object and binding are locked after
   * creation" is the norm every consumer of this hook wants; pass false
   * only for a picker that's allowed to repoint, e.g. Page Studio before
   * any widget has bound to it). */
  lockPrimaryAfterSelect?: boolean;
}

export function useBusinessObjectSelector(options: UseBusinessObjectSelectorOptions = {}) {
  const { initial, hydrationKey, initialSelectedTermIds, lockPrimaryAfterSelect = true } = options;

  const [boOptions, setBoOptions] = useState<BusinessObjectOption[]>([]);
  const [boOptionsLoading, setBoOptionsLoading] = useState(false);

  const [primary, setPrimary] = useState<SelectedObject | null>(null);
  const [related, setRelated] = useState<SelectedObject[]>([]);
  const [locked, setLocked] = useState(false);

  const [relationships, setRelationships] = useState<BORelationship[]>([]);
  const [relationshipsLoading, setRelationshipsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Load the BO list once (for the primary-object picker before anything
  // is selected).
  useEffect(() => {
    setBoOptionsLoading(true);
    listBusinessObjects()
      .then(setBoOptions)
      .catch((e) => setError(e.message))
      .finally(() => setBoOptionsLoading(false));
  }, []);

  const loadTermsInto = useCallback(async (obj: SelectedObject, selectedIds?: string[]): Promise<SelectedObject> => {
    try {
      const terms = await fetchBOTerms(obj.boId, obj.bindingId);
      const selected = new Set(selectedIds || []);
      return {
        ...obj,
        terms: terms.map((t) => ({ ...t, selected: selected.has(t.termNodeId) })),
        termsLoading: false,
        termsError: null,
      };
    } catch (e: any) {
      return { ...obj, terms: [], termsLoading: false, termsError: e.message || 'Failed to load fields' };
    }
  }, []);

  // Hydrate from `initial` (editing an existing selection). boName starts
  // as the raw id - a placeholder, not a real name - and gets backfilled
  // by the effect below once the BO list (with display names) has loaded;
  // there's no cheap single-BO "get display name" endpoint to call here.
  useEffect(() => {
    if (!initial?.boId || !initial?.bindingId) return;
    let cancelled = false;
    (async () => {
      setLocked(true);
      const primaryObj = await loadTermsInto(
        { boId: initial.boId, boName: initial.boId, bindingId: initial.bindingId, isPrimary: true, terms: [], termsLoading: true, termsError: null },
        initialSelectedTermIds?.[initial.boId],
      );
      const relatedObjs: SelectedObject[] = [];
      for (const boId of initial.relatedBoIds || []) {
        const bindings = await fetchBusinessObjectBindings(boId).catch(() => []);
        const def = bindings.find((b) => b.isDefault) || bindings[0];
        if (!def) continue;
        relatedObjs.push(await loadTermsInto(
          { boId, boName: boId, bindingId: def.bindingId, isPrimary: false, terms: [], termsLoading: true, termsError: null },
          initialSelectedTermIds?.[boId],
        ));
      }
      if (!cancelled) {
        setPrimary(primaryObj);
        setRelated(relatedObjs);
      }
    })();
    return () => { cancelled = true; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hydrationKey ?? initial?.boId]);

  // Backfill boName -> real display name once the BO list has loaded. Keyed
  // on boOptions AND on primary/related's own ids (not just boOptions) -
  // hydration is async and frequently finishes *after* the BO list has
  // already loaded, so a boOptions-only dependency array fires once too
  // early (primary is still null) and never re-fires once hydration sets
  // it, leaving the raw id on screen forever.
  const relatedIdsKey = related.map((r) => r.boId).join(',');
  useEffect(() => {
    if (boOptions.length === 0) return;
    const nameFor = (boId: string) => boOptions.find((b) => b.id === boId)?.displayName;
    setPrimary((p) => {
      if (!p) return p;
      const real = nameFor(p.boId);
      return real && real !== p.boName ? { ...p, boName: real } : p;
    });
    setRelated((prev) => prev.map((r) => {
      const real = nameFor(r.boId);
      return real && real !== r.boName ? { ...r, boName: real } : r;
    }));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [boOptions, primary?.boId, relatedIdsKey]);

  // Load relationships for the primary object once it's chosen, to power
  // the "add related object" picker.
  useEffect(() => {
    if (!primary?.boId) { setRelationships([]); return; }
    setRelationshipsLoading(true);
    fetchBORelationships(primary.boId)
      .then(setRelationships)
      .catch(() => setRelationships([]))
      .finally(() => setRelationshipsLoading(false));
  }, [primary?.boId]);

  const selectPrimary = useCallback(async (bo: BusinessObjectOption, bindingId: string) => {
    const obj = await loadTermsInto({ boId: bo.id, boName: bo.displayName || bo.name, bindingId, isPrimary: true, terms: [], termsLoading: true, termsError: null });
    setPrimary(obj);
    setRelated([]);
    if (lockPrimaryAfterSelect) setLocked(true);
  }, [loadTermsInto, lockPrimaryAfterSelect]);

  const addRelated = useCallback(async (boId: string, boName: string) => {
    if (related.some((r) => r.boId === boId) || primary?.boId === boId) return;
    const bindings = await fetchBusinessObjectBindings(boId).catch(() => []);
    const def = bindings.find((b) => b.isDefault) || bindings[0];
    // Some "related objects" (e.g. 1:N child tables reached from the
    // primary's own relationship graph, like Order -> Order Allocation)
    // aren't registered as top-level Business Objects with their own
    // binding - they're queried through the primary's binding via the
    // join the relationship graph already resolved server-side. Falling
    // back to the primary's binding, rather than refusing to add the
    // object, is what makes "+ Join" actually add something for those.
    const bindingId = def?.bindingId || primary?.bindingId;
    if (!bindingId) { setError(`No binding is configured for ${boName}`); return; }
    const obj = await loadTermsInto({ boId, boName, bindingId, isPrimary: false, terms: [], termsLoading: true, termsError: null });
    setRelated((prev) => [...prev, obj]);
  }, [related, primary, loadTermsInto]);

  const removeRelated = useCallback((boId: string) => {
    setRelated((prev) => prev.filter((r) => r.boId !== boId));
  }, []);

  const toggleField = useCallback((boId: string, termNodeId: string) => {
    const flip = (o: SelectedObject): SelectedObject => o.boId !== boId ? o : {
      ...o,
      terms: o.terms.map((t) => t.termNodeId === termNodeId ? { ...t, selected: !t.selected } : t),
    };
    setPrimary((p) => p && flip(p));
    setRelated((prev) => prev.map(flip));
  }, []);

  // Relationship options not already added, for the "add related object" picker.
  const availableRelationships = useMemo(() => {
    const already = new Set([primary?.boId, ...related.map((r) => r.boId)].filter(Boolean) as string[]);
    return relationships.filter((r) => r.kind !== 'relatedTable' && !already.has(r.targetObjectId));
  }, [relationships, primary, related]);

  const objects = useMemo(() => (primary ? [primary, ...related] : related), [primary, related]);

  return {
    // BO list (for the initial primary-object picker)
    boOptions, boOptionsLoading,
    // Current selection
    primary, related, objects, locked,
    // Related-object candidates
    availableRelationships, relationshipsLoading,
    // Mutations
    selectPrimary, addRelated, removeRelated, toggleField,
    error, setError,
  };
}
