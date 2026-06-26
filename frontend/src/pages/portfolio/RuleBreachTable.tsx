import React, { useMemo } from 'react';
import { DataGrid, GridColDef } from '@mui/x-data-grid';
import {
  Paper,
  Box,
  Typography,
  Chip,
  Alert,
  Skeleton,
  useTheme,
} from '@mui/material';
import { useMaterialTheme } from '../../hooks/useMaterialTheme';

interface RuleBreach {
  rule_code: string;
  description?: string;
  metric_value: number;
  threshold_value: number;
}

interface RuleBreachTableProps {
  hard_breaches?: RuleBreach[];
  soft_breaches?: RuleBreach[];
  isLoading?: boolean;
  error?: Error | null;
}

const SeverityBadge: React.FC<{ severity: 'HARD' | 'SOFT' }> = ({
  severity,
}) => {
  return (
    <Chip
      label={severity}
      size="small"
      color={severity === 'HARD' ? 'error' : 'warning'}
      variant="filled"
      sx={{
        fontWeight: 700,
        textTransform: 'uppercase',
        fontSize: '0.7rem',
      }}
    />
  );
};

export const RuleBreachTable: React.FC<RuleBreachTableProps> = ({
  hard_breaches = [],
  soft_breaches = [],
  isLoading,
  error,
}) => {
  const theme = useTheme();
  const { textColor, borderColor, backgroundColor } = useMaterialTheme();

  const rows = useMemo(() => {
    const allBreaches = [
      ...hard_breaches.map((b, idx) => ({
        id: `hard-${idx}`,
        ...b,
        severity: 'HARD' as const,
      })),
      ...soft_breaches.map((b, idx) => ({
        id: `soft-${idx}`,
        ...b,
        severity: 'SOFT' as const,
      })),
    ];
    return allBreaches;
  }, [hard_breaches, soft_breaches]);

  const columns: GridColDef[] = [
    {
      field: 'rule_code',
      headerName: 'Rule Code',
      flex: 1,
      minWidth: 120,
      headerAlign: 'left',
      align: 'left',
      sortable: true,
      renderCell: (params) => (
        <Typography
          variant="body2"
          sx={{
            fontFamily: 'monospace',
            fontWeight: 700,
            color: textColor,
          }}
        >
          {params.value}
        </Typography>
      ),
    },
    {
      field: 'description',
      headerName: 'Description',
      flex: 1.5,
      minWidth: 180,
      sortable: true,
      renderCell: (params) => (
        <Typography variant="body2" sx={{ color: textColor }}>
          {params.value || '—'}
        </Typography>
      ),
    },
    {
      field: 'severity',
      headerName: 'Severity',
      width: 100,
      sortable: true,
      renderCell: (params) => (
        <SeverityBadge severity={params.value} />
      ),
    },
    {
      field: 'metric_value',
      headerName: 'Metric Value',
      width: 120,
      sortable: true,
      type: 'number',
      align: 'right',
      headerAlign: 'right',
      renderCell: (params) => (
        <Typography
          variant="body2"
          sx={{
            fontFamily: 'monospace',
            fontWeight: 700,
            color: textColor,
          }}
        >
          {parseFloat(params.value).toFixed(4)}
        </Typography>
      ),
    },
    {
      field: 'threshold_value',
      headerName: 'Threshold',
      width: 120,
      sortable: true,
      type: 'number',
      align: 'right',
      headerAlign: 'right',
      renderCell: (params) => (
        <Typography
          variant="body2"
          sx={{
            fontFamily: 'monospace',
            color: 'textSecondary',
          }}
        >
          {parseFloat(params.value).toFixed(4)}
        </Typography>
      ),
    },
    {
      field: 'breach_pct',
      headerName: 'Breach %',
      width: 110,
      sortable: false,
      align: 'right',
      headerAlign: 'right',
      renderCell: (params) => {
        const metric = params.row.metric_value;
        const threshold = params.row.threshold_value;
        const breachPct = ((metric / threshold) * 100 - 100).toFixed(1);
        const isPositive = parseFloat(breachPct) > 0;
        return (
          <Typography
            variant="body2"
            sx={{
              fontFamily: 'monospace',
              fontWeight: 700,
              color: isPositive ? 'error.main' : 'textSecondary',
            }}
          >
            {isPositive ? '+' : ''}
            {breachPct}%
          </Typography>
        );
      },
    },
  ];

  // Loading state
  if (isLoading) {
    return (
      <Paper
        elevation={1}
        sx={{
          backgroundColor,
          borderColor,
          border: 1,
          overflow: 'hidden',
        }}
      >
        <Box sx={{ p: 3 }}>
          <Skeleton variant="text" width="30%" height={32} sx={{ mb: 2 }} />
          <Skeleton variant="rectangular" height={400} />
        </Box>
      </Paper>
    );
  }

  // Error state
  if (error) {
    return (
      <Paper
        elevation={1}
        sx={{
          backgroundColor,
          borderColor,
          border: 1,
          p: 3,
        }}
      >
        <Alert
          severity="error"
          sx={{
            backgroundColor: 'error.light',
            color: 'error.dark',
            '& .MuiAlert-icon': { color: 'error.main' },
          }}
        >
          {error?.message || 'Failed to load compliance breaches'}
        </Alert>
      </Paper>
    );
  }

  // Empty state
  if (rows.length === 0) {
    return (
      <Paper
        elevation={1}
        sx={{
          backgroundColor,
          borderColor,
          border: 1,
          p: 3,
          textAlign: 'center',
        }}
      >
        <Typography variant="body2" color="success.main" sx={{ fontWeight: 600 }}>
          ✓ No compliance breaches detected
        </Typography>
      </Paper>
    );
  }

  return (
    <Paper
      elevation={1}
      sx={{
        backgroundColor,
        borderColor,
        border: 1,
        overflow: 'hidden',
      }}
    >
      <Box
        sx={{
          px: 3,
          py: 2,
          borderBottom: 1,
          borderColor: 'divider',
        }}
      >
        <Typography
          variant="h6"
          sx={{
            fontWeight: 700,
            color: textColor,
          }}
        >
          Rule Breaches ({rows.length})
        </Typography>
      </Box>

      <Box sx={{ height: 500, width: '100%' }}>
        <DataGrid
          rows={rows}
          columns={columns}
          pageSizeOptions={[5, 10, 25]}
          initialState={{
            pagination: { paginationModel: { pageSize: 5 } },
            sorting: {
              sortModel: [{ field: 'severity', sort: 'asc' }],
            },
          }}
          sx={{
            border: 'none',
            backgroundColor: 'transparent',
            '& .MuiDataGrid-cell': {
              borderBottomColor: borderColor,
              padding: '12px 8px',
            },
            '& .MuiDataGrid-columnHeaders': {
              backgroundColor: theme.palette.action.hover,
              borderBottomColor: borderColor,
              fontWeight: 600,
              fontSize: '12px',
              textTransform: 'uppercase',
              letterSpacing: 0.5,
              color: 'textSecondary',
            },
            '& .MuiDataGrid-row': {
              '&:hover': {
                backgroundColor: theme.palette.action.hover,
              },
            },
            '& .MuiDataGrid-row.Mui-selected': {
              backgroundColor: theme.palette.action.selected,
              '&:hover': {
                backgroundColor: theme.palette.action.selected,
              },
            },
            '& .MuiTablePagination-root': {
              color: textColor,
              borderTopColor: borderColor,
            },
            '& .MuiIconButton-root': {
              color: textColor,
            },
            '& .MuiSelect-icon': {
              color: textColor,
            },
          }}
          disableSelectionOnClick
          density="compact"
        />
      </Box>
    </Paper>
  );
};

export default RuleBreachTable;
