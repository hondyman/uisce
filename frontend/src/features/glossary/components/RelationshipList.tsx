import React, { useState } from 'react';
import { IconButton, Tooltip, CircularProgress } from '@mui/material';
import { Delete as DeleteIcon, Edit as EditIcon, Settings as PropertiesIcon } from '@mui/icons-material';
import { useDeleteTermEdge } from '../../../api/glossary';
import EditEdgeDialog from '../../../components/EditEdgeDialog';
import { getPredicate } from '../constants/predicates';

export interface CatalogEdge {
  id: string;
  edge_type_id?: string;
  edge_type_name: string;
  description?: string;
  subject_node_id: string;
  subject_node_type_id?: string;
  object_node_id: string;
  object_node_type_id?: string;
  properties?: Record<string, any>;
  is_active?: boolean;
  created_at?: string;
  updated_at?: string;
  tenant_id?: string;
  core_id?: string | null;
  source_node_id?: string;
  target_node_id?: string;
  source_name?: string;
  target_name?: string;
  relationship_type?: string;
  source_node_type?: string;
  target_node_type?: string;
  source_path?: string;
  target_path?: string;
  predicate?: string;
}

interface RelationshipListProps {
  edges: CatalogEdge[];
  nodes?: (CatalogEdge['subject_node_id' | 'object_node_id'] | Record<string, any>)[];
  selectedNodeId?: string | null;
  onDeleted?: () => void;
  onUpdated?: () => void;
  darkMode?: boolean;
  getNodeName?: (nodeId: string) => string;
  getNodePath?: (nodeId: string) => string | undefined;
}

interface ThemeColors {
  bg: string;
  panel: string;
  border: string;
  accent: string;
  text: string;
  textMuted: string;
  danger: string;
  blue: string;
  teal: string;
  warning: string;
}

const DARK_COLORS: ThemeColors = {
  bg: '#0A0C10',
  panel: '#13161E',
  border: 'rgba(255,255,255,0.07)',
  accent: '#6366F1',
  text: '#E2E8F0',
  textMuted: '#8892A4',
  danger: '#EF4444',
  blue: '#60A5FA',
  teal: '#2DD4BF',
  warning: '#F59E0B',
};

const LIGHT_COLORS: ThemeColors = {
  bg: '#ffffff',
  panel: '#f8f9fa',
  border: 'rgba(0,0,0,0.08)',
  accent: '#6366F1',
  text: '#1a1a2e',
  textMuted: '#6b7280',
  danger: '#dc2626',
  blue: '#3b82f6',
  teal: '#14b8a6',
  warning: '#d97706',
};

const Badge: React.FC<{ label: string; color: string; bg?: string }> = ({ label, color, bg }) => (
  <span style={{
    display: 'inline-flex', alignItems: 'center', padding: '1px 7px',
    borderRadius: 9999, fontSize: 10, fontWeight: 700, letterSpacing: '0.04em',
    color, background: bg ?? `${color}22`, border: `1px solid ${color}44`,
    fontFamily: 'monospace', textTransform: 'uppercase' as const,
  }}>
    {label}
  </span>
);

