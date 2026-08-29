import React, { useEffect, useState } from 'react';
import { Box, Paper, Stack, Typography, TextField, Switch, MenuItem, Select, CircularProgress, Alert } from '@mui/material';
import { RelatedObjectGrid } from './RelatedObjectGrid';
import type { CanvasWidget, FieldWidget } from './pageStudioTypes';

export interface PageStudioPreviewProps {
  rootBoKey: string;
  recordId: string;
  canvas: CanvasWidget[];
}

function useRecord(boKey: string, recordId: string) {
  const [data, setData] = useState<Record<string, any> | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!boKey || !recordId) return;
    setData(null);
    setError(null);
    fetch(`/api/v1/bo/${encodeURIComponent(boKey)}/records/${encodeURIComponent(recordId)}`)
      .then(async (res) => {
        if (!res.ok) throw new Error(await res.text());
        return res.json();
      })
      .then(setData)
      .catch((err) => setError(err.message || 'Failed to load record'));
  }, [boKey, recordId]);

  return { data, error };
}

const liveControl = (widget: FieldWidget, value: any) => {
  switch (widget.controlType) {
    case 'switch':
      return <Switch size="small" checked={Boolean(value)} disabled />;
    case 'date':
      return <TextField size="small" type="date" value={value || ''} disabled sx={{ width: 160 }} />;
    case 'datetime':
      return <TextField size="small" type="datetime-local" value={value || ''} disabled sx={{ width: 200 }} />;
    case 'number':
      return <TextField size="small" type="number" value={value ?? ''} disabled sx={{ width: 140 }} />;
    case 'select':
      return (
        <Select size="small" disabled value={value || ''} displayEmpty sx={{ width: 180 }}>
          <MenuItem value={value || ''}>{value || '—'}</MenuItem>
        </Select>
      );
    default:
      return <TextField size="small" value={value ?? ''} disabled sx={{ width: 220 }} />;
  }
};

export const PageStudioPreview: React.FC<PageStudioPreviewProps> = ({ rootBoKey, recordId, canvas }) => {
  const { data, error } = useRecord(rootBoKey, recordId);

  const renderWidgets = (widgets: CanvasWidget[]): React.ReactNode =>
    widgets.map((widget) => {
      if (widget.type === 'field') {
        return (
          <Paper key={widget.id} variant="outlined" sx={{ p: 1, mb: 1 }}>
            <Stack>
              <Typography variant="caption" fontWeight={700}>{widget.label}</Typography>
              {liveControl(widget, data?.[widget.fieldKey])}
            </Stack>
          </Paper>
        );
      }
      if (widget.type === 'relatedObject') {
        return (
          <Box key={widget.id} sx={{ mb: 1.5 }}>
            <RelatedObjectGrid rootBoKey={rootBoKey} rootRecordId={recordId} widget={widget} />
          </Box>
        );
      }
      if (widget.type === 'grid' || widget.type === 'chart' || widget.type === 'kpi') {
        return (
          <Paper key={widget.id} variant="outlined" sx={{ p: 1.5, mb: 1 }}>
            <Typography variant="body2" color="text.secondary">
              {widget.title} — live rendering not yet implemented for {widget.type} widgets.
            </Typography>
          </Paper>
        );
      }
      // section / row
      return (
        <Paper key={widget.id} variant="outlined" sx={{ p: 1.5, mb: 1.5 }}>
          <Typography variant="subtitle2" fontWeight={700} sx={{ mb: 1 }}>{widget.title}</Typography>
          <Box sx={{ display: 'flex', flexDirection: (widget as any).flow === 'row' || widget.type === 'row' ? 'row' : 'column', gap: 1, flexWrap: 'wrap' }}>
            {renderWidgets(widget.children)}
          </Box>
        </Paper>
      );
    });

  if (error) return <Alert severity="error">{error}</Alert>;
  if (!data) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
        <CircularProgress size={24} />
      </Box>
    );
  }

  return <Box sx={{ p: 2 }}>{renderWidgets(canvas)}</Box>;
};

export default PageStudioPreview;
