import { useState, useEffect } from 'react';
import { listSemanticViews } from './api';
import { devError } from './utils/devLogger';
import type { SemanticViewMeta as BaseSemanticViewMeta } from './types';

// Extend the base type to include claim-aware properties
interface SemanticViewMeta extends BaseSemanticViewMeta {
  isRestricted?: boolean;
  restrictionType?: string | null;
}

interface SemanticViewPickerProps {
  datasourceId: string;
  onSelect: (view: SemanticViewMeta) => void;
  views?: SemanticViewMeta[]; // Allow parent to pass in pre-filtered, claim-aware views
}

export default function SemanticViewPicker({ datasourceId, onSelect, views: preloadedViews }: SemanticViewPickerProps) {
  const [localViews, setLocalViews] = useState<SemanticViewMeta[]>([]);

  useEffect(() => {
    if (preloadedViews) {
      setLocalViews(preloadedViews);
    } else if (datasourceId) {
  listSemanticViews(datasourceId).then(setLocalViews).catch((e) => { devError(e); });
    }
  }, [datasourceId, preloadedViews]);

  return (
    <div className="semantic-view-picker">
      <h4>Semantic Views</h4>
      <ul>
        {localViews.map(v => (
          <li key={v.id} onClick={() => onSelect(v)} title={v.restrictionType || v.description}>
            <strong>
              {v.name}
              {v.certified && ' ✅'}
              {v.isRestricted && (
                <span className="restricted-badge-small" title={v.restrictionType || 'Restricted access'}>
                  🔒
                </span>
              )}
            </strong>
            <small>{v.description}</small>
            <div>Owner: {v.owner}</div>
          </li>
        ))}
      </ul>
    </div>
  );
}