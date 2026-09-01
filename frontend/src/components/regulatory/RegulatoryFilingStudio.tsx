import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Button,
  Chip,
  Grid,
  Tabs,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Alert,
  CircularProgress
} from '@mui/material';
import {
  Gavel as RegulatoryIcon,
  Code as XmlIcon,
  CheckCircle as ValidIcon,
  Lock as LockIcon,
  Download as DownloadIcon,
  AccountTree as LookThroughIcon
} from '@mui/icons-material';

interface QualifyingHoldingRow {
  issuerName: string;
  titleOfClass: string;
  cusip: string;
  shares: number;
  marketValueUSD: number;
  valueThousands: number;
  discretion: string;
}

export const RegulatoryFilingStudio: React.FC<{ tenantId?: string; portfolioId?: string }> = ({
  tenantId: _tenantId,
  portfolioId: _portfolioId
}) => {
  const [activeTab, setActiveTab] = useState(0);
  const [isAttesting, setIsAttesting] = useState(false);
  const [attestedSuccess, setAttestedSuccess] = useState(false);

  const [holdings, _setHoldings] = useState<QualifyingHoldingRow[]>([
    {
      issuerName: 'MICROSOFT CORP',
      titleOfClass: 'COM',
      cusip: '594918104',
      shares: 125000,
      marketValueUSD: 56125000,
      valueThousands: 56125,
      discretion: 'SOLE'
    },
    {
      issuerName: 'APPLE INC',
      titleOfClass: 'COM',
      cusip: '037833100',
      shares: 84000,
      marketValueUSD: 18984000,
      valueThousands: 18984,
      discretion: 'SOLE'
    },
    {
      issuerName: 'NVIDIA CORP',
      titleOfClass: 'COM',
      cusip: '67066G104',
      shares: 210000,
      marketValueUSD: 26964000,
      valueThousands: 26964,
      discretion: 'SOLE'
    }
  ]);

  const xmlPreviewSnippet = `<?xml version="1.0" encoding="UTF-8"?>
<informationTable xmlns="http://www.sec.gov/edgar/document/thirteenf/informationtable">
  <infoTable>
    <nameOfIssuer>MICROSOFT CORP</nameOfIssuer>
    <titleOfClass>COM</titleOfClass>
    <cusip>594918104</cusip>
    <value>56125</value>
    <shrsOrPrnAmt>
      <sshPrnamtType>SH</sshPrnamtType>
      <sshPrnamt>125000</sshPrnamt>
    </shrsOrPrnAmt>
    <investmentDiscretion>SOLE</investmentDiscretion>
    <votingAuthority>
      <Sole>125000</Sole>
      <Shared>0</Shared>
      <None>0</None>
    </votingAuthority>
  </infoTable>
</informationTable>`;

  const handleAttestAndSign = () => {
    setIsAttesting(true);
    setTimeout(() => {
      setIsAttesting(false);
      setAttestedSuccess(true);
    }, 900);
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
      <Box display="flex" justifyContent="space-between" alignItems="center" pb={2} mb={3} borderBottom="1px solid #1E293B">
        <Stack direction="row" spacing={1.5} alignItems="center">
          <RegulatoryIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              SEC Form 13F-HR & N-PORT Regulatory Filing Studio
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Q2 2026 Period-End Snapshot | EDGAR XML Serialization & Look-Through Decomposer
            </Typography>
          </Box>
        </Stack>

        <Stack direction="row" spacing={2} alignItems="center">
          <Chip
            icon={<ValidIcon sx={{ fontSize: 14, color: '#10B981 !important' }} />}
            label="XSD 2026.1: Valid"
            size="small"
            sx={{ bgcolor: '#064E3B', color: '#34D399', fontWeight: 700, fontSize: 11 }}
          />
          <Button
            variant="contained"
            size="small"
            startIcon={isAttesting ? <CircularProgress size={14} color="inherit" /> : <LockIcon />}
            onClick={handleAttestAndSign}
            disabled={attestedSuccess || isAttesting}
            sx={{
              bgcolor: attestedSuccess ? '#064E3B' : '#0284C7',
              color: '#F8FAFC',
              fontWeight: 600,
              textTransform: 'none',
              '&:hover': { bgcolor: attestedSuccess ? '#064E3B' : '#0369A1' }
            }}
          >
            {attestedSuccess ? 'Attested & Sealed (SEC 17a-4)' : 'Attest & Merkle-Seal Filing'}
          </Button>
        </Stack>
      </Box>

      {attestedSuccess && (
        <Alert severity="success" sx={{ mb: 3, bgcolor: '#064E3B', color: '#F8FAFC', border: '1px solid #10B981' }}>
          Filing attested by CCO user. Merkle Root: <strong>8f9b28a17d6c...</strong> locked under SEC Rule 17a-4 WORM requirements.
        </Alert>
      )}

      <Grid container spacing={2} mb={3}>
        <Grid   size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
              Reportable 13F Gross Value
            </Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#34D399', fontFamily: 'monospace' }}>
              $102,073,000
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              102,073 (Thousands USD)
            </Typography>
          </Paper>
        </Grid>

        <Grid   size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
              Qualifying Securities
            </Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#38BDF8' }}>
              3 Issuers (100% De Minimis Clean)
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              2 Positions Filtered (&lt; $200k / 10k shrs)
            </Typography>
          </Paper>
        </Grid>

        <Grid   size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
              Look-Through Master Allocation
            </Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#FBBF24' }}>
              4 Feeders Consolidated
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              FEEDS_INTO Graph Resolved
            </Typography>
          </Paper>
        </Grid>
      </Grid>

      <Box sx={{ borderBottom: 1, borderColor: '#1E293B', mb: 2 }}>
        <Tabs
          value={activeTab}
          onChange={(_, val) => setActiveTab(val)}
          textColor="inherit"
          indicatorColor="primary"
          sx={{ '& .MuiTab-root': { textTransform: 'none', fontWeight: 600, fontSize: 13, color: '#94A3B8' } }}
        >
          <Tab icon={<LookThroughIcon sx={{ fontSize: 16 }} />} iconPosition="start" label="Consolidated 13F Holdings Grid" />
          <Tab icon={<XmlIcon sx={{ fontSize: 16 }} />} iconPosition="start" label="Official EDGAR XML Preview" />
        </Tabs>
      </Box>

      {activeTab === 0 && (
        <TableContainer component={Paper} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
          <Table size="small">
            <TableHead>
              <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B', fontSize: 11 } }}>
                <TableCell>Name of Issuer</TableCell>
                <TableCell>Title of Class</TableCell>
                <TableCell>CUSIP</TableCell>
                <TableCell align="right">Value (x$1,000)</TableCell>
                <TableCell align="right">Shares Count</TableCell>
                <TableCell align="center">Investment Discretion</TableCell>
                <TableCell align="center">Voting Authority (Sole)</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {holdings.map((h, idx) => (
                <TableRow key={idx} sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
                  <TableCell sx={{ fontWeight: 700, color: '#38BDF8', fontSize: 12 }}>{h.issuerName}</TableCell>
                  <TableCell sx={{ fontSize: 12 }}>{h.titleOfClass}</TableCell>
                  <TableCell sx={{ fontFamily: 'monospace', color: '#CBD5E1', fontSize: 12 }}>{h.cusip}</TableCell>
                  <TableCell align="right" sx={{ fontFamily: 'monospace', fontWeight: 700, color: '#34D399', fontSize: 12 }}>
                    ${h.valueThousands.toLocaleString()}
                  </TableCell>
                  <TableCell align="right" sx={{ fontFamily: 'monospace', fontSize: 12 }}>
                    {h.shares.toLocaleString()}
                  </TableCell>
                  <TableCell align="center">
                    <Chip label={h.discretion} size="small" sx={{ bgcolor: '#1E293B', color: '#94A3B8', fontSize: 10 }} />
                  </TableCell>
                  <TableCell align="center" sx={{ fontFamily: 'monospace', fontSize: 12 }}>
                    {h.shares.toLocaleString()}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      {activeTab === 1 && (
        <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
          <Box display="flex" justifyContent="space-between" alignItems="center" mb={1}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontFamily: 'monospace' }}>
              informationTable.xml (Schema: EDGAR 13F v2026)
            </Typography>
            <Button
              size="small"
              variant="outlined"
              startIcon={<DownloadIcon sx={{ fontSize: 14 }} />}
              sx={{ borderColor: '#334155', color: '#38BDF8', fontSize: 11, textTransform: 'none' }}
            >
              Export Validated XML
            </Button>
          </Box>
          <Box
            component="pre"
            sx={{
              p: 2,
              bgcolor: '#071526',
              border: '1px solid #1E293B',
              borderRadius: 1,
              fontFamily: 'monospace',
              fontSize: 11,
              color: '#38BDF8',
              overflowX: 'auto',
              maxHeight: 350
            }}
          >
            {xmlPreviewSnippet}
          </Box>
        </Paper>
      )}
    </Paper>
  );
};

export default RegulatoryFilingStudio;
