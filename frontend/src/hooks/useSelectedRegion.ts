import { useSyncExternalStore } from 'react';
import { getSelectedRegion, subscribeSelectedRegion } from '../lib/region';

/** The currently selected region ('' when none), kept in sync with what the API client sends. */
export function useSelectedRegion(): string {
  return useSyncExternalStore(subscribeSelectedRegion, getSelectedRegion, () => '');
}
