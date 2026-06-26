// React import removed (automatic JSX runtime in use)
import { lazy, Suspense, useState } from 'react';
import { Node as FlowNode, Edge } from 'reactflow';
import styles from './CatalogDetailsPane.module.css';
import RelationshipDetailsPanel from './RelationshipDetailsPanel';
import mergeProperties from '../../../utils/mergeProperties';
import '../../../styles/CatalogDetailsPane.css';
import CoreCustomIndicator from './../../../components/CoreCustomIndicator';
import { BusinessTermRelationship } from '../../../types';
import { EnhancedSelectedAsset } from '../../../types/SemanticTypes';
import AddEdgeDialog from '../../../components/AddEdgeDialog';
import EditColumnDialog from '../../../components/EditColumnDialog';
import KeyDetailsModal from '../../../components/KeyDetailsModal';
import { AddLink as AddLinkIcon, Edit as EditIcon, Assessment as AssessmentIcon } from '@mui/icons-material';
import { IconButton, Tooltip } from '@mui/material';
import { useUpdateTermEdge, useDeleteTermEdge } from '../../../api/glossary';
import TableDataProfileModal from '../../../components/TableDataProfileModal';

const DualLineageViewer = lazy(() => import('./DualLineageViewer'));

type CombinedSelectedAsset = EnhancedSelectedAsset;

interface DetailsPaneProps {
  selectedAsset: CombinedSelectedAsset | null;
  nodes: FlowNode[];
  edges: Edge[];
  businessTerms?: any[];
  semanticTerms?: any[];
  semanticViews?: any[];
  onEdgeClick?: (event: React.MouseEvent, edge: Edge) => void;
  isRelationshipPanelOpen: boolean;
  selectedEdge: Edge | null;
  onCloseRelationshipPanel: () => void;
  forceLineageType?: 'technical' | 'semantic';
  processedTechnicalData: any;
  processedSemanticData: any;
  onAssetSelect: (asset: EnhancedSelectedAsset) => void;
  hierarchicalData?: any | null;
  preferHierarchical?: boolean;
  onOpenColumnsModal?: (tableLabel: string, columns: any[]) => void;
  onRefresh?: () => void;
}

interface MappedColumn {
  edge: Edge;
  schema: string;
  table: string;
  column: string;
  node: FlowNode | undefined;
}

