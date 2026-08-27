import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Button,
  Chip,
  Grid,
  Alert,
  Card,
  CardContent
} from '@mui/material';
import {
  AutoAwesome as AiIcon,
  CheckCircle as ValidIcon,
  WarningAmber as BreakIcon,
  Shield as GovernanceIcon,
  Speed as LatencyIcon,
  ThumbUp as ApproveIcon,
  Edit as OverrideIcon
} from '@mui/icons-material';

interface OpenBreakItem {
  exceptionId: string;
  domainKey: string;
  masterEntitySID: string;
  entityName: string;
  fieldName: string;
  competingFeeds: Array<{ vendor: string; value: string; timeAge: string; variancePct?: number }>;
  aiWinner: string;
  aiConfidence: number;
  aiDiagnostic: string;
  status: 'PENDING_APPROVAL' | 'APPLIED' | 'OVERRIDDEN';
}

export const AIDataStewardWorkstation: React.FC<{ tenantId?: string }> = ({
  tenantId: _tenantId = '99e99e99-99e9-49e9-89e9-99e99e99e999'
}) => {
  const [breaks, setBreaks] = useState<OpenBreakItem[]>([
    {
      exceptionId: 'brk-0921-01',
      domainKey: 'PRICING',
      masterEntitySID: 'SEC_US912810TL44',
      entityName: 'US Treasury N/B 4.25% 2034',
      fieldName: 'market_price',
      competingFeeds: [
        { vendor: 'BLOOMBERG', value: '$98.42', timeAge: '12s ago', variancePct: 0.0 },
        { vendor: 'REFINITIV', value: '$92.70', timeAge: '4m ago', variancePct: 6.15 },
        { vendor: 'IDC', value: '$98.40', timeAge: '1m ago', variancePct: 0.02 }
      ],
      aiWinner: 'BLOOMBERG',
      aiConfidence: 0.9650,
      aiDiagnostic:
        'Refinitiv quote ($92.70) flagged for tolerance breach (+6.15% deviation) due to a 4-minute staleness delay. Bloomberg ($98.42) selected based on consensus with IDC ($98.40) and 99.8% historical fixed-income accuracy half-life weighting.',
      status: 'PENDING_APPROVAL'
    }
  ]);

  const [actionNotice, setActionNotice] = useState<string | null>(null);

  const handleApproveProposal = (exceptionId: string) => {
    setBreaks((prev) =>
      prev.map((b) => (b.exceptionId === exceptionId ? { ...b, status: 'APPLIED' } : b))
    );
    setActionNotice(
      'AI Recommendation Approved: Golden Record master updated. Downstream consumers (IBOR/ABOR) notified via event mesh with SEC Rule 17a-4 Merkle seal.'
    );
  };

  return (
    <Paper
      elevation={0}
      sx={{
        p: 3,
        bgcolor: '#071526',
        color: '#F8FAFC',
        border: '1px solid #1E293B',
        borderRadius: 2
      }}
    >
      {/* Header Bar */}
      <Box display="flex" justifyContent="space-between" alignItems="center" pb={2} mb={3} borderBottom="1px solid #1E293B">
        <Stack direction="row" spacing={1.5} alignItems="center">
          <AiIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              MDM Exception & Data Steward Studio (AI Triage Workstation)
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Side-by-side vendor variance analysis, neural survivorship scoring & 1-click Maker-Checker resolution
            </Typography>
          </Box>
        </Stack>

        <Stack direction="row" spacing={2} alignItems="center">
          <Chip
            icon={<LatencyIcon sx={{ fontSize: 14, color: '#00D4FF !important' }} />}
            label="Inference SLA: < 15ms"
            size="small"
            sx={{ bgcolor: '#0B1E36', color: '#00D4FF', fontWeight: 700, fontSize: 11, fontFamily: 'monospace' }}
          />
          <Chip
            icon={<GovernanceIcon sx={{ fontSize: 14, color: '#10B981 !important' }} />}
            label="SEC 17a-4 Audit Sealed"
            size="small"
            sx={{ bgcolor: '#064E3B', color: '#34D399', fontWeight: 700, fontSize: 11 }}
          />
        </Stack>
      </Box>

      {actionNotice && (
        <Alert severity="success" sx={{ mb: 3, bgcolor: '#064E3B', color: '#F8FAFC', border: '1px solid #10B981' }}>
          {actionNotice}
        </Alert>
      )}

      {/* KPI Cards */}
      <Grid container spacing={2} mb={3}>
        <Grid   size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
              AI Auto-Resolution Rate
            </Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#34D399', fontFamily: 'monospace' }}>
              94.8%
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              Zero Manual Intervention Required
            </Typography>
          </Paper>
        </Grid>

        <Grid   size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
              Active Exceptions Requiring Sign-off
            </Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#EF4444', fontFamily: 'monospace' }}>
              {breaks.filter((b) => b.status === 'PENDING_APPROVAL').length} Breaches
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              &gt; 5% Inter-Vendor Pricing Deviation
            </Typography>
          </Paper>
        </Grid>

        <Grid   size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
              GraphRAG XREF Disambiguation
            </Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#38BDF8' }}>
              100% Unambiguous
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              ISIN ↔ CUSIP ↔ SEDOL ↔ FIGI
            </Typography>
          </Paper>
        </Grid>
      </Grid>

      {/* Exception Breakdown Cards */}
      <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700, textTransform: 'uppercase', mb: 1.5, display: 'block' }}>
        Active Cross-Vendor Tolerance Exceptions (&gt;5% Variance)
      </Typography>

      {breaks.map((b) => (
        <Card key={b.exceptionId} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 2, mb: 2, color: '#F8FAFC' }}>
          <CardContent>
            <Box display="flex" justifyContent="space-between" alignItems="flex-start" mb={2}>
              <Box>
                <Stack direction="row" spacing={1} alignItems="center">
                  <Typography variant="subtitle1" sx={{ fontWeight: 700, color: '#38BDF8' }}>
                    {b.entityName}
                  </Typography>
                  <Chip label={b.domainKey} size="small" sx={{ bgcolor: '#1E293B', color: '#94A3B8', fontSize: 10, height: 20 }} />
                </Stack>
                <Typography variant="caption" sx={{ fontFamily: 'monospace', color: '#64748B' }}>
                  SID: {b.masterEntitySID} | Field: <strong>{b.fieldName}</strong>
                </Typography>
              </Box>

              <Chip
                icon={b.status === 'APPLIED' ? <ValidIcon sx={{ fontSize: 14 }} /> : <BreakIcon sx={{ fontSize: 14 }} />}
                label={b.status === 'APPLIED' ? 'Resolved & Mastered' : 'Tolerance Breach (>5%)'}
                size="small"
                sx={{
                  bgcolor: b.status === 'APPLIED' ? '#064E3B' : '#450A0A',
                  color: b.status === 'APPLIED' ? '#34D399' : '#FCA5A5',
                  fontWeight: 700,
                  fontSize: 10
                }}
              />
            </Box>

            {/* Side-by-Side Vendor Comparison Matrix */}
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600, display: 'block', mb: 1 }}>
              Side-by-Side Competing Vendor Payloads:
            </Typography>
            <Grid container spacing={2} mb={2}>
              {b.competingFeeds.map((feed) => (
                <Grid   key={feed.vendor} size={{ xs: 12, sm: 4 }}>
                  <Paper
                    sx={{
                      p: 1.5,
                      bgcolor: feed.vendor === b.aiWinner ? 'rgba(0, 212, 255, 0.08)' : '#071526',
                      border: feed.vendor === b.aiWinner ? '1px solid #00D4FF' : '1px solid #1E293B',
                      borderRadius: 1
                    }}
                  >
                    <Stack direction="row" justifyContent="space-between" alignItems="center">
                      <Typography variant="caption" sx={{ fontWeight: 700, color: feed.vendor === b.aiWinner ? '#00D4FF' : '#94A3B8' }}>
                        {feed.vendor} {feed.vendor === b.aiWinner && '(AI Winner)'}
                      </Typography>
                      <Typography variant="caption" sx={{ color: '#64748B', fontSize: 10 }}>{feed.timeAge}</Typography>
                    </Stack>
                    <Typography
                      variant="h6"
                      sx={{
                        fontWeight: 700,
                        color: feed.variancePct && feed.variancePct > 5 ? '#EF4444' : '#F8FAFC',
                        mt: 0.5,
                        fontFamily: 'monospace'
                      }}
                    >
                      {feed.value} {feed.variancePct ? `(Dev: +${feed.variancePct}%)` : ''}
                    </Typography>
                  </Paper>
                </Grid>
              ))}
            </Grid>

            {/* AI Steward Diagnostic Banner */}
            <Box sx={{ p: 1.5, bgcolor: '#071526', borderRadius: 1.5, border: '1px solid #1E293B', mb: 2 }}>
              <Stack direction="row" spacing={1.5} alignItems="flex-start">
                <AiIcon sx={{ color: '#00D4FF', fontSize: 20, mt: 0.3 }} />
                <Box>
                  <Typography variant="caption" sx={{ color: '#00D4FF', fontWeight: 700, textTransform: 'uppercase' }}>
                    AI Steward Diagnostic ({Math.round(b.aiConfidence * 100)}% Confidence)
                  </Typography>
                  <Typography variant="body2" sx={{ color: '#CBD5E1', fontSize: 12, mt: 0.2 }}>
                    {b.aiDiagnostic}
                  </Typography>
                </Box>
              </Stack>
            </Box>

            {/* Action Buttons */}
            {b.status === 'PENDING_APPROVAL' && (
              <Stack direction="row" spacing={2} justifyContent="flex-end">
                <Button
                  variant="outlined"
                  size="small"
                  startIcon={<OverrideIcon sx={{ fontSize: 14 }} />}
                  sx={{ borderColor: '#334155', color: '#94A3B8', textTransform: 'none', fontSize: 11 }}
                >
                  Manual Steward Override
                </Button>
                <Button
                  variant="contained"
                  size="small"
                  startIcon={<ApproveIcon sx={{ fontSize: 14 }} />}
                  onClick={() => handleApproveProposal(b.exceptionId)}
                  sx={{ bgcolor: '#0284C7', textTransform: 'none', fontWeight: 700, fontSize: 11, '&:hover': { bgcolor: '#0369A1' } }}
                >
                  Approve AI Winning Vendor ({b.aiWinner})
                </Button>
              </Stack>
            )}
          </CardContent>
        </Card>
      ))}
    </Paper>
  );
};

export default AIDataStewardWorkstation;
