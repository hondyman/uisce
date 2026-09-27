import React, { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  AppBar,
  Avatar,
  Box,
  Card,
  CardActionArea,
  CardContent,
  Chip,
  CircularProgress,
  Container,
  Grid,
  IconButton,
  InputAdornment,
  Paper,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Toolbar,
  Tooltip,
  Typography,
  useTheme,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import SearchIcon from '@mui/icons-material/Search';
import StorageIcon from '@mui/icons-material/Storage';
import SchemaIcon from '@mui/icons-material/Schema';
import ViewModuleIcon from '@mui/icons-material/ViewModule';
import ViewListIcon from '@mui/icons-material/ViewList';
import { useNavigate } from 'react-router-dom';
import { CustomIcon } from '../../../components/common/CoreCustomIcons';
import { listEligibleEntities } from '../api';
import type { EligibleEntity } from '../types';

export default function EntityPickerPage() {
  const navigate = useNavigate();
  const theme = useTheme();
  const [entities, setEntities] = useState<EligibleEntity[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [schemaFilter, setSchemaFilter] = useState<string>('all');
  const [viewMode, setViewMode] = useState<'card' | 'list'>('card');
  const [definedOnly, setDefinedOnly] = useState(false);

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await listEligibleEntities();
      setEntities(data);
    } catch (e: any) {
      setError(e?.message || 'Failed to load entities');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const schemas = useMemo(() => {
    const set = new Set(entities.map((e) => e.schema_name).filter(Boolean));
    return Array.from(set).sort();
  }, [entities]);

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    return entities.filter((e) => {
      if (schemaFilter !== 'all' && e.schema_name !== schemaFilter) return false;
      if (definedOnly && !(e.field_count > 0)) return false;
      if (!q) return true;
      return (
        e.display_name.toLowerCase().includes(q) ||
        e.table_ref.toLowerCase().includes(q) ||
        e.entity_type.toLowerCase().includes(q) ||
        e.schema_name.toLowerCase().includes(q)
      );
    });
  }, [entities, search, schemaFilter, definedOnly]);

  const openEntity = (entity: EligibleEntity) => {
    navigate(
      `/catalog/custom-fields/${encodeURIComponent(entity.entity_type)}?table_ref=${encodeURIComponent(entity.table_ref)}`,
    );
  };

  return (
    <Box sx={{ minHeight: '100%', bgcolor: 'background.default' }}>
      <AppBar position="sticky" color="default" elevation={0} sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Toolbar sx={{ gap: 2, flexWrap: 'wrap', py: 1 }}>
          <SchemaIcon color="primary" />
          <Box sx={{ flex: 1, minWidth: 200 }}>
            <Typography variant="h6" fontWeight={700}>
              Manage Custom Fields
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Tables with a <code>custom_attributes</code> column · tile or list view
            </Typography>
          </Box>
          <TextField
            size="small"
            placeholder="Search tables…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            sx={{ minWidth: 260 }}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon fontSize="small" />
                </InputAdornment>
              ),
            }}
          />
          <Paper
            variant="outlined"
            sx={{ display: 'flex', alignItems: 'center', borderRadius: 2, overflow: 'hidden' }}
          >
            <Tooltip title="Tile view">
              <IconButton
                size="small"
                color={viewMode === 'card' ? 'primary' : 'default'}
                onClick={() => setViewMode('card')}
                sx={{ borderRadius: 0 }}
              >
                <ViewModuleIcon fontSize="small" />
              </IconButton>
            </Tooltip>
            <Tooltip title="List view">
              <IconButton
                size="small"
                color={viewMode === 'list' ? 'primary' : 'default'}
                onClick={() => setViewMode('list')}
                sx={{ borderRadius: 0 }}
              >
                <ViewListIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          </Paper>
          <Tooltip title="Refresh">
            <IconButton onClick={load} disabled={loading}>
              <RefreshIcon />
            </IconButton>
          </Tooltip>
        </Toolbar>
        <Box sx={{ px: 3, pb: 1.5, display: 'flex', gap: 1, flexWrap: 'wrap', alignItems: 'center' }}>
          <Chip
            label={`All (${entities.length})`}
            size="small"
            color={schemaFilter === 'all' ? 'primary' : 'default'}
            onClick={() => setSchemaFilter('all')}
            variant={schemaFilter === 'all' ? 'filled' : 'outlined'}
          />
          {schemas.map((schema) => {
            const count = entities.filter((e) => e.schema_name === schema).length;
            return (
              <Chip
                key={schema}
                label={`${schema} (${count})`}
                size="small"
                color={schemaFilter === schema ? 'primary' : 'default'}
                onClick={() => setSchemaFilter(schema)}
                variant={schemaFilter === schema ? 'filled' : 'outlined'}
              />
            );
          })}
          <Chip
            icon={<CustomIcon fontSize="small" />}
            label="Has defined fields"
            size="small"
            color={definedOnly ? 'secondary' : 'default'}
            onClick={() => setDefinedOnly((v) => !v)}
            variant={definedOnly ? 'filled' : 'outlined'}
            sx={{ ml: { md: 'auto' } }}
          />
          <Typography variant="caption" color="text.secondary">
            Showing {filtered.length}
          </Typography>
        </Box>
      </AppBar>

      <Container maxWidth="xl" sx={{ py: 3 }}>
        {loading && (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}>
            <CircularProgress />
          </Box>
        )}

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {!loading && !error && filtered.length === 0 && (
          <Alert severity="info">
            {search || schemaFilter !== 'all' || definedOnly
              ? 'No tables match your filters.'
              : 'No tables with a custom_attributes column were found.'}
          </Alert>
        )}

        {!loading && filtered.length > 0 && viewMode === 'card' && (
          <Grid container spacing={3}>
            {filtered.map((entity) => {
              const hasCustomDefs = entity.field_count > 0;
              return (
                <Grid key={`${entity.entity_type}:${entity.table_ref}`} size={{ xs: 12, sm: 6, md: 4, lg: 3 }}>
                  <Card
                    variant="outlined"
                    sx={{
                      height: '100%',
                      transition: 'all 0.25s ease',
                      borderColor: hasCustomDefs ? 'secondary.light' : 'divider',
                      '&:hover': {
                        transform: 'translateY(-4px)',
                        boxShadow: theme.shadows[6],
                        borderColor: 'primary.main',
                      },
                    }}
                  >
                    <CardActionArea sx={{ height: '100%', alignItems: 'stretch' }} onClick={() => openEntity(entity)}>
                      <CardContent>
                        <Stack direction="row" spacing={1.5} alignItems="flex-start" sx={{ mb: 1.5 }}>
                          <Avatar
                            sx={{
                              bgcolor: hasCustomDefs ? 'secondary.main' : 'action.hover',
                              color: hasCustomDefs ? 'secondary.contrastText' : 'primary.main',
                              width: 44,
                              height: 44,
                            }}
                          >
                            {hasCustomDefs ? <CustomIcon fontSize="medium" sx={{ color: 'inherit' }} /> : <StorageIcon />}
                          </Avatar>
                          <Box sx={{ minWidth: 0, flex: 1 }}>
                            <Stack direction="row" spacing={0.75} alignItems="center">
                              <Typography variant="h6" sx={{ fontWeight: 700, fontSize: '1.05rem', lineHeight: 1.3 }}>
                                {entity.display_name || entity.entity_type}
                              </Typography>
                              {hasCustomDefs && <CustomIcon fontSize="small" />}
                            </Stack>
                            <Typography
                              variant="caption"
                              color="text.secondary"
                              sx={{ display: 'block', fontFamily: 'monospace' }}
                            >
                              {entity.table_ref}
                            </Typography>
                          </Box>
                        </Stack>

                        <Stack direction="row" spacing={0.5} flexWrap="wrap" useFlexGap sx={{ mb: 1 }}>
                          <Chip size="small" label={entity.schema_name || 'schema'} variant="outlined" />
                          <Chip size="small" label={entity.entity_type} color="primary" variant="outlined" />
                          <Chip
                            size="small"
                            icon={hasCustomDefs ? <CustomIcon fontSize="small" /> : undefined}
                            label={`${entity.field_count} field${entity.field_count === 1 ? '' : 's'}`}
                            color={hasCustomDefs ? 'secondary' : 'default'}
                            variant={hasCustomDefs ? 'filled' : 'outlined'}
                          />
                          {entity.column_node_id ? (
                            <Chip size="small" label="In catalog" color="info" variant="outlined" />
                          ) : (
                            <Chip size="small" label="Schema only" variant="outlined" />
                          )}
                        </Stack>

                        <Typography variant="body2" color="text.secondary">
                          {hasCustomDefs
                            ? 'Has custom field definitions — open to manage schema and preview values.'
                            : 'No definitions yet. Open to add custom attributes for this table.'}
                        </Typography>
                      </CardContent>
                    </CardActionArea>
                  </Card>
                </Grid>
              );
            })}
          </Grid>
        )}

        {!loading && filtered.length > 0 && viewMode === 'list' && (
          <TableContainer component={Paper} variant="outlined">
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Table</TableCell>
                  <TableCell>Schema</TableCell>
                  <TableCell>Entity</TableCell>
                  <TableCell align="right">Fields</TableCell>
                  <TableCell>Catalog</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {filtered.map((entity) => {
                  const hasCustomDefs = entity.field_count > 0;
                  return (
                    <TableRow
                      key={`${entity.entity_type}:${entity.table_ref}`}
                      hover
                      sx={{ cursor: 'pointer' }}
                      onClick={() => openEntity(entity)}
                    >
                      <TableCell>
                        <Stack direction="row" spacing={1} alignItems="center">
                          {hasCustomDefs ? <CustomIcon fontSize="small" /> : <StorageIcon fontSize="small" color="action" />}
                          <Box>
                            <Typography variant="body2" fontWeight={600}>
                              {entity.display_name}
                            </Typography>
                            <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                              {entity.table_ref}
                            </Typography>
                          </Box>
                        </Stack>
                      </TableCell>
                      <TableCell>{entity.schema_name}</TableCell>
                      <TableCell>{entity.entity_type}</TableCell>
                      <TableCell align="right">
                        <Chip
                          size="small"
                          label={entity.field_count}
                          color={hasCustomDefs ? 'secondary' : 'default'}
                          variant={hasCustomDefs ? 'filled' : 'outlined'}
                        />
                      </TableCell>
                      <TableCell>{entity.column_node_id ? 'Yes' : 'Schema only'}</TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Container>
    </Box>
  );
}
