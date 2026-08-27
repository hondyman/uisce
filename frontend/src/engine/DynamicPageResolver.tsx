import React, { useState, useEffect } from 'react';
import { DynamicBODataGrid, FieldMeta } from './DynamicBODataGrid';
import { CircularProgress, Box, Typography, Paper, Alert } from '@mui/material';
import { Error as ErrorIcon } from '@mui/icons-material';

export interface PageLayoutBlueprint {
  page_key: string;
  page_name: string;
  bo_key: string;
  layout_type: 'GRID' | 'FORM' | 'SPLIT_MDM_STUDIO';
  fields: FieldMeta[];
}

export const DynamicPageResolver: React.FC<{ pageKey: string; tenantId: string }> = ({ pageKey, tenantId }) => {
  const [layout, setLayout] = useState<PageLayoutBlueprint | null>(null);
  const [data, setData] = useState<any[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadDynamicPage();
  }, [pageKey, tenantId]);

  const loadDynamicPage = async () => {
    setLoading(true);
    setError(null);
    try {
      const layoutRes = await fetch(`/api/v1/layout/resolve?pageKey=${pageKey}`, {
        headers: { 'X-Tenant-ID': tenantId },
      });
      if (!layoutRes.ok) throw new Error('Failed to resolve page metadata');
      const layoutData: PageLayoutBlueprint = await layoutRes.json();
      setLayout(layoutData);

      const dataRes = await fetch(`/api/v1/bo/data/${layoutData.bo_key}`, {
        headers: { 'X-Tenant-ID': tenantId },
      });
      const records = await dataRes.json();
      setData(records.data || []);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          height: 256,
          color: 'grey.400',
          gap: 1,
        }}
      >
        <CircularProgress size={20} /> Resolving Metadata Layout...
      </Box>
    );
  }

  if (error || !layout) {
    return (
      <Alert
        severity="error"
        icon={<ErrorIcon />}
        sx={{
          p: 2,
          backgroundColor: 'rgba(127, 29, 29, 0.3)',
          border: '1px solid rgba(239, 68, 68, 0.3)',
          borderRadius: '12px',
          color: 'rgb(252, 165, 165)',
          '& .MuiAlert-icon': {
            color: 'inherit',
          },
        }}
      >
        Failed loading dynamic layout: {error}
      </Alert>
    );
  }

  return (
    <Box
      sx={{
        p: 3,
        backgroundColor: '#0f172a',
        minHeight: '100vh',
        color: '#f1f5f9',
      }}
    >
      <Box
        sx={{
          mb: 3,
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
        <Box>
          <Typography
            component="span"
            sx={{
              fontSize: '0.75rem',
              fontFamily: 'monospace',
              backgroundColor: 'rgba(3, 105, 161, 0.5)',
              color: '#38bdf8',
              px: 1,
              py: 0.25,
              borderRadius: '4px',
              textTransform: 'uppercase',
              letterSpacing: '0.05em',
            }}
          >
            BO: {layout.bo_key}
          </Typography>
          <Typography variant="h4" sx={{ fontWeight: 700, color: 'white', mt: 0.5 }}>
            {layout.page_name}
          </Typography>
        </Box>
        <Typography variant="body2" sx={{ fontFamily: 'monospace', color: 'grey.400' }}>
          Dynamic Attributes Rendered: {layout.fields.length}
        </Typography>
      </Box>

      <DynamicBODataGrid fields={layout.fields} data={data} />
    </Box>
  );
};
