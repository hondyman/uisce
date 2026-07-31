import { create } from 'zustand';

interface PageContextState {
  contextMap: Record<string, any>;
  activeRecord: Record<string, any> | null;
  overrideMap: Record<string, { hidden?: boolean; disabled?: boolean; readOnly?: boolean }>;

  setContextValue: (key: string, value: any) => void;
  clearContextValue: (key: string) => void;
  setActiveRecord: (record: Record<string, any> | null) => void;
  applyRuleOverrides: (overrides: Record<string, { hidden?: boolean; disabled?: boolean; readOnly?: boolean }>) => void;
  resetContext: () => void;
}

export const usePageContextStore = create<PageContextState>((set) => ({
  contextMap: {},
  activeRecord: null,
  overrideMap: {},

  setContextValue: (key, value) =>
    set((state) => ({
      contextMap: { ...state.contextMap, [key]: value },
    })),

  clearContextValue: (key) =>
    set((state) => {
      const next = { ...state.contextMap };
      delete next[key];
      return { contextMap: next };
    }),

  setActiveRecord: (record) => set({ activeRecord: record }),

  applyRuleOverrides: (overrides) => set({ overrideMap: overrides }),

  resetContext: () => set({ contextMap: {}, activeRecord: null, overrideMap: {} }),
}));
