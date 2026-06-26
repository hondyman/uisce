import { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Paper,
  Grid,
  Card,
  CardContent,
  CardActions,
  Button,
  Chip,
  TextField,
  InputAdornment,
  Tab,
  Tabs,
  Dialog,
  DialogContent,
  DialogActions,
  Alert,
  Tooltip,
} from '@mui/material';
import ModalHeader from '../components/ModalHeader';
import {
  Search as SearchIcon,
  Refresh as RefreshIcon,
  AutoAwesome as AutoAwesomeIcon,
  DataObject as DataObjectIcon,
  TableChart as TableChartIcon,
  Code as CodeIcon,
  Visibility as VisibilityIcon,
} from '@mui/icons-material';
// import CodeEditor from '../components/common/CodeEditor';
import { useTenant } from '../contexts/TenantContext';
import { useAuth } from '../contexts/AuthContext';
import { devLog, devWarn, devError } from '../utils/devLogger';
import { useNotification } from '../hooks/useNotification';

interface TableInfo {
  name: string;
  schema: string;
  description?: string;
  columnCount: number;
  rowCount?: number;
  hasCoreModel: boolean;
  coreModelId?: string;
  coreModelKey?: string;
  coreModelResolvedConfig?: unknown;
  lastGenerated?: Date;
  tags: string[];
}

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

const TabPanel: React.FC<TabPanelProps> = ({ children, value, index, ...other }) => (
  <div
    role="tabpanel"
    hidden={value !== index}
    id={`model-generator-tabpanel-${index}`}
    aria-labelledby={`model-generator-tab-${index}`}
    {...other}
  >
    {value === index && <Box sx={{ p: 3 }}>{children}</Box>}
  </div>
);

