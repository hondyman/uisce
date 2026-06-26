// React default import removed — using automatic JSX runtime
import type { SemanticModel } from './types';

interface ModelTreeProps {
  semanticModel: SemanticModel;
  onSelect?: (type: string, id: string) => void;
}

const ModelTree: React.FC<ModelTreeProps> = ({ semanticModel, onSelect }) => {
  return (
    <aside className="model-tree">
      {/* <h4 className="model-tree-title">Model Structure</h4> */}
      <div className="model-tree-section">
        <strong>Dimensions</strong>
        <ul>
          {(semanticModel.dimensions || []).map((d: any) => (
            <li key={d.id} onClick={() => onSelect?.('dimensions', d.id)}>{d.title || d.name}</li>
          ))}
        </ul>
      </div>
      <div className="model-tree-section">
        <strong>Measures</strong>
        <ul>
          {(semanticModel.measures || []).map((m: any) => (
            <li key={m.id} onClick={() => onSelect?.('measures', m.id)}>{m.title || m.name}</li>
          ))}
        </ul>
      </div>
      <div className="model-tree-section">
        <strong>Filters</strong>
        <ul>
          {(semanticModel.filters || []).map((f: any) => (
            <li key={f.id} onClick={() => onSelect?.('filters', f.id)}>{f.title || f.name}</li>
          ))}
        </ul>
      </div>
    </aside>
  );
};

export default ModelTree;
