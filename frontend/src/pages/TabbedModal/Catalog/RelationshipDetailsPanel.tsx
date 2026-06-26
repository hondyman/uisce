// React default import not required with new JSX transform
import React, { useState, useEffect } from 'react';
import { Edge, Node as FlowNode } from 'reactflow';
import { IconButton, Tooltip, TextField, Button, Stack, Box } from '@mui/material';
import { Edit as EditIcon, Delete as DeleteIcon, Close as CloseIcon } from '@mui/icons-material';
import '../../../styles/RelationshipDetailsPanel.css'; // Import your CSS styles
import mergeProperties from '../../../utils/mergeProperties';

interface RelationshipDetailsPanelProps {
  edge?: Edge;
  nodes: FlowNode[];
  onClose?: () => void;
  className?: string;
  onEdit?: (id: string, updates: any) => void;
  onDelete?: (id: string) => void;
}

const RelationshipDetailsPanel: React.FC<RelationshipDetailsPanelProps> = ({ 
  edge, 
  nodes, 
  onClose, 
  className = '',
  onEdit,
  onDelete
}) => {
  // State for editing
  const [isEditing, setIsEditing] = useState(false);
  const [description, setDescription] = useState('');

  useEffect(() => {
    if (edge) {
      setDescription(edge.data?.description || '');
      setIsEditing(false);
    }
  }, [edge]);

  // If no edge is selected, show empty state
  if (!edge) {
    return (
      <div className={`relationship-details-panel ${className}`}>
        <div className="no-relationship-selected">
          <div className="icon">🔗</div>
          <h3>No Relationship Selected</h3>
          <p>Click on a relationship edge in the diagram to view its details</p>
        </div>
      </div>
    );
  }

  const sourceNode = nodes.find((n) => n.id === edge.source);
  const targetNode = nodes.find((n) => n.id === edge.target);

  // Converts snake_case or camelCase to Title Case. e.g., 'business_term' -> 'Business Term'
  const toTitleCase = (str: string): string => {
    if (!str) return '';
    return str
      .replace(/_/g, ' ') // Replace underscores with spaces
      .replace(/([a-z])([A-Z])/g, '$1 $2') // Add space before capital letters
      .replace(/\b\w/g, char => char.toUpperCase()); // Capitalize first letter of each word
  };

  // Helper function to get node type display name
  const getNodeTypeDisplay = (node: FlowNode | undefined): string => {
    if (!node) return 'Unknown Node';
    const typeIdentifier = node.data?.nodeType || node.data?.type || node.type;
    return toTitleCase(typeIdentifier || 'Unknown Node Type');
  };

  // Helper function to get node color based on type
  const getNodeColor = (node: FlowNode | undefined): string => {
    if (!node) return '#6b7280'; // Default gray
    
    // Gather all possible type identifiers
    const nodeType = (node.data?.nodeType || '').toLowerCase();
    const type = (node.data?.type || node.type || '').toLowerCase();
    const catalogType = (node.data?.catalog_type_name || node.data?.catalog_type || '').toLowerCase();
    const nodeTypeId = node.data?.node_type_id || '';
    
    // Known semantic term node type ID
    const semanticTermTypeId = '820b942a-9c9e-4abc-acdc-84616db33098';
    
    // Check for semantic/business terms
    if (nodeTypeId === semanticTermTypeId ||
        nodeType.includes('semantic') || type.includes('semantic') || catalogType.includes('semantic') ||
        nodeType.includes('business') || type.includes('business') || catalogType.includes('business')) {
      return '#6366f1'; // Indigo (Semantic/Business)
    }
    
    // Check for columns
    if (nodeType.includes('column') || type.includes('column') || catalogType.includes('column') ||
        node.data?.parent_name) { // Has parent table = likely a column
      return '#10b981'; // Green (Column)
    }
    
    // Check for tables/schemas
    if (nodeType.includes('table') || type.includes('table') || catalogType.includes('table') ||
        nodeType.includes('schema') || type.includes('schema') || catalogType.includes('schema')) {
      return '#6b7280'; // Gray (Table/Schema)
    }
    
    return '#6b7280'; // Default gray
  };

  // Helper function to get edge type display name
  const getEdgeTypeDisplay = (edge: Edge): string => {
    const relType = edge.data?.relationship_type;
    if (relType) {
      // Convert CamelCase to Title Case with arrows
      return toTitleCase(relType)
        .replace(/To/g, '→')
        .trim();
    }
    
    // Check for database relationship
    if (edge.data?.constraintName) {
      return 'Foreign Key Constraint';
    }
    if (edge.data?.fromColumn) {
      return 'Database Foreign Key';
    }
    
    // Check edge label
    if (edge.label) {
      return `${edge.label} Relationship`;
    }
    
    return toTitleCase(edge.type || 'Generic Relationship');
  };

  // Helper function to get additional edge details
  const getEdgeDetails = (edge: Edge): string[] => {
    const details: string[] = [];
    
    if (edge.data?.relationship_type) {
      details.push(`Type: ${edge.data.relationship_type}`);
    }
    
    if (edge.data?.constraintName) {
      details.push(`Constraint: ${edge.data.constraintName}`);
    }
    
    if (edge.data?.fromColumn && edge.data?.toColumn) {
      details.push(`Mapping: ${edge.data.fromColumn} → ${edge.data.toColumn}`);
    }
    
    // Merge type-level defaults (edge.data.edge_defn or edge.data.catalog_defn) with instance properties.
    // Instance properties take precedence.
    const typeDefaults = edge.data?.edge_defn || edge.data?.catalog_defn || undefined;
    const mergedProps = mergeProperties(typeDefaults, edge.data?.properties);
    if (mergedProps) {
      if (mergedProps.schema) details.push(`Schema: ${mergedProps.schema}`);
      if (mergedProps.table) details.push(`Table: ${mergedProps.table}`);
      if (mergedProps.column) details.push(`Column: ${mergedProps.column}`);
    }
    
    return details;
  };

  const edgeDetails = getEdgeDetails(edge);

  const handleSave = () => {
    if (onEdit) {
      onEdit(edge.id, { description });
    }
    setIsEditing(false);
  };

  const handleDelete = () => {
    if (onDelete && confirm('Are you sure you want to delete this relationship?')) {
      onDelete(edge.id);
      if (onClose) onClose();
    }
  };

  return (
    <div className={`relationship-details-panel ${className}`}>
      <div className="panel-header">
        <h3>Relationship Details</h3>
        <div className="header-actions">
           {!isEditing && (
              <>
                <Tooltip title="Edit">
                  <IconButton size="small" onClick={() => setIsEditing(true)}>
                    <EditIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
                <Tooltip title="Delete">
                  <IconButton size="small" onClick={handleDelete} color="error">
                    <DeleteIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
              </>
           )}
           {onClose && (
             <IconButton size="small" onClick={onClose}>
               <CloseIcon fontSize="small" />
             </IconButton>
           )}
        </div>
      </div>

      <div className="panel-content">
        <div className="detail-section">
          <div className="detail-title">{getEdgeTypeDisplay(edge)}</div>
        </div>

        {/* Source Node */}
        <div className="detail-section node-details">
          <div className="node-header">
            <span className="node-role">Source</span>
            <span className="node-type-badge" style={{ backgroundColor: getNodeColor(sourceNode) }}>
              {getNodeTypeDisplay(sourceNode)}
            </span>
          </div>
          <div className="node-name">{sourceNode?.data?.label || sourceNode?.id}</div>
        </div>

        <div className="arrow-connector">↓</div>

        {/* Target Node */}
        <div className="detail-section node-details">
          <div className="node-header">
            <span className="node-role">Target</span>
            <span className="node-type-badge" style={{ backgroundColor: getNodeColor(targetNode) }}>
              {getNodeTypeDisplay(targetNode)}
            </span>
          </div>
          <div className="node-name">{targetNode?.data?.label || targetNode?.id}</div>
        </div>
        
        <div className="detail-section">
          <div className="detail-subtitle">Description</div>
          {isEditing ? (
             <Box sx={{ mt: 1 }}>
               <TextField
                 fullWidth
                 multiline
                 rows={3}
                 size="small"
                 value={description}
                 onChange={(e) => setDescription(e.target.value)}
                 placeholder="Enter description..."
               />
               <Stack direction="row" spacing={1} sx={{ mt: 1, justifyContent: 'flex-end' }}>
                 <Button size="small" onClick={() => setIsEditing(false)}>Cancel</Button>
                 <Button size="small" variant="contained" onClick={handleSave}>Save</Button>
               </Stack>
             </Box>
          ) : (
            <p className="description-text">
               {edge.data?.description || <span className="placeholder">No description provided</span>}
            </p>
          )}
        </div>

        {edgeDetails.length > 0 && (
          <div className="detail-section">
            <div className="detail-subtitle">Details</div>
            <ul className="details-list">
              {edgeDetails.map((detail) => <li key={detail}>{detail}</li>)}
            </ul>
          </div>
        )}

        {((typeof process !== 'undefined' && process.env?.NODE_ENV === 'development') || import.meta.env.DEV) && (
          <details className="debug-section">
            <summary>Debug Info</summary>
            <pre>
              <strong>Edge:</strong>
              {JSON.stringify(edge, null, 2)}
            </pre>
            <pre>
              <strong>Source Node:</strong>
              {JSON.stringify(sourceNode, null, 2)}
            </pre>
            <pre>
              <strong>Target Node:</strong>
              {JSON.stringify(targetNode, null, 2)}
            </pre>
          </details>
        )}
      </div>
    </div>
  );
};

export default RelationshipDetailsPanel;