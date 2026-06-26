import React, { Dispatch, SetStateAction, useEffect, useState } from 'react';
import WorkspaceTabs from './WorkspaceTabs';
import EditorArea from './EditorArea';
import { ProfessionalSearchInput } from '../common/ProfessionalSearchInput';
import { Chip, Tooltip, IconButton, Box } from '@mui/material';
import { SemanticModelCalculations } from '../../features/fabric/components/SemanticModelCalculations';
import * as PaletteIcons from './icons';
import { useGlobalSearch } from '../../contexts/GlobalSearchContext';
import type { SemanticModel, ShowCode } from './types';

interface Props {
  selectedElement?: any;
  setSelectedElementId?: (id: string | null) => void;
  isOver: boolean;
  drop: any;
  activeWorkspaceTab: 'canvas' | 'custom' | 'calculations';
  setActiveWorkspaceTab: (t: 'canvas' | 'custom' | 'calculations') => void;
  // model-related
  selectedColumn: any;
  addDimension: (...args: any[]) => any;
  addMeasure: (...args: any[]) => any;
  addFilter: (...args: any[]) => any;
  getBusinessTermForColumn: (nodeId: string, columnName: string) => any;
  semanticModel: SemanticModel;
  setSemanticModel: (model: SemanticModel) => void;
  modelName: string;
  showCode: ShowCode | null;
  setShowCode: (fmt: ShowCode | null | ((prev: ShowCode | null) => ShowCode | null)) => void;
  rawGenerateJSON: () => string;
  rawGenerateYAML: () => string;
  generateCustomJSON?: () => string;
  generateCustomYAML?: () => string;
  generateCoreJSON?: () => string;
  generateCoreYAML?: () => string;
  generateMergedModelObject?: () => any;
  selectedModel: any;
  openAddModal: (k: any) => void;
  enhancedRemoveSemanticElement: (...args: any[]) => any;
  toggleElementEdit: (type: 'dimensions' | 'measures' | 'filters' | 'joins', id: string) => void;
  updateSemanticElement: (...args: any[]) => any;
  coreOptions: any[];
  // Removed code sidebar props - using single editor in SemanticModelEditor
  // compatibility
  refreshCompatibility: () => Promise<void>;
  compatLoading: boolean;
  issueLevelFilter: 'all' | 'error' | 'warning';
  setIssueLevelFilter: (s: 'all' | 'error' | 'warning') => void;
  issueCodeFilter: string;
  setIssueCodeFilter: (s: string) => void;
  compatErr: string | null;
  filteredCompat: any;
  // search
  filteredNodes?: any[];
  // search handled via GlobalSearchContext
  setSearchTerm?: (s: string) => void;
  expandIssues: Record<string, boolean>;
  setExpandIssues: Dispatch<SetStateAction<Record<string, boolean>>>;
  expandChanges: Record<string, boolean>;
  setExpandChanges: Dispatch<SetStateAction<Record<string, boolean>>>;
  isCodeDirty: boolean;
  setIsCodeDirty: (dirty: boolean) => void;
  editMode: boolean;
  setEditMode: (v: boolean) => void;
  // Extends editing
  availableBaseModels?: Array<{ key: string; label: string; kind: 'core' | 'custom' }>;
  onChangeExtends?: (newBaseKey: string) => void;
  onImportCode?: (text: string, format: 'json'|'yaml'|'jsonc'|null) => Promise<void> | void;
}

