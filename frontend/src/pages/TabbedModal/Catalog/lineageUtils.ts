// /Users/eganpj/GitHub/semlayer/frontend/src/pages/TabbedModal/Catalog/lineageUtils.ts

// Layout Configuration - Horizontal (Left-to-Right) Layout
export const LINEAGE_LAYOUT = {
  nodeWidth: 220,
  nodeHeight: 70,
  horizontalSpacing: 300, // Increased for left-to-right flow
  verticalSpacing: 100,   // Increased vertical spacing between nodes
};

// Horizontal layout: level determines X position (left-to-right), index determines Y position
export const createLineagePosition = (level: number, index: number, count: number) => {
  const x = level * (LINEAGE_LAYOUT.nodeWidth + LINEAGE_LAYOUT.horizontalSpacing);
  const y = (index - (count - 1) / 2) * (LINEAGE_LAYOUT.nodeHeight + LINEAGE_LAYOUT.verticalSpacing);
  return { x, y };
};

// Color scheme functions matching DataCatalogTree
// Enhanced color scheme with better differentiation
const getBusinessAssetColors = (nodeType: string) => {
  switch (nodeType) {
    case 'business_term':
      return {
        backgroundColor: '#dbeafe',  // Stronger blue
        borderColor: '#2563eb',
        textColor: '#1e3a8a',
        hoverColor: '#bfdbfe'
      };
    case 'semantic_term':
      return {
        backgroundColor: '#e9d5ff',  // Stronger purple
        borderColor: '#7c3aed',
        textColor: '#5b21b6',
        hoverColor: '#ddd6fe'
      };
    case 'semantic_model':
    case 'semantic_view':
      return {
        backgroundColor: '#d1fae5',  // Stronger green
        borderColor: '#059669',
        textColor: '#065f46',
        hoverColor: '#a7f3d0'
      };
    case 'semantic_column':
      return {
        backgroundColor: '#e0f2fe',  // Stronger cyan
        borderColor: '#0284c7',
        textColor: '#075985',
        hoverColor: '#bae6fd'
      };
    default:
      return {
        backgroundColor: '#f1f5f9',
        borderColor: '#64748b',
        textColor: '#334155',
        hoverColor: '#e2e8f0'
      };
  }
};

// Enhanced technical asset colors with better contrast
const getTechnicalAssetColors = (nodeType: string) => {
  switch (nodeType) {
    case 'database':
    case 'schema':
      return {
        backgroundColor: '#fef3c7',  // Amber for database/schema
        borderColor: '#d97706',
        textColor: '#78350f',
        hoverColor: '#fde68a'
      };
    case 'table':
      return {
        backgroundColor: '#dbeafe',  // Blue for tables
        borderColor: '#3b82f6',
        textColor: '#1e3a8a',
        hoverColor: '#bfdbfe'
      };
    case 'column':
    case 'database_column':
      return {
        backgroundColor: '#e0e7ff',  // Indigo for columns
        borderColor: '#6366f1',
        textColor: '#3730a3',
        hoverColor: '#c7d2fe'
      };
    default:
      return {
        backgroundColor: '#f3f4f6',
        borderColor: '#9ca3af',
        textColor: '#374151',
        hoverColor: '#e5e7eb'
      };
  }
};

export const getLineageNodeStyle = (nodeType: string | undefined, isCenter: boolean) => {
  if (isCenter) {
    // Center node gets special highlight with gradient
    return {
      background: 'linear-gradient(135deg, #fbbf24 0%, #f59e0b 100%)',
      borderColor: '#d97706',
      color: '#78350f',
      borderWidth: '3px',
      fontWeight: '700',
      boxShadow: '0 4px 12px rgba(245, 158, 11, 0.4)'
    };
  }

  // Determine if it's a business or technical asset
  const businessTypes = ['business_term', 'semantic_term', 'semantic_model', 'semantic_view', 'semantic_column'];
  const isBusiness = businessTypes.includes(nodeType || '');

  const colors = isBusiness ? getBusinessAssetColors(nodeType || '') : getTechnicalAssetColors(nodeType || '');

  return {
    borderColor: colors.borderColor,
    color: colors.textColor,
    borderWidth: '2px',
    boxShadow: '0 2px 4px rgba(0, 0, 0, 0.1)'
  };
};

/**
 * Formats an edge label or type string for display.
 * Handles known database keys like 'related_to' -> 'Related To'
 */
export const formatEdgeLabel = (edge: any): string => {
  if (!edge) return 'Unknown';

  // 1. Try to get the raw type string
  let relType = '';

  if (typeof edge === 'string') {
    relType = edge;
  } else {
    // Prioritize explicit edge type name from join
    relType = edge.edge_type_name ||
      edge.edge_type?.edge_type_name ||
      edge.relationship_type ||
      edge.label ||
      edge.type ||
      edge.data?.edge_type_name ||
      edge.data?.relationship_type ||
      'Related';
  }

  // 2. Specific Overrides
  if (relType === 'related_to') return 'Related To';
  if (relType === 'maps_to') return 'Maps To';
  if (relType === 'SemanticToDatabase') return 'Maps To'; // Legacy
  if (relType === 'smoothstep') return 'Related'; // ReactFlow default type leakage
  if (relType === 'foreign_key') return 'Foreign Key';
  if (relType === 'has_context') return 'Has Context';
  if (relType === 'member_of') return 'Member Of';

  // 3. Generic Formatting
  if (!relType) return 'Related';
  return relType.replace(/_/g, ' ').replace(/\b\w/g, (c: string) => c.toUpperCase());
};