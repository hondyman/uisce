import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Stepper,
  Step,
  StepLabel,
  StepContent,
  Chip,
  Alert,
  CircularProgress,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Grid,
  Paper,
  IconButton,
  Tooltip,
} from '@mui/material';
import {
  CheckCircle,
  Error,
  Schedule,
  ExpandMore,
  Download,
  PlayArrow,
  Done,
  HourglassEmpty,
  Cancel,
} from '@mui/icons-material';
import { EvidenceBundleAPI, EvidenceBundle, StageEvidence, Artifact } from '../../api/evidenceBundle';

const STAGE_LABELS: Record<string, string> = {
  diff: 'Diff Analysis',
  rebase: 'Merge & Rebase',
  test: 'Regression Testing',
  approval: 'Approval Workflow',
  deploy: 'Production Deployment',
  rollback: 'Rollback',
  audit: 'Audit & Snapshot',
};

const STATUS_COLORS: Record<string, 'success' | 'error' | 'warning' | 'info' | 'default'> = {
  success: 'success',
  failed: 'error',
  running: 'warning',
  pending: 'info',
  skipped: 'default',
};

const STATUS_ICONS: Record<string, React.ReactNode> = {
  success: <CheckCircle color="success" />,
  failed: <Error color="error" />,
  running: <HourglassEmpty color="warning" />,
  pending: <Schedule color="info" />,
  skipped: <Cancel color="disabled" />,
};

