import React from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  IconButton,
  Grid,
  CircularProgress,
  Chip,
} from '@mui/material';
import {
  ShowChart as ShowChartIcon,
  DeleteOutline as DeleteIcon,
  AccessTime as AccessTimeIcon,
  Storage as StorageIcon,
} from '@mui/icons-material';
import type { SavedExplorerQuery } from '../types/dataExplorerTypes';
import { useExplorerTheme } from '../hooks/useExplorerTheme';

interface SavedQueriesGridProps {
  records: SavedExplorerQuery[];
  loading?: boolean;
  onOpen: (record: SavedExplorerQuery) => void;
  onDelete: (id: string) => void;
}

function summarizeQueryState(state: SavedExplorerQuery['queryState']): string {
  const dims = state.dimensions.length + state.timeDimensions.length;
  const meas = state.measures.length;
  const filt = state.filters.length;
  const parts: string[] = [];
  if (dims) parts.push(`${dims} dimension${dims > 1 ? 's' : ''}`);
  if (meas) parts.push(`${meas} measure${meas > 1 ? 's' : ''}`);
  if (filt) parts.push(`${filt} filter${filt > 1 ? 's' : ''}`);
  return parts.join(' · ') || 'Empty query';
}

function relativeTime(iso?: string): string {
  if (!iso) return 'Just now';
  const ts = new Date(iso).getTime();
  const diff = Math.max(0, Date.now() - ts);
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return 'Just now';
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days}d ago`;
  return new Date(iso).toLocaleDateString();
}

export const SavedQueriesGrid: React.FC<SavedQueriesGridProps> = ({
  records = [],
  loading,
  onOpen,
  onDelete,
}) => {
  const theme = useExplorerTheme();
  const safeRecords = Array.isArray(records) ? records : [];

  return (
    <Paper
      elevation={0}
      sx={{
        p: 3,
        borderRadius: 3,
        border: `1px solid ${theme.border}`,
        bgcolor: theme.backgroundElevated,
      }}
    >
      <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 2 }}>
        <Stack direction="row" alignItems="center" spacing={1}>
          <AccessTimeIcon fontSize="small" sx={{ color: theme.textMuted }} />
          <Typography variant="h6" fontWeight={700} sx={{ color: theme.text }}>
            Recent Explorations
          </Typography>
        </Stack>
        <Typography variant="caption" sx={{ color: theme.textMuted }}>
          {safeRecords.length} saved
        </Typography>
      </Stack>

      {loading ? (
        <Box sx={{ py: 6, display: 'flex', justifyContent: 'center' }}>
          <CircularProgress size={28} sx={{ color: theme.accent }} />
        </Box>
      ) : safeRecords.length === 0 ? (
        <Box
          sx={{
            py: 4,
            textAlign: 'center',
            border: `1px dashed ${theme.border}`,
            borderRadius: 2,
          }}
        >
          <Typography variant="body2" sx={{ color: theme.textMuted }}>
            No saved queries yet. Build a query and save it to revisit here.
          </Typography>
        </Box>
      ) : (
        <Grid container spacing={2}>
          {safeRecords.map((record) => (
            <Grid size={{ xs: 12, sm: 6, md: 4 }} key={record.id}>
              <Paper
                elevation={0}
                sx={{
                  p: 2,
                  border: `1px solid ${theme.border}`,
                  borderRadius: 2,
                  cursor: 'pointer',
                  transition: 'all 0.2s',
                  bgcolor: theme.backgroundElevated,
                  '&:hover': {
                    borderColor: theme.accent,
                    transform: 'translateY(-2px)',
                    boxShadow: `0 4px 12px ${theme.isDark ? 'rgba(0,0,0,0.4)' : 'rgba(0,0,0,0.08)'}`,
                  },
                }}
                onClick={() => onOpen(record)}
              >
                <Stack direction="row" justifyContent="space-between" alignItems="flex-start" sx={{ mb: 1 }}>
                  <Box
                    sx={{
                      p: 0.75,
                      borderRadius: 1,
                      bgcolor: theme.background,
                      color: theme.text,
                    }}
                  >
                    <ShowChartIcon fontSize="small" />
                  </Box>
                  <IconButton
                    size="small"
                    onClick={(e) => {
                      e.stopPropagation();
                      onDelete(record.id);
                    }}
                    sx={{ color: theme.textMuted, mt: -1, mr: -1 }}
                  >
                    <DeleteIcon fontSize="small" />
                  </IconButton>
                </Stack>
                <Typography
                  variant="subtitle1"
                  fontWeight={600}
                  noWrap
                  sx={{ color: theme.text, mb: 0.5 }}
                >
                  {record.name}
                </Typography>
                <Typography variant="caption" sx={{ color: theme.textMuted, display: 'block', mb: 1 }}>
                  {summarizeQueryState(record.queryState)}
                </Typography>
                <Stack direction="row" spacing={0.5} alignItems="center">
                  <Chip
                    size="small"
                    label={record.sourceKind.replace('_', ' ')}
                    sx={{
                      height: 20,
                      fontSize: 10,
                      fontWeight: 700,
                      textTransform: 'uppercase',
                      bgcolor: theme.background,
                      color: theme.text,
                    }}
                  />
                  <Typography variant="caption" sx={{ color: theme.textMuted, ml: 'auto' }}>
                    {relativeTime(record.updatedAt ?? record.createdAt)}
                  </Typography>
                </Stack>
              </Paper>
            </Grid>
          ))}
        </Grid>
      )}
    </Paper>
  );
};

export default SavedQueriesGrid;
