import React, { useState, useMemo, useEffect, lazy, Suspense } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import type { LazyExoticComponent, ComponentType } from 'react';
import {
  Box,
  Container,
  Grid,
  Card,
  CardContent,
  Typography,
  TextField,
  InputAdornment,
  Chip,
  IconButton,
  Breadcrumbs,
  Link as MuiLink,
  Skeleton,
  Alert,
  Button,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Paper,
  Tooltip,
  ToggleButtonGroup,
  ToggleButton,
} from '@mui/material';
import {
  Search as SearchIcon,
  TableChart as TableIcon,
  ViewColumn as ColumnIcon,
  Key as KeyIcon,
  Home as HomeIcon,
  Storage as StorageIcon,
  Refresh as RefreshIcon,
  AccountTree as LineageIcon,
  ViewList as ListIcon,
} from '@mui/icons-material';
import { useQuery } from '@apollo/client';
import { useTenant } from '../../../contexts/TenantContext';
import { GET_SCOPED_TENANT } from '../../../graphql/queries/tenantQueries';
import { GET_SCHEMA_TABLES } from '../../../graphql/queries/datasourceQueries';
import ColumnDetailsModal from '../../../components/ColumnDetailsModal';
import useBlockableNavigate from '../../../components/RouteBlocker/useBlockableNavigate';

// TabbedModal props
interface TabbedModalProps {
  datasourceId: string;
  tenantId?: string;
  onClose: () => void;
  isModal?: boolean;
}

// Lazy load TabbedModal with error handling
const TabbedModal = lazy(async () => {
  try {
    console.log('🔄 SchemaExplorer: Starting lazy load of TabbedModal...');
    const mod = await import('../../../pages/TabbedModal/TabbedModal');
    console.log('✅ SchemaExplorer: TabbedModal module loaded successfully', { hasDefault: !!mod.default });
    const component = (mod as { default?: ComponentType<TabbedModalProps> }).default ?? (mod as unknown as ComponentType<TabbedModalProps>);
    if (!component) {
      console.error('❌ SchemaExplorer: TabbedModal component is null or undefined!');
      throw new Error('TabbedModal component not found in module');
    }
    return { default: component } as { default: ComponentType<TabbedModalProps> };
  } catch (error) {
    console.error('❌ SchemaExplorer: Failed to load TabbedModal:', error);
    throw error;
  }
}) as LazyExoticComponent<ComponentType<TabbedModalProps>>;

interface TableMetadata {
  name: string;
  schema: string;
  columnCount: number;
  primaryKeys: number;
  foreignKeys: number;
  columns: any[];
}

