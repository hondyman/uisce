import { useState, useEffect, useCallback, FC } from 'react';
import { devError } from '../../../utils/devLogger';
import {
  Box, Card, CardContent, Typography, Grid, Button, Dialog,
  DialogContent, DialogActions, TextField, Select, MenuItem, FormControl,
  InputLabel, Chip, IconButton, Tabs, Tab, Accordion, AccordionSummary,
  AccordionDetails, List, ListItem, ListItemText, ListItemIcon,
  Switch, FormControlLabel, Tooltip, RadioGroup, FormLabel,
  Radio, FormControlLabel as RadioLabel, SpeedDial, SpeedDialAction,
  Avatar, AvatarGroup, Badge, Alert,
  Rating, CardActions, CardHeader, Step as _Step, StepLabel as _StepLabel, Divider
} from '@mui/material';
import ModalHeader from '../../../components/ModalHeader';
import {
  Add as AddIcon,
  PlayArrow as PlayIcon,
  Save as SaveIcon,
  Share as ShareIcon,
  Edit as EditIcon,
  ContentCopy as CloneIcon,
  Delete as DeleteIcon,
  ExpandMore as ExpandMoreIcon,
  Api as ApiIcon,
  Http as HttpIcon,
  Settings as _SettingsIcon,
  Code as CodeIcon,
  Security as SecurityIcon,
  SmartToy as AIIcon,
  People as PeopleIcon,
  Timeline as _TimelineIcon,
  Download as DownloadIcon,
  Upload as _UploadIcon,
  History as HistoryIcon,
  Science as TestIcon,
  Star as _StarIcon,
  ThumbUp as _LikeIcon,
  Comment as _CommentIcon,
  Lightbulb as SuggestionIcon,
  TrendingUp as _TrendingIcon,
  Speed as PerformanceIcon,
  CloudDownload as ExportIcon,
  LibraryBooks as TemplateIcon,
  GitHub as GitIcon,
  Analytics as AnalyticsIcon
} from '@mui/icons-material';
import { io as _io } from 'socket.io-client';
import renderCoreCustomChips from '../../../components/common/semanticChips';

interface API {
  id: string;
  name: string;
  description: string;
  type: 'public' | 'private';
  createdBy: string;
  createdAt: string;
  updatedAt: string;
  isCore: boolean;
  config: APIConfig;
  tags: string[];
  sharedWith: string[];
  endpoints: APIEndpoint[];
}

interface APIConfig {
  basePath: string;
  authentication: 'none' | 'jwt' | 'api-key' | 'oauth';
  rateLimit: number;
  corsEnabled: boolean;
  dataSource: string;
}

interface APIEndpoint {
  id: string;
  path: string;
  method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
  description: string;
  operation: 'create' | 'read' | 'update' | 'delete' | 'list';
  parameters: APIParameter[];
  responses: APIResponse[];
  sqlTemplate?: string;
}

interface APIParameter {
  name: string;
  type: string;
  required: boolean;
  location: 'path' | 'query' | 'body';
  description: string;
}

interface APIResponse {
  statusCode: number;
  description: string;
  schema: any;
}

