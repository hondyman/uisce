import React, { useCallback, useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  FormControl,
  InputLabel,
  Link as MuiLink,
  MenuItem,
  Paper,
  Select,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import SpeedIcon from '@mui/icons-material/Speed';
import { Link as RouterLink } from 'react-router-dom';
import { useLocale } from '../../../i18n/useLocale';
import { listCubes } from '../cubeDefinitionApi';
import type { CubeDefinition, CubeScope } from '../types';

function formatWhen(iso: string): string {
  try {
    return new Date(iso).toLocaleString();
  } catch {
    return iso;
  }
}

const CubesCatalogPage: React.FC = () => {
  const locale = useLocale();
  const [scope, setScope] = useState<CubeScope>('all');
  const [cubes, setCubes] = useState<CubeDefinition[]>([]);
  const [nextCursor, setNextCursor] = useState<string | undefined>();
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async (opts?: { cursor?: string; append?: boolean }) => {
    if (opts?.append) setLoadingMore(true);
    else setLoading(true);
    setError(null);
    try {
      const res = await listCubes({
        scope,
        limit: 50,
        cursor: opts?.cursor,
      });
      setCubes((prev) => (opts?.append ? [...prev, ...res.cubes] : res.cubes));
      setNextCursor(res.nextCursor);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      if (!opts?.append) setCubes([]);
    } finally {
      setLoading(false);
      setLoadingMore(false);
    }
  }, [scope]);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <Box sx={{ p: 3, maxWidth: 1200, mx: 'auto' }}>
      <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between" alignItems={{ sm: 'center' }} spacing={2} mb={3}>
        <Box>
          <Typography variant="h4" component="h1" gutterBottom>
            Cubes
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Published aggregation contracts — dimensions, governed metrics, grains, and materialization plan.
          </Typography>
        </Box>
        <Stack direction="row" spacing={1}>
          <Button
            component={RouterLink}
            to={`/${locale}/fabric/preaggregations`}
            startIcon={<SpeedIcon />}
            variant="outlined"
          >
            Preaggregations
          </Button>
          <Button
            component={RouterLink}
            to={`/${locale}/build/cubes/new`}
            startIcon={<AddIcon />}
            variant="contained"
          >
            New Cube
          </Button>
        </Stack>
      </Stack>

      <Stack direction="row" spacing={2} mb={2} alignItems="center">
        <FormControl size="small" sx={{ minWidth: 160 }}>
          <InputLabel id="cube-scope-label">Scope</InputLabel>
          <Select
            labelId="cube-scope-label"
            label="Scope"
            value={scope}
            onChange={(e) => setScope(e.target.value as CubeScope)}
          >
            <MenuItem value="all">All</MenuItem>
            <MenuItem value="core">Core</MenuItem>
            <MenuItem value="custom">Custom</MenuItem>
            <MenuItem value="adopted">Adopted</MenuItem>
          </Select>
        </FormControl>
      </Stack>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Paper variant="outlined">
        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
            <CircularProgress size={32} />
          </Box>
        ) : cubes.length === 0 ? (
          <Box sx={{ py: 6, textAlign: 'center' }}>
            <Typography color="text.secondary" gutterBottom>
              No cubes in this scope yet.
            </Typography>
            <Button component={RouterLink} to={`/${locale}/build/cubes/new`} variant="contained" sx={{ mt: 1 }}>
              Create the first cube
            </Button>
          </Box>
        ) : (
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Name</TableCell>
                <TableCell>Business Object</TableCell>
                <TableCell>Version</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Updated</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {cubes.map((c) => (
                <TableRow key={c.id} hover>
                  <TableCell>
                    <MuiLink
                      component={RouterLink}
                      to={`/${locale}/build/cubes/${c.id}`}
                      underline="hover"
                      fontWeight={600}
                    >
                      {c.name}
                    </MuiLink>
                    {c.isCore && (
                      <Chip label="Core" size="small" color="secondary" sx={{ ml: 1 }} />
                    )}
                  </TableCell>
                  <TableCell>{c.boId}</TableCell>
                  <TableCell>v{c.contractVersion}</TableCell>
                  <TableCell>
                    <Chip
                      label={c.status}
                      size="small"
                      color={c.status === 'active' ? 'success' : 'default'}
                      variant="outlined"
                    />
                  </TableCell>
                  <TableCell>{formatWhen(c.updatedAt)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Paper>

      {nextCursor && (
        <Box sx={{ mt: 2, textAlign: 'center' }}>
          <Button
            disabled={loadingMore}
            onClick={() => void load({ cursor: nextCursor, append: true })}
          >
            {loadingMore ? 'Loading…' : 'Load more'}
          </Button>
        </Box>
      )}
    </Box>
  );
};

export default CubesCatalogPage;
