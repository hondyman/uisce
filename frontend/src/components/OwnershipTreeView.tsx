/**
 * ============================================================================
 * OwnershipTreeView - Interactive Hierarchical Visualization
 * ============================================================================
 * 
 * Displays entity ownership hierarchy as an interactive tree with:
 * - Expand/collapse nodes
 * - Color-coded by model type or ownership type
 * - Drill-down to entity details
 * - Search within tree
 * - Portfolio metrics on hover
 * 
 * Usage:
 * <OwnershipTreeView rootId={householdId} depth={3} colorBy="modelType" />
 */

import React, { useState, useMemo } from 'react';
import { ChevronDown, ChevronRight, Search, Info } from 'lucide-react';
import { useQuery, gql } from '@apollo/client';

// ============================================================================
// GraphQL Query
// ============================================================================

const OWNERSHIP_TREE_QUERY = gql`
  query OwnershipTree($rootId: UUID!, $depth: Int!, $includeAttributes: Boolean!, $asOf: Date) {
    ownershipTree(
      rootId: $rootId
      depth: $depth
      includeAttributes: $includeAttributes
      asOf: $asOf
    ) {
      entity {
        id
        modelType
        displayName
        originalName
        ownershipType
        status
        attributes {
          key
          value
        }
      }
      position {
        id
        ownershipPercentage
        shares
        value
      }
      depth
      childCount
      children {
        entity { id modelType displayName ownershipType }
        position { ownershipPercentage shares value }
        childCount
        children {
          entity { id modelType displayName ownershipType }
          position { ownershipPercentage }
          childCount
        }
      }
    }
  }
`;

// ============================================================================
// Type Definitions
// ============================================================================

interface Entity {
  id: string;
  modelType: string;
  displayName: string;
  originalName: string;
  ownershipType: 'PERCENT_BASED' | 'SHARE_BASED' | 'VALUE_BASED';
  status: string;
  attributes?: Array<{ key: string; value: any }>;
}

interface Position {
  id: string;
  ownershipPercentage?: number;
  shares?: number;
  value?: number;
}

interface OwnershipNode {
  entity: Entity;
  position?: Position;
  depth: number;
  childCount: number;
  children?: OwnershipNode[];
}

interface OwnershipTreeViewProps {
  rootId: string;
  depth?: number;
  colorBy?: 'modelType' | 'ownershipType' | 'status';
  onNodeClick?: (node: OwnershipNode) => void;
  asOf?: string; // ISO date string
}

// ============================================================================
// Color Schemes
// ============================================================================

const MODEL_TYPE_COLORS: Record<string, string> = {
  // Top-level
  household: '#DC2626', // Red
  
  // Containers
  person_node: '#059669', // Green
  prospect: '#10B981',
  trust: '#14B8A6',
  managed_partnership: '#06B6D4',
  holding_company: '#0EA5E9',
  manager: '#3B82F6',
  vehicle: '#6366F1',
  
  // Sub-containers
  financial_account: '#8B5CF6',
  sleeve: '#A78BFA',
  fund: '#D8B4FE',
  hedge_fund: '#E9D5FF',
  private_equity_fund: '#F3E8FF',
  
  // Assets
  stock: '#EC4899',
  bond: '#F43F5E',
  etf: '#FB7185',
  cash: '#FCA5A5',
  real_estate: '#FED7AA',
  private_investment: '#FECACA',
  venture_capital: '#FCE7F3',
  default: '#9CA3AF', // Gray
};

const OWNERSHIP_TYPE_COLORS: Record<string, string> = {
  PERCENT_BASED: '#3B82F6', // Blue
  SHARE_BASED: '#8B5CF6', // Purple
  VALUE_BASED: '#EC4899', // Pink
};

const STATUS_COLORS: Record<string, string> = {
  ACTIVE: '#10B981', // Green
  INACTIVE: '#F59E0B', // Amber
  CLOSED: '#EF4444', // Red
  PENDING: '#3B82F6', // Blue
};

// ============================================================================
// Helper Functions
// ============================================================================

function getNodeColor(node: OwnershipNode, colorBy: 'modelType' | 'ownershipType' | 'status'): string {
  switch (colorBy) {
    case 'ownershipType':
      return OWNERSHIP_TYPE_COLORS[node.entity.ownershipType] || OWNERSHIP_TYPE_COLORS.PERCENT_BASED;
    case 'status':
      return STATUS_COLORS[node.entity.status] || STATUS_COLORS.ACTIVE;
    case 'modelType':
    default:
      return MODEL_TYPE_COLORS[node.entity.modelType] || MODEL_TYPE_COLORS.default;
  }
}

