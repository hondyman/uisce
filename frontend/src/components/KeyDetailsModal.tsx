import { useMemo, useState, useRef, useCallback } from 'react';
import {
  Dialog,
  DialogContent,
  DialogActions,
  Button,
  Table,
  TableHead,
  TableBody,
  TableRow,
  TableCell,
  TextField,
  Box,
  Typography,
  Chip,
} from '@mui/material';
import KeyIcon from '@mui/icons-material/Key';
import ModalHeader from './ModalHeader';
import { ColumnData } from '../types/SemanticTypes';

interface KeyDetailsModalProps {
  open: boolean;
  onClose: () => void;
  tableName?: string;
  keyType: 'primary' | 'foreign';
  columns: ColumnData[];
}

/**
 * Modal to display primary or foreign key columns for a table.
 * Shows filtered columns that are marked as primary or foreign keys.
 */
const KeyDetailsModal: React.FC<KeyDetailsModalProps> = ({ 
  open, 
  onClose, 
  tableName, 
  keyType,
  columns 
}) => {
  const [filter, setFilter] = useState('');
  const [displayCount, setDisplayCount] = useState(50);
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  const isPrimary = keyType === 'primary';
  const keyLabel = isPrimary ? 'Primary Key' : 'Foreign Key';
  const keyColor = isPrimary ? '#d32f2f' : '#1976d2';

  // Filter columns by key type and search term
  const filtered = useMemo(() => {
    const keyColumns = columns.filter(c => 
      isPrimary ? c.isPrimaryKey : c.isForeignKey
    );
    
    const f = filter.trim().toLowerCase();
    if (!f) return keyColumns;
    
    return keyColumns.filter((c) => {
      return (
        (c.name || '').toString().toLowerCase().includes(f) ||
        (c.type || '').toString().toLowerCase().includes(f) ||
        (c.description || '').toString().toLowerCase().includes(f)
      );
    });
  }, [columns, filter, isPrimary]);

  // Lazy loading: show only displayCount rows
  const visibleRows = useMemo(() => {
    return filtered.slice(0, displayCount);
  }, [filtered, displayCount]);

  // Infinite scroll handler
  const handleScroll = useCallback(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    const { scrollTop, scrollHeight, clientHeight } = container;
    if (scrollTop + clientHeight >= scrollHeight * 0.8) {
      setDisplayCount((prev) => Math.min(prev + 50, filtered.length));
    }
  }, [filtered.length]);

  // Reset display count when filter changes
  useMemo(() => {
    setDisplayCount(50);
  }, [filter]);

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="md">
      <ModalHeader 
        title={`${keyLabel} Columns`}
        subtitle={tableName || 'table'} 
        chipLabel={`${filtered.length} ${isPrimary ? 'PK' : 'FK'}`}
        onClose={onClose} 
        bg={isPrimary ? 'warning.light' : 'info.light'}
      />
      <DialogContent 
        ref={scrollContainerRef}
        onScroll={handleScroll}
        sx={{ p: 0 }}
      >
        <Box sx={{ p: 3, pb: 2, position: 'sticky', top: 0, bgcolor: 'background.paper', zIndex: 1 }}>
          <TextField
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            placeholder="Search key columns by name, type, description..."
            size="small"
            fullWidth
          />
          <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
            Showing {visibleRows.length} of {filtered.length} {keyLabel.toLowerCase()} columns
            {filter && ` (filtered)`}
          </Typography>
        </Box>

        {filtered.length === 0 ? (
          <Box sx={{ p: 4, textAlign: 'center' }}>
            <KeyIcon sx={{ fontSize: 48, color: 'text.disabled', mb: 2 }} />
            <Typography variant="body1" color="text.secondary">
              {filter 
                ? `No ${keyLabel.toLowerCase()} columns match your search.`
                : `This table has no ${keyLabel.toLowerCase()} columns.`
              }
            </Typography>
          </Box>
        ) : (
          <Table size="small" sx={{ tableLayout: 'fixed' }}>
            <TableHead sx={{ 
              position: 'sticky', 
              top: 88, 
              bgcolor: 'grey.300', 
              zIndex: 1,
              '& th': { 
                fontWeight: 600,
                color: 'text.primary',
                borderBottom: '2px solid',
                borderBottomColor: 'grey.400'
              }
            }}>
              <TableRow>
                <TableCell sx={{ width: '10%' }}>Key</TableCell>
                <TableCell sx={{ width: '30%' }}>Column Name</TableCell>
                <TableCell sx={{ width: '20%' }}>Type</TableCell>
                <TableCell sx={{ width: '40%' }}>Description</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {visibleRows.map((col, idx) => (
                <TableRow 
                  key={col.name || idx} 
                  hover
                  sx={{
                    bgcolor: idx % 2 === 0 ? 'background.paper' : 'action.hover',
                    '&:hover': {
                      bgcolor: 'action.selected',
                    }
                  }}
                >
                  <TableCell>
                    <Chip
                      icon={<KeyIcon sx={{ color: `${keyColor} !important` }} />}
                      label={isPrimary ? 'PK' : 'FK'}
                      size="small"
                      variant="outlined"
                      sx={{ 
                        borderColor: keyColor,
                        '& .MuiChip-label': { fontWeight: 600 }
                      }}
                    />
                  </TableCell>
                  <TableCell sx={{ fontWeight: 500 }}>{col.name}</TableCell>
                  <TableCell>
                    <Typography 
                      variant="body2" 
                      sx={{ 
                        fontFamily: 'monospace', 
                        fontSize: '0.85rem',
                        color: 'primary.main'
                      }}
                    >
                      {col.type}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2" color="text.secondary">
                      {col.description || '—'}
                    </Typography>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}

        {visibleRows.length < filtered.length && (
          <Box sx={{ p: 2, textAlign: 'center', bgcolor: 'background.paper' }}>
            <Typography variant="body2" color="text.secondary">
              Scroll down to load more columns...
            </Typography>
          </Box>
        )}
      </DialogContent>
      <DialogActions sx={{ px: 3, py: 2 }}>
        <Button onClick={onClose} variant="contained">Close</Button>
      </DialogActions>
    </Dialog>
  );
};

export default KeyDetailsModal;
