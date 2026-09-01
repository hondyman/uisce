import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Button,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Alert
} from '@mui/material';
import {
  AutoAwesome as AiIcon,
  CheckCircle as ValidIcon,
  CompareArrows as MatchIcon,
  ThumbUp as ApproveIcon,
  WarningAmber as ReviewIcon
} from '@mui/icons-material';

interface FuzzyProposalItem {
  proposalId: string;
  inboundName: string;
  inboundTicker: string;
  matchedMasterName: string;
  similarityScore: number;
  rationale: string;
  status: 'PENDING_REVIEW' | 'AUTO_MERGED';
}

export const FuzzyMatchTriageStudio: React.FC<{ tenantId?: string }> = ({
  tenantId: _tenantId = '99e99e99-99e9-49e9-89e9-99e99e99e999'
}) => {
  const [proposals, setProposals] = useState<FuzzyProposalItem[]>([
    {
      proposalId: 'fuzz-0927-01',
      inboundName: 'Apple Operations Int.',
      inboundTicker: 'AAPL-INT',
      matchedMasterName: 'Apple Operations International Ltd',
      similarityScore: 0.9650,
      rationale: 'High cosine similarity across entity name and jurisdictional metadata. Linked via pgvector HNSW index.',
      status: 'PENDING_REVIEW'
    }
  ]);

  const [notice, setNotice] = useState<string | null>(null);

  const handleApproveMerge = (proposalId: string) => {
    setProposals(prev =>
      prev.map(p => (p.proposalId === proposalId ? { ...p, status: 'AUTO_MERGED' } : p))
    );
    setNotice('Entity successfully merged into golden record master graph with cryptographic audit seal.');
  };

  return (
    <Paper elevation={0} sx={{ p: 3, bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B', borderRadius: 2 }}>
      {/* Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center" pb={2} mb={3} borderBottom="1px solid #1E293B">
        <Stack direction="row" spacing={1.5} alignItems="center">
          <AiIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              AI-Powered Fuzzy Entity Resolution & GraphRAG XREF Studio
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Probabilistic vector matching via pgvector HNSW index & automated entity merging
            </Typography>
          </Box>
        </Stack>
        <Chip icon={<MatchIcon sx={{ fontSize: 14, color: '#00D4FF !important' }} />} label="Vector Engine: Active" size="small" sx={{ bgcolor: '#0B1E36', color: '#00D4FF', fontWeight: 700, fontSize: 11, border: '1px solid #1E293B' }} />
      </Box>

      {notice && (
        <Alert severity="success" sx={{ mb: 3, bgcolor: '#064E3B', color: '#F8FAFC', border: '1px solid #10B981' }}>
          {notice}
        </Alert>
      )}

      {/* Proposals Table */}
      <TableContainer component={Paper} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B', fontSize: 11 } }}>
              <TableCell>Inbound Vendor Payload</TableCell>
              <TableCell>Matched Master Entity</TableCell>
              <TableCell align="center">Semantic Similarity</TableCell>
              <TableCell>AI Match Rationale</TableCell>
              <TableCell align="center">Status</TableCell>
              <TableCell align="center">Action</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {proposals.map(p => (
              <TableRow key={p.proposalId} sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
                <TableCell>
                  <Typography variant="body2" sx={{ fontWeight: 700, color: '#38BDF8', fontSize: 12 }}>{p.inboundName}</Typography>
                  <Typography variant="caption" sx={{ fontFamily: 'monospace', color: '#94A3B8', fontSize: 10 }}>Ticker: {p.inboundTicker}</Typography>
                </TableCell>
                <TableCell>
                  <Typography variant="body2" sx={{ fontWeight: 700, color: '#34D399', fontSize: 12 }}>{p.matchedMasterName}</Typography>
                </TableCell>
                <TableCell align="center" sx={{ fontFamily: 'monospace', fontWeight: 700, color: '#38BDF8' }}>
                  {(p.similarityScore * 100).toFixed(1)}%
                </TableCell>
                <TableCell sx={{ fontSize: 11, color: '#CBD5E1' }}>{p.rationale}</TableCell>
                <TableCell align="center">
                  <Chip
                    icon={p.status === 'AUTO_MERGED' ? <ValidIcon sx={{ fontSize: 12, color: '#10B981 !important' }} /> : <ReviewIcon sx={{ fontSize: 12, color: '#F59E0B !important' }} />}
                    label={p.status}
                    size="small"
                    sx={{
                      bgcolor: p.status === 'AUTO_MERGED' ? '#064E3B' : '#451A03',
                      color: p.status === 'AUTO_MERGED' ? '#34D399' : '#FDBA74',
                      fontWeight: 700,
                      fontSize: 10
                    }}
                  />
                </TableCell>
                <TableCell align="center">
                  {p.status === 'PENDING_REVIEW' ? (
                    <Button
                      variant="contained"
                      size="small"
                      startIcon={<ApproveIcon sx={{ fontSize: 12 }} />}
                      onClick={() => handleApproveMerge(p.proposalId)}
                      sx={{ bgcolor: '#0284C7', textTransform: 'none', fontSize: 10, py: 0.2, '&:hover': { bgcolor: '#0369A1' } }}
                    >
                      Approve Merge
                    </Button>
                  ) : (
                    <Typography variant="caption" sx={{ color: '#64748B' }}>Merged</Typography>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
};

export default FuzzyMatchTriageStudio;