const WorkspaceMain: React.FC<Props> = ({
  isOver,
  drop,
  activeWorkspaceTab,
  setActiveWorkspaceTab,
  selectedColumn,
  addDimension,
  addMeasure,
  addFilter,
  getBusinessTermForColumn,
  semanticModel,
  setSemanticModel: _setSemanticModel,
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
  // Removed code/sidebar props - using single editor in SemanticModelEditor
  // compatibility / misc
  refreshCompatibility: _refreshCompatibility,
  compatLoading: _compatLoading,
  issueLevelFilter: _issueLevelFilter,
  setIssueLevelFilter: _setIssueLevelFilter,
  issueCodeFilter: _issueCodeFilter,
  setIssueCodeFilter: _setIssueCodeFilter,
  compatErr: _compatErr,
  filteredCompat,
  
  expandIssues: _expandIssues,
  setExpandIssues: _setExpandIssues,
  expandChanges: _expandChanges,
  setExpandChanges: _setExpandChanges,
  isCodeDirty: _isCodeDirty,
  setIsCodeDirty,
  editMode,
  setEditMode,
  availableBaseModels = [],
  onChangeExtends,
  onImportCode,
  setSearchTerm: setSearchTermProp,
  selectedElement,
  setSelectedElementId,
}) => {
  const [showCompatPanel, setShowCompatPanel] = useState(false);
  // Removed useEffect for activeEditorTab - using single editor in SemanticModelEditor
  useEffect(() => {
    setIsCodeDirty(false);
  }, [selectedModel, setIsCodeDirty]);

  const { searchTerm: ctxTerm, setSearchTerm: ctxSet } = useGlobalSearch();
  // Safely read pre_aggregations without using `as any`
  const preAggs = (semanticModel as unknown as { pre_aggregations?: unknown[] } )?.pre_aggregations || [];

  // Sync the workspace tab with the code/canvas rendering
  // - 'canvas' tab hides code (tiles visible)
  // - 'custom' tab shows code (default to json if not set)
  useEffect(() => {
    if (activeWorkspaceTab === 'custom') {
      setShowCode((prev) => {
        if (prev === 'json' || prev === 'yaml') return prev;
        return 'json';
      });
    } else {
      setShowCode(() => null);
    }
  }, [activeWorkspaceTab, setShowCode]);

  return (
  <main className={`workspace ${isOver ? 'drop-target' : ''}`} ref={drop}>
      <div className="workspace-header">
  {/* Top bar: search + stats + copy/download ABOVE the tabs */}
  <div className="workspace-header-controls">
          <div className="workspace-search-with-stats">
            <div className="workspace-search-left">
              <ProfessionalSearchInput
                value={ctxTerm}
                onChange={(v) => {
                  if (setSearchTermProp) setSearchTermProp(v);
                  else if (ctxSet) ctxSet(v);
                }}
                placeholder="Search catalog"
                navigationEnabled={false}
                showSuggestions={false}
                className="workspace-search-input"
                size="sm"
                variant="compact"
              />
              <div className="workspace-item-stats" aria-hidden="false">
                {selectedModel?.is_custom && (
                  <Chip
                    label="Custom-only view"
                    size="small"
                    className="stat-chip custom-only"
                    icon={<PaletteIcons.IconEye size={14} color="#1E40AF" />}
                  />
                )}
                <Tooltip title="Dimensions" placement="bottom" arrow>
                  <Chip
                    label={semanticModel?.dimensions?.length || 0}
                    size="small"
                    className="stat-chip"
                    component="button"
                    onClick={() => window.dispatchEvent(new CustomEvent('semlayer.scrollToSection', { detail: { section: 'dimensions' } }))}
                    icon={<PaletteIcons.IconDatabase size={14} color="#3b82f6" />}
                    aria-label="Dimensions"
                    role="button"
                  />
                </Tooltip>
                <Tooltip title="Measures" placement="bottom" arrow>
                  <Chip
                    label={semanticModel?.measures?.length || 0}
                    size="small"
                    className="stat-chip"
                    onClick={() => window.dispatchEvent(new CustomEvent('semlayer.scrollToSection', { detail: { section: 'measures' } }))}
                    icon={<PaletteIcons.IconChartBar size={14} color="#8b5cf6" />}
                  />
                </Tooltip>
                <Tooltip title="Filters" placement="bottom" arrow>
                  <Chip
                    label={semanticModel?.filters?.length || 0}
                    size="small"
                    className="stat-chip"
                    onClick={() => window.dispatchEvent(new CustomEvent('semlayer.scrollToSection', { detail: { section: 'filters' } }))}
                    icon={<PaletteIcons.IconFilter size={14} color="#06b6d4" />}
                  />
                </Tooltip>
                <Tooltip title="Joins" placement="bottom" arrow>
                  <Chip
                    label={semanticModel?.joins?.length || 0}
                    size="small"
                    className="stat-chip"
                    onClick={() => window.dispatchEvent(new CustomEvent('semlayer.scrollToSection', { detail: { section: 'joins' } }))}
                    icon={<PaletteIcons.IconPlugConnected size={14} color="#ea580c" />}
                  />
                </Tooltip>
                <Tooltip title="Pre-Aggregations" placement="bottom" arrow>
                  <Chip
                    label={preAggs.length || 0}
                    size="small"
                    className="stat-chip"
                    onClick={() => window.dispatchEvent(new CustomEvent('semlayer.scrollToSection', { detail: { section: 'pre_aggregations' } }))}
                    icon={<PaletteIcons.IconStack3 size={14} color="#64748b" />}
                  />
                </Tooltip>
                <Tooltip title="Compatibility issues" placement="bottom" arrow>
                  <Chip
                    label={filteredCompat ? filteredCompat.length : 0}
                    size="small"
                    className="stat-chip compat"
                    component="button"
                    onClick={() => { setShowCompatPanel((s) => !s); window.dispatchEvent(new CustomEvent('compatibility.badge.click')); }}
                    icon={<PaletteIcons.IconAlertTriangle size={14} />}
                    aria-label="Compatibility issues"
                    role="button"
                    aria-pressed={showCompatPanel}
                  />
                </Tooltip>
              </div>
            </div>
            <div className="workspace-search-right">
              <IconButton aria-label="Copy code" size="small" onClick={() => window.dispatchEvent(new CustomEvent('semlayer.copyCode'))}>
                <PaletteIcons.IconCopy size={16} />
              </IconButton>
              <IconButton aria-label="Download code" size="small" onClick={() => window.dispatchEvent(new CustomEvent('semlayer.downloadCode'))}>
                <PaletteIcons.IconDownload size={16} />
              </IconButton>
            </div>
          </div>
          {showCompatPanel && (
            <div className="compat-panel" data-testid="compat-panel">
              <div className="compat-panel-header">
                <strong>Compatibility Issues</strong>
                <button aria-label="Close compatibility panel" onClick={() => setShowCompatPanel(false)}>Close</button>
              </div>
              <div className="compat-panel-body">{filteredCompat && filteredCompat.length ? `${filteredCompat.length} issues` : 'No issues'}</div>
            </div>
          )}
        </div>
  {/* Tabs row below the top bar */}
  <WorkspaceTabs activeWorkspaceTab={activeWorkspaceTab} setActiveWorkspaceTab={setActiveWorkspaceTab} />
      </div>
    <div className={`workspace-content ${activeWorkspaceTab === 'canvas' ? 'canvas-mode' : ''}`}>
        <EditorArea
          selectedColumn={selectedColumn}
          addDimension={addDimension}
          addMeasure={addMeasure}
          addFilter={addFilter}
          getBusinessTermForColumn={getBusinessTermForColumn}
          semanticModel={semanticModel}
          modelName={modelName}
          showCode={showCode ?? null}
          setShowCode={setShowCode}
          rawGenerateJSON={rawGenerateJSON}
          rawGenerateYAML={rawGenerateYAML}
          generateCustomJSON={generateCustomJSON}
          generateCustomYAML={generateCustomYAML}
          generateCoreJSON={generateCoreJSON}
          generateCoreYAML={generateCoreYAML}
          generateMergedModelObject={generateMergedModelObject}
          selectedModel={selectedModel}
          openAddModal={openAddModal}
          enhancedRemoveSemanticElement={enhancedRemoveSemanticElement}
          toggleElementEdit={toggleElementEdit}
          updateSemanticElement={updateSemanticElement}
          coreOptions={coreOptions}
          editMode={editMode}
          setEditMode={setEditMode}
          availableBaseModels={availableBaseModels}
          onChangeExtends={(k: string) => onChangeExtends && onChangeExtends(k)}
          onImportCode={onImportCode}
          selectedElement={selectedElement}
          setSelectedElementId={setSelectedElementId}
        />
        
        {activeWorkspaceTab === 'calculations' && selectedModel && (
          <Box sx={{ p: 2, height: '100%', overflow: 'auto' }}>
            <SemanticModelCalculations modelId={selectedModel.id} />
          </Box>
        )}
      </div>
    </main>
  );
};

export default WorkspaceMain;

