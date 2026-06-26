import React, { useState } from 'react';
import { Box, Button, Dialog, DialogTitle, DialogContent, DialogActions, Table, TableHead, TableRow, TableCell, TableBody, TextField, CircularProgress, FormHelperText, Chip, Stack, IconButton, Tooltip } from '@mui/material';
import { Edit as EditIcon, Delete as DeleteIcon } from '@mui/icons-material';
import { useInfiniteLookupValues, updateLookupValue } from '../../../api/lookups';
import { useNotification } from '../../../hooks/useNotification';
import { useQueryClient } from '@tanstack/react-query';

/**
 * FlatLookupValuesPanel - specialized UX for flat table lookups like semantic_types and data_domains
 * Shows metadata in a more structured way and hides the "Parent" column
 */
export function FlatLookupValuesPanel({
  tenantId,
  lookupId,
  onRequestDelete,
}: {
  tenantId: string;
  lookupId: string;
  onRequestDelete?: (id: string) => void;
}) {
  const notification = useNotification();
  const { data, isLoading, fetchNextPage, hasNextPage, isFetchingNextPage } = useInfiniteLookupValues(tenantId, lookupId, null, null, 50);
  const qc = useQueryClient();
  const [searchValues, setSearchValues] = useState('');
  const [editValue, setEditValue] = useState<any | null>(null);
  const [valueFormError, setValueFormError] = useState<string | null>(null);

  const displayValues = data?.pages.flatMap((page: any) => page.items) || [];

  const filteredValues = displayValues.filter((v: any) =>
    v.name.toLowerCase().includes(searchValues.toLowerCase()) ||
    JSON.stringify(v.metadata || {}).toLowerCase().includes(searchValues.toLowerCase())
  );

  const handleOpenEditValue = (v: any) =>
    setEditValue({
      id: v.id,
      value: v.value || v.name,
      label: v.label || v.name,
      metadata: v.metadata || {},
    });

  const handleUpdateValue = async (id: string, value: string, label?: string, metadata?: any) => {
    if (!tenantId) return;
    await updateLookupValue(tenantId, lookupId, id, { value, label, metadata });
    notification.success('Lookup value updated');
    qc.invalidateQueries({ queryKey: ['lookup-values', tenantId, lookupId] });
    qc.invalidateQueries({ queryKey: ['lookup-values/infinite', tenantId, lookupId] });
    setEditValue(null);
  };

  if (isLoading) return <CircularProgress size={18} />;

  // Detect if this is a semantic_types lookup by checking metadata structure
  const isSemanticsLookup = filteredValues.length > 0 && filteredValues[0].metadata?.semantic_type;

  if (isSemanticsLookup) {
    // Render semantic types in a structured card/grid layout
    return (
      <>
        <Box sx={{ display: 'flex', gap: 2, mb: 2 }}>
          <TextField
            label="Search semantic types"
            value={searchValues}
            onChange={(e) => setSearchValues(e.target.value)}
            size="small"
            fullWidth
            placeholder="Search by name or metadata..."
          />
        </Box>

        <Box
          sx={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(350px, 1fr))',
            gap: 2,
            mb: 2,
          }}
        >
          {filteredValues.map((v: any) => (
            <Box
              key={v.id}
              sx={{
                p: 2,
                border: '1px solid #e0e0e0',
                borderRadius: 1,
                backgroundColor: '#fafafa',
                transition: 'all 0.2s',
                '&:hover': {
                  boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
                  backgroundColor: '#fff',
                },
              }}
            >
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 1 }}>
                <Box sx={{ flex: 1 }}>
                  <Box sx={{ fontWeight: 600, fontSize: '0.95rem', mb: 0.5 }}>{v.label || v.name}</Box>
                  <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                    <Chip
                      label={v.metadata?.semantic_type || 'Unknown'}
                      size="small"
                      color={
                        v.metadata?.semantic_type === 'Dimension'
                          ? 'primary'
                          : v.metadata?.semantic_type === 'Measure'
                            ? 'success'
                            : 'default'
                      }
                      variant="filled"
                    />
                    <Chip label={v.metadata?.data_type || ''} size="small" variant="outlined" />
                    <Chip label={v.metadata?.format || 'default'} size="small" variant="outlined" />
                    {v.is_core ? (
                      <Chip 
                        label="CORE" 
                        size="small" 
                        color="info" 
                        title="Cloned from Gold Copy"
                        sx={{ height: 24, fontSize: '0.7rem', fontWeight: 'bold' }} 
                      />
                    ) : (
                      <Chip 
                        label="CUSTOM" 
                        size="small" 
                        variant="outlined"
                        sx={{ height: 24, fontSize: '0.7rem', fontWeight: 'bold' }} 
                      />
                    )}
                  </Box>
                </Box>
              </Box>

              {v.metadata?.notes && (
                <Box sx={{ fontSize: '0.85rem', color: '#666', mt: 1, fontStyle: 'italic', mb: 1 }}>
                  {v.metadata.notes}
                </Box>
              )}

              <Box sx={{ display: 'flex', gap: 1, pt: 1 }}>
                <Tooltip title={v.tenant_id === tenantId ? "Edit" : "Read Only"}>
                  <span>
                    <IconButton size="small" onClick={() => handleOpenEditValue(v)} color="primary" disabled={v.tenant_id !== tenantId}>
                      <EditIcon fontSize="small" />
                    </IconButton>
                  </span>
                </Tooltip>
                <Tooltip title={v.tenant_id === tenantId ? "Delete" : "Read Only"}>
                   <span>
                    <IconButton size="small" onClick={() => onRequestDelete?.(v.id)} color="error" disabled={v.tenant_id !== tenantId}>
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  </span>
                </Tooltip>
              </Box>
            </Box>
          ))}
        </Box>

        {hasNextPage && (
          <Box sx={{ display: 'flex', justifyContent: 'center', mt: 2 }}>
            <Button
              variant="outlined"
              size="small"
              onClick={() => fetchNextPage()}
              disabled={isFetchingNextPage}
            >
              {isFetchingNextPage ? <CircularProgress size={16} /> : 'Load More'}
            </Button>
          </Box>
        )}

        {/* Edit Dialog */}
        <Dialog open={!!editValue} onClose={() => setEditValue(null)} maxWidth="sm" fullWidth>
          <DialogTitle>Edit Semantic Type</DialogTitle>
          <DialogContent>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 2 }}>
              <TextField
                autoFocus
                margin="dense"
                label="Label"
                fullWidth
                value={editValue?.label || ''}
                onChange={(e) => setEditValue((s: any) => (s ? { ...s, label: e.target.value } : s))}
              />
              {editValue?.metadata?.semantic_type && (
                <Box sx={{ p: 1.5, backgroundColor: '#f5f5f5', borderRadius: 1 }}>
                  <Box sx={{ fontSize: '0.9rem', fontWeight: 600, mb: 1 }}>Metadata</Box>
                  <Stack spacing={1} sx={{ fontSize: '0.85rem' }}>
                    <Box>
                      <strong>Type:</strong> {editValue.metadata.semantic_type}
                    </Box>
                    <Box>
                      <strong>Data Type:</strong> {editValue.metadata.data_type}
                    </Box>
                    <Box>
                      <strong>Format:</strong> {editValue.metadata.format}
                    </Box>
                    {editValue.metadata.notes && (
                      <Box>
                        <strong>Notes:</strong> {editValue.metadata.notes}
                      </Box>
                    )}
                  </Stack>
                </Box>
              )}
              {valueFormError && <FormHelperText error>{valueFormError}</FormHelperText>}
            </Box>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setEditValue(null)}>Cancel</Button>
            <Button
              variant="contained"
              onClick={() => {
                setValueFormError(null);
                if (!editValue?.label) {
                  setValueFormError('Label required');
                  return;
                }
                handleUpdateValue(editValue.id, editValue.value, editValue.label, editValue.metadata);
              }}
            >
              Save
            </Button>
          </DialogActions>
        </Dialog>
      </>
    );
  }

  // Fallback to standard table for other flat lookups
  return (
    <>
      <Box sx={{ display: 'flex', gap: 2, mb: 2, alignItems: 'center' }}>
        <TextField
          label="Search values"
          value={searchValues}
          onChange={(e) => setSearchValues(e.target.value)}
          size="small"
          fullWidth
          placeholder="Search by name or label..."
        />
      </Box>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Name</TableCell>
            <TableCell>Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {filteredValues.map((v: any) => (
            <TableRow key={v.id}>
              <TableCell>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  {v.name}
                  {v.is_core ? (
                    <Chip 
                      label="CORE" 
                      size="small" 
                      color="info" 
                      title="Cloned from Gold Copy"
                      sx={{ height: 20, fontSize: '0.65rem', fontWeight: 'bold' }} 
                    />
                  ) : (
                    <Chip 
                      label="CUSTOM" 
                      size="small" 
                      variant="outlined"
                      sx={{ height: 20, fontSize: '0.65rem', fontWeight: 'bold' }} 
                    />
                  )}
                </Box>
              </TableCell>
              <TableCell>
                <Tooltip title={v.tenant_id === tenantId ? "Edit" : "Read Only"}>
                  <span>
                    <IconButton size="small" onClick={() => handleOpenEditValue(v)} color="primary" disabled={v.tenant_id !== tenantId}>
                      <EditIcon fontSize="small" />
                    </IconButton>
                  </span>
                </Tooltip>
                <Tooltip title={v.tenant_id === tenantId ? "Delete" : "Read Only"}>
                  <span>
                    <IconButton size="small" onClick={() => onRequestDelete?.(v.id)} color="error" disabled={v.tenant_id !== tenantId}>
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  </span>
                </Tooltip>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      {hasNextPage && (
        <Box sx={{ display: 'flex', justifyContent: 'center', mt: 2 }}>
          <Button
            variant="outlined"
            size="small"
            onClick={() => fetchNextPage()}
            disabled={isFetchingNextPage}
          >
            {isFetchingNextPage ? <CircularProgress size={16} /> : 'Load More'}
          </Button>
        </Box>
      )}
    </>
  );
}
