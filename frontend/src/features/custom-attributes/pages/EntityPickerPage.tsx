import React, { useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Card,
  CardActionArea,
  CardContent,
  Chip,
  CircularProgress,
  Typography,
} from '@mui/material';
import { useNavigate } from 'react-router-dom';
import { listEligibleEntities } from '../api';
import type { EligibleEntity } from '../types';

export default function EntityPickerPage() {
  const navigate = useNavigate();
  const [entities, setEntities] = useState<EligibleEntity[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await listEligibleEntities();
        if (!cancelled) setEntities(data);
      } catch (e: any) {
        if (!cancelled) setError(e?.message || 'Failed to load entities');
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h5" gutterBottom>
        Custom Fields
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
        Select a catalog table with a <code>custom_attributes</code> column to manage
        field definitions. Tenant values are read-only here and edited via Business Objects.
      </Typography>

      {loading && (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
          <CircularProgress />
        </Box>
      )}

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      {!loading && !error && entities.length === 0 && (
        <Alert severity="info">
          No eligible tables found. Run a catalog scan so tables with a{' '}
          <code>custom_attributes</code> column appear here. Definitions already seeded for
          PRODUCT will appear once the API can read <code>attribute_def</code>.
        </Alert>
      )}

      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))',
          gap: 2,
        }}
      >
        {entities.map((entity) => (
          <Card key={`${entity.entity_type}:${entity.table_ref}`} variant="outlined">
            <CardActionArea
              onClick={() =>
                navigate(
                  `/catalog/custom-fields/${encodeURIComponent(entity.entity_type)}?table_ref=${encodeURIComponent(entity.table_ref)}`,
                )
              }
            >
              <CardContent>
                <Typography variant="h6">{entity.display_name || entity.entity_type}</Typography>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                  {entity.table_ref}
                </Typography>
                <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                  <Chip size="small" label={entity.entity_type} />
                  <Chip size="small" label={`${entity.field_count} fields`} color="primary" variant="outlined" />
                  {entity.column_node_id ? (
                    <Chip size="small" label="catalog" color="success" variant="outlined" />
                  ) : (
                    <Chip size="small" label="defs only" variant="outlined" />
                  )}
                </Box>
              </CardContent>
            </CardActionArea>
          </Card>
        ))}
      </Box>
    </Box>
  );
}
