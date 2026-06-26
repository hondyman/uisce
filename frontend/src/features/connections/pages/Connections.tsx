import { useState, useMemo } from 'react';
import { useQuery } from '@apollo/client';
import { GET_SCOPED_TENANT, GET_TENANTS } from '../../../graphql/queries/tenantQueries';
import { IconPlayerPlay, IconPlugConnected, IconSelect } from '@tabler/icons-react';
import { Tooltip, Box, CircularProgress, LinearProgress } from '@mui/material';
import { useNavigate } from 'react-router-dom';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorIcon from '@mui/icons-material/Error';
import HourglassEmptyIcon from '@mui/icons-material/HourglassEmpty';
import axios from 'axios';
import { devLog } from '../../../utils/devLogger';
import renderCoreCustomChips from '../../../components/common/semanticChips';
import { useAccess } from '../../../contexts/AccessContext';
import { Tenant, Product, DataSource } from '../../../types';
import ScanResultsPanel from '../../../components/ScanResultsPanel';
import ConnectionsFacet from '../components/ConnectionsFacet';
import '../../../styles/Connections.css';

// Local lightweight type for scan results to avoid TS import issues in this file
type LocalScanResult = { tenant_instance_id: string; name?: string; success: boolean; error?: string };

interface ConnectionInfo {
  tenantName: string;
  instanceName: string;
  productName: string;
  sourceName: string;
  datasourceType: string;
  datasourceName: string;
  datasourceId: string;
  config: any;
  driver: string;
  tenantProductId: string;
  alphaDatasourceId:string;
  isGoldCopy: boolean;
  isTenantGoldCopy: boolean;
  // Add full objects for context selection
  tenant: Tenant;
  instance: any; // Using any as the type in flatMap is inferred as any/object
  product: Product;
  datasource: DataSource;
}

