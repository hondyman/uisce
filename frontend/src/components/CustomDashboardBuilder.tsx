import { useState, useCallback, useRef } from 'react';
import {
  Box,
  Typography,
  Paper,
  Grid,
  Button,
  Dialog,
  DialogContent,
  DialogActions,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  IconButton,
  Card,
  CardContent,
  CardActions as _CardActions,
  Chip,
  Divider as _Divider,
  Fab as _Fab,
  Drawer,
  // List components removed - not used in this file
} from '@mui/material';
import ModalHeader from './ModalHeader';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  Save as SaveIcon,
  Settings as SettingsIcon,
  DragIndicator as DragIcon,
  BarChart as BarChartIcon,
  PieChart as PieChartIcon,
  ShowChart as LineChartIcon,
  TableChart as TableIcon,
  TrendingUp as TrendingIcon,
  AccountBalance as PortfolioIcon,
  Assessment as ReportIcon,
  Notifications as NotificationIcon,
} from '@mui/icons-material';
import { DragDropContext, Droppable, Draggable } from 'react-beautiful-dnd';
// Use the DropResult type from react-beautiful-dnd via a type-only import to satisfy TS
import { DropResult as _DropResult } from 'react-beautiful-dnd';

export interface DashboardWidget {
  id: string;
  type: string;
  title: string;
  position: { x: number; y: number };
  size: { width: number; height: number };
  config: any;
  data?: any;
}

export interface Dashboard {
  id: string;
  name: string;
  description?: string;
  widgets: DashboardWidget[];
  layout: 'grid' | 'freeform';
  theme?: string;
  isPublic: boolean;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
}

interface CustomDashboardBuilderProps {
  initialDashboard?: Dashboard;
  availableWidgets: WidgetType[];
  onSave: (dashboard: Dashboard) => void;
  onCancel: () => void;
  userId: string;
}

interface WidgetType {
  id: string;
  name: string;
  description: string;
  icon: React.ReactNode;
  category: string;
  defaultConfig: any;
  defaultSize: { width: number; height: number };
}

const AVAILABLE_WIDGET_TYPES: WidgetType[] = [
  {
    id: 'bar-chart',
    name: 'Bar Chart',
    description: 'Display data as vertical bars',
    icon: <BarChartIcon />,
    category: 'Charts',
    defaultConfig: { dataSource: '', xAxis: '', yAxis: '' },
    defaultSize: { width: 4, height: 3 },
  },
  {
    id: 'line-chart',
    name: 'Line Chart',
    description: 'Display data as connected lines',
    icon: <LineChartIcon />,
    category: 'Charts',
    defaultConfig: { dataSource: '', xAxis: '', yAxis: '' },
    defaultSize: { width: 4, height: 3 },
  },
  {
    id: 'pie-chart',
    name: 'Pie Chart',
    description: 'Display data as proportional segments',
    icon: <PieChartIcon />,
    category: 'Charts',
    defaultConfig: { dataSource: '', valueField: '', labelField: '' },
    defaultSize: { width: 3, height: 3 },
  },
  {
    id: 'data-table',
    name: 'Data Table',
    description: 'Display data in tabular format',
    icon: <TableIcon />,
    category: 'Data',
    defaultConfig: { dataSource: '', columns: [] },
    defaultSize: { width: 6, height: 4 },
  },
  {
    id: 'kpi-card',
    name: 'KPI Card',
    description: 'Display key performance indicators',
    icon: <TrendingIcon />,
    category: 'KPIs',
    defaultConfig: { metric: '', format: 'number', target: null },
    defaultSize: { width: 2, height: 2 },
  },
  {
    id: 'portfolio-summary',
    name: 'Portfolio Summary',
    description: 'Summary of portfolio performance',
    icon: <PortfolioIcon />,
    category: 'Finance',
    defaultConfig: { portfolioId: '', showReturns: true, showRisk: true },
    defaultSize: { width: 4, height: 3 },
  },
  {
    id: 'report-viewer',
    name: 'Report Viewer',
    description: 'Display generated reports',
    icon: <ReportIcon />,
    category: 'Reports',
    defaultConfig: { reportId: '', autoRefresh: false },
    defaultSize: { width: 6, height: 4 },
  },
  {
    id: 'notification-center',
    name: 'Notification Center',
    description: 'Display recent notifications',
    icon: <NotificationIcon />,
    category: 'Communication',
    defaultConfig: { maxItems: 10, showUnreadOnly: false },
    defaultSize: { width: 3, height: 4 },
  },
];

