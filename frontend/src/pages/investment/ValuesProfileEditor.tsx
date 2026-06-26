import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Paper,
  Grid,
  TextField,
  Button,
  Chip,
  Switch,
  FormControlLabel,
  Divider,
  List,
  ListItem,
  ListItemText,
  ListItemSecondaryAction,
  IconButton,
  CircularProgress,
  Alert
} from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';
import AutoFixHighIcon from '@mui/icons-material/AutoFixHigh';
import SaveIcon from '@mui/icons-material/Save';
import { useParams } from 'react-router-dom';
import axios from 'axios';

// Types (mirroring backend)
interface Constraint {
  name: string;
  description: string;
  operator: string;
  scope: {
    sector?: string;
    region?: string;
    issuer?: string;
  };
  severity: string;
}

const ValuesProfileEditor: React.FC = () => {
  const { clientId } = useParams<{ clientId: string }>();
  const [constraints, setConstraints] = useState<Constraint[]>([]);
  const [aiPrompt, setAiPrompt] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Mock initial load (replace with API call)
  useEffect(() => {
    // fetchProfile(clientId);
  }, [clientId]);

  const handleGenerateConstraints = async () => {
    if (!aiPrompt) return;
    setLoading(true);
    setError(null);
    try {
      const response = await axios.post('http://localhost:8080/api/ai/generate-constraints', {
        prompt: aiPrompt
      });
      const newConstraints = response.data;
      setConstraints([...constraints, ...newConstraints]);
      setAiPrompt('');
    } catch (err) {
      setError('Failed to generate constraints. Please try again.');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteConstraint = (index: number) => {
    const newConstraints = [...constraints];
    newConstraints.splice(index, 1);
    setConstraints(newConstraints);
  };

  const handleSaveProfile = async () => {
    setLoading(true);
    try {
      // await axios.post('/api/values/constraints', { profile_id: ..., constraints });
      setSuccess('Profile saved successfully!');
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      setError('Failed to save profile.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Values Profile Editor
      </Typography>
      <Typography variant="subtitle1" color="textSecondary" gutterBottom>
        Client ID: {clientId}
      </Typography>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      {success && <Alert severity="success" sx={{ mb: 2 }}>{success}</Alert>}

      <Grid container spacing={3}>
        {/* Left Column: AI Assistant */}
        <Grid item xs={12} md={4}>
          <Paper elevation={3} sx={{ p: 2 }}>
            <Box display="flex" alignItems="center" mb={2}>
              <AutoFixHighIcon color="primary" sx={{ mr: 1 }} />
              <Typography variant="h6">AI Assistant</Typography>
            </Box>
            <Typography variant="body2" paragraph>
              Describe the client's values in natural language (e.g., "Avoid fossil fuels and tobacco companies").
            </Typography>
            <TextField
              fullWidth
              multiline
              rows={4}
              variant="outlined"
              placeholder="Enter client preferences..."
              value={aiPrompt}
              onChange={(e) => setAiPrompt(e.target.value)}
              disabled={loading}
            />
            <Button
              variant="contained"
              color="primary"
              fullWidth
              sx={{ mt: 2 }}
              onClick={handleGenerateConstraints}
              disabled={loading || !aiPrompt}
              startIcon={loading ? <CircularProgress size={20} /> : <AutoFixHighIcon />}
            >
              Generate Constraints
            </Button>
          </Paper>
        </Grid>

        {/* Right Column: Active Constraints */}
        <Grid item xs={12} md={8}>
          <Paper elevation={3} sx={{ p: 2 }}>
            <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
              <Typography variant="h6">Active Constraints</Typography>
              <Button
                variant="contained"
                color="secondary"
                startIcon={<SaveIcon />}
                onClick={handleSaveProfile}
                disabled={loading}
              >
                Save Profile
              </Button>
            </Box>
            <Divider />
            
            {constraints.length === 0 ? (
              <Box p={4} textAlign="center">
                <Typography color="textSecondary">No constraints defined yet.</Typography>
              </Box>
            ) : (
              <List>
                {constraints.map((c, index) => (
                  <React.Fragment key={index}>
                    <ListItem>
                      <ListItemText
                        primary={
                          <Box display="flex" alignItems="center">
                            <Typography variant="subtitle1" sx={{ mr: 1 }}>
                              {c.name}
                            </Typography>
                            <Chip
                              label={c.operator}
                              size="small"
                              color={c.operator === 'EXCLUDE' ? 'error' : 'primary'}
                              variant="outlined"
                            />
                          </Box>
                        }
                        secondary={c.description}
                      />
                      <ListItemSecondaryAction>
                        <IconButton edge="end" aria-label="delete" onClick={() => handleDeleteConstraint(index)}>
                          <DeleteIcon />
                        </IconButton>
                      </ListItemSecondaryAction>
                    </ListItem>
                    <Divider component="li" />
                  </React.Fragment>
                ))}
              </List>
            )}
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
};

export default ValuesProfileEditor;
