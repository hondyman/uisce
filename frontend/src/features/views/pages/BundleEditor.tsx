import { useState, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Box,
  Typography,
  List,
  ListItem,
  ListItemText,
  IconButton,
  Paper,
  InputAdornment,
  CircularProgress,
  Grid,
} from '@mui/material';
import ModalHeader from '../../../components/ModalHeader';
import { devError } from '../../../utils/devLogger';
import { Add, Delete, Search } from '@mui/icons-material';
import { View, getViewIdentifier } from '../../../types/views';
import { BundleForm } from '../../../types/bundles';
type ViewItem = View;
type Bundle = BundleForm;

interface BundleEditorProps {
  open: boolean;
  onClose: () => void;
  onSave: (bundle: Bundle) => void;
  existingViews: ViewItem[];
}

export const BundleEditor: React.FC<BundleEditorProps> = ({
  open,
  onClose,
  onSave,
  existingViews,
}) => {
  const [bundle, setBundle] = useState<Bundle>({
    name: '',
    description: '',
    audience: [],
    view_refs: [],
  });
  const [searchTerm, setSearchTerm] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (open) {
      // Reset form when dialog opens
      setBundle({
        name: '',
        description: '',
        audience: [],
        view_refs: [],
      });
      setSearchTerm('');
    }
  }, [open]);

  const handleSave = async () => {
    setLoading(true);
    try {
      // In a real app, this would call POST /api/bundles
      await new Promise(resolve => setTimeout(resolve, 1000)); // Simulate network delay
      onSave(bundle);
      onClose();
    } catch (error) {
      try { devError("Failed to save bundle", error); } catch {}
    } finally {
      setLoading(false);
    }
  };

  const addViewToBundle = (view: ViewItem) => {
  const idOrName = getViewIdentifier(view) || view.name;
    if (!bundle.view_refs.some(ref => (ref.view_id || ref.view_name) === idOrName)) {
      setBundle(prev => ({
        ...prev,
        view_refs: [...prev.view_refs, { view_name: view.name, view_id: idOrName } as any],
      }));
    }
  };

  const removeViewFromBundle = (viewIdOrName: string) => {
    setBundle(prev => ({
      ...prev,
      view_refs: prev.view_refs.filter(ref => (ref.view_id || ref.view_name) !== viewIdOrName),
    }));
  };

    const filteredViews = existingViews.filter(view => {
  const idOrName = getViewIdentifier(view) || view.name;
    return (
      (view.title?.toLowerCase().includes(searchTerm.toLowerCase()) ||
       view.name.toLowerCase().includes(searchTerm.toLowerCase())) &&
      !bundle.view_refs.some(ref => (ref.view_id || ref.view_name) === idOrName)
    );
  });

  return (
    <Dialog open={open} onClose={onClose} maxWidth="lg" fullWidth>
      <ModalHeader title="Create Data Bundle" onClose={onClose} />
      <DialogContent>
        <Grid container spacing={3} sx={{ pt: 2 }}>
          {/* Left Panel: Bundle Metadata */}
          <Grid item xs={12} md={4}>
            <Typography variant="h6" gutterBottom>Bundle Details</Typography>
            <TextField
              label="Bundle Name"
              fullWidth
              value={bundle.name}
              onChange={(e) => setBundle({ ...bundle, name: e.target.value })}
              sx={{ mb: 2 }}
            />
            <TextField
              label="Description"
              fullWidth
              multiline
              rows={3}
              value={bundle.description}
              onChange={(e) => setBundle({ ...bundle, description: e.target.value })}
              sx={{ mb: 2 }}
            />
            <TextField
              label="Audience (comma-separated)"
              fullWidth
              value={bundle.audience.join(', ')}
              onChange={(e) => setBundle({ ...bundle, audience: e.target.value.split(',').map(s => s.trim()) })}
              helperText="e.g., executives, risk_analysts"
            />
          </Grid>

          {/* Center Panel: Available Views */}
          <Grid item xs={12} md={4}>
            <Typography variant="h6" gutterBottom>Available Views</Typography>
            <Paper variant="outlined" sx={{ height: 400, display: 'flex', flexDirection: 'column' }}>
              <Box sx={{ p: 1, borderBottom: 1, borderColor: 'divider' }}>
                <TextField
                  fullWidth
                  size="small"
                  placeholder="Search views..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  InputProps={{
                    startAdornment: <InputAdornment position="start"><Search /></InputAdornment>,
                  }}
                />
              </Box>
              <List dense sx={{ overflow: 'auto', flex: 1 }}>
                {filteredViews.map(view => (
                  <ListItem
                      key={getViewIdentifier(view) || view.name}
                    secondaryAction={
                      <IconButton edge="end" aria-label="add" onClick={() => addViewToBundle(view)}>
                        <Add />
                      </IconButton>
                    }
                  >
                      <ListItemText primary={view.title || view.name} secondary={view.description} />
                  </ListItem>
                ))}
              </List>
            </Paper>
          </Grid>

          {/* Right Panel: Selected Views */}
          <Grid item xs={12} md={4}>
            <Typography variant="h6" gutterBottom>Selected Views in Bundle</Typography>
            <Paper variant="outlined" sx={{ height: 400, overflow: 'auto' }}>
              {bundle.view_refs.length === 0 ? (
                <Box sx={{ p: 2, textAlign: 'center', color: 'text.secondary' }}>
                  <Typography>No views selected.</Typography>
                </Box>
              ) : (
                <List dense>
                  {bundle.view_refs.map(ref => (
                    <ListItem
                      key={ref.view_name}
                      secondaryAction={
                        <IconButton edge="end" aria-label="delete" onClick={() => removeViewFromBundle((ref as any).view_id || (ref as any).view_name)}>
                          <Delete />
                        </IconButton>
                      }
                    >
                      <ListItemText primary={ref.view_name} />
                    </ListItem>
                  ))}
                </List>
              )}
            </Paper>
          </Grid>
        </Grid>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button onClick={handleSave} variant="contained" disabled={loading || !bundle.name}>
          {loading ? <CircularProgress size={24} /> : 'Save Bundle'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};