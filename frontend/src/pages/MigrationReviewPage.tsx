import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Box,
  Paper,
  Typography,
  Button,
  TextField,
  Grid,
  Chip,
  CircularProgress,
  Alert,
  Divider,
  IconButton,
  Tabs,
  Tab
} from '@mui/material';
import {
  Check as CheckIcon,
  Close as CloseIcon,
  Refresh as RefreshIcon,
  Code as CodeIcon,
  Psychology as PsychologyIcon,
  AccountTree as AccountTreeIcon
} from '@mui/icons-material';
import { useTenant } from '../contexts/TenantContext';
import { Light as SyntaxHighlighter } from 'react-syntax-highlighter';
import { atomOneDark } from 'react-syntax-highlighter/dist/esm/styles/hljs';

interface MigrationJob {
  id: string;
  name: string;
  status: string;
  sourceCode: string;
  sourceLanguage: string;
  extractedIntent?: {
    summary?: string;
    preconditions?: Array<{ description: string }>;
    actions?: Array<{ description: string }>;
  };
  generatedDag?: object;
  generatedRego?: string;
  reviewNotes?: string;
  createdAt: string;
}

const MigrationReviewPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { tenant } = useTenant();
  
  const [migration, setMigration] = useState<MigrationJob | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [reviewNotes, setReviewNotes] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [rightPanelTab, setRightPanelTab] = useState(0);

  const fetchMigration = async () => {
    if (!id) return;
    
    setLoading(true);
    try {
      const res = await fetch(`/api/migrations/${id}`, {
        headers: {
          ...(tenant?.id && { 'X-Tenant-ID': tenant.id }),
        },
      });
      
      if (!res.ok) throw new Error('Failed to fetch migration');
      
      const data = await res.json();
      setMigration(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMigration();
  }, [id, tenant?.id]);

  const handleApprove = async () => {
    if (!id) return;
    setSubmitting(true);
    
    try {
      const res = await fetch(`/api/migrations/${id}/approve`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(tenant?.id && { 'X-Tenant-ID': tenant.id }),
        },
        body: JSON.stringify({ notes: reviewNotes }),
      });
      
      if (!res.ok) throw new Error('Failed to approve');
      
      navigate('/migrations');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Approval failed');
    } finally {
      setSubmitting(false);
    }
  };

  const handleReject = async () => {
    if (!id) return;
    setSubmitting(true);
    
    try {
      const res = await fetch(`/api/migrations/${id}/reject`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(tenant?.id && { 'X-Tenant-ID': tenant.id }),
        },
        body: JSON.stringify({ notes: reviewNotes }),
      });
      
      if (!res.ok) throw new Error('Failed to reject');
      
      navigate('/migrations');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Rejection failed');
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error || !migration) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">{error || 'Migration not found'}</Alert>
      </Box>
    );
  }

  const statusColor = {
    PENDING: 'default',
    ANALYZING: 'info',
    EXTRACTED: 'info',
    GENERATING: 'info',
    REVIEW: 'warning',
    APPROVED: 'success',
    REJECTED: 'error',
  }[migration.status] || 'default';

  return (
    <Box sx={{ p: 3, height: 'calc(100vh - 64px)', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Box>
          <Typography variant="h5">{migration.name}</Typography>
          <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', mt: 0.5 }}>
            <Chip label={migration.status} color={statusColor as any} size="small" />
            <Chip label={migration.sourceLanguage.toUpperCase()} variant="outlined" size="small" />
          </Box>
        </Box>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <IconButton onClick={fetchMigration}>
            <RefreshIcon />
          </IconButton>
        </Box>
      </Box>

      {/* Three-Panel Layout */}
      <Grid container spacing={2} sx={{ flex: 1, minHeight: 0 }}>
        {/* Left Panel: Source Code */}
        <Grid item xs={4}>
          <Paper sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
            <Box sx={{ p: 1.5, borderBottom: 1, borderColor: 'divider', display: 'flex', alignItems: 'center', gap: 1 }}>
              <CodeIcon fontSize="small" />
              <Typography variant="subtitle2">Legacy Code</Typography>
            </Box>
            <Box sx={{ flex: 1, overflow: 'auto' }}>
              <SyntaxHighlighter
                language={migration.sourceLanguage}
                style={atomOneDark}
                customStyle={{ margin: 0, height: '100%', fontSize: '12px' }}
              >
                {migration.sourceCode}
              </SyntaxHighlighter>
            </Box>
          </Paper>
        </Grid>

        {/* Center Panel: Extracted Intent */}
        <Grid item xs={4}>
          <Paper sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
            <Box sx={{ p: 1.5, borderBottom: 1, borderColor: 'divider', display: 'flex', alignItems: 'center', gap: 1 }}>
              <PsychologyIcon fontSize="small" />
              <Typography variant="subtitle2">Business Intent (AI Extracted)</Typography>
            </Box>
            <Box sx={{ flex: 1, overflow: 'auto', p: 2 }}>
              {migration.extractedIntent ? (
                <>
                  <Typography variant="h6" gutterBottom>
                    {migration.extractedIntent.summary}
                  </Typography>
                  
                  {migration.extractedIntent.preconditions?.length > 0 && (
                    <>
                      <Typography variant="subtitle2" color="primary" sx={{ mt: 2 }}>
                        Preconditions
                      </Typography>
                      <ul>
                        {migration.extractedIntent.preconditions.map((p, i) => (
                          <li key={i}><Typography variant="body2">{p.description}</Typography></li>
                        ))}
                      </ul>
                    </>
                  )}
                  
                  {migration.extractedIntent.actions?.length > 0 && (
                    <>
                      <Typography variant="subtitle2" color="secondary" sx={{ mt: 2 }}>
                        Actions
                      </Typography>
                      <ul>
                        {migration.extractedIntent.actions.map((a, i) => (
                          <li key={i}><Typography variant="body2">{a.description}</Typography></li>
                        ))}
                      </ul>
                    </>
                  )}
                </>
              ) : (
                <Alert severity="info">Intent extraction pending...</Alert>
              )}
            </Box>
          </Paper>
        </Grid>

        {/* Right Panel: Generated Config */}
        <Grid item xs={4}>
          <Paper sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
            <Box sx={{ p: 1.5, borderBottom: 1, borderColor: 'divider', display: 'flex', alignItems: 'center', gap: 1 }}>
              <AccountTreeIcon fontSize="small" />
              <Typography variant="subtitle2">Titan Configuration</Typography>
            </Box>
            
            <Tabs value={rightPanelTab} onChange={(_, v) => setRightPanelTab(v)} sx={{ borderBottom: 1, borderColor: 'divider' }}>
              <Tab label="DAG JSON" />
              <Tab label="OPA Rego" disabled={!migration.generatedRego} />
            </Tabs>
            
            <Box sx={{ flex: 1, overflow: 'auto' }}>
              {rightPanelTab === 0 && (
                <SyntaxHighlighter
                  language="json"
                  style={atomOneDark}
                  customStyle={{ margin: 0, height: '100%', fontSize: '12px' }}
                >
                  {migration.generatedDag ? JSON.stringify(migration.generatedDag, null, 2) : '// No DAG generated yet'}
                </SyntaxHighlighter>
              )}
              {rightPanelTab === 1 && (
                <SyntaxHighlighter
                  language="rego"
                  style={atomOneDark}
                  customStyle={{ margin: 0, height: '100%', fontSize: '12px' }}
                >
                  {migration.generatedRego || '# No Rego policy generated'}
                </SyntaxHighlighter>
              )}
            </Box>
          </Paper>
        </Grid>
      </Grid>

      {/* Review Actions */}
      {migration.status === 'REVIEW' && (
        <Paper sx={{ mt: 2, p: 2 }}>
          <Typography variant="subtitle2" gutterBottom>Review Notes</Typography>
          <TextField
            fullWidth
            multiline
            rows={2}
            placeholder="Add notes about this migration..."
            value={reviewNotes}
            onChange={(e) => setReviewNotes(e.target.value)}
            sx={{ mb: 2 }}
          />
          <Box sx={{ display: 'flex', gap: 2 }}>
            <Button
              variant="contained"
              color="success"
              startIcon={submitting ? <CircularProgress size={16} /> : <CheckIcon />}
              onClick={handleApprove}
              disabled={submitting}
            >
              Approve & Commit
            </Button>
            <Button
              variant="outlined"
              color="error"
              startIcon={<CloseIcon />}
              onClick={handleReject}
              disabled={submitting}
            >
              Reject
            </Button>
          </Box>
        </Paper>
      )}
    </Box>
  );
};

export default MigrationReviewPage;
