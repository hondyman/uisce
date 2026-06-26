/* eslint-disable @typescript-eslint/no-unused-vars */
import { useState, useEffect, useMemo, FC } from 'react';
import { devLog, devError } from '../../../utils/devLogger';
import { SimpleTreeView as TreeView, TreeItem, treeItemClasses } from '@mui/x-tree-view';
import { Node as FlowNode } from 'reactflow';
import { 
  Checkbox, 
  Box, 
  Typography, 
  Tooltip, 
  alpha,
  styled,
  IconButton,
  Chip,
  useTheme,
  Menu,
  MenuItem
} from '@mui/material';
import renderCoreCustomChips from '../../../components/common/semanticChips';
import MenuIcon from '@mui/icons-material/Menu';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import ViewInArIcon from '@mui/icons-material/ViewInAr';
import StarIcon from '@mui/icons-material/Star';
import ClearIcon from '@mui/icons-material/Clear';
// ContentCopyIcon replaced by a Chip for gold copy display
import { EnhancedSelectedAsset } from '../../../types/SemanticTypes';

// Local runtime narrowers to avoid unsafe `any` casts on node.data
const asRecord = (v: unknown): Record<string, unknown> => (v && typeof v === 'object' ? (v as Record<string, unknown>) : {});
const getColumnsFromData = (data: unknown): unknown[] => {
  const cols = asRecord(data).columns;
  return Array.isArray(cols) ? (cols as unknown[]) : [];
};
const safeCloneForDebug = (v: unknown): unknown => {
  try {
    return JSON.parse(JSON.stringify(v));
  } catch (e) {
    return { ...asRecord(v) };
  }
};

export interface DataCatalogTreeProps {
  nodes: FlowNode[];
  onAssetSelect: (asset: EnhancedSelectedAsset) => void; // Keep this for single-select mode
  onColumnCountClick?: (node: FlowNode) => void;
  searchTerm: string;
  highlightedItem: string | null;
  showColumns?: boolean;
  multiselect?: boolean;
  selection?: Set<string>;
  onSelectionChange?: (selection: Set<string>) => void;
  modelMetadata?: Record<string, unknown>;
  onModelIconClick?: (tableName: string) => void;
  onGenerateModelForTable?: (tableName: string) => void;
  showGoldCopyIcon?: boolean;
  hideAssignmentControls?: boolean;
  onTotalColumnsClick?: (columns: unknown[], label?: string) => void;
}

const StyledTreeItem = styled(TreeItem)(({ theme }) => ({
  [`& .${treeItemClasses.content}`]: {
    padding: theme.spacing(0.5, 1),
    borderRadius: theme.shape.borderRadius,
    '&:hover': {
      backgroundColor: alpha(theme.palette.action.active, theme.palette.action.hoverOpacity),
    },
    [`&.${treeItemClasses.focused}`]: {
      backgroundColor: alpha(theme.palette.primary.main, theme.palette.action.focusOpacity),
      color: theme.palette.primary.main,
    },
    [`&.${treeItemClasses.selected}`]: {
      backgroundColor: alpha(theme.palette.primary.main, theme.palette.action.selectedOpacity),
      color: theme.palette.primary.main,
      '&:hover': {
        backgroundColor: alpha(theme.palette.primary.main, theme.palette.action.selectedOpacity + theme.palette.action.hoverOpacity),
      },
      [`&.${treeItemClasses.focused}`]: {
        backgroundColor: alpha(theme.palette.primary.main, theme.palette.action.selectedOpacity + theme.palette.action.focusOpacity),
      },
    },
  },
}));