export const EvidenceBundleTimeline: React.FC = () => {
  const { bundleId } = useParams<{ bundleId: string }>();
  const [bundle, setBundle] = useState<EvidenceBundle | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeStep, setActiveStep] = useState(0);

  useEffect(() => {
    if (bundleId) {
      loadBundle();
    }
  }, [bundleId]);

  const loadBundle = async () => {
    try {
      setLoading(true);
      const data = await EvidenceBundleAPI.getBundle(bundleId!);
      setBundle(data);
      
      // Set active step to the first non-completed stage
      const activeIndex = data.stages.findIndex(
        (s) => s.status !== 'success' && s.status !== 'skipped'
      );
      setActiveStep(activeIndex !== -1 ? activeIndex : data.stages.length - 1);
    } catch (err: any) {
      setError(err.message || 'Failed to load evidence bundle');
    } finally {
      setLoading(false);
    }
  };

  const handleDownloadReport = async () => {
    try {
      await EvidenceBundleAPI.downloadComplianceReport(bundleId!);
    } catch (err: any) {
      alert('Failed to download compliance report');
    }
  };

  const formatDateTime = (dateString: string) => {
    return new Date(dateString).toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const renderArtifact = (artifact: Artifact) => {
    return (
      <Paper key={artifact.checksum} elevation={0} sx={{ p: 2, mb: 1, bgcolor: 'grey.50' }}>
        <Grid container spacing={2} alignItems="center">
          <Grid item xs={12} md={4}>
            <Typography variant="subtitle2" color="text.secondary">
              Type
            </Typography>
            <Chip label={artifact.type} size="small" />
          </Grid>
          <Grid item xs={12} md={4}>
            <Typography variant="subtitle2" color="text.secondary">
              Created
            </Typography>
            <Typography variant="body2">{formatDateTime(artifact.created_at)}</Typography>
          </Grid>
          <Grid item xs={12} md={4}>
            <Typography variant="subtitle2" color="text.secondary">
              Size
            </Typography>
            <Typography variant="body2">
              {artifact.size_bytes ? `${(artifact.size_bytes / 1024).toFixed(2)} KB` : 'N/A'}
            </Typography>
          </Grid>
          {artifact.metadata && (
            <Grid item xs={12}>
              <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                Metadata
              </Typography>
              <pre style={{ fontSize: '12px', whiteSpace: 'pre-wrap' }}>
                {JSON.stringify(artifact.metadata, null, 2)}
              </pre>
            </Grid>
          )}
        </Grid>
      </Paper>
    );
  };

  const renderStageContent = (stage: StageEvidence) => {
    return (
      <Box sx={{ pt: 1 }}>
        <Grid container spacing={2} sx={{ mb: 2 }}>
          <Grid item xs={6}>
            <Typography variant="subtitle2" color="text.secondary">
              Started
            </Typography>
            <Typography variant="body2">{formatDateTime(stage.started_at)}</Typography>
          </Grid>
          <Grid item xs={6}>
            <Typography variant="subtitle2" color="text.secondary">
              Completed
            </Typography>
            <Typography variant="body2">
              {stage.completed_at ? formatDateTime(stage.completed_at) : 'In progress'}
            </Typography>
          </Grid>
          {stage.actor_id && (
            <Grid item xs={12}>
              <Typography variant="subtitle2" color="text.secondary">
                Actor
              </Typography>
              <Typography variant="body2">{stage.actor_id}</Typography>
            </Grid>
          )}
        </Grid>

        {stage.artifacts.length > 0 && (
          <Box>
            <Typography variant="subtitle2" gutterBottom>
              Evidence Artifacts ({stage.artifacts.length})
            </Typography>
            {stage.artifacts.map(renderArtifact)}
          </Box>
        )}
      </Box>
    );
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  if (error || !bundle) {
    return (
      <Alert severity="error" sx={{ m: 3 }}>
        {error || 'Evidence bundle not found'}
      </Alert>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Box>
          <Typography variant="h4" gutterBottom>
            Evidence Bundle Timeline
          </Typography>
          <Typography variant="body2" color="text.secondary">
            {bundle.old_version} → {bundle.new_version}
          </Typography>
        </Box>
        <Box>
          <Chip
            label={bundle.status}
            color={STATUS_COLORS[bundle.status] || 'default'}
            sx={{ mr: 2 }}
          />
          <Tooltip title="Download Compliance Report">
            <IconButton onClick={handleDownloadReport} color="primary">
              <Download />
            </IconButton>
          </Tooltip>
        </Box>
      </Box>

      <Card>
        <CardContent>
          <Stepper activeStep={activeStep} orientation="vertical">
            {bundle.stages.map((stage, index) => (
              <Step key={stage.id} expanded>
                <StepLabel
                  icon={STATUS_ICONS[stage.status]}
                  optional={
                    <Chip
                      label={stage.status}
                      size="small"
                      color={STATUS_COLORS[stage.status]}
                    />
                  }
                >
                  <Typography variant="h6">{STAGE_LABELS[stage.stage_name]}</Typography>
                </StepLabel>
                <StepContent>
                  {renderStageContent(stage)}
                </StepContent>
              </Step>
            ))}
          </Stepper>
        </CardContent>
      </Card>

      <Card sx={{ mt: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Bundle Information
          </Typography>
          <Grid container spacing={2}>
            <Grid item xs={12} md={6}>
              <Typography variant="subtitle2" color="text.secondary">
                Bundle ID
              </Typography>
              <Typography variant="body2" fontFamily="monospace">
                {bundle.id}
              </Typography>
            </Grid>
            <Grid item xs={12} md={6}>
              <Typography variant="subtitle2" color="text.secondary">
                Upgrade Request ID
              </Typography>
              <Typography variant="body2" fontFamily="monospace">
                {bundle.upgrade_request_id}
              </Typography>
            </Grid>
            <Grid item xs={12} md={6}>
              <Typography variant="subtitle2" color="text.secondary">
                Created
              </Typography>
              <Typography variant="body2">{formatDateTime(bundle.created_at)}</Typography>
            </Grid>
            {bundle.completed_at && (
              <Grid item xs={12} md={6}>
                <Typography variant="subtitle2" color="text.secondary">
                  Completed
                </Typography>
                <Typography variant="body2">{formatDateTime(bundle.completed_at)}</Typography>
              </Grid>
            )}
          </Grid>
        </CardContent>
      </Card>
    </Box>
  );
};
