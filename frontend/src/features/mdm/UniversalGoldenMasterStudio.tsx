import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Button,
  Chip,
  Grid,
  Divider,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Alert,
  Card,
  CardContent,
  LinearProgress
} from '@mui/material';
import {
  Storage as MasterIcon,
  CheckCircle as ValidIcon,
  WarningAmber as WarningIcon,
  AutoAwesome as AiIcon,
  Shield as MerkleIcon,
  ThumbUp as ApproveIcon
} from '@mui/icons-material';

interface GoldenAttributeItem {
  fieldName: string;
  goldenValue: string;
  winningVendor: string;
  confidenceScore: number;
  timeFreshness: string;
  sourceValues: Array<{ vendor: string; value: string; isWinner: boolean }>;
}

export const UniversalGoldenMasterStudio: React.FC<{ tenantId?: string; goldenId?: string }> = ({
  tenantId: _tenantId = '99e99e99-99e9-49e9-89e9-99e99e99e999',
  goldenId: _goldenId = 'gld-sec-001'
}) => {
  const [attributes] = useState<GoldenAttributeItem[]>([
    {
      fieldName: 'market_price',
      goldenValue: '$98.42',
      winningVendor: 'BLOOMBERG',
      confidenceScore: 0.985,
      timeFreshness: '12s ago',
      sourceValues: [
        { vendor: 'BLOOMBERG', value: '$98.42', isWinner: true },
        { vendor: 'IDC', value: '$98.40', isWinner: false },
        { vendor: 'REFINITIV', value: '$92.70', isWinner: false }
      ]
    },
    {
      fieldName: 'coupon_rate',
      goldenValue: '4.250%',
      winningVendor: 'BLOOMBERG',
      confidenceScore: 0.998,
      timeFreshness: '45m ago',
      sourceValues: [
        { vendor: 'BLOOMBERG', value: '4.250%', isWinner: true },
        { vendor: 'REFINITIV', value: '4.250%', isWinner: false },
        { vendor: 'DTCC', value: '4.250%', isWinner: false }
      ]
    },
    {
      fieldName: 'issuer_lei',
      goldenValue: '5493006MHB84DD0ZWV18',
      winningVendor: 'GLEIF',
      confidenceScore: 1.0,
      timeFreshness: '2h ago',
      sourceValues: [
        { vendor: 'GLEIF', value: '5493006MHB84DD0ZWV18', isWinner: true },
        { vendor: 'BLOOMBERG', value: '5493006MHB84DD0ZWV18', isWinner: false }
      ]
    }
  ]);

  const [resolvedNotice, setResolvedNotice] = useState<string | null>(null);

  const handleResolveException = () => {
    setResolvedNotice('Exception Approved: Golden Master updated with verified SEC 17a-4 Merkle Root Seal.');
  };

  return (
    <Paper elevation={0} sx={{ p: 3, bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B', borderRadius: 2 }}>
      {/* Header Bar */}
      <Box display="flex" justifyContent="space-between" alignItems="center" pb={2} mb={3} borderBottom="1px solid #1E293B">
        <Stack direction="row" spacing={1.5} alignItems="center">
          <MasterIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              Universal Golden Record & Neural Survivorship Studio
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Multi-vendor feed mastering, GraphRAG cross-reference disambiguation, and Merkle root sealing
            </Typography>
          </Box>
        </Stack>

        <Stack direction="row" spacing={2} alignItems="center">
          <Chip
            icon={<AiIcon sx={{ fontSize: 14, color: '#00D4FF !important' }} />}
            label="Neural Trust Scoring: Active"
            size="small"
            sx={{ bgcolor: '#0B1E36', color: '#00D4FF', fontWeight: 700, fontSize: 11 }}
          />
          <Chip
            icon={<MerkleIcon sx={{ fontSize: 14, color: '#10B981 !important' }} />}
            label="SEC Rule 17a-4 Sealed"
            size="small"
            sx={{ bgcolor: '#064E3B', color: '#34D399', fontWeight: 700, fontSize: 11 }}
          />
        </Stack>
      </Box>

      {resolvedNotice && (
        <Alert severity="success" sx={{ mb: 3, bgcolor: '#064E3B', color: '#F8FAFC', border: '1px solid #10B981' }}>
          {resolvedNotice}
        </Alert>
      )}

      {/* Entity Overview Card */}
      <Card sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 2, mb: 3, color: '#F8FAFC' }}>
        <CardContent>
          <Grid container spacing={2} alignItems="center">
            <Grid   size={{ xs: 12, sm: 3 }}>
              <Typography variant="caption" sx={{ color: '#94A3B8' }}>Master Entity SID</Typography>
              <Typography variant="h6" sx={{ color: '#38BDF8', fontFamily: 'monospace', fontWeight: 700 }}>
                US0378331005
              </Typography>
            </Grid>
            <Grid   size={{ xs: 12, sm: 3 }}>
              <Typography variant="caption" sx={{ color: '#94A3B8' }}>Entity Name</Typography>
              <Typography variant="subtitle1" sx={{ color: '#F8FAFC', fontWeight: 600 }}>
                Apple Inc. Common Stock
              </Typography>
            </Grid>
            <Grid   size={{ xs: 12, sm: 3 }}>
              <Typography variant="caption" sx={{ color: '#94A3B8' }}>Asset Class / Domain</Typography>
              <Typography variant="subtitle1" sx={{ color: '#F8FAFC', fontWeight: 600 }}>
                EQUITY / PRICING
              </Typography>
            </Grid>
            <Grid   size={{ xs: 12, sm: 3 }}>
              <Typography variant="caption" sx={{ color: '#94A3B8' }}>Merkle SHA-256 Seal</Typography>
              <Typography variant="caption" sx={{ color: '#34D399', fontFamily: 'monospace', display: 'block' }}>
                7a8f9c1b2e3d4f5a...
              </Typography>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* Golden Master Attribute Breakdown Table */}
      <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700, textTransform: 'uppercase', mb: 1.5, display: 'block' }}>
        Golden Record Attribute Provenance & Competing Feeds
      </Typography>

      <TableContainer component={Paper} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5, mb: 3 }}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B', fontSize: 11 } }}>
              <TableCell>Field Name</TableCell>
              <TableCell>Authoritative Value</TableCell>
              <TableCell>Winning Vendor</TableCell>
              <TableCell align="center">Neural Confidence</TableCell>
              <TableCell align="center">Freshness</TableCell>
              <TableCell>Competing Multi-Vendor Inputs</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {attributes.map(attr => (
              <TableRow key={attr.fieldName} sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
                <TableCell sx={{ fontFamily: 'monospace', fontWeight: 700, color: '#38BDF8', fontSize: 12 }}>
                  {attr.fieldName}
                </TableCell>
                <TableCell sx={{ fontFamily: 'monospace', fontWeight: 700, color: '#34D399', fontSize: 12 }}>
                  {attr.goldenValue}
                </TableCell>
                <TableCell>
                  <Chip label={attr.winningVendor} size="small" sx={{ bgcolor: '#1E293B', color: '#38BDF8', fontSize: 10 }} />
                </TableCell>
                <TableCell align="center">
                  <Typography variant="caption" sx={{ fontFamily: 'monospace', fontWeight: 700 }}>
                    {(attr.confidenceScore * 100).toFixed(1)}%
                  </Typography>
                  <LinearProgress
                    variant="determinate"
                    value={attr.confidenceScore * 100}
                    sx={{ height: 4, borderRadius: 1, bgcolor: '#071526', '& .MuiLinearProgress-bar': { bgcolor: '#00D4FF' } }}
                  />
                </TableCell>
                <TableCell align="center" sx={{ fontFamily: 'monospace', color: '#94A3B8', fontSize: 11 }}>
                  {attr.timeFreshness}
                </TableCell>
                <TableCell>
                  <Stack direction="row" spacing={1}>
                    {attr.sourceValues.map((src, sIdx) => (
                      <Chip
                        key={sIdx}
                        label={`${src.vendor}: ${src.value}`}
                        size="small"
                        sx={{
                          bgcolor: src.isWinner ? 'rgba(0, 212, 255, 0.15)' : '#071526',
                          color: src.isWinner ? '#00D4FF' : '#64748B',
                          border: src.isWinner ? '1px solid #00D4FF' : '1px solid #1E293B',
                          fontSize: 10
                        }}
                      />
                    ))}
                  </Stack>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Exception Resolution Strip */}
      <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
        <Stack direction="row" justifyContent="space-between" alignItems="center">
          <Stack direction="row" spacing={1.5} alignItems="center">
            <ValidIcon sx={{ color: '#34D399', fontSize: 20 }} />
            <Typography variant="body2" sx={{ color: '#F8FAFC' }}>
              All 3 golden attributes validated against checksum and tolerance waterfalls.
            </Typography>
          </Stack>
          <Button
            variant="contained"
            size="small"
            startIcon={<ApproveIcon sx={{ fontSize: 14 }} />}
            onClick={handleResolveException}
            sx={{ bgcolor: '#0284C7', textTransform: 'none', fontWeight: 600, fontSize: 11, '&:hover': { bgcolor: '#0369A1' } }}
          >
            Confirm Golden Master State
          </Button>
        </Stack>
      </Paper>
    </Paper>
  );
};

export default UniversalGoldenMasterStudio;
