// React import removed (automatic JSX runtime in use)
import './DetailsPane.css';
import { formatEdgeLabel } from './lineageUtils';

interface DetailsPanelProps {
  edge: any;
  nodes: any[];
  lineageType: 'technical' | 'semantic';
  selectedNode?: any; // Add support for selected node
}

const DualLineageDetailsPanel: React.FC<DetailsPanelProps> = ({ 
  edge, 
  nodes, 
  lineageType,
  selectedNode 
}) => {
  // If a node is selected, show node details
  if (selectedNode) {
    const nodeType = selectedNode.data?.nodeType || selectedNode.type || 'Unknown';
    const metadata = selectedNode.data?.metadata || selectedNode.data?.properties || {};
    
    // Parse metadata if it's a string
    let parsedMetadata = metadata;
    if (typeof metadata === 'string') {
      try {
        parsedMetadata = JSON.parse(metadata);
      } catch (e) {
        parsedMetadata = {};
      }
    }

    return (
      <div className="details-root">
        <div className="details-header">
          <div className="details-header-left">
            <div className="details-header-icon">🔍</div>
          </div>
          <div className={`details-relationship-badge ${lineageType === 'technical' ? 'tech' : 'sem'}`}>
            Node Details
          </div>
        </div>

        <div className="details-section">
          <h5 className="details-section-title">Node Information</h5>
          <div className="details-box">
            <div className={`node-type-badge ${lineageType === 'technical' ? 'tech' : 'sem'}`}>
              {nodeType}
            </div>
            <div className="details-node-title" style={{ marginTop: '8px', fontSize: '16px', fontWeight: '600' }}>
              {selectedNode.data?.label || selectedNode.label || 'Unknown'}
            </div>
            {selectedNode.data?.description && (
              <div className="details-node-desc" style={{ marginTop: '8px' }}>
                {selectedNode.data.description}
              </div>
            )}
            {selectedNode.data?.qualifiedPath && (
              <div className="details-row" style={{ marginTop: '8px' }}>
                <strong>Path:</strong> {selectedNode.data.qualifiedPath}
              </div>
            )}
          </div>
        </div>

        {/* Display metadata/properties */}
        {Object.keys(parsedMetadata).length > 0 && (
          <div className="details-section">
            <h5 className="details-section-title">Properties</h5>
            <div className="details-box">
              {Object.entries(parsedMetadata).map(([key, value]: [string, any]) => {
                // Skip rendering complex nested objects or very long strings
                if (typeof value === 'object' && value !== null) {
                  return (
                    <div key={key} className="details-row">
                      <strong>{key.replace(/_/g, ' ').replace(/\b\w/g, (l: string) => l.toUpperCase())}:</strong>{' '}
                      {JSON.stringify(value).substring(0, 100)}
                      {JSON.stringify(value).length > 100 ? '...' : ''}
                    </div>
                  );
                }
                return (
                  <div key={key} className="details-row">
                    <strong>{key.replace(/_/g, ' ').replace(/\b\w/g, (l: string) => l.toUpperCase())}:</strong>{' '}
                    {String(value)}
                  </div>
                );
              })}
            </div>
          </div>
        )}
      </div>
    );
  }

  // Original edge selection logic
  if (!edge) {
    return (
      <div className="details-empty">
        <div className="details-empty-inner">
          <div className="details-empty-icon">{lineageType === 'technical' ? '🔗' : '💫'}</div>
          <h4 className="details-empty-title">No Selection</h4>
          <p className="details-empty-sub">Click on a node or edge to view details</p>
        </div>
      </div>
    );
  }

  const sourceNode = nodes.find((n) => n.id === edge.source);
  const targetNode = nodes.find((n) => n.id === edge.target);

  const getNodeTypeDisplay = (node: any) => {
    if (!node) return 'Unknown Node';
    return node.data?.nodeType || node.type || 'Unknown';
  };

  const getRelationshipTypeDisplay = (edge: any) => {
    if (lineageType === 'technical') {
      return edge.data?.relationship_type === 'foreign_key' ? 'Foreign Key' : 'Database Relationship';
    }

    return formatEdgeLabel(edge);
  };

  // Parse edge metadata if it's a string
  let edgeMetadata = edge.data?.metadata || {};
  if (typeof edgeMetadata === 'string') {
    try {
      edgeMetadata = JSON.parse(edgeMetadata);
    } catch (e) {
      edgeMetadata = {};
    }
  }

  return (
    <div className="details-root">
      <div className="details-header">
        <div className="details-header-left">
          <div className="details-header-icon">{lineageType === 'technical' ? '🔗' : '💫'}</div>
        </div>
        <div className={`details-relationship-badge ${lineageType === 'technical' ? 'tech' : 'sem'}`}>
          {getRelationshipTypeDisplay(edge)}
        </div>
      </div>

      {lineageType === 'technical' && edge.data && (
        <div className="details-section">
          <h5 className="details-section-title">Details</h5>
          <div className="details-box">
            {edge.data.constraintName && (
              <div className="details-row"><strong>Constraint:</strong> {edge.data.constraintName}</div>
            )}
            {edge.data.fromColumn && edge.data.toColumn && (
              <div className="details-row"><strong>Mapping:</strong> {edge.data.fromColumn} → {edge.data.toColumn}</div>
            )}
            {edge.data.onDeleteAction && (
              <div className="details-row"><strong>On Delete:</strong> {edge.data.onDeleteAction}</div>
            )}
            {edge.data.onUpdateAction && (
              <div className="details-row"><strong>On Update:</strong> {edge.data.onUpdateAction}</div>
            )}
          </div>
        </div>
      )}

      {/* Display edge metadata/properties */}
      {Object.keys(edgeMetadata).length > 0 && (
        <div className="details-section">
          <h5 className="details-section-title">Edge Properties</h5>
          <div className="details-box">
            {Object.entries(edgeMetadata).map(([key, value]: [string, any]) => {
              if (typeof value === 'object' && value !== null) {
                return (
                  <div key={key} className="details-row">
                    <strong>{key.replace(/_/g, ' ').replace(/\b\w/g, (l: string) => l.toUpperCase())}:</strong>{' '}
                    {JSON.stringify(value).substring(0, 100)}
                    {JSON.stringify(value).length > 100 ? '...' : ''}
                  </div>
                );
              }
              return (
                <div key={key} className="details-row">
                  <strong>{key.replace(/_/g, ' ').replace(/\b\w/g, (l: string) => l.toUpperCase())}:</strong>{' '}
                  {String(value)}
                </div>
              );
            })}
          </div>
        </div>
      )}

      {lineageType === 'semantic' && edge.data?.properties && (
        <div className="details-section">
          <h5 className="details-section-title">Properties</h5>
          <div className="details-box">
            {Object.entries(edge.data.properties as Record<string, unknown>).map(([key, value]: [string, any]) => (
              <div key={key} className="details-row">
                <strong>{key.replace(/_/g, ' ').replace(/\b\w/g, (l: string) => l.toUpperCase())}:</strong>{' '}
                {String(value)}
              </div>
            ))}
          </div>
        </div>
      )}

        {sourceNode && (
        <div className="details-section">
          <h5 className="details-section-title">Source</h5>
          <div className="details-box">
            <div className={`node-type-badge ${lineageType === 'technical' ? 'tech' : 'sem'}`}>{getNodeTypeDisplay(sourceNode)}</div>
            <div className="details-node-title">{sourceNode?.data?.label || sourceNode?.label || 'Unknown'}</div>
            {sourceNode?.data?.description && (
              <div className="details-node-desc">{sourceNode.data.description}</div>
            )}
          </div>
        </div>
      )}

      {targetNode && (
        <div className="details-section">
          <h5 className="details-section-title">Target</h5>
          <div className="details-box">
            <div className={`node-type-badge ${lineageType === 'technical' ? 'tech-target' : 'sem-target'}`}>{getNodeTypeDisplay(targetNode)}</div>
            <div className="details-node-title">{targetNode?.data?.label || targetNode?.label || 'Unknown'}</div>
            {targetNode?.data?.description && (
              <div className="details-node-desc">{targetNode.data.description}</div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default DualLineageDetailsPanel;