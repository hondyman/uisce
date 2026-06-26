import { useCallback } from 'react';

interface Params {
  selectedModel: any | null;
  createCustomModel: (modelKey: string) => Promise<any>;
  showNotification: (message: string, type?: 'success' | 'error') => void;
}

const useEnsureCustomAndAdd = ({ selectedModel, createCustomModel, showNotification }: Params) => {
  const ensureCustomAndApply = useCallback(async (fn: (...args: any[]) => any, ...args: any[]) => {
    if (selectedModel && selectedModel.is_core && !selectedModel.is_custom) {
      if (!selectedModel.model_key) {
        showNotification('Cannot create custom model; core model key is missing.', 'error');
        return;
      }
      try {
        await createCustomModel(selectedModel.model_key);
        showNotification('A writable custom model has been created for your edits. Please re-apply your action.', 'success');
      } catch (e) {
        showNotification(`Failed to create custom model: ${e instanceof Error ? e.message : 'Unknown error'}`, 'error');
      }
    } else {
      fn(...args);
    }
  }, [selectedModel, createCustomModel, showNotification]);

  const wrapAdd = useCallback((rawAdd: (...args: any[]) => any) => {
    return (...args: any[]) => ensureCustomAndApply(rawAdd, ...args);
  }, [ensureCustomAndApply]);

  const enhancedRemove = useCallback((removeFn: (...args: any[]) => any) => {
    return (type: string, id: string) => ensureCustomAndApply(removeFn, type, id);
  }, [ensureCustomAndApply]);

  return { ensureCustomAndApply, wrapAdd, enhancedRemove } as const;
};

export default useEnsureCustomAndAdd;
