
import type { FC } from 'react';
import type { RelatedEntity } from '../../api/relationships';
import './RelationshipDetailsPanel.css';
import SVGIcon from './SVGIcon';

interface RelationshipDetailsPanelProps {
  selectedObject: { type: 'node' | 'edge'; data: RelatedEntity } | null;
  onClose: () => void;
  entityName: string;
}

const RelationshipDetailsPanel: FC<RelationshipDetailsPanelProps> = ({ selectedObject, onClose, entityName }) => {
  if (!selectedObject) {
    return null;
  }

  const { type, data } = selectedObject;
  const isEdge = type === 'edge';
  const title = isEdge ? 'Relationship Details' : data.targetEntity;

  return (
    <div className="relationship-details-panel">
        <div className="panel-header">
        <h2>{title}</h2>
        <button onClick={onClose} className="close-button" aria-label="Close">
          <SVGIcon name="close" ariaLabel="close" />
        </button>
      </div>
      <div className="panel-content">
        {isEdge ? (
          <>
            <div className="relationship-path">
              <span>{entityName}</span>
              <SVGIcon name="arrow_forward" className="mx-2 text-slate-400" ariaLabel="to" />
              <span>{data.targetEntity}</span>
            </div>
            <div className="cardinality-display">
              <p>Cardinality</p>
              <p>{data.cardinality}</p>
            </div>
            <div className="linking-mechanism">
              <p>Linking Mechanism</p>
              <div>
                <SVGIcon name="link" className="inline-block mr-2" ariaLabel="link" />
                <code>{`${data.keyFields.source} → ${data.keyFields.target}`}</code>
              </div>
            </div>
            <div>
              <p>Description</p>
              <p>{data.description}</p>
            </div>
          </>
        ) : (
          <>
            <div className="entity-header">
              <h2>{data.targetEntity}</h2>
              <span className={`status-badge ${data.isApplied ? 'applied' : ''}`}>
                {data.isApplied ? 'Active' : 'Inactive'}
              </span>
            </div>
            <p className="entity-description">{data.description}</p>
            <div className="entity-fields">
              <h3>Fields</h3>
              <ul>
                {Object.entries(data.keyFields).map(([key, value]) => (
                  <li key={key}>
                    <span>{key}</span>
                    <span>{value}</span>
                  </li>
                ))}
              </ul>
            </div>
          </>
        )}
      </div>
    </div>
  );
};

export default RelationshipDetailsPanel;
