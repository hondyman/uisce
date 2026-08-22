import {
  Box,
  Typography,
  Stack,
  IconButton,
  Tooltip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Card,
  CardContent,
  Chip,
  Button,
} from '@mui/material';
import {
  AddLink as AddLinkIcon,
  Apps as AppsIcon,
  TableChart as TableChartIcon,
  AccountTree as AccountTreeIcon,
  Link as LinkIcon,
} from '@mui/icons-material';

interface RelatedObjectsTabProps {
  relatedObjectsView: 'tile' | 'table' | 'graph';
  relatedObjects?: Array<{
    relatedObjectName?: string;
    bo_name?: string;
    relationshipType?: string;
    edge?: string;
    description?: string;
  }>;
  onAddRelationship: () => void;
  onViewChange: (view: 'tile' | 'table' | 'graph') => void;
}

export function RelatedObjectsTab({
  relatedObjectsView,
  relatedObjects = [],
  onAddRelationship,
  onViewChange,
}: RelatedObjectsTabProps) {
  const items = relatedObjects.map(r => ({
    name: r.relatedObjectName || r.bo_name || 'Related Object',
    relationship: r.relationshipType || r.edge || 'Association',
    description: r.description || 'Linked via semantic catalog edge',
  }));

  return (
    <Box sx={{ p: 3 }}>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Stack direction="row" spacing={1.5} alignItems="center">
          <Typography variant="h6">Related Objects</Typography>
          <Chip label={items.length} size="small" color={items.length > 0 ? "primary" : "default"} />
        </Stack>
        <Stack direction="row" spacing={2} alignItems="center">
          <Button
            variant="contained"
            size="small"
            startIcon={<AddLinkIcon />}
            onClick={onAddRelationship}
          >
            Add Relationship
          </Button>
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
              <AppsIcon sx={{ fontSize: 24 }} />
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
              <TableChartIcon sx={{ fontSize: 24 }} />
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
              <AccountTreeIcon sx={{ fontSize: 24 }} />
            </IconButton>
          </Tooltip>
        </Stack>
      </Stack>

      {/* Tile View */}
      {relatedObjectsView === 'tile' && (
        items.length === 0 ? (
          <Box sx={{ textAlign: 'center', py: 6 }}>
            <LinkIcon sx={{ fontSize: 48, color: 'text.secondary', mb: 1.5, opacity: 0.6 }} />
            <Typography variant="subtitle1" fontWeight={600} gutterBottom>No Related Objects Yet</Typography>
            <Typography variant="body2" color="text.secondary" sx={{ maxWidth: 400, mx: 'auto', mb: 2 }}>
              Establish relationships to connect this business object with other canonical entities across the semantic catalog.
            </Typography>
            <Button variant="outlined" startIcon={<AddLinkIcon />} onClick={onAddRelationship}>
              Connect Object
            </Button>
          </Box>
        ) : (
          <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: 2 }}>
            {relatedObjects.map((item: any, idx) => (
              <Card key={idx} variant="outlined" sx={{ borderRadius: 2 }}>
                <CardContent>
                  <Stack direction="row" justifyContent="space-between" alignItems="flex-start" sx={{ mb: 1 }}>
                    <Typography variant="subtitle1" fontWeight={600}>
                      {item.relatedObjectName || item.bo_name || 'Related Object'}
                    </Typography>
                    <Stack direction="row" spacing={0.5}>
                      <Chip label={item.relationshipType || item.edge || 'Association'} size="small" variant="outlined" color="primary" />
                      {item.cardinality && <Chip label={item.cardinality} size="small" variant="filled" />}
                    </Stack>
                  </Stack>
                  <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                    {item.description || 'Linked via semantic catalog edge'}
                  </Typography>
                  {item.joinCondition && (
                    <Box sx={{ p: 1, bgcolor: 'action.hover', borderRadius: 1, border: '1px dashed', borderColor: 'divider' }}>
                      <Typography variant="caption" sx={{ fontFamily: 'monospace', color: 'primary.dark' }}>
                        🔗 {item.joinCondition}
                      </Typography>
                    </Box>
                  )}
                </CardContent>
              </Card>
            ))}
          </Box>
        )
      )}

      {/* Table View */}
      {relatedObjectsView === 'table' && (
        items.length === 0 ? (
          <Box sx={{ textAlign: 'center', py: 6 }}>
            <LinkIcon sx={{ fontSize: 48, color: 'text.secondary', mb: 1.5, opacity: 0.6 }} />
            <Typography variant="subtitle1" fontWeight={600} gutterBottom>No Related Objects Yet</Typography>
            <Typography variant="body2" color="text.secondary" sx={{ maxWidth: 400, mx: 'auto', mb: 2 }}>
              Establish relationships to connect this business object with other canonical entities across the semantic catalog.
            </Typography>
            <Button variant="outlined" startIcon={<AddLinkIcon />} onClick={onAddRelationship}>
              Connect Object
            </Button>
          </Box>
        ) : (
          <TableContainer component={Paper} variant="outlined" sx={{ borderRadius: 2 }}>
            <Table size="small">
              <TableHead sx={{ bgcolor: 'action.hover' }}>
                <TableRow>
                  <TableCell sx={{ fontWeight: 600 }}>Target Business Object</TableCell>
                  <TableCell sx={{ fontWeight: 600 }}>Relationship Type</TableCell>
                  <TableCell sx={{ fontWeight: 600 }}>Cardinality</TableCell>
                  <TableCell sx={{ fontWeight: 600 }}>Driver Table Join Keys / Binding</TableCell>
                  <TableCell sx={{ fontWeight: 600 }}>Description</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {relatedObjects.map((item: any, idx) => (
                  <TableRow key={idx} hover>
                    <TableCell sx={{ fontWeight: 600 }}>{item.relatedObjectName || item.bo_name || 'Related Object'}</TableCell>
                    <TableCell>
                      <Chip label={item.relationshipType || item.edge || 'Association'} size="small" color="primary" variant="outlined" />
                    </TableCell>
                    <TableCell>
                      <Chip label={item.cardinality || '1:N'} size="small" variant="filled" />
                    </TableCell>
                    <TableCell>
                      <Typography variant="caption" sx={{ fontFamily: 'monospace', color: 'primary.dark' }}>
                        {item.joinCondition || `${item.sourceDriverTable || 'driver_table'} ⟷ ${item.targetDriverTable || 'target_table'}`}
                      </Typography>
                    </TableCell>
                    <TableCell sx={{ color: 'text.secondary' }}>{item.description}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        )
      )}

      {/* Graph View */}
      {relatedObjectsView === 'graph' && (
        <Box sx={{ display: 'flex', flexDirection: 'column', justifyContent: 'center', alignItems: 'center', py: 6, minHeight: 300, bgcolor: 'background.default', borderRadius: 2, border: '1px dashed', borderColor: 'divider' }}>
          <AccountTreeIcon sx={{ fontSize: 48, color: 'primary.main', mb: 1, opacity: 0.8 }} />
          <Typography variant="subtitle1" fontWeight={600}>Graph Visualization</Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
            {items.length > 0
              ? `${items.length} connected entity relationships linked via catalog edges.`
              : 'No relationship edges to plot.'}
          </Typography>
        </Box>
      )}
    </Box>
  );
}