const DetailsPane: React.FC<DetailsPaneProps> = ({ 
  selectedAsset, 
  nodes, 
  edges, 
  businessTerms = [],
  semanticTerms = [],
  semanticViews = [],
  onEdgeClick, 
  isRelationshipPanelOpen, 
  selectedEdge, 
  onCloseRelationshipPanel,
  forceLineageType,
  processedTechnicalData,
  processedSemanticData,
  onAssetSelect,
  hierarchicalData,
  preferHierarchical,
  onOpenColumnsModal,
  onRefresh
}) => {
  const [addEdgeDialogOpen, setAddEdgeDialogOpen] = useState(false);
  const [editColumnDialogOpen, setEditColumnDialogOpen] = useState(false);
  const [columnToEdit, setColumnToEdit] = useState<any>(null);
  const [keyModalOpen, setKeyModalOpen] = useState(false);
  const [keyModalType, setKeyModalType] = useState<'primary' | 'foreign'>('primary');
  const [dataProfileModalOpen, setDataProfileModalOpen] = useState(false);

  const updateEdgeMutation = useUpdateTermEdge();
  const deleteEdgeMutation = useDeleteTermEdge();

  const handleEdgeUpdate = (id: string, updates: any) => {
    updateEdgeMutation.mutate(
      { id, updates },
      {
        onSuccess: () => {
          if (onRefresh) onRefresh();
        },
        onError: (err) => {
          console.error("Failed to update edge:", err);
          alert("Failed to update edge"); // Basic error handling for now
        }
      }
    );
  };

  const handleEdgeDelete = (id: string) => {
    deleteEdgeMutation.mutate(id, {
      onSuccess: () => {
        if (onRefresh) onRefresh();
        onCloseRelationshipPanel();
      },
      onError: (err) => {
        console.error("Failed to delete edge:", err);
        alert("Failed to delete edge");
      }
    });
  };

  if (!selectedAsset) {
    return (
      <div className="details-placeholder">
        <p>Select a table, column, or business term to view details</p>
      </div>
    );
  }

  // Local safe accessors to avoid ad-hoc `as any` casts throughout the render
  const sel = selectedAsset as CombinedSelectedAsset & Record<string, unknown>;
  const selNode = (sel.node as unknown) as Record<string, unknown> | undefined;
  // local narrowers
  const asRecord = (v: unknown): Record<string, unknown> => (v && typeof v === 'object' ? (v as Record<string, unknown>) : {});
  const getColumnsFrom = (v: unknown): unknown[] => {
    if (Array.isArray(v)) return v as unknown[];
    const rec = asRecord(v);
    const cols = rec.columns;
    return Array.isArray(cols) ? (cols as unknown[]) : [];
  };

  const selNodeData = asRecord(selNode?.data || undefined);
  const selNodeProps = asRecord(selNodeData.properties || undefined);
  const selColumns = Array.isArray(sel.columns) ? (sel.columns as unknown[]) : getColumnsFrom(selNodeData);
  const selColumn = asRecord(sel.column ?? undefined);

  const getBusinessTermRelationships = (): BusinessTermRelationship[] => {
    if (selectedAsset.type !== 'column') return [];
    
    const relationships: BusinessTermRelationship[] = [];
    
    const semanticViewEdges = edges.filter(edge => {
      if (edge.data?.relationship_type !== 'SemanticViewToColumn') return false;
      const typeDefaults = edge.data?.edge_defn || edge.data?.catalog_defn || undefined;
      const merged = mergeProperties(typeDefaults, edge.data?.properties);
      return merged?.column === selectedAsset.columnName &&
             merged?.table === selectedAsset.tableName?.split('.')[1] &&
             merged?.schema === selectedAsset.tableName?.split('.')[0];
    });
    
    semanticViewEdges.forEach(edge => {
      const semanticView = semanticViews.find(sv => sv.id === edge.source);
      if (semanticView) {
        const semanticTermEdge = edges.find(e => 
          e.data?.relationship_type === 'SemanticToView' && 
          e.target === semanticView.id
        );
        
        if (semanticTermEdge) {
          const semanticTerm = semanticTerms.find(st => st.id === semanticTermEdge.source);
          if (semanticTerm) {
            const businessTerm = businessTerms.find(bt => bt.id === semanticTerm.parent_id);
            if (businessTerm) {
              const relationship: BusinessTermRelationship = {
                businessTerm,
                semanticTerm,
                semanticView,
              };
              relationships.push(relationship);
            }
          }
        }
      }
    });
    
    return relationships;
  };

  const getSemanticTermsForBusinessTerm = () => {
    if (selectedAsset.type !== 'business_term') return [];
    return semanticTerms.filter(st => st.parent_id === selectedAsset.nodeId);
  };

  const getTableSemanticTerms = () => {
    if (selectedAsset.type !== 'table') return {};
    // Extract schema and table from tableName (format: schema.table)
    const tableNameFull = selectedAsset.tableName || '';
    const parts = tableNameFull.split('.');
    const schema = parts.length > 1 ? parts[0] : undefined;
    const table = parts.length > 1 ? parts[1] : parts[0];
    
    const columnTerms: Record<string, any[]> = {};

    const semanticViewEdges = edges.filter(edge => {
      if (edge.data?.relationship_type !== 'SemanticViewToColumn') return false;
      const typeDefaults = edge.data?.edge_defn || edge.data?.catalog_defn || undefined;
      const merged = mergeProperties(typeDefaults, edge.data?.properties);
      // Check if edge points to this table
      return (merged?.table === table && (!schema || merged?.schema === schema)) || 
             (edge.target === selectedAsset.nodeId); // Fallback: checks if target is the table node
    });

    semanticViewEdges.forEach(edge => {
      const typeDefaults = edge.data?.edge_defn || edge.data?.catalog_defn || undefined;
      const merged = mergeProperties(typeDefaults, edge.data?.properties);
      const columnName = merged?.column;
      
      if (!columnName) return;

      const semanticViewId = edge.source;
      const semanticView = semanticViews.find(sv => sv.id === semanticViewId);
      
      if (semanticView) {
          const termEdges = edges.filter(e => 
              e.data?.relationship_type === 'SemanticToView' && 
              e.target === semanticViewId
          );

          termEdges.forEach(te => {
              const termNode = semanticTerms.find(st => st.id === te.source);
              if (termNode) {
                  if (!columnTerms[columnName]) columnTerms[columnName] = [];
                  if (!columnTerms[columnName].find(t => t.id === termNode.id)) {
                       columnTerms[columnName].push(termNode);
                  }
              }
          });
      }
    });

    return columnTerms;
  };

  const getColumnsForSemanticModel = () => {
    if (selectedAsset.type !== 'semantic_model') return [];
    
    const columnEdges = edges.filter(e => 
      e.data?.relationship_type === 'SemanticViewToColumn' && 
      e.source === selectedAsset.nodeId
    );
    
    return columnEdges.map(edge => {
      const typeDefaults = edge.data?.edge_defn || edge.data?.catalog_defn || undefined;
      const merged = mergeProperties(typeDefaults, edge.data?.properties);
      return ({
        edge,
        schema: merged?.schema,
        table: merged?.table,
        column: merged?.column,
        node: nodes.find(n => n.id === edge.target)
      });
    }).filter(item => item.node);
  };

  const getGenericRelationships = () => {
    // Filter edges connected to this node that aren't the standard lineage types we already display
    const standardTypes = ['SemanticViewToColumn', 'SemanticToView', 'DataFlow', 'Lineage'];
    
    return edges.filter(e => 
      (e.source === selectedAsset.nodeId || e.target === selectedAsset.nodeId) &&
      !standardTypes.includes(e.data?.relationship_type)
    ).map(e => {
        const isSource = e.source === selectedAsset.nodeId;
        const otherNodeId = isSource ? e.target : e.source;
        const otherNode = nodes.find(n => n.id === otherNodeId);
        return {
            edge: e,
            otherNode,
            direction: isSource ? 'outgoing' : 'incoming',
            type: e.data?.relationship_type || 'Related to'
        };
    }).filter(r => r.otherNode);
  };

  const genericRelationships = getGenericRelationships();

  const businessTermRelationships: BusinessTermRelationship[] = getBusinessTermRelationships();
  const relatedSemanticTerms = getSemanticTermsForBusinessTerm();
  const relatedColumns: MappedColumn[] = getColumnsForSemanticModel();

  // Helper to check if a value is meaningful (not empty/null/undefined)
  const hasValue = (value: unknown): boolean => {
    if (value === null || value === undefined || value === '') return false;
    if (Array.isArray(value)) return value.length > 0;
    if (typeof value === 'object' && Object.keys(value as Record<string, unknown>).length === 0) return false;
    // Filter out strings that literally say "not set" or similar placeholder text
    if (typeof value === 'string') {
      const lowerValue = value.toLowerCase().trim();
      if (lowerValue === 'not set' || lowerValue === 'n/a' || lowerValue === 'unknown' || lowerValue === 'undefined') return false;
    }
    return true;
  };

  const renderTableDetails = () => (
    <div className="details-grid">
      {(hasValue(selectedAsset.tableName) || hasValue(selNodeData.schema)) && (
        <div className="detail-card">
          <div className="detail-card-header">
            <div className="header-with-indicator">
              <h4 className="detail-card-title">
                <span className="detail-card-icon">📋</span>
                Basic Information
              </h4>
                <div className={styles.headerActions}>
                <CoreCustomIndicator isCore={selectedAsset.isCore} />
                <Tooltip title="View Data Profile">
                  <IconButton size="small" onClick={() => setDataProfileModalOpen(true)}>
                    <AssessmentIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
                <Tooltip title="Add Relationship">
                  <IconButton size="small" onClick={() => setAddEdgeDialogOpen(true)}>
                    <AddLinkIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
              </div>
            </div>
          </div>
          <div className="basic-info-grid">
            {hasValue(selectedAsset.tableName) && (
              <div className="detail-field">
                <label>Table Name</label>
                <span>{selectedAsset.tableName}</span>
              </div>
            )}
            {hasValue(selNodeData.schema) && (
              <div className="detail-field">
                <label>Schema</label>
                <span>{String(selNodeData.schema)}</span>
              </div>
            )}
          </div>
        </div>
      )}

      {hasValue(selColumns) && (
        <div className="detail-card" role="button" tabIndex={0} onClick={() => {
          // when clicked, open columns modal if handler provided
          // derive columns robustly
          let cols: any[] = [];
          if (Array.isArray(selColumns)) cols = selColumns as any[];
          const candidateNode = selNode || nodes.find(n => String(n.id) === String(selectedAsset.nodeId) || n.data?.tableName === selectedAsset.tableName || n.data?.qualifiedPath === selectedAsset.tableName);
          if (cols.length === 0 && candidateNode && Array.isArray(candidateNode.data?.columns)) cols = candidateNode.data.columns as unknown[];
          // fallback: gather nodes that look like columns
          if (cols.length === 0) {
            const tableName = sel.tableName as string | undefined;
            cols = nodes.filter(n => n.data && (n.data.table === tableName || n.data.tableName === tableName || n.data.qualifiedPath === tableName)).map(n => {
              const nd = n.data as Record<string, unknown>;
              return ({ name: String(nd.label || nd['column'] || n.id), ...nd });
            });
          }
          if (onOpenColumnsModal) {
            const columnTerms = getTableSemanticTerms();
            const enrichedCols = cols.map((col: any) => ({
              ...col, 
              semanticTerms: columnTerms[col.name || col.column] || []
            }));
            onOpenColumnsModal((sel.tableName as string) || (sel.name as string) || 'table', enrichedCols);
          }
        }}>
          <div className="detail-card-header">
            <h4 className="detail-card-title">
              <span className="detail-card-icon">📊</span>
              Statistics
            </h4>
          </div>
          <div className="stats-card">
            <div className="stats-number">{(Array.isArray(selColumns) ? selColumns.length : (nodes.find(n => n.id === (selNode?.id || selectedAsset.nodeId || selectedAsset.id))?.data?.columns?.length || 0))}</div>
            <div className="stats-label">Total Columns</div>
          </div>
        </div>
      )}

      {hasValue(selColumns) && Array.isArray(selColumns) && selColumns.length > 0 && (
        <div className={`detail-card ${styles.gridColSpanFull}`}>
          <div className="detail-card-header">
            <h4 className="detail-card-title">
              <span className="detail-card-icon">🔗</span>
              Constraints Summary
            </h4>
          </div>
          <div className="column-types">
            <div 
              className="type-count primary" 
              style={{ cursor: 'pointer' }}
              onClick={(e) => { e.stopPropagation(); setKeyModalType('primary'); setKeyModalOpen(true); }}
              role="button"
              tabIndex={0}
              title="Click to view primary key columns"
            >
              <span className="key-icon" style={{filter: 'grayscale(100%) brightness(0) sepia(100%) hue-rotate(-50deg) saturate(600%) contrast(0.8)'}}>🔑</span>
              Primary Keys: {Array.isArray(selColumns) ? selColumns.filter((c) => Boolean((c as Record<string, unknown>)?.isPrimaryKey)).length : 0}
            </div>
            <div 
              className="type-count foreign" 
              style={{ cursor: 'pointer' }}
              onClick={(e) => { e.stopPropagation(); setKeyModalType('foreign'); setKeyModalOpen(true); }}
              role="button"
              tabIndex={0}
              title="Click to view foreign key columns"
            >
              <span className="key-icon" style={{filter: 'grayscale(100%) brightness(0) sepia(100%) hue-rotate(190deg) saturate(600%) contrast(0.8)'}}>🔗</span>
              Foreign Keys: {Array.isArray(selColumns) ? selColumns.filter((c) => Boolean((c as Record<string, unknown>)?.isForeignKey)).length : 0}
            </div>
          </div>
        </div>
      )}


    </div>
  );

  const renderColumnDetails = () => (
    <>
      <div className="details-grid">
        {(hasValue(selectedAsset.columnName) || hasValue(selectedAsset.tableName) || hasValue((selColumn as Record<string, unknown>)?.['type'])) && (
          <div className="detail-card">
            <div className="detail-card-header">
              <div className="header-with-indicator">
                <h4 className="detail-card-title">
                  <span className="detail-card-icon">📄</span>
                  Basic Information
                </h4>
                <div className={styles.headerActions}>
                  <CoreCustomIndicator isCore={selectedAsset.isCore} />
                  <Tooltip title="Edit Column">
                    <IconButton size="small" onClick={() => setEditColumnDialogOpen(true)}>
                      <EditIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                  <Tooltip title="Add Relationship">
                    <IconButton size="small" onClick={() => setAddEdgeDialogOpen(true)}>
                      <AddLinkIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                </div>
              </div>
            </div>
            <div className="basic-info-grid">
              {hasValue(selectedAsset.columnName) && (
                <div className="detail-field">
                  <label>Column Name</label>
                  <span>{selectedAsset.columnName}</span>
                </div>
              )}
              {hasValue(selectedAsset.tableName) && (
                <div className="detail-field">
                  <label>Table</label>
                  <span>{selectedAsset.tableName}</span>
                </div>
              )}
              {hasValue((selColumn as Record<string, unknown>)?.['type']) && (
                <div className="detail-field highlighted">
                  <label>Data Type</label>
                  <span>{String((selColumn as Record<string, unknown>)?.['type'])}</span>
                </div>
              )}
            </div>
          </div>
        )}

        {(hasValue((selColumn as Record<string, unknown>)?.['isPrimaryKey']) || hasValue((selColumn as Record<string, unknown>)?.['isForeignKey'])) && (
          <div className="detail-card">
            <div className="detail-card-header">
              <h4 className="detail-card-title">
                <span className="detail-card-icon">🏷️</span>
                Constraints
              </h4>
            </div>
            <div className="constraint-tags">
              {Boolean((selColumn as Record<string, unknown>)?.['isPrimaryKey']) && <span className="tag primary">Primary Key</span>}
              {Boolean((selColumn as Record<string, unknown>)?.['isForeignKey']) && <span className="tag foreign">Foreign Key</span>}
              {!(hasValue((selColumn as Record<string, unknown>)?.['isPrimaryKey'])) && !(hasValue((selColumn as Record<string, unknown>)?.['isForeignKey'])) && (
                <span className="tag regular">Regular Column</span>
              )}
            </div>
          </div>
        )}
      </div>
      
      {businessTermRelationships.length > 0 && (
        <div className="business-term-relationships">
          <div className="detail-card-header">
            <h4 className="detail-card-title">
              <span className="detail-card-icon">💼</span>
              Business Context
            </h4>
          </div>
          <div className="business-relationships">
            {businessTermRelationships.map((rel: BusinessTermRelationship) => (
              <div key={`${rel.businessTerm?.node_name}-${rel.semanticTerm?.node_name}`} className="business-relationship">
                <div className="relationship-chain">
                  <span className="business-term">
                    💼 {rel.businessTerm?.node_name}
                  </span>
                  <span className="arrow">→</span>
                  <span className="semantic-term">
                    🔍 {rel.semanticTerm?.node_name}
                  </span>
                  <span className="arrow">→</span>
                  <span className="semantic-view">
                    👁️ {rel.semanticView?.node_name}
                  </span>
                </div>
                {rel.businessTerm?.description && (
                  <div className="business-description">
                    {rel.businessTerm.description}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {genericRelationships.length > 0 && (
        <div className="detail-card">
          <div className="detail-card-header">
            <h4 className="detail-card-title">
              <span className="detail-card-icon">🔗</span>
              Other Relationships
            </h4>
          </div>
          <div className="basic-info-grid">
             {genericRelationships.map((rel, idx) => (
                 <div key={idx} className="detail-field">
                     <label>{rel.type} ({rel.direction === 'outgoing' ? 'to' : 'from'})</label>
                     <span>{rel.otherNode?.data?.label || rel.otherNode?.id}</span>
                 </div>
             ))}
          </div>
        </div>
      )}
    </>
  );

  const renderBusinessTermDetails = () => (
    <div className="details-grid">
      <div className="detail-card">
        <div className="detail-card-header">
          <div className="header-with-indicator">
            <h4 className="detail-card-title">
              <span className="detail-card-icon">💼</span>
              Basic Information
            </h4>
            <div className={styles.headerActions}>
              <CoreCustomIndicator isCore={selectedAsset.isCore} />
              <Tooltip title="Add Relationship">
                <IconButton size="small" onClick={() => setAddEdgeDialogOpen(true)}>
                  <AddLinkIcon fontSize="small" />
                </IconButton>
              </Tooltip>
            </div>
          </div>
        </div>
        <div className="basic-info-grid">
          {hasValue(selectedAsset.name) && (
            <div className="detail-field">
              <label>Business Term</label>
              <span>{selectedAsset.name}</span>
            </div>
          )}
          {hasValue(selNodeProps['business_term_id']) && (
            <div className="detail-field">
              <label>ID</label>
              <span>{String(selNodeProps['business_term_id'])}</span>
            </div>
          )}
          {hasValue(selNodeData.description) && (
            <div className="detail-field">
              <label>Description</label>
              <span>{String(selNodeData.description)}</span>
            </div>
          )}
          {hasValue(selNodeProps['category']) && (
            <div className="detail-field">
              <label>Category</label>
              <span>{String(selNodeProps['category'])}</span>
            </div>
          )}
          {hasValue(selNodeProps['sub_category']) && (
            <div className="detail-field">
              <label>Sub-Category</label>
              <span>{String(selNodeProps['sub_category'])}</span>
            </div>
          )}
        </div>
      </div>

      <div className="detail-card">
        <div className="detail-card-header">
          <h4 className="detail-card-title">
            <span className="detail-card-icon"></span>
            Governance & Ownership
          </h4>
        </div>
        <div className="basic-info-grid">
          {Boolean(selNodeProps['owner']) && (
            <div className="detail-field">
              <label>Owner</label>
              <span>{String(selNodeProps['owner'])}</span>
            </div>
          )}
          {Boolean(selNodeProps['steward']) && (
            <div className="detail-field">
              <label>Data Steward</label>
              <span>{String(selNodeProps['steward'])}</span>
            </div>
          )}
          {Boolean(selNodeProps['status']) && (
            <div className="detail-field">
              <label>Status</label>
              <span className={`status-badge status-${String(selNodeProps['status'])}`}>
                {String(selNodeProps['status'])}
              </span>
            </div>
          )}
          {Boolean(selNodeProps['version']) && (
            <div className="detail-field">
              <label>Version</label>
              <span>{String(selNodeProps['version'])}</span>
            </div>
          )}
        </div>
      </div>

      {(hasValue(selNodeProps['pii']) || Array.isArray(selNodeProps?.classifications) && selNodeProps.classifications.length > 0 || Array.isArray(selNodeProps?.tags) && selNodeProps.tags.length > 0) && (
        <div className="detail-card">
          <div className="detail-card-header">
            <h4 className="detail-card-title">
              <span className="detail-card-icon">🏷️</span>
              Classification & Tags
            </h4>
          </div>
          {hasValue(selNodeProps['pii']) && (
            <div className="detail-field">
              <label>PII Status</label>
              <div className="constraint-tags">
                <span className={`tag ${selNodeProps['pii'] ? 'pii' : 'non-pii'}`}>
                  {selNodeProps['pii'] ? 'Contains PII' : 'No PII'}
                </span>
              </div>
            </div>
          )}

          {Array.isArray(selNodeProps?.classifications) && selNodeProps.classifications.length > 0 && (
            <div className="detail-field">
              <label>Classifications</label>
              <div className="classification-tags">
                {selNodeProps.classifications.map((classification: any) => (
                  <span key={String(classification)} className="tag classification">{String(classification)}</span>
                ))}
              </div>
            </div>
          )}

          {Array.isArray(selNodeProps?.tags) && selNodeProps.tags.length > 0 && (
            <div className="detail-field">
              <label>Tags</label>
              <div className="tag-list">
                {selNodeProps.tags.map((tag: any) => (
                  <span key={String(tag)} className="tag user-tag">{String(tag)}</span>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

  {selNodeProps?.parent_id != null && (
        <div className="detail-card">
          <div className="detail-card-header">
            <h4 className="detail-card-title">
              <span className="detail-card-icon">🔗</span>
              Hierarchy
            </h4>
          </div>
            <div className="detail-field">
            <label>Parent Term</label>
            <span>{selNodeProps.parent_id ? String(selNodeProps.parent_id) : ''}</span>
          </div>
        </div>
      )}
      
      {relatedSemanticTerms.length > 0 && (
        <div className={`detail-card ${styles.gridColSpanFull}`}>
          <div className="detail-card-header">
            <h4 className="detail-card-title">
              <span className="detail-card-icon">🔍</span>
              Semantic Terms
            </h4>
          </div>
          <div className="semantic-terms-list">
            {relatedSemanticTerms.map((term) => (
              <span key={term.node_name} className="tag semantic-term-tag">
                {term.node_name}
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  );

  const renderSemanticTermDetails = () => (
    <div className="details-single-column">
      {(hasValue((selectedAsset as EnhancedSelectedAsset).semanticTerm) || hasValue(selNodeData.description)) && (
        <div className="detail-card">
          <div className="detail-card-header">
            <div className="header-with-indicator">
              <h4 className="detail-card-title">
                <span className="detail-card-icon">🔍</span>
                Semantic Relationships
              </h4>
              <div className={styles.headerActions}>
                <CoreCustomIndicator isCore={selectedAsset.isCore} />
                <Tooltip title="Add Relationship">
                  <IconButton size="small" onClick={() => setAddEdgeDialogOpen(true)}>
                    <AddLinkIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
              </div>
            </div>
          </div>
          <div className="basic-info-grid">
            {hasValue((selectedAsset as EnhancedSelectedAsset).semanticTerm) && (
              <div className="detail-field">
                <label>Semantic Term</label>
                <span>{(selectedAsset as EnhancedSelectedAsset).semanticTerm}</span>
              </div>
            )}
            {hasValue(selNodeData.description) && (
              <div className="detail-field">
                <label>Description</label>
                <span>{String(selNodeData.description)}</span>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );

  const renderSemanticModelDetails = () => (
    <>
      <div className="details-grid">
        {(hasValue((selectedAsset as EnhancedSelectedAsset).semanticModel) || hasValue(selNodeData.description)) && (
          <div className="detail-card">
            <div className="detail-card-header">
              <div className="header-with-indicator">
                <h4 className="detail-card-title">
                  <span className="detail-card-icon">👁️</span>
                  Basic Information
                </h4>
                <div className={styles.headerActions}>
                  <CoreCustomIndicator isCore={selectedAsset.isCore} />
                  <Tooltip title="Add Relationship">
                    <IconButton size="small" onClick={() => setAddEdgeDialogOpen(true)}>
                      <AddLinkIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                </div>
              </div>
            </div>
            <div className="basic-info-grid">
              {hasValue((selectedAsset as EnhancedSelectedAsset).semanticModel) && (
                <div className="detail-field">
                  <label>Semantic View</label>
                  <span>{(selectedAsset as EnhancedSelectedAsset).semanticModel}</span>
                </div>
              )}
              {hasValue(selNodeData.description) && (
                <div className="detail-field">
                  <label>Description</label>
                  <span>{String(selNodeData.description)}</span>
                </div>
              )}
            </div>
          </div>
        )}

        {hasValue(relatedColumns) && (
          <div className="detail-card">
            <div className="detail-card-header">
              <h4 className="detail-card-title">
                <span className="detail-card-icon">📊</span>
                Mapping Statistics
              </h4>
            </div>
            <div className="stats-card">
              <div className="stats-number">{relatedColumns.length}</div>
              <div className="stats-label">Mapped Columns</div>
            </div>
          </div>
        )}
      </div>
      
      {relatedColumns.length > 0 && (
        <div className={`mapped-columns ${styles.gridColSpanFull}`}>
          <div className="detail-card-header">
            <h4 className="detail-card-title">
              <span className="detail-card-icon">🔗</span>
              Column Mappings
            </h4>
          </div>
          <div className="column-mappings">
            {relatedColumns.map((col: MappedColumn) => (
              <div key={`${col.schema}.${col.table}.${col.column}`} className="column-mapping">
                <span className="column-path">
                  {col.schema}.{col.table}.{col.column}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
    </>
  );

  const getHeaderIcon = () => {
    switch (selectedAsset.type) {
      case 'table': return '📋';
      case 'column': return '📄';
      case 'business_term': return '💼';
      case 'semantic_term': return '🔍';
      case 'semantic_model': return '👁️';
      default: return '🔗';
    }
  };

  const getHeaderTitle = () => {
    switch (selectedAsset.type) {
      case 'table': return 'Table Details';
      case 'column': return 'Column Details';
      case 'business_term': return 'Business Term Details';
      case 'semantic_term': return 'Term Details';
      case 'semantic_model': return 'Semantic Model Details';
      default: return 'Relationship Details';
    }
  };

  const handleEditColumn = (col: any) => {
      setColumnToEdit({
          id: col.id || selectedAsset.nodeId, // careful here with IDs
          name: col.name || col.column,
          description: col.description,
          type: col.type,
          properties: col.properties || {},
          tenant_id: selNode?.tenant_id, // inherit from table
          tenant_datasource_id: selNode?.tenant_datasource_id, 
      });
      setEditColumnDialogOpen(true);
  };

  return (
    <div className="details-pane">
      {/* ... header ... */}
      
      <div className="details-content">
        {selectedAsset.type === 'table' && renderTableDetails()}
         {/* ... other render calls ... */}
        {selectedAsset.type === 'column' && renderColumnDetails()}
        {selectedAsset.type === 'business_term' && renderBusinessTermDetails()}
        {selectedAsset.type === 'semantic_term' && renderSemanticTermDetails()}
        {selectedAsset.type === 'semantic_model' && renderSemanticModelDetails()}
        
        <div className="lineage-section lineage-container">
            {/* ... lineage viewer ... */}
           <Suspense fallback={<div className="lineage-loading">Loading lineage…</div>}>
            <DualLineageViewer
              selectedAsset={selectedAsset}
              technicalData={processedTechnicalData}
              semanticData={processedSemanticData}
              hierarchicalData={hierarchicalData}
              preferHierarchical={preferHierarchical}
              onAssetClick={onAssetSelect}
              forceLineageType={forceLineageType}
              onRelationshipClick={onEdgeClick ? (edge) => onEdgeClick({} as React.MouseEvent, edge) : undefined}
            />
          </Suspense>
          {selectedEdge && (
            <RelationshipDetailsPanel
              edge={selectedEdge}
              nodes={nodes}
              onClose={onCloseRelationshipPanel}
              className={isRelationshipPanelOpen ? 'open' : ''}
              onEdit={handleEdgeUpdate}
              onDelete={handleEdgeDelete}
            />
          )}
        </div>
      </div>
      <AddEdgeDialog
        open={addEdgeDialogOpen}
        onClose={() => setAddEdgeDialogOpen(false)}
        sourceNodeId={selectedAsset.nodeId}
        sourceNodeType={selectedAsset.type}
        onEdgeAdded={onRefresh}
      />
      <EditColumnDialog
        open={editColumnDialogOpen}
        onClose={() => {
            setEditColumnDialogOpen(false);
            setColumnToEdit(null);
        }}
        column={columnToEdit || {
          id: selectedAsset.nodeId,
          name: selectedAsset.columnName || selectedAsset.name,
          description: selectedAsset.node?.description,
          type: (selColumn as Record<string, unknown>)?.['type'] as string,
          properties: selectedAsset.node?.properties || {},
          tenant_id: selectedAsset.node?.tenant_id,
          tenant_datasource_id: selectedAsset.node?.tenant_datasource_id,
        }}
        onSave={onRefresh}
      />
      <KeyDetailsModal
        open={keyModalOpen}
        onClose={() => setKeyModalOpen(false)}
        tableName={selectedAsset.tableName || selectedAsset.name}
        keyType={keyModalType}
        columns={Array.isArray(selColumns) ? (selColumns as any[]) : []}
      />
      <TableDataProfileModal
        open={dataProfileModalOpen}
        onClose={() => setDataProfileModalOpen(false)}
        tableName={selectedAsset.tableName || selectedAsset.name || 'Table'}
        columns={
          Array.isArray(selColumns)
            ? (selColumns as any[]).map((col: any) => ({
                name: col.name || col.column || 'unknown',
                data_type: col.type || col.data_type || col.properties?.data_type,
                unique_count: col.unique_count ?? col.properties?.unique_count,
                total_count: col.total_count ?? col.properties?.total_count,
                cardinality_ratio: col.cardinality_ratio ?? col.properties?.cardinality_ratio,
                is_low_cardinality: col.is_low_cardinality ?? col.properties?.is_low_cardinality,
                is_nullable: col.is_nullable ?? col.properties?.is_nullable,
                sample_values: col.sample_values ?? col.properties?.sample_values,
                detected_format: col.detected_format ?? col.properties?.detected_format,
                max_length: col.max_length ?? col.properties?.max_length,
              }))
            : []
        }
      />
    </div>
  );

};

export default DetailsPane;