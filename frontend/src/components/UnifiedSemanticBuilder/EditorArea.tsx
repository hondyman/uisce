import { useMemo } from 'react';
import ActionsPanel from './ActionsPanel';
import SemanticModelEditor from './SemanticModelEditor';
import type { ShowCode } from './types';

interface Props {
  selectedElement?: any;
  setSelectedElementId?: (id: string | null) => void;
  selectedColumn: any;
  addDimension: (...args: any[]) => any;
  addMeasure: (...args: any[]) => any;
  addFilter: (...args: any[]) => any;
  getBusinessTermForColumn: (nodeId: string, columnName: string) => any;
  semanticModel: any;
  modelName: string;
  showCode: ShowCode | null;
  setShowCode: (fmt: ShowCode | null) => void;
  rawGenerateJSON: () => string;
  rawGenerateYAML: () => string;
  generateCustomJSON?: () => string;
  generateCustomYAML?: () => string;
  generateCoreJSON?: () => string;
  generateCoreYAML?: () => string;
  generateMergedModelObject?: () => any;
  // Removed code sidebar props - using single editor in SemanticModelEditor
  selectedModel: any;
  openAddModal: (k: any) => void;
  enhancedRemoveSemanticElement: (...args: any[]) => any;
  toggleElementEdit: (type: 'dimensions' | 'measures' | 'filters' | 'joins', id: string) => void;
  updateSemanticElement: (...args: any[]) => any;
  coreOptions: any[];
  editMode: boolean;
  setEditMode: (v: boolean) => void;
  availableBaseModels: Array<{ key: string; label: string; kind: 'core' | 'custom' }>;
  onChangeExtends: (newBaseKey: string) => void;
  onImportCode?: (text: string, format: 'json'|'yaml'|'jsonc'|null) => Promise<void> | void;
  // Removed searchTerm and setSearchTerm props
}

const EditorArea: React.FC<Props> = ({
  selectedColumn,
  addDimension,
  addMeasure,
  addFilter,
  getBusinessTermForColumn,
  semanticModel,
  modelName,
  showCode,
  setShowCode,
  rawGenerateJSON,
  rawGenerateYAML,
  generateCustomJSON,
  generateCustomYAML,
  generateCoreJSON,
  generateCoreYAML,
  generateMergedModelObject,
  selectedModel,
  openAddModal,
  enhancedRemoveSemanticElement,
  toggleElementEdit,
  updateSemanticElement,
  coreOptions,
  editMode,
  setEditMode: _setEditMode,
  availableBaseModels,
  onChangeExtends,
  onImportCode,
  selectedElement,
  setSelectedElementId,
}) => {
  // ...existing code...
  const computedExtendsModel = useMemo(() => {
    if (!selectedModel || !selectedModel.is_custom) return null;
    // 1) direct field
    let ext: any = selectedModel.parent_model_key || selectedModel?.metadata?.inherits_from || null;
    // 2) try from source/resolved config shapes
    const tryFromConfig = (cfg: any) => {
      if (!cfg || typeof cfg !== 'object') return null;
      // cube.js-like shapes: cube, cubes[], or top-level
      const pickCube = (raw: any) => {
        if (!raw) return null;
        if (Array.isArray((raw as any).cubes)) {
          const matchName = selectedModel?.model_key || selectedModel?.display_name || selectedModel?.title;
          return (raw as any).cubes.find((c: any) => c?.name === matchName) || (raw as any).cubes[0] || raw;
        }
        if ((raw as any).cube && typeof (raw as any).cube === 'object') return (raw as any).cube;
        return raw;
      };
      const chosen = pickCube(cfg) || cfg;
      const val = (chosen as any)?.extends ?? (cfg as any)?.extends ?? null;
      if (!val) return null;
      // could be string or array; prefer first string
      if (Array.isArray(val)) return String(val[0] ?? '');
      return String(val);
    };
    if (!ext) {
      ext = tryFromConfig((selectedModel as any).source_config) || tryFromConfig((selectedModel as any).resolved_config) || null;
    }
    // 3) last resort, peek at generated JSON (custom) for an 'extends' field
    if (!ext) {
      try {
        const gen = generateCustomJSON ? generateCustomJSON() : rawGenerateJSON();
        const obj = JSON.parse(gen);
        const top = (obj && typeof obj === 'object') ? obj : null;
        const val = (top as any)?.extends || (Array.isArray((top as any)?.cubes) ? (top as any).cubes?.[0]?.extends : null);
        if (val) ext = Array.isArray(val) ? String(val[0]) : String(val);
      } catch {}
    }
    return ext || null;
  }, [selectedModel?.id, selectedModel?.is_custom, selectedModel?.parent_model_key, selectedModel?.metadata, selectedModel?.model_key, selectedModel?.display_name, selectedModel?.title, generateCustomJSON, rawGenerateJSON]);

  // ...existing code...
  // Safely extract selected element id without using `as any`
  const selectedElementIdValue = selectedElement ? (selectedElement as unknown as { id?: string }).id ?? undefined : undefined;

  return (
    <>
      <ActionsPanel
        selectedColumn={selectedColumn}
        addDimension={addDimension}
        addMeasure={addMeasure}
        addFilter={addFilter}
        getBusinessTermForColumn={getBusinessTermForColumn}
      />

  <SemanticModelEditor
        semanticModel={semanticModel}
        modelName={modelName}
        showCode={showCode}
        onToggleCode={setShowCode}
        generateJSON={rawGenerateJSON}
        generateYAML={rawGenerateYAML}
        generateCustomJSON={generateCustomJSON}
        generateCustomYAML={generateCustomYAML}
        generateCoreJSON={generateCoreJSON}
        generateCoreYAML={generateCoreYAML}
        generateMergedModelObject={generateMergedModelObject}
        codeEditable={selectedModel ? (selectedModel.is_custom || selectedModel.can_edit) : true}
        extendsModel={computedExtendsModel}
  selectedModel={selectedModel}
  selectedColumn={selectedColumn}
  // Controlled selection: if parent didn't provide selection, pass undefined so editor uses internal state
  selectedElementId={selectedElementIdValue}
  setSelectedElementId={setSelectedElementId || undefined}
        onAdd={openAddModal}
        removeSemanticElement={enhancedRemoveSemanticElement}
        toggleElementEdit={toggleElementEdit}
        updateSemanticElement={updateSemanticElement}
        coreOptions={coreOptions}
        editMode={editMode}
        availableBaseModels={availableBaseModels}
        onChangeExtends={onChangeExtends}
        onImportCode={onImportCode}
      />
    </>
  );
}
export default EditorArea;
