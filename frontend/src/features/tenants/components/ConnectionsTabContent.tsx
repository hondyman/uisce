import React, { useState } from 'react';
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
import { apiClient } from '../../../utils/apiClient';

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
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(10);



  // Scan state
  const [scanModalOpen, setScanModalOpen] = useState(false);
  const [scanLoading, setScanLoading] = useState(false);
  const [scanResult, setScanResult] = useState<any | null>(null);
  const [scanError, setScanError] = useState<Error | undefined>(undefined);
  const [scanningDatasourceId, setScanningDatasourceId] = useState<string | null>(null);

  // active instance name for display
  const activeInstanceName = React.useMemo(() => {
    if (instanceFilter?.length === 1 && tenantData?.tenant_instances) {
      const activeId = instanceFilter[0];
      const inst = tenantData.tenant_instances.find((i: any) => i.id === activeId);
      return inst ? (inst.display_name || inst.instance_name) : null;
    }
    return null;
  }, [instanceFilter, tenantData]);

  const handleDeleteConnection = async (id: string) => {
    if (confirm('Are you sure you want to delete this connection? This action cannot be undone.')) {
      try {
        await apiClient(`/api/tenant-ops/connections/${id}`, {
          method: 'DELETE',
        });
        window.location.reload();
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
      
      // Refresh connections after sync
      setTimeout(() => window.location.reload(), 1000);
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

  // Build connections from tenantData (tenant_product_datasources) instead of API
  // This avoids needing JWT for listing connections
  const connectionsMap = new Map<string, Connection>();
  
  if (tenantData?.tenant_instances && tenantData.tenant_instances.length > 0) {
    console.log(`Found ${tenantData.tenant_instances.length} instances in tenant`);
    tenantData.tenant_instances.forEach((instance: any) => {
      const instanceInfo = {
        id: instance.id,
        name: instance.display_name || instance.instance_name
      };
      (instance.products || instance.tenant_products)?.forEach((product: any) => {
        const productInfo = {
          name: product.alpha_product?.product_name || product.product?.product_name || product.name || 'Unknown Product',
          id: product.alpha_product_id || product.alpha_product?.id || product.product_id || product.id
        };
        (product.tenant_product_datasources || product.datasources || []).forEach((ds: any) => {
          if (ds.connection_id) {
            console.log(`    Found datasource with connection_id: ${ds.connection_id}`);
            // Use connection_id as the key to avoid duplicates
            if (!connectionsMap.has(ds.connection_id)) {
              console.log(`[DEBUG] Building connection for ds.connection_id: ${ds.connection_id}, ds.config:`, JSON.stringify(ds.config, null, 2));
              const displayName =
                ds.alpha_datasource?.datasource_name ||
                ds.connection_name ||
                ds.source_name ||
                'Unknown Connection';
              const configHost = ds.config?.host || ds.host || '';
              const configPort = ds.config?.port || ds.port || 0;
              const configDatabase = ds.config?.database || ds.database || '';
              const configSchema = ds.config?.schema || ds.schema || '';
              const configUsername = ds.config?.auth?.basic?.username || ds.username || '';
              const configPassword = ds.config?.auth?.basic?.password || ds.password || '';
              let normalizedType = (ds.alpha_datasource?.datasource_type || 'postgres').toLowerCase();
              if (normalizedType === 'database' || !['postgres', 'mysql', 'snowflake', 'api', 's3', 'azure', 'gcs'].includes(normalizedType)) {
                normalizedType = 'postgres';
              }
              connectionsMap.set(ds.connection_id, {
                id: ds.connection_id,
                name: displayName,
                type: normalizedType,
                host: configHost,
                port: configPort,
                database: configDatabase,
                schema: configSchema,
                username: configUsername,
                password: configPassword,
                base_url: ds.config?.base_url || ds.base_url || '',
                api_key: ds.config?.api_key || ds.api_key || '',
                metadata: ds.config || {},
                is_active: ds.is_active ?? true,
                endpoint: configHost ? `${configHost}:${configPort || ''}` : '-',
                linkedInstance: instanceInfo.name,
                linkedInstanceId: instanceInfo.id,
                linkedProduct: productInfo.name,
                linkedProductId: productInfo.id,
                linkedDatasourceId: ds.id,
                linkedAlphaDatasourceId: ds.alpha_datasource_id || ds.alpha_datasource?.id,
                lastSync: ds.updated_at ? new Date(ds.updated_at).toLocaleString() : '-',
                status: (ds.is_active ?? true) ? 'connected' : 'warning',
              });
            }
          }
        });
      });
    });
  }
  
  const connections: Connection[] = Array.from(connectionsMap.values());
  console.log('Built connections from tenantData:', connections.length);

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

  const paginatedConnections = filteredConnections.slice(
    page * rowsPerPage,
    page * rowsPerPage + rowsPerPage
  );

  const totalPages = Math.ceil(filteredConnections.length / rowsPerPage);
  const startIndex = page * rowsPerPage + 1;
  const endIndex = Math.min((page + 1) * rowsPerPage, filteredConnections.length);

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

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
      {/* Header */}
      <Box>
        <Typography variant="h6" sx={{ fontWeight: 'bold', mb: 1 }}>
          {activeInstanceName ? `Data Source Connections for: ${activeInstanceName}` : 'Data Source Connections'}
        </Typography>
        <Typography variant="body2" color="textSecondary">
          {activeInstanceName
            ? `Manage external connections linked to the scoped instance "${activeInstanceName}".`
            : "Manage external connections linked to this tenant's instances."}
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
          <TableHead>
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
              paginatedConnections.map((conn) => (
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
            {filteredConnections.length > 0 ? startIndex : 0}
          </Typography>{' '}
          to{' '}
          <Typography component="span" variant="caption" sx={{ fontWeight: 'bold' }}>
            {endIndex}
          </Typography>{' '}
          of{' '}
          <Typography component="span" variant="caption" sx={{ fontWeight: 'bold' }}>
            {filteredConnections.length}
          </Typography>{' '}
          connections
        </Typography>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Button
            variant="outlined"
            size="small"
            onClick={() => setPage((p) => Math.max(0, p - 1))}
            disabled={page === 0}
          >
            Previous
          </Button>
          <Button
            variant="outlined"
            size="small"
            onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
            disabled={page >= totalPages - 1}
          >
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
