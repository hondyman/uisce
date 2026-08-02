import React, { useState, useMemo, useEffect } from 'react';
import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import {
  Box,
  Button,
  Card,
  CircularProgress,
  Alert,
  Typography,
  Breadcrumbs,
  Tabs,
  Tab,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  FormControlLabel,
  Switch,
  Stack,
  Chip,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
} from '@mui/material';
import {
  Edit as EditIcon,
  Delete as DeleteIcon,
} from '@mui/icons-material';
import { useApiMutation } from '../../../hooks/useApiMutation';
import { useApiQuery } from '../../../hooks/useApiQuery';
import { useTenant } from '../../../contexts/TenantContext';
import { useAccess } from '../../../contexts/AccessContext';
import type { TenantInstance } from '../../../types';
import { apiClient } from '../../../utils/apiClient';
import InstancesTableV2 from '../components/InstancesTableV2';
import { ScopedInstanceEditor } from '../components/ScopedInstanceEditor';
import { ConnectionsTabContent } from '../components/ConnectionsTabContent';
import { AuditLogTabContent } from '../components/AuditLogTabContent';
import { ConfigurationTabContent } from '../components/ConfigurationTabContent';
import { ProductsTabContent } from '../components/ProductsTabContent';
import LookupsManagementTab from '../components/LookupsManagementTabV2';
import AbbreviationsTab from '../components/AbbreviationsTab';
import { ConnectionsFacets } from '../components/ConnectionsFacets';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;

  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`tenant-tabpanel-${index}`}
      aria-labelledby={`tenant-tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ pt: 3 }}>{children}</Box>}
    </div>
  );
}

export const TenantDetailPageV2: React.FC = () => {
  const { tenantId } = useParams<{ tenantId: string }>();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const {
    tenant: scopedTenant,
    product: scopedProduct,
    datasource: scopedDatasource,
  } = useTenant();
  const { scope } = useAccess();

  const { data: tenantData, loading, error, refetch: refetchTenant } = useApiQuery<{ tenant: any }>(
    tenantId ? `/api/tenants/${tenantId}` : '',
    { skip: !tenantId }
  );

  const { data: datasourcesData } = useApiQuery<{ alpha_datasource: any[] }>(
    '/api/rest/datasources'
  );

  const { data: alphaProductsData } = useApiQuery<{ alpha_product: any[] }>(
    '/api/rest/products'
  );

  const updateTenant = useApiMutation<any, any>(
    '/api/tenant-ops/update',
    'PATCH',
    { onCompleted: () => refetchTenant() }
  );
  const addTenantProduct = useApiMutation<any, any>(
    `/api/tenant-ops/products`,
    'POST',
    { onCompleted: () => refetchTenant() }
  );

  const createTenantInstance = useApiMutation<any, any>(
    `/api/tenant-ops/instances`,
    'POST',
    { onCompleted: () => refetchTenant() }
  );
  const updateTenantInstance = useApiMutation<any, any>(
    `/api/tenant-ops/instances`,
    'PATCH',
    { onCompleted: () => refetchTenant() }
  );
  const deleteTenantInstance = useApiMutation<any, any>(
    `/api/tenant-ops/instances`,
    'DELETE',
    { onCompleted: () => refetchTenant() }
  );
  const createConnection = useApiMutation<any, any>(
    `/api/tenant-ops/connections`,
    'POST',
    { onCompleted: () => refetchTenant() }
  );
  const addTenantProductDatasource = useApiMutation<any, any>(
    `/api/tenant-ops/product-datasources`,
    'POST',
    { onCompleted: () => refetchTenant() }
  );
  const testConnection = useApiMutation<any, any>(
    `/api/tenant-ops/connections/test`,
    'POST',
    {
      onCompleted: (data) => {
        setTestConnectionResult(data);
        setTestConnectionLoading(false);
      },
      onError: (err) => {
        setTestConnectionResult({
          success: false,
          message: err.message || 'Failed to test connection',
        });
        setTestConnectionLoading(false);
      },
    }
  );

  // State
  const [activeTab, setActiveTab] = useState(0);
  const [editMode, setEditMode] = useState(false);
  const [tenantEditForm, setTenantEditForm] = useState({
    display_name: '',
    description: '',
    is_active: true,
  });
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [instanceDialogOpen, setInstanceDialogOpen] = useState(false);
  const [editingInstance, setEditingInstance] = useState<TenantInstance | null>(null);
  const [instanceForm, setInstanceForm] = useState({
    instance_name: '',
    display_name: '',
    description: '',
    url: '',
    is_active: true,
  });
  const [connectionDialogOpen, setConnectionDialogOpen] = useState(false);
  const [editingConnection, setEditingConnection] = useState<any>(null);
  const [selectedConnectionProduct, setSelectedConnectionProduct] = useState<string>('');
  const [selectedConnectionInstance, setSelectedConnectionInstance] = useState<string>('');
  const [selectedAlphaDatasource, setSelectedAlphaDatasource] = useState<string>('');
  const [selectedInstanceFilters, setSelectedInstanceFilters] = useState<string[]>([]);
  const [selectedProductFilters, setSelectedProductFilters] = useState<string[]>([]);
  const [selectedProductForCounts, setSelectedProductForCounts] = useState<string | null>(null);
  const [connectionForm, setConnectionForm] = useState({
    name: '',
    type: 'postgres',
    host: '',
    port: '5432',
    database: '',
    schema: '',
    username: '',
    secret_path: '',
    base_url: '',
    api_key: '',
    auth_type: 'basic', // basic, oauth, key_pair, bearer
    metadata: {} as Record<string, any>,
    is_active: true,
  });
  const [connectionConfigJson, setConnectionConfigJson] = useState('{}');
  const [testConnectionLoading, setTestConnectionLoading] = useState(false);
  const [connectionsRefreshKey, setConnectionsRefreshKey] = useState(0);
  const [testConnectionResult, setTestConnectionResult] = useState<{ success: boolean; message: string } | null>(null);

  const tenant = useMemo(() => tenantData?.tenant ?? null, [tenantData]);
  const instances = useMemo(() => tenant?.tenant_instances ?? [], [tenant]);

  // Automatically switch context to this tenant if not already selected
  useEffect(() => {
    if (tenant && tenant.id) {
       // Only switch if we are NOT already on this tenant.
       // scopedTenant might be null on first load, so we check ID match.
        if (!scopedTenant || scopedTenant.id !== tenant.id) {
             localStorage.setItem('selected_tenant', JSON.stringify(tenant));
            // Triggers for Apollo Client next request
       }
    }
  }, [tenant, scopedTenant]);

  const enrichedInstances = useMemo(() => {
    if (!tenant) return [];

    const instanceResourcesMap = new Map<string, { products: string[], connections: any[] }>();

    // New REST shape: products are nested inside instances (tenant.instances[].products[].Datasources[])
    tenant.tenant_instances?.forEach((instance: any) => {
      (instance.products || instance.tenant_products)?.forEach((product: any) => {
        (product.tenant_product_datasources || product.datasources || []).forEach((ds: any) => {
          if (!instanceResourcesMap.has(instance.id)) {
            instanceResourcesMap.set(instance.id, { products: [], connections: [] });
          }
          const resource = instanceResourcesMap.get(instance.id)!;
          resource.connections.push({
            id: ds.id, // Use TPD ID for uniqueness
            connectionId: ds.connection_id || ds.id,
            name: ds.source_name || ds.alpha_datasource?.datasource_name || 'Datasource',
            type: 'Datasource',
            productName: product.alpha_product?.product_name,
            productId: product.alpha_product_id
          });
        });
      });
    });

    return (tenant.tenant_instances ?? []).map((instance: any) => {
        const resources = instanceResourcesMap.get(instance.id);
        const connections = resources?.connections || [];

        const detailsMap = new Map<string, any>();
        connections.forEach((conn: any) => {
            if (!detailsMap.has(conn.productId)) {
                const displayName = conn.productName || `Product ${conn.productId?.substring(0, 8)}...` || 'Unknown Product';
                detailsMap.set(conn.productId, {
                    productId: conn.productId,
                    productName: displayName,
                    connections: []
                });
            }
            detailsMap.get(conn.productId).connections.push({
                id: conn.id,
                name: conn.name,
                type: conn.type
            });
        });

        const details = Array.from(detailsMap.values());

        return {
            ...instance,
            linkedResources: {
                productCount: details.length,
                connectionCount: connections.length,
                details: details
            }
        };
    });
  }, [tenant]);

  // Calculate instance and connection counts per product
  const productCounts = useMemo(() => {
    if (!tenant) return new Map<string, { instances: Set<string>, connections: number }>();

    const countsMap = new Map<string, { instances: Set<string>, connections: number }>();

    tenant.tenant_instances?.forEach((instance: any) => {
      (instance.products || instance.tenant_products)?.forEach((product: any) => {
        const productId = product.alpha_product_id;
        if (!countsMap.has(productId)) {
          countsMap.set(productId, { instances: new Set(), connections: 0 });
        }

        const counts = countsMap.get(productId)!;

        (product.tenant_product_datasources || product.datasources || []).forEach((ds: any) => {
          counts.instances.add(instance.id);
          counts.connections++;
        });
      });
    });

    return countsMap;
  }, [tenant]);

  const urlInstanceId = searchParams.get('instanceId');
  const activeInstanceId = urlInstanceId || (scopedDatasource as any)?.alpha_tenant_instance_id || (scopedDatasource as any)?.tenant_instance_id || scopedDatasource?.id || scope?.instanceId || scopedProduct?.tenant_instance_id;
  const activeInstanceName = (scopedDatasource as any)?.instance_name || (scopedDatasource as any)?.display_name || (scopedDatasource as any)?.source_name || (scopedDatasource as any)?.name;

  const activeInstance = useMemo(() => {
    if (!enrichedInstances.length) return null;
    if (activeInstanceId) {
      const found = enrichedInstances.find((i: any) => i.id === activeInstanceId);
      if (found) return found;
    }
    if (activeInstanceName) {
      const foundByName = enrichedInstances.find((i: any) => 
        (i.instance_name && i.instance_name.toLowerCase() === activeInstanceName.toLowerCase()) ||
        (i.display_name && i.display_name.toLowerCase() === activeInstanceName.toLowerCase())
      );
      if (foundByName) return foundByName;
    }
    // Check if any instance contains the scopedDatasource in its nested product datasources
    if (scopedDatasource?.id) {
      const foundByDs = enrichedInstances.find((i: any) => 
        (i.tenant_products || (i as any).products || []).some((p: any) =>
          (p.tenant_product_datasources || p.datasources || []).some((ds: any) => ds.id === scopedDatasource.id)
        )
      );
      if (foundByDs) return foundByDs;
    }
    // Fallback: if only 1 instance exists for the tenant, use it
    if (enrichedInstances.length === 1) {
      return enrichedInstances[0];
    }
    return null;
  }, [enrichedInstances, activeInstanceId, activeInstanceName, scopedDatasource]);

  // Initialize edit form when tenant loads
  React.useEffect(() => {
    if (tenant && editMode) {
      setTenantEditForm({
        display_name: tenant.display_name || '',
        description: (tenant as any).description || '',
        is_active: tenant.is_active || true,
      });
    }
  }, [tenant, editMode]);

  if (!scopedTenant) {
    return <Alert severity="warning">Select a tenant to view its details.</Alert>;
  }

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '400px' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return <Alert severity="error">Failed to load tenant: {error.message}</Alert>;
  }

  if (!tenant) {
    return <Alert severity="error">Tenant not found</Alert>;
  }

  const handleSaveTenantEdit = async () => {
    try {
      await updateTenant.mutate({
        id: tenant.id,
        ...tenantEditForm,
      });
      setEditMode(false);
    } catch (err) {
      console.error('Error updating tenant:', err);
    }
  };

  const handleDeleteTenant = async () => {
    // TODO: Implement delete mutation and navigate back
    setDeleteConfirmOpen(false);
  };

  const handleAddInstance = () => {
    setEditingInstance(null);
    setInstanceForm({
      instance_name: '',
      display_name: '',
      description: '',
      url: '',
      is_active: true,
    });
    setInstanceDialogOpen(true);
  };

  const handleEditInstance = (instance: TenantInstance) => {
    setEditingInstance(instance);
    setInstanceForm({
      instance_name: instance.instance_name || '',
      display_name: instance.display_name || '',
      description: (instance as any).description || '',
      url: (instance as any).url || '',
      is_active: instance.is_active || true,
    });
    setInstanceDialogOpen(true);
  };

  const handleSaveInstance = async () => {
    try {
      if (editingInstance) {
        await updateTenantInstance.mutate({
          id: editingInstance.id,
          ...instanceForm,
        });
      } else {
        await createTenantInstance.mutate({
          ...instanceForm,
        });
      }
      setInstanceDialogOpen(false);
      setEditingInstance(null);
    } catch (err) {
      console.error('Error saving instance:', err);
    }
  };

  const handleDeleteInstance = async (instanceId: string) => {
    try {
      await deleteTenantInstance.mutate({ id: instanceId });
    } catch (err) {
      console.error('Error deleting instance:', err);
    }
  };

  const handleAddConnection = () => {
    setEditingConnection(null);
    setSelectedConnectionProduct(scopedProduct?.id || '');
    setSelectedConnectionInstance(scopedDatasource?.id || '');
    setConnectionForm({
      name: '',
      type: 'postgres',
      host: '',
      port: '5432',
      database: '',
      schema: '',
      username: '',
      secret_path: '',
      base_url: '',
      api_key: '',
      auth_type: 'basic',
      metadata: {},
      is_active: true,
    });
    setConnectionConfigJson('{}');
    setTestConnectionResult(null);
    setConnectionDialogOpen(true);
  };

  const handleEditConnection = (connection: any) => {
    setEditingConnection(connection);
    setSelectedConnectionProduct(connection.linkedProductId || '');
    setSelectedConnectionInstance(connection.linkedInstanceId || '');
    setSelectedAlphaDatasource(connection.linkedAlphaDatasourceId || '');
    setConnectionForm({
      name: connection.name || '',
      type: connection.type || 'postgres',
      host: connection.host || '',
      port: connection.port?.toString() || '5432',
      database: connection.database || '',
      schema: connection.schema || '',
      username: connection.username || '',
      secret_path: connection.secret_path || '',
      base_url: connection.base_url || connection.metadata?.base_url || '',
      api_key: connection.api_key || connection.metadata?.api_key || '',
      auth_type: connection.metadata?.auth_type || 'basic',
      metadata: connection.metadata || {},
      is_active: connection.is_active !== false,
    });
    setConnectionConfigJson(JSON.stringify(connection.metadata || {}));
    setTestConnectionResult(null);
    setConnectionDialogOpen(true);
  };

  const handleTestConnection = async () => {
    try {
      setTestConnectionLoading(true);
      setTestConnectionResult(null);

      const connectionConfig = {
        type: connectionForm.type,
        host: connectionForm.host,
        port: connectionForm.port ? parseInt(connectionForm.port) : undefined,
        database: connectionForm.database,
        schema: connectionForm.schema,
        username: connectionForm.username,
        secret_path: connectionForm.secret_path,
        base_url: connectionForm.base_url,
        api_key: connectionForm.api_key,
        auth_type: connectionForm.auth_type,
        ...connectionForm.metadata,
      };

      Object.keys(connectionConfig).forEach(
        key => connectionConfig[key as keyof typeof connectionConfig] === undefined && delete connectionConfig[key as keyof typeof connectionConfig]
      );

      await testConnection.mutate(connectionConfig);
    } catch (err: any) {
      console.error('Error testing connection:', err);
      setTestConnectionResult({
        success: false,
        message: err.message || 'Failed to test connection',
      });
      setTestConnectionLoading(false);
    }
  };

  const handleSaveConnection = async () => {
    try {
      if (!connectionForm.name || !connectionForm.type) {
        alert('Please fill in required fields: Connection Name and Connection Type');
        return;
      }
      if (!selectedConnectionProduct || !selectedConnectionInstance) {
        alert('Please select both Product and Instance');
        return;
      }

      // Validation: Prevent duplicate connections of the same type for the same product on the same instance
      if (!editingConnection?.id && tenant?.tenant_instances) {
        let isDuplicate = false;
        for (const instance of tenant.tenant_instances) {
          if (instance.id === selectedConnectionInstance) {
            const targetProduct = ((instance.products || instance.tenant_products) || []).find((p: any) => p.alpha_product_id === selectedConnectionProduct);
            if (targetProduct) {
              const dsList = targetProduct.tenant_product_datasources || targetProduct.datasources;
              if (dsList) {
                // If there's already a datasource of this type linked to this product/instance
                const existing = dsList.find((ds: any) => ds.datasource_code?.toLowerCase() === connectionForm.type.toLowerCase() || ds.alpha_datasource?.datasource_type?.toLowerCase() === connectionForm.type.toLowerCase() || (ds.alpha_datasource_id && datasourcesData?.alpha_datasource?.find((a: any) => a.id === ds.alpha_datasource_id)?.datasource_code?.toLowerCase() === connectionForm.type.toLowerCase()));
                if (existing) {
                  isDuplicate = true;
                  break;
                }
              }
            }
          }
        }
        if (isDuplicate) {
          alert(`A ${connectionForm.type} connection already exists for this Product on this Instance. Duplicate connections are not allowed.`);
          return;
        }
      }

      if (!datasourcesData?.alpha_datasource || datasourcesData.alpha_datasource.length === 0) {
        console.warn('Datasource types not yet loaded, attempting to refetch...');
        const freshDatasources = await apiClient<{ alpha_datasource: any[] }>('/api/rest/datasources');
        if (!freshDatasources?.alpha_datasource?.length) {
          alert('Unable to load datasource types. Please try again.');
          return;
        }
      }

      let connectionId;

      if (editingConnection?.id) {
        const updateObject = {
            name: connectionForm.name,
            type: connectionForm.type,
            host: connectionForm.host || null,
            port: connectionForm.port ? parseInt(connectionForm.port) : null,
            database: connectionForm.database || null,
            schema: connectionForm.schema || null,
            username: connectionForm.username || null,
            secret_path: connectionForm.secret_path || null,
            metadata: {
              auth_type: connectionForm.auth_type,
              base_url: connectionForm.base_url,
              api_key: connectionForm.api_key,
              product_id: selectedConnectionProduct,
              ...connectionForm.metadata,
            },
            is_active: connectionForm.is_active,
        };

        const updateResult = await apiClient(`/api/tenant-ops/connections/${editingConnection.id}`, {
          method: 'PATCH',
          body: JSON.stringify(updateObject),
        });
        connectionId = (updateResult as any).id;
      } else {
        const createObject = {
            name: connectionForm.name,
            type: connectionForm.type,
            host: connectionForm.host || null,
            port: connectionForm.port ? parseInt(connectionForm.port) : null,
            database: connectionForm.database || null,
            schema: connectionForm.schema || null,
            username: connectionForm.username || null,
            secret_path: connectionForm.secret_path || null,
            metadata: {
              auth_type: connectionForm.auth_type,
              base_url: connectionForm.base_url,
              api_key: connectionForm.api_key,
              product_id: selectedConnectionProduct,
              ...connectionForm.metadata,
            },
            is_active: connectionForm.is_active,
        };

        const createResult = await createConnection.mutate(createObject);
        connectionId = (createResult as any).id;
      }

      if (connectionId) {
        if (tenant?.tenant_instances) {
          for (const instance of tenant.tenant_instances) {
            for (const product of ((instance.products || instance.tenant_products) || [])) {
              for (const ds of (product.tenant_product_datasources || product.datasources || [])) {
                if (ds.connection_id === connectionId) {
                  await apiClient(`/api/tenant-ops/product-datasources/${ds.id}/connection`, {
                    method: 'PATCH',
                    body: JSON.stringify({ connection_id: null }),
                  });
                }
              }
            }
          }
        }

        let tenantProductId = selectedConnectionProduct;
        let existingProduct = null;
        if (tenant?.tenant_instances) {
          for (const instance of tenant.tenant_instances) {
            existingProduct = ((instance.products || instance.tenant_products) || []).find(
              (p: any) => p.alpha_product_id === selectedConnectionProduct
            );
            if (existingProduct) break;
          }
        }

        if (!existingProduct) {
          const registerResult = await addTenantProduct.mutate({
            alpha_product_id: selectedConnectionProduct,
            tenant_instance_id: selectedConnectionInstance,
            version: 1.0,
            is_active: true,
          });
          tenantProductId = (registerResult as any).id;
          await refetchTenant();
        } else {
          tenantProductId = existingProduct.id;
        }

        if (tenantProductId) {
          let resolvedAlphaDatasourceId = selectedAlphaDatasource;

          if (!resolvedAlphaDatasourceId && (datasourcesData as any)?.alpha_datasource?.length > 0) {
            const typeStr = connectionForm.type.toLowerCase();
            const match = (datasourcesData as any).alpha_datasource.find((ds: any) =>
              ds.datasource_code?.toLowerCase().includes(typeStr)
            );
            if (match) {
              resolvedAlphaDatasourceId = match.id;
            } else {
              const fallback = (datasourcesData as any).alpha_datasource[0];
              resolvedAlphaDatasourceId = fallback?.id;
              console.warn(`Could not match datasource type for ${connectionForm.type}; using ${fallback?.datasource_code}`);
            }
          }

          if (!resolvedAlphaDatasourceId) {
            console.error('Datasource type resolution failed:', {
              selected: selectedAlphaDatasource,
              connectionType: connectionForm.type,
              availableDatasources: datasourcesData?.alpha_datasource || [],
              availableCount: datasourcesData?.alpha_datasource?.length || 0,
            });
            alert('Unable to link connection: datasource type could not be determined. Please ensure Datasource Types are loaded and try again.');
            return;
          }

          const freshTenantData = await apiClient<{ tenant: any }>(`/api/tenants/${tenantId}`);
          const freshTenant = freshTenantData?.tenant;

          let existingDatasource = null;

          if (freshTenant?.tenant_instances) {
            for (const instance of freshTenant.tenant_instances) {
              const targetProduct = ((instance.products || instance.tenant_products) || []).find((p: any) => p.id === tenantProductId);
              const dsList = targetProduct.tenant_product_datasources || targetProduct.datasources;
              if (dsList) {
                existingDatasource = dsList.find(
                    (ds: any) => {
                       return ds.tenant_instance_id === selectedConnectionInstance &&
                       ds.alpha_datasource_id === resolvedAlphaDatasourceId;
                    }
                );
              }
              if (existingDatasource) break;
            }
          }

          if (existingDatasource) {
            if (!connectionId) {
                console.error("Critical: connectionId is missing during linking phase");
                alert("Internal Error: Connection ID is missing. Please refresh and try again.");
                return;
            }

            if (!resolvedAlphaDatasourceId) {
                alert('Unable to link connection: missing datasource type.');
                return;
            }

            const verifyConn = await apiClient(`/api/tenant-ops/connections/${connectionId}`, { method: 'GET' });
            if (!verifyConn) {
                console.error("Critical: Connection created/updated but not found in subsequent query.", connectionId);
                alert("Warning: Connection saved, but not visible yet. Linking to product skipped to prevent errors.");
                return;
            }

            const updateVars = {
              tenant_instance_id: selectedConnectionInstance || null,
              alpha_datasource_id: resolvedAlphaDatasourceId || null,
              connection_id: connectionId,
              is_active: existingDatasource.is_active,
              source_name: connectionForm.name,
              config: existingDatasource.config,
            };

            try {
                await apiClient(`/api/tenant-ops/product-datasources/${existingDatasource.id}/linking`, {
                  method: 'PATCH',
                  body: JSON.stringify(updateVars),
                });
            } catch (innerErr) {
                console.error("Failed to update TPD link:", innerErr);
                alert("Connection saved, but failed to link to Product. Please try linking again.");
            }
          } else {
            if (!selectedConnectionInstance) {
              alert('Unable to link connection: missing instance.');
              return;
            }
            if (!resolvedAlphaDatasourceId) {
              alert('Unable to link connection: missing datasource type.');
              return;
            }

            await addTenantProductDatasource.mutate({
              tenant_product_id: tenantProductId,
              tenant_instance_id: selectedConnectionInstance,
              alpha_datasource_id: resolvedAlphaDatasourceId,
              config: {},
              is_active: true,
              source_name: connectionForm.name,
              connection_id: connectionId,
            });
          }
        }

        setConnectionDialogOpen(false);
        setEditingConnection(null);
        setConnectionForm({
          name: '',
          type: 'postgres',
          host: '',
          port: '5432',
          database: '',
          schema: '',
          username: '',
          secret_path: '',
          base_url: '',
          api_key: '',
          auth_type: 'basic',
          metadata: {},
          is_active: true,
        });
        setConnectionConfigJson('{}');
        setSelectedConnectionProduct('');
        setSelectedConnectionInstance('');
        setConnectionsRefreshKey(prev => prev + 1);
        setSelectedAlphaDatasource('');
      }
    } catch (err: any) {
      console.error('Error saving connection:', err);
      alert(`Failed to save connection: ${err.message || 'Unknown error'}`);
    }
  };

  return (
    <Box sx={{ p: { xs: 2, md: 3 } }}>
      {/* Breadcrumb */}
      <Breadcrumbs sx={{ mb: 3 }}>
        <Typography color="textPrimary">
          {tenant.display_name || tenant.name || 'Tenant Details'}
        </Typography>
      </Breadcrumbs>

      {/* Tenant Header Card */}
      <Card sx={{ mb: 3 }}>
        <Box sx={{ p: 3 }}>
          {!editMode ? (
            // View Mode
            <>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 2 }}>
                <Box>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 1 }}>
                    <Typography variant="h4" sx={{ fontWeight: 'bold' }}>
                      {tenant.display_name || tenant.name || 'Unnamed Tenant'}
                    </Typography>
                    {tenant.gold_copy && (
                      <Chip
                        label="Gold Copy"
                        color="warning"
                        size="small"
                        sx={{ fontWeight: 'bold' }}
                      />
                    )}
                    {(tenant as any).tier && (
                      <Chip
                        label={(tenant as any).tier.toUpperCase()}
                        color="warning"
                        size="small"
                        variant="outlined"
                      />
                    )}
                  </Box>
                  <Typography variant="body2" color="textSecondary" sx={{ mb: 2 }}>
                    {(tenant as any).description || 'No description provided'}
                  </Typography>
                  <Stack direction="row" spacing={3} sx={{ flexWrap: 'wrap' }}>
                    <Box>
                      <Typography variant="caption" color="textSecondary">
                        Tenant ID
                      </Typography>
                      <Typography
                        variant="body2"
                        sx={{ fontFamily: 'monospace', fontWeight: 500 }}
                      >
                        {tenant.id}
                      </Typography>
                    </Box>
                    <Box>
                      <Typography variant="caption" color="textSecondary">
                        Created
                      </Typography>
                      <Typography variant="body2">
                        {tenant.created_at
                          ? new Date(tenant.created_at).toLocaleDateString()
                          : 'N/A'}
                      </Typography>
                    </Box>
                    <Box>
                      <Typography variant="caption" color="textSecondary">
                        Status
                      </Typography>
                      <Chip
                        label={tenant.is_active ? 'Active' : 'Inactive'}
                        color={tenant.is_active ? 'success' : 'default'}
                        size="small"
                        sx={{ mt: 0.5 }}
                      />
                    </Box>
                  </Stack>
                </Box>
                <Stack direction="row" spacing={1} sx={{ flexWrap: 'wrap' }}>
                  <Button
                    variant="outlined"
                    startIcon={<EditIcon />}
                    onClick={() => setEditMode(true)}
                  >
                    Edit
                  </Button>
                  <span title={tenant.gold_copy ? "Gold Copy tenants cannot be deleted" : "Delete tenant"}>
                    <Button
                      variant="outlined"
                      color="error"
                      startIcon={<DeleteIcon />}
                      onClick={() => setDeleteConfirmOpen(true)}
                      disabled={tenant.gold_copy}
                    >
                      Delete
                    </Button>
                  </span>
                </Stack>
              </Box>
            </>
          ) : (
            // Edit Mode
            <Box>
              <Typography variant="h6" sx={{ fontWeight: 'bold', mb: 2 }}>
                Edit Tenant
              </Typography>
              <Stack spacing={2}>
                <TextField
                  label="Display Name"
                  value={tenantEditForm.display_name}
                  onChange={(e) =>
                    setTenantEditForm({
                      ...tenantEditForm,
                      display_name: e.target.value,
                    })
                  }
                  fullWidth
                />
                <TextField
                  label="Description"
                  value={tenantEditForm.description}
                  onChange={(e) =>
                    setTenantEditForm({
                      ...tenantEditForm,
                      description: e.target.value,
                    })
                  }
                  fullWidth
                  multiline
                  rows={3}
                />
                <FormControlLabel
                  control={
                    <Switch
                      checked={tenantEditForm.is_active}
                      onChange={(e) =>
                        setTenantEditForm({
                          ...tenantEditForm,
                          is_active: e.target.checked,
                        })
                      }
                    />
                  }
                  label="Active"
                />
                <Stack direction="row" spacing={1}>
                  <Button
                    variant="contained"
                    onClick={handleSaveTenantEdit}
                  >
                    Save Changes
                  </Button>
                  <Button
                    variant="outlined"
                    onClick={() => setEditMode(false)}
                  >
                    Cancel
                  </Button>
                </Stack>
              </Stack>
            </Box>
          )}
        </Box>
      </Card>

      {/* Tabs */}
      <Card>
        <Tabs
          value={activeTab}
          onChange={(_, newValue) => setActiveTab(newValue)}
          sx={{
            borderBottom: '1px solid',
            borderColor: 'divider',
            px: 3,
          }}
        >
          <Tab label="Instances" id="tenant-tab-0" aria-controls="tenant-tabpanel-0" />
          <Tab label="Products" id="tenant-tab-1" aria-controls="tenant-tabpanel-1" />
          <Tab label="Connections" id="tenant-tab-2" aria-controls="tenant-tabpanel-2" />
          <Tab label="Lookups" id="tenant-tab-3" aria-controls="tenant-tabpanel-3" />
          <Tab label="Abbreviations" id="tenant-tab-4" aria-controls="tenant-tabpanel-4" />
          <Tab label="Audit Log" id="tenant-tab-5" aria-controls="tenant-tabpanel-5" />
          <Tab label="Configuration" id="tenant-tab-6" aria-controls="tenant-tabpanel-6" />
        </Tabs>

        {/* Instances Tab */}
        <TabPanel value={activeTab} index={0}>
          <Box sx={{ p: 3 }}>
            {activeInstance && (
              <Box sx={{ mb: 3 }}>
                <Typography variant="h6" sx={{ fontWeight: 'bold', mb: 1 }}>
                  Scoped Instance Detail
                </Typography>
                <ScopedInstanceEditor
                  instance={activeInstance}
                  tenantId={tenantId}
                  updateMutation={(data: any) => updateTenantInstance.mutate(data).then(() => refetchTenant())}
                />
              </Box>
            )}

            <Box sx={{ mt: activeInstance ? 4 : 0 }}>
              {activeInstance && (
                <Typography variant="subtitle1" sx={{ fontWeight: 'bold', mb: 1 }}>
                  All Tenant Instances ({enrichedInstances.length})
                </Typography>
              )}
              <InstancesTableV2
                instances={enrichedInstances}
                tenantId={tenantId || ''}
                onAddInstance={handleAddInstance}
                onEditInstance={handleEditInstance}
                onDeleteInstance={handleDeleteInstance}
                onReload={refetchTenant}
              />
            </Box>
          </Box>
        </TabPanel>

        {/* Products Tab */}
        <TabPanel value={activeTab} index={1}>
          <Box sx={{ p: 3 }}>
            <ProductsTabContent 
              tenantId={tenantId || ''}
              activeInstanceId={activeInstanceId}
              datasourceId={scopedDatasource?.id || ''}
              productCounts={productCounts}
              onProductInstancesClick={(productId) => {
                setSelectedProductForCounts(productId);
                setActiveTab(0);
              }}
              onProductConnectionsClick={(productId) => {
                setSelectedProductForCounts(productId);
                setActiveTab(2);
              }}
            />
          </Box>
        </TabPanel>

        {/* Connections Tab */}
        <TabPanel value={activeTab} index={2}>
          <Box sx={{ display: 'flex', gap: 3, p: 3 }}>
            {/* Facets Sidebar */}
            <ConnectionsFacets
              instances={instances}
              products={Array.from(
                new Map(
                  (tenant?.tenant_instances || []).flatMap((i: any) => {
                    return ((i.products || i.tenant_products) || []).map((p: any) => [
                      p.alpha_product_id,
                      {
                        id: p.alpha_product_id,
                        product_name: p.alpha_product?.product_name || p.product?.product_name || 'Unknown Product'
                      }
                    ]);
                  })
                ).values()
              ) as Array<{ id: string; product_name: string }>}
              selectedInstances={selectedInstanceFilters}
              selectedProducts={selectedProductFilters}
              onInstanceChange={setSelectedInstanceFilters}
              onProductChange={setSelectedProductFilters}
            />
            
            {/* Connections Content */}
            <Box sx={{ flex: 1 }}>
              <ConnectionsTabContent 
                key={`${tenant?.id || 'connections-tab'}-${connectionsRefreshKey}`}
                tenantId={tenant?.id || tenantId || ''}
                datasourceId={scopedDatasource?.id || ''}
                isGoldCopy={tenant?.gold_copy || false}
                instanceFilter={selectedInstanceFilters.length > 0 ? selectedInstanceFilters : null}
                productFilter={selectedProductFilters.length > 0 ? selectedProductFilters : null}
                onAddConnection={handleAddConnection}
                onEditConnection={handleEditConnection}
                tenantData={tenantData?.tenant}
              />
            </Box>
          </Box>
        </TabPanel>

        {/* Lookups Tab */}
        <TabPanel value={activeTab} index={3}>
          <Box sx={{ p: 3 }}>
            <LookupsManagementTab 
              tenantId={scopedTenant?.id || ''} 
              instanceFilter={null}
            />
          </Box>
        </TabPanel>

        {/* Abbreviations Tab */}
        <TabPanel value={activeTab} index={4}>
          <Box sx={{ p: 3 }}>
            <AbbreviationsTab tenantId={scopedTenant?.id || ''} />
          </Box>
        </TabPanel>

        {/* Audit Log Tab */}
        <TabPanel value={activeTab} index={5}>
          <Box sx={{ p: 3 }}>
            <AuditLogTabContent 
              tenantId={scopedTenant?.id || ''}
              datasourceId={scopedDatasource?.id || ''}
            />
          </Box>
        </TabPanel>

        {/* Configuration Tab */}
        <TabPanel value={activeTab} index={6}>
          <Box sx={{ p: 3 }}>
            <ConfigurationTabContent 
              tenantId={scopedTenant?.id || ''}
              datasourceId={scopedDatasource?.id || ''}
            />
          </Box>
        </TabPanel>
      </Card>

      {/* Instance Dialog */}
      <Dialog open={instanceDialogOpen} onClose={() => setInstanceDialogOpen(false)} maxWidth="sm" fullWidth disableEnforceFocus>
        <DialogTitle>
          {editingInstance ? 'Edit Instance' : 'Add Instance'}
        </DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 2 }}>
            <TextField
              label="Instance Name"
              value={instanceForm.instance_name}
              onChange={(e) =>
                setInstanceForm({ ...instanceForm, instance_name: e.target.value })
              }
              fullWidth
            />
            <TextField
              label="Display Name"
              value={instanceForm.display_name}
              onChange={(e) =>
                setInstanceForm({ ...instanceForm, display_name: e.target.value })
              }
              fullWidth
            />
            <TextField
              label="Description"
              value={instanceForm.description}
              onChange={(e) =>
                setInstanceForm({ ...instanceForm, description: e.target.value })
              }
              fullWidth
              multiline
              rows={2}
            />
            <TextField
              label="URL"
              value={instanceForm.url}
              onChange={(e) =>
                setInstanceForm({ ...instanceForm, url: e.target.value })
              }
              fullWidth
            />
            <FormControlLabel
              control={
                <Switch
                  checked={instanceForm.is_active}
                  onChange={(e) =>
                    setInstanceForm({ ...instanceForm, is_active: e.target.checked })
                  }
                />
              }
              label="Active"
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setInstanceDialogOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleSaveInstance}>
            {editingInstance ? 'Update' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteConfirmOpen} onClose={() => setDeleteConfirmOpen(false)} disableEnforceFocus>
        <DialogTitle>Delete Tenant</DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to delete "{tenant.display_name || tenant.name}"?
            This action cannot be undone and will affect all associated instances and data.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteConfirmOpen(false)}>Cancel</Button>
          <Button
            onClick={handleDeleteTenant}
            color="error"
            variant="contained"
          >
            Delete
          </Button>
        </DialogActions>
      </Dialog>

      {/* Add Connection Dialog */}
      <Dialog 
        open={connectionDialogOpen} 
        onClose={() => setConnectionDialogOpen(false)}
        maxWidth="md"
        fullWidth
        disableEnforceFocus
      >
        <DialogTitle>{editingConnection ? 'Edit Connection' : 'Add New Connection'}</DialogTitle>
        <DialogContent sx={{ pt: 2, maxHeight: '80vh', overflow: 'auto' }}>
          <form onSubmit={(e) => e.preventDefault()}>
            <Stack spacing={2}>
            {/* Logic for disabled fields: If not gold copy AND has core_id, it is derived/inherited */}
            {(() => {
              const isDerived = !tenant?.gold_copy && !!editingConnection?.core_id;
              
              return (
                <>
            {/* Basic Information */}
            <Typography variant="subtitle2" sx={{ fontWeight: 'bold', mt: 2 }}>
              Connection Information
            </Typography>
            
            <TextField
              label="Connection Name"
              value={connectionForm.name}
              onChange={(e) =>
                setConnectionForm({ ...connectionForm, name: e.target.value })
              }
              fullWidth
              required
              disabled={isDerived}
              helperText={isDerived ? "Inherited from Gold Copy - cannot rename" : "Unique name for this connection"}
            />

            {/* Product and Instance Selection */}
            <FormControl fullWidth required disabled={isDerived}>
              <InputLabel>Product</InputLabel>
              <Select
                value={selectedConnectionProduct}
                onChange={(e) => setSelectedConnectionProduct(e.target.value)}
                label="Product"
              >
                {alphaProductsData?.alpha_product?.map((product: any) => {
                  // Check if this product is already registered
                  const isRegistered = (tenant?.tenant_instances || []).some(
                    (i: any) => ((i.products || i.tenant_products) || []).some(
                      (p: any) => p.alpha_product_id === product.id
                    )
                  );
                  return (
                    <MenuItem key={product.id} value={product.id}>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, width: '100%' }}>
                        {product.product_name}
                        {isRegistered && (
                          <Chip 
                            label="Registered" 
                            size="small" 
                            color="primary"
                            sx={{ height: 20, fontSize: '0.65rem' }} 
                          />
                        )}
                      </Box>
                    </MenuItem>
                  );
                })}
              </Select>
              <Typography variant="caption" sx={{ mt: 0.5, color: 'text.secondary' }}>
                Select the product this connection will serve
              </Typography>
            </FormControl>

            <FormControl fullWidth required disabled>
              <InputLabel>Instance</InputLabel>
              <Select
                value={selectedConnectionInstance}
                onChange={(e) => setSelectedConnectionInstance(e.target.value)}
                label="Instance"
              >
                {instances.map((instance: any) => (
                  <MenuItem key={instance.id} value={instance.id}>
                    {instance.display_name || instance.instance_name}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>

            {/* Datasource Type removed as it is redundant with Connection Type */}

            <FormControl fullWidth required disabled={isDerived}>
              <InputLabel>Connection Type</InputLabel>
              <Select
                value={connectionForm.type}
                onChange={(e) =>
                  setConnectionForm({ ...connectionForm, type: e.target.value })
                }
                label="Connection Type"
              >
                <MenuItem value="postgres">PostgreSQL</MenuItem>
                <MenuItem value="mysql">MySQL</MenuItem>
                <MenuItem value="snowflake">Snowflake</MenuItem>
                <MenuItem value="api">REST API</MenuItem>
                <MenuItem value="s3">S3 Storage</MenuItem>
                <MenuItem value="azure">Azure Storage</MenuItem>
                <MenuItem value="gcs">Google Cloud Storage</MenuItem>
              </Select>
            </FormControl>

            {/* Authentication Type */}
            <FormControl fullWidth>
              <InputLabel>Authentication Method</InputLabel>
              <Select
                value={connectionForm.auth_type}
                onChange={(e) =>
                  setConnectionForm({ ...connectionForm, auth_type: e.target.value })
                }
                label="Authentication Method"
              >
                <MenuItem value="basic">Basic (Username/Password)</MenuItem>
                <MenuItem value="bearer">Bearer Token</MenuItem>
                <MenuItem value="api_key">API Key</MenuItem>
                <MenuItem value="oauth">OAuth 2.0</MenuItem>
                <MenuItem value="key_pair">Key Pair (SSH/TLS)</MenuItem>
                <MenuItem value="iam">IAM Role</MenuItem>
              </Select>
            </FormControl>

            {/* Database Connection Fields */}
            {['postgres', 'mysql', 'snowflake'].includes(connectionForm.type) && (
              <>
                <Typography variant="subtitle2" sx={{ fontWeight: 'bold', mt: 2 }}>
                  Database Configuration
                </Typography>
                  <TextField
                    label="Host"
                    value={connectionForm.host}
                    onChange={(e) =>
                      setConnectionForm({ ...connectionForm, host: e.target.value })
                    }
                    fullWidth
                    placeholder="e.g., db.example.com"
                  />
                  <TextField
                    label="Port"
                    type="number"
                    value={connectionForm.port}
                    onChange={(e) =>
                      setConnectionForm({ ...connectionForm, port: e.target.value })
                    }
                    fullWidth
                    placeholder={connectionForm.type === 'postgres' ? '5432' : '3306'}
                  />
                  <TextField
                    label="Database"
                    value={connectionForm.database}
                    onChange={(e) =>
                      setConnectionForm({ ...connectionForm, database: e.target.value })
                    }
                    fullWidth
                    placeholder="Database name"
                  />
                  <TextField
                    label="Schema"
                    value={connectionForm.schema}
                    onChange={(e) =>
                      setConnectionForm({ ...connectionForm, schema: e.target.value })
                    }
                    fullWidth
                    placeholder="e.g., public (optional)"
                  />
              </>
            )}

            {/* API Connection Fields */}
            {connectionForm.type === 'api' && (
              <>
                <Typography variant="subtitle2" sx={{ fontWeight: 'bold', mt: 2 }}>
                  API Configuration
                </Typography>
                  <TextField
                    label="Base URL"
                    value={connectionForm.base_url}
                    onChange={(e) =>
                      setConnectionForm({ ...connectionForm, base_url: e.target.value })
                    }
                    fullWidth
                    placeholder="e.g., https://api.example.com/v1"
                  />
              </>
            )}

            {/* Authentication Credentials */}
            <Typography variant="subtitle2" sx={{ fontWeight: 'bold', mt: 2 }}>
              Authentication Credentials
            </Typography>

            {['basic', 'bearer', 'api_key'].includes(connectionForm.auth_type) && (
              <>
                {connectionForm.auth_type === 'basic' && (
                  <>
                    <TextField
                      label="Username"
                      value={connectionForm.username}
                      onChange={(e) =>
                        setConnectionForm({ ...connectionForm, username: e.target.value })
                      }
                      fullWidth
                    />
                    <TextField
                      label="Infisical Secret Path"
                      value={connectionForm.secret_path}
                      onChange={(e) =>
                        setConnectionForm({ ...connectionForm, secret_path: e.target.value })
                      }
                      fullWidth
                      placeholder="/connections/tenant-<tenant_id>/<connection_name>"
                      helperText="Infisical path where DB_USERNAME and DB_PASSWORD are stored"
                    />
                  </>
                )}
                {connectionForm.auth_type === 'bearer' && (
                  <TextField
                    label="Bearer Token"
                    type="password"
                    value={connectionForm.api_key}
                    onChange={(e) =>
                      setConnectionForm({ ...connectionForm, api_key: e.target.value })
                    }
                    fullWidth
                    multiline
                    rows={2}
                  />
                )}
                {connectionForm.auth_type === 'api_key' && (
                  <TextField
                    label="API Key"
                    type="password"
                    value={connectionForm.api_key}
                    onChange={(e) =>
                      setConnectionForm({ ...connectionForm, api_key: e.target.value })
                    }
                    fullWidth
                    multiline
                    rows={2}
                  />
                )}
              </>
            )}

            {connectionForm.auth_type === 'oauth' && (
              <>
                <TextField
                  label="Client ID"
                  value={connectionForm.metadata?.client_id || ''}
                  onChange={(e) =>
                    setConnectionForm({
                      ...connectionForm,
                      metadata: { ...connectionForm.metadata, client_id: e.target.value }
                    })
                  }
                  fullWidth
                />
                <TextField
                  label="Client Secret"
                  type="password"
                  value={connectionForm.metadata?.client_secret || ''}
                  onChange={(e) =>
                    setConnectionForm({
                      ...connectionForm,
                      metadata: { ...connectionForm.metadata, client_secret: e.target.value }
                    })
                  }
                  fullWidth
                />
                <TextField
                  label="Authorization URL"
                  value={connectionForm.metadata?.auth_url || ''}
                  onChange={(e) =>
                    setConnectionForm({
                      ...connectionForm,
                      metadata: { ...connectionForm.metadata, auth_url: e.target.value }
                    })
                  }
                  fullWidth
                />
                <TextField
                  label="Token URL"
                  value={connectionForm.metadata?.token_url || ''}
                  onChange={(e) =>
                    setConnectionForm({
                      ...connectionForm,
                      metadata: { ...connectionForm.metadata, token_url: e.target.value }
                    })
                  }
                  fullWidth
                />
              </>
            )}

            {connectionForm.auth_type === 'key_pair' && (
              <>
                <TextField
                  label="Private Key"
                  type="password"
                  value={connectionForm.metadata?.private_key || ''}
                  onChange={(e) =>
                    setConnectionForm({
                      ...connectionForm,
                      metadata: { ...connectionForm.metadata, private_key: e.target.value }
                    })
                  }
                  fullWidth
                  multiline
                  rows={4}
                  placeholder="Paste your private key (PEM format)"
                />
                <TextField
                  label="Key Passphrase (Optional)"
                  type="password"
                  value={connectionForm.metadata?.key_passphrase || ''}
                  onChange={(e) =>
                    setConnectionForm({
                      ...connectionForm,
                      metadata: { ...connectionForm.metadata, key_passphrase: e.target.value }
                    })
                  }
                  fullWidth
                />
              </>
            )}

            {/* Advanced Options */}
            <Typography variant="subtitle2" sx={{ fontWeight: 'bold', mt: 2 }}>
              Advanced Options
            </Typography>

            <TextField
              label="Connection Configuration (JSON)"
              value={connectionConfigJson}
              onChange={(e) => {
                setConnectionConfigJson(e.target.value);
                try {
                  const parsed = JSON.parse(e.target.value);
                  setConnectionForm({ ...connectionForm, metadata: parsed });
                } catch (e) {
                  // Allow invalid JSON while typing
                }
              }}
              fullWidth
              multiline
              rows={4}
              placeholder='{"ssl_mode": "require", "connection_timeout": 30, "pool_size": 10}'
              helperText="Additional connection metadata as JSON (ssl_mode, connection_timeout, pool_size, etc.)"
            />

            {/* Test Connection Section */}
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mt: 2 }}>
              <Button
                variant="outlined"
                onClick={handleTestConnection}
                disabled={testConnectionLoading || !connectionForm.type}
              >
                {testConnectionLoading ? (
                  <>
                    <CircularProgress size={16} sx={{ mr: 1 }} />
                    Testing...
                  </>
                ) : (
                  'Test Connection'
                )}
              </Button>
              {testConnectionResult && (
                <Chip
                  label={testConnectionResult.success ? 'Connection Successful' : 'Connection Failed'}
                  color={testConnectionResult.success ? 'success' : 'error'}
                  variant="outlined"
                />
              )}
            </Box>
            {testConnectionResult && !testConnectionResult.success && (
              <Alert severity="error" sx={{ mt: 1 }}>
                {testConnectionResult.message}
              </Alert>
            )}
            {testConnectionResult && testConnectionResult.success && (
              <Alert severity="success" sx={{ mt: 1 }}>
                {testConnectionResult.message || 'Connection test successful!'}
              </Alert>
            )}

            <FormControlLabel
              control={
                <Switch
                  checked={connectionForm.is_active}
                  onChange={(e) =>
                    setConnectionForm({ ...connectionForm, is_active: e.target.checked })
                  }
                  disabled={isDerived}
                />
              }
              label="Active"
            />
            </>
            );
            })()}
          </Stack>
          </form>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConnectionDialogOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleSaveConnection}>
            {editingConnection ? 'Update Connection' : 'Create Connection'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default TenantDetailPageV2;
