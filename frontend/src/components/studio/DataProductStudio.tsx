import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Button,
  Chip,
  Stack,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  RadioGroup,
  FormControlLabel,
  Radio,
  TextField,
} from '@mui/material';
import HubIcon from '@mui/icons-material/Hub';
import ThumbUpAltIcon from '@mui/icons-material/ThumbUpAlt';
import ThumbDownAltIcon from '@mui/icons-material/ThumbDownAlt';

export const DataProductStudio: React.FC<{ tenantId?: string; boKey?: string }> = ({
  tenantId = 'default',
  boKey = 'portfolio_performance',
}) => {
  const [feedbackState, setFeedbackState] = useState<'NONE' | 'POSITIVE' | 'NEGATIVE'>('NONE');
  const [errorModalOpen, setErrorModalOpen] = useState(false);
  const [errorCategory, setErrorCategory] = useState('WRONG_TABLE');
  const [userNotes, setUserNotes] = useState('');

  const handleThumbUp = async () => {
    setFeedbackState('POSITIVE');
  };

  const handleThumbDown = () => {
    setFeedbackState('NEGATIVE');
    setErrorModalOpen(true);
  };

  const submitNegativeFeedback = async () => {
    setErrorModalOpen(false);
  };

  return (
    <Box sx={{ width: '100%', bgcolor: '#050D1A', color: '#fff', p: 3, fontFamily: 'sans-serif' }}>
      
      {/* Contract Header & SLA Card */}
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', pb: 2, borderBottom: '1px solid #1E293B', mb: 3 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <HubIcon sx={{ color: '#00D4FF', fontSize: 26 }} />
          <Box>
            <Typography variant="h6" fontWeight="700">
              OpenDataContract Gateway & SLA Monitor
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              Target BO: <span style={{ color: '#00D4FF', fontFamily: 'monospace' }}>{boKey}</span> &bull; Version: 1.2.0 &bull; SLA: 15m Freshness / 250ms p95
            </Typography>
          </Box>
        </Box>

        {/* Ambient Two-Stage AI Feedback Trigger */}
        <Stack direction="row" spacing={1} alignItems="center">
          <Typography variant="caption" sx={{ color: '#64748B', mr: 1 }}>AI Response Accurate?</Typography>
          <Tooltip title="Helpful & Accurate">
            <IconButton
              size="small"
              onClick={handleThumbUp}
              sx={{ color: feedbackState === 'POSITIVE' ? '#10B981' : '#64748B', bgcolor: '#071526' }}
            >
              <ThumbUpAltIcon fontSize="small" />
            </IconButton>
          </Tooltip>
          <Tooltip title="Incorrect or Broken">
            <IconButton
              size="small"
              onClick={handleThumbDown}
              sx={{ color: feedbackState === 'NEGATIVE' ? '#EF4444' : '#64748B', bgcolor: '#071526' }}
            >
              <ThumbDownAltIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        </Stack>
      </Box>

      {/* Contract YAML / Endpoint Surface */}
      <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: '1fr 360px' }, gap: 3 }}>
        <Paper sx={{ p: 2.5, bgcolor: '#071526', border: '1px solid #1E293B', borderRadius: 2 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
            <Typography variant="caption" fontWeight="700" sx={{ color: '#00D4FF', textTransform: 'uppercase' }}>
              Declarative Contract (YAML)
            </Typography>
            <Chip size="small" label="SLA ACTIVE: 99.95%" sx={{ bgcolor: 'rgba(16, 185, 129, 0.15)', color: '#10B981', fontSize: '10px', fontWeight: 700 }} />
          </Box>

          <pre style={{ fontFamily: 'monospace', fontSize: '11px', color: '#CBD5E1', overflowX: 'auto', padding: '12px', backgroundColor: '#030914', borderRadius: '4px', border: '1px solid #1E293B' }}>
{`apiVersion: datacontract.com/v2alpha
kind: DataContract
metadata:
  contractId: "dc_${boKey}_v1"
  businessObjectKey: "${boKey}"
  owner: "wealth_analytics@fund.com"
serviceLevelAgreement:
  freshness: "15m"
  maxLatencyMs: 250
  availabilityPct: 99.95
schema:
  fields:
    - name: "account_bk"
      type: "string"
      primaryKey: true
    - name: "net_fund_yield"
      type: "percentage"
      precision: 4`}
          </pre>
        </Paper>

        {/* Live Multi-Protocol Ingress Endpoints */}
        <Paper sx={{ p: 2.5, bgcolor: '#071526', border: '1px solid #1E293B', borderRadius: 2 }}>
          <Typography variant="caption" fontWeight="700" sx={{ color: '#94A3B8', textTransform: 'uppercase', display: 'block', mb: 2 }}>
            Generated Multi-Protocol Endpoints
          </Typography>

          <Stack spacing={2}>
            <Box sx={{ p: 1.5, bgcolor: '#050D1A', borderRadius: 1, border: '1px solid #1E293B' }}>
              <Typography variant="caption" sx={{ color: '#C084FC', fontWeight: 'bold', display: 'block' }}>REST JSON API</Typography>
              <Typography variant="caption" sx={{ fontFamily: 'monospace', color: '#CBD5E1' }}>GET /api/v1/products/{boKey}</Typography>
            </Box>
            <Box sx={{ p: 1.5, bgcolor: '#050D1A', borderRadius: 1, border: '1px solid #1E293B' }}>
              <Typography variant="caption" sx={{ color: '#22D3EE', fontWeight: 'bold', display: 'block' }}>GRAPHQL GATEWAY</Typography>
              <Typography variant="caption" sx={{ fontFamily: 'monospace', color: '#CBD5E1' }}>POST /api/v1/graphql</Typography>
            </Box>
            <Box sx={{ p: 1.5, bgcolor: '#050D1A', borderRadius: 1, border: '1px solid #1E293B' }}>
              <Typography variant="caption" sx={{ color: '#34D399', fontWeight: 'bold', display: 'block' }}>ARROW FLIGHT STREAM</Typography>
              <Typography variant="caption" sx={{ fontFamily: 'monospace', color: '#CBD5E1' }}>flight.uisce.io:443</Typography>
            </Box>
          </Stack>
        </Paper>
      </Box>

      {/* Stage 2: Negative Feedback Categorization Popover Modal */}
      <Dialog
        open={errorModalOpen}
        onClose={() => setErrorModalOpen(false)}
        PaperProps={{ sx: { bgcolor: '#071526', color: '#fff', border: '1px solid #1E293B', minWidth: 420 } }}
      >
        <DialogTitle sx={{ fontWeight: 700, fontSize: '14px' }}>
          Help Us Improve: Classify AI Issue
        </DialogTitle>
        <DialogContent>
          <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block', mb: 2 }}>
            Your feedback automatically adjusts pgvector ranking weights and alerts domain stewards.
          </Typography>

          <RadioGroup value={errorCategory} onChange={(e) => setErrorCategory(e.target.value)}>
            <FormControlLabel value="WRONG_TABLE" control={<Radio size="small" sx={{ color: '#00D4FF' }} />} label={<Typography variant="body2" sx={{ fontSize: '12px' }}>Selected wrong physical table</Typography>} />
            <FormControlLabel value="INCORRECT_FORMULA" control={<Radio size="small" sx={{ color: '#00D4FF' }} />} label={<Typography variant="body2" sx={{ fontSize: '12px' }}>Incorrect formula calculation</Typography>} />
            <FormControlLabel value="BAD_JOIN" control={<Radio size="small" sx={{ color: '#00D4FF' }} />} label={<Typography variant="body2" sx={{ fontSize: '12px' }}>Improper relationship join / cardinality</Typography>} />
            <FormControlLabel value="HALLUCINATION" control={<Radio size="small" sx={{ color: '#00D4FF' }} />} label={<Typography variant="body2" sx={{ fontSize: '12px' }}>Hallucinated non-existent terms</Typography>} />
          </RadioGroup>

          <TextField
            fullWidth
            size="small"
            multiline
            rows={2}
            placeholder="Additional optional notes for data stewards..."
            value={userNotes}
            onChange={(e) => setUserNotes(e.target.value)}
            sx={{ mt: 2, bgcolor: '#050D1A', input: { color: '#fff', fontSize: '11px' } }}
          />
        </DialogContent>
        <DialogActions sx={{ p: 2, borderTop: '1px solid #1E293B' }}>
          <Button onClick={() => setErrorModalOpen(false)} sx={{ color: '#64748B', fontSize: '11px' }}>Cancel</Button>
          <Button variant="contained" onClick={submitNegativeFeedback} sx={{ bgcolor: '#EF4444', color: '#fff', fontSize: '11px', fontWeight: 700 }}>
            Submit Telemetry
          </Button>
        </DialogActions>
      </Dialog>

    </Box>
  );
};

export default DataProductStudio;
