import React, { useState, useEffect, useCallback } from 'react';
import {
  FileText,
  Save,
  Download,
  Settings,
  Database,
  Table,
  BarChart3,
  Filter,
  Grid3x3,
  Type,
  Image,
  Trash2,
  Copy,
  Layers,
  Eye,
  MousePointer,
  Maximize2,
  Square,
  Circle,
} from 'lucide-react';

// Drag and drop imports
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { Rnd } from 'react-rnd';

// Chart library imports
import { 
  Chart as ChartJS, 
  CategoryScale, 
  LinearScale, 
  BarElement, 
  LineElement, 
  PointElement, 
  ArcElement, 
  Title, 
  Tooltip as ChartTooltip, 
  Legend,
  Filler
} from 'chart.js';
import { Bar, Line, Pie, Doughnut } from 'react-chartjs-2';

// MUI components for polished header and controls
import {
  AppBar,
  Toolbar,
  Button,
  Drawer,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  TextField,
  useMediaQuery,
  useTheme,
  Paper,
  Typography,
  Box,
  Grid,
  Card,
  CardContent,
  CardActions,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Slider,
  IconButton,
  Tooltip,
  Chip,
  Avatar,
  SpeedDial,
  SpeedDialAction,
  SpeedDialIcon,
  Alert,
  Snackbar,
  ButtonGroup,
  ToggleButton,
  ToggleButtonGroup,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  // ListItemAvatar,
  // ListItemSecondaryAction
} from '@mui/material';
import { styled, alpha } from '@mui/material/styles';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';

// Register Chart.js components
ChartJS.register(
  CategoryScale,
  LinearScale,
  BarElement,
  LineElement,
  PointElement,
  ArcElement,
  Title,
  ChartTooltip,
  Legend,
  Filler
);

// Styled components for premium UI
const StyledCanvas = styled(Paper)(({ theme }) => ({
  position: 'relative',
  minHeight: '600px',
  background: `linear-gradient(45deg, ${alpha(theme.palette.primary.main, 0.02)} 25%, transparent 25%), 
               linear-gradient(-45deg, ${alpha(theme.palette.primary.main, 0.02)} 25%, transparent 25%), 
               linear-gradient(45deg, transparent 75%, ${alpha(theme.palette.primary.main, 0.02)} 75%), 
               linear-gradient(-45deg, transparent 75%, ${alpha(theme.palette.primary.main, 0.02)} 75%)`,
  backgroundSize: '20px 20px',
  backgroundPosition: '0 0, 0 10px, 10px -10px, -10px 0px',
  border: `2px dashed ${alpha(theme.palette.primary.main, 0.3)}`,
  borderRadius: theme.spacing(2),
  overflow: 'hidden',
  '&:hover': {
    borderColor: theme.palette.primary.main,
    boxShadow: theme.shadows[4],
  },
  transition: 'all 0.3s ease-in-out'
}));

const ToolboxCard = styled(Card)(({ theme }) => ({
  background: `linear-gradient(135deg, ${theme.palette.background.paper} 0%, ${alpha(theme.palette.primary.main, 0.05)} 100%)`,
  border: `1px solid ${alpha(theme.palette.primary.main, 0.1)}`,
  transition: 'all 0.3s ease-in-out',
  '&:hover': {
    transform: 'translateY(-2px)',
    boxShadow: theme.shadows[8],
    borderColor: theme.palette.primary.main,
  }
}));

const ElementToolbox = ({ onAddElement }: { onAddElement: (type: string) => void }) => {
  const theme = useTheme();
  
  const elements = [
    { type: 'chart', label: 'Chart', icon: <BarChart3 />, color: theme.palette.primary.main },
    { type: 'table', label: 'Table', icon: <Table />, color: theme.palette.secondary.main },
    { type: 'text', label: 'Text', icon: <Type />, color: theme.palette.success.main },
    { type: 'image', label: 'Image', icon: <Image />, color: theme.palette.warning.main },
    { type: 'filter', label: 'Filter', icon: <Filter />, color: theme.palette.error.main },
  ];

  return (
    <Box sx={{ p: 2 }}>
      <Typography variant="h6" gutterBottom sx={{ fontWeight: 700, color: 'primary.main' }}>
        <Layers className="inline w-5 h-5 mr-2" />
        Elements
      </Typography>
      <Grid container spacing={2}>
        {elements.map((element) => (
          <Grid item xs={6} key={element.type}>
            <ToolboxCard 
              sx={{ cursor: 'pointer' }}
              onClick={() => onAddElement(element.type)}
            >
              <CardContent sx={{ p: 2, textAlign: 'center' }}>
                <Box sx={{ color: element.color, mb: 1 }}>
                  {element.icon}
                </Box>
                <Typography variant="caption" fontWeight={600}>
                  {element.label}
                </Typography>
              </CardContent>
            </ToolboxCard>
          </Grid>
        ))}
      </Grid>
    </Box>
  );
};

