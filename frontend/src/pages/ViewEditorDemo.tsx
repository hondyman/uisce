import { useState } from 'react';
import { 
  Box, 
  TextField, 
  Button, 
  Card, 
  CardContent, 
  Typography,
  Container,
  Paper
} from '@mui/material';
import { Edit, Add } from '@mui/icons-material';
import EnhancedViewEditor from '../components/ViewEditor/EnhancedViewEditor';
import { devLog } from '../utils/devLogger';

const ViewEditorDemo: React.FC = () => {
  const [currentView, setCurrentView] = useState<string>('');
  const [showEditor, setShowEditor] = useState(false);
  const [tenantId] = useState('550e8400-e29b-41d4-a716-446655440000');
  const [datasourceId] = useState('550e8400-e29b-41d4-a716-446655440001');

  const handleOpenEditor = () => {
    if (currentView.trim()) {
      setShowEditor(true);
    }
  };

  const handleCreateNew = () => {
    const newViewName = `new-view-${Date.now()}`;
    setCurrentView(newViewName);
    setShowEditor(true);
  };

  const handleViewSaved = (viewName: string, viewData: any) => {
  devLog('View saved:', { viewName, viewData });
    // You could show a notification, refresh a list, etc.
  };

  const sampleViews = [
    'customer-analytics',
    'sales-dashboard',
    'inventory-metrics',
    'financial-overview'
  ];

  if (showEditor && currentView) {
    return (
      <Box sx={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
        <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
          <Button 
            onClick={() => setShowEditor(false)}
            variant="outlined"
            size="small"
          >
            ← Back to View List
          </Button>
        </Box>
        <Box sx={{ flex: 1 }}>
          <EnhancedViewEditor
            viewName={currentView}
            tenantId={tenantId}
            datasourceId={datasourceId}
            onViewSaved={handleViewSaved}
          />
        </Box>
      </Box>
    );
  }

  return (
    <Container maxWidth="lg" sx={{ py: 4 }}>
      <Typography variant="h3" component="h1" gutterBottom>
        Semantic View Editor Demo
      </Typography>
      
      <Typography variant="body1" color="text.secondary" paragraph>
        This demo shows the enhanced view editor with:
      </Typography>
      
      <Box component="ul" sx={{ mb: 4 }}>
        <li><strong>UI-driven editing:</strong> Use the stats box palette to add cubes, dimensions, measures, and folders</li>
        <li><strong>Code editing:</strong> Switch to code mode for direct JSON editing with syntax highlighting</li>
        <li><strong>Live validation:</strong> Real-time validation with error/warning display</li>
        <li><strong>Auto-save validation:</strong> Automatic validation on save with detailed feedback</li>
        <li><strong>Rich skeleton:</strong> New views start with intelligent templates and examples</li>
      </Box>

      <Paper sx={{ p: 3, mb: 4 }}>
        <Typography variant="h5" gutterBottom>
          Open Existing View
        </Typography>
        
        <Box sx={{ display: 'flex', gap: 2, mb: 3 }}>
          <TextField
            label="View Name"
            value={currentView}
            onChange={(e) => setCurrentView(e.target.value)}
            placeholder="Enter view name or select from samples below"
            fullWidth
          />
          <Button
            variant="contained"
            onClick={handleOpenEditor}
            disabled={!currentView.trim()}
            startIcon={<Edit />}
          >
            Edit View
          </Button>
        </Box>

        <Typography variant="h6" gutterBottom>
          Sample Views:
        </Typography>
        <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
          {sampleViews.map((viewName) => (
            <Button
              key={viewName}
              variant="outlined"
              size="small"
              onClick={() => setCurrentView(viewName)}
            >
              {viewName}
            </Button>
          ))}
        </Box>
      </Paper>

      <Paper sx={{ p: 3 }}>
        <Typography variant="h5" gutterBottom>
          Create New View
        </Typography>
        <Typography variant="body2" color="text.secondary" paragraph>
          Create a new view with a rich skeleton template including example cubes, dimensions, measures, and schema documentation.
        </Typography>
        <Button
          variant="contained"
          onClick={handleCreateNew}
          startIcon={<Add />}
          color="success"
        >
          Create New View
        </Button>
      </Paper>

      <Box sx={{ mt: 4 }}>
        <Card>
          <CardContent>
            <Typography variant="h6" gutterBottom>
              Features Demonstration
            </Typography>
            <Typography variant="body2" paragraph>
              <strong>Palette-driven UI:</strong> Click on the colored chips in the stats bar to add new cubes, dimensions, measures, or folders. Each item type has its own icon and color coding.
            </Typography>
            <Typography variant="body2" paragraph>
              <strong>Dual editing modes:</strong> Toggle between UI editor (visual forms) and Code editor (Monaco with JSON syntax highlighting) using the tabs.
            </Typography>
            <Typography variant="body2" paragraph>
              <strong>Live validation:</strong> Use the "Validate" button for manual validation, or save the view for automatic validation with detailed error/warning feedback.
            </Typography>
            <Typography variant="body2">
              <strong>Rich skeleton:</strong> New views automatically include example cubes, dimensions, measures, and comprehensive schema documentation for IntelliSense.
            </Typography>
          </CardContent>
        </Card>
      </Box>
    </Container>
  );
};

export default ViewEditorDemo;
