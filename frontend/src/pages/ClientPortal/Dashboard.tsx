import React, { useState, useEffect } from 'react';
import { Responsive, WidthProvider, Layout } from 'react-grid-layout';
import {
  Box,
  Paper,
  Typography,
  IconButton,
  Menu,
  MenuItem,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Checkbox,
  Switch,
  FormControlLabel,
  Tabs,
  Tab,
} from '@mui/material';
import {
  Settings as SettingsIcon,
  Add as AddIcon,
  DragIndicator as DragIcon,
} from '@mui/icons-material';
import 'react-grid-layout/css/styles.css';
import 'react-resizable/css/styles.css';

// Import widgets
import { PortfolioWidget } from './widgets/PortfolioWidget';
import { GoalsWidget } from './widgets/GoalsWidget';
import { TransactionsWidget } from './widgets/TransactionsWidget';
import { MeetingsWidget } from './widgets/MeetingsWidget';
import { DocumentsWidget } from './widgets/DocumentsWidget';
import { AccountSummaryWidget } from './widgets/AccountSummaryWidget';
import { NewsWidget } from './widgets/NewsWidget';
import { NBAWidget } from './widgets/NBAWidget';

const ResponsiveGridLayout = WidthProvider(Responsive);

// Widget Registry
interface WidgetConfig {
  id: string;
  title: string;
  component: React.ComponentType<WidgetProps>;
  defaultSize: { w: number; h: number };
  minSize: { w: number; h: number };
  category: 'finance' | 'planning' | 'communication' | 'admin';
  icon: React.ReactNode;
  description: string;
}

const WIDGET_REGISTRY: Record<string, WidgetConfig> = {
  portfolio: {
    id: 'portfolio',
    title: 'Portfolio Performance',
    component: PortfolioWidget,
    defaultSize: { w: 6, h: 4 },
    minSize: { w: 4, h: 3 },
    category: 'finance',
    icon: '📈',
    description: 'View your portfolio returns and allocation',
  },
  goals: {
    id: 'goals',
    title: 'Financial Goals',
    component: GoalsWidget,
    defaultSize: { w: 6, h: 4 },
    minSize: { w: 4, h: 3 },
    category: 'planning',
    icon: '🎯',
    description: 'Track progress toward your goals',
  },
  transactions: {
    id: 'transactions',
    title: 'Recent Transactions',
    component: TransactionsWidget,
    defaultSize: { w: 12, h: 3 },
    minSize: { w: 6, h: 2 },
    category: 'finance',
    icon: '💰',
    description: 'View recent account activity',
  },
  meetings: {
    id: 'meetings',
    title: 'Upcoming Meetings',
    component: MeetingsWidget,
    defaultSize: { w: 6, h: 3 },
    minSize: { w: 4, h: 2 },
    category: 'communication',
    icon: '📅',
    description: 'Manage your advisor meetings',
  },
  documents: {
    id: 'documents',
    title: 'Documents',
    component: DocumentsWidget,
    defaultSize: { w: 6, h: 3 },
    minSize: { w: 4, h: 2 },
    category: 'admin',
    icon: '📄',
    description: 'Access statements and tax documents',
  },
  account_summary: {
    id: 'account_summary',
    title: 'Account Summary',
    component: AccountSummaryWidget,
    defaultSize: { w: 4, h: 3 },
    minSize: { w: 3, h: 2 },
    category: 'finance',
    icon: '💼',
    description: 'Quick overview of all accounts',
  },
  news: {
    id: 'news',
    title: 'Market News & Insights',
    component: NewsWidget,
    defaultSize: { w: 8, h: 4 },
    minSize: { w: 6, h: 3 },
    category: 'planning',
    icon: '📰',
    description: 'Personalized market updates',
  },
  nba_recommendations: {
    id: 'nba_recommendations',
    title: 'Recommended Actions',
    component: NBAWidget,
    defaultSize: { w: 6, h: 4 },
    minSize: { w: 4, h: 3 },
    category: 'planning',
    icon: '🤖',
    description: 'AI-powered recommendations',
  },
};

