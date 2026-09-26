import React, { useState } from 'react';
import {
  Box,
  Chip,
  Drawer,
  Link,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
  Alert,
} from '@mui/material';
import type { PreviewResponse } from '../types';

interface DataPreviewGridProps {
  preview: PreviewResponse | null;
  loading?: boolean;
  error?: string | null;
  highlightedField?: string | null;
  businessObjectPath?: string;
}

function formatCell(value: unknown): string {
  if (value === null || value === undefined || value === '') return '—';
  if (typeof value === 'object') return JSON.stringify(value);
  return String(value);
}

export function DataPreviewGrid({
  preview,
  loading,
  error,
  highlightedField,
  businessObjectPath,
}: DataPreviewGridProps) {
  const [rowJSON, setRowJSON] = useState<Record<string, unknown> | null>(null);

  if (error) {
    return <Alert severity="warning" sx={{ m: 2 }}>{error}</Alert>;
  }

  if (loading) {
    return (
      <Typography sx={{ p: 2 }} color="text.secondary">
        Loading preview…
      </Typography>
    );
  }

  if (!preview) {
    return (
      <Typography sx={{ p: 2 }} color="text.secondary">
        Select an entity to preview tenant values.
      </Typography>
    );
  }

  const customCols = preview.columns.filter((c) => c.source !== 'CORE');
  const coreCols = preview.columns
    .filter((c) => c.source === 'CORE')
    .filter((c) => ['id', 'product_cd', 'name', 'status_id', 'is_active'].includes(c.key))
    .slice(0, 4);
  const visible = [...coreCols, ...customCols];

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Box sx={{ px: 2, py: 1.5, display: 'flex', alignItems: 'center', gap: 2 }}>
        <Typography variant="subtitle2">Data preview</Typography>
        <Chip size="small" label={`Rows ${preview.showing} of ${preview.total}`} />
        <Typography variant="caption" color="text.secondary">
          Read-only · values are edited via Business Object
        </Typography>
        {businessObjectPath && (
          <Link href={businessObjectPath} underline="hover" sx={{ ml: 'auto' }}>
            Edit values on Business Object
          </Link>
        )}
      </Box>
      <TableContainer component={Paper} variant="outlined" sx={{ flex: 1, mx: 2, mb: 2 }}>
        <Table size="small" stickyHeader>
          <TableHead>
            <TableRow>
              {visible.map((col) => (
                <TableCell
                  key={col.key}
                  sx={{
                    fontWeight: 600,
                    bgcolor:
                      highlightedField && col.field_cd === highlightedField
                        ? 'warning.light'
                        : undefined,
                  }}
                >
                  {col.label}
                  {col.source !== 'CORE' && (
                    <Chip size="small" label={col.source} sx={{ ml: 0.5 }} />
                  )}
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {preview.rows.map((row, idx) => (
              <TableRow
                key={String(row.id ?? idx)}
                hover
                sx={{ cursor: 'pointer' }}
                onClick={() => setRowJSON(row)}
              >
                {visible.map((col) => (
                  <TableCell
                    key={col.key}
                    sx={{
                      bgcolor:
                        highlightedField && col.field_cd === highlightedField
                          ? 'warning.light'
                          : undefined,
                      color:
                        row[col.key] === null || row[col.key] === undefined
                          ? 'text.disabled'
                          : undefined,
                    }}
                  >
                    {formatCell(row[col.key])}
                  </TableCell>
                ))}
              </TableRow>
            ))}
            {preview.rows.length === 0 && (
              <TableRow>
                <TableCell colSpan={Math.max(visible.length, 1)}>
                  <Typography color="text.secondary">No rows for this tenant.</Typography>
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>

      <Drawer anchor="right" open={!!rowJSON} onClose={() => setRowJSON(null)}>
        <Box sx={{ width: 420, p: 2 }}>
          <Typography variant="h6" gutterBottom>
            Row JSON (read-only)
          </Typography>
          <Box
            component="pre"
            sx={{
              bgcolor: 'grey.100',
              p: 1.5,
              borderRadius: 1,
              overflow: 'auto',
              fontSize: 12,
            }}
          >
            {JSON.stringify(rowJSON, null, 2)}
          </Box>
        </Box>
      </Drawer>
    </Box>
  );
}
