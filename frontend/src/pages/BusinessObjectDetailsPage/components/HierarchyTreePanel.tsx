import { Paper, Box, Typography, TextField, Stack, InputAdornment, IconButton, Tooltip } from '@mui/material';
import {
  Search as SearchIcon,
  ChevronLeft as CollapseIcon,
  ChevronRight as ExpandIcon,
  AccountTree as TreeIcon,
} from '@mui/icons-material';
import { HierarchyTree } from './HierarchyTree/HierarchyTree';

interface HierarchyTreePanelProps {
  hierarchyNodes: any[];
  expandedNodes: Set<string> | any;
  selectedNode: any;
  businessObject?: {
    version?: string;
    updatedAt?: string;
  };
  width?: number;
  isCollapsed?: boolean;
  onToggleCollapse?: () => void;
  onNodeToggle: (nodeId: string) => void;
  onNodeSelect: (node: any) => void;
  onRenameSubtype: (subtypeId: string, subtypeKey: string) => void;
  onDeleteSubtype: (subtypeId: string) => void;
}

export function HierarchyTreePanel({
  hierarchyNodes,
  expandedNodes,
  selectedNode,
  businessObject,
  width = 280,
  isCollapsed = false,
  onToggleCollapse,
  onNodeToggle,
  onNodeSelect,
  onRenameSubtype,
  onDeleteSubtype,
}: HierarchyTreePanelProps) {
  const formatLastModified = (dateStr?: string) => {
    if (!dateStr) return 'Unknown';
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
    if (diffHours < 1) return 'Just now';
    if (diffHours < 24) return `${diffHours} hour${diffHours > 1 ? 's' : ''} ago`;
    const diffDays = Math.floor(diffHours / 24);
    return `${diffDays} day${diffDays > 1 ? 's' : ''} ago`;
  };

  if (isCollapsed) {
    return (
      <Paper
        elevation={0}
        sx={{
          width: 48,
          minWidth: 48,
          border: '1px solid',
          borderColor: 'divider',
          borderRadius: 1,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          py: 2,
          bgcolor: 'background.paper',
          transition: 'width 0.2s ease',
        }}
      >
        <Tooltip title="Expand Object Structure" placement="right">
          <IconButton size="small" onClick={onToggleCollapse} color="primary">
            <ExpandIcon />
          </IconButton>
        </Tooltip>
        <Tooltip title="Object Structure" placement="right">
          <Box sx={{ mt: 2, display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
            <TreeIcon color="action" fontSize="small" />
            <Typography
              variant="caption"
              sx={{
                writingMode: 'vertical-rl',
                textOrientation: 'mixed',
                transform: 'rotate(180deg)',
                mt: 2,
                fontWeight: 700,
                color: 'text.secondary',
                letterSpacing: 1,
                fontSize: '0.7rem',
              }}
            >
              STRUCTURE
            </Typography>
          </Box>
        </Tooltip>
      </Paper>
    );
  }

  return (
    <Paper
      elevation={0}
      sx={{
        width: { xs: '100%', lg: width },
        minWidth: { lg: 180 },
        maxWidth: { lg: 600 },
        flexShrink: 0,
        border: '1px solid',
        borderColor: 'divider',
        borderRadius: 1,
        overflow: 'hidden',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <Box sx={{ p: 1.5, borderBottom: '1px solid', borderBottomColor: 'divider' }}>
        <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
          <Stack direction="row" spacing={1} alignItems="center">
            <TreeIcon fontSize="small" color="primary" />
            <Typography variant="subtitle2" sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem', letterSpacing: 0.5 }}>
              Object Structure
            </Typography>
          </Stack>
          {onToggleCollapse && (
            <Tooltip title="Collapse sidebar">
              <IconButton size="small" onClick={onToggleCollapse} sx={{ p: 0.5 }}>
                <CollapseIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          )}
        </Stack>
        <TextField
          fullWidth
          placeholder="Filter hierarchy..."
          variant="outlined"
          size="small"
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon fontSize="small" />
              </InputAdornment>
            ),
          }}
          sx={{ '& .MuiInputBase-input': { fontSize: '0.8rem', py: 0.5 } }}
        />
      </Box>

      <Box sx={{ flex: 1, overflow: 'auto', p: 1, maxHeight: 'calc(100vh - 350px)' }}>
        <HierarchyTree
          nodes={hierarchyNodes}
          expandedNodes={expandedNodes}
          onNodeToggle={onNodeToggle}
          selectedNode={selectedNode}
          onNodeSelect={onNodeSelect}
          onRenameSubtype={onRenameSubtype}
          onDeleteSubtype={onDeleteSubtype}
        />
      </Box>

      <Box
        sx={{
          p: 1.5,
          borderTop: '1px solid',
          borderTopColor: 'divider',
          bgcolor: 'action.hover',
        }}
      >
        <Stack direction="row" justifyContent="space-between" alignItems="center">
          <Typography variant="caption" color="text.secondary" sx={{ fontSize: '0.7rem' }}>
            {formatLastModified(businessObject?.updatedAt)}
          </Typography>
          <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600, fontSize: '0.7rem' }}>
            {businessObject?.version || 'v1.0'}
          </Typography>
        </Stack>
      </Box>
    </Paper>
  );
}
