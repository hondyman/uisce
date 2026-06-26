import React, { useState, useMemo, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, gql, useApolloClient } from '@apollo/client';
import {
  Box,
  Button,
  Card,
  CircularProgress,
  Alert,
  Typography,
  Breadcrumbs,
  Link,
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
import { devLog, devWarn, devError } from '../../../utils/devLogger';
import { GET_SCOPED_TENANT } from '../../../graphql/queries/tenantQueries';
import { GET_AVAILABLE_DATASOURCES } from '../../../graphql/queries/datasourceQueries';
import {
  UPDATE_TENANT,
  CREATE_TENANT_INSTANCE,
  UPDATE_TENANT_INSTANCE,
  DELETE_TENANT_INSTANCE,
  CREATE_CONNECTION,
  UPDATE_CONNECTION,
  TEST_DATASOURCE_CONNECTION,
  ADD_TENANT_PRODUCT_DATASOURCE,
  UPDATE_TENANT_PRODUCT_DATASOURCE,
  UPDATE_TENANT_PRODUCT_DATASOURCE_LINKING,
  UPDATE_TPD_CONNECTION_ONLY,
} from '../../../graphql/mutations/tenantMutations';
import { useTenant } from '../../../contexts/TenantContext';
import type { TenantInstance } from '../../../types';
import InstancesTableV2 from '../components/InstancesTableV2';
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
  const client = useApolloClient();
  const { 
    tenant: scopedTenant, 
    datasource: scopedDatasource,
    setSelection
  } = useTenant();



  const { loading, error, data, refetch } = useQuery(GET_SCOPED_TENANT, {
    variables: { tenantId: tenantId ?? '' },
    skip: !tenantId,
  });

  // Fetch available alpha datasources - critical for connection linking, load eagerly
  const { data: datasourcesData, loading: datasourcesLoading } = useQuery(GET_AVAILABLE_DATASOURCES, {
    fetchPolicy: 'cache-first',  // Cache aggressively to avoid repeated loads
  });

  // Fetch all available alpha products
  const { data: alphaProductsData } = useQuery(gql`
    query GetAlphaProducts {
      alpha_product(where: { is_active: { _eq: true } }, order_by: { product_name: asc }) {
        id
        product_name
        product_code
        is_active
      }
    }
  `);

  const [updateTenant] = useMutation(UPDATE_TENANT, {
    onCompleted: () => refetch(),
  });
  const [addTenantProduct] = useMutation(gql`
    mutation AddTenantProduct($tenant_id: uuid!, $tenant_instance_id: uuid!, $alpha_product_id: uuid!, $version: Float!, $is_active: Boolean!) {
      insert_tenant_product_one(object: { tenant_id: $tenant_id, tenant_instance_id: $tenant_instance_id, alpha_product_id: $alpha_product_id, version: $version, is_active: $is_active }) {
        id
      }
    }
  `, {
    onCompleted: () => refetch(),
  });

  const [createTenantInstance] = useMutation(CREATE_TENANT_INSTANCE, {
    onCompleted: () => refetch(),
  });
  const [updateTenantInstance] = useMutation(UPDATE_TENANT_INSTANCE, {
    onCompleted: () => refetch(),
  });
  const [deleteTenantInstance] = useMutation(DELETE_TENANT_INSTANCE, {
    onCompleted: () => refetch(),
  });
  const [createConnection] = useMutation(CREATE_CONNECTION, {
    onCompleted: () => {
      refetch();
    },
  });
  const [updateConnection] = useMutation(UPDATE_CONNECTION, {
    onCompleted: () => {
      refetch();
    },
  });
  const [addTenantProductDatasource] = useMutation(ADD_TENANT_PRODUCT_DATASOURCE, {
    onCompleted: () => {
      refetch();
    },
  });
  const [updateTenantProductDatasource] = useMutation(UPDATE_TENANT_PRODUCT_DATASOURCE, {
    onCompleted: () => {
      refetch();
    },
  });
  const [updateTenantProductDatasourceLinking] = useMutation(UPDATE_TENANT_PRODUCT_DATASOURCE_LINKING, {
    onCompleted: () => {
      refetch();
    },
  });
  const [updateTpdConnectionOnly] = useMutation(UPDATE_TPD_CONNECTION_ONLY, {
    onCompleted: () => {
      refetch();
    },
  });
  const [testConnection] = useMutation(TEST_DATASOURCE_CONNECTION, {
    onCompleted: (data) => {
      setTestConnectionResult(data.test_datasource_connection);
      setTestConnectionLoading(false);
    },
    onError: (err) => {
      setTestConnectionResult({
        success: false,
        message: err.message || 'Failed to test connection',
      });
      setTestConnectionLoading(false);
    },
  });

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
    password: '',
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

  const tenant = useMemo(() => data?.tenants?.[0] ?? null, [data]);
  const instances = useMemo(() => tenant?.tenant_instances ?? [], [tenant]);

  // Automatically switch context to this tenant if not already selected
  useEffect(() => {
    if (tenant && tenant.id) {
       // Only switch if we are NOT already on this tenant.
       // scopedTenant might be null on first load, so we check ID match.
       if (!scopedTenant || scopedTenant.id !== tenant.id) {
            console.log('Switching tenant context to:', tenant.display_name);
            localStorage.setItem('selected_tenant', JSON.stringify(tenant));
            // Triggers for Apollo Client next request
       }
    }
  }, [tenant, scopedTenant]);

  const enrichedInstances = useMemo(() => {
    if (!tenant) return [];
    
    // Map instance ID to associated resources
    const instanceResourcesMap = new Map<string, { products: string[], connections: any[] }>();
    
    // Iterate over tenant products to find linked datasources
    tenant.tenant_products?.forEach((tp: any) => {
      tp.tenant_product_datasources?.forEach((tpd: any) => {
        // Only count datasources that have an actual connection_id assigned
        if (tpd.tenant_instance_id && tpd.connection_id) {
            // Found a datasource linked to an instance with an active connection
            if (!instanceResourcesMap.has(tpd.tenant_instance_id)) {
                instanceResourcesMap.set(tpd.tenant_instance_id, { products: [], connections: [] });
            }
            const resource = instanceResourcesMap.get(tpd.tenant_instance_id)!;
            
            // Add product details (if not already added)
            // Wait, we just need the product info for grouping
            
            resource.connections.push({
                id: tpd.connection_id || tpd.id, 
                name: tpd.source_name || 'Unknown',
                type: 'Datasource',
                productName: tp.alpha_product?.product_name,
                productId: tp.alpha_product_id
            });
        }
      });
    });

    return (tenant.tenant_instances ?? []).map((instance: any) => {
        const resources = instanceResourcesMap.get(instance.id);
        const connections = resources?.connections || [];
        
        // Group by product for the dialog details
        const detailsMap = new Map<string, any>();
        connections.forEach((conn: any) => {
            if (!detailsMap.has(conn.productId)) {
                // Use productName if available, otherwise use productId as fallback
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
    
    // Iterate over tenant products to count instances and connections per product
    tenant.tenant_products?.forEach((tp: any) => {
      const productId = tp.alpha_product_id;
      if (!countsMap.has(productId)) {
        countsMap.set(productId, { instances: new Set(), connections: 0 });
      }
      
      const counts = countsMap.get(productId)!;
      
      tp.tenant_product_datasources?.forEach((tpd: any) => {
        // Only count datasources that have an actual connection_id assigned
        if (tpd.tenant_instance_id && tpd.connection_id) {
          counts.instances.add(tpd.tenant_instance_id);
          counts.connections++;
        }
      });
    });
    
    return countsMap;
  }, [tenant]);

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
      await updateTenant({
        variables: {
          id: tenant.id,
          ...tenantEditForm,
        },
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
        await updateTenantInstance({
          variables: {
            id: editingInstance.id,
            tenant_id: tenant.id,
            ...instanceForm,
          },
        });
      } else {
        await createTenantInstance({
          variables: {
            tenant_id: tenant.id,
            ...instanceForm,
          },
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
      await deleteTenantInstance({
        variables: { id: instanceId },
      });
    } catch (err) {
      console.error('Error deleting instance:', err);
    }
  };

  const handleAddConnection = () => {
    setEditingConnection(null);
    setSelectedConnectionProduct('');
    setSelectedConnectionInstance('');
    setConnectionForm({
      name: '',
      type: 'postgres',
      host: '',
      port: '5432',
      database: '',
      schema: '',
      username: '',
      password: '',
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
    // Use mapped fields from Connection interface
    console.log('Editing connection:', {
      id: connection.id,
      name: connection.name,
      linkedProductId: connection.linkedProductId,
      linkedInstanceId: connection.linkedInstanceId,
      linkedAlphaDatasourceId: connection.linkedAlphaDatasourceId,
    });
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
      password: connection.password || '',
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

      // Build connection config object
      const connectionConfig = {
        type: connectionForm.type,
        host: connectionForm.host,
        port: connectionForm.port ? parseInt(connectionForm.port) : undefined,
        database: connectionForm.database,
        schema: connectionForm.schema,
        username: connectionForm.username,
        password: connectionForm.password,
        base_url: connectionForm.base_url,
        api_key: connectionForm.api_key,
        auth_type: connectionForm.auth_type,
        ...connectionForm.metadata,
      };

      // Remove undefined values
      Object.keys(connectionConfig).forEach(
        key => connectionConfig[key as keyof typeof connectionConfig] === undefined && delete connectionConfig[key as keyof typeof connectionConfig]
      );

      await testConnection({
        variables: {
          connection_details: JSON.stringify(connectionConfig),
        },
      });
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

      // Ensure datasources are loaded before proceeding
      if (!datasourcesData?.alpha_datasource || datasourcesData.alpha_datasource.length === 0) {
        console.warn('Datasource types not yet loaded, attempting to refetch...');
        // Try to fetch fresh if we don't have the data
        const freshDatasources = await client.query({
          query: GET_AVAILABLE_DATASOURCES,
          fetchPolicy: 'network-only',
        });
        if (!freshDatasources.data?.alpha_datasource?.length) {
          alert('Unable to load datasource types. Please try again.');
          return;
        }
        // Proceed with fresh data
      }

      let result;
      let connectionId;

      if (editingConnection?.id) {
        // Update existing connection - exclude tenant_id
        const updateObject = {
            name: connectionForm.name,
            type: connectionForm.type,
            host: connectionForm.host || null,
            port: connectionForm.port ? parseInt(connectionForm.port) : null,
            database: connectionForm.database || null,
            schema: connectionForm.schema || null,
            username: connectionForm.username || null,
            password: connectionForm.password || null,
            metadata: {
              auth_type: connectionForm.auth_type,
              base_url: connectionForm.base_url,
              api_key: connectionForm.api_key,
              product_id: selectedConnectionProduct,
              ...connectionForm.metadata,
            },
            is_active: connectionForm.is_active,
        };

        result = await updateConnection({
          variables: {
            id: editingConnection.id,
            object: updateObject,
          },
        });
        connectionId = editingConnection.id;
      } else {
        // Create new connection - include tenant_id
        const createObject = {
            tenant_id: tenant?.id || scopedTenant?.id || '',
            name: connectionForm.name,
            type: connectionForm.type,
            host: connectionForm.host || null,
            port: connectionForm.port ? parseInt(connectionForm.port) : null,
            database: connectionForm.database || null,
            schema: connectionForm.schema || null,
            username: connectionForm.username || null,
            password: connectionForm.password || null,
            metadata: {
              auth_type: connectionForm.auth_type,
              base_url: connectionForm.base_url,
              api_key: connectionForm.api_key,
              product_id: selectedConnectionProduct,
              ...connectionForm.metadata,
            },
            is_active: connectionForm.is_active,
        };

        result = await createConnection({
          variables: {
            object: createObject,
          },
        });
        connectionId = result.data?.insert_connections_one?.id;
      }

      if (connectionId) {
        // Unlink this connection from any existing datasources to ensure 1:1 relationship
        // and prevent "stale" links when moving a connection between products/instances.
        if (tenant?.tenant_products) {
          for (const tp of tenant.tenant_products) {
            if (tp.tenant_product_datasources) {
              for (const ds of tp.tenant_product_datasources) {
                if (ds.connection_id === connectionId) {
                  // Use the CONNECTION_ONLY mutation which just updates connection_id
                  await updateTpdConnectionOnly({
                    variables: {
                      id: ds.id,
                      connection_id: null,
                    },
                  });
                }
              }
            }
          }
        }

        // Now link the connection to the product/instance via tenant_product_datasource
        // First, ensure the product is registered to the tenant
        let tenantProductId = selectedConnectionProduct;
        const existingProduct = tenant?.tenant_products?.find(
          (tp: any) => tp.alpha_product_id === selectedConnectionProduct
        );


        if (!existingProduct) {
          // Register the product to the tenant first
          const registerResult = await addTenantProduct({
            variables: {
              tenant_id: scopedTenant?.id,
              tenant_instance_id: selectedConnectionInstance,
              alpha_product_id: selectedConnectionProduct,
              version: 1.0,
              is_active: true,
            },
          });
          tenantProductId = registerResult.data?.insert_tenant_product_one?.id;
          // Refetch to get updated tenant data
          await refetch();
        } else {
          tenantProductId = existingProduct.id;
        }
        
        if (tenantProductId) {
          // Determine the correct Alpha Datasource ID (Datasource Type)
          let resolvedAlphaDatasourceId = selectedAlphaDatasource;
          
          // If not explicitly selected, try to auto-resolve based on connection type
          if (!resolvedAlphaDatasourceId && datasourcesData?.alpha_datasource?.length > 0) {
            const typeStr = connectionForm.type.toLowerCase();
            const match = datasourcesData.alpha_datasource.find((ds: any) => 
              ds.datasource_code?.toLowerCase().includes(typeStr)
            );
            if (match) {
              resolvedAlphaDatasourceId = match.id;
              console.log(`Auto-resolved datasource type: ${match.datasource_code}`);
            } else {
              // Fallback to first available
              const fallback = datasourcesData.alpha_datasource[0];
              resolvedAlphaDatasourceId = fallback?.id;
              console.warn(`Could not match datasource type for ${connectionForm.type}; using ${fallback?.datasource_code}`);
            }
          }

          // Final validation: require non-null datasource type
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

          // Use fresh data to avoid stale closure issues
          const refreshResult = await refetch();
          const freshTenant = refreshResult.data?.tenants?.[0];

          // Priority 1: Search for an EXACT MATCH for the target (Product, Instance)
          // We want to update this specific record if it exists, rather than stealing another record.
          let existingDatasource = null;
          
          if (freshTenant?.tenant_products) {
             const targetProduct = freshTenant.tenant_products.find((tp: any) => tp.id === tenantProductId);
             console.log(`Looking for TPD in product ${tenantProductId}:`, targetProduct?.alpha_product?.product_name);
             if (targetProduct?.tenant_product_datasources) {
                console.log(`  Product has ${targetProduct.tenant_product_datasources.length} datasources`);
                existingDatasource = targetProduct.tenant_product_datasources.find(
                    (ds: any) => {
                       const matches = ds.tenant_instance_id === selectedConnectionInstance &&
                       ds.alpha_tenant_instance_id === resolvedAlphaDatasourceId;
                       console.log(`  Checking TPD ${ds.id}: instance=${ds.tenant_instance_id} vs ${selectedConnectionInstance}, alpha=${ds.alpha_tenant_instance_id} vs ${resolvedAlphaDatasourceId}, matches=${matches}`);
                       return matches;
                    }
                );
                console.log(`  Found matching datasource: ${existingDatasource?.id || 'NONE'}`);
             }
          }

          
          if (existingDatasource) {
            // Validate connectionId before attempting update
            if (!connectionId) {
                console.error("Critical: connectionId is missing during linking phase");
                alert("Internal Error: Connection ID is missing. Please refresh and try again.");
                return;
            }

            if (!resolvedAlphaDatasourceId) {
                alert('Unable to link connection: missing datasource type.');
                return;
            }

            console.log('Updating TPD:', {
                id: existingDatasource.id,
                tenant_product_id: tenantProductId,
                tenant_instance_id: selectedConnectionInstance,
                alpha_tenant_instance_id: resolvedAlphaDatasourceId,
                connection_id: connectionId,
            });

            // Verify connection exists before linking
            // We use a direct client query to bypass potential cache/query structure issues with the main query
            // AND we explicitly inject the Tenant ID to ensure Hasura context is valid even if global localStorage is empty/stale.
            const verifyResult = await client.query({
              query: gql`
                query VerifyConnection($id: uuid!) {
                  connections(where: { id: { _eq: $id } }) {
                    id
                    name
                    is_active
                  }
                }
              `,
              variables: { id: connectionId },
              fetchPolicy: 'network-only',
              context: {
                headers: {
                  'X-Tenant-ID': tenantId, // Ensure we send the current tenant context
                }
              }
            });
            
            const verifyConn = verifyResult.data?.connections?.[0];
            
            if (!verifyConn) {
                console.error("Critical: Connection created/updated but not found in subsequent query.", connectionId);
                alert("Warning: Connection saved, but not visible yet. Linking to product skipped to prevent errors.");
                return;
            }

            // Update existing datasource with new instance
            // We use the LINKING mutation which ignores tenant_product_id to avoid null value errors
            // since we are not moving the datasource between products.
            const updateVars = {
              id: existingDatasource.id,
              // tenant_product_id intentionally omitted
              tenant_instance_id: selectedConnectionInstance || null,
              alpha_tenant_instance_id: resolvedAlphaDatasourceId || null,
              connection_id: connectionId,
              is_active: existingDatasource.is_active,
              source_name: connectionForm.name,
              config: existingDatasource.config,
            };

            console.log('Linking mutation vars:', updateVars);
            
            try {
                const linkResult = await updateTenantProductDatasourceLinking({
                  variables: updateVars,
                });
                console.log('Link result:', linkResult.data);
            } catch (innerErr) {
                console.error("Failed to update TPD link:", innerErr);
                // Don't crash the whole save flow, but alert user
                alert("Connection saved, but failed to link to Product. Please try linking again.");
            }
          } else {
            // Ensure we have valid IDs before creating
            if (!selectedConnectionInstance) {
              alert('Unable to link connection: missing instance.');
              return;
            }
            if (!resolvedAlphaDatasourceId) {
              alert('Unable to link connection: missing datasource type.');
              return;
            }

            // Create new datasource link
            await addTenantProductDatasource({
              variables: {
                tenant_product_id: tenantProductId,
                tenant_instance_id: selectedConnectionInstance,
                alpha_tenant_instance_id: resolvedAlphaDatasourceId,
                config: {},
                is_active: true,
                source_name: connectionForm.name,
                connection_id: connectionId,
              },
            });
          }
        }

        setConnectionDialogOpen(false);
        setEditingConnection(null);
        // Reset form
        setConnectionForm({
          name: '',
          type: 'postgres',
          host: '',
          port: '5432',
          database: '',
          schema: '',
          username: '',
          password: '',
          base_url: '',
          api_key: '',
          auth_type: 'basic',
          metadata: {},
          is_active: true,
        });
        setConnectionConfigJson('{}');
        setSelectedConnectionProduct('');
        setSelectedConnectionInstance('');
        setConnectionsRefreshKey(prev => prev + 1); // Force ConnectionsTabContent to refetch
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
        <Link
          color="inherit"
          onClick={() => navigate('/tenants')}
          sx={{ cursor: 'pointer' }}
        >
          Home
        </Link>
        <Link
          color="inherit"
          onClick={() => navigate('/tenants')}
          sx={{ cursor: 'pointer' }}
        >
          Tenants
        </Link>
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
          <Tab label={`Instances (${instances.length})`} id="tenant-tab-0" aria-controls="tenant-tabpanel-0" />
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
            <InstancesTableV2
              instances={enrichedInstances}
              onAddInstance={handleAddInstance}
              onEditInstance={handleEditInstance}
              onDeleteInstance={handleDeleteInstance}
              onReload={() => refetch()}
            />
          </Box>
        </TabPanel>

        {/* Products Tab */}
        <TabPanel value={activeTab} index={1}>
          <Box sx={{ p: 3 }}>
            <ProductsTabContent 
              tenantId={tenantId || ''}
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
                  (tenant?.tenant_products || []).map((tp: any) => [
                    tp.alpha_product_id,
                    {
                      id: tp.alpha_product_id,
                      product_name: tp.alpha_product?.product_name || 'Unknown'
                    }
                  ])
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
                tenantData={tenant}
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
      <Dialog open={instanceDialogOpen} onClose={() => setInstanceDialogOpen(false)} maxWidth="sm" fullWidth>
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
      <Dialog open={deleteConfirmOpen} onClose={() => setDeleteConfirmOpen(false)}>
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
      >
        <DialogTitle>{editingConnection ? 'Edit Connection' : 'Add New Connection'}</DialogTitle>
        <DialogContent sx={{ pt: 2, maxHeight: '80vh', overflow: 'auto' }}>
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
                  const isRegistered = tenant?.tenant_products?.some(
                    (tp: any) => tp.alpha_product_id === product.id
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

            <FormControl fullWidth required disabled={isDerived}>
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
                      label="Password"
                      type="password"
                      value={connectionForm.password}
                      onChange={(e) =>
                        setConnectionForm({ ...connectionForm, password: e.target.value })
                      }
                      fullWidth
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
