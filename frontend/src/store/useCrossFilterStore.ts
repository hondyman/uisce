import { create } from 'zustand';

/**
 * Cross-filter bus shared by BI-style widgets (Report Builder canvas
 * elements, Page Studio widgets, Query Builder results). A widget publishes
 * a value when the user clicks a data point (a chart category, a table
 * row); every other widget on the same page/report whose query selects the
 * same (boId, termNodeId) as a dimension applies it as an equality filter
 * on its next fetch — Charles River Workbench-style "click one visual,
 * everything else reacts."
 *
 * Keys are `${boId}:${termNodeId}` so filters only apply to widgets bound to
 * the same Business Object field, never across unrelated fields that happen
 * to share a name.
 */
export interface CrossFilterState {
  filters: Record<string, unknown>;
  setFilter: (key: string, value: unknown) => void;
  clearFilter: (key: string) => void;
  clearAll: () => void;
}

export const crossFilterKey = (boId: string, termNodeId: string) => `${boId}:${termNodeId}`;

export const useCrossFilterStore = create<CrossFilterState>((set) => ({
  filters: {},
  setFilter: (key, value) =>
    set((state) => ({ filters: { ...state.filters, [key]: value } })),
  clearFilter: (key) =>
    set((state) => {
      const next = { ...state.filters };
      delete next[key];
      return { filters: next };
    }),
  clearAll: () => set({ filters: {} }),
}));
