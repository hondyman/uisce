import { Box, Stack, Typography, IconButton } from '@mui/material';
import {
  Business as BusinessObjectIcon,
  Layers as SubtypeIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
} from '@mui/icons-material';
import type { HierarchyNode } from '../../../../types/entity-schema';

export interface HierarchyTreeProps {
  nodes: HierarchyNode[];
  expandedNodes: Set<string>;
  onNodeToggle: (nodeId: string) => void;
  selectedNode: any;
  onNodeSelect: (node: any) => void;
  onRenameSubtype: (key: string, name: string) => void;
  onDeleteSubtype: (key: string) => void;
}

export function HierarchyTree({
  nodes,
  expandedNodes,
  onNodeToggle,
  selectedNode,
  onNodeSelect,
  onRenameSubtype,
  onDeleteSubtype,
}: HierarchyTreeProps) {
  return (
    <Box component="ul" sx={{ listStyle: 'none', p: 0, m: 0 }}>
      {nodes.map((node) => (
        <HierarchyTreeNode
          key={node.id}
          node={node}
          expandedNodes={expandedNodes}
          onNodeToggle={onNodeToggle}
          selectedNode={selectedNode}
          onNodeSelect={onNodeSelect}
          onRenameSubtype={onRenameSubtype}
          onDeleteSubtype={onDeleteSubtype}
        />
      ))}
    </Box>
  );
}

interface HierarchyTreeNodeProps {
  node: HierarchyNode;
  expandedNodes: Set<string>;
  onNodeToggle: (nodeId: string) => void;
  selectedNode: any;
  onNodeSelect: (node: any) => void;
  onRenameSubtype: (key: string, name: string) => void;
  onDeleteSubtype: (key: string) => void;
}

function HierarchyTreeNode({
  node,
  expandedNodes,
  onNodeToggle,
  selectedNode,
  onNodeSelect,
  onRenameSubtype,
  onDeleteSubtype,
}: HierarchyTreeNodeProps) {
  const hasChildren = node.children && node.children.length > 0;
  const isSubtypeNode = (node as any).isSubtype;
  const subtypeKey = (node as any).subtypeKey;
  const technicalName = (node as any).technicalName;
  const isRootNode = node.id === 'root';
  const isSelected = (isRootNode && !selectedNode) || (selectedNode?.type === 'subtype' && selectedNode.subtypeKey === subtypeKey && isSubtypeNode);

  const handleNodeClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (isRootNode) {
      onNodeSelect(null);
    } else if (isSubtypeNode) {
      onNodeSelect({ type: 'subtype', subtypeKey, key: node.id });
    }
  };

  return (
    <Box component="li" sx={{ listStyle: 'none', mb: 0.5 }}>
      <Stack
        direction="row"
        spacing={1}
        alignItems="center"
        onClick={handleNodeClick}
        sx={{
          p: 1,
          borderRadius: 1,
          cursor: 'pointer',
          bgcolor: isSelected ? 'primary.light' : 'transparent',
          color: isSelected ? 'primary.main' : 'text.primary',
          fontWeight: isSelected ? 700 : 400,
          transition: 'all 0.2s ease',
          '&:hover': {
            bgcolor: isSelected ? 'primary.light' : 'action.hover',
          },
        }}
      >
        <Box sx={{ width: 24, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          {isRootNode ? (
            <BusinessObjectIcon sx={{ fontSize: '1.25rem', color: 'primary.main' }} />
          ) : isSubtypeNode ? (
            <SubtypeIcon sx={{ fontSize: '1.25rem', color: 'info.main' }} />
          ) : null}
        </Box>
        <Stack direction="column" spacing={0} flex={1}>
          <Typography variant="body2">{node.displayName}</Typography>
          {technicalName && isSubtypeNode && (
            <Typography 
              variant="caption" 
              color="text.secondary" 
              sx={{ fontFamily: 'monospace', fontSize: '0.7rem' }}
              title={`Technical name: ${technicalName}`}
            >
              {technicalName}
            </Typography>
          )}
        </Stack>

        {isSubtypeNode && (
          <Stack direction="row" spacing={0.5} sx={{ ml: 'auto' }}>
            <IconButton
              size="small"
              color="primary"
              onClick={(e) => {
                e.stopPropagation();
                onRenameSubtype(subtypeKey || '', node.displayName || '');
              }}
              title="Edit subtype"
              sx={{
                '&:hover': { bgcolor: 'primary.light', color: 'primary.dark' },
              }}
            >
              <EditIcon fontSize="small" />
            </IconButton>
            <IconButton
              size="small"
              color="error"
              onClick={(e) => {
                e.stopPropagation();
                onDeleteSubtype(subtypeKey);
              }}
              title="Delete subtype"
              sx={{
                '&:hover': { bgcolor: 'error.light', color: 'error.dark' },
              }}
            >
              <DeleteIcon fontSize="small" />
            </IconButton>
          </Stack>
        )}
      </Stack>

      {hasChildren && (
        <Box component="ul" sx={{ listStyle: 'none', p: 0, m: 0, pl: 2, borderLeft: '2px solid', borderLeftColor: 'divider', ml: 2 }}>
          {node.children?.map((child) => (
            <HierarchyTreeNode
              key={child.id}
              node={child}
              expandedNodes={expandedNodes}
              onNodeToggle={onNodeToggle}
              selectedNode={selectedNode}
              onNodeSelect={onNodeSelect}
              onRenameSubtype={onRenameSubtype}
              onDeleteSubtype={onDeleteSubtype}
            />
          ))}
        </Box>
      )}
    </Box>
  );
}