const DataCatalogTree: FC<DataCatalogTreeProps> = ({
  nodes,
  onAssetSelect,
  onColumnCountClick,
  searchTerm,
  highlightedItem: _highlightedItem,
  showColumns: _showColumns = true,
  multiselect = false,
  selection = new Set(),
  onSelectionChange,
  modelMetadata = {},
  onModelIconClick,
  onGenerateModelForTable,
  showGoldCopyIcon = false,
  hideAssignmentControls = false,
  onTotalColumnsClick,
}) => {
  const theme = useTheme();
  const [modelFilter, setModelFilter] = useState<'all' | 'assigned' | 'unassigned'>('all');
  const [expandedItems, setExpandedItems] = useState<string[]>([]);
  useEffect(() => {
    if (hideAssignmentControls) {
      setModelFilter('all');
    }
  }, [hideAssignmentControls]);
  const handleSelectionToggle = (nodeId: string) => {
    if (!onSelectionChange) return;
    const newSelection = new Set(selection);
    if (newSelection.has(nodeId)) {
      newSelection.delete(nodeId);
    } else {
      if (!multiselect) {
        newSelection.clear();
      }
      newSelection.add(nodeId);
    }
    onSelectionChange(newSelection);
  };

  const filteredAndGroupedNodes = useMemo(() => {
    try {
      const schemaGroups: Record<string, FlowNode[]> = {};

      // Group nodes by schema
      nodes.forEach((node) => {
        const schemaName = node.data?.schemaName || 'default';
        if (!schemaGroups[schemaName]) {
          schemaGroups[schemaName] = [];
        }
        schemaGroups[schemaName].push(node);
      });

      // Filter based on model assignment status
      let filteredSchemas = schemaGroups;
      if (modelFilter !== 'all') {
        const newFilteredSchemas: Record<string, FlowNode[]> = {};
        
        Object.entries(schemaGroups).forEach(([schema, tables]) => {
          const filteredTables = tables.filter(table => {
            const tableName = table.data.tableName || table.data.qualifiedPath;
            
            // Check if table has a model
            const lookupKeys: string[] = [];
            if (tableName) {
              lookupKeys.push(tableName);
              if (typeof tableName === 'string' && tableName.includes('.')) {
                const slashForm = tableName.replace('.', '/');
                lookupKeys.push(slashForm);
                lookupKeys.push('/' + slashForm);
              }
            }
            
            let hasModel = false;
            try {
              for (const k of lookupKeys) {
                if (k && modelMetadata && Object.prototype.hasOwnProperty.call(modelMetadata, k)) {
                  const mm = modelMetadata[k] as Record<string, unknown> | undefined;
                  hasModel = Boolean(mm && (mm['exists'] === true || Boolean(mm['exists'])));
                  break;
                }
              }
            } catch (err) {
              // If there's an error checking model metadata, assume no model exists
              hasModel = false;
            }
            
            return modelFilter === 'assigned' ? hasModel : !hasModel;
          });
          
          if (filteredTables.length > 0) {
            newFilteredSchemas[schema] = filteredTables;
          }
        });
        
        filteredSchemas = newFilteredSchemas;
      }

      if (!searchTerm) {
        return filteredSchemas;
      }

      // Filter based on search term
      const lowercasedFilter = searchTerm.toLowerCase();
      const filteredGroups: Record<string, FlowNode[]> = {};

      Object.entries(filteredSchemas).forEach(([schema, tables]) => {
        if (schema.toLowerCase().includes(lowercasedFilter)) {
          filteredGroups[schema] = tables;
          return;
        }

        const filteredTables = tables.filter(table =>
          table.data?.label?.toLowerCase().includes(lowercasedFilter)
        );

        if (filteredTables.length > 0) {
          filteredGroups[schema] = filteredTables;
        }
      });

      return filteredGroups;
    } catch (err) {
      // If there's an error in filtering, return empty result
      try { devError('Error in filteredAndGroupedNodes:', err); } catch {}
      return {};
    }
  }, [nodes, searchTerm, modelFilter, modelMetadata]);

  // Calculate model assignment statistics
  const modelStats = useMemo(() => {
    try {
      let assigned = 0;
      let unassigned = 0;
      
      nodes.forEach((node) => {
        const tableName = node.data.tableName || node.data.qualifiedPath;
        
        const lookupKeys: string[] = [];
        if (tableName) {
          lookupKeys.push(tableName);
          if (typeof tableName === 'string' && tableName.includes('.')) {
            const slashForm = tableName.replace('.', '/');
            lookupKeys.push(slashForm);
            lookupKeys.push('/' + slashForm);
          }
        }
        
            let hasModel = false;
            try {
              for (const k of lookupKeys) {
                if (k && modelMetadata && Object.prototype.hasOwnProperty.call(modelMetadata, k)) {
                  const mm = modelMetadata[k] as Record<string, unknown> | undefined;
                  hasModel = Boolean(mm && (mm['exists'] === true || Boolean(mm['exists'])));
                  break;
                }
              }
            } catch (err) {
              // If there's an error checking model metadata, assume no model exists
              hasModel = false;
            }
        
        if (hasModel) {
          assigned++;
        } else {
          unassigned++;
        }
      });
      
  const totalColumns = nodes.reduce((acc, n) => acc + getColumnsFromData(n.data).length, 0);
      return { assigned, unassigned, total: assigned + unassigned, totalColumns };
    } catch (err) {
      // If there's an error calculating stats, return zeros
      try { devError('Error calculating model stats:', err); } catch {}
      return { assigned: 0, unassigned: 0, total: 0, totalColumns: 0 };
    }
  }, [nodes, modelMetadata]);

  // Create a stable key for filtered nodes to prevent unnecessary re-renders
  const filteredNodesKey = useMemo(() => Object.keys(filteredAndGroupedNodes).sort().join(','), [filteredAndGroupedNodes]);

  // Update expanded items when filtered nodes change
  useEffect(() => {
    setExpandedItems(Object.keys(filteredAndGroupedNodes));
  }, [filteredNodesKey]);

  const [menuAnchor, setMenuAnchor] = useState<HTMLElement | null>(null);
  const [menuForNodeId, setMenuForNodeId] = useState<string | null>(null);
  const openMenuForNode = (el: HTMLElement | null, nodeId?: string | null) => { setMenuAnchor(el); setMenuForNodeId(nodeId || null); };

  const renderTableNode = (node: FlowNode) => {
    const data = asRecord(node.data);
      const nodeId = `table-${node.id}`;
    const tableName = (data.tableName as string) || (data.qualifiedPath as string);

      // Try several key shapes when looking up model metadata because different
      // endpoints sometimes return keys as `public.users`, `public/users` or `/public/users`.
      const lookupKeys: string[] = [];
      if (tableName) {
        lookupKeys.push(tableName);
        if (typeof tableName === 'string' && tableName.includes('.')) {
          const slashForm = tableName.replace('.', '/');
          lookupKeys.push(slashForm);
          lookupKeys.push('/' + slashForm);
        }
      }
      // Also consider a qualifiedPath if present
      if (data.qualifiedPath && !lookupKeys.includes(String(data.qualifiedPath))) {
        lookupKeys.push(String(data.qualifiedPath));
      }

      let metadata: Record<string, unknown> | undefined;
      for (const k of lookupKeys) {
        if (k && modelMetadata && Object.prototype.hasOwnProperty.call(modelMetadata, k)) {
          metadata = modelMetadata[k] as Record<string, unknown> | undefined;
          break;
        }
      }

  const modelInfo = (data.modelInfo as Record<string, unknown> | undefined) || undefined;
  const modelExists = Boolean((metadata && (metadata['exists'] === true || Boolean(metadata['exists']))) || (modelInfo && (modelInfo['exists'] === true || Boolean(modelInfo['exists']))));

      // Robust core detection: check multiple possible fields and shapes returned
      // by various services: isCore, is_core, core_id, core object, tags/labels, metadata flags.
  // Normalize detection values to strict booleans so logs and UI are stable
  const nodeHasCoreId = Boolean(typeof data.core_id === 'string' && String(data.core_id).length > 0);
  const nodeHasCoreObj = Boolean(data.core && typeof data.core === 'object');
  const nodeIsCoreFlag = Boolean(data.isCore === true || data.is_core === true);
  const nodeTagCore = Boolean(Array.isArray(data.tags) && (data.tags as unknown[]).includes('core'));

  const metaHasCoreId = Boolean(typeof (metadata && metadata['core_id']) === 'string' && String((metadata && metadata['core_id']) || '').length > 0);
  const metaHasCoreObj = Boolean(metadata && metadata['core'] && typeof (metadata['core']) === 'object');
  const metaIsCoreFlag = Boolean((metadata && metadata['isCore'] === true) || (metadata && metadata['is_core'] === true) || (metadata && metadata['core'] === true));

  const isCore = Boolean(nodeIsCoreFlag || nodeHasCoreId || nodeHasCoreObj || nodeTagCore || metaIsCoreFlag || metaHasCoreId || metaHasCoreObj);

      // --- DEBUGGING LOG ---
      // This will log the data for each table node to help validate if the 'isCore'
      // or 'core_id' properties are being received correctly from the backend.
      // You can check your browser's developer console to see these logs.
      if (data.tableName) {
        // Stringify/copy rawData to freeze its shape in the console output and
        // ensure `isCore` prints as a stable primitive rather than appearing
        // inconsistent due to live object inspection in devtools.
        const frozen = safeCloneForDebug(data);
        devLog(`[DataCatalogTree] Node: ${String(data.tableName)}`, { isCore, rawData: frozen });
      }
      // --- END DEBUGGING ---

  const handleItemClick = () => {
        if (!multiselect) {
          const asset: EnhancedSelectedAsset = {
            type: 'table',
            id: nodeId,
            nodeId: node.id,
            name: String(data.label || 'Unknown'),
            tableName: tableName,
            node: node,
            isCore: isCore, // Pass the calculated isCore status
          };
          onAssetSelect(asset);
          
          // Also load the model in editor if it exists
          if (modelExists && onModelIconClick && tableName) {
            onModelIconClick(tableName);
          }
        }
      };

  // reuse the lifted menu state via menuAnchor/menuForNodeId
  return (
        <StyledTreeItem
          key={node.id != null && typeof node.id !== 'object' ? String(node.id) : ''}
          itemId={nodeId}
          onClick={handleItemClick}
          label={
            <Box sx={{ display: 'flex', alignItems: 'center', py: 0.5, gap: 1 }}>
      {multiselect && (
                <Checkbox
                  checked={selection.has(nodeId)}
                  onChange={() => handleSelectionToggle(nodeId)}
                  onClick={(e) => e.stopPropagation()}
                  size="small"
                />
              )}
            <Typography variant="body1" sx={{ flexGrow: 1 }}>
                {String(data.label || '') || 'Unknown'}
              </Typography>
              {isCore && (
                <Tooltip title={showGoldCopyIcon ? "Core model (read-only)" : "Core models are read-only"} placement="right">
                  {showGoldCopyIcon
                    ? (
                        <Tooltip title="core — read-only">
                          <Box component="span">{renderCoreCustomChips({ is_core: true })}</Box>
                        </Tooltip>
                      )
                    : <StarIcon sx={{ fontSize: 16, color: theme.palette.text.secondary }} />
                  }
                </Tooltip>
              )}
              {modelExists && (
                <Tooltip title={`Load model: ${metadata ? String(metadata['title'] ?? '') : (modelInfo ? String(modelInfo['title'] ?? '') : tableName)}`} placement="right">
                  <IconButton
                    size="small"
                    onClick={(e) => {
                      e.stopPropagation(); // Prevent row selection/deselection
                      if (onModelIconClick && tableName) {
                        onModelIconClick(tableName);
                      }
                    }}
                    sx={{ p: 0.2, '&:hover': { background: 'rgba(0,0,0,0.08)' } }}
                  >
                    <ViewInArIcon sx={{ fontSize: 16, color: theme.palette.primary.main }} />
                  </IconButton>
                </Tooltip>
              )}
              {/* Column count badge - clickable to open details */}
              <Box sx={{ ml: 1 }}>
                <Chip
                  label={`${getColumnsFromData(data).length} cols`}
                  size="small"
                  onClick={(e) => { e.stopPropagation(); if (onColumnCountClick) onColumnCountClick(node); }}
                  sx={{ cursor: 'pointer' }}
                />
              </Box>
              {/* Actions hamburger/menu: show when handler exists (consistent visibility) */}
              {onGenerateModelForTable && (
                <>
                  <Tooltip title="Actions">
                    <IconButton
                      size="small"
                      aria-label="Open actions"
                      onClick={(e) => {
                        e.stopPropagation();
                        openMenuForNode(e.currentTarget as HTMLElement, nodeId);
                      }}
                      sx={{ p: 0.2, '&:hover': { background: 'rgba(0,0,0,0.04)' } }}
                    >
                      <MenuIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                </>
              )}
            </Box>
          }
        />
      );
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Model Filter Statistics and Controls */}
      {!hideAssignmentControls && (
        <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 1 }}>
            <Typography variant="body2" color="text.secondary">
              Model Status:
            </Typography>
            <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
              <Chip
                label={`Assigned: ${modelStats.assigned}`}
                size="small"
                variant={modelFilter === 'assigned' ? 'filled' : 'outlined'}
                color={modelFilter === 'assigned' ? 'success' : 'default'}
                onClick={() => setModelFilter(modelFilter === 'assigned' ? 'all' : 'assigned')}
                sx={{ 
                  cursor: 'pointer',
                  ...(modelFilter !== 'assigned' && {
                    '& .MuiChip-label': { color: 'success.main' },
                    borderColor: 'success.main'
                  })
                }}
              />
              <Chip
                label={`Columns: ${modelStats.totalColumns}`}
                size="small"
                onClick={() => {
                  if (onTotalColumnsClick) {
                    // Aggregate all columns from nodes safely (intermediate type unknown[])
                    const cols = nodes.reduce((acc: unknown[], n) => acc.concat(getColumnsFromData(n.data)), [] as unknown[]);
                    // Cast once at the API boundary
                    const colsAny = cols as any[];
                    onTotalColumnsClick(colsAny, 'All Tables');
                  }
                }}
                sx={{ cursor: onTotalColumnsClick ? 'pointer' : 'default' }}
              />
              <Chip
                label={`Unassigned: ${modelStats.unassigned}`}
                size="small"
                variant={modelFilter === 'unassigned' ? 'filled' : 'outlined'}
                color={modelFilter === 'unassigned' ? 'warning' : 'default'}
                onClick={() => setModelFilter(modelFilter === 'unassigned' ? 'all' : 'unassigned')}
                sx={{ 
                  cursor: 'pointer',
                  ...(modelFilter !== 'unassigned' && {
                    '& .MuiChip-label': { color: 'warning.main' },
                    borderColor: 'warning.main'
                  })
                }}
              />
              {modelFilter !== 'all' && (
                <Tooltip title="Clear filter">
                  <IconButton
                    size="small"
                    onClick={() => setModelFilter('all')}
                    sx={{ 
                      color: 'text.secondary',
                      '&:hover': { 
                        color: 'text.primary',
                        backgroundColor: 'action.hover'
                      }
                    }}
                  >
                    <ClearIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
              )}
            </Box>
          </Box>
          <Typography variant="caption" color="text.secondary">
            Total tables: {modelStats.total} • Click chips to filter by model assignment status
          </Typography>
        </Box>
      )}

      <TreeView
        expandedItems={expandedItems}
        onExpandedItemsChange={(_event, itemIds) => setExpandedItems(itemIds)}
        slots={{
          collapseIcon: ExpandMoreIcon,
          expandIcon: ChevronRightIcon,
        }}
        sx={{ flexGrow: 1, overflowY: 'auto' }}
      >
        {Object.entries(filteredAndGroupedNodes).map(([schema, tables]) => {
          const tableIdsInSchema = tables.map(t => `table-${t.id}`);
          const selectedInSchemaCount = tableIdsInSchema.filter(id => selection.has(id)).length;
          const allSelected = tableIdsInSchema.length > 0 && selectedInSchemaCount === tableIdsInSchema.length;
          const indeterminate = selectedInSchemaCount > 0 && !allSelected;

          const handleSchemaToggle = () => {
            if (!onSelectionChange) return;
            const newSelection = new Set(selection);
            if (allSelected) {
              // If all are selected, deselect them
              tableIdsInSchema.forEach(id => newSelection.delete(id));
            } else {
              // Otherwise, select all
              tableIdsInSchema.forEach(id => newSelection.add(id));
            }
            onSelectionChange(newSelection);
          };

          return (
            <TreeItem
              key={schema != null && typeof schema !== 'object' ? String(schema) : ''}
              itemId={schema}
              label={
                <Box sx={{ display: 'flex', alignItems: 'center', p: 0.5, gap: 0.5 }}>
                  {multiselect && (
                    <Checkbox
                      indeterminate={indeterminate}
                      checked={allSelected}
                      onChange={handleSchemaToggle}
                      onClick={(e) => e.stopPropagation()}
                    />
                  )}
                  <Typography variant="subtitle2" sx={{ fontWeight: 'bold' }}>{schema}</Typography>
                </Box>
              }
            >
              {tables.map(renderTableNode)}
            </TreeItem>
          );
        })}
      </TreeView>
        {/* Shared actions menu for nodes - rendered once using lifted state */}
        <Menu
          anchorEl={menuAnchor}
          open={Boolean(menuAnchor)}
          onClose={() => openMenuForNode(null)}
          anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
          transformOrigin={{ vertical: 'top', horizontal: 'right' }}
          onClick={(e) => e.stopPropagation()}
        >
          <MenuItem onClick={() => { openMenuForNode(null); if (onGenerateModelForTable && menuForNodeId) {
              // derive tableName from node id if possible
              const tableId = menuForNodeId.replace(/^table-/, '');
              onGenerateModelForTable(tableId as unknown as string);
            } }}>
            Generate model
          </MenuItem>
        </Menu>
    </Box>
  );
};

export default DataCatalogTree;