const Connections = () => {
  const { currentTenant: scopedTenant, setDatasourceScope, isPlatformOperator } = useAccess();
  
  // State definitions
  const [message, setMessage] = useState('');
  const [errorMessage, setErrorMessage] = useState('');
  const [searchTerm, setSearchTerm] = useState('');
  const [sortConfig, setSortConfig] = useState({ key: 'tenantName', direction: 'ascending' });
  const [scanResultsOpen, setScanResultsOpen] = useState(false);
  const [scanResults, setScanResults] = useState<LocalScanResult[]>([]);
  const [selectedDatasources, setSelectedDatasources] = useState<string[]>([]);
  const [scanStatuses, setScanStatuses] = useState<Record<string, 'idle' | 'running' | 'success' | 'error'>>({});
  const [scanErrors, setScanErrors] = useState<Record<string, string>>({});

  // Facet State
  const [selectedFacetTenantId, setSelectedFacetTenantId] = useState<string | null>(null);
  const [selectedFacetInstanceId, setSelectedFacetInstanceId] = useState<string | null>(null);

  // Batch Progress State
  const [isBatchScanning, setIsBatchScanning] = useState(false);
  const [batchProgress, setBatchProgress] = useState({ completed: 0, total: 0 });

  const navigate = useNavigate();

  // Determine which query to run
  const query = scopedTenant ? GET_SCOPED_TENANT : GET_TENANTS;
  const variables = scopedTenant ? { tenantId: scopedTenant.id } : {};

  const { loading, error, data } = useQuery(query, {
    variables,
    skip: !scopedTenant && !isPlatformOperator, 
  });

  const handleTestConnection = async (datasourceId: string) => {
    setMessage('');
    setErrorMessage('');
    // show running state for this datasource
    setScanStatuses(prev => ({ ...prev, [datasourceId]: 'running' }));
    setScanErrors(prev => ({ ...prev, [datasourceId]: '' }));

    try {
      const response = await axios.post('/api/connections/test', {
        id: datasourceId,
      });
      // success
      setMessage(response.data.message || 'Connection test succeeded');
      setScanStatuses(prev => ({ ...prev, [datasourceId]: 'success' }));
    } catch (err: any) {
      const resp = err?.response;
      const errorMsg = resp?.data?.error || resp?.statusText || 'An unexpected error occurred.';
      setErrorMessage(errorMsg);
      setScanStatuses(prev => ({ ...prev, [datasourceId]: 'error' }));
      setScanErrors(prev => ({ ...prev, [datasourceId]: errorMsg }));
    }
  };

  const handleRunScanner = async (datasourceId: string) => {
    setMessage('');
    setErrorMessage('');
    setScanStatuses(prev => ({ ...prev, [datasourceId]: 'running' }));
    setScanErrors(prev => ({ ...prev, [datasourceId]: '' }));

    try {
      const response = await axios.post('/api/catalog/scan', {
        tenant_instance_id: datasourceId,
      });
      // Handle 207 Multi-Status which may include per-datasource results
      if (response.status === 207 || response.data?.status === 'partial') {
        const results = response.data.results || [];
        setScanResults(results);
        setScanResultsOpen(true);
        const ok = results.filter((r: any) => r.success).length;
        const failed = results.length - ok;
        setMessage(`${response.data.message} (${ok} succeeded, ${failed} failed)`);
        // Update status based on results
        const result = results.find((r: any) => r.tenant_instance_id === datasourceId);
        if (result) {
          setScanStatuses(prev => ({ ...prev, [datasourceId]: result.success ? 'success' : 'error' }));
          if (!result.success && result.error) {
            setScanErrors(prev => ({ ...prev, [datasourceId]: result.error }));
          }
        }
      } else {
        setMessage(response.data.message);
        setScanStatuses(prev => ({ ...prev, [datasourceId]: 'success' }));
      }
    } catch (err: any) {
      // If server responds with 207 but axios treats it as an error, handle that case too
      const resp = err.response;
      if (resp && (resp.status === 207 || resp.data?.status === 'partial')) {
        const results = resp.data.results || [];
        setScanResults(results);
        setScanResultsOpen(true);
        const ok = results.filter((r: any) => r.success).length;
        const failed = results.length - ok;
        setMessage(`${resp.data.message} (${ok} succeeded, ${failed} failed)`);
        // Update status based on results
        const result = results.find((r: any) => r.tenant_instance_id === datasourceId);
        if (result) {
          setScanStatuses(prev => ({ ...prev, [datasourceId]: result.success ? 'success' : 'error' }));
          if (!result.success && result.error) {
            setScanErrors(prev => ({ ...prev, [datasourceId]: result.error }));
          }
        }
        return;
      }
      const errorMsg = resp?.data?.error || 'An unexpected error occurred.';
      setErrorMessage(errorMsg);
      setScanStatuses(prev => ({ ...prev, [datasourceId]: 'error' }));
      setScanErrors(prev => ({ ...prev, [datasourceId]: errorMsg }));
    }
  };

  // Process queue helper
  const processQueue = async (datasourceIds: string[]) => {
    setIsBatchScanning(true);
    setBatchProgress({ completed: 0, total: datasourceIds.length });
    setMessage('');
    setErrorMessage('');

    // Reset processing statuses
    const resetStatuses = datasourceIds.reduce((acc, id) => {
        acc[id] = 'running'; 
        return acc; 
    }, {} as Record<string, 'running'>);
    setScanStatuses(prev => ({ ...prev, ...resetStatuses }));
    
    // Clear previous errors for these IDs
    setScanErrors(prev => {
        const next = { ...prev };
        datasourceIds.forEach(id => delete next[id]);
        return next;
    });

    const CONCURRENCY_LIMIT = 3;
    let completedCount = 0;
    
    // Chunk the IDs
    const chunks = [];
    for (let i = 0; i < datasourceIds.length; i += CONCURRENCY_LIMIT) {
        chunks.push(datasourceIds.slice(i, i + CONCURRENCY_LIMIT));
    }

    for (const chunk of chunks) {
        await Promise.all(chunk.map(async (id) => {
            try {
                const response = await axios.post('/api/catalog/scan', { tenant_instance_id: id });
                // Check for single result success (or multiple if backend returns 207-like structure for single)
                const isSuccess = response.status === 200 || response.data?.success; 
                
                // Usually single scan returns { success: true, ... } or { results: [...] }
                // Let's standardize interpretation:
                if (response.data?.results) {
                     // Multi-status result (rare for single ID call but possible)
                     const res = response.data.results.find((r: any) => r.tenant_instance_id === id);
                     if (res && res.success) {
                         setScanStatuses(prev => ({ ...prev, [id]: 'success' }));
                     } else {
                         setScanStatuses(prev => ({ ...prev, [id]: 'error' }));
                         if (res?.error) setScanErrors(prev => ({ ...prev, [id]: res.error }));
                     }
                } else if (response.data?.success) {
                    setScanStatuses(prev => ({ ...prev, [id]: 'success' }));
                } else {
                    // Fallback assume success if 200 OK and no error field
                    setScanStatuses(prev => ({ ...prev, [id]: 'success' }));
                }
            } catch (err: any) {
                const msg = err.response?.data?.error || err.message || 'Scan failed';
                setScanStatuses(prev => ({ ...prev, [id]: 'error' }));
                setScanErrors(prev => ({ ...prev, [id]: msg }));
            } finally {
                completedCount++;
                setBatchProgress(prev => ({ ...prev, completed: completedCount }));
            }
        }));
    }

    setIsBatchScanning(false);
    setMessage(`Batch scan complete. ${completedCount} processed.`);
  };

  const handleRunAllScanners = () => {
    const allIds = sortedAndFilteredConnections.map((conn: ConnectionInfo) => conn.datasourceId);
    if (allIds.length === 0) return;
    processQueue(allIds);
  };

  const handleRunSelectedScanners = () => {
    if (selectedDatasources.length === 0) return;
    processQueue(selectedDatasources);
  };

  /* 
     Legacy monadic bulk handlers removed in favor of processQueue 
     to prevent timeouts and provide granular progress 
  */

  const handleToggleDatasource = (datasourceId: string) => {
    setSelectedDatasources(prev =>
      prev.includes(datasourceId)
        ? prev.filter(id => id !== datasourceId)
        : [...prev, datasourceId]
    );
  };

  const handleSelectAll = () => {
    const allIds = sortedAndFilteredConnections.map((conn: ConnectionInfo) => conn.datasourceId);
    setSelectedDatasources(prev =>
      prev.length === allIds.length ? [] : allIds
    );
  };

  // Retry handler for an individual datasource triggered from modal
  const handleRetryDatasource = async (datasourceId: string) => {
    try {
      // Call scan endpoint for a single datasource
      const resp = await axios.post('/api/catalog/scan', { tenant_instance_id: datasourceId });
      // Update the modal results if we get updated per-datasource results
      const results = resp.data?.results || [];
      if (results.length) {
        setScanResults(results);
      } else {
        // If backend doesn't return full results, optimistically mark the retried one as success
        setScanResults(prev => prev.map(r => r.tenant_instance_id === datasourceId ? { ...r, success: true, error: undefined } : r));
      }
    } catch (err: any) {
      const resp = err.response;
      if (resp && resp.data?.results) {
        setScanResults(resp.data.results);
      } else {
        // mark the item as failed with the returned error message
        const errMsg = resp?.data?.error || resp?.statusText || 'Retry failed';
        setScanResults(prev => prev.map(r => r.tenant_instance_id === datasourceId ? { ...r, success: false, error: String(errMsg) } : r));
      }
    }
  };

    // Update connections mapping to handle list of tenants (data.tenants is array)
    const connections = useMemo(() => {
      const tenants = data?.tenants || [];
      if (tenants.length === 0) return [] as ConnectionInfo[];

      // Map across ALL returned tenants
      return tenants.flatMap((tenant: Tenant) =>
        (tenant.tenant_instances ?? []).flatMap((instance: any) =>
          (instance.tenant_products ?? []).flatMap((product: Product) =>
            (product.tenant_product_datasources ?? []).map((datasource: DataSource) => ({
              tenantName: String(tenant.display_name || tenant.name || 'Unnamed Tenant'),
              instanceName: String(instance.display_name || instance.instance_name || instance.id || 'Unnamed Instance'),
              productName: product.alpha_product?.product_name,
              sourceName: datasource.source_name,
              datasourceType: datasource.alpha_datasource?.datasource_type,
              datasourceName: datasource.alpha_datasource?.datasource_name,
              datasourceId: datasource.id,
              config: datasource.config as any,
              driver: datasource.alpha_datasource?.datasource_code,
              tenantProductId: product.id,
              alphaDatasourceId: datasource.alpha_tenant_instance_id || datasource.alpha_datasource?.id,
              isGoldCopy: !!datasource.core_id,
              isTenantGoldCopy: tenant.gold_copy ?? false,
              tenant,
              instance,
              product,
              datasource,
            }))
          )
        )
      );
    }, [data]);

  const sortedAndFilteredConnections = useMemo(() => {
    let filtered = connections;
    
    // 1. Facet Filter
    if (selectedFacetTenantId) {
        filtered = filtered.filter((conn: ConnectionInfo) => conn.tenant?.id === selectedFacetTenantId);
        if (selectedFacetInstanceId) {
            filtered = filtered.filter((conn: ConnectionInfo) => conn.instance?.id === selectedFacetInstanceId);
        }
    }

    // 2. Search Filter
    filtered = filtered.filter((conn: ConnectionInfo) =>
      conn.tenantName.toLowerCase().includes(searchTerm.toLowerCase()) ||
      conn.instanceName.toLowerCase().includes(searchTerm.toLowerCase()) ||
      conn.productName.toLowerCase().includes(searchTerm.toLowerCase()) ||
      conn.sourceName.toLowerCase().includes(searchTerm.toLowerCase())
    );

    if (sortConfig.key) {      
      const sortKey = sortConfig.key as keyof ConnectionInfo;
      filtered.sort((a: ConnectionInfo, b: ConnectionInfo) => {
        if (a[sortKey] < b[sortKey]) {
          return sortConfig.direction === 'ascending' ? -1 : 1;
        }
        if (a[sortKey] > b[sortKey]) {
          return sortConfig.direction === 'ascending' ? 1 : -1;
        }
        return 0;
      });
    }

    return filtered;
  }, [connections, searchTerm, sortConfig, selectedFacetTenantId, selectedFacetInstanceId]);

  // Clear selected datasources when search/filter changes
  useMemo(() => {
    const currentIds = sortedAndFilteredConnections.map(conn => conn.datasourceId);
    setSelectedDatasources(prev => prev.filter(id => currentIds.includes(id)));
  }, [sortedAndFilteredConnections]);

  const requestSort = (key: string) => {
    let direction = 'ascending';
    if (sortConfig.key === key && sortConfig.direction === 'ascending') {
      direction = 'descending';
    }
    setSortConfig({ key, direction });
  };

  if (!scopedTenant && !useAccess().isPlatformOperator) return <p>Select a tenant scope to view connections.</p>;
  if (loading) return <p>Loading...</p>;
  if (error) return <p>Error: {error.message}</p>;

  const getSortIndicator = (key: string) => {
    if (sortConfig.key !== key) return null;
    return sortConfig.direction === 'ascending' ? '🔼' : '🔽';
  };

  return (
    <div className="connections-container">
      <div className="connections-header">
        <h1>Connection Management</h1>
      </div>
      
      <Box sx={{ display: 'flex', flexDirection: 'row', alignItems: 'flex-start', gap: 3, width: '100%' }}>
      
      {/* Main Content Area */}
      <div style={{ flex: 1, minWidth: 0 }}>
      
      {/* Progress Bar */}
      {isBatchScanning && (
        <Box sx={{ width: '100%', mb: 2 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
                <span style={{ fontSize: '0.8rem', fontWeight: 600, color: '#555' }}>
                    Batch Processing: {batchProgress.completed} / {batchProgress.total}
                </span>
                <span style={{ fontSize: '0.8rem', color: '#888' }}>
                    {Math.round((batchProgress.completed / batchProgress.total) * 100)}%
                </span>
            </Box>
            <LinearProgress 
                variant="determinate" 
                value={(batchProgress.completed / batchProgress.total) * 100} 
                sx={{ height: 8, borderRadius: 4 }}
            />
        </Box>
      )}

      <div className="connections-controls">
        <input
          type="text"
          placeholder="Search connections..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="connections-search"
        />
        <button 
          onClick={handleRunSelectedScanners} 
          className="connections-run-selected-btn"
          disabled={selectedDatasources.length === 0 || isBatchScanning}
        >
          {isBatchScanning ? 'Running...' : `Run Selected (${selectedDatasources.length})`}
        </button>
        <button 
          onClick={handleRunAllScanners} 
          className="connections-run-all-btn"
          disabled={isBatchScanning}
        >
          {isBatchScanning ? 'Running...' : 'Run All Scanners'}
        </button>
      </div>

      {message && (
        <div className="connections-message success">
          {message}
        </div>
      )}
      
      {errorMessage && (
        <div className="connections-message error">
          {errorMessage}
        </div>
      )}

      {/* Status Summary */}
      {Object.keys(scanStatuses).length > 0 && (
        <div className="connections-status-summary">
          <div className="status-summary-item">
            <CheckCircleIcon color="success" fontSize="small" />
            <span>{Object.values(scanStatuses).filter(s => s === 'success').length} succeeded</span>
          </div>
          <div className="status-summary-item">
            <ErrorIcon color="error" fontSize="small" />
            <span>{Object.values(scanStatuses).filter(s => s === 'error').length} failed</span>
          </div>
          <div className="status-summary-item">
            <CircularProgress size={16} />
            <span>{Object.values(scanStatuses).filter(s => s === 'running').length} running</span>
          </div>
        </div>
      )}

      <div className="connections-table-container">
        <table className="connections-table">
          <thead>
            <tr>
              <th className="checkbox-column">
                <input
                  type="checkbox"
                  checked={selectedDatasources.length === sortedAndFilteredConnections.length && sortedAndFilteredConnections.length > 0}
                  onChange={handleSelectAll}
                  title="Select All Datasources"
                />
              </th>
              <th onClick={() => requestSort('tenantName')}>
                Tenant
                <span className="sort-indicator">{getSortIndicator('tenantName')}</span>
              </th>
              <th onClick={() => requestSort('instanceName')}>
                Instance
                <span className="sort-indicator">{getSortIndicator('instanceName')}</span>
              </th>
              <th onClick={() => requestSort('productName')}>
                Product
                <span className="sort-indicator">{getSortIndicator('productName')}</span>
              </th>
              <th onClick={() => requestSort('sourceName')}>
                Source Name
                <span className="sort-indicator">{getSortIndicator('sourceName')}</span>
              </th>
              <th onClick={() => requestSort('datasourceType')}>
                Type
                <span className="sort-indicator">{getSortIndicator('datasourceType')}</span>
              </th>
              <th className="no-sort">Status</th>
              <th className="no-sort">Actions</th>
            </tr>
          </thead>
          <tbody>
            {sortedAndFilteredConnections.map((conn: ConnectionInfo, index: number) => (
              <tr key={index}>
                <td className="checkbox-column">
                  <input
                    type="checkbox"
                    checked={selectedDatasources.includes(conn.datasourceId)}
                    onChange={() => handleToggleDatasource(conn.datasourceId)}
                    title={`Select ${conn.sourceName}`}
                  />
                </td>
                <td>
                  {conn.tenantName}
                  {conn.isTenantGoldCopy && <Tooltip title="core — read-only"><Box component="span" sx={{ ml: 0.75 }}>{renderCoreCustomChips({ is_core: true })}</Box></Tooltip>}
                </td>
                <td>{conn.instanceName}</td>
                <td>{conn.productName}</td>
                <td>
                  {conn.sourceName}
                  {conn.isGoldCopy && <Tooltip title="core — read-only"><Box component="span" sx={{ ml: 0.75 }}>{renderCoreCustomChips({ is_core: true })}</Box></Tooltip>}
                </td>
                <td>
                  <div className="type-cell-container">
                    <span className="datasource-type-badge">{conn.datasourceType}</span>
                    <span className="datasource-name-badge">{conn.datasourceName}</span>
                  </div>
                </td>
                <td>
                  <div className="status-cell">
                    {scanStatuses[conn.datasourceId] === 'running' && (
                      <Tooltip title="Scan in progress">
                        <CircularProgress size={20} />
                      </Tooltip>
                    )}
                    {scanStatuses[conn.datasourceId] === 'success' && (
                      <Tooltip title="Scan succeeded">
                        <CheckCircleIcon color="success" />
                      </Tooltip>
                    )}
                    {scanStatuses[conn.datasourceId] === 'error' && (
                      <Tooltip title={`Scan failed: ${scanErrors[conn.datasourceId] || 'Unknown error'}`}>
                        <ErrorIcon color="error" />
                      </Tooltip>
                    )}
                    {(!scanStatuses[conn.datasourceId] || scanStatuses[conn.datasourceId] === 'idle') && (
                      <Tooltip title="Ready to scan">
                        <HourglassEmptyIcon color="disabled" />
                      </Tooltip>
                    )}
                  </div>
                </td>
                <td>
                  <div className="connections-actions">
                    <button
                      onClick={() => {
                        // update global access context selection and navigate to schema explorer
                        setDatasourceScope(conn.tenant, conn.instance, conn.product, conn.datasource);
                        try {
                          navigate(`/schema-explorer/${conn.datasourceId}`);
                        } catch (e) {
                          devLog('Navigation to schema explorer failed:', e);
                        }
                      }}
                      className="connections-action-btn"
                      title="Select Datasource"
                    >
                      <IconSelect size={18} />
                    </button>
                    <button 
                      onClick={() => handleTestConnection(conn.datasourceId)} 
                      className="connections-action-btn"
                      title="Test Connection"
                    >
                      <IconPlugConnected size={18} />
                    </button>
                    <button 
                      onClick={() => handleRunScanner(conn.datasourceId)} 
                      className="connections-action-btn"
                      title="Run Scanner"
                      disabled={scanStatuses[conn.datasourceId] === 'running'}
                    >
                      <IconPlayerPlay size={18} />
                    </button>
                    {scanStatuses[conn.datasourceId] === 'error' && (
                      <button
                        onClick={() => handleRunScanner(conn.datasourceId)}
                        className="connections-action-btn retry-btn"
                        title="Retry Scan"
                      >
                        ⟳
                      </button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {sortedAndFilteredConnections.length === 0 && searchTerm && (
        <div className="connections-no-results">
          No connections found matching "{searchTerm}"
        </div>
      )}

      </div> {/* End Main Content */}

      {/* Sidebar Facet */}
      <ConnectionsFacet 
        connections={connections}
        selectedTenantId={selectedFacetTenantId}
        selectedInstanceId={selectedFacetInstanceId}
        onFilterChange={(t, i) => {
            setSelectedFacetTenantId(t);
            setSelectedFacetInstanceId(i);
        }}
      />
      
      </Box>

  {/* Data Catalog & ERD are now shown in the Schema Explorer route. */}
      <ScanResultsPanel
        opened={scanResultsOpen}
        onClose={() => { setScanResultsOpen(false); setScanResults([]); }}
        results={scanResults}
        onRetry={handleRetryDatasource}
      />
    </div>
  );
};

export default Connections;