export const ClientPortalDashboard: React.FC = () => {
  const [layouts, setLayouts] = useState<{ lg: Layout[] }>({ lg: [] });
  const [enabledWidgets, setEnabledWidgets] = useState<string[]>([]);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [addWidgetOpen, setAddWidgetOpen] = useState(false);
  const [editMode, setEditMode] = useState(false);
  const [preferences, setPreferences] = useState<any>(null);

  // Load preferences from API
  useEffect(() => {
    loadPreferences();
  }, []);

  const loadPreferences = async () => {
    try {
      const response = await fetch('/api/portal/preferences');
      const data = await response.json();
      
      if (data.dashboard_layout?.widgets) {
        setLayouts({ lg: data.dashboard_layout.widgets });
      } else {
        // Use default layout
        setLayouts({ lg: getDefaultLayout() });
      }
      
      setEnabledWidgets(data.enabled_widgets || Object.keys(WIDGET_REGISTRY));
      setPreferences(data);
    } catch (error) {
      console.error('Failed to load preferences:', error);
      // Fallback to defaults
      setLayouts({ lg: getDefaultLayout() });
      setEnabledWidgets(Object.keys(WIDGET_REGISTRY));
    }
  };

  const getDefaultLayout = (): Layout[] => {
    return [
      { i: 'portfolio', x: 0, y: 0, w: 6, h: 4, minW: 4, minH: 3 },
      { i: 'goals', x: 6, y: 0, w: 6, h: 4, minW: 4, minH: 3 },
      { i: 'transactions', x: 0, y: 4, w: 12, h: 3, minW: 6, minH: 2 },
      { i: 'meetings', x: 0, y: 7, w: 6, h: 3, minW: 4, minH: 2 },
      { i: 'documents', x: 6, y: 7, w: 6, h: 3, minW: 4, minH: 2 },
    ];
  };

  const saveLayout = async (newLayout: Layout[]) => {
    try {
      await fetch('/api/portal/preferences', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          dashboard_layout: { widgets: newLayout },
        }),
      });

      // Track analytics event
      await fetch('/api/portal/analytics', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          event_type: 'SETTINGS_CHANGED',
          event_data: { setting: 'dashboard_layout' },
        }),
      });
    } catch (error) {
      console.error('Failed to save layout:', error);
    }
  };

  const handleLayoutChange = (layout: Layout[]) => {
    setLayouts({ lg: layout });
    if (editMode) {
      saveLayout(layout);
    }
  };

  const addWidget = (widgetId: string) => {
    const config = WIDGET_REGISTRY[widgetId];
    if (!config) return;

    // Find position for new widget (bottom of grid)
    const maxY = Math.max(...layouts.lg.map(l => l.y + l.h), 0);
    
    const newWidget: Layout = {
      i: widgetId,
      x: 0,
      y: maxY,
      w: config.defaultSize.w,
      h: config.defaultSize.h,
      minW: config.minSize.w,
      minH: config.minSize.h,
    };

    const newLayout = [...layouts.lg, newWidget];
    setLayouts({ lg: newLayout });
    setEnabledWidgets([...enabledWidgets, widgetId]);
    saveLayout(newLayout);
    setAddWidgetOpen(false);
  };

  const removeWidget = (widgetId: string) => {
    const newLayout = layouts.lg.filter(l => l.i !== widgetId);
    setLayouts({ lg: newLayout });
    setEnabledWidgets(enabledWidgets.filter(id => id !== widgetId));
    saveLayout(newLayout);
  };

  const resetToDefault = async () => {
    const defaultLayout = getDefaultLayout();
    setLayouts({ lg: defaultLayout });
    setEnabledWidgets(['portfolio', 'goals', 'transactions', 'meetings', 'documents']);
    await saveLayout(defaultLayout);
  };

  return (
    <Box sx={{ p: 3, bgcolor: 'background.default', minHeight: '100vh' }}>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Box>
          <Typography variant="h4">Welcome Back!</Typography>
          <Typography variant="body2" color="text.secondary">
            {new Date().toLocaleDateString('en-US', { 
              weekday: 'long', 
              year: 'numeric', 
              month: 'long', 
              day: 'numeric' 
            })}
          </Typography>
        </Box>

        <Box sx={{ display: 'flex', gap: 1 }}>
          <FormControlLabel
            control={
              <Switch
                checked={editMode}
                onChange={(e) => setEditMode(e.target.checked)}
              />
            }
            label="Edit Mode"
          />
          <Button
            startIcon={<AddIcon />}
            variant="outlined"
            onClick={() => setAddWidgetOpen(true)}
            disabled={!editMode}
          >
            Add Widget
          </Button>
          <IconButton onClick={() => setSettingsOpen(true)}>
            <SettingsIcon />
          </IconButton>
        </Box>
      </Box>

      {/* Dashboard Grid */}
      <ResponsiveGridLayout
        className="layout"
        layouts={layouts}
        breakpoints={{ lg: 1200, md: 996, sm: 768, xs: 480, xxs: 0 }}
        cols={{ lg: 12, md: 10, sm: 6, xs: 4, xxs: 2 }}
        rowHeight={80}
        onLayoutChange={handleLayoutChange}
        isDraggable={editMode}
        isResizable={editMode}
        compactType="vertical"
        preventCollision={false}
      >
        {layouts.lg.map((item) => {
          const config = WIDGET_REGISTRY[item.i];
          if (!config || !enabledWidgets.includes(item.i)) return null;

          const WidgetComponent = config.component;

          return (
            <Paper
              key={item.i}
              elevation={2}
              sx={{
                p: 2,
                display: 'flex',
                flexDirection: 'column',
                height: '100%',
                overflow: 'hidden',
                position: 'relative',
                '&:hover .drag-handle': editMode ? { opacity: 1 } : {},
              }}
            >
              {editMode && (
                <Box
                  className="drag-handle"
                  sx={{
                    position: 'absolute',
                    top: 8,
                    right: 8,
                    opacity: 0.5,
                    cursor: 'move',
                    transition: 'opacity 0.2s',
                    display: 'flex',
                    gap: 1,
                  }}
                >
                  <DragIcon />
                  <IconButton
                    size="small"
                    onClick={() => removeWidget(item.i)}
                    sx={{ width: 24, height: 24 }}
                  >
                    ×
                  </IconButton>
                </Box>
              )}

              <Typography variant="h6" gutterBottom sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <span>{config.icon}</span>
                {config.title}
              </Typography>

              <Box sx={{ flex: 1, overflow: 'auto' }}>
                <WidgetComponent />
              </Box>
            </Paper>
          );
        })}
      </ResponsiveGridLayout>

      {/* Add Widget Dialog */}
      <Dialog open={addWidgetOpen} onClose={() => setAddWidgetOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Add Widget</DialogTitle>
        <DialogContent>
          <List>
            {Object.values(WIDGET_REGISTRY)
              .filter(config => !enabledWidgets.includes(config.id))
              .map((config) => (
                <ListItem
                  key={config.id}
                  button
                  onClick={() => addWidget(config.id)}
                >
                  <ListItemIcon>
                    <span style={{ fontSize: '24px' }}>{config.icon}</span>
                  </ListItemIcon>
                  <ListItemText
                    primary={config.title}
                    secondary={config.description}
                  />
                </ListItem>
              ))}
          </List>
        </DialogContent>
      </Dialog>

      {/* Settings Dialog */}
      <Dialog open={settingsOpen} onClose={() => setSettingsOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Dashboard Settings</DialogTitle>
        <DialogContent>
          <Box sx={{ mb: 3 }}>
            <Typography variant="subtitle2" gutterBottom>
              Active Widgets
            </Typography>
            <Typography variant="body2" color="text.secondary" gutterBottom>
              {enabledWidgets.length} of {Object.keys(WIDGET_REGISTRY).length} widgets enabled
            </Typography>
            <List>
              {enabledWidgets.map((widgetId) => {
                const config = WIDGET_REGISTRY[widgetId];
                return (
                  <ListItem key={widgetId}>
                    <ListItemIcon>
                      <span>{config.icon}</span>
                    </ListItemIcon>
                    <ListItemText primary={config.title} />
                    <IconButton size="small" onClick={() => removeWidget(widgetId)}>
                      ×
                    </IconButton>
                  </ListItem>
                );
              })}
            </List>
          </Box>

          <Button
            variant="outlined"
            fullWidth
            onClick={resetToDefault}
            sx={{ mt: 2 }}
          >
            Reset to Default Layout
          </Button>
        </DialogContent>
      </Dialog>
    </Box>
  );
};

// Widget Props (shared interface)
export interface WidgetProps {
  // Common props for all widgets
  onRefresh?: () => void;
  onSettings?: () => void;
}
