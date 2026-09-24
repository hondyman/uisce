import type {
  CorePageDefinition,
  PresentationAction,
  PresentationEventKind,
  PresentationRule,
  PresentationTarget,
} from '../../types/pageStudio';

export const PRESENTATION_EVENT_LABELS: Record<PresentationEventKind, string> = {
  pageActivate: 'Page activate',
  fieldChange: 'Field change',
  fieldEdit: 'Field edit',
  selectionChange: 'Selection change',
};

export interface PresentationOverlay {
  hidden?: boolean;
  readOnly?: boolean;
  collapsed?: boolean;
  style?: Record<string, string>;
  label?: string;
  /** Report Studio only — see PresentationAction.suppressRepeat. */
  suppressRepeat?: 'band' | 'page';
}

export type OverlayMap = Record<string, PresentationOverlay>;

export const targetKey = (target: PresentationTarget): string => {
  if (target.kind === 'field') return `field:${target.id}:${target.fieldName || ''}`;
  return target.id;
};

export const isTruthy = (result: unknown): boolean => {
  if (result === true || result === 1) return true;
  if (result === false || result === 0 || result == null) return false;
  if (typeof result === 'string') {
    const s = result.trim().toLowerCase();
    return s !== '' && s !== 'false' && s !== '0';
  }
  return Boolean(result);
};

export const applyActions = (actions: PresentationAction[], into: OverlayMap = {}): OverlayMap => {
  const next: OverlayMap = { ...into };
  for (const action of actions) {
    const key = targetKey(action.target);
    const prev = next[key] || {};
    next[key] = {
      ...prev,
      ...(action.hidden !== undefined ? { hidden: action.hidden } : {}),
      ...(action.readOnly !== undefined ? { readOnly: action.readOnly } : {}),
      ...(action.collapsed !== undefined ? { collapsed: action.collapsed } : {}),
      ...(action.label !== undefined ? { label: action.label } : {}),
      ...(action.suppressRepeat !== undefined ? { suppressRepeat: action.suppressRepeat } : {}),
      style: { ...prev.style, ...action.style },
    };
  }
  return next;
};

/** FieldEdit is transient (in-progress value). Everything else is re-evaluated against the current record. */
export const isPersistentEvent = (event: PresentationEventKind): boolean => event !== 'fieldEdit';

export const newPresentationRule = (
  event: PresentationEventKind,
  patch?: Partial<PresentationRule>,
): PresentationRule => ({
  id: `pe_${Math.random().toString(36).slice(2, 10)}`,
  event,
  when: '',
  actions: [],
  ...patch,
});

export interface LayoutTargetOption {
  id: string;
  kind: 'widget' | 'section' | 'field';
  label: string;
  fieldName?: string;
}

export const collectPresentationTargets = (draft: CorePageDefinition): LayoutTargetOption[] => {
  const out: LayoutTargetOption[] = [];
  Object.values(draft.components || {}).forEach((c) => {
    out.push({ id: c.id, kind: 'widget', label: `${c.type}: ${c.label || c.id}` });
  });
  const walkLayout = (nodes: Record<string, { id: string; type: string; children?: string[] }> | undefined) => {
    if (!nodes) return;
    Object.values(nodes).forEach((n) => {
      out.push({ id: n.id, kind: 'section', label: `${n.type} section (${n.id})` });
    });
  };
  walkLayout(draft.layout?.nodes);
  (draft.tabs || []).forEach((t) => walkLayout(t.layout?.nodes));
  walkLayout(draft.filterBar?.nodes);
  return out;
};