const APIBuilderPage: FC = () => {
  const [apis, setApis] = useState<API[]>([]);
  const [selectedAPI, setSelectedAPI] = useState<API | null>(null);
  const [apiConfig, setApiConfig] = useState<APIConfig>({
    basePath: '/api',
    authentication: 'jwt',
    rateLimit: 100,
    corsEnabled: true,
    dataSource: ''
  });
  const [endpoints, setEndpoints] = useState<APIEndpoint[]>([]);
  const [_isEditing, setIsEditing] = useState(false);
  const [showSaveDialog, setShowSaveDialog] = useState(false);
  const [showShareDialog, setShowShareDialog] = useState(false);
  const [showEndpointDialog, setShowEndpointDialog] = useState(false);
  const [selectedEndpoint, setSelectedEndpoint] = useState<APIEndpoint | null>(null);
  const [activeTab, setActiveTab] = useState(0);

  // World-class feature states
  const [showAIAssistant, setShowAIAssistant] = useState(false);
  const [showCollaboration, setShowCollaboration] = useState(false);
  const [showPerformanceInsights, setShowPerformanceInsights] = useState(false);
  const [showTemplateMarketplace, setShowTemplateMarketplace] = useState(false);
  const [showVersionControl, setShowVersionControl] = useState(false);
  const [showExportDialog, setShowExportDialog] = useState(false);
  const [showTestingSuite, setShowTestingSuite] = useState(false);
  const [_executionStats, _setExecutionStats] = useState({
    totalRequests: 0,
    avgResponseTime: 0,
    errorRate: 0,
    uptime: 0
  });
  // keep a runtime-facing name for components that expect executionStats prop
  const executionStats = _executionStats;
  const [currentUser] = useState('jane.doe');

  // Available data sources
  const [dataSources] = useState([
    { id: 'orders', name: 'Orders', table: 'orders', fields: ['id', 'customer_id', 'order_date', 'total_amount', 'status'] },
    { id: 'customers', name: 'Customers', table: 'customers', fields: ['id', 'name', 'email', 'created_at', 'segment'] },
    { id: 'products', name: 'Products', table: 'products', fields: ['id', 'name', 'category', 'price', 'stock_quantity'] }
  ]);

  useEffect(() => {
    loadAPIs();
  }, []);

  const loadAPIs = async () => {
    // Mock data - replace with actual API call
    const mockAPIs: API[] = [
      {
        id: '1',
        name: 'Customer Management API',
        description: 'CRUD operations for customer data',
        type: 'public',
        createdBy: 'john.doe',
        createdAt: '2024-01-15',
        updatedAt: '2024-01-15',
        isCore: true,
        config: {
          basePath: '/api/customers',
          authentication: 'jwt',
          rateLimit: 100,
          corsEnabled: true,
          dataSource: 'customers'
        },
        tags: ['customers', 'crud'],
        sharedWith: ['team.marketing', 'team.sales'],
        endpoints: [
          {
            id: '1',
            path: '/customers',
            method: 'GET',
            description: 'List all customers',
            operation: 'list',
            parameters: [
              { name: 'limit', type: 'integer', required: false, location: 'query', description: 'Number of records to return' },
              { name: 'offset', type: 'integer', required: false, location: 'query', description: 'Number of records to skip' }
            ],
            responses: [
              { statusCode: 200, description: 'Success', schema: { type: 'array', items: { $ref: '#/components/schemas/Customer' } } }
            ],
            sqlTemplate: 'SELECT * FROM customers LIMIT $limit OFFSET $offset'
          },
          {
            id: '2',
            path: '/customers/{id}',
            method: 'GET',
            description: 'Get customer by ID',
            operation: 'read',
            parameters: [
              { name: 'id', type: 'string', required: true, location: 'path', description: 'Customer ID' }
            ],
            responses: [
              { statusCode: 200, description: 'Success', schema: { $ref: '#/components/schemas/Customer' } },
              { statusCode: 404, description: 'Customer not found', schema: { type: 'object', properties: { error: { type: 'string' } } } }
            ],
            sqlTemplate: 'SELECT * FROM customers WHERE id = $id'
          }
        ]
      }
    ];
    setApis(mockAPIs);
  };

  const handleSaveAPI = async (name: string, description: string, type: 'public' | 'private') => {
    const newAPI: API = {
      id: selectedAPI?.id || Date.now().toString(),
      name,
      description,
      type,
      createdBy: 'current.user',
      createdAt: selectedAPI?.createdAt || new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      isCore: false,
      config: apiConfig,
      tags: [],
      sharedWith: [],
      endpoints
    };

    if (selectedAPI) {
      setApis(prev => prev.map(api => api.id === selectedAPI.id ? newAPI : api));
    } else {
      setApis(prev => [...prev, newAPI]);
    }

    setShowSaveDialog(false);
    setSelectedAPI(newAPI);
  };

  const handleCloneAPI = (api: API) => {
    const clonedAPI: API = {
      ...api,
      id: Date.now().toString(),
      name: `${api.name} (Copy)`,
      isCore: false,
      createdBy: 'current.user',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      endpoints: api.endpoints.map(endpoint => ({ ...endpoint, id: Date.now().toString() + Math.random() }))
    };
    setApis(prev => [...prev, clonedAPI]);
    setSelectedAPI(clonedAPI);
    setApiConfig(clonedAPI.config);
    setEndpoints(clonedAPI.endpoints);
    setIsEditing(true);
  };

  const handleAddEndpoint = () => {
    const newEndpoint: APIEndpoint = {
      id: Date.now().toString(),
      path: '/new-endpoint',
      method: 'GET',
      description: 'New endpoint',
      operation: 'read',
      parameters: [],
      responses: [
        { statusCode: 200, description: 'Success', schema: { type: 'object' } }
      ]
    };
    setEndpoints(prev => [...prev, newEndpoint]);
    setSelectedEndpoint(newEndpoint);
    setShowEndpointDialog(true);
  };

  const handleGenerateEndpoints = () => {
    if (!apiConfig.dataSource) return;

    const dataSource = dataSources.find(ds => ds.id === apiConfig.dataSource);
    if (!dataSource) return;

    const generatedEndpoints: APIEndpoint[] = [
      // List endpoint
      {
        id: Date.now().toString(),
        path: `/${dataSource.id}`,
        method: 'GET',
        description: `List all ${dataSource.name.toLowerCase()}`,
        operation: 'list',
        parameters: [
          { name: 'limit', type: 'integer', required: false, location: 'query', description: 'Number of records to return' },
          { name: 'offset', type: 'integer', required: false, location: 'query', description: 'Number of records to skip' }
        ],
        responses: [
          { statusCode: 200, description: 'Success', schema: { type: 'array', items: { type: 'object' } } }
        ],
        sqlTemplate: `SELECT * FROM ${dataSource.table} LIMIT $limit OFFSET $offset`
      },
      // Create endpoint
      {
        id: (Date.now() + 1).toString(),
        path: `/${dataSource.id}`,
        method: 'POST',
        description: `Create a new ${dataSource.name.toLowerCase().slice(0, -1)}`,
        operation: 'create',
        parameters: [
          { name: 'body', type: 'object', required: true, location: 'body', description: `${dataSource.name.slice(0, -1)} data` }
        ],
        responses: [
          { statusCode: 201, description: 'Created', schema: { type: 'object' } }
        ],
        sqlTemplate: `INSERT INTO ${dataSource.table} (${dataSource.fields.join(', ')}) VALUES (${dataSource.fields.map(f => `$${f}`).join(', ')}) RETURNING *`
      },
      // Read endpoint
      {
        id: (Date.now() + 2).toString(),
        path: `/${dataSource.id}/{id}`,
        method: 'GET',
        description: `Get ${dataSource.name.toLowerCase().slice(0, -1)} by ID`,
        operation: 'read',
        parameters: [
          { name: 'id', type: 'string', required: true, location: 'path', description: 'Record ID' }
        ],
        responses: [
          { statusCode: 200, description: 'Success', schema: { type: 'object' } },
          { statusCode: 404, description: 'Not found', schema: { type: 'object', properties: { error: { type: 'string' } } } }
        ],
        sqlTemplate: `SELECT * FROM ${dataSource.table} WHERE id = $id`
      },
      // Update endpoint
      {
        id: (Date.now() + 3).toString(),
        path: `/${dataSource.id}/{id}`,
        method: 'PUT',
        description: `Update ${dataSource.name.toLowerCase().slice(0, -1)} by ID`,
        operation: 'update',
        parameters: [
          { name: 'id', type: 'string', required: true, location: 'path', description: 'Record ID' },
          { name: 'body', type: 'object', required: true, location: 'body', description: 'Updated data' }
        ],
        responses: [
          { statusCode: 200, description: 'Updated', schema: { type: 'object' } },
          { statusCode: 404, description: 'Not found', schema: { type: 'object', properties: { error: { type: 'string' } } } }
        ],
        sqlTemplate: `UPDATE ${dataSource.table} SET ${dataSource.fields.filter(f => f !== 'id').map(f => `${f} = $${f}`).join(', ')} WHERE id = $id RETURNING *`
      },
      // Delete endpoint
      {
        id: (Date.now() + 4).toString(),
        path: `/${dataSource.id}/{id}`,
        method: 'DELETE',
        description: `Delete ${dataSource.name.toLowerCase().slice(0, -1)} by ID`,
        operation: 'delete',
        parameters: [
          { name: 'id', type: 'string', required: true, location: 'path', description: 'Record ID' }
        ],
        responses: [
          { statusCode: 204, description: 'Deleted', schema: { type: 'object' } },
          { statusCode: 404, description: 'Not found', schema: { type: 'object', properties: { error: { type: 'string' } } } }
        ],
        sqlTemplate: `DELETE FROM ${dataSource.table} WHERE id = $id`
      }
    ];

    setEndpoints(generatedEndpoints);
  };

  // World-class feature handlers
  const handleToggleAIAssistant = () => {
    setShowAIAssistant(!showAIAssistant);
  };

  const handleToggleCollaboration = () => {
    setShowCollaboration(!showCollaboration);
  };

  const handleAISuggestion = (_suggestion: any) => {
    // Apply AI suggestion to API configuration or endpoints
  };

  const handleTemplateSelect = (template: any) => {
    // Apply template to current API
    setApiConfig(template.config);
    setEndpoints(template.endpoints);
    setShowTemplateMarketplace(false);
  };

  const handleVersionRestore = (version: any) => {
    // Restore API to previous version
    setApiConfig(version.config);
    setEndpoints(version.endpoints);
    setShowVersionControl(false);
  };

  const handleExportAPI = () => {
    // Export API as OpenAPI spec or other formats
    setShowExportDialog(true);
  };

  const handleExecuteAPITest = () => {
    // Execute API tests
    setShowTestingSuite(true);
  };

  const renderAPIBuilder = () => (
    <Box sx={{ display: 'flex', height: 'calc(100vh - 200px)' }}>
      {/* Left Panel - API Configuration */}
      <Card sx={{ width: 300, mr: 2 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            API Configuration
          </Typography>

          <TextField
            fullWidth
            label="Base Path"
            value={apiConfig.basePath}
            onChange={(e) => setApiConfig(prev => ({ ...prev, basePath: e.target.value }))}
            sx={{ mb: 2 }}
          />

          <FormControl fullWidth sx={{ mb: 2 }}>
            <InputLabel>Data Source</InputLabel>
            <Select
              value={apiConfig.dataSource}
              onChange={(e) => setApiConfig(prev => ({ ...prev, dataSource: e.target.value }))}
            >
              {dataSources.map(ds => (
                <MenuItem key={ds.id} value={ds.id}>{ds.name}</MenuItem>
              ))}
            </Select>
          </FormControl>

          <FormControl fullWidth sx={{ mb: 2 }}>
            <InputLabel>Authentication</InputLabel>
            <Select
              value={apiConfig.authentication}
              onChange={(e) => setApiConfig(prev => ({ ...prev, authentication: e.target.value as any }))}
            >
              <MenuItem value="none">None</MenuItem>
              <MenuItem value="jwt">JWT</MenuItem>
              <MenuItem value="api-key">API Key</MenuItem>
              <MenuItem value="oauth">OAuth</MenuItem>
            </Select>
          </FormControl>

          <TextField
            fullWidth
            label="Rate Limit (requests/minute)"
            type="number"
            value={apiConfig.rateLimit}
            onChange={(e) => setApiConfig(prev => ({ ...prev, rateLimit: parseInt(e.target.value) }))}
            sx={{ mb: 2 }}
          />

          <FormControlLabel
            control={
              <Switch
                checked={apiConfig.corsEnabled}
                onChange={(e) => setApiConfig(prev => ({ ...prev, corsEnabled: e.target.checked }))}
              />
            }
            label="Enable CORS"
            sx={{ mb: 2 }}
          />

          <Button
            fullWidth
            variant="outlined"
            onClick={handleGenerateEndpoints}
            disabled={!apiConfig.dataSource}
          >
            Generate CRUD Endpoints
          </Button>
        </CardContent>
      </Card>

      {/* Center Panel - Endpoints */}
      <Card sx={{ flex: 1, mr: 2 }}>
        <CardContent>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
            <Typography variant="h6">API Endpoints</Typography>
            <Box>
              <Button
                variant="outlined"
                startIcon={<AddIcon />}
                onClick={handleAddEndpoint}
                sx={{ mr: 1 }}
              >
                Add Endpoint
              </Button>
              <Button
                variant="contained"
                startIcon={<SaveIcon />}
                onClick={() => setShowSaveDialog(true)}
              >
                Save API
              </Button>
            </Box>
          </Box>

          <List>
            {endpoints.map((endpoint) => (
              <ListItem key={endpoint.id} divider>
                <ListItemIcon>
                  <HttpIcon color={
                    endpoint.method === 'GET' ? 'primary' :
                    endpoint.method === 'POST' ? 'success' :
                    endpoint.method === 'PUT' ? 'warning' :
                    endpoint.method === 'DELETE' ? 'error' : 'secondary'
                  } />
                </ListItemIcon>
                <ListItemText
                  primary={
                    <Box sx={{ display: 'flex', alignItems: 'center' }}>
                      <Chip
                        label={endpoint.method}
                        size="small"
                        color={
                          endpoint.method === 'GET' ? 'primary' :
                          endpoint.method === 'POST' ? 'success' :
                          endpoint.method === 'PUT' ? 'warning' :
                          endpoint.method === 'DELETE' ? 'error' : 'default'
                        }
                        sx={{ mr: 1 }}
                      />
                      <Typography variant="body1">{endpoint.path}</Typography>
                    </Box>
                  }
                  secondary={endpoint.description}
                />
                <Box>
                  <Tooltip title="Edit">
                    <IconButton
                      size="small"
                      onClick={() => {
                        setSelectedEndpoint(endpoint);
                        setShowEndpointDialog(true);
                      }}
                    >
                      <EditIcon />
                    </IconButton>
                  </Tooltip>
                  <Tooltip title="Delete">
                    <IconButton
                      size="small"
                      onClick={() => {
                        setEndpoints(prev => prev.filter(e => e.id !== endpoint.id));
                      }}
                    >
                      <DeleteIcon />
                    </IconButton>
                  </Tooltip>
                </Box>
              </ListItem>
            ))}
          </List>

          {endpoints.length === 0 && (
            <Box sx={{ textAlign: 'center', py: 4 }}>
              <ApiIcon sx={{ fontSize: 48, color: 'grey.400', mb: 2 }} />
              <Typography variant="h6" color="textSecondary">
                No endpoints yet
              </Typography>
              <Typography variant="body2" color="textSecondary" sx={{ mb: 2 }}>
                Generate CRUD endpoints or add custom endpoints
              </Typography>
              <Button
                variant="outlined"
                startIcon={<AddIcon />}
                onClick={handleAddEndpoint}
              >
                Add First Endpoint
              </Button>
            </Box>
          )}
        </CardContent>
      </Card>

      {/* Right Panel - API Preview */}
      <Card sx={{ width: 300 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            API Preview
          </Typography>

          <Accordion>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography>OpenAPI Spec</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Button
                fullWidth
                variant="outlined"
                startIcon={<CodeIcon />}
                onClick={() => {/* Generate OpenAPI spec */}}
              >
                Generate Spec
              </Button>
            </AccordionDetails>
          </Accordion>

          <Accordion>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography>Security</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                <SecurityIcon sx={{ mr: 1, color: 'success.main' }} />
                <Typography variant="body2">
                  {apiConfig.authentication === 'none' ? 'No authentication' :
                   apiConfig.authentication === 'jwt' ? 'JWT authentication' :
                   apiConfig.authentication === 'api-key' ? 'API key authentication' :
                   'OAuth authentication'}
                </Typography>
              </Box>
              <Typography variant="body2" color="textSecondary">
                Rate limit: {apiConfig.rateLimit} req/min
              </Typography>
            </AccordionDetails>
          </Accordion>

          <Accordion>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography>Endpoints Summary</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Typography variant="body2">
                Total endpoints: {endpoints.length}
              </Typography>
              <Typography variant="body2">
                GET: {endpoints.filter(e => e.method === 'GET').length}
              </Typography>
              <Typography variant="body2">
                POST: {endpoints.filter(e => e.method === 'POST').length}
              </Typography>
              <Typography variant="body2">
                PUT: {endpoints.filter(e => e.method === 'PUT').length}
              </Typography>
              <Typography variant="body2">
                DELETE: {endpoints.filter(e => e.method === 'DELETE').length}
              </Typography>
            </AccordionDetails>
          </Accordion>
        </CardContent>
      </Card>
    </Box>
  );

  const renderAPILibrary = () => (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
        <Typography variant="h5">API Library</Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => {
            setSelectedAPI(null);
            setApiConfig({
              basePath: '/api',
              authentication: 'jwt',
              rateLimit: 100,
              corsEnabled: true,
              dataSource: ''
            });
            setEndpoints([]);
            setIsEditing(true);
            setActiveTab(1);
          }}
        >
          New API
        </Button>
      </Box>

      <Grid container spacing={2}>
        {apis.map((api) => (
          <Grid item xs={12} md={6} lg={4} key={api.id}>
            <Card>
              <CardContent>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                  <Typography variant="h6">{api.name}</Typography>
                  <Box>
                    {api.isCore && renderCoreCustomChips({ is_core: true })}
                    <Chip
                      label={api.type}
                      color={api.type === 'public' ? 'success' : 'default'}
                      size="small"
                      sx={{ ml: 1 }}
                    />
                  </Box>
                </Box>

                <Typography variant="body2" color="textSecondary" sx={{ mb: 2 }}>
                  {api.description}
                </Typography>

                <Box sx={{ mb: 2 }}>
                  <Typography variant="body2" sx={{ mb: 1 }}>
                    Base Path: <code>{api.config.basePath}</code>
                  </Typography>
                  <Typography variant="body2" sx={{ mb: 1 }}>
                    Endpoints: {api.endpoints.length}
                  </Typography>
                  <Typography variant="body2">
                    Auth: {api.config.authentication}
                  </Typography>
                </Box>

                <Box sx={{ display: 'flex', gap: 1, mb: 2 }}>
                  {api.tags.map((tag) => (
                    <Chip key={tag} label={tag} size="small" variant="outlined" />
                  ))}
                </Box>

                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Typography variant="caption" color="textSecondary">
                    By {api.createdBy} • {new Date(api.updatedAt).toLocaleDateString()}
                  </Typography>

                  <Box>
                    <Tooltip title="Edit">
                      <IconButton
                        size="small"
                        onClick={() => {
                          setSelectedAPI(api);
                          setApiConfig(api.config);
                          setEndpoints(api.endpoints);
                          setIsEditing(true);
                          setActiveTab(1);
                        }}
                        disabled={api.isCore}
                      >
                        <EditIcon />
                      </IconButton>
                    </Tooltip>

                    <Tooltip title="Clone">
                      <IconButton
                        size="small"
                        onClick={() => handleCloneAPI(api)}
                      >
                        <CloneIcon />
                      </IconButton>
                    </Tooltip>

                    <Tooltip title="Share">
                      <IconButton
                        size="small"
                        onClick={() => {
                          setSelectedAPI(api);
                          setShowShareDialog(true);
                        }}
                      >
                        <ShareIcon />
                      </IconButton>
                    </Tooltip>

                    <Tooltip title="Test API">
                      <IconButton
                        size="small"
                        onClick={() => {
                          // Open API testing interface
                        }}
                      >
                        <PlayIcon />
                      </IconButton>
                    </Tooltip>
                  </Box>
                </Box>
              </CardContent>
            </Card>
          </Grid>
        ))}
      </Grid>
    </Box>
  );

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>
        API Builder & CRUD Generator
      </Typography>

      {/* World-class features toolbar */}
      <Box sx={{ mb: 2, display: 'flex', gap: 1, alignItems: 'center' }}>
        <Button
          startIcon={<AIIcon />}
          variant={showAIAssistant ? "contained" : "outlined"}
          onClick={handleToggleAIAssistant}
          size="small"
        >
          AI Assistant
        </Button>
        <Button
          startIcon={<PeopleIcon />}
          variant={showCollaboration ? "contained" : "outlined"}
          onClick={handleToggleCollaboration}
          size="small"
        >
          Collaborate
        </Button>
        <Button
          startIcon={<TemplateIcon />}
          variant="outlined"
          onClick={() => setShowTemplateMarketplace(true)}
          size="small"
        >
          Templates
        </Button>
        <Button
          startIcon={<HistoryIcon />}
          variant="outlined"
          onClick={() => setShowVersionControl(true)}
          size="small"
        >
          Version History
        </Button>
        <Button
          startIcon={<ExportIcon />}
          variant="outlined"
          onClick={handleExportAPI}
          size="small"
        >
          Export
        </Button>
        <Button
          startIcon={<TestIcon />}
          variant="outlined"
          onClick={handleExecuteAPITest}
          size="small"
        >
          Test Suite
        </Button>
      </Box>

      <Tabs value={activeTab} onChange={(_, newValue) => setActiveTab(newValue)} sx={{ mb: 3 }}>
        <Tab label="API Library" />
        <Tab label="API Builder" />
      </Tabs>

      {activeTab === 0 && renderAPILibrary()}
      {activeTab === 1 && renderAPIBuilder()}

      {/* AI Assistant */}
      {showAIAssistant && selectedAPI && (
        <AIQueryAssistant
          query={selectedAPI}
          onSuggestion={handleAISuggestion}
        />
      )}

      {/* Collaboration Panel */}
      {showCollaboration && selectedAPI && (
        <CollaborationPanel
          queryId={selectedAPI.id}
          currentUser={currentUser}
        />
      )}

      {/* Performance Insights */}
      {showPerformanceInsights && (
        <PerformanceInsights
          query={selectedAPI!}
          executionStats={executionStats}
        />
      )}

      {/* Template Marketplace */}
      {showTemplateMarketplace && (
        <TemplateMarketplace onTemplateSelect={handleTemplateSelect} />
      )}

      {/* Version Control */}
      {showVersionControl && selectedAPI && (
        <VersionControl
          queryId={selectedAPI.id}
          onVersionRestore={handleVersionRestore}
        />
      )}

      {/* Export Dialog */}
      {showExportDialog && selectedAPI && (
        <ExportDialog
          query={selectedAPI}
          onClose={() => setShowExportDialog(false)}
        />
      )}

      {/* Testing Suite */}
      {showTestingSuite && selectedAPI && (
        <TestingSuite
          api={selectedAPI}
          onClose={() => setShowTestingSuite(false)}
        />
      )}

      {/* Floating Action Button for Quick Actions */}
      <SpeedDial
        ariaLabel="Quick Actions"
        sx={{ position: 'fixed', bottom: 16, right: 16 }}
        icon={<AddIcon />}
      >
        <SpeedDialAction
          icon={<AIIcon />}
          tooltipTitle="AI Suggestions"
          onClick={handleToggleAIAssistant}
        />
        <SpeedDialAction
          icon={<PeopleIcon />}
          tooltipTitle="Collaborate"
          onClick={handleToggleCollaboration}
        />
        <SpeedDialAction
          icon={<TemplateIcon />}
          tooltipTitle="Templates"
          onClick={() => setShowTemplateMarketplace(true)}
        />
        <SpeedDialAction
          icon={<AnalyticsIcon />}
          tooltipTitle="Performance"
          onClick={() => setShowPerformanceInsights(true)}
        />
        <SpeedDialAction
          icon={<TestIcon />}
          tooltipTitle="Test API"
          onClick={handleExecuteAPITest}
        />
      </SpeedDial>

      {/* Save API Dialog */}
      <Dialog open={showSaveDialog} onClose={() => setShowSaveDialog(false)} maxWidth="sm" fullWidth>
  <ModalHeader title="Save API" onClose={() => setShowSaveDialog(false)} />
        <DialogContent>
          <TextField
            fullWidth
            label="API Name"
            sx={{ mb: 2, mt: 1 }}
            defaultValue={selectedAPI?.name || ''}
          />
          <TextField
            fullWidth
            label="Description"
            multiline
            rows={3}
            sx={{ mb: 2 }}
            defaultValue={selectedAPI?.description || ''}
          />
          <FormControl fullWidth sx={{ mb: 2 }}>
            <InputLabel>Visibility</InputLabel>
            <Select defaultValue={selectedAPI?.type || 'private'}>
              <MenuItem value="private">Private</MenuItem>
              <MenuItem value="public">Public</MenuItem>
            </Select>
          </FormControl>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowSaveDialog(false)}>Cancel</Button>
          <Button
            variant="contained"
            onClick={() => {
              handleSaveAPI('New API', 'Description', 'private');
            }}
          >
            Save
          </Button>
        </DialogActions>
      </Dialog>

      {/* Share Dialog */}
      <Dialog open={showShareDialog} onClose={() => setShowShareDialog(false)} maxWidth="sm" fullWidth>
  <ModalHeader title="Share API" onClose={() => setShowShareDialog(false)} />
        <DialogContent>
          <Typography variant="body2" sx={{ mb: 2 }}>
            Share "{selectedAPI?.name}" with other users or teams
          </Typography>
          <TextField
            fullWidth
            label="Add users or teams"
            placeholder="Enter email or team name"
            sx={{ mb: 2 }}
          />
          <Typography variant="subtitle2" gutterBottom>
            Current shares:
          </Typography>
          {selectedAPI?.sharedWith.map((share) => (
            <Chip
              key={share}
              label={share}
              onDelete={() => {/* Remove share */}}
              sx={{ mr: 1, mb: 1 }}
            />
          ))}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowShareDialog(false)}>Done</Button>
        </DialogActions>
      </Dialog>

      {/* Endpoint Dialog */}
      <Dialog open={showEndpointDialog} onClose={() => setShowEndpointDialog(false)} maxWidth="md" fullWidth>
        <DialogContent>
          <ModalHeader title={selectedEndpoint ? 'Edit Endpoint' : 'Add Endpoint'} onClose={() => setShowEndpointDialog(false)} />
          <Grid container spacing={2} sx={{ mt: 1 }}>
            <Grid item xs={12} sm={6}>
              <TextField
                fullWidth
                label="Path"
                defaultValue={selectedEndpoint?.path || ''}
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <FormControl fullWidth>
                <InputLabel>Method</InputLabel>
                <Select defaultValue={selectedEndpoint?.method || 'GET'}>
                  <MenuItem value="GET">GET</MenuItem>
                  <MenuItem value="POST">POST</MenuItem>
                  <MenuItem value="PUT">PUT</MenuItem>
                  <MenuItem value="DELETE">DELETE</MenuItem>
                  <MenuItem value="PATCH">PATCH</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Description"
                multiline
                rows={2}
                defaultValue={selectedEndpoint?.description || ''}
              />
            </Grid>
            <Grid item xs={12}>
              <FormControl component="fieldset">
                <FormLabel component="legend">Operation Type</FormLabel>
                <RadioGroup
                  row
                  defaultValue={selectedEndpoint?.operation || 'read'}
                >
                  <RadioLabel value="create" control={<Radio />} label="Create" />
                  <RadioLabel value="read" control={<Radio />} label="Read" />
                  <RadioLabel value="update" control={<Radio />} label="Update" />
                  <RadioLabel value="delete" control={<Radio />} label="Delete" />
                  <RadioLabel value="list" control={<Radio />} label="List" />
                </RadioGroup>
              </FormControl>
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowEndpointDialog(false)}>Cancel</Button>
          <Button variant="contained" onClick={() => setShowEndpointDialog(false)}>
            Save Endpoint
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default APIBuilderPage;

// AI API Assistant Component
const AIQueryAssistant: FC<{
  query: API;
  onSuggestion: (suggestion: any) => void;
}> = ({ query, onSuggestion }) => {
  const [suggestions, setSuggestions] = useState<any[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  const getAISuggestions = async () => {
    setIsLoading(true);
    try {
      const response = await fetch('/api/ai/api-suggestions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ api: query })
      });
      const data = await response.json();
      setSuggestions(data.suggestions);
    } catch (error) {
  devError('AI suggestions error:', error);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Card sx={{ mb: 2 }}>
      <CardHeader
        avatar={<AIIcon color="primary" />}
        title="AI API Assistant"
        action={
          <Button
            startIcon={<SuggestionIcon />}
            onClick={getAISuggestions}
            disabled={isLoading}
            size="small"
          >
            {isLoading ? 'Analyzing...' : 'Get Suggestions'}
          </Button>
        }
      />
      <CardContent>
        {suggestions.map((suggestion, index) => (
          <Alert
            key={index}
            severity={suggestion.type}
            sx={{ mb: 1 }}
            action={
              <Button
                size="small"
                onClick={() => onSuggestion(suggestion)}
              >
                Apply
              </Button>
            }
          >
            <Typography variant="body2">{suggestion.message}</Typography>
            {suggestion.impact && (
              <Typography variant="caption" color="text.secondary">
                Impact: {suggestion.impact}
              </Typography>
            )}
          </Alert>
        ))}
      </CardContent>
    </Card>
  );
};

// Real-time Collaboration Component
const CollaborationPanel: FC<{
  queryId: string;
  currentUser: string;
}> = ({ queryId, currentUser }) => {
  const [collaborators, setCollaborators] = useState<any[]>([]);
  const [_socket, _setSocket] = useState<any>(null);
  const [messages, setMessages] = useState<any[]>([]);

  useEffect(() => {
    // Mock socket connection for collaboration
    const mockCollaborators = [
      { id: '1', name: 'Alice Johnson' },
      { id: '2', name: 'Bob Smith' },
      { id: '3', name: 'Carol Davis' }
    ];
    setCollaborators(mockCollaborators);

    const mockMessages = [
      { text: 'Updated authentication method', user: 'Alice', timestamp: new Date() },
      { text: 'Added new endpoint', user: 'Bob', timestamp: new Date() }
    ];
    setMessages(mockMessages);
  }, [queryId, currentUser]);

  return (
    <Card sx={{ position: 'fixed', right: 16, top: 100, width: 300, zIndex: 1000 }}>
      <CardHeader
        title="Collaborators"
        avatar={
          <Badge color="success" variant="dot">
            <PeopleIcon />
          </Badge>
        }
      />
      <CardContent>
        <AvatarGroup max={4}>
          {collaborators.map((collab) => (
            <Tooltip key={collab.id} title={collab.name}>
              <Avatar>{collab.name[0]}</Avatar>
            </Tooltip>
          ))}
        </AvatarGroup>

        <Divider sx={{ my: 2 }} />

        <Typography variant="subtitle2" gutterBottom>
          Recent Activity
        </Typography>
        <List dense>
          {messages.slice(-3).map((msg, index) => (
            <ListItem key={index}>
              <ListItemText
                primary={msg.text}
                secondary={`${msg.user} • ${new Date(msg.timestamp).toLocaleTimeString()}`}
              />
            </ListItem>
          ))}
        </List>
      </CardContent>
    </Card>
  );
};

// Performance Insights Component
const PerformanceInsights: FC<{
  query: API;
  executionStats: any;
}> = ({ query: _query, executionStats }) => {
  const [insights, setInsights] = useState<any[]>([]);

  const analyzePerformance = useCallback(() => {
    const newInsights: any[] = [];

    if (executionStats.avgResponseTime > 1000) {
      newInsights.push({
        type: 'warning',
        title: 'Slow API Response',
        description: 'Average response time is above 1 second',
        suggestion: 'Consider optimizing database queries or adding caching'
      });
    }

    if (executionStats.errorRate > 5) {
      newInsights.push({
        type: 'error',
        title: 'High Error Rate',
        description: 'Error rate is above 5%',
        suggestion: 'Review error handling and add proper validation'
      });
    }

    setInsights(newInsights);
  }, [executionStats]);

  useEffect(() => {
    analyzePerformance();
  }, [analyzePerformance]);

  return (
    <Card sx={{ mb: 2 }}>
      <CardHeader
        avatar={<PerformanceIcon color="primary" />}
        title="API Performance Insights"
        subheader={`Avg Response: ${executionStats.avgResponseTime}ms`}
      />
      <CardContent>
        <Grid container spacing={2}>
          <Grid item xs={6}>
            <Typography variant="body2" color="text.secondary">
              Total Requests
            </Typography>
            <Typography variant="h6">
              {executionStats.totalRequests.toLocaleString()}
            </Typography>
          </Grid>
          <Grid item xs={6}>
            <Typography variant="body2" color="text.secondary">
              Error Rate
            </Typography>
            <Typography variant="h6">
              {executionStats.errorRate}%
            </Typography>
          </Grid>
        </Grid>

        {insights.map((insight, index) => (
          <Alert
            key={index}
            severity={insight.type}
            sx={{ mt: 2 }}
          >
            <Typography variant="subtitle2">{insight.title}</Typography>
            <Typography variant="body2">{insight.description}</Typography>
            <Typography variant="caption" color="text.secondary">
              💡 {insight.suggestion}
            </Typography>
          </Alert>
        ))}
      </CardContent>
    </Card>
  );
};

// Template Marketplace Component
const TemplateMarketplace: FC<{
  onTemplateSelect: (template: any) => void;
}> = ({ onTemplateSelect }) => {
  const [templates, setTemplates] = useState<any[]>([]);
  const [searchTerm, setSearchTerm] = useState('');

  useEffect(() => {
    fetchTemplates();
  }, []);

  const fetchTemplates = async () => {
    // Mock templates
    const mockTemplates = [
      {
        id: '1',
        name: 'E-commerce API',
        category: 'Business',
        description: 'Complete CRUD API for e-commerce platform',
        author: 'API Team',
        rating: 4.5,
        downloads: 1250
      },
      {
        id: '2',
        name: 'User Management',
        category: 'Authentication',
        description: 'User registration, login, and profile management',
        author: 'Auth Team',
        rating: 4.8,
        downloads: 890
      }
    ];
    setTemplates(mockTemplates);
  };

  const filteredTemplates = templates.filter(template =>
    template.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    template.category.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <Dialog open={true} onClose={() => {}} maxWidth="md" fullWidth>
      <ModalHeader title={<Box display="flex" alignItems="center" gap={1}><TemplateIcon />API Templates</Box>} onClose={() => {}} />
      <DialogContent>
        <TextField
          fullWidth
          placeholder="Search templates..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          sx={{ mb: 2 }}
        />

        <Grid container spacing={2}>
          {filteredTemplates.map((template) => (
            <Grid item xs={12} sm={6} md={4} key={template.id}>
              <Card sx={{ cursor: 'pointer' }} onClick={() => onTemplateSelect(template)}>
                <CardHeader
                  avatar={<Avatar>{template.author[0]}</Avatar>}
                  title={template.name}
                  subheader={template.category}
                />
                <CardContent>
                  <Typography variant="body2" color="text.secondary">
                    {template.description}
                  </Typography>
                  <Box display="flex" alignItems="center" gap={1} mt={1}>
                    <Rating value={template.rating} readOnly size="small" />
                    <Typography variant="caption">
                      ({template.downloads} downloads)
                    </Typography>
                  </Box>
                </CardContent>
                <CardActions>
                  <Button size="small" startIcon={<DownloadIcon />}>
                    Use Template
                  </Button>
                </CardActions>
              </Card>
            </Grid>
          ))}
        </Grid>
      </DialogContent>
    </Dialog>
  );
};

// Version Control Component
const VersionControl: FC<{
  queryId: string;
  onVersionRestore: (version: any) => void;
}> = ({ queryId, onVersionRestore }) => {
  const [versions, setVersions] = useState<any[]>([]);

  useEffect(() => {
    fetchVersions();
  }, [queryId]);

  const fetchVersions = async () => {
    // Mock versions
    const mockVersions = [
      {
        id: '1',
        version: '1.2.0',
        author: 'John Doe',
        createdAt: new Date(),
        changes: 'Added authentication middleware'
      },
      {
        id: '2',
        version: '1.1.0',
        author: 'Jane Smith',
        createdAt: new Date(Date.now() - 86400000),
        changes: 'Added user management endpoints'
      }
    ];
    setVersions(mockVersions);
  };

  return (
    <Dialog open={true} onClose={() => {}} maxWidth="md" fullWidth>
      <ModalHeader title={<Box display="flex" alignItems="center" gap={1}><GitIcon />API Version History</Box>} onClose={() => {}} />
      <DialogContent>
        <List>
          {versions.map((version) => (
            <ListItem
              key={version.id}
              secondaryAction={
                <Button
                  size="small"
                  onClick={() => onVersionRestore(version)}
                >
                  Restore
                </Button>
              }
            >
              <ListItemIcon>
                <GitIcon />
              </ListItemIcon>
              <ListItemText
                primary={`Version ${version.version}`}
                secondary={
                  <Box>
                    <Typography variant="caption">
                      {version.author} • {new Date(version.createdAt).toLocaleString()}
                    </Typography>
                    <Typography variant="body2">
                      {version.changes}
                    </Typography>
                  </Box>
                }
              />
            </ListItem>
          ))}
        </List>
      </DialogContent>
    </Dialog>
  );
};

// Advanced Export Options
const ExportDialog: FC<{
  query: API;
  onClose: () => void;
}> = ({ query, onClose }) => {
  const [exportFormat, setExportFormat] = useState('openapi');
  const [includeMetadata, setIncludeMetadata] = useState(true);

  const handleExport = async () => {
    try {
      const response = await fetch(`/api/apis/${query.id}/export`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          format: exportFormat,
          includeMetadata
        })
      });

      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `api-${query.name}.${exportFormat === 'openapi' ? 'yaml' : exportFormat}`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);

      onClose();
    } catch (error) {
      devError('Export failed:', error);
    }
  };

  return (
    <Dialog open={true} onClose={onClose} maxWidth="sm" fullWidth>
      <ModalHeader title="Export API" onClose={onClose} />
      <DialogContent>
        <FormControl fullWidth sx={{ mt: 2 }}>
          <InputLabel>Export Format</InputLabel>
          <Select
            value={exportFormat}
            onChange={(e) => setExportFormat(e.target.value)}
          >
            <MenuItem value="openapi">OpenAPI 3.0</MenuItem>
            <MenuItem value="postman">Postman Collection</MenuItem>
            <MenuItem value="swagger">Swagger 2.0</MenuItem>
            <MenuItem value="json">JSON Schema</MenuItem>
          </Select>
        </FormControl>

        <FormControlLabel
          control={
            <Switch
              checked={includeMetadata}
              onChange={(e) => setIncludeMetadata(e.target.checked)}
            />
          }
          label="Include metadata and examples"
          sx={{ mt: 2 }}
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button onClick={handleExport} variant="contained">
          Export
        </Button>
      </DialogActions>
    </Dialog>
  );
};

