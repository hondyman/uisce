import { get } from 'svelte/store';
import { getStores } from '$lib/stores.js';
import type { Core, CoreRef } from '$lib/types.js';

/**
 * A reference to a core tile, used to identify it in events.
 */
export interface CoreTileRef {
  id: string;
}

// NOTE: handlers intentionally avoid toggling 'hasFocus' on blur/focus to
// prevent races between focus/blur/click events. Blur should always clear
// focus (idempotent). Focus should always set focus.

export const handleCoreFocus = (coreRef: CoreRef) => {
  const { coreStore, updateCore } = getStores();
  const currentCore = get(coreStore).find((c) => c.id === coreRef.id);
  if (!currentCore) return;

  // Explicitly set focus to true. Avoid toggling to prevent race conditions
  // where blur/focus fire in quick succession and flip state unexpectedly.
  if (!currentCore.hasFocus) {
    updateCore(currentCore.id, { ...currentCore, hasFocus: true });
  }
};

export const handleCoreBlur = (coreRef: CoreRef) => {
  const { coreStore, updateCore } = getStores();
  const core = get(coreStore).find((c) => c.id === coreRef.id);
  if (!core) return;

  // Idempotent: always set hasFocus to false, but only update if it was true
  // to avoid unnecessary store writes and rerenders.
  if (core.hasFocus) {
    updateCore(coreRef.id, { ...core, hasFocus: false });
  }
};

export const handleCoreClick = (coreRef: CoreRef) => {
  const { coreStore, updateCore } = getStores();
  const cores = get(coreStore);

  // Remove focus from any other core that might have it.
  const previouslyFocused = cores.find((c) => c.hasFocus);
  if (previouslyFocused && previouslyFocused.id !== coreRef.id) {
    updateCore(previouslyFocused.id, { ...previouslyFocused, hasFocus: false });
  }

  // Set focus on the clicked core.
  const coreToFocus = cores.find((c) => c.id === coreRef.id);
  if (coreToFocus && !coreToFocus.hasFocus) {
    updateCore(coreToFocus.id, { ...coreToFocus, hasFocus: true });
  }
};
