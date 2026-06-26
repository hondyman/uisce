import { X } from 'lucide-react';
import { Edge } from 'reactflow';
import './ErdInfoPanel.css';

interface ColumnInfo {
  name: string;
  type: string;
  nullable?: boolean;
  isPrimaryKey?: boolean;
  isForeignKey?: boolean;
  description?: string;
}

interface ErdInfoPanelProps {
  isOpen: boolean;
  selectedColumn?: ColumnInfo | null;
  selectedEdge?: Edge | null;
  tableName?: string;
  onClose: () => void;
}

const ErdInfoPanel: React.FC<ErdInfoPanelProps> = ({
  isOpen,
  selectedColumn,
  selectedEdge,
  tableName,
  onClose,
}) => {
  if (!isOpen) return null;

  return (
    <div className={`erd-info-panel ${isOpen ? 'open' : ''}`}>
      <div className="info-panel-header">
        <h3>{selectedColumn ? 'Column Information' : 'Relationship Information'}</h3>
        <button
          className="info-panel-close"
          onClick={onClose}
          aria-label="Close info panel"
        >
          <X size={18} />
        </button>
      </div>

      <div className="info-panel-content">
        {selectedColumn && (
          <>
            {tableName && (
              <div className="info-section">
                <label>Table:</label>
                <span className="info-value">{tableName}</span>
              </div>
            )}
            <div className="info-section">
              <label>Column Name:</label>
              <span className="info-value info-value-primary">{selectedColumn.name}</span>
            </div>
            <div className="info-section">
              <label>Data Type:</label>
              <span className="info-value info-value-code">{selectedColumn.type}</span>
            </div>
            <div className="info-section">
              <label>Nullable:</label>
              <span className={`info-badge ${selectedColumn.nullable ? 'badge-yes' : 'badge-no'}`}>
                {selectedColumn.nullable !== false ? 'Yes' : 'No'}
              </span>
            </div>
            <div className="info-section">
              <label>Primary Key:</label>
              <span className={`info-badge ${selectedColumn.isPrimaryKey ? 'badge-pk' : 'badge-no'}`}>
                {selectedColumn.isPrimaryKey ? 'Yes' : 'No'}
              </span>
            </div>
            <div className="info-section">
              <label>Foreign Key:</label>
              <span className={`info-badge ${selectedColumn.isForeignKey ? 'badge-fk' : 'badge-no'}`}>
                {selectedColumn.isForeignKey ? 'Yes' : 'No'}
              </span>
            </div>
            {selectedColumn.description && (
              <div className="info-section info-section-description">
                <label>Description:</label>
                <p className="info-description">{selectedColumn.description}</p>
              </div>
            )}
          </>
        )}

        {selectedEdge && (
          <>
            <div className="info-section">
              <label>Source Table:</label>
              <span className="info-value info-value-primary">{selectedEdge.source}</span>
            </div>
            <div className="info-section">
              <label>Target Table:</label>
              <span className="info-value info-value-primary">{selectedEdge.target}</span>
            </div>
            <div className="info-section">
              <label>Relationship Type:</label>
              <span className="info-badge badge-relationship">
                {selectedEdge.data?.relationship || 'Foreign Key'}
              </span>
            </div>
            {selectedEdge.data?.sourceColumn && (
              <div className="info-section">
                <label>Source Column:</label>
                <span className="info-value info-value-code">{selectedEdge.data.sourceColumn}</span>
              </div>
            )}
            {selectedEdge.data?.targetColumn && (
              <div className="info-section">
                <label>Target Column:</label>
                <span className="info-value info-value-code">{selectedEdge.data.targetColumn}</span>
              </div>
            )}
            {selectedEdge.label && (
              <div className="info-section">
                <label>Label:</label>
                <span className="info-value">{selectedEdge.label}</span>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};

export default ErdInfoPanel;
