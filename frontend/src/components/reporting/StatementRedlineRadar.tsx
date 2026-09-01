import React, { useState } from 'react';
import {
  Box,
  Typography,
  Chip,
  Button,
  Stack,
  TextField,
} from '@mui/material';
import RadarIcon from '@mui/icons-material/Radar';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import ThumbUpIcon from '@mui/icons-material/ThumbUp';
import ThumbDownIcon from '@mui/icons-material/ThumbDown';

export interface RedlineItem {
  itemKey: string;
  displayName: string;
  baselineValue: number;
  currentValue: number;
  varianceAmount: number;
  varianceBps: number;
  variancePct: number;
  severity: 'INFO' | 'WARNING' | 'CRITICAL';
  breachDiagnostic: string;
  isBreached: boolean;
}

interface StatementRedlineRadarProps {
  statementId?: string;
  portfolioName?: string;
  makerIdentity?: string;
  anomalyScore?: number;
  status?: 'PENDING_APPROVAL' | 'APPROVED' | 'REJECTED' | 'AUTO_APPROVED';
  items?: RedlineItem[];
  onApprove?: (notes: string) => void;
  onReject?: (notes: string) => void;
}

export const StatementRedlineRadar: React.FC<StatementRedlineRadarProps> = ({
  statementId = 'STMT-2026-Q2-7721',
  portfolioName = 'Flagship Multi-Asset Alpha Fund',
  makerIdentity = 'pm_jane_doe@fund.com',
  anomalyScore = 58.4,
  status = 'PENDING_APPROVAL',
  items = [
    {
      itemKey: 'total_nav',
      displayName: 'Total Ending Portfolio NAV',
      baselineValue: 100000000.0,
      currentValue: 102500000.0,
      varianceAmount: 2500000.0,
      varianceBps: 250.0,
      variancePct: 2.50,
      severity: 'CRITICAL',
      breachDiagnostic: 'Drift of 250.0 bps exceeds tolerance threshold 50.0 bps',
      isBreached: true,
    },
    {
      itemKey: 'management_fee',
      displayName: 'Management Fee Accrual',
      baselineValue: 150000.0,
      currentValue: 175000.0,
      varianceAmount: 25000.0,
      varianceBps: 1666.7,
      variancePct: 16.67,
      severity: 'WARNING',
      breachDiagnostic: 'Variance of 16.67% exceeds tolerance threshold 5.00%',
      isBreached: true,
    },
  ],
  onApprove,
  onReject,
}) => {
  const [reviewNotes, setReviewNotes] = useState('');
  const breachedCount = items.filter((i) => i.isBreached).length;

  return (
    <Box sx={{ width: '100%', bgcolor: '#050D1A', color: '#fff', borderRadius: 2, border: '1px solid #1E293B', overflow: 'hidden', fontFamily: 'sans-serif' }}>
      
      {/* Header Banner */}
      <Box sx={{ p: 2, bgcolor: '#071526', borderBottom: '1px solid #1E293B', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <RadarIcon sx={{ color: '#00D4FF', fontSize: 24 }} />
          <Box>
            <Typography variant="subtitle2" fontWeight="700" sx={{ letterSpacing: 0.5 }}>
              Pre-Burst Statement Discrepancy Radar & Redline Comparison
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              Statement: <span style={{ fontFamily: 'monospace', color: '#22D3EE' }}>{statementId}</span> | Portfolio: {portfolioName}
            </Typography>
          </Box>
        </Box>

        <Stack direction="row" spacing={1.5} alignItems="center">
          <Chip
            size="small"
            icon={breachedCount > 0 ? <WarningAmberIcon sx={{ fontSize: 14 }} /> : <CheckCircleIcon sx={{ fontSize: 14 }} />}
            label={breachedCount > 0 ? `${breachedCount} DRIFT BREACHES DETECTED` : 'CLEAN RECONCILIATION'}
            color={breachedCount > 0 ? 'warning' : 'success'}
            sx={{ fontWeight: 700, fontSize: '10px' }}
          />
          <Chip
            size="small"
            label={`Anomaly Score: ${anomalyScore.toFixed(1)}`}
            sx={{ bgcolor: anomalyScore >= 50 ? 'rgba(239, 68, 68, 0.2)' : 'rgba(0, 212, 255, 0.1)', color: anomalyScore >= 50 ? '#EF4444' : '#00D4FF', fontWeight: 700, fontSize: '10px', border: '1px solid rgba(255,255,255,0.1)' }}
          />
        </Stack>
      </Box>

      {/* Side-by-Side Redline Table */}
      <Box sx={{ p: 2 }}>
        <Box sx={{ overflowX: 'auto', border: '1px solid #1E293B', borderRadius: 1 }}>
          <table style={{ width: '100%', textAlign: 'left', borderCollapse: 'collapse', fontSize: '11px', fontFamily: 'monospace' }}>
            <thead style={{ backgroundColor: '#071526', color: '#94A3B8', textTransform: 'uppercase', fontSize: '10px', borderBottom: '1px solid #1E293B' }}>
              <tr>
                <th style={{ padding: '10px 12px' }}>Monitored Metric</th>
                <th style={{ padding: '10px 12px', textAlign: 'right' }}>Prior Baseline (T0)</th>
                <th style={{ padding: '10px 12px', textAlign: 'right' }}>Current Valuation (T1)</th>
                <th style={{ padding: '10px 12px', textAlign: 'right' }}>Variance</th>
                <th style={{ padding: '10px 12px', textAlign: 'right' }}>Drift</th>
                <th style={{ padding: '10px 12px' }}>Radar Diagnostic</th>
                <th style={{ padding: '10px 12px', textAlign: 'center' }}>Status</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr
                  key={item.itemKey}
                  style={{
                    borderBottom: '1px solid rgba(30, 41, 59, 0.6)',
                    backgroundColor: item.isBreached ? 'rgba(120, 53, 15, 0.2)' : 'transparent',
                  }}
                >
                  <td style={{ padding: '10px 12px', color: '#E2E8F0', fontFamily: 'sans-serif', fontWeight: 500 }}>{item.displayName}</td>
                  <td style={{ padding: '10px 12px', textAlign: 'right', color: '#94A3B8' }}>
                    ${item.baselineValue.toLocaleString(undefined, { minimumFractionDigits: 2 })}
                  </td>
                  <td style={{ padding: '10px 12px', textAlign: 'right', color: '#F8FAFC', fontWeight: 700 }}>
                    ${item.currentValue.toLocaleString(undefined, { minimumFractionDigits: 2 })}
                  </td>
                  <td style={{ padding: '10px 12px', textAlign: 'right', fontWeight: 700, color: item.varianceAmount >= 0 ? '#34D399' : '#FB7185' }}>
                    {item.varianceAmount >= 0 ? '+' : ''}${item.varianceAmount.toLocaleString(undefined, { minimumFractionDigits: 2 })}
                  </td>
                  <td style={{ padding: '10px 12px', textAlign: 'right', fontWeight: 700, color: item.isBreached ? '#FBBF24' : '#94A3B8' }}>
                    {item.varianceBps.toFixed(1)} bps
                  </td>
                  <td style={{ padding: '10px 12px', fontFamily: 'sans-serif', color: '#CBD5E1', maxWidth: 280 }}>
                    {item.breachDiagnostic}
                  </td>
                  <td style={{ padding: '10px 12px', textAlign: 'center' }}>
                    {item.isBreached ? (
                      <span style={{ padding: '2px 8px', borderRadius: 4, fontSize: '10px', fontWeight: 700, backgroundColor: 'rgba(245, 158, 11, 0.2)', color: '#FCD34D', border: '1px solid rgba(245, 158, 11, 0.3)' }}>
                        BREACH
                      </span>
                    ) : (
                      <span style={{ padding: '2px 8px', borderRadius: 4, fontSize: '10px', fontWeight: 700, backgroundColor: 'rgba(16, 185, 129, 0.15)', color: '#34D399' }}>
                        OK
                      </span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </Box>
      </Box>

      {/* Maker-Checker Approval Console */}
      {status === 'PENDING_APPROVAL' && (
        <Box sx={{ p: 2, bgcolor: '#0B1E36', borderTop: '1px solid #1E293B', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 3 }}>
          <Box sx={{ flex: 1 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block', mb: 0.5 }}>
              4-Eyes Signoff Notes (Maker: <span style={{ fontFamily: 'monospace', color: '#22D3EE' }}>{makerIdentity}</span>)
            </Typography>
            <TextField
              size="small"
              fullWidth
              placeholder="Enter compliance justification or override rationale before authorizing burst..."
              value={reviewNotes}
              onChange={(e) => setReviewNotes(e.target.value)}
              sx={{ bgcolor: '#050D1A', input: { color: '#fff', fontSize: '11px', fontFamily: 'monospace' } }}
            />
          </Box>

          <Stack direction="row" spacing={1.5} sx={{ mt: 2 }}>
            <Button
              variant="contained"
              color="error"
              size="small"
              startIcon={<ThumbDownIcon />}
              onClick={() => onReject && onReject(reviewNotes)}
              sx={{ textTransform: 'none', fontWeight: 700, fontSize: '11px', px: 2 }}
            >
              Reject Batch
            </Button>
            <Button
              variant="contained"
              color="success"
              size="small"
              startIcon={<ThumbUpIcon />}
              onClick={() => onApprove && onApprove(reviewNotes)}
              sx={{ textTransform: 'none', fontWeight: 700, fontSize: '11px', px: 2 }}
            >
              Authorize & Burst
            </Button>
          </Stack>
        </Box>
      )}

    </Box>
  );
};

export default StatementRedlineRadar;
