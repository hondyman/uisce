import React, { useState } from 'react';
import {
  Box,
  Typography,
  Select,
  MenuItem,
  Button,
  CircularProgress,
  Chip,
  Paper,
  Stack,
  Alert,
} from '@mui/material';
import {
  Storage as TableIcon,
  AutoAwesome as AIIcon,
  CheckCircle as CheckIcon,
  ArrowForward as ArrowIcon,
} from '@mui/icons-material';

export interface DriverTableOption {
  tableId: string;
  tableName: string;
  schema: string;
  fkCount: number;
  isSuggested?: boolean;
}

export interface DriverTableSelectorProps {
  datasourceName?: string;
  schemaName?: string;
  tables: DriverTableOption[];
  onSelectTable: (tableId: string, tableName: string) => Promise<void>;
}

export const DriverTableSelector: React.FC<DriverTableSelectorProps> = ({
  datasourceName = 'CRIMS ORM (crims.orm)',
  schemaName = 'orm',
  tables = [],
  onSelectTable,
}) => {
  const [selectedTableId, setSelectedTableId] = useState<string>('');
  const [loading, setLoading] = useState(false);

  const handleNext = async () => {
    if (!selectedTableId) return;
    const selected = tables.find((t) => t.tableId === selectedTableId);
    if (!selected) return;
    setLoading(true);
    try {
      await onSelectTable(selected.tableId, selected.tableName);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Paper
      elevation={0}
      sx={{
        p: 3,
        borderRadius: 3,
        bgcolor: '#0F1117',
        border: '1px solid #252D3D',
        color: '#F0F4FF',
      }}
    >
      <Typography variant="overline" sx={{ color: '#F5A623', fontWeight: 700, tracking: '0.08em' }}>
        STEP 1 OF 3: BINDING PANEL 1 — PRIMARY DRIVING TABLE
      </Typography>

      <Typography variant="h6" sx={{ fontWeight: 700, mt: 0.5, mb: 2 }}>
        Select Driving Table to Initialize Business Object Scope
      </Typography>

      <Stack spacing={2.5}>
        <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
          <Box>
            <Typography variant="caption" sx={{ color: '#94A3B8', mb: 0.5, display: 'block' }}>
              Datasource
            </Typography>
            <Box sx={{ p: 1.5, borderRadius: 1.5, bgcolor: '#161B25', border: '1px solid #252D3D', color: '#F0F4FF', fontSize: 13, fontFamily: 'monospace' }}>
              {datasourceName}
            </Box>
          </Box>

          <Box>
            <Typography variant="caption" sx={{ color: '#94A3B8', mb: 0.5, display: 'block' }}>
              Schema
            </Typography>
            <Box sx={{ p: 1.5, borderRadius: 1.5, bgcolor: '#161B25', border: '1px solid #252D3D', color: '#F0F4FF', fontSize: 13, fontFamily: 'monospace' }}>
              {schemaName}
            </Box>
          </Box>
        </Box>

        {/* AI Suggestions Rail */}
        <Box sx={{ p: 2, borderRadius: 2, bgcolor: 'rgba(245, 166, 35, 0.08)', border: '1px solid rgba(245, 166, 35, 0.3)' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
            <AIIcon sx={{ color: '#F5A623', fontSize: 18 }} />
            <Typography variant="subtitle2" sx={{ color: '#F5A623', fontWeight: 700 }}>
              AI Centrality Recommendations
            </Typography>
          </Box>
          <Typography variant="body2" sx={{ color: '#94A3B8', fontSize: 12 }}>
            Based on foreign key centrality, the following driving tables are primary candidates:
          </Typography>

          <Stack direction="row" spacing={1} sx={{ mt: 1.5, flexWrap: 'wrap', gap: 1 }}>
            {tables.filter((t) => t.isSuggested).map((t) => (
              <Chip
                key={t.tableId}
                icon={<TableIcon sx={{ fontSize: 14, color: '#F5A623 !important' }} />}
                label={`${t.tableName} (${t.fkCount} FKs)`}
                onClick={() => setSelectedTableId(t.tableId)}
                variant={selectedTableId === t.tableId ? 'filled' : 'outlined'}
                sx={{
                  bgcolor: selectedTableId === t.tableId ? '#F5A623' : 'transparent',
                  color: selectedTableId === t.tableId ? '#0A0C10' : '#F5A623',
                  borderColor: '#F5A623',
                  fontWeight: 700,
                  fontSize: 12,
                }}
              />
            ))}
          </Stack>
        </Box>

        {/* Table Selector Dropdown */}
        <Box>
          <Typography variant="caption" sx={{ color: '#94A3B8', mb: 0.5, display: 'block' }}>
            All Available Driving Tables
          </Typography>
          <Select
            fullWidth
            size="small"
            value={selectedTableId}
            onChange={(e) => setSelectedTableId(e.target.value)}
            displayEmpty
            sx={{
              bgcolor: '#161B25',
              color: '#F0F4FF',
              '.MuiOutlinedInput-notchedOutline': { borderColor: '#252D3D' },
              '&:hover .MuiOutlinedInput-notchedOutline': { borderColor: '#F5A623' },
            }}
          >
            <MenuItem value="" disabled>
              <em>Select a driving table...</em>
            </MenuItem>
            {tables.map((t) => (
              <MenuItem key={t.tableId} value={t.tableId} sx={{ fontFamily: 'monospace', fontSize: 13 }}>
                {t.tableName} ({t.fkCount} FK relationships)
              </MenuItem>
            ))}
          </Select>
        </Box>

        {/* Next Trigger */}
        <Box sx={{ display: 'flex', justifyContent: 'flex-end', pt: 1 }}>
          <Button
            variant="contained"
            disabled={!selectedTableId || loading}
            onClick={handleNext}
            endIcon={loading ? <CircularProgress size={16} color="inherit" /> : <ArrowIcon />}
            sx={{
              bgcolor: '#F5A623',
              color: '#0A0C10',
              fontWeight: 700,
              '&:hover': { bgcolor: '#D97706' },
              '&.Mui-disabled': { bgcolor: '#252D3D', color: '#4B5563' },
            }}
          >
            {loading ? 'Auto-Discovering...' : 'Next: Auto-Discover Terms'}
          </Button>
        </Box>
      </Stack>
    </Paper>
  );
};
