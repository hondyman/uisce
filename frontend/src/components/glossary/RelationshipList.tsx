import React, { useState } from 'react';
import { IconButton, Tooltip, CircularProgress } from '@mui/material';
import { Delete as DeleteIcon, Edit as EditIcon, Settings as PropertiesIcon } from '@mui/icons-material';
import { useDeleteTermEdge } from '../../api/glossary';
import EditEdgeDialog from '../EditEdgeDialog';

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
  relationship_type?: string;
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
  bg: '#0A0C12',
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

  if (!edges || edges.length === 0) {
    return (
      <div style={{
        padding: 24, textAlign: 'center', color: C.textMuted,
        fontFamily: 'system-ui, sans-serif', fontSize: 14,
      }}>
        No relationships found
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16, padding: 24 }}>
      {edges.map((edge) => {
        const sourceId = edge.subject_node_id || edge.source_node_id;
        const targetId = edge.object_node_id || edge.target_node_id;
        const source = sourceId ? resolveNode(sourceId) : undefined;
        const target = targetId ? resolveNode(targetId) : undefined;
        const isOutbound = sourceId === selectedNodeId;

        const sourceName = (sourceId ? getNodeName?.(sourceId) : undefined) || source?.node_name || source?.name || sourceId?.substring(0, 8);
        const targetName = (targetId ? getNodeName?.(targetId) : undefined) || target?.node_name || target?.name || targetId?.substring(0, 8);
        const sourcePath = (sourceId ? getNodePath?.(sourceId) : undefined) || source?.qualified_path;
        const targetPath = (targetId ? getNodePath?.(targetId) : undefined) || target?.qualified_path;
        const sourceType = source?.catalog_type_name || source?.node_type || 'Node';
        const targetType = target?.catalog_type_name || target?.node_type || 'Node';
        const edgeHasProps = hasProperties(edge);

        return (
          <div
            key={edge.id}
            style={{
              background: C.panel,
              border: `1px solid ${C.border}`,
              borderRadius: 8,
              overflow: 'hidden',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'stretch' }}>
              <div style={{ flex: 1, padding: 16 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                  <Badge label={sourceType} color={C.blue} />
                  <span style={{ fontWeight: 600, color: C.text }}>{sourceName}</span>
                </div>
                {sourcePath && (
                  <div style={{ fontSize: 12, color: C.textMuted, fontFamily: 'monospace' }}>{sourcePath}</div>
                )}
              </div>
              <div style={{
                display: 'flex', flexDirection: 'column', alignItems: 'center',
                justifyContent: 'center', padding: '0 24px',
                background: darkMode ? 'rgba(255,255,255,0.02)' : 'rgba(0,0,0,0.02)',
                borderLeft: `1px solid ${C.border}`, borderRight: `1px solid ${C.border}`,
              }}>
                <Badge label={edge.edge_type_name || edge.relationship_type || 'RELATION'} color={C.accent} />
                <div style={{ color: C.textMuted, marginTop: 4 }}>{isOutbound ? '→' : '←'}</div>
              </div>
              <div style={{ flex: 1, padding: 16, textAlign: 'right' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4, justifyContent: 'flex-end' }}>
                  <span style={{ fontWeight: 600, color: C.text }}>{targetName}</span>
                  <Badge label={targetType} color={C.teal} />
                </div>
                {targetPath && (
                  <div style={{ fontSize: 12, color: C.textMuted, fontFamily: 'monospace' }}>{targetPath}</div>
                )}
              </div>
            </div>
            <div style={{
              padding: '8px 16px',
              background: darkMode ? 'rgba(255,255,255,0.02)' : 'rgba(0,0,0,0.02)',
              borderTop: `1px solid ${C.border}`,
              display: 'flex',
              justifyContent: 'flex-end',
              gap: 4,
            }}>
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
                    sx={{ color: C.accent }}
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
                    sx={{ color: C.danger }}
                  >
                    {deletingId === edge.id ? (
                      <CircularProgress size={16} sx={{ color: C.danger }} />
                    ) : (
                      <DeleteIcon fontSize="small" />
                    )}
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