export const RelationshipList: React.FC<RelationshipListProps> = ({
  edges,
  nodes = [],
  selectedNodeId,
  onDeleted,
  onUpdated,
  darkMode = true,
  getNodeName,
  getNodePath,
}) => {
  const C = darkMode ? DARK_COLORS : LIGHT_COLORS;
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [editingEdge, setEditingEdge] = useState<CatalogEdge | null>(null);
  const deleteEdge = useDeleteTermEdge();

  const nodeMap = React.useMemo(() => {
    const map = new Map<string, any>();
    nodes.forEach(n => {
      if (n && typeof n === 'object' && 'id' in n) {
        map.set(n.id, n);
      }
    });
    return map;
  }, [nodes]);

  const resolveNode = (nodeId: string) => {
    return nodeMap.get(nodeId);
  };

  const handleDelete = async (edgeId: string) => {
    if (!window.confirm('Are you sure you want to delete this relationship?')) return;
    setDeletingId(edgeId);
    try {
      await deleteEdge.mutateAsync(edgeId);
      onDeleted?.();
    } catch (err) {
      console.error('Failed to delete relationship:', err);
      alert('Failed to delete relationship');
    } finally {
      setDeletingId(null);
    }
  };

  const handleEdgeUpdated = () => {
    onUpdated?.();
    onDeleted?.(); // Also refresh to show updated data
  };

  const hasProperties = (edge: CatalogEdge): boolean => {
    if (!edge.properties) return false;
    return Object.keys(edge.properties).length > 0;
  };

  // Resolve the canonical predicate key from whichever alias the backend carries
  const resolvePredicateKey = React.useCallback((edge: CatalogEdge): string => {
    return edge.predicate || edge.edge_type_name || edge.relationship_type || 'RELATED_TO';
  }, []);

  // Deduplicate edges by key (source, target, predicate)
  const uniqueEdges = React.useMemo(() => {
    if (!edges) return [];
    const seen = new Set<string>();
    const result: CatalogEdge[] = [];
    for (const edge of edges) {
      const sId = edge.subject_node_id || edge.source_node_id || '';
      const tId = edge.object_node_id || edge.target_node_id || '';
      const key = `${sId}->${tId}:${resolvePredicateKey(edge)}`;
      if (!seen.has(key)) {
        seen.add(key);
        result.push(edge);
      }
    }
    return result;
  }, [edges, resolvePredicateKey]);

  if (!uniqueEdges || uniqueEdges.length === 0) {
    return (
      <div style={{
        padding: 48, textAlign: 'center', color: C.textMuted,
        fontFamily: 'system-ui, sans-serif', fontSize: 14,
        background: C.panel, borderRadius: 12, border: `1px solid ${C.border}`,
      }}>
        <div style={{ fontSize: 32, marginBottom: 8, opacity: 0.5 }}>🔗</div>
        <div style={{ fontWeight: 600, color: C.text }}>No Relationships Found</div>
        <div style={{ fontSize: 12, marginTop: 4 }}>This entity has no direct ontology links or relationship edges.</div>
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12, padding: '16px 24px' }}>
      <div style={{ fontSize: 12, color: C.textMuted, marginBottom: 4 }}>
        Showing {uniqueEdges.length} connected relationship{uniqueEdges.length !== 1 ? 's' : ''}
      </div>

      {uniqueEdges.map((edge) => {
        const sourceId = edge.subject_node_id || edge.source_node_id;
        const targetId = edge.object_node_id || edge.target_node_id;
        const isOutbound = sourceId === selectedNodeId;
        const relatedNodeId = isOutbound ? targetId : sourceId;
        const relatedNode = relatedNodeId ? resolveNode(relatedNodeId) : undefined;

        let rawName = (isOutbound ? edge.target_name : edge.source_name) 
          || (relatedNodeId ? getNodeName?.(relatedNodeId) : undefined) 
          || relatedNode?.node_name 
          || relatedNode?.name 
          || relatedNodeId?.substring(0, 8) 
          || '';

        let parsedType = (isOutbound ? edge.target_node_type : edge.source_node_type) 
          || relatedNode?.catalog_type_name 
          || relatedNode?.node_type 
          || relatedNode?.type;

        let cleanName = rawName;
        if (rawName.startsWith('business_object/') || rawName.startsWith('/business_object/')) {
          parsedType = 'business_object';
          cleanName = rawName.split('/').filter(Boolean).pop() || rawName;
        } else if (rawName.startsWith('semantic_term/')) {
          parsedType = 'semantic_term';
          cleanName = rawName.replace('semantic_term/', '');
        } else if (rawName.startsWith('business_term/')) {
          parsedType = 'business_term';
          cleanName = rawName.replace('business_term/', '');
        } else if (rawName.startsWith('api_endpoint/') || rawName.startsWith('/api_endpoint/')) {
          parsedType = 'api_endpoint';
          cleanName = rawName.split('/').filter(Boolean).pop() || rawName;
        } else if (rawName.startsWith('/orm/') || rawName.startsWith('/public/')) {
          const parts = rawName.split('/').filter(Boolean);
          if (parts.length >= 3) {
            parsedType = 'column';
            cleanName = parts[parts.length - 1];
          } else if (parts.length === 2) {
            parsedType = 'table';
            cleanName = parts[1];
          }
        }

        const relatedPath = (isOutbound ? edge.target_path : edge.source_path) 
          || (relatedNodeId ? getNodePath?.(relatedNodeId) : undefined) 
          || relatedNode?.qualified_path 
          || (rawName.startsWith('/') ? rawName : undefined);

        const nodeTypeLower = (parsedType || 'node').toLowerCase();
        const isBO = nodeTypeLower.includes('business_object') || nodeTypeLower.includes('bo');
        const isApi = nodeTypeLower.includes('api_endpoint') || nodeTypeLower.includes('endpoint') || nodeTypeLower.includes('api');
        const isColumn = nodeTypeLower.includes('column');
        const isTable = nodeTypeLower.includes('table');
        const isSemTerm = nodeTypeLower.includes('semantic');
        const isBusTerm = nodeTypeLower.includes('business_term') || nodeTypeLower.includes('businessterm');

        const typeColor = isBO ? '#A855F7' : isApi ? '#38BDF8' : isColumn ? C.teal : isTable ? C.blue : isSemTerm ? C.accent : isBusTerm ? '#10B981' : C.textMuted;
        const typeIcon = isBO ? '🏢' : isApi ? '🌐' : isColumn ? '🏷️' : isTable ? '📊' : isSemTerm ? '🧠' : isBusTerm ? '💼' : '📄';
        const typeLabel = isBO ? 'Business Object' : isApi ? 'API Endpoint' : isColumn ? 'Database Column' : isTable ? 'Database Table' : isSemTerm ? 'Semantic Term' : isBusTerm ? 'Business Term' : (parsedType || 'Entity');

        const predicateMeta = getPredicate(resolvePredicateKey(edge));
        const relationshipLabel = predicateMeta.label;
        const edgeHasProps = hasProperties(edge);

        return (
          <div
            key={edge.id}
            style={{
              background: C.panel,
              border: `1px solid ${C.border}`,
              borderRadius: 10,
              padding: '14px 18px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 16,
              transition: 'all 0.15s ease',
              boxShadow: '0 2px 8px rgba(0,0,0,0.15)',
            }}
          >
            {/* Left: Direction & Predicate Badge */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, minWidth: 200 }}>
              <div style={{
                width: 32, height: 32, borderRadius: 8,
                background: isOutbound ? 'rgba(99,102,241,0.12)' : 'rgba(16,185,129,0.12)',
                color: isOutbound ? C.accent : '#10B981',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                fontWeight: 800, fontSize: 16,
              }}>
                {isOutbound ? '➔' : '⬅'}
              </div>
              <div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  <span style={{ fontSize: 10, fontWeight: 700, letterSpacing: '0.05em', textTransform: 'uppercase', color: C.textMuted }}>
                    {isOutbound ? 'Outgoing' : 'Incoming'}
                  </span>
                </div>
                <Badge
                  label={`${predicateMeta.icon} ${relationshipLabel}`}
                  color={predicateMeta.color}
                />
              </div>
            </div>

            {/* Middle: Connected Target Entity Details */}
            <div style={{ flex: 1, minWidth: 0, paddingLeft: 12, borderLeft: `1px solid ${C.border}` }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 3 }}>
                <span style={{ fontSize: 14 }}>{typeIcon}</span>
                <span style={{ fontWeight: 700, fontSize: 14, color: C.text, wordBreak: 'break-word' }}>
                  {cleanName}
                </span>
                <Badge label={typeLabel} color={typeColor} />
              </div>
              {relatedPath && (
                <div style={{
                  fontSize: 11, color: C.textMuted, fontFamily: 'monospace',
                  background: 'rgba(255,255,255,0.03)', padding: '2px 6px',
                  borderRadius: 4, display: 'inline-block', maxWidth: '100%',
                  overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                }}>
                  {relatedPath}
                </div>
              )}
            </div>

            {/* Right: Actions */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <Tooltip title={edgeHasProps ? 'View/Edit Properties' : 'No properties'}>
                <span>
                  <IconButton
                    size="small"
                    onClick={() => setEditingEdge(edge)}
                    sx={{ color: edgeHasProps ? C.warning : C.textMuted }}
                  >
                    <PropertiesIcon fontSize="small" />
                  </IconButton>
                </span>
              </Tooltip>
              <Tooltip title="Edit Relationship">
                <span>
                  <IconButton
                    size="small"
                    onClick={() => setEditingEdge(edge)}
                    sx={{ color: C.textMuted, '&:hover': { color: C.accent } }}
                  >
                    <EditIcon fontSize="small" />
                  </IconButton>
                </span>
              </Tooltip>
              <Tooltip title="Delete Relationship">
                <span>
                  <IconButton
                    size="small"
                    onClick={() => handleDelete(edge.id)}
                    disabled={deletingId === edge.id}
                    sx={{ color: C.danger, '&:hover': { color: '#FF6B6B' } }}
                  >
                    {deletingId === edge.id ? <CircularProgress size={16} /> : <DeleteIcon fontSize="small" />}
                  </IconButton>
                </span>
              </Tooltip>
            </div>
          </div>
        );
      })}

      {editingEdge && (
        <EditEdgeDialog
          open={true}
          onClose={() => setEditingEdge(null)}
          edge={editingEdge}
          onEdgeUpdated={handleEdgeUpdated}
        />
      )}
    </div>
  );
};

export default RelationshipList;
