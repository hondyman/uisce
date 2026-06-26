import { useState, useCallback } from 'react';

interface Params {
  selectedModel: { is_custom?: boolean; is_core?: boolean; id?: string; model_key?: string } | null;
  modelName: string;
  semanticModel: unknown;
  createCustomModel: (modelKey: string) => Promise<any>;
  updateModel: (id: string, payload: any) => Promise<unknown>;
  setSemanticModel?: (m: any) => void;
  setModelName?: (name: string) => void;
  showNotification: (msg: string, level?: 'success' | 'error') => void;
}
// align showNotification with unified hook signatures (level is 'success' | 'error')

const useModelSaver = ({ selectedModel, modelName, semanticModel, createCustomModel, updateModel, showNotification }: Params) => {
  const [isSaving, setIsSaving] = useState(false);

  const handleSave = useCallback(async () => {
    if (!selectedModel) {
      showNotification('No Model Selected. Please select a model from the catalog before saving.', 'error');
      return;
    }
    setIsSaving(true);
    try {
      const resolvedConfig = semanticModel; // caller may pass precomputed resolved object
      const asRec = (v: unknown) => (v && typeof v === 'object' && !Array.isArray(v) ? v as Record<string, unknown> : {} as Record<string, unknown>);
      const semRec = asRec(semanticModel);

      const updatePayload = {
        title: modelName,
        description: String(semRec.description ?? ''),
        resolved_config: resolvedConfig,
      } as unknown;

      if (selectedModel?.is_custom) {
        await updateModel(selectedModel.id ?? '', updatePayload);
        showNotification('✅ Custom model saved successfully!', 'success');
      } else if (selectedModel?.is_core) {
        try {
          const safeModelId = selectedModel?.id || '';
          const safeModelKey = selectedModel?.model_key || '';

          await updateModel(safeModelId, updatePayload);
          const customModel = await createCustomModel(safeModelKey);
          const customRec = asRec(customModel);
          const customId = customRec?.id ? String(customRec.id) : undefined;
          if (customId) {
            await updateModel(customId, updatePayload);
            showNotification('✅ Custom model created and saved successfully!', 'success');
          }
        } catch (e) {
          showNotification('❌ Failed to create custom model', 'error');
        }
      }
    } catch (e) {
      showNotification(`❌ Save failed: ${e instanceof Error ? e.message : 'Unknown error'}`, 'error');
    } finally {
      setIsSaving(false);
    }
  }, [selectedModel, modelName, semanticModel, createCustomModel, updateModel, showNotification]);

  return { isSaving, handleSave };
};

export default useModelSaver;