// Testing Suite Component
const TestingSuite: FC<{
  api: API;
  onClose: () => void;
}> = ({ api, onClose }) => {
  const [testResults, setTestResults] = useState<any[]>([]);
  const [isRunning, setIsRunning] = useState(false);

  const runTests = async () => {
    setIsRunning(true);
    // Mock test execution
    setTimeout(() => {
      const mockResults = [
        { endpoint: '/customers', method: 'GET', status: 'PASS', responseTime: 245 },
        { endpoint: '/customers', method: 'POST', status: 'PASS', responseTime: 312 },
        { endpoint: '/customers/{id}', method: 'GET', status: 'FAIL', responseTime: 0, error: '404 Not Found' }
      ];
      setTestResults(mockResults);
      setIsRunning(false);
    }, 2000);
  };

  return (
    <Dialog open={true} onClose={onClose} maxWidth="md" fullWidth>
      <ModalHeader title={<Box display="flex" alignItems="center" gap={1}><TestIcon />API Testing Suite</Box>} onClose={onClose} />
      <DialogContent>
        <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
          <Typography variant="h6">{api.name}</Typography>
          <Button
            variant="contained"
            startIcon={<PlayIcon />}
            onClick={runTests}
            disabled={isRunning}
          >
            {isRunning ? 'Running Tests...' : 'Run All Tests'}
          </Button>
        </Box>

        <List>
          {testResults.map((result, index) => (
            <ListItem key={index}>
              <ListItemIcon>
                <Chip
                  label={result.method}
                  size="small"
                  color={result.method === 'GET' ? 'primary' : result.method === 'POST' ? 'success' : 'warning'}
                />
              </ListItemIcon>
              <ListItemText
                primary={result.endpoint}
                secondary={
                  result.status === 'PASS'
                    ? `Response time: ${result.responseTime}ms`
                    : `Error: ${result.error}`
                }
              />
              <Chip
                label={result.status}
                color={result.status === 'PASS' ? 'success' : 'error'}
                size="small"
              />
            </ListItem>
          ))}
        </List>

        {testResults.length === 0 && !isRunning && (
          <Typography variant="body2" color="text.secondary" align="center" sx={{ py: 4 }}>
            Click "Run All Tests" to execute the test suite
          </Typography>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  );
};
