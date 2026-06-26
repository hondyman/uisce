import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Paper,
  Typography,
  TextField,
  Button,
  Grid,
  CircularProgress,
  Alert,
  Divider,
  List,
  ListItem,
  ListItemText,
  ListItemSecondaryAction,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Chip
} from '@mui/material';
import {
  Psychology as AIIcon,
  Save as SaveIcon,
  Add as AddIcon,
  Delete as DeleteIcon,
  Edit as EditIcon
} from '@mui/icons-material';
import { useTenant } from '../contexts/TenantContext';
import Editor from '@monaco-editor/react';

interface Policy {
  id: string;
  name: string;
  description: string;
  regoCode: string;
  createdAt: string;
}

const PolicyEditorPage: React.FC = () => {
  const { tenant } = useTenant();
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [selectedPolicy, setSelectedPolicy] = useState<Policy | null>(null);

  // AI Generation state
  const [ruleText, setRuleText] = useState('');
  const [generatedRego, setGeneratedRego] = useState('');
  const [explanation, setExplanation] = useState('');
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Save dialog
  const [saveDialogOpen, setSaveDialogOpen] = useState(false);
  const [policyName, setPolicyName] = useState('');
  const [policyDescription, setPolicyDescription] = useState('');
  const [saving, setSaving] = useState(false);

  // Fetch policies on mount
  useEffect(() => {
    fetchPolicies();
  }, [tenant?.id]);

  const fetchPolicies = async () => {
    try {
      const response = await fetch('/api/v1/policies', {
        headers: {
          ...(tenant?.id && { 'X-Tenant-ID': tenant.id }),
        },
      });
      if (response.ok) {
        const data = await response.json();
        setPolicies(data || []);
      }
    } catch (err) {
      console.error('Failed to fetch policies:', err);
    }
  };

  const handleGenerate = useCallback(async () => {
    if (!ruleText.trim()) return;

    setGenerating(true);
    setError(null);

    try {
      const response = await fetch('/api/v1/copilot/generate-rego', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(tenant?.id && { 'X-Tenant-ID': tenant.id }),
        },
        body: JSON.stringify({
          text: ruleText,
          policyName: policyName || 'custom_policy',
        }),
      });

      if (!response.ok) {
        throw new Error(await response.text());
      }

      const data = await response.json();
      setGeneratedRego(data.regoCode);
      setExplanation(data.explanation);
      if (data.policyName && !policyName) {
        setPolicyName(data.policyName);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Generation failed');
    } finally {
      setGenerating(false);
    }
  }, [ruleText, policyName, tenant?.id]);

  const handleSave = async () => {
    if (!generatedRego || !policyName) return;

    setSaving(true);
    try {
      const response = await fetch('/api/v1/policies', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(tenant?.id && { 'X-Tenant-ID': tenant.id }),
        },
        body: JSON.stringify({
          name: policyName,
          description: policyDescription || explanation,
          regoCode: generatedRego,
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to save policy');
      }

      setSaveDialogOpen(false);
      setRuleText('');
      setGeneratedRego('');
      setPolicyName('');
      setPolicyDescription('');
      fetchPolicies();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Save failed');
    } finally {
      setSaving(false);
    }
  };

  const handleDeletePolicy = async (id: string) => {
    try {
      await fetch(`/api/v1/policies/${id}`, {
        method: 'DELETE',
        headers: {
          ...(tenant?.id && { 'X-Tenant-ID': tenant.id }),
        },
      });
      fetchPolicies();
    } catch (err) {
      console.error('Failed to delete policy:', err);
    }
  };

  const exampleRules = [
    "Block all trades over $1 million",
    "Require manager approval for orders from new customers",
    "Deny access if user location is outside approved regions",
    "Flag transactions with amount above customer's credit limit"
  ];

  return (
    <Box sx={{ p: 3, height: 'calc(100vh - 64px)', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 3 }}>
        <AIIcon sx={{ fontSize: 32, color: 'primary.main' }} />
        <Box>
          <Typography variant="h5">AI Policy Editor</Typography>
          <Typography variant="body2" color="text.secondary">
            Describe business rules in plain English and generate OPA Rego policies
          </Typography>
        </Box>
      </Box>

      <Grid container spacing={3} sx={{ flex: 1, minHeight: 0 }}>
        {/* Left: Rule Input & Generation */}
        <Grid item xs={12} md={4}>
          <Paper sx={{ p: 2, height: '100%', display: 'flex', flexDirection: 'column' }}>
            <Typography variant="subtitle1" fontWeight="bold" gutterBottom>
              Describe Your Rule
            </Typography>

            <TextField
              fullWidth
              multiline
              rows={4}
              placeholder="Example: Block all trades over $1 million"
              value={ruleText}
              onChange={(e) => setRuleText(e.target.value)}
              sx={{ mb: 2 }}
            />

            <Button
              fullWidth
              variant="contained"
              startIcon={generating ? <CircularProgress size={16} color="inherit" /> : <AIIcon />}
              onClick={handleGenerate}
              disabled={generating || !ruleText.trim()}
              sx={{ mb: 2 }}
            >
              {generating ? 'Generating...' : 'Generate Policy'}
            </Button>

            <Divider sx={{ my: 2 }} />

            <Typography variant="caption" color="text.secondary" gutterBottom>
              Example rules:
            </Typography>
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
              {exampleRules.map((rule, i) => (
                <Chip
                  key={i}
                  label={rule.length > 25 ? rule.substring(0, 25) + '...' : rule}
                  size="small"
                  onClick={() => setRuleText(rule)}
                  sx={{ cursor: 'pointer' }}
                />
              ))}
            </Box>

            {error && (
              <Alert severity="error" sx={{ mt: 2 }}>
                {error}
              </Alert>
            )}

            <Divider sx={{ my: 2 }} />

            <Typography variant="subtitle2" gutterBottom>
              Saved Policies
            </Typography>
            <List dense sx={{ flex: 1, overflow: 'auto' }}>
              {policies.map((policy) => (
                <ListItem
                  key={policy.id}
                  button
                  selected={selectedPolicy?.id === policy.id}
                  onClick={() => {
                    setSelectedPolicy(policy);
                    setGeneratedRego(policy.regoCode);
                  }}
                >
                  <ListItemText
                    primary={policy.name}
                    secondary={policy.description?.substring(0, 50)}
                  />
                  <ListItemSecondaryAction>
                    <IconButton size="small" onClick={() => handleDeletePolicy(policy.id)}>
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  </ListItemSecondaryAction>
                </ListItem>
              ))}
              {policies.length === 0 && (
                <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
                  No policies saved yet
                </Typography>
              )}
            </List>
          </Paper>
        </Grid>

        {/* Right: Generated Rego Code */}
        <Grid item xs={12} md={8}>
          <Paper sx={{ p: 2, height: '100%', display: 'flex', flexDirection: 'column' }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
              <Typography variant="subtitle1" fontWeight="bold">
                Generated Rego Policy
              </Typography>
              <Button
                variant="contained"
                startIcon={<SaveIcon />}
                disabled={!generatedRego}
                onClick={() => setSaveDialogOpen(true)}
              >
                Save Policy
              </Button>
            </Box>

            {explanation && (
              <Alert severity="info" sx={{ mb: 2 }}>
                {explanation}
              </Alert>
            )}

            <Box sx={{ flex: 1, minHeight: 0, border: 1, borderColor: 'divider', borderRadius: 1 }}>
              <Editor
                height="100%"
                language="rego"
                theme="vs-dark"
                value={generatedRego || '# Generated Rego policy will appear here\n'}
                onChange={(value) => setGeneratedRego(value || '')}
                options={{
                  minimap: { enabled: false },
                  fontSize: 14,
                  lineNumbers: 'on',
                  scrollBeyondLastLine: false,
                }}
              />
            </Box>
          </Paper>
        </Grid>
      </Grid>

      {/* Save Dialog */}
      <Dialog open={saveDialogOpen} onClose={() => setSaveDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Save Policy</DialogTitle>
        <DialogContent>
          <TextField
            fullWidth
            label="Policy Name"
            value={policyName}
            onChange={(e) => setPolicyName(e.target.value)}
            sx={{ mt: 2, mb: 2 }}
          />
          <TextField
            fullWidth
            multiline
            rows={2}
            label="Description"
            value={policyDescription}
            onChange={(e) => setPolicyDescription(e.target.value)}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setSaveDialogOpen(false)}>Cancel</Button>
          <Button
            variant="contained"
            onClick={handleSave}
            disabled={saving || !policyName}
            startIcon={saving ? <CircularProgress size={16} /> : <SaveIcon />}
          >
            Save
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default PolicyEditorPage;
