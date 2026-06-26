// React default import not required with new JSX transform
import './LineageTypeSelector.css';

interface LineageTypeSelectorProps {
  lineageType: 'technical' | 'semantic';
  onLineageTypeChange: (type: 'technical' | 'semantic') => void;
  technicalCount: number;
  semanticCount: number;
}

const LineageTypeSelector: React.FC<LineageTypeSelectorProps> = ({
  lineageType,
  onLineageTypeChange,
  technicalCount,
  semanticCount,
}) => {
  // Styles moved to CSS module imported above.

  return (
    <div className="lineage-type-selector">
      <button
        className={`lts-tab ${lineageType === 'technical' ? 'active' : 'inactive'}`}
        onClick={() => onLineageTypeChange('technical')}
      >
        <span>Technical</span>
        <span className="lts-count">{technicalCount}</span>
      </button>
      <button
        className={`lts-tab ${lineageType === 'semantic' ? 'active' : 'inactive'}`}
        onClick={() => onLineageTypeChange('semantic')}
      >
        <span>Semantic</span>
        <span className="lts-count">{semanticCount}</span>
      </button>
    </div>
  );
};

export default LineageTypeSelector;