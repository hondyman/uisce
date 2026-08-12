import { Paper, Box, Typography, TextField, Stack, InputAdornment } from '@mui/material';
import { Search as SearchIcon } from '@mui/icons-material';
import { HierarchyTree } from './HierarchyTree/HierarchyTree';

interface HierarchyTreePanelProps {
  hierarchyNodes: any[];
  expandedNodes: Record<string, boolean>;
  selectedNode: any;
  businessObject?: {
    version?: string;
    updatedAt?: string;
  };
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

  return (
    <Paper
      elevation={0}
      sx={{
        width: { xs: '100%', lg: '30%' },
        border: '1px solid',
        borderColor: 'divider',
        borderRadius: 1,
        overflow: 'hidden',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <Box sx={{ p: 2, borderBottom: '1px solid', borderBottomColor: 'divider' }}>
        <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 2, textTransform: 'uppercase' }}>
          Object Structure
        </Typography>
        <TextField
          fullWidth
          placeholder="Filter hierarchy..."
          variant="outlined"
          size="small"
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon />
              </InputAdornment>
            ),
          }}
        />
      </Box>

      <Box sx={{ flex: 1, overflow: 'auto', p: 1 }}>
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
          p: 2,
          borderTop: '1px solid',
          borderTopColor: 'divider',
          bgcolor: 'action.hover',
        }}
      >
        <Stack direction="row" justifyContent="space-between" alignItems="center">
          <Typography variant="caption" color="text.secondary">
            Last modified: {formatLastModified(businessObject?.updatedAt)}
          </Typography>
          <Typography variant="caption" color="text.secondary">
            {businessObject?.version}
          </Typography>
        </Stack>
      </Box>
    </Paper>
  );
}
