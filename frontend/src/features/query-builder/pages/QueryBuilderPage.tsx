import { useState, useEffect, useCallback } from 'react';
import { devError } from '../../utils/devLogger';
import {
  Box, Card, CardContent, Typography, Grid, Button, Dialog,
  DialogContent, DialogActions, TextField, Select, MenuItem, FormControl,
  InputLabel, Chip, IconButton, Tabs, Tab, Accordion, AccordionSummary,
  AccordionDetails, List, ListItem, ListItemText, ListItemIcon,
  Switch, FormControlLabel, Tooltip, Avatar, AvatarGroup, Badge, Alert,
  SpeedDial, SpeedDialAction, Divider,
  Rating, CardActions, CardHeader
} from '@mui/material';
import renderCoreCustomChips from '../../../components/common/semanticChips';
import {
  Add as AddIcon,
  PlayArrow as PlayIcon,
  Save as SaveIcon,
  Share as ShareIcon,
  Edit as EditIcon,
  ContentCopy as CloneIcon,
  Delete as _DeleteIcon,
  ExpandMore as ExpandMoreIcon,
  FilterList as FilterIcon,
  Sort as SortIcon,
  GroupWork as GroupIcon,
  Functions as FunctionsIcon,
  TableChart as TableIcon,
  Code as CodeIcon,
  Settings as _SettingsIcon,
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
import ModalHeader from '../../../components/ModalHeader';
import { Droppable, Draggable } from 'react-beautiful-dnd';
import { io as _io } from 'socket.io-client';
import { Socket as _SocketIO } from 'socket.io-client';

// Type definitions for world-class features
interface QuerySuggestion {
  type: 'info' | 'warning' | 'error' | 'success';
  message: string;
  impact?: string;
  suggestedConfig?: Partial<QueryConfig>;
}

interface Collaborator {
  id: string;
  name: string;
  avatar?: string;
  status: 'online' | 'away' | 'offline';
  cursor?: { x: number; y: number };
}

interface ChatMessage {
  id: string;
  user: string;
  text: string;
  timestamp: string;
  type: 'message' | 'system' | 'suggestion';
}

interface QueryExecutionStats {
  executionTime: number;
  rowsScanned: number;
  rowsReturned: number;
  cacheHit: boolean;
  queryPlan?: any;
}

interface PerformanceInsight {
  type: 'info' | 'warning' | 'error';
  title: string;
  description: string;
  suggestion: string;
}

interface QueryTemplate {
  id: string;
  name: string;
  description: string;
  category: string;
  author: string;
  rating: number;
  downloads: number;
  config: QueryConfig;
  tags: string[];
}

interface QueryVersion {
  id: string;
  version: string;
  author: string;
  createdAt: string;
  changes: string;
  config: QueryConfig;
}

// AI Query Assistant Component
const AIQueryAssistant: React.FC<{
  query: Query;
  onSuggestion: (suggestion: QuerySuggestion) => void;
}> = ({ query, onSuggestion }) => {
  const [suggestions, setSuggestions] = useState<QuerySuggestion[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  const getAISuggestions = async () => {
    setIsLoading(true);
    try {
      const response = await fetch('/api/ai/query-suggestions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query })
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
        title="AI Query Assistant"
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
const CollaborationPanel: React.FC<{
  queryId: string;
  currentUser: string;
}> = ({ queryId, currentUser }) => {
  const [collaborators, setCollaborators] = useState<Collaborator[]>([]);
  const [_socket, _setSocket] = useState<any>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);

  useEffect(() => {
  const newSocket = _io('/collaboration', {
      query: { queryId, userId: currentUser }
    });

    newSocket.on('collaborators-update', (data: any) => {
      setCollaborators(data.collaborators);
    });

    newSocket.on('chat-message', (message: any) => {
      setMessages(prev => [...prev, message]);
    });

  _setSocket(newSocket);

    return () => {
  newSocket.disconnect();
    };
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
const PerformanceInsights: React.FC<{
  query: Query;
  executionStats: QueryExecutionStats;
}> = ({ query: _query, executionStats: _executionStats }) => {
  const [insights, setInsights] = useState<PerformanceInsight[]>([]);

  const analyzePerformance = useCallback(() => {
    const newInsights: PerformanceInsight[] = [];

  if (_executionStats.executionTime > 5000) {
      newInsights.push({
        type: 'warning',
        title: 'Slow Query Detected',
        description: 'Consider adding indexes or optimizing joins',
        suggestion: 'Add composite indexes on frequently filtered columns'
      });
    }

  if (_executionStats.rowsScanned > _executionStats.rowsReturned * 10) {
      newInsights.push({
        type: 'info',
        title: 'Inefficient Filtering',
        description: 'Query is scanning more rows than necessary',
        suggestion: 'Review filter conditions and consider partitioning'
      });
    }

    setInsights(newInsights);
  }, [_executionStats]);

  useEffect(() => {
    analyzePerformance();
  }, [analyzePerformance]);

  return (
    <Card sx={{ mb: 2 }}>
      <CardHeader
        avatar={<PerformanceIcon color="primary" />}
        title="Performance Insights"
  subheader={`Execution time: ${_executionStats.executionTime}ms`}
      />
      <CardContent>
        <Grid container spacing={2}>
          <Grid item xs={6}>
            <Typography variant="body2" color="text.secondary">
              Rows Scanned
            </Typography>
            <Typography variant="h6">
              {_executionStats.rowsScanned.toLocaleString()}
            </Typography>
          </Grid>
          <Grid item xs={6}>
            <Typography variant="body2" color="text.secondary">
              Rows Returned
            </Typography>
            <Typography variant="h6">
              {_executionStats.rowsReturned.toLocaleString()}
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
const TemplateMarketplace: React.FC<{
  onTemplateSelect: (template: QueryTemplate) => void;
}> = ({ onTemplateSelect }) => {
  const [templates, setTemplates] = useState<QueryTemplate[]>([]);
  const [searchTerm, setSearchTerm] = useState('');

  useEffect(() => {
    fetchTemplates();
  }, []);

  const fetchTemplates = async () => {
    try {
      const response = await fetch('/api/templates/queries');
      const data = await response.json();
      setTemplates(data.templates);
    } catch (error) {
      devError('Failed to fetch templates:', error);
    }
  };

  const filteredTemplates = templates.filter(template =>
    template.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    template.category.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <Dialog open={true} onClose={() => {}} maxWidth="md" fullWidth>
      <ModalHeader title={<Box display="flex" alignItems="center" gap={1}><TemplateIcon />Query Templates</Box>} onClose={() => {}} />
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
const VersionControl: React.FC<{
  queryId: string;
  onVersionRestore: (version: QueryVersion) => void;
}> = ({ queryId, onVersionRestore }) => {
  const [versions, setVersions] = useState<QueryVersion[]>([]);
  const [_selectedVersion, _setSelectedVersion] = useState<QueryVersion | null>(null);

  const fetchVersions = useCallback(async () => {
    try {
      const response = await fetch(`/api/queries/${queryId}/versions`);
      const data = await response.json();
      setVersions(data.versions);
    } catch (error) {
      devError('Failed to fetch versions:', error);
    }
  }, [queryId]);

  useEffect(() => {
    fetchVersions();
  }, [fetchVersions]);

  return (
    <Dialog open={true} onClose={() => {}} maxWidth="md" fullWidth>
      <ModalHeader title={<Box display="flex" alignItems="center" gap={1}><GitIcon />Version History</Box>} onClose={() => {}} />
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
const ExportDialog: React.FC<{
  query: Query;
  onClose: () => void;
}> = ({ query, onClose }) => {
  const [exportFormat, setExportFormat] = useState('json');
  const [includeMetadata, setIncludeMetadata] = useState(true);

  const handleExport = async () => {
    try {
      const response = await fetch(`/api/queries/${query.id}/export`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          format: exportFormat,
          includeMetadata,
          destination: 'download' // or 'powerbi', 'tableau', etc.
        })
      });

      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `query-${query.name}.${exportFormat}`;
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
      <ModalHeader title="Export Query" onClose={onClose} />
      <DialogContent>
        <FormControl fullWidth sx={{ mt: 2 }}>
          <InputLabel>Export Format</InputLabel>
          <Select
            value={exportFormat}
            onChange={(e) => setExportFormat(e.target.value)}
          >
            <MenuItem value="json">JSON</MenuItem>
            <MenuItem value="sql">SQL</MenuItem>
            <MenuItem value="yaml">YAML</MenuItem>
            <MenuItem value="powerbi">Power BI</MenuItem>
            <MenuItem value="tableau">Tableau</MenuItem>
            <MenuItem value="excel">Excel</MenuItem>
          </Select>
        </FormControl>

        <FormControlLabel
          control={
            <Switch
              checked={includeMetadata}
              onChange={(e) => setIncludeMetadata(e.target.checked)}
            />
          }
          label="Include metadata and comments"
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

interface Query {
  id: string;
  name: string;
  description: string;
  type: 'public' | 'private';
  createdBy: string;
  createdAt: string;
  updatedAt: string;
  isCore: boolean;
  config: QueryConfig;
  tags: string[];
  sharedWith: string[];
}

interface QueryConfig {
  dataSource: string;
  measures: Measure[];
  dimensions: Dimension[];
  filters: Filter[];
  joins: Join[];
  aggregations: Aggregation[];
  sorting: Sort[];
  limit?: number;
}

interface Measure {
  id: string;
  name: string;
  type: string;
  aggregation?: string;
}

interface Dimension {
  id: string;
  name: string;
  type: string;
}

interface Filter {
  id: string;
  field: string;
  operator: string;
  value: any;
}

interface Join {
  id: string;
  table: string;
  type: 'INNER' | 'LEFT' | 'RIGHT' | 'FULL';
  condition: string;
}

interface Aggregation {
  id: string;
  field: string;
  function: string;
}

interface Sort {
  field: string;
  direction: 'ASC' | 'DESC';
}

const QueryBuilderPage: React.FC = () => {
  const [queries, setQueries] = useState<Query[]>([]);
  const [selectedQuery, setSelectedQuery] = useState<Query | null>(null);
  const [queryConfig, setQueryConfig] = useState<QueryConfig>({
    dataSource: '',
    measures: [],
    dimensions: [],
    filters: [],
    joins: [],
    aggregations: [],
    sorting: []
  });
  const [_isEditing, setIsEditing] = useState(false);
  const [showSaveDialog, setShowSaveDialog] = useState(false);
  const [showShareDialog, setShowShareDialog] = useState(false);
  const [activeTab, setActiveTab] = useState(0);
  const [playgroundMode, setPlaygroundMode] = useState(false);

  // World-class feature states
  const [showAIAssistant, setShowAIAssistant] = useState(false);
  const [showCollaboration, setShowCollaboration] = useState(false);
  const [showPerformanceInsights, setShowPerformanceInsights] = useState(false);
  const [showTemplateMarketplace, setShowTemplateMarketplace] = useState(false);
  const [showVersionControl, setShowVersionControl] = useState(false);
  const [showExportDialog, setShowExportDialog] = useState(false);
  const [executionStats, setExecutionStats] = useState<QueryExecutionStats>({
    executionTime: 0,
    rowsScanned: 0,
    rowsReturned: 0,
    cacheHit: false
  });
  const [_aiSuggestions, setAiSuggestions] = useState<QuerySuggestion[]>([]);
  const [currentUser] = useState('john.doe'); // Replace with actual user context

  // Available data sources and fields
  const [dataSources] = useState([
    { id: 'orders', name: 'Orders', fields: ['id', 'customer_id', 'order_date', 'total_amount', 'status'] },
    { id: 'customers', name: 'Customers', fields: ['id', 'name', 'email', 'created_at', 'segment'] },
    { id: 'products', name: 'Products', fields: ['id', 'name', 'category', 'price', 'stock_quantity'] }
  ]);

  useEffect(() => {
    loadQueries();
  }, []);

  const loadQueries = async () => {
    // Mock data - replace with actual API call
    const mockQueries: Query[] = [
      {
        id: '1',
        name: 'Monthly Sales Report',
        description: 'Sales performance by month and product category',
        type: 'public',
        createdBy: 'john.doe',
        createdAt: '2024-01-15',
        updatedAt: '2024-01-15',
        isCore: true,
        config: {
          dataSource: 'orders',
          measures: [{ id: 'total_amount', name: 'Total Amount', type: 'number', aggregation: 'SUM' }],
          dimensions: [{ id: 'order_date', name: 'Order Date', type: 'date' }],
          filters: [],
          joins: [],
          aggregations: [],
          sorting: [{ field: 'order_date', direction: 'DESC' }]
        },
        tags: ['sales', 'monthly'],
        sharedWith: ['team.marketing', 'team.sales']
      }
    ];
    setQueries(mockQueries);
  };

  // Drag-and-drop handlers are not used in this page variant.

  const handleSaveQuery = async (name: string, description: string, type: 'public' | 'private') => {
    const newQuery: Query = {
      id: selectedQuery?.id || Date.now().toString(),
      name,
      description,
      type,
      createdBy: 'current.user', // Replace with actual user
      createdAt: selectedQuery?.createdAt || new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      isCore: false,
      config: queryConfig,
      tags: [],
      sharedWith: []
    };

    if (selectedQuery) {
      setQueries(prev => prev.map(q => q.id === selectedQuery.id ? newQuery : q));
    } else {
      setQueries(prev => [...prev, newQuery]);
    }

    setShowSaveDialog(false);
    setSelectedQuery(newQuery);
  };

  const handleCloneQuery = (query: Query) => {
    const clonedQuery: Query = {
      ...query,
      id: Date.now().toString(),
      name: `${query.name} (Copy)`,
      isCore: false,
      createdBy: 'current.user',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    };
    setQueries(prev => [...prev, clonedQuery]);
    setSelectedQuery(clonedQuery);
    setQueryConfig(clonedQuery.config);
    setIsEditing(true);
  };

  const handleExecuteQuery = async () => {
    // Mock execution - replace with actual API call

    // Simulate execution stats for demo
    setExecutionStats({
      executionTime: Math.random() * 2000 + 500,
      rowsScanned: Math.floor(Math.random() * 100000) + 10000,
      rowsReturned: Math.floor(Math.random() * 1000) + 100,
      cacheHit: Math.random() > 0.5
    });

    setShowPerformanceInsights(true);
  };

  // World-class feature handlers
  const handleAISuggestion = (suggestion: QuerySuggestion) => {
    if (suggestion.suggestedConfig) {
      setQueryConfig(prev => ({ ...prev, ...suggestion.suggestedConfig }));
    }
    setAiSuggestions(prev => prev.filter(s => s !== suggestion));
  };

  const handleTemplateSelect = (template: QueryTemplate) => {
    setQueryConfig(template.config);
    setSelectedQuery(null);
    setShowTemplateMarketplace(false);
    setIsEditing(true);
  };

  const handleVersionRestore = (version: QueryVersion) => {
    setQueryConfig(version.config);
    setShowVersionControl(false);
  };

  const handleExportQuery = () => {
    setShowExportDialog(true);
  };

  const handleToggleAIAssistant = () => {
    setShowAIAssistant(!showAIAssistant);
  };

  const handleToggleCollaboration = () => {
    setShowCollaboration(!showCollaboration);
  };

  const renderQueryBuilder = () => (
    <Box sx={{ display: 'flex', height: 'calc(100vh - 200px)' }}>
      {/* Left Panel - Available Fields */}
      <Card sx={{ width: 300, mr: 2, overflow: 'auto' }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Available Fields
          </Typography>
          <FormControl fullWidth sx={{ mb: 2 }}>
            <InputLabel>Data Source</InputLabel>
            <Select
              value={queryConfig.dataSource}
              onChange={(e) => setQueryConfig(prev => ({ ...prev, dataSource: e.target.value }))}
            >
              {dataSources.map(ds => (
                <MenuItem key={ds.id} value={ds.id}>{ds.name}</MenuItem>
              ))}
            </Select>
          </FormControl>

          {queryConfig.dataSource && (
            <Droppable droppableId="available-fields">
              {(provided: any) => (
                <Box ref={provided.innerRef} {...provided.droppableProps}>
                  {dataSources.find(ds => ds.id === queryConfig.dataSource)?.fields.map((field, index) => (
                    <Draggable key={field} draggableId={field} index={index}>
                      {(provided: any) => (
                        <Card
                          ref={provided.innerRef}
                          {...provided.draggableProps}
                          {...provided.dragHandleProps}
                          sx={{ mb: 1, p: 1, cursor: 'grab' }}
                        >
                          <Typography variant="body2">{field}</Typography>
                        </Card>
                      )}
                    </Draggable>
                  ))}
                  {provided.placeholder}
                </Box>
              )}
            </Droppable>
          )}
        </CardContent>
      </Card>

      {/* Center Panel - Query Canvas */}
      <Card sx={{ flex: 1, mr: 2 }}>
        <CardContent>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
            <Typography variant="h6">Query Builder</Typography>
            <Box>
              <Button
                variant="outlined"
                startIcon={<PlayIcon />}
                onClick={handleExecuteQuery}
                sx={{ mr: 1 }}
              >
                Execute
              </Button>
              <Button
                variant="contained"
                startIcon={<SaveIcon />}
                onClick={() => setShowSaveDialog(true)}
              >
                Save Query
              </Button>
            </Box>
          </Box>

          <Droppable droppableId="query-fields">
            {(provided: any) => (
              <Box ref={provided.innerRef} {...provided.droppableProps} sx={{ minHeight: 200, border: '2px dashed #ccc', p: 2 }}>
                <Typography variant="body2" color="textSecondary" sx={{ mb: 2 }}>
                  Drag fields here to build your query
                </Typography>

                {/* Dimensions */}
                {queryConfig.dimensions.length > 0 && (
                  <Box sx={{ mb: 2 }}>
                    <Typography variant="subtitle2" sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                      <TableIcon sx={{ mr: 1 }} />
                      Dimensions
                    </Typography>
                    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                      {queryConfig.dimensions.map((dim, index) => (
                        <Draggable key={dim.id} draggableId={`dim-${dim.id}`} index={index}>
                          {(provided: any) => (
                            <Chip
                              ref={provided.innerRef}
                              {...provided.draggableProps}
                              {...provided.dragHandleProps}
                              label={dim.name}
                              onDelete={() => {
                                setQueryConfig(prev => ({
                                  ...prev,
                                  dimensions: prev.dimensions.filter(d => d.id !== dim.id)
                                }));
                              }}
                              color="primary"
                              variant="outlined"
                            />
                          )}
                        </Draggable>
                      ))}
                    </Box>
                  </Box>
                )}

                {/* Measures */}
                {queryConfig.measures.length > 0 && (
                  <Box sx={{ mb: 2 }}>
                    <Typography variant="subtitle2" sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                      <FunctionsIcon sx={{ mr: 1 }} />
                      Measures
                    </Typography>
                    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                      {queryConfig.measures.map((measure, index) => (
                        <Draggable key={measure.id} draggableId={`measure-${measure.id}`} index={index}>
                           {(provided: any) => (
                            <Chip
                              ref={provided.innerRef}
                              {...provided.draggableProps}
                              {...provided.dragHandleProps}
                              label={`${measure.aggregation}(${measure.name})`}
                              onDelete={() => {
                                setQueryConfig(prev => ({
                                  ...prev,
                                  measures: prev.measures.filter(m => m.id !== measure.id)
                                }));
                              }}
                              color="secondary"
                              variant="outlined"
                            />
                          )}
                        </Draggable>
                      ))}
                    </Box>
                  </Box>
                )}

                {provided.placeholder}
              </Box>
            )}
          </Droppable>
        </CardContent>
      </Card>

      {/* Right Panel - Query Properties */}
      <Card sx={{ width: 300 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Query Properties
          </Typography>

          <Accordion>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography>Filters</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Button
                fullWidth
                variant="outlined"
                startIcon={<FilterIcon />}
                onClick={() => {/* Add filter logic */}}
              >
                Add Filter
              </Button>
            </AccordionDetails>
          </Accordion>

          <Accordion>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography>Sorting</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Button
                fullWidth
                variant="outlined"
                startIcon={<SortIcon />}
                onClick={() => {/* Add sort logic */}}
              >
                Add Sort
              </Button>
            </AccordionDetails>
          </Accordion>

          <Accordion>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography>Aggregations</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Button
                fullWidth
                variant="outlined"
                startIcon={<GroupIcon />}
                onClick={() => {/* Add aggregation logic */}}
              >
                Add Aggregation
              </Button>
            </AccordionDetails>
          </Accordion>
        </CardContent>
      </Card>
    </Box>
  );

  const renderQueryLibrary = () => (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
        <Typography variant="h5">Query Library</Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => {
            setSelectedQuery(null);
            setQueryConfig({
              dataSource: '',
              measures: [],
              dimensions: [],
              filters: [],
              joins: [],
              aggregations: [],
              sorting: []
            });
            setIsEditing(true);
            setActiveTab(1);
          }}
        >
          New Query
        </Button>
      </Box>

      <Grid container spacing={2}>
        {queries.map((query) => (
          <Grid item xs={12} md={6} lg={4} key={query.id}>
            <Card>
              <CardContent>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                  <Typography variant="h6">{query.name}</Typography>
                  <Box>
                    {query.isCore && renderCoreCustomChips({ is_core: true })}
                    <Chip
                      label={query.type}
                      color={query.type === 'public' ? 'success' : 'default'}
                      size="small"
                      sx={{ ml: 1 }}
                    />
                  </Box>
                </Box>

                <Typography variant="body2" color="textSecondary" sx={{ mb: 2 }}>
                  {query.description}
                </Typography>

                <Box sx={{ display: 'flex', gap: 1, mb: 2 }}>
                  {query.tags.map((tag) => (
                    <Chip key={tag} label={tag} size="small" variant="outlined" />
                  ))}
                </Box>

                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Typography variant="caption" color="textSecondary">
                    By {query.createdBy} • {new Date(query.updatedAt).toLocaleDateString()}
                  </Typography>

                  <Box>
                    <Tooltip title="Edit">
                      <IconButton
                        size="small"
                        onClick={() => {
                          setSelectedQuery(query);
                          setQueryConfig(query.config);
                          setIsEditing(true);
                          setActiveTab(1);
                        }}
                        disabled={query.isCore}
                      >
                        <EditIcon />
                      </IconButton>
                    </Tooltip>

                    <Tooltip title="Clone">
                      <IconButton
                        size="small"
                        onClick={() => handleCloneQuery(query)}
                      >
                        <CloneIcon />
                      </IconButton>
                    </Tooltip>

                    <Tooltip title="Share">
                      <IconButton
                        size="small"
                        onClick={() => {
                          setSelectedQuery(query);
                          setShowShareDialog(true);
                        }}
                      >
                        <ShareIcon />
                      </IconButton>
                    </Tooltip>

                    <Tooltip title="Execute">
                      <IconButton
                        size="small"
                        onClick={() => {
                          setSelectedQuery(query);
                          setQueryConfig(query.config);
                          setPlaygroundMode(true);
                          setActiveTab(2);
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

  const renderPlayground = () => (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
        <Typography variant="h5">Query Playground</Typography>
        <FormControlLabel
          control={
            <Switch
              checked={playgroundMode}
              onChange={(e) => setPlaygroundMode(e.target.checked)}
            />
          }
          label="Simulation Mode"
        />
      </Box>

      <Grid container spacing={2}>
        <Grid item xs={12} md={8}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Generated SQL
              </Typography>
              <Box sx={{ bgcolor: 'grey.100', p: 2, borderRadius: 1, fontFamily: 'monospace' }}>
                {generateSQL(queryConfig)}
              </Box>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={4}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Execution Options
              </Typography>
              <Button
                fullWidth
                variant="contained"
                startIcon={<PlayIcon />}
                onClick={handleExecuteQuery}
                sx={{ mb: 1 }}
              >
                {playgroundMode ? 'Simulate Execution' : 'Execute Query'}
              </Button>

              <Button
                fullWidth
                variant="outlined"
                startIcon={<CodeIcon />}
                onClick={() => {/* Show results */}}
              >
                View Results
              </Button>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );

  const generateSQL = (config: QueryConfig): string => {
    if (!config.dataSource) return 'SELECT * FROM table_name';

    let sql = 'SELECT ';

    if (config.measures.length > 0) {
      sql += config.measures.map(m => `${m.aggregation}(${m.name}) as ${m.name}`).join(', ');
    } else {
      sql += '*';
    }

    sql += ` FROM ${config.dataSource}`;

    if (config.dimensions.length > 0) {
      sql += ` GROUP BY ${config.dimensions.map(d => d.name).join(', ')}`;
    }

    if (config.sorting.length > 0) {
      sql += ` ORDER BY ${config.sorting.map(s => `${s.field} ${s.direction}`).join(', ')}`;
    }

    if (config.limit) {
      sql += ` LIMIT ${config.limit}`;
    }

    return sql;
  };

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Query Builder & API Creator
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
          onClick={handleExportQuery}
          size="small"
        >
          Export
        </Button>
      </Box>

      <Tabs value={activeTab} onChange={(_, newValue) => setActiveTab(newValue)} sx={{ mb: 3 }}>
        <Tab label="Query Library" />
        <Tab label="Query Builder" />
        <Tab label="Playground" />
      </Tabs>

      {/* AI Assistant */}
      {showAIAssistant && selectedQuery && (
        <AIQueryAssistant
          query={selectedQuery}
          onSuggestion={handleAISuggestion}
        />
      )}

      {/* Performance Insights */}
      {showPerformanceInsights && (
        <PerformanceInsights
          query={selectedQuery!}
          executionStats={executionStats}
        />
      )}

      {activeTab === 0 && renderQueryLibrary()}
      {activeTab === 1 && renderQueryBuilder()}
      {activeTab === 2 && renderPlayground()}

      {/* Collaboration Panel */}
      {showCollaboration && selectedQuery && (
        <CollaborationPanel
          queryId={selectedQuery.id}
          currentUser={currentUser}
        />
      )}

      {/* Template Marketplace */}
      {showTemplateMarketplace && (
        <TemplateMarketplace onTemplateSelect={handleTemplateSelect} />
      )}

      {/* Version Control */}
      {showVersionControl && selectedQuery && (
        <VersionControl
          queryId={selectedQuery.id}
          onVersionRestore={handleVersionRestore}
        />
      )}

      {/* Export Dialog */}
      {showExportDialog && selectedQuery && (
        <ExportDialog
          query={selectedQuery}
          onClose={() => setShowExportDialog(false)}
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
          tooltipTitle="Test Query"
          onClick={handleExecuteQuery}
        />
      </SpeedDial>

      {/* Save Query Dialog */}
      <Dialog open={showSaveDialog} onClose={() => setShowSaveDialog(false)} maxWidth="sm" fullWidth>
        <ModalHeader title="Save Query" onClose={() => setShowSaveDialog(false)} />
        <DialogContent>
          <TextField
            fullWidth
            label="Query Name"
            sx={{ mb: 2, mt: 1 }}
            defaultValue={selectedQuery?.name || ''}
          />
          <TextField
            fullWidth
            label="Description"
            multiline
            rows={3}
            sx={{ mb: 2 }}
            defaultValue={selectedQuery?.description || ''}
          />
          <FormControl fullWidth sx={{ mb: 2 }}>
            <InputLabel>Visibility</InputLabel>
            <Select defaultValue={selectedQuery?.type || 'private'}>
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
              // Get values from form and save
              handleSaveQuery('New Query', 'Description', 'private');
            }}
          >
            Save
          </Button>
        </DialogActions>
      </Dialog>

      {/* Share Dialog */}
      <Dialog open={showShareDialog} onClose={() => setShowShareDialog(false)} maxWidth="sm" fullWidth>
        <ModalHeader title="Share Query" onClose={() => setShowShareDialog(false)} />
        <DialogContent>
          <Typography variant="body2" sx={{ mb: 2 }}>
            Share "{selectedQuery?.name}" with other users or teams
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
          {selectedQuery?.sharedWith.map((share) => (
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
    </Box>
  );
};

export default QueryBuilderPage;
