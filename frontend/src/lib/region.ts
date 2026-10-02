const KEY = 'selected_region';
const EVENT = 'selected-region-changed';

// The region the user selected, or '' when none is. There is deliberately no default:
// an unselected region is sent as nothing and the backend rejects the request
// ("region is required"), rather than silently scoping it to a guessed region.
export function getSelectedRegion(): string {
  try {
    return localStorage.getItem(KEY) || '';
  } catch {
    return '';
  }
}

export function setSelectedRegion(region: string): void {
  const value = (region || '').trim();
  if (!value) {
    clearSelectedRegion();
    return;
  }
  localStorage.setItem(KEY, value);
  notify();
}

/** Forget the selection, so nothing is sent until a region is chosen again. */
export function clearSelectedRegion(): void {
  try {
    localStorage.removeItem(KEY);
  } catch {
    /* storage unavailable: nothing was stored */
  }
  notify();
}

function notify(): void {
  if (typeof window !== 'undefined') window.dispatchEvent(new Event(EVENT));
}

/** Subscribe to selection changes (this tab via our event, other tabs via 'storage'). */
export function subscribeSelectedRegion(onChange: () => void): () => void {
  const onStorage = (e: StorageEvent) => {
    if (e.key === null || e.key === KEY) onChange();
  };
  window.addEventListener(EVENT, onChange);
  window.addEventListener('storage', onStorage);
  return () => {
    window.removeEventListener(EVENT, onChange);
    window.removeEventListener('storage', onStorage);
  };
}
