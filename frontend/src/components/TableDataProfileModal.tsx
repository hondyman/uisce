// React import not needed - automatic JSX runtime in use
import { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  IconButton,
  Box,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Chip,
  LinearProgress,
  Tooltip,
  TextField,
  InputAdornment,
} from '@mui/material';
import {
  Close as CloseIcon,
  Assessment as AssessmentIcon,
  Search as SearchIcon,
} from '@mui/icons-material';

interface ColumnProfile {
  name: string;
  data_type?: string;
  unique_count?: number;
  total_count?: number;
  cardinality_ratio?: number;
  is_low_cardinality?: boolean;
  is_nullable?: boolean;
  sample_values?: string[];
  detected_format?: string;
  max_length?: number;
}

interface TableDataProfileModalProps {
  open: boolean;
  onClose: () => void;
  tableName: string;
  columns: ColumnProfile[];
}

const TableDataProfileModal: React.FC<TableDataProfileModalProps> = ({
  open,
  onClose,
  tableName,
  columns,
}) => {
  const [searchTerm, setSearchTerm] = useState('');

  const filteredColumns = columns.filter((col) =>
    col.name.toLowerCase().includes(searchTerm.toLowerCase())
  );

  // Calculate table-level stats
  const totalColumns = columns.length;
  const columnsWithProfile = columns.filter(c => c.total_count != null && c.total_count > 0).length;
  const lowCardinalityCount = columns.filter(c => c.is_low_cardinality).length;

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="lg"
      fullWidth
      PaperProps={{
        sx: {
          bgcolor: 'background.paper',
          backgroundImage: 'none',
          borderRadius: 2,
          maxHeight: '85vh',
        },
      }}
    >
      <DialogTitle
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          borderBottom: 1,
          borderColor: 'divider',
          pb: 2,
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <AssessmentIcon color="primary" />
          <Typography variant="h6" component="span">
            Data Profile: {tableName}
          </Typography>
        </Box>
        <IconButton onClick={onClose} size="small">
          <CloseIcon />
        </IconButton>
      </DialogTitle>

      <DialogContent sx={{ p: 3 }}>
        {/* Summary Stats */}
        <Box sx={{ display: 'flex', gap: 3, mb: 3 }}>
          <Box
            sx={{
              p: 2,
              bgcolor: 'action.hover',
              borderRadius: 1,
              textAlign: 'center',
              flex: 1,
            }}
          >
            <Typography variant="h4" color="primary.main" fontWeight="bold">
              {totalColumns}
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Total Columns
            </Typography>
          </Box>
          <Box
            sx={{
              p: 2,
              bgcolor: 'action.hover',
              borderRadius: 1,
              textAlign: 'center',
              flex: 1,
            }}
          >
            <Typography variant="h4" color="success.main" fontWeight="bold">
              {columnsWithProfile}
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Profiled Columns
            </Typography>
          </Box>
          <Box
            sx={{
              p: 2,
              bgcolor: 'action.hover',
              borderRadius: 1,
              textAlign: 'center',
              flex: 1,
            }}
          >
            <Typography variant="h4" color="warning.main" fontWeight="bold">
              {lowCardinalityCount}
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Low Cardinality
            </Typography>
          </Box>
        </Box>

        {/* Search */}
        <TextField
          fullWidth
          size="small"
          placeholder="Search columns..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          sx={{ mb: 2 }}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon fontSize="small" />
              </InputAdornment>
            ),
          }}
        />

        {/* Column Profiles Table */}
        <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: 400 }}>
          <Table stickyHeader size="small">
            <TableHead>
              <TableRow>
                <TableCell sx={{ fontWeight: 'bold', width: '20%' }}>Column</TableCell>
                <TableCell sx={{ fontWeight: 'bold', width: '12%' }}>Data Type</TableCell>
                <TableCell sx={{ fontWeight: 'bold', width: '10%' }} align="right">Total</TableCell>
                <TableCell sx={{ fontWeight: 'bold', width: '10%' }} align="right">Unique</TableCell>
                <TableCell sx={{ fontWeight: 'bold', width: '18%' }}>Cardinality</TableCell>
                <TableCell sx={{ fontWeight: 'bold', width: '10%' }}>Nullable</TableCell>
                <TableCell sx={{ fontWeight: 'bold', width: '20%' }}>Sample Values</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredColumns.map((col) => {
                const cardinalityPercent = col.cardinality_ratio
                  ? Math.round(col.cardinality_ratio * 100)
                  : null;

                return (
                  <TableRow key={col.name} hover>
                    <TableCell>
                      <Typography variant="body2" fontWeight="medium">
                        {col.name}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={col.data_type || 'unknown'}
                        size="small"
                        variant="outlined"
                        sx={{ fontSize: '0.7rem' }}
                      />
                    </TableCell>
                    <TableCell align="right">
                      <Typography variant="body2">
                        {col.total_count?.toLocaleString() ?? '-'}
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Typography variant="body2">
                        {col.unique_count?.toLocaleString() ?? '-'}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      {cardinalityPercent != null ? (
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <LinearProgress
                            variant="determinate"
                            value={cardinalityPercent}
                            sx={{
                              flex: 1,
                              height: 6,
                              borderRadius: 3,
                              bgcolor: 'action.disabledBackground',
                              '& .MuiLinearProgress-bar': {
                                bgcolor: col.is_low_cardinality
                                  ? 'warning.main'
                                  : 'success.main',
                              },
                            }}
                          />
                          <Typography variant="caption" sx={{ minWidth: 35 }}>
                            {cardinalityPercent}%
                          </Typography>
                        </Box>
                      ) : (
                        <Typography variant="body2" color="text.secondary">
                          -
                        </Typography>
                      )}
                    </TableCell>
                    <TableCell>
                      {col.is_nullable != null && (
                        <Chip
                          label={col.is_nullable ? 'Yes' : 'No'}
                          size="small"
                          color={col.is_nullable ? 'default' : 'error'}
                          variant="outlined"
                          sx={{ fontSize: '0.7rem' }}
                        />
                      )}
                    </TableCell>
                    <TableCell>
                      {col.sample_values && col.sample_values.length > 0 ? (
                        <Tooltip
                          title={col.sample_values.slice(0, 10).join(', ')}
                          arrow
                        >
                          <Typography
                            variant="body2"
                            sx={{
                              fontFamily: 'monospace',
                              fontSize: '0.75rem',
                              maxWidth: 150,
                              overflow: 'hidden',
                              textOverflow: 'ellipsis',
                              whiteSpace: 'nowrap',
                            }}
                          >
                            {col.sample_values.slice(0, 3).join(', ')}
                            {col.sample_values.length > 3 && '...'}
                          </Typography>
                        </Tooltip>
                      ) : (
                        <Typography variant="body2" color="text.secondary">
                          -
                        </Typography>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </TableContainer>

        {filteredColumns.length === 0 && (
          <Box sx={{ textAlign: 'center', py: 4 }}>
            <Typography color="text.secondary">
              {searchTerm
                ? 'No columns match your search'
                : 'No column profile data available. Run a schema scan to generate profiling data.'}
            </Typography>
          </Box>
        )}
      </DialogContent>
    </Dialog>
  );
};

export default TableDataProfileModal;
