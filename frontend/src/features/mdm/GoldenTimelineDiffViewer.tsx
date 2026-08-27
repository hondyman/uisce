import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow
} from '@mui/material';
import {
  History as TimelineIcon,
  Shield as MerkleIcon
} from '@mui/icons-material';

interface GoldenSnapshotRow {
  effectiveDate: string;
  knowledgeTimestamp: string;
  fieldName: string;
  winningVendor: string;
  previousValue: string;
  currentValue: string;
  merkleSeal: string;
}

export const GoldenTimelineDiffViewer: React.FC<{ entitySID?: string }> = ({
  entitySID = 'SEC_US912810TL44'
}) => {
  const [history] = useState<GoldenSnapshotRow[]>([
    {
      effectiveDate: '2026-08-25',
      knowledgeTimestamp: '2026-08-25 12:45:10 UTC',
      fieldName: 'market_price',
      winningVendor: 'BLOOMBERG',
      previousValue: '$98.35',
      currentValue: '$98.42',
      merkleSeal: '8f9b28a17d6c4e01...'
    },
    {
      effectiveDate: '2026-08-25',
      knowledgeTimestamp: '2026-08-25 12:40:02 UTC',
      fieldName: 'market_price',
      winningVendor: 'IDC',
      previousValue: '$98.10',
      currentValue: '$98.35',
      merkleSeal: '3e1a77c89b02f4a1...'
    },
    {
      effectiveDate: '2026-08-25',
      knowledgeTimestamp: '2026-08-25 08:00:00 UTC',
      fieldName: 'coupon_rate',
      winningVendor: 'REFINITIV',
      previousValue: '—',
      currentValue: '4.250%',
      merkleSeal: '1a90c5e3d7890b22...'
    }
  ]);

  return (
    <Paper elevation={0} sx={{ p: 3, bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B', borderRadius: 2 }}>
      <Box display="flex" justifyContent="space-between" alignItems="center" pb={2} mb={3} borderBottom="1px solid #1E293B">
        <Stack direction="row" spacing={1.5} alignItems="center">
          <TimelineIcon sx={{ color: '#00D4FF', fontSize: 26 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              Bi-Temporal Golden Master Diff & Provenance
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              SID: <code>{entitySID}</code> | Effective Time (Te) vs Knowledge Time (Tk)
            </Typography>
          </Box>
        </Stack>
        <Chip
          icon={<MerkleIcon sx={{ fontSize: 14, color: '#10B981 !important' }} />}
          label="SEC Rule 17a-4 Immutability Verified"
          size="small"
          sx={{ bgcolor: '#064E3B', color: '#34D399', fontWeight: 700, fontSize: 11 }}
        />
      </Box>

      <TableContainer component={Paper} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B', fontSize: 11 } }}>
              <TableCell>Effective Date (Te)</TableCell>
              <TableCell>Knowledge Time (Tk)</TableCell>
              <TableCell>Field Name</TableCell>
              <TableCell>Winning Source</TableCell>
              <TableCell align="right">Previous Value</TableCell>
              <TableCell align="right">Golden Value</TableCell>
              <TableCell>Merkle Leaf Seal</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {history.map((h, idx) => (
              <TableRow key={idx} sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
                <TableCell sx={{ fontFamily: 'monospace', fontSize: 11, color: '#38BDF8' }}>{h.effectiveDate}</TableCell>
                <TableCell sx={{ fontFamily: 'monospace', fontSize: 11, color: '#94A3B8' }}>{h.knowledgeTimestamp}</TableCell>
                <TableCell sx={{ fontWeight: 600, fontSize: 12 }}>{h.fieldName}</TableCell>
                <TableCell>
                  <Chip label={h.winningVendor} size="small" sx={{ bgcolor: '#1E293B', color: '#38BDF8', fontSize: 10 }} />
                </TableCell>
                <TableCell align="right" sx={{ fontFamily: 'monospace', color: '#64748B', fontSize: 11 }}>{h.previousValue}</TableCell>
                <TableCell align="right" sx={{ fontFamily: 'monospace', fontWeight: 700, color: '#34D399', fontSize: 12 }}>{h.currentValue}</TableCell>
                <TableCell sx={{ fontFamily: 'monospace', fontSize: 10, color: '#94A3B8' }}>{h.merkleSeal}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
};

export default GoldenTimelineDiffViewer;
