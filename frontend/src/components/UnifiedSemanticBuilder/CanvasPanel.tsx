import React, { useEffect, useRef } from 'react';
import { devDebug } from '../../utils/devLogger';
import SemanticModelOverview from './SemanticModelOverview';
import type { SemanticModel } from './types';

interface CanvasPanelProps {
  semanticModel: SemanticModel;
  coreOptions: any[];
  removeSemanticElement: (type: 'dimensions'|'measures'|'filters'|'joins', id: string) => void;
  toggleElementEdit: (type: 'dimensions'|'measures'|'filters'|'joins', id: string) => void;
  updateSemanticElement: (type: 'dimensions'|'measures'|'filters'|'joins', id: string, updates: any) => void;
  extendsModel?: string | null;
  editMode?: boolean;
  onElementSelect?: (element: any) => void;
  allModelKeys?: string[];
}

const CanvasPanel: React.FC<CanvasPanelProps> = ({ semanticModel, coreOptions, removeSemanticElement, toggleElementEdit, updateSemanticElement, extendsModel = null, editMode = false, onElementSelect, allModelKeys = [] }) => {
  const ref = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const handler = () => {
      // Focus the canvas container so keyboard interactions (e.g. palette navigation)
      // land on the canvas when edit mode is entered.
      if (ref.current) {
        try { ref.current.focus(); } catch (_) { /* no-op */ }
      }
    };
    window.addEventListener('semlayer.focusCanvas', handler);
    return () => window.removeEventListener('semlayer.focusCanvas', handler);
  }, []);

  return (
    <div className="canvas-panel" ref={ref} tabIndex={0} aria-label="Model canvas">
  {(() => { try { const d=semanticModel?.dimensions?.length||0; const m=semanticModel?.measures?.length||0; const f=semanticModel?.filters?.length||0; const j=(semanticModel?.joins||[]).length||0; const p=((semanticModel as any)?.pre_aggregations||[]).length||0; devDebug('[CanvasPanel] render counts', {d,m,f,j,p}); } catch {} return null; })()}
      {(() => {
        const d = semanticModel?.dimensions?.length || 0;
        const m = semanticModel?.measures?.length || 0;
        const f = semanticModel?.filters?.length || 0;
        const j = (semanticModel?.joins || []).length || 0;
        const p = ((semanticModel as any)?.pre_aggregations || []).length || 0;
        const isEmpty = d === 0 && m === 0 && f === 0 && j === 0 && p === 0;
        // If the model is custom and has an extendsModel, we still want to show the Overview so the Extends tile is visible.
        const forceShowOverview = Boolean((semanticModel as any)?.is_custom && extendsModel);
        if (isEmpty && !forceShowOverview) {
          return (
            <div className="empty-semantic-model">
              <div className="getting-started-tips">
                {semanticModel?.name ?
                  `The "${semanticModel.name}" model is currently empty. Add dimensions, measures, filters, or joins to get started.` :
                  'Select a model in the sidebar to view its dimensions, measures, filters, and joins here.'
                }
              </div>
            </div>
          );
        }
        return (
          <SemanticModelOverview
            semanticModel={semanticModel}
            coreOptions={coreOptions}
            removeSemanticElement={removeSemanticElement}
            toggleElementEdit={toggleElementEdit}
            updateSemanticElement={updateSemanticElement}
            extendsModel={extendsModel}
            editMode={editMode}
            onElementSelect={onElementSelect}
            allModelKeys={allModelKeys}
          />
        );
      })()}
    </div>
  );
};

export default CanvasPanel;