function getNodeIcon(modelType: string): string {
  const assetEmoji: Record<string, string> = {
    household: '🏠',
    person_node: '👤',
    trust: '📋',
    financial_account: '🏦',
    stock: '📈',
    bond: '📊',
    etf: '📦',
    cash: '💰',
    real_estate: '🏢',
    private_investment: '🎯',
    venture_capital: '🚀',
    hedge_fund: '💼',
    private_equity_fund: '📈',
    managed_partnership: '🤝',
    holding_company: '🏢',
    digital_asset: '₿',
  };
  return assetEmoji[modelType] || '📌';
}

function formatOwnershipValue(position?: Position): string {
  if (!position) return '';
  if (position.ownershipPercentage !== undefined) {
    return `${position.ownershipPercentage.toFixed(2)}%`;
  } else if (position.shares !== undefined) {
    return `${position.shares.toLocaleString()} shares`;
  } else if (position.value !== undefined) {
    return `$${(position.value / 1000000).toFixed(2)}M`;
  }
  return '';
}

function searchTree(node: OwnershipNode, query: string): boolean {
  const queryLower = query.toLowerCase();
  if (
    node.entity.displayName.toLowerCase().includes(queryLower) ||
    node.entity.originalName.toLowerCase().includes(queryLower) ||
    node.entity.modelType.toLowerCase().includes(queryLower)
  ) {
    return true;
  }
  if (node.children) {
    return node.children.some(child => searchTree(child, query));
  }
  return false;
}

// ============================================================================
// TreeNode Component
// ============================================================================

interface TreeNodeProps {
  node: OwnershipNode;
  colorBy: 'modelType' | 'ownershipType' | 'status';
  onNodeClick?: (node: OwnershipNode) => void;
  searchQuery?: string;
  level?: number;
}