const InteractiveChart = ({ element }: { element: any }) => {
  const [chartData] = useState({
    labels: ['January', 'February', 'March', 'April', 'May', 'June'],
    datasets: [{
      label: 'Sales Data',
      data: [12, 19, 3, 5, 2, 3],
      backgroundColor: [
        'rgba(255, 99, 132, 0.6)',
        'rgba(54, 162, 235, 0.6)',
        'rgba(255, 206, 86, 0.6)',
        'rgba(75, 192, 192, 0.6)',
        'rgba(153, 102, 255, 0.6)',
        'rgba(255, 159, 64, 0.6)',
      ],
      borderColor: [
        'rgba(255, 99, 132, 1)',
        'rgba(54, 162, 235, 1)',
        'rgba(255, 206, 86, 1)',
        'rgba(75, 192, 192, 1)',
        'rgba(153, 102, 255, 1)',
        'rgba(255, 159, 64, 1)',
      ],
      borderWidth: 2,
    }]
  });

  const chartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        position: 'top' as const,
      },
      title: {
        display: true,
        text: element.properties?.title || 'Chart Title',
      },
    },
    animation: {
      duration: 1000,
      easing: 'easeInOutQuart' as const,
    },
  };

  const renderChart = () => {
    switch (element.properties?.chartType || 'bar') {
      case 'line':
        return <Line data={chartData} options={chartOptions} />;
      case 'pie':
        return <Pie data={chartData} options={chartOptions} />;
      case 'doughnut':
        return <Doughnut data={chartData} options={chartOptions} />;
      default:
        return <Bar data={chartData} options={chartOptions} />;
    }
  };

  return (
    <Box sx={{ height: '100%', p: 1 }}>
      {renderChart()}
    </Box>
  );
};

const AdvancedPropertiesPanel = ({ selectedElement, onUpdate }: { selectedElement: any, onUpdate: (updates: any) => void }) => {
  const [activeAccordion, setActiveAccordion] = useState<string | false>('general');

  if (!selectedElement) {
    return (
      <Box sx={{ p: 3, textAlign: 'center' }}>
        <Typography variant="h6" color="text.secondary" gutterBottom>
          <Settings className="inline w-5 h-5 mr-2" />
          Properties
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Select an element to edit its properties
        </Typography>
        <Box sx={{ mt: 2 }}>
          <Avatar sx={{ mx: 'auto', bgcolor: 'primary.main' }}>
            <MousePointer />
          </Avatar>
        </Box>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 2 }}>
      <Typography variant="h6" gutterBottom sx={{ fontWeight: 700, color: 'primary.main' }}>
        <Settings className="inline w-5 h-5 mr-2" />
        Properties
      </Typography>
      
      <Accordion expanded={activeAccordion === 'general'} onChange={() => setActiveAccordion(activeAccordion === 'general' ? false : 'general')}>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="subtitle2" fontWeight={600}>General</Typography>
        </AccordionSummary>
        <AccordionDetails>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <TextField
              label="Element ID"
              value={selectedElement.id}
              variant="outlined"
              size="small"
              disabled
            />
            <TextField
              label="Width"
              value={selectedElement.size?.width || 200}
              variant="outlined"
              size="small"
              type="number"
              onChange={(e) => onUpdate({ 
                size: { ...selectedElement.size, width: parseInt(e.target.value) }
              })}
            />
            <TextField
              label="Height"
              value={selectedElement.size?.height || 150}
              variant="outlined"
              size="small"
              type="number"
              onChange={(e) => onUpdate({ 
                size: { ...selectedElement.size, height: parseInt(e.target.value) }
              })}
            />
          </Box>
        </AccordionDetails>
      </Accordion>

      {selectedElement.type === 'chart' && (
        <Accordion expanded={activeAccordion === 'chart'} onChange={() => setActiveAccordion(activeAccordion === 'chart' ? false : 'chart')}>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="subtitle2" fontWeight={600}>Chart Settings</Typography>
          </AccordionSummary>
          <AccordionDetails>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <FormControl size="small">
                <InputLabel>Chart Type</InputLabel>
                <Select
                  value={selectedElement.properties?.chartType || 'bar'}
                  label="Chart Type"
                  onChange={(e) => onUpdate({ 
                    properties: { ...selectedElement.properties, chartType: e.target.value }
                  })}
                >
                  <MenuItem value="bar">Bar Chart</MenuItem>
                  <MenuItem value="line">Line Chart</MenuItem>
                  <MenuItem value="pie">Pie Chart</MenuItem>
                  <MenuItem value="doughnut">Doughnut Chart</MenuItem>
                </Select>
              </FormControl>
              <TextField
                label="Chart Title"
                value={selectedElement.properties?.title || ''}
                variant="outlined"
                size="small"
                onChange={(e) => onUpdate({ 
                  properties: { ...selectedElement.properties, title: e.target.value }
                })}
              />
            </Box>
          </AccordionDetails>
        </Accordion>
      )}

      <Accordion expanded={activeAccordion === 'style'} onChange={() => setActiveAccordion(activeAccordion === 'style' ? false : 'style')}>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="subtitle2" fontWeight={600}>Style</Typography>
        </AccordionSummary>
        <AccordionDetails>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <TextField
              label="Background Color"
              value={selectedElement.properties?.style?.backgroundColor || '#ffffff'}
              variant="outlined"
              size="small"
              type="color"
              onChange={(e) => onUpdate({ 
                properties: { 
                  ...selectedElement.properties, 
                  style: { ...selectedElement.properties?.style, backgroundColor: e.target.value }
                }
              })}
            />
            <Typography variant="caption">Border Radius</Typography>
            <Slider
              value={selectedElement.properties?.style?.borderRadius || 0}
              onChange={(_, value) => onUpdate({ 
                properties: { 
                  ...selectedElement.properties, 
                  style: { ...selectedElement.properties?.style, borderRadius: value }
                }
              })}
              min={0}
              max={20}
              step={1}
              marks
              valueLabelDisplay="auto"
            />
          </Box>
        </AccordionDetails>
      </Accordion>
    </Box>
  );
};

