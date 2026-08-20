import React, { useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  FormControlLabel,
  Grid,
  IconButton,
  Paper,
  Stack,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
  useTheme,
} from '@mui/material';
import {
  AutoAwesome as AIIcon,
  Close as CloseIcon,
  Functions as CalcIcon,
  Rule as RuleIcon,
  Schema as SchemaIcon,
  Storage as TableIcon,
  CheckCircle as SuccessIcon,
  ArrowForward as ArrowForwardIcon,
} from '@mui/icons-material';
import { useNavigate } from 'react-router-dom';
import { fetchAPI } from '../../api';
import { useNotification } from '../../hooks/useNotification';

interface BOAIAssistantModalProps {
  open: boolean;
  onClose: () => void;
  onCreated?: (boId: string) => void;
}

export default function BOAIAssistantModal({
  open,
  onClose,
  onCreated,
}: BOAIAssistantModalProps) {
  const theme = useTheme();
  const navigate = useNavigate();
  const notification = useNotification();

  const [prompt, setPrompt] = useState('');
  const [tableName, setTableName] = useState('');
  const [category, setCategory] = useState('Wealth Management');
  const [includeRules, setIncludeRules] = useState(true);
  const [includeCalc, setIncludeCalc] = useState(true);

  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [blueprint, setBlueprint] = useState<any>(null);

  const handleSynthesize = async () => {
    if (!prompt.trim() && !tableName.trim()) {
      setError('Please provide a natural language prompt or table name.');
      return;
    }

    setLoading(true);
    setError(null);
    try {
      const resp = await fetchAPI<any>('/business-objects/ai/synthesize', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          prompt,
          tableName,
          category,
          includeRules,
          includeCalculatedFields: includeCalc,
        }),
      });
      setBlueprint(resp);
    } catch (err: any) {
      setError(err?.message || 'Failed to synthesize Business Object with AI.');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateAndOpen = async () => {
    if (!blueprint) return;
    setCreating(true);
    setError(null);
    try {
      const payload = {
        key: blueprint.suggestedKey,
        name: blueprint.suggestedName,
        displayName: blueprint.suggestedDisplayName || blueprint.suggestedName,
        description: blueprint.description,
        category: blueprint.category || category,
        driverTableName: blueprint.suggestedDriverTable || tableName || '',
        isActive: true,
        fields: (blueprint.suggestedFields || []).map((f: any) => ({
          key: f.key || f.name,
          name: f.name,
          displayName: f.displayName || f.name,
          type: f.type || 'text',
          role: f.role || 'DIMENSION',
          isRequired: f.isRequired ?? false,
          description: f.description,
        })),
      };

      const result = await fetchAPI<any>('/business-objects', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });

      const newId = result.id || blueprint.suggestedKey;
      notification.success(`Business Object '${payload.displayName}' created with AI blueprint!`);
      onCreated?.(newId);
      onClose();
      navigate(`/business-objects/${newId}`);
    } catch (err: any) {
      setError(err?.message || 'Failed to create Business Object.');
    } finally {
      setCreating(false);
    }
  };

  const samplePrompts = [
    'Portfolio Management entity with asset weights, benchmark performance, cash reserves, and Sharpe ratio',
    'Customer Onboarding entity with KYC status, accredited investor check, tax residency, and risk score',
    'Order Execution entity with trade volume, limit price, fill percentage, venue, and execution timestamps',
  ];

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle sx={{ m: 0, p: 2, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Stack direction="row" spacing={1.5} alignItems="center">
          <AIIcon color="primary" sx={{ fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 800 }}>
              AI Semantic Blueprint Generator
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Synthesize production-grade Business Objects with fields, formulas, and governance rules in seconds
            </Typography>
          </Box>
        </Stack>
        <IconButton onClick={onClose} size="small">
          <CloseIcon />
        </IconButton>
      </DialogTitle>

      <Divider />

      <DialogContent dividers sx={{ p: 3 }}>
        {error && (
          <Alert severity="error" onClose={() => setError(null)} sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {/* Input Form */}
        <Stack spacing={2.5} sx={{ mb: 3 }}>
          <TextField
            fullWidth
            multiline
            rows={3}
            label="Domain Description or Business Object Prompt"
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="e.g. Wealth Investment Account with balance, currency, risk score, margin ratio, and compliance flags..."
          />

          {/* Sample prompts */}
          <Box>
            <Typography variant="caption" sx={{ fontWeight: 700, color: 'text.secondary', display: 'block', mb: 0.75 }}>
              Sample Suggestions:
            </Typography>
            <Stack direction="row" spacing={1} flexWrap="wrap" gap={1}>
              {samplePrompts.map((sp, idx) => (
                <Chip
                  key={idx}
                  label={sp}
                  size="small"
                  onClick={() => setPrompt(sp)}
                  variant="outlined"
                  sx={{ cursor: 'pointer', maxWidth: '100%', textOverflow: 'ellipsis' }}
                />
              ))}
            </Stack>
          </Box>

          <Grid container spacing={2}>
            <Grid size={{ xs: 12, sm: 6 }}>
              <TextField
                fullWidth
                size="small"
                label="Physical Driver Table (Optional)"
                value={tableName}
                onChange={(e) => setTableName(e.target.value)}
                placeholder="e.g. public.portfolios"
              />
            </Grid>
            <Grid size={{ xs: 12, sm: 6 }}>
              <TextField
                fullWidth
                size="small"
                label="Domain Category"
                value={category}
                onChange={(e) => setCategory(e.target.value)}
              />
            </Grid>
          </Grid>

          <Stack direction="row" spacing={3}>
            <FormControlLabel
              control={<Switch checked={includeCalc} onChange={(e) => setIncludeCalc(e.target.checked)} size="small" />}
              label={<Typography variant="body2">Generate Calculated Metric Formulas</Typography>}
            />
            <FormControlLabel
              control={<Switch checked={includeRules} onChange={(e) => setIncludeRules(e.target.checked)} size="small" />}
              label={<Typography variant="body2">Generate Starlark Validation Rules</Typography>}
            />
          </Stack>

          <Button
            variant="contained"
            color="primary"
            startIcon={loading ? <CircularProgress size={18} color="inherit" /> : <AIIcon />}
            disabled={loading}
            onClick={handleSynthesize}
            size="large"
            sx={{ fontWeight: 700, textTransform: 'none' }}
          >
            {loading ? 'Synthesizing Architecture with AI...' : 'Generate Business Object Blueprint'}
          </Button>
        </Stack>

        {/* Blueprint Preview */}
        {blueprint && (
          <Box sx={{ mt: 3 }}>
            <Paper variant="outlined" sx={{ p: 2.5, bgcolor: 'action.hover', borderRadius: 2, mb: 3 }}>
              <Stack direction="row" justifyContent="space-between" alignItems="flex-start" sx={{ mb: 1.5 }}>
                <Box>
                  <Typography variant="h6" sx={{ fontWeight: 800 }}>
                    {blueprint.suggestedDisplayName}
                  </Typography>
                  <Typography variant="caption" sx={{ fontFamily: 'monospace', color: 'primary.main', fontWeight: 600 }}>
                    key: {blueprint.suggestedKey} | category: {blueprint.category}
                  </Typography>
                </Box>
                <Chip icon={<SuccessIcon />} label="AI Synthesized" color="success" size="small" sx={{ fontWeight: 700 }} />
              </Stack>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                {blueprint.description}
              </Typography>
              <Typography variant="caption" color="text.secondary" sx={{ fontStyle: 'italic', display: 'block' }}>
                Reasoning: {blueprint.reasoning}
              </Typography>
            </Paper>

            {/* Fields Table */}
            <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 1 }}>
              Synthesized Fields ({(blueprint.suggestedFields || []).length})
            </Typography>
            <TableContainer component={Paper} variant="outlined" sx={{ mb: 3 }}>
              <Table size="small">
                <TableHead>
                  <TableRow sx={{ bgcolor: 'action.hover' }}>
                    <TableCell sx={{ fontWeight: 700 }}>Name</TableCell>
                    <TableCell sx={{ fontWeight: 700 }}>Display Name</TableCell>
                    <TableCell sx={{ fontWeight: 700 }}>Data Type</TableCell>
                    <TableCell sx={{ fontWeight: 700 }}>Role</TableCell>
                    <TableCell sx={{ fontWeight: 700 }}>Description</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {(blueprint.suggestedFields || []).map((f: any, idx: number) => (
                    <TableRow key={idx}>
                      <TableCell sx={{ fontFamily: 'monospace', fontWeight: 600 }}>{f.name}</TableCell>
                      <TableCell>{f.displayName}</TableCell>
                      <TableCell>
                        <Chip label={f.type} size="small" variant="outlined" sx={{ height: 20, fontSize: '0.65rem' }} />
                      </TableCell>
                      <TableCell>
                        <Chip
                          label={f.role}
                          size="small"
                          color={f.role === 'MEASURE' ? 'primary' : f.role === 'IDENTIFIER' ? 'secondary' : 'default'}
                          sx={{ height: 20, fontSize: '0.65rem' }}
                        />
                      </TableCell>
                      <TableCell sx={{ color: 'text.secondary', fontSize: '0.8rem' }}>{f.description}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>

            {/* Calculated fields & rules preview */}
            <Grid container spacing={2}>
              {(blueprint.suggestedCalculatedFields || []).length > 0 && (
                <Grid size={{ xs: 12, md: 6 }}>
                  <Card variant="outlined">
                    <CardContent sx={{ p: 2 }}>
                      <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 1 }}>
                        <CalcIcon color="primary" fontSize="small" />
                        <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                          Calculated Metrics
                        </Typography>
                      </Stack>
                      {blueprint.suggestedCalculatedFields.map((cf: any, i: number) => (
                        <Box key={i} sx={{ mb: 1.5 }}>
                          <Typography variant="caption" sx={{ fontWeight: 700, display: 'block' }}>
                            {cf.displayName} ({cf.name})
                          </Typography>
                          <Typography variant="caption" sx={{ fontFamily: 'monospace', bgcolor: 'action.hover', p: 0.5, borderRadius: 0.5, display: 'block' }}>
                            {cf.formula}
                          </Typography>
                        </Box>
                      ))}
                    </CardContent>
                  </Card>
                </Grid>
              )}

              {(blueprint.suggestedRules || []).length > 0 && (
                <Grid size={{ xs: 12, md: 6 }}>
                  <Card variant="outlined">
                    <CardContent sx={{ p: 2 }}>
                      <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 1 }}>
                        <RuleIcon color="secondary" fontSize="small" />
                        <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                          Governance Validation Rules
                        </Typography>
                      </Stack>
                      {blueprint.suggestedRules.map((r: any, i: number) => (
                        <Box key={i} sx={{ mb: 1.5 }}>
                          <Typography variant="caption" sx={{ fontWeight: 700, display: 'block' }}>
                            {r.ruleName} ({r.severity})
                          </Typography>
                          <Typography variant="caption" color="text.secondary" display="block">
                            {r.description}
                          </Typography>
                        </Box>
                      ))}
                    </CardContent>
                  </Card>
                </Grid>
              )}
            </Grid>
          </Box>
        )}
      </DialogContent>

      <DialogActions sx={{ p: 2.5, justifyContent: 'space-between' }}>
        <Button onClick={onClose}>Close</Button>
        {blueprint && (
          <Button
            variant="contained"
            color="success"
            startIcon={creating ? <CircularProgress size={18} color="inherit" /> : <ArrowForwardIcon />}
            disabled={creating}
            onClick={handleCreateAndOpen}
            size="large"
            sx={{ fontWeight: 700, textTransform: 'none' }}
          >
            {creating ? 'Creating Business Object...' : 'Create Business Object & Open Studio'}
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
}