const ModelGeneratorPage: React.FC = () => {
  devLog('ModelGeneratorPage component rendered');
  const { datasource, tenant } = useTenant();
  const { isCoreAdmin, canManageCustomAssets } = useAuth();
  const canManageCore = isCoreAdmin();
  const canManageCustom = canManageCustomAssets();
  const isCoreTable = (table: TableInfo | null | undefined): boolean => {
    if (!table) return false;
    if (table.hasCoreModel || table.coreModelId) {
      return true;
    }
    const normalizedTags = table.tags?.map(tag => tag.toLowerCase()) ?? [];
    return normalizedTags.includes('core');
  };
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedTab, setSelectedTab] = useState(0);
  const [tables, setTables] = useState<TableInfo[]>([]);
  const [selectedTable, setSelectedTable] = useState<TableInfo | null>(null);
  const [showGeneratedModel, setShowGeneratedModel] = useState(false);
  const [generatedModelCode, setGeneratedModelCode] = useState('');
  const [generationInProgress, setGenerationInProgress] = useState(false);
  const [loading, setLoading] = useState(true);
  const notification = useNotification();

  // Fetch real tables data from the catalog API
  useEffect(() => {
  devLog('useEffect running - about to fetch tables');
    const fetchTables = async () => {
      if (!datasource?.id) {
  devLog('No datasource selected');
        setLoading(false);
        return;
      }
      setLoading(true);
      try {
        const datasourceId = datasource.id;
  devLog('Fetching tables from API...');
        // Use the Vite proxy to forward /api requests to the backend
        const response = await fetch(`/api/catalog/tables?tenant_instance_id=${datasourceId}`);

        if (!response.ok) {
          throw new Error(`Failed to fetch tables: ${response.status}`);
        }

        const data = await response.json();
  devLog(`API response: ${data.count} tables found`);

        const normalizeModelKey = (value?: string | null): string | null => {
          if (!value) return null;
          return value.replace(/^\//, '').replace(/\//g, '.').toLowerCase();
        };

        const normalizeTableKey = (table: any): string | null => {
          const tableData = table?.data ?? {};
          const rawQualified =
            tableData.tableName ||
            tableData.qualifiedPath ||
            (tableData.schema && tableData.label ? `${tableData.schema}.${tableData.label}` : null);

          if (!rawQualified) {
            return null;
          }

          return rawQualified.replace(/\//g, '.').toLowerCase();
        };

        // Build a lookup of core semantic models for this datasource so we can resolve
        // the actual fabric_defn IDs needed to fetch model definitions later.
        const coreModelLookup = new Map<string, { id: string; key?: string; resolvedConfig?: unknown }>();
        try {
          const coreResponse = await fetch(`/api/fabric/models?tenant_instance_id=${datasourceId}`);
          if (coreResponse.ok) {
            const coreJson = await coreResponse.json();
            const coreModels: any[] = Array.isArray(coreJson.models)
              ? coreJson.models
              : Array.isArray(coreJson.data)
                ? coreJson.data
                : [];

            coreModels.forEach((model: any) => {
              const isCoreModel = model.is_core ?? model.isCore ?? false;
              const modelId = model.id;
              const modelKey = model.model_key ?? model.modelKey ?? null;
              const normalizedKey = normalizeModelKey(modelKey);

              if (!isCoreModel || !modelId || !normalizedKey) {
                return;
              }

              if (!coreModelLookup.has(normalizedKey)) {
                coreModelLookup.set(normalizedKey, {
                  id: modelId,
                  key: modelKey ?? undefined,
                  resolvedConfig: model.resolved_config ?? model.resolvedConfig,
                });
              }
            });

            devLog(`Loaded ${coreModelLookup.size} core models for datasource ${datasourceId}`);
          } else {
            devWarn(`Failed to fetch fabric models: ${coreResponse.status}`);
          }
        } catch (coreError) {
          devError('Error fetching core models:', coreError);
        }

        // Transform backend data to match our TableInfo interface and enrich with core metadata.
        const transformedTables: TableInfo[] = data.tables.map((table: any) => {
          const tableData = table.data ?? {};
          const coreIdFromNode = tableData.core_id ?? tableData.coreId ?? undefined;
          const isCoreFlag = tableData.isCore ?? tableData.is_core ?? false;
          const normalizedKey = normalizeTableKey(table);
          const coreModel = normalizedKey ? coreModelLookup.get(normalizedKey) : undefined;

          const hasCoreModel = Boolean(coreModel) || Boolean(coreIdFromNode) || Boolean(isCoreFlag);
          const rawTags = [
            tableData.nodeType || 'table',
            tableData.schemaName || tableData.schema || 'unknown_schema',
            ...(hasCoreModel ? ['Core'] : []),
          ].filter(Boolean) as string[];
          const tags = Array.from(new Set(rawTags));

          return {
            name: tableData.label || tableData.tableName,
            schema: tableData.schema || tableData.schemaName || 'public',
            description: tableData.description || `Table: ${tableData.qualifiedPath}`,
            columnCount: tableData.columnCount || (Array.isArray(tableData.columns) ? tableData.columns.length : 0),
            rowCount: undefined, // Not provided by backend
            hasCoreModel,
            coreModelId: coreModel?.id || coreIdFromNode,
            coreModelKey: coreModel?.key,
            coreModelResolvedConfig: coreModel?.resolvedConfig,
            lastGenerated: undefined, // Not provided by backend
            tags,
          };
        });

        setTables(transformedTables);
        devLog(`Loaded ${transformedTables.length} tables into Model Generator`);
      } catch (error) {
        devError('Error fetching tables:', error);
        // Fallback to empty array if API fails
        setTables([]);
      } finally {
        setLoading(false);
      }
    };

    fetchTables();
  }, [datasource]);

  const filteredTables = tables.filter(table =>
    table.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    table.description?.toLowerCase().includes(searchTerm.toLowerCase()) ||
    table.tags.some(tag => tag.toLowerCase().includes(searchTerm.toLowerCase()))
  );

  const tablesWithModels = filteredTables.filter(table => table.hasCoreModel);
  const tablesWithoutModels = filteredTables.filter(table => !table.hasCoreModel);

  const handleGenerateModel = async (table: TableInfo) => {
    const core = isCoreTable(table);
    if (core && !canManageCore) {
      notification.warning('Core models are read-only for your role.');
      return;
    }
    if (!core && !canManageCustom) {
      notification.warning('You do not have permission to generate models for this table.');
      return;
    }

    setGenerationInProgress(true);
    setSelectedTable(table);

    try {
      // Simulate API call to generate model
      await new Promise(resolve => setTimeout(resolve, 2000));
      
      // Mock generated Cube.js model with hierarchies and drill members
      const mockModel = {
        sql: `SELECT * FROM ${table.schema}.${table.name}`,
        measures: {
          count: {
            type: 'count',
            title: `${table.name} Count`
          }
        },
        dimensions: {
          id: {
            sql: 'id',
            type: 'number',
            primaryKey: true
          },
          name: {
            sql: 'name',
            type: 'string',
            title: 'Name'
          },
          created_at: {
            sql: 'created_at',
            type: 'time',
            title: 'Created At'
          }
        },
        hierarchies: {
          temporal: {
            title: 'Time Hierarchy',
            levels: ['created_at.year', 'created_at.quarter', 'created_at.month', 'created_at.day']
          }
        },
        drillMembers: ['name', 'created_at']
      };

      setGeneratedModelCode(JSON.stringify(mockModel, null, 2));
      setShowGeneratedModel(true);
    } catch (error) {
      devError('Failed to generate model:', error);
    } finally {
      setGenerationInProgress(false);
    }
  };

  const handleRegenerateModel = async (table: TableInfo) => {
    await handleGenerateModel(table);
  };

  const handleViewModel = async (table: TableInfo) => {
    setSelectedTable(table);

    if (!tenant?.id || !datasource?.id) {
      setGeneratedModelCode(`// Unable to load model for ${table.name}\n// Please select a tenant and datasource first.`);
      setShowGeneratedModel(true);
      return;
    }

    if (!table.coreModelId) {
      const fallback = table.coreModelResolvedConfig
        ? JSON.stringify({ resolved_config: table.coreModelResolvedConfig }, null, 2)
        : `// No core model has been generated for ${table.name} yet.`;
      setGeneratedModelCode(fallback);
      setShowGeneratedModel(true);
      return;
    }

    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id,
      });

      const response = await fetch(`/api/models/${table.coreModelId}?${params.toString()}`);
      if (response.ok) {
        const modelData = await response.json();
        const resolved = modelData.resolved_config ?? modelData.resolvedConfig ?? {};
        const displayPayload = {
          model_id: modelData.id ?? table.coreModelId,
          model_key: modelData.model_key ?? modelData.modelKey ?? table.coreModelKey,
          version: modelData.version,
          status: modelData.status,
          resolved_config: resolved,
        };
        setGeneratedModelCode(JSON.stringify(displayPayload, null, 2));
      } else {
        devWarn(`Failed to load core model ${table.coreModelId}: ${response.status}`);
        const fallback = table.coreModelResolvedConfig
          ? JSON.stringify({ resolved_config: table.coreModelResolvedConfig }, null, 2)
          : `// Backend returned ${response.status} ${response.statusText} for ${table.name}`;
        setGeneratedModelCode(fallback);
      }
    } catch (error) {
      devError('Error fetching core model:', error);
      const fallback = table.coreModelResolvedConfig
        ? JSON.stringify({ resolved_config: table.coreModelResolvedConfig }, null, 2)
        : `// Error loading model for ${table.name}\n// ${error instanceof Error ? error.message : String(error)}`;
      setGeneratedModelCode(fallback);
    } finally {
      setShowGeneratedModel(true);
    }
  };

  const handleSaveModel = async () => {
    if (!selectedTable || !datasource || !tenant) return;

    const targetTableName = selectedTable.name;
    const isCoreTarget = isCoreTable(selectedTable);
    if (isCoreTarget && !canManageCore) {
      notification.warning('Core models are read-only for your role.');
      return;
    }
    if (!isCoreTarget && !canManageCustom) {
      notification.warning('You do not have permission to save models for this table.');
      return;
    }

    let parsedGeneratedModel: unknown;
    try {
      parsedGeneratedModel = JSON.parse(generatedModelCode);
    } catch (parseErr) {
      devError('Generated model JSON is invalid:', parseErr);
      notification.error('Generated model is not valid JSON. Please fix the model before saving.');
      return;
    }

    try {
      devLog('Saving model:', generatedModelCode);
      
      // Save model to backend via API gateway
      // Use a simple endpoint that accepts generated model JSON
      const response = await fetch(`/api/models/generated?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          table_name: selectedTable.name,
          schema: selectedTable.schema,
          model_definition: parsedGeneratedModel
        })
      });
      
      if (response.ok) {
        let resultPayload: any = null;
        try {
          resultPayload = await response.json();
        } catch (parseError) {
          devWarn('Model save response was not JSON:', parseError);
        }

        const newModelId: string | undefined = resultPayload?.id;
        const newModelKey: string | undefined = resultPayload?.model_key ?? resultPayload?.modelKey;

        const parsedModel = parsedGeneratedModel;

  devLog('Model saved successfully');
        notification.success('Model saved successfully to backend!');
        setShowGeneratedModel(false);

        setTables(prev => prev.map(t => {
          if (t.name !== targetTableName) {
            return t;
          }

          const isCore = isCoreTable(t);
          const tagSet = new Set(t.tags);
          if (isCore) {
            tagSet.add('Core');
          } else {
            tagSet.delete('Core');
            tagSet.add('Custom');
          }
          return {
            ...t,
            hasCoreModel: t.hasCoreModel || isCore || Boolean(newModelId ?? t.coreModelId ?? parsedModel),
            coreModelId: newModelId ?? t.coreModelId,
            coreModelKey: newModelKey ?? t.coreModelKey,
            coreModelResolvedConfig: parsedModel ?? t.coreModelResolvedConfig,
            lastGenerated: new Date(),
            tags: Array.from(tagSet),
          };
        }));

        setSelectedTable(prev => {
          if (!prev || prev.name !== targetTableName) {
            return prev;
          }

          const isCore = isCoreTable(prev);
          const tagSet = new Set(prev.tags);
          if (isCore) {
            tagSet.add('Core');
          } else {
            tagSet.delete('Core');
            tagSet.add('Custom');
          }
          return {
            ...prev,
            hasCoreModel: prev.hasCoreModel || isCore || Boolean(newModelId ?? prev.coreModelId ?? parsedModel),
            coreModelId: newModelId ?? prev.coreModelId,
            coreModelKey: newModelKey ?? prev.coreModelKey,
            coreModelResolvedConfig: parsedModel ?? prev.coreModelResolvedConfig,
            lastGenerated: new Date(),
            tags: Array.from(tagSet),
          };
        });
      } else {
        const errorText = await response.text();
        devError('Failed to save model:', response.status, errorText);
        notification.error(`Failed to save model: ${response.status} ${errorText}`);
      }
    } catch (error) {
      devError('Error saving model:', error);
      notification.error(`Error saving model: ${error instanceof Error ? error.message : String(error)}`);
    }
  };

  const TableCard: React.FC<{ table: TableInfo }> = ({ table }) => (
    <Card sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <CardContent sx={{ flexGrow: 1 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
          <TableChartIcon sx={{ mr: 1, color: 'primary.main' }} />
          <Typography variant="h6" component="h3">
            {table.name}
          </Typography>
          {table.hasCoreModel && (
            <Chip 
              label="Core Model" 
              size="small" 
              color="success" 
              sx={{ ml: 'auto' }}
            />
          )}
        </Box>
        
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          {table.description || 'No description available'}
        </Typography>
        
        <Box sx={{ mb: 2 }}>
          <Typography variant="caption" color="text.secondary">
            Schema: {table.schema} • Columns: {table.columnCount}
            {table.rowCount && ` • Rows: ${table.rowCount.toLocaleString()}`}
          </Typography>
        </Box>
        
        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mb: 2 }}>
          {table.tags.map((tag) => (
            <Chip key={tag} label={tag} size="small" variant="outlined" />
          ))}
        </Box>
        
        {table.lastGenerated && (
          <Typography variant="caption" color="text.secondary">
            Last generated: {table.lastGenerated.toLocaleDateString()}
          </Typography>
        )}
      </CardContent>
      
      <CardActions>
        {table.hasCoreModel ? (
          <>
            <Button 
              size="small" 
              startIcon={<VisibilityIcon />}
              onClick={() => handleViewModel(table)}
            >
              View Model
            </Button>
            {(() => {
              const tableIsCore = isCoreTable(table);
              const permissionDisabled = tableIsCore ? !canManageCore : !canManageCustom;
              const regenerateDisabled = permissionDisabled || generationInProgress;
              const tooltip = permissionDisabled
                ? (tableIsCore
                    ? 'Core models are read-only for your role.'
                    : 'You do not have permission to modify this model.')
                : '';
              return (
                <Tooltip title={tooltip} disableHoverListener={!tooltip}>
                  <span>
                    <Button 
                      size="small" 
                      startIcon={<RefreshIcon />}
                      onClick={() => handleRegenerateModel(table)}
                      disabled={regenerateDisabled}
                    >
                      Regenerate
                    </Button>
                  </span>
                </Tooltip>
              );
            })()}
          </>
        ) : (
          (() => {
            const tableIsCore = isCoreTable(table);
            const permissionDisabled = tableIsCore ? !canManageCore : !canManageCustom;
            const generateDisabled = permissionDisabled || generationInProgress;
            const tooltip = permissionDisabled
              ? (tableIsCore
                  ? 'Core models are read-only for your role.'
                  : 'You do not have permission to generate models for this table.')
              : '';
            return (
              <Tooltip title={tooltip} disableHoverListener={!tooltip}>
                <span>
                  <Button 
                    size="small" 
                    variant="contained"
                    startIcon={<AutoAwesomeIcon />}
                    onClick={() => handleGenerateModel(table)}
                    disabled={generateDisabled}
                  >
                    Generate Model
                  </Button>
                </span>
              </Tooltip>
            );
          })()
        )}
      </CardActions>
    </Card>
  );

  if (!datasource) {
    return (
      <Box sx={{ p: 3 }}>
        <Typography variant="h4" gutterBottom>
          Model Generator
        </Typography>
        <Alert severity="info">
          Please select a datasource from the Connections page to generate models.
        </Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom sx={{ display: 'flex', alignItems: 'center' }}>
        <DataObjectIcon sx={{ mr: 2, color: 'primary.main' }} />
        Model Generator
      </Typography>
      
      <Typography variant="body1" color="text.secondary" sx={{ mb: 3 }}>
        Generate core semantic models from database tables and schema (Updated: {new Date().toLocaleTimeString()})
      </Typography>

      {loading && (
        <Alert severity="info" sx={{ mb: 3 }}>
          Loading tables from catalog... ({tables.length} loaded so far)
        </Alert>
      )}

      {/* Search and Actions */}
      <Box sx={{ mb: 3, display: 'flex', gap: 2, alignItems: 'center' }}>
        <TextField
          placeholder="Search tables, descriptions, or tags..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon />
              </InputAdornment>
            ),
          }}
          sx={{ flexGrow: 1 }}
        />
        <Button
          variant="outlined"
          startIcon={<RefreshIcon />}
          onClick={async () => {
            // Refresh tables from API
            try {
              const datasourceId = 'f938c8e6-6e11-405c-a700-ce5eacc5f45b';
              const response = await fetch(`http://localhost:5175/api/catalog/tables?tenant_instance_id=${datasourceId}`);
              
              if (!response.ok) {
                throw new Error(`Failed to fetch tables: ${response.status}`);
              }
              
              const data = await response.json();
              
              const transformedTables: TableInfo[] = data.tables.map((table: any) => {
                const coreId = table.data.core_id ?? table.data.coreId ?? undefined;
                const isCoreFlag = table.data.isCore ?? table.data.is_core ?? (coreId != null);

                return {
                  name: table.data.label || table.data.tableName,
                  schema: table.data.schema || table.data.schemaName || 'public',
                  description: table.data.description || `Table: ${table.data.qualifiedPath}`,
                  columnCount: table.data.columnCount || 0,
                  rowCount: undefined,
                  hasCoreModel: isCoreFlag || false,
                  coreModelId: coreId,
                  lastGenerated: undefined,
                  tags: [
                    table.data.nodeType || 'table',
                    table.data.schemaName || 'unknown_schema',
                    ...(isCoreFlag ? ['Core'] : [])
                  ]
                };
              });
              
              setTables(transformedTables);
              devLog(`Refreshed ${transformedTables.length} tables from catalog`);
            } catch (error) {
              devError('Error refreshing tables:', error);
            }
          }}
        >
          Refresh
        </Button>
        <Button
          variant="outlined"
          color="secondary"
          onClick={async () => {
            devLog('Manual debug API test...');
            const devProxyUrl = 'http://localhost:5175/_debug';
            try {
              // First try a relative path. In production or when Vite proxies are
              // configured this will work. If the server returns HTML (index.html)
              // the JSON.parse below will fail and we'll fall back to the dev-proxy.
              const tryRelative = await fetch('/_debug');
              const bodyText = await tryRelative.text();
              try {
                const parsed = JSON.parse(bodyText);
                devLog('Debug response (relative):', parsed);
                notification.success(`Debug API works! Status: ${parsed.status}`);
                return;
              } catch (parseErr) {
                devWarn('Relative /_debug did not return JSON, falling back to dev-proxy', parseErr);
              }

              // Fallback to explicit dev-proxy URL
              const response = await fetch(devProxyUrl);
              const data = await response.json();
              devLog('Debug response (dev-proxy):', data);
              notification.success(`Debug API works! Status: ${data.status}`);
            } catch (error) {
              devError('Debug API failed:', error);
              notification.error(`Debug API failed: ${error instanceof Error ? error.message : String(error)}`);
            }
          }}
        >
          Test API
        </Button>
      </Box>

      {/* Tabs */}
      <Paper sx={{ mb: 3 }}>
        <Tabs 
          value={selectedTab} 
          onChange={(_, newValue) => setSelectedTab(newValue)}
          indicatorColor="primary"
          textColor="primary"
        >
          <Tab label={`All Tables (${filteredTables.length})`} />
          <Tab label={`With Models (${tablesWithModels.length})`} />
          <Tab label={`Without Models (${tablesWithoutModels.length})`} />
        </Tabs>
      </Paper>

      {/* Table Grids */}
      <TabPanel value={selectedTab} index={0}>
        <Grid container spacing={3}>
          {filteredTables.map((table, index) => (
            <Grid
              item
              xs={12}
              md={6}
              lg={4}
              key={`${table.schema}.${table.name}-${table.coreModelId ?? table.coreModelKey ?? index}`}
            >
              <TableCard table={table} />
            </Grid>
          ))}
        </Grid>
      </TabPanel>

      <TabPanel value={selectedTab} index={1}>
        <Grid container spacing={3}>
          {tablesWithModels.map((table, index) => (
            <Grid
              item
              xs={12}
              md={6}
              lg={4}
              key={`${table.schema}.${table.name}-${table.coreModelId ?? table.coreModelKey ?? index}`}
            >
              <TableCard table={table} />
            </Grid>
          ))}
        </Grid>
      </TabPanel>

      <TabPanel value={selectedTab} index={2}>
        <Grid container spacing={3}>
          {tablesWithoutModels.map((table, index) => (
            <Grid
              item
              xs={12}
              md={6}
              lg={4}
              key={`${table.schema}.${table.name}-${table.coreModelId ?? table.coreModelKey ?? index}`}
            >
              <TableCard table={table} />
            </Grid>
          ))}
        </Grid>
      </TabPanel>

      {/* Generated Model Dialog */}
      <Dialog 
        open={showGeneratedModel} 
        onClose={() => setShowGeneratedModel(false)}
        maxWidth="lg"
        fullWidth
      >
        <ModalHeader
          title={
            <Box sx={{ display: 'flex', alignItems: 'center' }}>
              <CodeIcon sx={{ mr: 1 }} />
              Generated Model: {selectedTable?.name}
            </Box>
          }
          onClose={() => setShowGeneratedModel(false)}
        />
        <DialogContent>
          <Alert severity="info" sx={{ mb: 2 }}>
            Review the generated model before saving. You can edit the code directly.
          </Alert>
          <TextField
            multiline
            rows={20}
            value={generatedModelCode}
            onChange={(e) => setGeneratedModelCode(e.target.value)}
            variant="outlined"
            fullWidth
            sx={{ fontFamily: 'monospace' }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowGeneratedModel(false)}>
            Cancel
          </Button>
          <Button 
            variant="contained" 
            onClick={handleSaveModel}
            startIcon={<AutoAwesomeIcon />}
          >
            Save Model
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default ModelGeneratorPage;
