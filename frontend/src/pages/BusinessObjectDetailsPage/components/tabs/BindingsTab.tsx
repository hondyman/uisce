import {
  Box,
  Typography,
  Paper,
  Stack,
  Grid,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from '@mui/material';
import { Storage as StorageIcon } from '@mui/icons-material';

const MAX_FIELDS_DISPLAY = 10;

interface BindingsTabProps {
  bindings: any[];
}

export function BindingsTab({ bindings }: BindingsTabProps) {
  return (
    <Box sx={{ p: 3 }}>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Box>
          <Typography variant="h6" sx={{ fontWeight: 700 }}>
            Physical Source Bindings
          </Typography>
          <Typography variant="body2" color="text.secondary">
            {bindings.length} binding{bindings.length !== 1 ? 's' : ''} configured for this Business Object
          </Typography>
        </Box>
      </Stack>

      {bindings.length === 0 ? (
        <Paper variant="outlined" sx={{ p: 4, textAlign: 'center' }}>
          <StorageIcon sx={{ fontSize: 48, color: 'text.disabled', mb: 2 }} />
          <Typography variant="h6" color="text.secondary" gutterBottom>
            No Bindings Configured
          </Typography>
          <Typography variant="body2" color="text.secondary">
            This Business Object does not yet have physical source bindings.
            Bindings define how this object maps to physical database tables.
          </Typography>
        </Paper>
      ) : (
        <Stack spacing={2}>
          {bindings.map((binding: any, index: number) => (
            <Paper key={binding.boBindingId || index} variant="outlined" sx={{ p: 2 }}>
              <Grid container spacing={2}>
                <Grid size={12}>
                  <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 1 }}>
                    <StorageIcon color="primary" />
                    <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                      {binding.bindingName || `Binding ${index + 1}`}
                    </Typography>
                    {binding.isActive !== undefined && (
                      <Chip
                        label={binding.isActive ? 'Active' : 'Inactive'}
                        size="small"
                        color={binding.isActive ? 'success' : 'default'}
                      />
                    )}
                    {binding.isCore && (
                      <Chip label="Core" size="small" variant="outlined" />
                    )}
                  </Stack>
                </Grid>
                <Grid size={{ xs: 12, md: 6 }}>
                  <Typography variant="caption" color="text.secondary">Binding ID</Typography>
                  <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                    {binding.boBindingId || 'N/A'}
                  </Typography>
                </Grid>
                <Grid size={{ xs: 12, md: 6 }}>
                  <Typography variant="caption" color="text.secondary">Backend</Typography>
                  <Typography variant="body2">
                    {binding.backendName || binding.backendId || 'N/A'}
                  </Typography>
                </Grid>
                <Grid size={{ xs: 12, md: 6 }}>
                  <Typography variant="caption" color="text.secondary">Driving Node</Typography>
                  <Typography variant="body2">
                    {binding.drivingNodeName || binding.drivingNodeId || 'N/A'}
                  </Typography>
                </Grid>
                <Grid size={{ xs: 12, md: 6 }}>
                  <Typography variant="caption" color="text.secondary">Temporal Mode</Typography>
                  <Typography variant="body2">
                    {binding.temporalMode || 'N/A'}
                  </Typography>
                </Grid>
                {binding.baseSql && (
                  <Grid size={12}>
                    <Typography variant="caption" color="text.secondary">Base SQL</Typography>
                    <Paper variant="outlined" sx={{ p: 1, mt: 0.5, bgcolor: 'grey.50' }}>
                      <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: '0.75rem' }}>
                        {binding.baseSql}
                      </Typography>
                    </Paper>
                  </Grid>
                )}
                {binding.Fields && binding.Fields.length > 0 && (
                  <Grid size={12}>
                    <Typography variant="caption" color="text.secondary" sx={{ mb: 1, display: 'block' }}>
                      Field Mappings ({binding.Fields.length})
                    </Typography>
                    <TableContainer component={Paper} variant="outlined">
                      <Table size="small">
                        <TableHead>
                          <TableRow>
                            <TableCell>Field Name</TableCell>
                            <TableCell>Role</TableCell>
                            <TableCell>Data Type</TableCell>
                            <TableCell>PK</TableCell>
                            <TableCell>Aggregation</TableCell>
                          </TableRow>
                        </TableHead>
                        <TableBody>
                          {binding.Fields.slice(0, MAX_FIELDS_DISPLAY).map((field: any, fIdx: number) => (
                            <TableRow key={field.fieldId || fIdx}>
                              <TableCell>{field.fieldName}</TableCell>
                              <TableCell>
                                <Chip label={field.fieldRole || 'ATTRIBUTE'} size="small" variant="outlined" />
                              </TableCell>
                              <TableCell>{field.dataType}</TableCell>
                              <TableCell>{field.isPrimaryKey ? 'Yes' : 'No'}</TableCell>
                              <TableCell>{field.aggregationType || 'NONE'}</TableCell>
                            </TableRow>
                          ))}
                          {binding.Fields.length > MAX_FIELDS_DISPLAY && (
                            <TableRow>
                              <TableCell colSpan={5} align="center">
                                <Typography variant="body2" color="text.secondary">
                                  +{binding.Fields.length - MAX_FIELDS_DISPLAY} more fields
                                </Typography>
                              </TableCell>
                            </TableRow>
                          )}
                        </TableBody>
                      </Table>
                    </TableContainer>
                  </Grid>
                )}
              </Grid>
            </Paper>
          ))}
        </Stack>
      )}
    </Box>
  );
}
