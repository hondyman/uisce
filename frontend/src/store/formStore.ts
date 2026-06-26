import create from 'zustand';

interface FormState {
  draft: Record<string, any> | null;
  setDraft: (d: Record<string, any> | null) => void;
}

const useFormStore = create<FormState>((set) => ({
  draft: null,
  setDraft: (d) => set({ draft: d }),
}));

export default useFormStore;
