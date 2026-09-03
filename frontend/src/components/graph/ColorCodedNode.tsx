import React from 'react';
import { Handle, Position, NodeProps } from 'reactflow';
import { CatalogNodeProps } from './CatalogNodeProps';

// Dynamic color mapping based on node type, ported from
// features/impact-analysis/components/ImpactGraph.tsx's getNodeTypeColor.
function getNodeTypeColor(nodeType: string): { bg: string; border: string; text: string } {
  const normalizedType = (nodeType || '').toLowerCase().replace(/\s+/g, '_');

  const colorMap: Record<string, { bg: string; border: string; text: string }> = {
    business_object: { bg: '#DBEAFE', border: '#1E40AF', text: '#001F3F' },
    business_term: { bg: '#DBEAFE', border: '#1E40AF', text: '#001F3F' },
    semantic_term: { bg: '#E9D5FF', border: '#6B21A8', text: '#2D0052' },
    semantic_model: { bg: '#E9D5FF', border: '#6B21A8', text: '#2D0052' },
    semantic_view: { bg: '#E9D5FF', border: '#6B21A8', text: '#2D0052' },
    semantic_column: { bg: '#FED7AA', border: '#92400E', text: '#3F2305' },
    database_column: { bg: '#DCFCE7', border: '#15803D', text: '#052E16' },
    db_column: { bg: '#DCFCE7', border: '#15803D', text: '#052E16' },
    column: { bg: '#DCFCE7', border: '#15803D', text: '#052E16' },
    table: { bg: '#F3E8FF', border: '#7E22CE', text: '#3F0F5C' },
    schema: { bg: '#FCE7F3', border: '#BE185D', text: '#500724' },
    database: { bg: '#FEE2E2', border: '#DC2626', text: '#4C0519' },
    bo_field: { bg: '#DBEAFE', border: '#0284C7', text: '#001F3F' },
    api_endpoint: { bg: '#FED7AA', border: '#D97706', text: '#3F2305' },
    bi_artifact: { bg: '#FEE2E2', border: '#DC2626', text: '#4C0519' },
    ai_artifact: { bg: '#FCE7F3', border: '#EC4899', text: '#500724' },
    access_rule: { bg: '#E2E8F0', border: '#475569', text: '#0F172A' },
  };

  return colorMap[normalizedType] || { bg: '#F3F4F6', border: '#9CA3AF', text: '#374151' };
}

// Generic node whose look is entirely driven by its own type + highlight/root
// state rather than a per-type registered component — used by viewers (like
// the impact-analysis graph) whose node types are open-ended (any catalog
// type name), so a fixed per-type nodeRegistry entry per type isn't feasible.
// Register this once under a synthetic type key (e.g. 'colorCoded') and tag
// every incoming node with that key; the real catalog type still drives the
// color via properties.realType.
export const ColorCodedNode: React.FC<NodeProps<CatalogNodeProps>> = ({ id, data }) => {
  const realType = data.properties?.realType || data.type;
  const colors = getNodeTypeColor(realType);
  const isRoot = !!data.properties?.isRoot;
  const isHighlighted = !!data.properties?.highlighted;

  return (
    <div
      style={{
        border: isHighlighted ? `3px solid ${colors.border}` : isRoot ? `2px solid ${colors.border}` : `1px solid ${colors.border}`,
        background: isRoot ? colors.bg : '#fff',
        padding: 12,
        borderRadius: 8,
        fontSize: 12,
        fontWeight: isRoot ? 'bold' : 'normal',
        width: 200,
        textAlign: 'center',
        boxShadow: isHighlighted ? `0 0 15px ${colors.border}` : isRoot ? `0 0 10px ${colors.border}44` : 'none',
        transform: isHighlighted ? 'scale(1.05)' : 'scale(1)',
        transition: 'all 0.3s ease',
        color: colors.text,
        position: 'relative',
      }}
    >
      <Handle type="target" position={Position.Left} style={{ background: colors.border }} />

      <div style={{ fontWeight: 'bold' }}>{data.label}</div>
      <div style={{ fontSize: '0.7rem', marginTop: 6 }}>
        <span
          style={{
            display: 'inline-block',
            padding: '1px 5px',
            background: colors.bg,
            borderRadius: 4,
            border: `1px solid ${colors.border}`,
            color: colors.text,
            fontWeight: 'bold',
          }}
        >
          {realType.toUpperCase()}
        </span>
      </div>

      <Handle type="source" position={Position.Right} style={{ background: colors.border }} />
    </div>
  );
};

export { getNodeTypeColor };
