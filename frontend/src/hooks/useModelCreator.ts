import { useCallback, Dispatch, SetStateAction } from 'react';

interface CreateFormData {
  cubeName?: string;
}

// Use `unknown` for external shapes to avoid unsafe `any` and keep the hook flexible.
const useModelCreator = ({ createCustomModel, setSemanticModel, setModelName, showNotification }: {
  createCustomModel: (modelKey: string) => Promise<unknown>;
  // Accept a React state setter for compatibility with callers
  setSemanticModel: Dispatch<SetStateAction<any>>;
  setModelName: (n: string) => void;
  showNotification: (m: string, t?: 'success'|'error') => void;
}) => {
  const handleCreateCustomModel = useCallback(async (formData: CreateFormData) => {
    try {
      await createCustomModel(formData.cubeName || '');

      // Build a minimal model object and pass as `unknown` (safer than casting to `any`).
      const newModel: Record<string, unknown> = {
        name: formData.cubeName || '',
        dimensions: [],
        measures: [],
        filters: [],
        joins: [],
        is_custom: true,
      };

      setSemanticModel(newModel);
      setModelName(formData.cubeName || '');
      showNotification(`Custom model "${formData.cubeName}" created successfully!`, 'success');
    } catch (error) {
      showNotification(`Failed to create custom model: ${error instanceof Error ? error.message : 'Unknown error'}`, 'error');
      throw error;
    }
  }, [createCustomModel, setSemanticModel, setModelName, showNotification]);

  return { handleCreateCustomModel } as const;
};

export default useModelCreator;