const TreeNode: React.FC<TreeNodeProps> = ({
  node,
  colorBy,
  onNodeClick,
  searchQuery,
  level = 0,
}) => {
  const [isExpanded, setIsExpanded] = useState(level < 2); // Auto-expand first 2 levels
  const [showInfo, setShowInfo] = useState(false);

  const hasChildren = (node.children?.length ?? 0) > 0;
  const matches = !searchQuery || searchTree(node, searchQuery);
  const color = getNodeColor(node, colorBy);
  const ownershipValue = formatOwnershipValue(node.position);

  if (!matches && hasChildren) {
    // Check if any children match
    const childMatches = node.children?.some(child => searchTree(child, searchQuery));
    if (!childMatches) return null;
  }

  return (
    <div className="select-none">
      <div
        className="flex items-center py-1 px-2 rounded hover:bg-gray-100 cursor-pointer group"
        onClick={() => {
          if (hasChildren) setIsExpanded(!isExpanded);
          onNodeClick?.(node);
        }}
      >
        {/* Expand/Collapse Toggle */}
        <div className="w-6 flex items-center justify-center">
          {hasChildren ? (
            isExpanded ? (
              <ChevronDown size={16} className="text-gray-500" />
            ) : (
              <ChevronRight size={16} className="text-gray-500" />
            )
          ) : (
            <div className="w-4" />
          )}
        </div>

        {/* Color Indicator */}
        <div
          className="w-3 h-3 rounded-full mr-2"
          style={{ backgroundColor: color }}
          title={`Type: ${node.entity.modelType}`}
        />

        {/* Icon & Display Name */}
        <span className="text-lg mr-1">{getNodeIcon(node.entity.modelType)}</span>
        <span className="font-medium text-sm flex-1 truncate">{node.entity.displayName}</span>

        {/* Ownership Value */}
        {node.position && (
          <span className="text-xs text-gray-600 mr-2 hidden group-hover:inline-block">
            {ownershipValue}
          </span>
        )}

        {/* Info Button */}
        <button
          className="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-blue-100 rounded"
          onClick={(e) => {
            e.stopPropagation();
            setShowInfo(!showInfo);
          }}
          title="Show details"
        >
          <Info size={14} className="text-blue-600" />
        </button>
      </div>

      {/* Info Tooltip */}
      {showInfo && (
        <div className="ml-8 mb-2 p-2 bg-blue-50 rounded border border-blue-200 text-xs">
          <div><strong>Type:</strong> {node.entity.modelType}</div>
          <div><strong>Ownership:</strong> {node.entity.ownershipType}</div>
          <div><strong>Status:</strong> {node.entity.status}</div>
          <div><strong>ID:</strong> {node.entity.id.substring(0, 8)}...</div>
          {node.position && (
            <div><strong>Position:</strong> {ownershipValue}</div>
          )}
          {node.entity.attributes && node.entity.attributes.length > 0 && (
            <div className="mt-1">
              <strong>Attributes:</strong>
              <ul className="ml-2">
                {node.entity.attributes.slice(0, 3).map(attr => (
                  <li key={attr.key}>
                    {attr.key}: {JSON.stringify(attr.value).substring(0, 20)}...
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}

      {/* Child Nodes (Recursive) */}
      {isExpanded && node.children && node.children.length > 0 && (
        <div className="ml-6 border-l border-gray-300">
          {node.children.map(child => (
            <TreeNode
              key={child.entity.id}
              node={child}
              colorBy={colorBy}
              onNodeClick={onNodeClick}
              searchQuery={searchQuery}
              level={level + 1}
            />
          ))}
        </div>
      )}

      {/* Empty state for collapsed nodes with children */}
      {!isExpanded && hasChildren && (
        <div className="ml-6 text-xs text-gray-500 italic">
          ({node.childCount} children)
        </div>
      )}
    </div>
  );
};

// ============================================================================
// Main Component
// ============================================================================

export const OwnershipTreeView: React.FC<OwnershipTreeViewProps> = ({
  rootId,
  depth = 3,
  colorBy = 'modelType',
  onNodeClick,
  asOf,
}) => {
  const [searchQuery, setSearchQuery] = useState('');

  const { data, loading, error } = useQuery(OWNERSHIP_TREE_QUERY, {
    variables: {
      rootId,
      depth,
      includeAttributes: true,
      asOf,
    },
    pollInterval: 0, // No auto-refresh by default
  });

  // Legend
  const legendItems = colorBy === 'modelType'
    ? [
        { label: 'Household', color: MODEL_TYPE_COLORS.household },
        { label: 'Containers', color: MODEL_TYPE_COLORS.person_node },
        { label: 'Accounts', color: MODEL_TYPE_COLORS.financial_account },
        { label: 'Assets', color: MODEL_TYPE_COLORS.stock },
      ]
    : colorBy === 'ownershipType'
    ? [
        { label: 'Percent-based', color: OWNERSHIP_TYPE_COLORS.PERCENT_BASED },
        { label: 'Share-based', color: OWNERSHIP_TYPE_COLORS.SHARE_BASED },
        { label: 'Value-based', color: OWNERSHIP_TYPE_COLORS.VALUE_BASED },
      ]
    : [
        { label: 'Active', color: STATUS_COLORS.ACTIVE },
        { label: 'Inactive', color: STATUS_COLORS.INACTIVE },
        { label: 'Closed', color: STATUS_COLORS.CLOSED },
        { label: 'Pending', color: STATUS_COLORS.PENDING },
      ];

  if (loading) {
    return (
      <div className="p-4 text-center text-gray-500">
        <div className="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full mx-auto mb-2" />
        Loading ownership tree...
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 bg-red-50 border border-red-200 rounded">
        <div className="text-red-800 font-semibold">Error loading tree</div>
        <div className="text-red-700 text-sm">{error.message}</div>
      </div>
    );
  }

  if (!data?.ownershipTree) {
    return (
      <div className="p-4 text-center text-gray-500">
        No data available
      </div>
    );
  }

  const rootNode = data.ownershipTree as OwnershipNode;

  return (
    <div className="w-full bg-white rounded-lg shadow-sm">
      {/* Header */}
      <div className="p-4 border-b border-gray-200">
        <h2 className="text-lg font-semibold mb-3">
          {rootNode.entity.displayName} {rootNode.entity.modelType && `(${rootNode.entity.modelType})`}
        </h2>

        {/* Search Bar */}
        <div className="relative mb-3">
          <Search size={16} className="absolute left-3 top-2.5 text-gray-400" />
          <input
            type="text"
            placeholder="Search entities by name, type, or ID..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-9 pr-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
          />
        </div>

        {/* Legend */}
        <div className="flex flex-wrap gap-3 text-xs">
          {legendItems.map((item) => (
            <div key={item.label} className="flex items-center gap-2">
              <div
                className="w-3 h-3 rounded-full"
                style={{ backgroundColor: item.color }}
              />
              <span className="text-gray-600">{item.label}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Tree */}
      <div className="p-4 font-mono text-xs max-h-96 overflow-y-auto">
        <TreeNode
          node={rootNode}
          colorBy={colorBy}
          onNodeClick={onNodeClick}
          searchQuery={searchQuery}
          level={0}
        />
      </div>

      {/* Footer Stats */}
      <div className="p-3 border-t border-gray-200 bg-gray-50 text-xs text-gray-600 flex justify-between">
        <span>Total children: {rootNode.childCount}</span>
        <span>Depth: {rootNode.depth}</span>
      </div>
    </div>
  );
};

export default OwnershipTreeView;
