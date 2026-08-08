import {
  Box,
  Typography,
  Stack,
  IconButton,
  Tooltip,
} from '@mui/material';
import {
  AddLink as AddLinkIcon,
  Apps as AppsIcon,
  TableChart as TableChartIcon,
  AccountTree as AccountTreeIcon,
} from '@mui/icons-material';

interface RelatedObjectsTabProps {
  relatedObjectsView: 'tile' | 'table' | 'graph';
  onAddRelationship: () => void;
  onViewChange: (view: 'tile' | 'table' | 'graph') => void;
}

export function RelatedObjectsTab({
  relatedObjectsView,
  onAddRelationship,
  onViewChange,
}: RelatedObjectsTabProps) {
  return (
    <Box sx={{ p: 3 }}>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Typography variant="h6">Related Objects</Typography>
        <Stack direction="row" spacing={2} alignItems="center">
          <Tooltip title="Add Relationship">
            <IconButton
              size="large"
              onClick={onAddRelationship}
              sx={{ color: 'primary.main' }}
            >
              <AddLinkIcon sx={{ fontSize: 32 }} />
            </IconButton>
          </Tooltip>
          <Tooltip title="Tile View">
            <IconButton
              size="medium"
              onClick={() => onViewChange('tile')}
              component="button"
              color={relatedObjectsView === 'tile' ? 'primary' : 'default'}
              sx={{
                border: relatedObjectsView === 'tile' ? '2px solid' : '1px solid',
                borderColor: relatedObjectsView === 'tile' ? 'primary.main' : 'divider',
              }}
            >
              <AppsIcon sx={{ fontSize: 28 }} />
            </IconButton>
          </Tooltip>
          <Tooltip title="Table View">
            <IconButton
              size="medium"
              onClick={() => onViewChange('table')}
              component="button"
              color={relatedObjectsView === 'table' ? 'primary' : 'default'}
              sx={{
                border: relatedObjectsView === 'table' ? '2px solid' : '1px solid',
                borderColor: relatedObjectsView === 'table' ? 'primary.main' : 'divider',
              }}
            >
              <TableChartIcon sx={{ fontSize: 28 }} />
            </IconButton>
          </Tooltip>
          <Tooltip title="Graph View">
            <IconButton
              size="medium"
              onClick={() => onViewChange('graph')}
              component="button"
              color={relatedObjectsView === 'graph' ? 'primary' : 'default'}
              sx={{
                border: relatedObjectsView === 'graph' ? '2px solid' : '1px solid',
                borderColor: relatedObjectsView === 'graph' ? 'primary.main' : 'divider',
              }}
            >
              <AccountTreeIcon sx={{ fontSize: 28 }} />
            </IconButton>
          </Tooltip>
        </Stack>
      </Stack>

      {/* Tile View */}
      {relatedObjectsView === 'tile' && (
        <Box>
          <Typography variant="body2" color="text.secondary" sx={{ textAlign: 'center', py: 5 }}>
            No related objects found. Related objects will appear here once they are linked to this business object.
          </Typography>
        </Box>
      )}

      {/* Table View */}
      {relatedObjectsView === 'table' && (
        <Box>
          <Typography variant="body2" color="text.secondary" sx={{ textAlign: 'center', py: 5 }}>
            No related objects found. Related objects will appear here once they are linked to this business object.
          </Typography>
        </Box>
      )}

      {/* Graph View */}
      {relatedObjectsView === 'graph' && (
        <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', py: 5, minHeight: 300 }}>
          <Typography variant="body2" color="text.secondary">
            No related objects to display. Graph visualization will appear here once relationships are established.
          </Typography>
        </Box>
      )}
    </Box>
  );
}
