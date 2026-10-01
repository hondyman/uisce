import type { CorePageDefinition } from '../../../types/pageStudio';

/**
 * A generated page travels from the list to the editor as an unsaved draft.
 * Navigation state would do, but the locale redirect (/page-studio -> /en/page-studio)
 * drops it, and the editor mounts more than once on the way (the redirect, and
 * StrictMode in development), so a read-once hand-off loses it. The draft waits
 * in session storage for a short while instead, so a stale one never reappears.
 */
const KEY = 'page-studio.generated-draft';
export const GENERATED_DRAFT_TTL_MS = 60_000;

export function handOverGeneratedDraft(draft: Partial<CorePageDefinition>, now = Date.now()): void {
  try { sessionStorage.setItem(KEY, JSON.stringify({ at: now, draft })); } catch { /* storage unavailable: the editor opens blank */ }
}

export function takeGeneratedDraft(now = Date.now()): Partial<CorePageDefinition> | undefined {
  try {
    const raw = sessionStorage.getItem(KEY);
    if (!raw) return undefined;
    const { at, draft } = JSON.parse(raw) as { at: number; draft: Partial<CorePageDefinition> };
    if (now - at > GENERATED_DRAFT_TTL_MS) { sessionStorage.removeItem(KEY); return undefined; }
    return draft;
  } catch {
    return undefined;
  }
}
