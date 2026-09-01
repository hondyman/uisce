import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Chip,
  Stack,
  Button,
} from '@mui/material';
import LockIcon from '@mui/icons-material/Lock';
import ShieldCheckIcon from '@mui/icons-material/VerifiedUser';
import DifferenceIcon from '@mui/icons-material/Difference';

export interface AuditPassportViewModel {
  passportId: string;
  statementId: string;
  effectiveDate: string;
  knowledgeDate: string;
  pdfArtifactSha256: string;
  merklePassportHash: string;
  signerIdentity: string;
  status: 'SEALED' | 'VERIFIED' | 'RESTATEMENT_PENDING';
}

export interface RestatementDeltaViewModel {
  positionId: string;
  securityName: string;
  fieldKey: string;
  originalValue: number;
  restatedValue: number;
  varianceAmount: number;
  correctionReason: string;
}

interface StatementSnapshotVaultProps {
  passport?: AuditPassportViewModel;
  originalNav?: number;
  restatedNav?: number;
  deltas?: RestatementDeltaViewModel[];
  onVerifyIntegrity?: () => void;
}

export const StatementSnapshotVault: React.FC<StatementSnapshotVaultProps> = ({
  passport = {
    passportId: 'pass_2026_q1_001',
    statementId: 'STMT-2026-Q1-001',
    effectiveDate: '2026-03-31',
    knowledgeDate: '2026-04-02 14:30:00 UTC',
    pdfArtifactSha256: 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
    merklePassportHash: '8f434346648f6b96df89dda901c5176b10a6d83961dd3c1ac88b59b2dc327aa4',
    signerIdentity: 'Chief Compliance Officer (CCO)',
    status: 'SEALED',
  },
  originalNav = 100000000,
  restatedNav = 100125000,
  deltas = [
    {
      positionId: 'pos_01',
      securityName: 'Apex Growth Fund LP',
      fieldKey: 'nav_settled',
      originalValue: 12500000,
      restatedValue: 12625000,
      varianceAmount: 125000,
      correctionReason: 'Late Q1 LP Capital Call / Distribution True-Up',
    },
  ],
  onVerifyIntegrity,
}) => {
  const [activeTab, setActiveTab] = useState<'PASSPORT' | 'RESTATEMENT_DIFF'>('PASSPORT');
  const varianceBps = ((restatedNav - originalNav) / originalNav) * 10000;
  const hasDrift = deltas.length > 0;

  return (
    <Box sx={{ width: '100%', bgcolor: '#050D1A', color: '#fff', borderRadius: 2, border: '1px solid #1E293B', overflow: 'hidden', fontFamily: 'sans-serif' }}>
      
      {/* Header Bar */}
      <Box sx={{ p: 2, bgcolor: '#071526', borderBottom: '1px solid #1E293B', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <LockIcon sx={{ color: '#00D4FF', fontSize: 22 }} />
          <Box>
            <Typography variant="subtitle2" fontWeight="700" sx={{ letterSpacing: 0.5 }}>
              Immutable Statement Vault & SEC Rule 17a-4 Ledger
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              Statement ID: <span style={{ fontFamily: 'monospace', color: '#22D3EE' }}>{passport.statementId}</span> | Effective Date: {passport.effectiveDate}
            </Typography>
          </Box>
        </Box>

        <Stack direction="row" spacing={1.5} alignItems="center">
          <Chip
            size="small"
            icon={<ShieldCheckIcon sx={{ fontSize: 14 }} />}
            label="HMAC-SHA256 SEALED"
            sx={{ bgcolor: 'rgba(16, 185, 129, 0.15)', color: '#10B981', fontWeight: 700, fontSize: '10px', border: '1px solid rgba(16, 185, 129, 0.3)' }}
          />
          <Button
            size="small"
            variant="outlined"
            onClick={onVerifyIntegrity}
            sx={{ color: '#00D4FF', borderColor: 'rgba(0, 212, 255, 0.4)', fontSize: '11px', textTransform: 'none', px: 1.5 }}
          >
            Verify Passport
          </Button>
        </Stack>
      </Box>

      {/* Navigation Tabs */}
      <Box sx={{ px: 2, pt: 1.5, bgcolor: '#0B1E36', display: 'flex', gap: 2, borderBottom: '1px solid #1E293B' }}>
        <button
          onClick={() => setActiveTab('PASSPORT')}
          style={{
            paddingBottom: '8px',
            paddingLeft: '4px',
            paddingRight: '4px',
            fontSize: '12px',
            fontWeight: 600,
            display: 'flex',
            alignItems: 'center',
            gap: '6px',
            background: 'none',
            border: 'none',
            cursor: 'pointer',
            borderBottom: activeTab === 'PASSPORT' ? '2px solid #00D4FF' : '2px solid transparent',
            color: activeTab === 'PASSPORT' ? '#00D4FF' : '#94A3B8',
          }}
        >
          <LockIcon sx={{ fontSize: 14 }} /> Cryptographic Passport
        </button>
        <button
          onClick={() => setActiveTab('RESTATEMENT_DIFF')}
          style={{
            paddingBottom: '8px',
            paddingLeft: '4px',
            paddingRight: '4px',
            fontSize: '12px',
            fontWeight: 600,
            display: 'flex',
            alignItems: 'center',
            gap: '6px',
            background: 'none',
            border: 'none',
            cursor: 'pointer',
            borderBottom: activeTab === 'RESTATEMENT_DIFF' ? '2px solid #F59E0B' : '2px solid transparent',
            color: activeTab === 'RESTATEMENT_DIFF' ? '#F59E0B' : '#94A3B8',
          }}
        >
          <DifferenceIcon sx={{ fontSize: 14 }} /> Bitemporal Restatements ({deltas.length})
        </button>
      </Box>

      {/* Tab 1: Cryptographic Passport View */}
      {activeTab === 'PASSPORT' && (
        <Box sx={{ p: 2.5 }}>
          <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2, fontSize: '11px', fontFamily: 'monospace' }}>
            <Paper sx={{ p: 2, bgcolor: '#071526', border: '1px solid rgba(255,255,255,0.06)', borderRadius: 1.5 }}>
              <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block', mb: 1, textTransform: 'uppercase', fontWeight: 700 }}>
                Artifact Fingerprints (Immutable Hash)
              </Typography>
              <Stack spacing={1.5}>
                <Box>
                  <span style={{ color: '#64748B', display: 'block', fontSize: '10px' }}>PDF/A Binary SHA-256 Digest:</span>
                  <span style={{ color: '#34D399', wordBreak: 'break-all' }}>{passport.pdfArtifactSha256}</span>
                </Box>
                <Box>
                  <span style={{ color: '#64748B', display: 'block', fontSize: '10px' }}>Merkle Passport Signature:</span>
                  <span style={{ color: '#00D4FF', wordBreak: 'break-all' }}>{passport.merklePassportHash}</span>
                </Box>
              </Stack>
            </Paper>

            <Paper sx={{ p: 2, bgcolor: '#071526', border: '1px solid rgba(255,255,255,0.06)', borderRadius: 1.5 }}>
              <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block', mb: 1, textTransform: 'uppercase', fontWeight: 700 }}>
                Bitemporal Timeline & Provenance
              </Typography>
              <Stack spacing={1.5}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <span style={{ color: '#94A3B8' }}>Effective As-Of Date (Te):</span>
                  <span style={{ color: '#fff', fontWeight: 700 }}>{passport.effectiveDate}</span>
                </Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <span style={{ color: '#94A3B8' }}>Sealed Knowledge Timestamp (Tk):</span>
                  <span style={{ color: '#FBBF24' }}>{passport.knowledgeDate}</span>
                </Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <span style={{ color: '#94A3B8' }}>Signer Officer Identity:</span>
                  <span style={{ color: '#E2E8F0' }}>{passport.signerIdentity}</span>
                </Box>
              </Stack>
            </Paper>
          </Box>
        </Box>
      )}

      {/* Tab 2: Bitemporal Restatement Diff View */}
      {activeTab === 'RESTATEMENT_DIFF' && (
        <Box sx={{ p: 2.5 }}>
          {/* Summary Reconcile Banner */}
          <Box sx={{ p: 2, mb: 2, bgcolor: '#071526', border: '1px solid #1E293B', borderRadius: 1.5, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Box sx={{ display: 'flex', gap: 4, fontSize: '11px', fontFamily: 'monospace' }}>
              <Box>
                <span style={{ color: '#94A3B8', display: 'block', fontSize: '10px' }}>Original Published NAV (Tk):</span>
                <span style={{ color: '#F8FAFC', fontWeight: 700, fontSize: '13px' }}>${originalNav.toLocaleString()}</span>
              </Box>
              <Box>
                <span style={{ color: '#94A3B8', display: 'block', fontSize: '10px' }}>Live Restated NAV (Tnow):</span>
                <span style={{ color: '#22D3EE', fontWeight: 700, fontSize: '13px' }}>${restatedNav.toLocaleString()}</span>
              </Box>
              <Box>
                <span style={{ color: '#94A3B8', display: 'block', fontSize: '10px' }}>Net Restatement Variance:</span>
                <span style={{ fontWeight: 700, fontSize: '13px', color: varianceBps >= 0 ? '#34D399' : '#FB7185' }}>
                  {varianceBps >= 0 ? '+' : ''}{varianceBps.toFixed(2)} bps
                </span>
              </Box>
            </Box>

            <Chip
              size="small"
              icon={hasDrift ? <DifferenceIcon sx={{ fontSize: 14 }} /> : <ShieldCheckIcon sx={{ fontSize: 14 }} />}
              label={hasDrift ? 'LATE CORRECTIONS DETECTED' : 'PERFECT RECONCILIATION'}
              color={hasDrift ? 'warning' : 'success'}
              sx={{ fontWeight: 700, fontSize: '10px' }}
            />
          </Box>

          {/* Delta Itemization Table */}
          <Box sx={{ overflowX: 'auto', border: '1px solid #1E293B', borderRadius: 1 }}>
            <table style={{ width: '100%', textAlign: 'left', borderCollapse: 'collapse', fontSize: '11px', fontFamily: 'monospace' }}>
              <thead style={{ backgroundColor: '#071526', color: '#94A3B8', textTransform: 'uppercase', fontSize: '10px', borderBottom: '1px solid #1E293B' }}>
                <tr>
                  <th style={{ padding: '8px 12px' }}>Position</th>
                  <th style={{ padding: '8px 12px' }}>Field</th>
                  <th style={{ padding: '8px 12px', textAlign: 'right' }}>Published (Tk)</th>
                  <th style={{ padding: '8px 12px', textAlign: 'right' }}>Restated (Tnow)</th>
                  <th style={{ padding: '8px 12px', textAlign: 'right' }}>Delta ($)</th>
                  <th style={{ padding: '8px 12px' }}>Correction Reason</th>
                </tr>
              </thead>
              <tbody>
                {deltas.map((d, idx) => (
                  <tr key={idx} style={{ borderBottom: '1px solid rgba(30, 41, 59, 0.6)' }}>
                    <td style={{ padding: '8px 12px', color: '#E2E8F0', fontFamily: 'sans-serif', fontWeight: 500 }}>{d.securityName}</td>
                    <td style={{ padding: '8px 12px', color: '#67E8F9' }}>{d.fieldKey}</td>
                    <td style={{ padding: '8px 12px', textAlign: 'right', color: '#94A3B8' }}>${d.originalValue.toLocaleString()}</td>
                    <td style={{ padding: '8px 12px', textAlign: 'right', color: '#F8FAFC', fontWeight: 700 }}>${d.restatedValue.toLocaleString()}</td>
                    <td style={{ padding: '8px 12px', textAlign: 'right', fontWeight: 700, color: d.varianceAmount >= 0 ? '#34D399' : '#FB7185' }}>
                      {d.varianceAmount >= 0 ? '+' : ''}${d.varianceAmount.toLocaleString()}
                    </td>
                    <td style={{ padding: '8px 12px', color: '#94A3B8', fontFamily: 'sans-serif' }}>{d.correctionReason}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Box>
        </Box>
      )}

    </Box>
  );
};

export default StatementSnapshotVault;
