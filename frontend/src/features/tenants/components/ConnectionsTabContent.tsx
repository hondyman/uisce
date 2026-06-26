import React, { useState } from 'react';
import { useQuery, useMutation, gql } from '@apollo/client';
import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Button,
  Select,
  MenuItem,
  CircularProgress,
  IconButton,
  Chip,
  Box,
  Typography,
  Alert,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TableSortLabel,
} from '@mui/material';
import {
  Add,
  Refresh,
  Edit,
  FilterList,
  PlayArrow,
  Schema as SchemaIcon,
  Visibility as VisibilityIcon,
  DeleteOutline,
  Sync,
} from '@mui/icons-material';
import { ConnectionTestDialog } from '../../connections/components/ConnectionTestDialog';
import ScanProgressModal from './ScanProgressModal';
import { SCAN_DATASOURCE, DELETE_CONNECTION } from '../../../graphql/mutations/tenantMutations';

export interface Connection {
  id: string;
  name: string;
  type: string;
  endpoint: string;
  linkedInstance?: string;
  linkedInstanceId?: string;
  linkedProduct?: string;
  linkedProductId?: string;
  linkedDatasourceId?: string;
  linkedAlphaDatasourceId?: string;
  lastSync?: string;
  status: 'connected' | 'warning' | 'error';
  // Raw data from API
  host?: string;
  port?: number;
  database?: string;
  schema?: string;
  username?: string;
  password?: string;
  base_url?: string;
  api_key?: string;
  metadata?: Record<string, any>;
  is_active?: boolean;
}

interface ConnectionsTabContentProps {
  tenantId: string;
  datasourceId: string;
  instanceFilter?: string[] | null;
  productFilter?: string[] | null;
  isGoldCopy?: boolean;
  onAddConnection?: () => void;
  onEditConnection?: (connection: Connection) => void;
  tenantData?: any;
}

// GraphQL Queries
const GET_TENANT_CONNECTIONS = gql`
  query GetTenantConnections($tenantId: uuid!) {
    connections(where: { tenant_id: { _eq: $tenantId } }) {
      id
      name
      type
      host
      port
      database
      schema
      username
      password
      is_active
      created_at
      updated_at
      metadata
    }
  }
`;

// Query to get product and instance info linked to connections via tenant_product_datasources
const GET_CONNECTION_RELATIONSHIPS = gql`
  query GetConnectionRelationships($tenantId: uuid!) {
    tenant_product_datasource(
      where: {
        tenant_product: { tenant_id: { _eq: $tenantId } }
      }
    ) {
      id
      connection_id
      last_scan_at
      tenant_instance {
        id
        instance_name
        display_name
      }
      tenant_product {
        id
        tenant_id
        alpha_product {
          id
          product_name
        }
      }
    }
  }
`;

// GraphQL Mutations

const getStatusColor = (status: string) => {
  switch (status) {
    case 'connected':
    case 'success':
      return 'success';
    case 'warning':
      return 'warning';
    case 'error':
    case 'failed':
      return 'error';
    default:
      return 'default';
  }
};

const getStatusLabel = (status: string) => {
  switch (status) {
    case 'connected':
    case 'success':
      return 'Connected';
    case 'warning':
      return 'Warning';
    case 'error':
    case 'failed':
      return 'Error';
    default:
      return 'Unknown';
  }
};

const getTypeIcon = (type: string) => {
  switch (type?.toLowerCase()) {
    case 'postgres':
    case 'mysql':
    case 'database':
      return '🗄️';
    case 'api':
    case 'rest':
      return '☁️';
    case 'storage':
    case 's3':
      return '📦';
    case 'snowflake':
      return '❄️';
    default:
      return '🔌';
  }
};

const getTypeLabel = (type: string) => {
  switch (type?.toLowerCase()) {
    case 'postgres':
      return 'PostgreSQL';
    case 'mysql':
      return 'MySQL';
    case 'snowflake':
      return 'Snowflake';
    case 'api':
    case 'rest':
      return 'REST API';
    case 'storage':
    case 's3':
      return 'S3 Storage';
    default:
      return type || 'Unknown';
  }
};