const SchemaExplorerPage: React.FC = () => {
  console.log('🚀 SchemaExplorerPage: Component mounting...');
  
  const { datasourceId: urlDatasourceId } = useParams<{ datasourceId: string }>();
  const [searchParams] = useSearchParams();
  const queryDatasourceId = searchParams.get('datasource');
  const navigate = useBlockableNavigate();
  
  const { tenant: scopedTenant, datasource: contextDatasource } = useTenant();
  
  console.log('📊 SchemaExplorerPage: State initialized', {
    urlDatasourceId,
    queryDatasourceId,
    contextDatasourceId: contextDatasource?.id,
    hasScopedTenant: !!scopedTenant
  });
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedSchema, setSelectedSchema] = useState<string>('all');
  const [selectedTable, setSelectedTable] = useState<TableMetadata | null>(null);
  const [columnModalOpen, setColumnModalOpen] = useState(false);
  const [viewMode, setViewMode] = useState<'list' | 'lineage'>('lineage'); // Default to lineage view

  // Determine effective datasource ID
  let effectiveDatasourceId = urlDatasourceId || queryDatasourceId || contextDatasource?.id;

  // Note: Datasource ID 982aef38-418f-46dc-acd0-35fe8f3b97b0 is correct and contains Northwinds catalog data
  // Do not auto-correct this ID as it would break schema exploration

  const { loading, error, data, refetch } = useQuery(GET_SCOPED_TENANT, {
    variables: { tenantId: scopedTenant?.id ?? '' },
    skip: !scopedTenant,
  });

  // Fetch real table metadata from catalog
  const { loading: tablesLoading, data: tablesData, refetch: refetchTables } = useQuery(GET_SCHEMA_TABLES, {
    variables: { datasourceId: effectiveDatasourceId },
    skip: !effectiveDatasourceId,
  });

  // Parse catalog nodes into table metadata
  const realTables: TableMetadata[] = useMemo(() => {
    console.log('Schema Explorer - tablesData:', tablesData);
    console.log('Schema Explorer - effectiveDatasourceId:', effectiveDatasourceId);
    
    if (!tablesData?.tables || !tablesData?.columns) {
      console.log('Schema Explorer - No tables or columns data');
      return [];
    }
    
    console.log('Schema Explorer - Tables count:', tablesData.tables.length);
    console.log('Schema Explorer - Columns count:', tablesData.columns.length);
    
    return tablesData.tables.map((table: any) => {
      const tableColumns = tablesData.columns.filter((col: any) => col.parent_id === table.id);
      
      // Parse schema from qualified_path (format: schema.table)
      const pathParts = table.qualified_path?.split('.') || [];
      const schema = pathParts.length > 1 ? pathParts[0] : 'public';
      
      // Count keys from column properties
      const columnData = tableColumns.map((col: any) => {
        const props = col.properties || {};
        return {
          name: col.node_name,
          type: props.data_type || 'unknown',
          nullable: props.is_nullable !== false,
          isPrimaryKey: props.is_primary_key === true,
          isForeignKey: props.is_foreign_key === true,
          isCore: props.is_core === true,
        };
      });
      
      return {
        name: table.node_name,
        schema,
        columnCount: tableColumns.length,
        primaryKeys: columnData.filter((c: any) => c.isPrimaryKey).length,
        foreignKeys: columnData.filter((c: any) => c.isForeignKey).length,
        columns: columnData,
      };
    });
  }, [tablesData]);

  const schemas = useMemo(() => {
    const schemaSet = new Set(realTables.map(t => t.schema));
    return ['all', ...Array.from(schemaSet)];
  }, [realTables]);

  const filteredTables = useMemo(() => {
    let filtered = realTables;
    
    if (selectedSchema !== 'all') {
      filtered = filtered.filter(t => t.schema === selectedSchema);
    }
    
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(t => 
        t.name.toLowerCase().includes(query) ||
        t.schema.toLowerCase().includes(query)
      );
    }
    
    return filtered;
  }, [realTables, selectedSchema, searchQuery]);

  const handleTableClick = (table: TableMetadata) => {
    setSelectedTable(table);
    setColumnModalOpen(true);
  };

  const handleClose = () => {
    navigate(-1);
  };

  if (!scopedTenant) {
    return (
      <Container maxWidth="md" sx={{ py: 8 }}>
        <Alert severity="info">
          Please select a tenant scope to browse database schemas.
        </Alert>
      </Container>
    );
  }

  if (!effectiveDatasourceId) {
    return (
      <Container maxWidth="md" sx={{ py: 8 }}>
        <Alert severity="warning">
          No datasource selected. Please select a datasource from the Tenants page.
        </Alert>
        <Button 
          variant="contained" 
          sx={{ mt: 2 }}
          onClick={() => navigate('/')}
        >
          Go to Tenants
        </Button>
      </Container>
    );
  }

  // Show query error if present
  if (error || (tablesData && !tablesLoading && realTables.length === 0 && tablesData.tables && tablesData.tables.length === 0)) {
    console.error('Schema Explorer - Query error or no data:', error);
  }

  // Guard against missing tenant
  if (!scopedTenant) {
    return (
      <Box sx={{ display: 'flex', flexDirection: 'column', height: '100vh', bgcolor: 'grey.50', p: 3, alignItems: 'center', justifyContent: 'center' }}>
        <Typography variant="h6" color="error" gutterBottom>
          Tenant Selection Required
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 3, maxWidth: 400, textAlign: 'center' }}>
          To view the schema explorer, please first select a tenant from the Tenants page. Then navigate to the datasource and access the Schema Explorer from there.
        </Typography>
        <Button variant="contained" onClick={() => navigate('/')} sx={{ mt: 2 }}>
          Go to Tenants Page
        </Button>
      </Box>
    );
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100vh', bgcolor: 'grey.50' }}>
      {/* Header */}
      <Paper elevation={1} sx={{ px: 3, py: 2, borderRadius: 0 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
          <Breadcrumbs>
            <MuiLink 
              component="button"
              variant="body2"
              onClick={() => navigate('/')}
              sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}
            >
              <HomeIcon fontSize="small" />
              Tenants
            </MuiLink>
            <Typography color="text.primary" sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
              <StorageIcon fontSize="small" />
              Schema Explorer
            </Typography>
          </Breadcrumbs>
          <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
            <ToggleButtonGroup
              value={viewMode}
              exclusive
              onChange={(_, newMode) => newMode && setViewMode(newMode)}
              size="small"
            >
              <ToggleButton value="lineage">
                <Tooltip title="Lineage & Catalog View">
                  <LineageIcon fontSize="small" />
                </Tooltip>
              </ToggleButton>
              <ToggleButton value="list">
                <Tooltip title="Table List View">
                  <ListIcon fontSize="small" />
                </Tooltip>
              </ToggleButton>
            </ToggleButtonGroup>
            <Tooltip title="Refresh schema">
              <IconButton onClick={() => { refetch(); refetchTables(); }} size="small">
                <RefreshIcon />
              </IconButton>
            </Tooltip>
          </Box>
        </Box>
        <Typography variant="h5" fontWeight={600}>
          Database Schema Explorer
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {scopedTenant?.display_name || scopedTenant?.name || 'Datasource'}
        </Typography>
      </Paper>

      {/* Content - Toggle between views */}
      {viewMode === 'lineage' ? (
        // Lineage/Catalog View (TabbedModal)
        <Box sx={{ flex: 1, overflow: 'hidden' }}>
          <Suspense fallback={
            <Box sx={{ 
              display: 'flex', 
              flexDirection: 'column',
              justifyContent: 'center', 
              alignItems: 'center', 
              height: '100%',
              gap: 2,
              bgcolor: 'background.default'
            }}>
              <Typography variant="h6" color="text.secondary">Loading Schema Explorer...</Typography>
              <Typography variant="body2" color="text.secondary">
                Initializing database catalog and lineage view
              </Typography>
            </Box>
          }>
            {(() => {
              console.log('🎯 SchemaExplorerPage: Rendering TabbedModal with datasourceId:', effectiveDatasourceId);
              try {
                return <TabbedModal datasourceId={effectiveDatasourceId} tenantId={scopedTenant?.id || 'default'} onClose={handleClose} isModal={false} />;
              } catch (error) {
                console.error('❌ SchemaExplorerPage: Error rendering TabbedModal:', error);
                return (
                  <Box sx={{ p: 3 }}>
                    <Alert severity="error">
                      <Typography variant="h6">Failed to load Schema Explorer</Typography>
                      <Typography variant="body2">Error: {String(error)}</Typography>
                    </Alert>
                  </Box>
                );
              }
            })()}
          </Suspense>
        </Box>
      ) : (
        // Table List View
        <Box sx={{ display: 'flex', flex: 1, overflow: 'hidden' }}>
          {/* Sidebar */}
          <Paper 
            elevation={0} 
            sx={{ 
              width: 280, 
              borderRadius: 0, 
              borderRight: 1, 
              borderColor: 'divider',
              display: 'flex',
              flexDirection: 'column'
            }}
          >
            <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
              <Typography variant="subtitle2" fontWeight={600} gutterBottom>
                Schemas
              </Typography>
              <Typography variant="caption" color="text.secondary">
                {schemas.length - 1} schema{schemas.length !== 2 ? 's' : ''}
              </Typography>
            </Box>
            <List sx={{ flex: 1, overflow: 'auto', py: 0 }}>
              {schemas.map((schema) => (
                <ListItem key={schema} disablePadding>
                  <ListItemButton
                    selected={selectedSchema === schema}
                    onClick={() => setSelectedSchema(schema)}
                    sx={{
                      '&.Mui-selected': {
                        bgcolor: 'primary.light',
                        '&:hover': {
                          bgcolor: 'primary.light',
                        },
                      },
                    }}
                  >
                    <ListItemIcon sx={{ minWidth: 36 }}>
                      <StorageIcon fontSize="small" color={selectedSchema === schema ? 'primary' : 'action'} />
                    </ListItemIcon>
                    <ListItemText 
                      primary={schema === 'all' ? 'All Schemas' : schema}
                      primaryTypographyProps={{ 
                        variant: 'body2',
                        fontWeight: selectedSchema === schema ? 600 : 400
                      }}
                    />
                    <Chip 
                      label={schema === 'all' ? realTables.length : realTables.filter(t => t.schema === schema).length}
                      size="small"
                      sx={{ height: 20, fontSize: '0.7rem' }}
                    />
                  </ListItemButton>
                </ListItem>
              ))}
            </List>
          </Paper>

          {/* Main Panel */}
          <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
            {/* Search Bar */}
            <Box sx={{ p: 3, bgcolor: 'background.paper' }}>
              <TextField
                fullWidth
                size="small"
                placeholder="Search tables..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                InputProps={{
                  startAdornment: (
                    <InputAdornment position="start">
                      <SearchIcon />
                    </InputAdornment>
                  ),
                }}
              />
              <Box sx={{ mt: 2, display: 'flex', gap: 2, alignItems: 'center' }}>
                <Typography variant="body2" color="text.secondary">
                  {filteredTables.length} table{filteredTables.length !== 1 ? 's' : ''} found
                </Typography>
              </Box>
            </Box>

            {/* Table Grid */}
            <Box sx={{ flex: 1, overflow: 'auto', p: 3, pt: 0 }}>
              {tablesLoading ? (
                <Grid container spacing={2}>
                  {[1, 2, 3, 4, 5, 6].map((i) => (
                    <Grid item xs={12} sm={6} md={4} key={i}>
                      <Skeleton variant="rectangular" height={180} />
                    </Grid>
                  ))}
                </Grid>
              ) : realTables.length === 0 ? (
                <Box sx={{ textAlign: 'center', py: 8 }}>
                  <TableIcon sx={{ fontSize: 64, color: 'text.disabled', mb: 2 }} />
                  <Typography variant="h6" color="text.secondary">
                    No tables found in this datasource
                  </Typography>
                  <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                    The datasource may not have been scanned yet or contains no tables.
                  </Typography>
                  <Button
                    variant="outlined"
                    sx={{ mt: 2 }}
                    onClick={() => navigate('/')}
                  >
                    Return to Datasources
                  </Button>
                </Box>
              ) : loading ? (
                <Grid container spacing={2}>
                  {[1, 2, 3, 4, 5, 6].map((i) => (
                    <Grid item xs={12} sm={6} md={4} key={i}>
                      <Skeleton variant="rectangular" height={180} />
                    </Grid>
                  ))}
                </Grid>
              ) : filteredTables.length === 0 ? (
                <Box sx={{ textAlign: 'center', py: 8 }}>
                  <TableIcon sx={{ fontSize: 64, color: 'text.disabled', mb: 2 }} />
                  <Typography variant="h6" color="text.secondary">
                    No tables found
                  </Typography>
                </Box>
              ) : (
                <Grid container spacing={2}>
                  {filteredTables.map((table) => (
                    <Grid item xs={12} sm={6} md={4} key={`${table.schema}.${table.name}`}>
                      <Card 
                        sx={{ 
                          height: '100%',
                          cursor: 'pointer',
                          transition: 'all 0.2s',
                          '&:hover': {
                            transform: 'translateY(-4px)',
                            boxShadow: 4,
                          },
                        }}
                        onClick={() => handleTableClick(table)}
                      >
                        <CardContent>
                          <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1, mb: 2 }}>
                            <TableIcon color="primary" />
                            <Box sx={{ flex: 1 }}>
                              <Typography variant="h6" fontSize="1rem" fontWeight={600}>
                                {table.name}
                              </Typography>
                              <Typography variant="caption" color="text.secondary">
                                {table.schema}
                              </Typography>
                            </Box>
                          </Box>

                          <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                            <Chip
                              icon={<ColumnIcon />}
                              label={`${table.columnCount} columns`}
                              size="small"
                              variant="outlined"
                            />
                            {table.primaryKeys > 0 && (
                              <Chip
                                icon={<KeyIcon sx={{ color: '#ffd700' }} />}
                                label={`${table.primaryKeys} PK`}
                                size="small"
                                variant="outlined"
                              />
                            )}
                            {table.foreignKeys > 0 && (
                              <Chip
                                icon={<KeyIcon sx={{ color: '#2196f3' }} />}
                                label={`${table.foreignKeys} FK`}
                                size="small"
                                variant="outlined"
                              />
                            )}
                          </Box>
                        </CardContent>
                      </Card>
                    </Grid>
                  ))}
                </Grid>
              )}
            </Box>
          </Box>
        </Box>
      )}

      {/* Column Details Modal */}
      {selectedTable && (
        <ColumnDetailsModal
          open={columnModalOpen}
          onClose={() => {
            setColumnModalOpen(false);
            setSelectedTable(null);
          }}
          tableName={`${selectedTable.schema}.${selectedTable.name}`}
          columns={selectedTable.columns}
        />
      )}
    </Box>
  );
};

export default SchemaExplorerPage;