const LivePreviewPanel = ({ reportConfig }: { reportConfig: any }) => {
  const [previewMode, setPreviewMode] = useState<'desktop' | 'tablet' | 'mobile'>('desktop');
  
  const getPreviewSize = () => {
    switch (previewMode) {
      case 'tablet': return { width: '768px', height: '1024px' };
      case 'mobile': return { width: '375px', height: '667px' };
      default: return { width: '100%', height: '100%' };
    }
  };

  return (
    <Box sx={{ p: 2, height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="h6" sx={{ fontWeight: 700, color: 'primary.main' }}>
          <Eye className="inline w-5 h-5 mr-2" />
          Live Preview
        </Typography>
        <ToggleButtonGroup
          value={previewMode}
          exclusive
          onChange={(_, value) => value && setPreviewMode(value)}
          size="small"
        >
          <ToggleButton value="desktop"><Maximize2 className="w-4 h-4" /></ToggleButton>
          <ToggleButton value="tablet"><Square className="w-4 h-4" /></ToggleButton>
          <ToggleButton value="mobile"><Circle className="w-4 h-4" /></ToggleButton>
        </ToggleButtonGroup>
      </Box>
      
      <Paper 
        sx={{ 
          flex: 1, 
          overflow: 'auto', 
          bgcolor: 'grey.100',
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'flex-start',
          p: 2
        }}
      >
        <Paper 
          sx={{ 
            ...getPreviewSize(),
            maxWidth: '100%',
            maxHeight: '100%',
            bgcolor: '#ffffff',
            boxShadow: 3,
            overflow: 'auto'
          }}
        >
          <Box sx={{ p: 3 }}>
            <Typography variant="h4" gutterBottom>
              {reportConfig?.name || 'Untitled Report'}
            </Typography>
            <Typography variant="body2" color="text.secondary" paragraph>
              {reportConfig?.description || 'Report description'}
            </Typography>
            
            {reportConfig?.elements?.map((element: any) => (
              <Box key={element.id} sx={{ mb: 3, p: 2, border: '1px solid #e0e0e0', borderRadius: 1 }}>
                {element.type === 'chart' && (
                  <Box sx={{ height: 300 }}>
                    <InteractiveChart element={element} />
                  </Box>
                )}
                {element.type === 'text' && (
                  <Typography variant="body1">
                    {element.properties?.textContent || 'Sample text content'}
                  </Typography>
                )}
                {element.type === 'table' && (
                  <Typography variant="body2" color="text.secondary">
                    [Table placeholder - Connect to data source]
                  </Typography>
                )}
              </Box>
            ))}
          </Box>
        </Paper>
      </Paper>
    </Box>
  );
};

// Enhanced Responsive Drawer with premium styling
const ResponsiveDrawer: React.FC<{ activeTab: string; setActiveTab: (t: 'reports' | 'designer' | 'data' | 'preview') => void }> = ({ activeTab, setActiveTab }) => {
  const theme = useTheme();
  const isSmall = useMediaQuery(theme.breakpoints.down('md'));
  const [open, setOpen] = useState(!isSmall);

  useEffect(() => {
    setOpen(!isSmall);
  }, [isSmall]);

  const items = [
    { key: 'reports', label: 'Reports', icon: <FileText />, color: theme.palette.primary.main },
    { key: 'designer', label: 'Designer', icon: <Grid3x3 />, color: theme.palette.secondary.main },
    { key: 'preview', label: 'Preview', icon: <Eye />, color: theme.palette.success.main },
    { key: 'data', label: 'Data', icon: <Database />, color: theme.palette.warning.main },
  ];

  return (
    <Drawer 
      variant={isSmall ? 'temporary' : 'permanent'} 
      open={open} 
      onClose={() => setOpen(false)}
      PaperProps={{ 
        sx: { 
          width: 240,
          background: `linear-gradient(135deg, ${theme.palette.grey[900]} 0%, ${theme.palette.grey[800]} 100%)`,
          color: '#ffffff',
          borderRight: `1px solid ${alpha(theme.palette.primary.main, 0.3)}`
        }
      }}
    >
      <Box sx={{ p: 3, textAlign: 'center', borderBottom: `1px solid ${alpha('#ffffff', 0.1)}` }}>
        <Typography variant="h5" fontWeight={700} sx={{ color: theme.palette.primary.main }}>
          Report Builder
        </Typography>
        <Typography variant="caption" sx={{ color: alpha('#ffffff', 0.7) }}>
          Professional Edition
        </Typography>
      </Box>
      
      <List sx={{ mt: 2 }}>
        {items.map((item) => (
          <ListItemButton 
            key={item.key} 
            selected={activeTab === item.key}
            onClick={() => setActiveTab(item.key as any)}
            sx={{
              mx: 1,
              mb: 1,
              borderRadius: 2,
              '&.Mui-selected': {
                bgcolor: alpha(item.color, 0.2),
                borderLeft: `4px solid ${item.color}`,
                '&:hover': {
                  bgcolor: alpha(item.color, 0.3),
                }
              },
              '&:hover': {
                bgcolor: alpha('#ffffff', 0.1),
              }
            }}
          >
            <ListItemIcon sx={{ color: activeTab === item.key ? item.color : '#ffffff' }}>
              {item.icon}
            </ListItemIcon>
            <ListItemText 
              primary={item.label} 
              primaryTypographyProps={{ fontWeight: activeTab === item.key ? 600 : 400 }}
            />
          </ListItemButton>
        ))}
      </List>
      
      <Box sx={{ mt: 'auto', p: 2, textAlign: 'center', borderTop: `1px solid ${alpha('#ffffff', 0.1)}` }}>
        <Chip 
          label="Pro" 
          size="small" 
          sx={{ 
            bgcolor: theme.palette.primary.main, 
            color: '#ffffff',
            fontWeight: 600
          }} 
        />
        <Typography variant="caption" display="block" sx={{ color: alpha('#ffffff', 0.7), mt: 1 }}>
          v2.0.1
        </Typography>
      </Box>
    </Drawer>
  );
};

export default function WorldClassReportingBuilder() {
  const theme = useTheme();
  const [activeTab, setActiveTab] = useState<'reports' | 'designer' | 'data' | 'preview'>('designer');
  const [selectedReport, setSelectedReport] = useState<any>({
    id: '1',
    name: 'Sales Dashboard',
    description: 'Comprehensive sales analysis report',
    elements: [],
    parameters: [],
    dataSourceId: 'default'
  });
  const [selectedElement, setSelectedElement] = useState<any>(null);
  // preview state intentionally unused for now
  const [notification, setNotification] = useState<{ message: string; severity: 'success' | 'error' | 'info' } | null>(null);

  const addElement = useCallback((type: string) => {
    const newElement = {
      id: `element_${Date.now()}`,
      type,
      position: { x: 50, y: 50 },
      size: { width: 300, height: 200 },
      properties: {
        title: `New ${type.charAt(0).toUpperCase() + type.slice(1)}`,
        ...(type === 'chart' && { chartType: 'bar' }),
        ...(type === 'text' && { textContent: 'Sample text content' }),
        style: {
          backgroundColor: '#ffffff',
          borderRadius: 4,
        }
      }
    };
    
    setSelectedReport((prev: any) => ({
      ...prev,
      elements: [...(prev.elements || []), newElement]
    }));
    
    setSelectedElement(newElement);
    setNotification({ message: `${type.charAt(0).toUpperCase() + type.slice(1)} element added!`, severity: 'success' });
  }, []);

  const updateElement = useCallback((elementId: string, updates: any) => {
    setSelectedReport((prev: any) => ({
      ...prev,
      elements: prev.elements.map((el: any) => 
        el.id === elementId ? { ...el, ...updates } : el
      )
    }));
    
    if (selectedElement?.id === elementId) {
      setSelectedElement((prev: any) => ({ ...prev, ...updates }));
    }
  }, [selectedElement]);

  const renderDesignerCanvas = () => (
    <DndProvider backend={HTML5Backend}>
      <Box sx={{ display: 'flex', height: '100%' }}>
        {/* Left Toolbox */}
        <Paper sx={{ width: 280, borderRadius: 0, borderRight: 1, borderColor: 'divider' }}>
          <ElementToolbox onAddElement={addElement} />
        </Paper>
        
        {/* Main Canvas */}
        <Box sx={{ flex: 1, p: 2, position: 'relative' }}>
          <StyledCanvas sx={{ height: '100%', position: 'relative' }}>
            <Typography 
              variant="h6" 
              sx={{ 
                position: 'absolute', 
                top: 20, 
                left: 20, 
                color: 'text.secondary',
                display: selectedReport.elements.length === 0 ? 'block' : 'none'
              }}
            >
              Drop elements here to start building your report
            </Typography>
            
            {selectedReport.elements.map((element: any) => (
              <Rnd
                key={element.id}
                size={{ width: element.size.width, height: element.size.height }}
                position={{ x: element.position.x, y: element.position.y }}
                onDragStop={(_, d) => updateElement(element.id, { position: { x: d.x, y: d.y } })}
                onResizeStop={(_, _direction, ref, _delta, position) => {
                  updateElement(element.id, {
                    size: { width: ref.offsetWidth, height: ref.offsetHeight },
                    position
                  });
                }}
                bounds="parent"
              >
                <Paper
                  sx={{
                    width: '100%',
                    height: '100%',
                    cursor: 'move',
                    border: selectedElement?.id === element.id ? `2px solid ${theme.palette.primary.main}` : '1px solid #e0e0e0',
                    borderRadius: element.properties?.style?.borderRadius || 1,
                    backgroundColor: element.properties?.style?.backgroundColor || '#ffffff',
                    overflow: 'hidden',
                    transition: 'all 0.2s ease-in-out',
                    '&:hover': {
                      boxShadow: theme.shadows[4],
                      borderColor: theme.palette.primary.light,
                    }
                  }}
                  onClick={() => setSelectedElement(element)}
                >
                  {element.type === 'chart' && (
                    <InteractiveChart element={element} />
                  )}
                  {element.type === 'text' && (
                    <Box sx={{ p: 2, height: '100%', display: 'flex', alignItems: 'center' }}>
                      <Typography variant="body1">
                        {element.properties?.textContent || 'Sample text'}
                      </Typography>
                    </Box>
                  )}
                  {element.type === 'table' && (
                    <Box sx={{ p: 2, height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                      <Box sx={{ textAlign: 'center' }}>
                        <Table className="w-8 h-8 mx-auto mb-2 text-gray-400" />
                        <Typography variant="caption" color="text.secondary">
                          Table Element
                        </Typography>
                      </Box>
                    </Box>
                  )}
                  
                  {selectedElement?.id === element.id && (
                    <Box sx={{ position: 'absolute', top: -40, right: 0, display: 'flex', gap: 1 }}>
                      <Tooltip title="Duplicate">
                        <IconButton size="small" sx={{ bgcolor: 'background.paper', boxShadow: 1 }}>
                          <Copy className="w-4 h-4" />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="Delete">
                        <IconButton 
                          size="small" 
                          sx={{ bgcolor: 'background.paper', boxShadow: 1 }}
                          onClick={() => {
                            setSelectedReport((prev: any) => ({
                              ...prev,
                              elements: prev.elements.filter((el: any) => el.id !== element.id)
                            }));
                            setSelectedElement(null);
                          }}
                        >
                          <Trash2 className="w-4 h-4" />
                        </IconButton>
                      </Tooltip>
                    </Box>
                  )}
                </Paper>
              </Rnd>
            ))}
          </StyledCanvas>
        </Box>
        
        {/* Right Properties Panel */}
        <Paper sx={{ width: 320, borderRadius: 0, borderLeft: 1, borderColor: 'divider' }}>
          <AdvancedPropertiesPanel 
            selectedElement={selectedElement} 
            onUpdate={(updates) => selectedElement && updateElement(selectedElement.id, updates)}
          />
        </Paper>
      </Box>
    </DndProvider>
  );

  const renderActiveTab = () => {
    switch (activeTab) {
      case 'reports':
        return (
          <Box sx={{ p: 3 }}>
            <Typography variant="h4" gutterBottom>Reports</Typography>
            <Grid container spacing={3}>
              <Grid item xs={12} md={6} lg={4}>
                <Card sx={{ cursor: 'pointer' }} onClick={() => setActiveTab('designer')}>
                  <CardContent>
                    <Typography variant="h6">Sales Dashboard</Typography>
                    <Typography variant="body2" color="text.secondary">
                      Comprehensive sales analysis report
                    </Typography>
                  </CardContent>
                  <CardActions>
                    <Button size="small">Edit</Button>
                    <Button size="small">Preview</Button>
                  </CardActions>
                </Card>
              </Grid>
            </Grid>
          </Box>
        );
      case 'designer':
        return renderDesignerCanvas();
      case 'preview':
        return <LivePreviewPanel reportConfig={selectedReport} />;
      case 'data':
        return (
          <Box sx={{ p: 3 }}>
            <Typography variant="h4" gutterBottom>Data Sources</Typography>
            <Typography variant="body1" color="text.secondary">
              Configure your data connections here.
            </Typography>
          </Box>
        );
      default:
        return null;
    }
  };

  return (
    <Box sx={{ display: 'flex', height: '100vh', bgcolor: 'grey.50' }}>
      {/* Enhanced App Bar */}
      <AppBar position="fixed" sx={{ zIndex: theme.zIndex.drawer + 1 }}>
        <Toolbar>
          <Typography variant="h6" sx={{ flexGrow: 1, fontWeight: 700 }}>
            World-Class Report Builder
          </Typography>
          <ButtonGroup variant="contained" sx={{ mr: 2 }}>
            <Button startIcon={<Save />} size="small">
              Save
            </Button>
            <Button startIcon={<Eye />} size="small" onClick={() => setActiveTab('preview')}>
              Preview
            </Button>
            <Button startIcon={<Download />} size="small">
              Export
            </Button>
          </ButtonGroup>
        </Toolbar>
      </AppBar>
      
      {/* Responsive Drawer */}
      <ResponsiveDrawer activeTab={activeTab} setActiveTab={setActiveTab} />
      
      {/* Main Content */}
      <Box 
        component="main" 
        sx={{ 
          flexGrow: 1, 
          mt: 8, 
          ml: { md: '240px' },
          height: 'calc(100vh - 64px)',
          overflow: 'hidden'
        }}
      >
        {renderActiveTab()}
      </Box>

      {/* Speed Dial for Quick Actions */}
      <SpeedDial
        ariaLabel="Quick Actions"
        sx={{ position: 'fixed', bottom: 24, right: 24 }}
        icon={<SpeedDialIcon />}
      >
        <SpeedDialAction
          icon={<BarChart3 />}
          tooltipTitle="Add Chart"
          onClick={() => addElement('chart')}
        />
        <SpeedDialAction
          icon={<Table />}
          tooltipTitle="Add Table"
          onClick={() => addElement('table')}
        />
        <SpeedDialAction
          icon={<Type />}
          tooltipTitle="Add Text"
          onClick={() => addElement('text')}
        />
      </SpeedDial>

      {/* Notification Snackbar */}
      <Snackbar
        open={!!notification}
        autoHideDuration={3000}
        onClose={() => setNotification(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
      >
        <Alert 
          onClose={() => setNotification(null)} 
          severity={notification?.severity}
          variant="filled"
        >
          {notification?.message}
        </Alert>
      </Snackbar>
    </Box>
  );
}