export const CustomDashboardBuilder: React.FC<CustomDashboardBuilderProps> = ({
  initialDashboard,
  availableWidgets = AVAILABLE_WIDGET_TYPES,
  onSave,
  onCancel,
  userId,
}) => {
  const [dashboard, setDashboard] = useState<Dashboard>(
    initialDashboard || {
      id: `dashboard-${Date.now()}`,
      name: 'New Dashboard',
      description: '',
      widgets: [],
      layout: 'grid',
      isPublic: false,
      createdBy: userId,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    }
  );

  const [selectedWidget, setSelectedWidget] = useState<DashboardWidget | null>(null);
  const [widgetDialogOpen, setWidgetDialogOpen] = useState(false);
  const [settingsDialogOpen, setSettingsDialogOpen] = useState(false);
  const [widgetLibraryOpen, setWidgetLibraryOpen] = useState(false);
  const [_draggedWidget, _setDraggedWidget] = useState<WidgetType | null>(null);

  const gridRef = useRef<HTMLDivElement>(null);

  const handleDragEnd = useCallback((result: typeof _DropResult) => {
    if (!result.destination) return;

    const { source, destination, draggableId } = result;

    if (source.droppableId === 'widget-library' && destination.droppableId === 'dashboard-grid') {
      // Adding new widget from library
      const widgetType = availableWidgets.find(w => w.id === draggableId);
      if (widgetType) {
        const newWidget: DashboardWidget = {
          id: `${widgetType.id}-${Date.now()}`,
          type: widgetType.id,
          title: widgetType.name,
          position: { x: destination.index % 12, y: Math.floor(destination.index / 12) },
          size: widgetType.defaultSize,
          config: { ...widgetType.defaultConfig },
        };

        setDashboard(prev => ({
          ...prev,
          widgets: [...prev.widgets, newWidget],
          updatedAt: new Date().toISOString(),
        }));
      }
    } else if (source.droppableId === 'dashboard-grid' && destination.droppableId === 'dashboard-grid') {
      // Reordering existing widgets
      const widgets = Array.from(dashboard.widgets);
      const [reorderedWidget] = widgets.splice(source.index, 1);
      widgets.splice(destination.index, 0, reorderedWidget);

      setDashboard(prev => ({
        ...prev,
        widgets,
        updatedAt: new Date().toISOString(),
      }));
    }
  }, [dashboard.widgets, availableWidgets]);

  const handleAddWidget = (widgetType: WidgetType) => {
    const newWidget: DashboardWidget = {
      id: `${widgetType.id}-${Date.now()}`,
      type: widgetType.id,
      title: widgetType.name,
      position: { x: 0, y: 0 },
      size: widgetType.defaultSize,
      config: { ...widgetType.defaultConfig },
    };

    setDashboard(prev => ({
      ...prev,
      widgets: [...prev.widgets, newWidget],
      updatedAt: new Date().toISOString(),
    }));
    setWidgetLibraryOpen(false);
  };

  const handleEditWidget = (widget: DashboardWidget) => {
    setSelectedWidget(widget);
    setWidgetDialogOpen(true);
  };

  const handleDeleteWidget = (widgetId: string) => {
    setDashboard(prev => ({
      ...prev,
      widgets: prev.widgets.filter(w => w.id !== widgetId),
      updatedAt: new Date().toISOString(),
    }));
  };

  const handleSaveWidget = (updatedWidget: DashboardWidget) => {
    setDashboard(prev => ({
      ...prev,
      widgets: prev.widgets.map(w => w.id === updatedWidget.id ? updatedWidget : w),
      updatedAt: new Date().toISOString(),
    }));
    setWidgetDialogOpen(false);
    setSelectedWidget(null);
  };

  const handleSaveDashboard = () => {
    onSave(dashboard);
  };

  const renderWidget = (widget: DashboardWidget) => {
    const widgetType = availableWidgets.find(w => w.id === widget.type);

    return (
      <Card sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
        <CardContent sx={{ flex: 1, pb: 1 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
            {widgetType?.icon}
            <Typography variant="h6" sx={{ ml: 1, flex: 1 }}>
              {widget.title}
            </Typography>
            <IconButton size="small" onClick={() => handleEditWidget(widget)}>
              <EditIcon />
            </IconButton>
            <IconButton size="small" onClick={() => handleDeleteWidget(widget.id)}>
              <DeleteIcon />
            </IconButton>
          </Box>
          <Typography variant="body2" color="text.secondary">
            {widgetType?.description}
          </Typography>
          {/* Widget content would be rendered here based on type and config */}
          <Box sx={{ mt: 2, p: 2, bgcolor: 'grey.50', borderRadius: 1 }}>
            <Typography variant="body2" color="text.secondary">
              Widget Preview - {widget.type}
            </Typography>
          </Box>
        </CardContent>
      </Card>
    );
  };

  return (
    <Box sx={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <Paper sx={{ p: 2, mb: 2 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Box>
            <TextField
              label="Dashboard Name"
              value={dashboard.name}
              onChange={(e) => setDashboard(prev => ({ ...prev, name: e.target.value }))}
              variant="outlined"
              size="small"
              sx={{ mr: 2 }}
            />
            <TextField
              label="Description"
              value={dashboard.description}
              onChange={(e) => setDashboard(prev => ({ ...prev, description: e.target.value }))}
              variant="outlined"
              size="small"
              sx={{ width: 300 }}
            />
          </Box>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button
              variant="outlined"
              startIcon={<SettingsIcon />}
              onClick={() => setSettingsDialogOpen(true)}
            >
              Settings
            </Button>
            <Button
              variant="outlined"
              startIcon={<AddIcon />}
              onClick={() => setWidgetLibraryOpen(true)}
            >
              Add Widget
            </Button>
            <Button
              variant="contained"
              startIcon={<SaveIcon />}
              onClick={handleSaveDashboard}
            >
              Save Dashboard
            </Button>
            <Button variant="outlined" onClick={onCancel}>
              Cancel
            </Button>
          </Box>
        </Box>
      </Paper>

      {/* Dashboard Canvas */}
      <Box sx={{ flex: 1, overflow: 'auto' }}>
        <DragDropContext onDragEnd={handleDragEnd}>
          <Grid container spacing={2} ref={gridRef}>
            <Droppable droppableId="dashboard-grid">
              {(provided: any, snapshot: any) => (
                <Grid
                  item
                  xs={12}
                  ref={provided.innerRef}
                  {...provided.droppableProps}
                  sx={{
                    minHeight: 400,
                    bgcolor: snapshot.isDraggingOver ? 'grey.100' : 'transparent',
                    border: snapshot.isDraggingOver ? '2px dashed' : '2px solid transparent',
                    borderColor: snapshot.isDraggingOver ? 'primary.main' : 'transparent',
                    borderRadius: 1,
                    p: 2,
                  }}
                >
                  {dashboard.widgets.length === 0 ? (
                    <Box
                      sx={{
                        display: 'flex',
                        flexDirection: 'column',
                        alignItems: 'center',
                        justifyContent: 'center',
                        height: 300,
                        color: 'text.secondary',
                      }}
                    >
                      <Typography variant="h6" gutterBottom>
                        Your dashboard is empty
                      </Typography>
                      <Typography variant="body2">
                        Drag widgets from the library or click "Add Widget" to get started
                      </Typography>
                    </Box>
                  ) : (
                    dashboard.widgets.map((widget, index) => (
                      <Draggable key={widget.id} draggableId={widget.id} index={index}>
                        {(provided: any, snapshot: any) => (
                          <Grid
                            item
                            xs={12}
                            sm={6}
                            md={4}
                            lg={3}
                            ref={provided.innerRef}
                            {...provided.draggableProps}
                            sx={{
                              mb: 2,
                              opacity: snapshot.isDragging ? 0.5 : 1,
                            }}
                          >
                            <Box {...provided.dragHandleProps} sx={{ cursor: 'grab', mb: 1 }}>
                              <DragIcon />
                            </Box>
                            {renderWidget(widget)}
                          </Grid>
                        )}
                      </Draggable>
                    ))
                  )}
                  {provided.placeholder}
                </Grid>
              )}
            </Droppable>
          </Grid>
        </DragDropContext>
      </Box>

      {/* Widget Library Drawer */}
      <Drawer
        anchor="right"
        open={widgetLibraryOpen}
        onClose={() => setWidgetLibraryOpen(false)}
      >
        <Box sx={{ width: 350, p: 2 }}>
          <Typography variant="h6" gutterBottom>
            Widget Library
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Drag widgets to add them to your dashboard
          </Typography>

            <Droppable droppableId="widget-library" isDropDisabled>
            {(provided: any) => (
              <Box ref={provided.innerRef} {...provided.droppableProps}>
                {availableWidgets.map((widgetType, index) => (
                  <Draggable key={widgetType.id} draggableId={widgetType.id} index={index}>
                    {(provided: any, snapshot: any) => (
                      <Card
                        ref={provided.innerRef}
                        {...provided.draggableProps}
                        {...provided.dragHandleProps}
                        sx={{
                          mb: 1,
                          cursor: 'grab',
                          opacity: snapshot.isDragging ? 0.5 : 1,
                        }}
                        onClick={() => handleAddWidget(widgetType)}
                      >
                        <CardContent sx={{ py: 1 }}>
                          <Box sx={{ display: 'flex', alignItems: 'center' }}>
                            {widgetType.icon}
                            <Box sx={{ ml: 2 }}>
                              <Typography variant="subtitle2">
                                {widgetType.name}
                              </Typography>
                              <Typography variant="body2" color="text.secondary">
                                {widgetType.description}
                              </Typography>
                            </Box>
                          </Box>
                        </CardContent>
                      </Card>
                    )}
                    </Draggable>
                ))}
                {provided.placeholder}
              </Box>
            )}
          </Droppable>
        </Box>
      </Drawer>

      {/* Widget Configuration Dialog */}
      <Dialog open={widgetDialogOpen} onClose={() => setWidgetDialogOpen(false)} maxWidth="md" fullWidth>
        <ModalHeader title="Configure Widget" onClose={() => setWidgetDialogOpen(false)} />
        <DialogContent>
          {selectedWidget && (
            <Box sx={{ pt: 2 }}>
              <TextField
                fullWidth
                label="Widget Title"
                value={selectedWidget.title}
                onChange={(e) => setSelectedWidget({ ...selectedWidget, title: e.target.value })}
                sx={{ mb: 2 }}
              />

              <Typography variant="h6" gutterBottom>
                Configuration
              </Typography>

              {/* Configuration fields would be rendered here based on widget type */}
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                {Object.entries(selectedWidget.config).map(([key, value]) => (
                  <TextField
                    key={key}
                    label={key.charAt(0).toUpperCase() + key.slice(1).replace(/([A-Z])/g, ' $1')}
                    value={value}
                    onChange={(e) => setSelectedWidget({
                      ...selectedWidget,
                      config: { ...selectedWidget.config, [key]: e.target.value }
                    })}
                    size="small"
                  />
                ))}
              </Box>
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setWidgetDialogOpen(false)}>Cancel</Button>
          <Button onClick={() => selectedWidget && handleSaveWidget(selectedWidget)} variant="contained">
            Save
          </Button>
        </DialogActions>
      </Dialog>

      {/* Dashboard Settings Dialog */}
      <Dialog open={settingsDialogOpen} onClose={() => setSettingsDialogOpen(false)}>
        <ModalHeader title="Dashboard Settings" onClose={() => setSettingsDialogOpen(false)} />
        <DialogContent>
          <Box sx={{ pt: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
            <FormControl>
              <InputLabel>Layout</InputLabel>
              <Select
                value={dashboard.layout}
                label="Layout"
                onChange={(e) => setDashboard(prev => ({ ...prev, layout: e.target.value as 'grid' | 'freeform' }))}
              >
                <MenuItem value="grid">Grid</MenuItem>
                <MenuItem value="freeform">Freeform</MenuItem>
              </Select>
            </FormControl>

            <FormControl>
              <InputLabel>Theme</InputLabel>
              <Select
                value={dashboard.theme || 'default'}
                label="Theme"
                onChange={(e) => setDashboard(prev => ({ ...prev, theme: e.target.value }))}
              >
                <MenuItem value="default">Default</MenuItem>
                <MenuItem value="dark">Dark</MenuItem>
                <MenuItem value="light">Light</MenuItem>
              </Select>
            </FormControl>

            <Box sx={{ display: 'flex', alignItems: 'center' }}>
              <Typography variant="body2" sx={{ mr: 2 }}>
                Public Dashboard
              </Typography>
              <Chip
                label={dashboard.isPublic ? 'Public' : 'Private'}
                color={dashboard.isPublic ? 'success' : 'default'}
                onClick={() => setDashboard(prev => ({ ...prev, isPublic: !prev.isPublic }))}
                sx={{ cursor: 'pointer' }}
              />
            </Box>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setSettingsDialogOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