export const ConnectionsTabContent: React.FC<ConnectionsTabContentProps> = ({
  tenantId,
  datasourceId,
  instanceFilter,
  productFilter,
  isGoldCopy = false,
  onAddConnection,
  onEditConnection,
  tenantData,
}) => {
  const [filterType, setFilterType] = useState('all');
  const [sortBy, setSortBy] = useState('name');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc');
  const [testingConnectionId, setTestingConnectionId] = useState<string | null>(null);
  const [testDialogOpen, setTestDialogOpen] = useState(false);
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);
  const [viewConfigConnection, setViewConfigConnection] = useState<Connection | null>(null);
  const [syncing, setSyncing] = useState(false);
  const [syncResult, setSyncResult] = useState<{ success: boolean; message: string } | null>(null);



  // Scan state
  const [scanModalOpen, setScanModalOpen] = useState(false);
  const [scanLoading, setScanLoading] = useState(false);
  const [scanResult, setScanResult] = useState<any | null>(null);
  const [scanError, setScanError] = useState<Error | undefined>(undefined);
  const [scanningDatasourceId, setScanningDatasourceId] = useState<string | null>(null);

  // Fetch connections from backend
  const { data, loading, error, refetch } = useQuery(GET_TENANT_CONNECTIONS, {
    variables: { tenantId },
    skip: !tenantId,
  });

  // Fetch connection relationships for this tenant
  // NOTE: Temporarily disabled due to schema filter issues; using tenantData.tenant_products instead
  const { data: relationshipData } = useQuery(GET_CONNECTION_RELATIONSHIPS, {
    variables: { tenantId },
    skip: true,  // Skip this query; we'll use tenantData instead
  });

  const [deleteConnection] = useMutation(DELETE_CONNECTION, {
    onCompleted: () => {
      refetch();
    },
    onError: (err) => {
      console.error('Error deleting connection:', err);
      // Optional: Show error alert
    }
  });

  const handleDeleteConnection = async (id: string) => {
    if (confirm('Are you sure you want to delete this connection? This action cannot be undone.')) {
      try {
        await deleteConnection({ variables: { id } });
      } catch (err) {
        console.error(err);
      }
    }
  };

  const handleSyncConnections = async () => {
    if (!tenantId || isGoldCopy) return;
    
    setSyncing(true);
    setSyncResult(null);
    
    try {
      const response = await fetch('/api/instance/sync-connections-from-goldcopy', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          tenant_id: tenantId,
        }),
      });
      
      if (!response.ok) {
        throw new Error(`Sync failed: ${response.statusText}`);
      }
      
      const result = await response.json();
      setSyncResult({
        success: true,
        message: `Successfully synced ${result.connections_created || 0} new and ${result.connections_updated || 0} updated connections`,
      });
      
      // Refetch connections after sync
      setTimeout(() => refetch(), 1000);
    } catch (err: any) {
      console.error('Error syncing connections:', err);
      setSyncResult({
        success: false,
        message: err.message || 'Failed to sync connections',
      });
    } finally {
      setSyncing(false);
    }
  };

  // Map database connections to the Connection interface
  
  // Build product and instance lookup maps from tenant_product_datasources
  const productMap = new Map<string, any>();
  const instanceMap = new Map<string, any>();
  const lastScanMap = new Map<string, any>();
  const tpdIdMap = new Map<string, string>(); // Map connection_id -> tenant_product_datasource.id
  
  // Build maps from the relationship data
  if (relationshipData?.tenant_product_datasource) {
    console.log(`Relationship query returned ${relationshipData.tenant_product_datasource.length} TPDs`);
    relationshipData.tenant_product_datasource.forEach((tpd: any) => {
      const connectionId = tpd.connection_id;
      
      // Map TPD ID by connection ID (for scan operations)
      if (tpd.id && connectionId) {
        tpdIdMap.set(connectionId, tpd.id);
      }
      
      // Map product info by connection ID
      const productInfo = {
        name: tpd.tenant_product?.alpha_product?.product_name,
        id: tpd.tenant_product?.alpha_product?.id
      };
      productMap.set(connectionId, productInfo);
      
      // Map instance info by connection ID
      const instanceInfo = {
        id: tpd.tenant_instance?.id,
        name: tpd.tenant_instance?.display_name || tpd.tenant_instance?.instance_name
      };
      instanceMap.set(connectionId, instanceInfo);
      
      // Map last scan time by connection ID
      if (tpd.last_scan_at) {
        lastScanMap.set(connectionId, tpd.last_scan_at);
      }
    });
  } else {
    console.log('No relationship data returned from query');
  }
  
  // Also try tenant_products if available (from GET_SCOPED_TENANT)
  if (tenantData?.tenant_products && tenantData.tenant_products.length > 0) {
    console.log(`Found ${tenantData.tenant_products.length} products in tenant`);
    tenantData.tenant_products.forEach((tp: any) => {
      const productInfo = {
        name: tp.alpha_product?.product_name,
        id: tp.alpha_product?.id
      };
      console.log(`  Checking product ${productInfo.name}: has ${tp.tenant_product_datasources?.length || 0} datasources`);
      tp.tenant_product_datasources?.forEach((tpd: any) => {
        if (tpd.connection_id) {
          console.log(`    Found TPD with connection_id: ${tpd.connection_id}`);
          
          // Map TPD ID by connection ID (for scan operations)
          if (tpd.id) {
            tpdIdMap.set(tpd.connection_id, tpd.id);
            console.log(`      Mapped TPD ID: ${tpd.id}`);
          }
          
          productMap.set(tpd.connection_id, productInfo);
          
          // Map instance from tenant_instances array using tenant_instance_id
          if (tpd.tenant_instance_id && tenantData.tenant_instances) {
            const foundInstance = tenantData.tenant_instances.find((ti: any) => ti.id === tpd.tenant_instance_id);
            if (foundInstance) {
              instanceMap.set(tpd.connection_id, {
                id: foundInstance.id,
                name: foundInstance.display_name || foundInstance.instance_name
              });
              console.log(`      Mapped instance: ${foundInstance.display_name || foundInstance.instance_name}`);
            }
          }
        } else {
          console.log(`    Found TPD without connection_id (connection_id=null)`);
        }
      });
    });
  }
  
  console.log('Final product map:', Object.fromEntries(productMap));
  console.log('Final instance map:', Object.fromEntries(instanceMap));
  console.log('Final TPD ID map:', Object.fromEntries(tpdIdMap));
  
  const connections: Connection[] = (data?.connections || []).map((conn: any) => {
    // Get linked instance and product from the maps
    const linkedInstanceInfo = instanceMap.get(conn.id);
    const linkedProduct = productMap.get(conn.id);
    const lastScan = lastScanMap.get(conn.id);
    const tpdId = tpdIdMap.get(conn.id); // Get the tenant_product_datasource ID
    
    if (linkedProduct) {
      console.log(`Connection ${conn.id} (${conn.name}): found product ${linkedProduct.name}`);
    } else {
      console.log(`Connection ${conn.id} (${conn.name}): NO PRODUCT FOUND`);
    }
    
    if (tpdId) {
      console.log(`Connection ${conn.id} (${conn.name}): mapped to TPD ID ${tpdId}`);
    }
    
    return {
      id: conn.id,
      name: conn.name,
      type: conn.type,
      host: conn.host,
      port: conn.port,
      database: conn.database,
      schema: conn.schema,
      username: conn.username,
      password: conn.password,
      base_url: conn.base_url || conn.metadata?.base_url,
      api_key: conn.api_key || conn.metadata?.api_key,
      metadata: conn.metadata,
      is_active: conn.is_active,
      endpoint: conn.host ? `${conn.host}:${conn.port || ''}` : (conn.base_url || '-'),
      linkedInstance: linkedInstanceInfo?.name || '-',
      linkedInstanceId: linkedInstanceInfo?.id,
      linkedProduct: linkedProduct?.name || '-',
      linkedProductId: linkedProduct?.id,
      linkedDatasourceId: tpdId, // Use TPD ID instead of connection ID
      linkedAlphaDatasourceId: undefined,
      lastSync: lastScan ? new Date(lastScan).toLocaleString() : '-',
      status: (conn.is_active ? 'connected' : 'warning'),
    };
  });

  const filteredConnections: Connection[] = connections.filter((conn: Connection) => {
    const matchesType = filterType === 'all' || conn.type?.toLowerCase() === filterType;
    
    // Filter by instance (array of IDs)
    const matchesInstance = !instanceFilter || instanceFilter.length === 0 || 
      (conn.linkedInstanceId && instanceFilter.includes(conn.linkedInstanceId));
    
    // Filter by product (array of IDs)
    const matchesProduct = !productFilter || productFilter.length === 0 ||
      (conn.linkedProductId && productFilter.includes(conn.linkedProductId));
    
    return matchesType && matchesInstance && matchesProduct;
  }).sort((a, b) => {
    let aValue: any = '';
    let bValue: any = '';

    switch (sortBy) {
      case 'name':
        aValue = a.name?.toLowerCase() || '';
        bValue = b.name?.toLowerCase() || '';
        break;
      case 'type':
        aValue = a.type?.toLowerCase() || '';
        bValue = b.type?.toLowerCase() || '';
        break;
      case 'linkedProduct':
        aValue = a.linkedProduct?.toLowerCase() || '';
        bValue = b.linkedProduct?.toLowerCase() || '';
        break;
      case 'linkedInstance':
        aValue = a.linkedInstance?.toLowerCase() || '';
        bValue = b.linkedInstance?.toLowerCase() || '';
        break;
      case 'status':
        aValue = a.status || '';
        bValue = b.status || '';
        break;
      default:
        aValue = (a as any)[sortBy] || '';
        bValue = (b as any)[sortBy] || '';
    }

    if (aValue < bValue) return sortOrder === 'asc' ? -1 : 1;
    if (aValue > bValue) return sortOrder === 'asc' ? 1 : -1;
    return 0;
  });

  const handleSort = (property: string) => {
    const isAsc = sortBy === property && sortOrder === 'asc';
    setSortOrder(isAsc ? 'desc' : 'asc');
    setSortBy(property);
  };

  const handleTestConnection = async (id: string) => {
      setTestResult({
        success: true,
        message: 'Connection test initiated via backend',
      });
      setTestDialogOpen(true);
  };

  const handleScan = async (datasourceId: string) => {
    setScanningDatasourceId(datasourceId);
    setScanLoading(true);
    setScanError(undefined);
    setScanResult(null);
    setScanModalOpen(true);

    // When using SSE streaming, the SSE endpoint triggers the scan.
    // We don't call the GraphQL mutation to avoid duplicate scans.
    // The SSE will stream progress and the modal will show completion.
    // For now, just open the modal and let SSE handle it.
    setScanLoading(false);
  };

  if (!tenantId) {
    return (
      <Alert severity="warning">
        Please select a tenant to view connections
      </Alert>
    );
  }

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error">
        Error loading connections: {error.message}
      </Alert>
    );
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
      {/* Header */}
      <Box>
        <Typography variant="h6" sx={{ fontWeight: 'bold', mb: 1 }}>
          Data Source Connections
        </Typography>
        <Typography variant="body2" color="textSecondary">
          Manage external connections linked to this tenant's instances.
        </Typography>
      </Box>

      {/* Controls */}
      <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap', alignItems: 'center' }}>
        <Select
          value={filterType}
          onChange={(e) => setFilterType(e.target.value)}
          size="small"
          sx={{ minWidth: 160 }}
          startAdornment={<FilterList sx={{ mr: 1 }} />}
        >
          <MenuItem value="all">All Types</MenuItem>
          <MenuItem value="database">Databases</MenuItem>
          <MenuItem value="api">API Endpoints</MenuItem>
          <MenuItem value="storage">File Stores</MenuItem>
        </Select>
        <Box sx={{ ml: 'auto', display: 'flex', gap: 1 }}>
          {!isGoldCopy && (
            <Button
              variant="outlined"
              startIcon={syncing ? <CircularProgress size={20} /> : <Refresh />}
              onClick={handleSyncConnections}
              disabled={syncing}
            >
              {syncing ? 'Syncing...' : 'Sync from Gold Copy'}
            </Button>
          )}
          {isGoldCopy && (
            <Button
              variant="contained"
              startIcon={<Add />}
              onClick={onAddConnection}
            >
              Add Connection
            </Button>
          )}
        </Box>
      </Box>

      {/* Connections Table */}
      <TableContainer component={Paper} variant="outlined">
        <Table>
          <TableHead sx={{ backgroundColor: '#f5f5f5' }}>
            <TableRow>
              <TableCell sx={{ fontWeight: 'bold' }}>
                <TableSortLabel
                  active={sortBy === 'name'}
                  direction={sortBy === 'name' ? sortOrder : 'asc'}
                  onClick={() => handleSort('name')}
                >
                  Connection Name
                </TableSortLabel>
              </TableCell>
              <TableCell sx={{ fontWeight: 'bold' }}>
                <TableSortLabel
                  active={sortBy === 'type'}
                  direction={sortBy === 'type' ? sortOrder : 'asc'}
                  onClick={() => handleSort('type')}
                >
                  Type
                </TableSortLabel>
              </TableCell>
              <TableCell sx={{ fontWeight: 'bold', width: 120 }}>
                Core/Custom
              </TableCell>
              <TableCell sx={{ fontWeight: 'bold' }}>
                <TableSortLabel
                  active={sortBy === 'linkedProduct'}
                  direction={sortBy === 'linkedProduct' ? sortOrder : 'asc'}
                  onClick={() => handleSort('linkedProduct')}
                >
                  Product
                </TableSortLabel>
              </TableCell>
              <TableCell sx={{ fontWeight: 'bold' }}>
                <TableSortLabel
                  active={sortBy === 'linkedInstance'}
                  direction={sortBy === 'linkedInstance' ? sortOrder : 'asc'}
                  onClick={() => handleSort('linkedInstance')}
                >
                  Linked Instance
                </TableSortLabel>
              </TableCell>
              <TableCell sx={{ fontWeight: 'bold' }}>Last Sync</TableCell>
              <TableCell sx={{ fontWeight: 'bold' }}>
                <TableSortLabel
                  active={sortBy === 'status'}
                  direction={sortBy === 'status' ? sortOrder : 'asc'}
                  onClick={() => handleSort('status')}
                >
                  Status
                </TableSortLabel>
              </TableCell>
              <TableCell align="right" sx={{ fontWeight: 'bold' }}>
                Actions
              </TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredConnections.length === 0 ? (
              <TableRow>
                <TableCell colSpan={8} align="center" sx={{ py: 4 }}>
                  <Typography color="textSecondary">
                    No connections found
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              filteredConnections.map((conn) => (
                <TableRow key={conn.id} hover>
                  <TableCell>
                    <Box>
                      <Typography variant="body2" sx={{ fontWeight: 'bold' }}>
                        {conn.name}
                      </Typography>
                      <Typography
                        variant="caption"
                        sx={{ color: '#666', fontFamily: 'monospace' }}
                      >
                        {conn.endpoint}
                      </Typography>
                    </Box>
                  </TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <span>{getTypeIcon(conn.type)}</span>
                      <Typography variant="body2">{getTypeLabel(conn.type)}</Typography>
                    </Box>
                  </TableCell>
                  <TableCell align="center">
                    {isGoldCopy ? (
                      <Chip 
                        label="CORE" 
                        size="small" 
                        color="info" 
                        title="Gold Copy Definition"
                        sx={{ fontWeight: 'bold' }} 
                      />
                    ) : (
                      <Chip 
                        label="CUSTOM" 
                        size="small" 
                        variant="outlined"
                        sx={{ fontWeight: 'bold' }} 
                      />
                    )}
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2">{conn.linkedProduct || '-'}</Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2">{conn.linkedInstance || '-'}</Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2">{conn.lastSync || '-'}</Typography>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={getStatusLabel(conn.status)}
                      color={getStatusColor(conn.status) as any}
                      size="small"
                      variant="outlined"
                    />
                  </TableCell>
                  <TableCell align="right">
                    <Box sx={{ display: 'flex', gap: 1, justifyContent: 'flex-end' }}>
                      {conn.linkedDatasourceId && (
                        <>
                          <IconButton
                            size="small"
                            onClick={() => handleScan(conn.linkedDatasourceId!)}
                            title="Run Metadata Scan"
                            sx={{ color: 'primary.main' }}
                          >
                            <PlayArrow fontSize="small" />
                          </IconButton>
                          <IconButton
                            size="small"
                            component="a"
                            href={`/schema-explorer?datasource=${conn.linkedDatasourceId}`}
                            onClick={(e) => e.stopPropagation()}
                            title="View Catalog"
                            sx={{ color: 'info.main' }}
                          >
                            <SchemaIcon fontSize="small" />
                          </IconButton>
                        </>
                      )}
                      <IconButton
                        size="small"
                        onClick={() => handleTestConnection(conn.id)}
                        disabled={testingConnectionId === conn.id}
                        title="Test Connection"
                      >
                        {testingConnectionId === conn.id ? (
                          <CircularProgress size={20} />
                        ) : (
                          <Refresh fontSize="small" />
                        )}
                      </IconButton>
                      <IconButton
                        size="small"
                        onClick={() => onEditConnection?.(conn)}
                        title="Edit Connection"
                      >
                        <Edit fontSize="small" />
                      </IconButton>
                      <IconButton
                        size="small"
                        onClick={() => setViewConfigConnection(conn)}
                        title="View Configuration"
                      >
                        <VisibilityIcon fontSize="small" />
                      </IconButton>
                      <IconButton
                        size="small"
                        color="error"
                        onClick={() => handleDeleteConnection(conn.id)}
                        title="Delete Connection"
                      >
                       <DeleteOutline fontSize="small" />
                      </IconButton>
                    </Box>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Pagination Footer */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography variant="caption" color="textSecondary">
          Showing{' '}
          <Typography component="span" variant="caption" sx={{ fontWeight: 'bold' }}>
            1
          </Typography>{' '}
          to{' '}
          <Typography component="span" variant="caption" sx={{ fontWeight: 'bold' }}>
            {Math.min(4, filteredConnections.length)}
          </Typography>{' '}
          of{' '}
          <Typography component="span" variant="caption" sx={{ fontWeight: 'bold' }}>
            {connections.length}
          </Typography>{' '}
          connections
        </Typography>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Button variant="outlined" size="small" disabled>
            Previous
          </Button>
          <Button variant="outlined" size="small">
            Next
          </Button>
        </Box>
      </Box>

      {/* Connection Test Dialog */}
      <ConnectionTestDialog
        open={testDialogOpen}
        loading={testingConnectionId !== null}
        result={testResult}
        onClose={() => {
          setTestDialogOpen(false);
          setTestResult(null);
        }}
      />

      {/* View Configuration Dialog */}
      <Dialog 
        open={!!viewConfigConnection} 
        onClose={() => setViewConfigConnection(null)}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>Connection Configuration</DialogTitle>
        <DialogContent>
          {viewConfigConnection && (
            <Box sx={{ mt: 2 }}>
              <Typography variant="subtitle2" sx={{ fontWeight: 'bold', mb: 1 }}>
                Connection Details
              </Typography>
              <Box sx={{ 
                backgroundColor: '#f5f5f5', 
                p: 2, 
                borderRadius: 1, 
                fontFamily: 'monospace',
                fontSize: '0.85rem',
                maxHeight: '400px',
                overflow: 'auto',
              }}>
                <div>
                  <strong>Name:</strong> {viewConfigConnection.name}
                </div>
                <div>
                  <strong>Type:</strong> {viewConfigConnection.type}
                </div>
                {viewConfigConnection.host && (
                  <div>
                    <strong>Host:</strong> {viewConfigConnection.host}
                  </div>
                )}
                {viewConfigConnection.port && (
                  <div>
                    <strong>Port:</strong> {viewConfigConnection.port}
                  </div>
                )}
                {viewConfigConnection.database && (
                  <div>
                    <strong>Database:</strong> {viewConfigConnection.database}
                  </div>
                )}
                {viewConfigConnection.schema && (
                  <div>
                    <strong>Schema:</strong> {viewConfigConnection.schema}
                  </div>
                )}
                {viewConfigConnection.username && (
                  <div>
                    <strong>Username:</strong> {viewConfigConnection.username}
                  </div>
                )}
                {viewConfigConnection.base_url && (
                  <div>
                    <strong>Base URL:</strong> {viewConfigConnection.base_url}
                  </div>
                )}
                {viewConfigConnection.api_key && (
                  <div>
                    <strong>API Key:</strong> ••••••••
                  </div>
                )}
                <Box sx={{ mt: 1.25 }}>
                  <strong>Status:</strong> {viewConfigConnection.is_active ? 'Active' : 'Inactive'}
                </Box>
                {viewConfigConnection.metadata && Object.keys(viewConfigConnection.metadata).length > 0 && (
                  <Box sx={{ mt: 1.25 }}>
                    <strong>Metadata:</strong>
                    <Box 
                      component="pre" 
                      sx={{ fontSize: '0.75rem', margin: '5px 0', overflow: 'auto' }}
                    >
                      {JSON.stringify(viewConfigConnection.metadata, null, 2)}
                    </Box>
                  </Box>
                )}
              </Box>
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setViewConfigConnection(null)}>Close</Button>
        </DialogActions>
      </Dialog>
      
      <ScanProgressModal
        open={scanModalOpen}
        onClose={() => {
          setScanModalOpen(false);
          setScanningDatasourceId(null);
        }}
        loading={scanLoading}
        result={scanResult}
        error={scanError}
        datasourceId={scanningDatasourceId || undefined}
        useStreaming={true}
      />
    </Box>
  );
};
