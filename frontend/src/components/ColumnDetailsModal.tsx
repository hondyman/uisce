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
  Tooltip,
  Typography,
  Chip,
} from '@mui/material';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import KeyIcon from '@mui/icons-material/Key';
import ModalHeader from './ModalHeader';
import { ColumnData } from '../types/SemanticTypes';

interface ColumnDetailsModalProps {
  open: boolean;
  onClose: () => void;
  tableName?: string;
  columns: ColumnData[];
}

const ColumnDetailsModal: React.FC<ColumnDetailsModalProps> = ({ open, onClose, tableName, columns }) => {
  const [filter, setFilter] = useState('');
  const [displayCount, setDisplayCount] = useState(50); // Initial load
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  const filtered = useMemo(() => {
    const f = filter.trim().toLowerCase();
    if (!f) return columns;
    return columns.filter((c) => {
      return (
        (c.name || '').toString().toLowerCase().includes(f) ||
        (c.type || '').toString().toLowerCase().includes(f) ||
        (c.description || '').toString().toLowerCase().includes(f)
      );
    });
  }, [columns, filter]);

  // Lazy loading: show only displayCount rows
  const visibleRows = useMemo(() => {
    return filtered.slice(0, displayCount);
  }, [filtered, displayCount]);

  // Infinite scroll handler
  const handleScroll = useCallback(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    const { scrollTop, scrollHeight, clientHeight } = container;
    // Load more when scrolled to 80% of content
    if (scrollTop + clientHeight >= scrollHeight * 0.8) {
      setDisplayCount((prev) => Math.min(prev + 50, filtered.length));
    }
  }, [filtered.length]);

  // Reset display count when filter changes
  useMemo(() => {
    setDisplayCount(50);
  }, [filter]);

  const renderBool = (v: any, label: string) =>
    v ? (
      <Tooltip title={`${label}: Yes`}>
        <span>
          <CheckCircleIcon fontSize="small" sx={{ color: '#81c784' }} aria-label="true" />
        </span>
      </Tooltip>
    ) : (
      <Tooltip title={`${label}: No`}>
        <span>
          <CancelIcon fontSize="small" sx={{ color: '#e57373' }} aria-label="false" />
        </span>
      </Tooltip>
    );

  const renderKeys = (col: ColumnData) => {
    const keys = [];
    
    if (col.isPrimaryKey) {
      keys.push(
        <Tooltip key="pk" title="Primary Key">
          <KeyIcon fontSize="small" sx={{ color: '#d32f2f', mr: 0.5 }} />
        </Tooltip>
      );
    }
    
    if (col.isForeignKey) {
      keys.push(
        <Tooltip key="fk" title="Foreign Key">
          <KeyIcon fontSize="small" sx={{ color: '#1976d2' }} />
        </Tooltip>
      );
    }
    
    return keys.length > 0 ? <Box sx={{ display: 'flex', alignItems: 'center' }}>{keys}</Box> : <span>—</span>;
  };

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="lg">
      <ModalHeader 
        title="Columns" 
        subtitle={tableName || 'table'} 
        chipLabel={`${columns.length} cols`} 
        onClose={onClose} 
        bg="primary.light" 
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
            placeholder="Search columns by name, type, description..."
            size="small"
            fullWidth
          />
          <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
            Showing {visibleRows.length} of {filtered.length} columns
            {filter && ` (filtered from ${columns.length} total)`}
          </Typography>
        </Box>

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
                <TableCell sx={{ width: '20%' }}>Name</TableCell>
                <TableCell sx={{ width: '10%' }}>Type</TableCell>
                <TableCell align="center" sx={{ width: '8%' }}>Keys</TableCell>
                <TableCell sx={{ width: '20%' }}>Semantic Terms</TableCell>
                <TableCell align="center" sx={{ width: '8%' }}>Nullable</TableCell>
                <TableCell align="center" sx={{ width: '7%' }}>Core</TableCell>
                <TableCell sx={{ width: '27%' }}>Description</TableCell>
              </TableRow>
          </TableHead>
          <TableBody>
            {visibleRows.map((col, idx) => {
              return (
                <TableRow 
                  key={col.name || idx} 
                  hover
                  sx={{
                    // Alternating row colors (banded)
                    bgcolor: idx % 2 === 0 ? 'background.paper' : 'action.hover',
                    '&:hover': {
                      bgcolor: 'action.selected',
                    }
                  }}
                >
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
                  <TableCell align="center">{renderKeys(col)}</TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                      {(col as any).semanticTerms && (col as any).semanticTerms.length > 0 ? (
                        (col as any).semanticTerms.map((term: any) => (
                          <Chip
                            key={term.id}
                            label={term.node_name}
                            size="small"
                            variant="outlined"
                            sx={{ 
                              height: 20, 
                              fontSize: '0.7rem',
                              borderColor: 'primary.light',
                              color: 'primary.main',
                              bgcolor: 'primary.50'
                            }}
                          />
                        ))
                      ) : (
                        <Typography variant="caption" color="text.disabled" sx={{ fontStyle: 'italic' }}>None</Typography>
                      )}
                    </Box>
                  </TableCell>
                  <TableCell align="center">{renderBool(Boolean(col.nullable), 'Nullable')}</TableCell>
                  <TableCell align="center">{renderBool(Boolean(col.isCore), 'Core Column')}</TableCell>
                  <TableCell>
                    <Typography variant="body2" color="text.secondary">
                      {col.description || '—'}
                    </Typography>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>

        {visibleRows.length < filtered.length && (
          <Box sx={{ p: 2, textAlign: 'center', bgcolor: 'background.paper' }}>
            <Typography variant="body2" color="text.secondary">
              Scroll down to load more columns...
            </Typography>
          </Box>
        )}

        {visibleRows.length === 0 && (
          <Box sx={{ p: 4, textAlign: 'center' }}>
            <Typography variant="body1" color="text.secondary">
              {filter ? 'No columns match your search.' : 'No columns found.'}
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

export default ColumnDetailsModal;
