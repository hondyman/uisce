import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Chip,
  Button,
  Stack,
  Select,
  MenuItem,
  FormControl,
  LinearProgress,
} from '@mui/material';
import ReceiptLongIcon from '@mui/icons-material/ReceiptLong';
import CloudDownloadIcon from '@mui/icons-material/CloudDownload';

export interface MonthlyInvoiceSummary {
  billingPeriod: string;
  totalInvoiceUSD: number;
  monthlyBudgetUSD: number;
  totalReportsRendered: number;
  totalPagesGenerated: number;
  totalGigabytesScanned: number;
  paymentStatus: 'DRAFT' | 'ISSUED' | 'PAID' | 'OVERDUE';
}

export interface ItemizedReportCharge {
  entryId: string;
  clientSliceId: string;
  exportFormat: 'PDF' | 'EXCEL';
  pageCount: number;
  scannedBytes: number;
  astComplexityScore: number;
  queryCostUsd: number;
  renderCostUsd: number;
  storageCostUsd: number;
  totalCostUsd: number;
  createdAt: string;
}

interface ChargebackDashboardProps {
  tenantName?: string;
  invoice?: MonthlyInvoiceSummary;
  charges?: ItemizedReportCharge[];
  onExportInvoiceCSV?: () => void;
}

export const ReportChargebackDashboard: React.FC<ChargebackDashboardProps> = ({
  tenantName = 'Meridian Global Asset Management',
  invoice = {
    billingPeriod: '2026-08',
    totalInvoiceUSD: 1420.50,
    monthlyBudgetUSD: 5000.00,
    totalReportsRendered: 12500,
    totalPagesGenerated: 25000,
    totalGigabytesScanned: 48.5,
    paymentStatus: 'DRAFT',
  },
  charges = [
    {
      entryId: 'chg_01',
      clientSliceId: 'CLIENT_ACC_8819',
      exportFormat: 'PDF',
      pageCount: 2,
      scannedBytes: 1073741824,
      astComplexityScore: 35.0,
      queryCostUsd: 0.0389,
      renderCostUsd: 0.0153,
      storageCostUsd: 0.000012,
      totalCostUsd: 0.054212,
      createdAt: '2026-08-23 14:15:00',
    },
  ],
  onExportInvoiceCSV,
}) => {
  const [selectedPeriod, setSelectedPeriod] = useState(invoice.billingPeriod || '2026-08');
  const budgetUsagePct = Math.min(100, (invoice.totalInvoiceUSD / (invoice.monthlyBudgetUSD || 5000)) * 100);

  return (
    <Box sx={{ width: '100%', bgcolor: '#050D1A', color: '#fff', borderRadius: 2, border: '1px solid #1E293B', overflow: 'hidden', fontFamily: 'sans-serif' }}>
      
      {/* Header Bar */}
      <Box sx={{ p: 2.5, bgcolor: '#071526', borderBottom: '1px solid #1E293B', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <ReceiptLongIcon sx={{ color: '#00D4FF', fontSize: 24 }} />
          <Box>
            <Typography variant="subtitle1" fontWeight="700" sx={{ letterSpacing: 0.5 }}>
              FinOps Report Chargeback & Invoicing Center
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              Tenant: <span style={{ fontWeight: 600, color: '#E2E8F0' }}>{tenantName}</span> | Period: {selectedPeriod}
            </Typography>
          </Box>
        </Box>

        <Stack direction="row" spacing={2} alignItems="center">
          <FormControl size="small">
            <Select
              value={selectedPeriod}
              onChange={(e) => setSelectedPeriod(e.target.value)}
              sx={{ bgcolor: '#050D1A', color: '#00D4FF', fontSize: '12px', height: 32, border: '1px solid rgba(0, 212, 255, 0.3)' }}
            >
              <MenuItem value="2026-08">August 2026</MenuItem>
              <MenuItem value="2026-07">July 2026</MenuItem>
              <MenuItem value="2026-06">June 2026</MenuItem>
            </Select>
          </FormControl>

          <Button
            size="small"
            variant="contained"
            startIcon={<CloudDownloadIcon />}
            onClick={onExportInvoiceCSV}
            sx={{ bgcolor: '#00D4FF', color: '#050D1A', fontWeight: 700, fontSize: '11px', textTransform: 'none', '&:hover': { bgcolor: '#38BDF8' } }}
          >
            Export Invoice CSV
          </Button>
        </Stack>
      </Box>

      {/* KPI Cards & Budget Usage */}
      <Box sx={{ p: 2.5, bgcolor: '#0B1E36', display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 2, borderBottom: '1px solid #1E293B' }}>
        
        <Paper sx={{ p: 2, bgcolor: '#071526', border: '1px solid rgba(255,255,255,0.06)', borderRadius: 1.5 }}>
          <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block' }}>Total Period Spend</Typography>
          <Typography variant="h5" fontWeight="700" sx={{ color: '#00D4FF', my: 0.5 }}>
            ${invoice.totalInvoiceUSD.toFixed(2)}
          </Typography>
          <Box sx={{ mt: 1 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', fontSize: '10px', color: '#94A3B8', mb: 0.5 }}>
              <span>Budget Usage</span>
              <span style={{ fontWeight: 700, color: '#fff' }}>{budgetUsagePct.toFixed(1)}%</span>
            </Box>
            <LinearProgress
              variant="determinate"
              value={budgetUsagePct}
              sx={{
                height: 6,
                borderRadius: 1,
                bgcolor: '#050D1A',
                '& .MuiLinearProgress-bar': {
                  bgcolor: budgetUsagePct >= 95 ? '#EF4444' : budgetUsagePct >= 80 ? '#F59E0B' : '#10B981',
                },
              }}
            />
          </Box>
        </Paper>

        <Paper sx={{ p: 2, bgcolor: '#071526', border: '1px solid rgba(255,255,255,0.06)', borderRadius: 1.5 }}>
          <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block' }}>Reports Rendered</Typography>
          <Typography variant="h5" fontWeight="700" sx={{ color: '#10B981', my: 0.5 }}>
            {invoice.totalReportsRendered.toLocaleString()}
          </Typography>
          <Typography variant="caption" sx={{ color: '#64748B' }}>
            {invoice.totalPagesGenerated.toLocaleString()} Compiled Pages
          </Typography>
        </Paper>

        <Paper sx={{ p: 2, bgcolor: '#071526', border: '1px solid rgba(255,255,255,0.06)', borderRadius: 1.5 }}>
          <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block' }}>Data Scanned Volume</Typography>
          <Typography variant="h5" fontWeight="700" sx={{ color: '#F59E0B', my: 0.5 }}>
            {invoice.totalGigabytesScanned.toFixed(2)} GB
          </Typography>
          <Typography variant="caption" sx={{ color: '#64748B' }}>
            Hybrid StarRocks / Iceberg Scans
          </Typography>
        </Paper>

        <Paper sx={{ p: 2, bgcolor: '#071526', border: '1px solid rgba(255,255,255,0.06)', borderRadius: 1.5, display: 'flex', flexDirection: 'column', justifyContent: 'space-between' }}>
          <div>
            <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block' }}>Invoice Status</Typography>
            <Chip
              size="small"
              label={invoice.paymentStatus}
              sx={{
                mt: 1,
                fontWeight: 700,
                fontSize: '11px',
                bgcolor: invoice.paymentStatus === 'PAID' ? 'rgba(16, 185, 129, 0.15)' : 'rgba(0, 212, 255, 0.15)',
                color: invoice.paymentStatus === 'PAID' ? '#10B981' : '#00D4FF',
                border: '1px solid rgba(255,255,255,0.1)',
              }}
            />
          </div>
          <Typography variant="caption" sx={{ color: '#64748B', fontSize: '10px' }}>
            Rule 7 Tenant Isolated
          </Typography>
        </Paper>

      </Box>

      {/* Itemized Chargeback Ledger Table */}
      <Box sx={{ p: 2.5 }}>
        <Typography variant="caption" fontWeight="700" sx={{ color: '#00D4FF', textTransform: 'uppercase', letterSpacing: 0.5, mb: 1.5, display: 'block' }}>
          Itemized Report Execution Ledger
        </Typography>

        <Box sx={{ overflowX: 'auto', border: '1px solid #1E293B', borderRadius: 1 }}>
          <table style={{ width: '100%', textAlign: 'left', borderCollapse: 'collapse', fontSize: '11px', fontFamily: 'monospace' }}>
            <thead style={{ backgroundColor: '#071526', color: '#94A3B8', textTransform: 'uppercase', fontSize: '10px', borderBottom: '1px solid #1E293B' }}>
              <tr>
                <th style={{ padding: '10px 12px' }}>Slice / Client ID</th>
                <th style={{ padding: '10px 12px' }}>Format</th>
                <th style={{ padding: '10px 12px', textAlign: 'right' }}>Pages</th>
                <th style={{ padding: '10px 12px', textAlign: 'right' }}>Complexity</th>
                <th style={{ padding: '10px 12px', textAlign: 'right' }}>Query ($)</th>
                <th style={{ padding: '10px 12px', textAlign: 'right' }}>Render ($)</th>
                <th style={{ padding: '10px 12px', textAlign: 'right' }}>Storage ($)</th>
                <th style={{ padding: '10px 12px', textAlign: 'right', color: '#34D399' }}>Total Attributed</th>
              </tr>
            </thead>
            <tbody>
              {charges.map((c) => (
                <tr key={c.entryId} style={{ borderBottom: '1px solid rgba(30, 41, 59, 0.6)' }}>
                  <td style={{ padding: '8px 12px', color: '#E2E8F0', fontFamily: 'sans-serif', fontWeight: 500 }}>{c.clientSliceId}</td>
                  <td style={{ padding: '8px 12px' }}>
                    <span style={{
                      padding: '2px 6px',
                      borderRadius: 4,
                      fontSize: '10px',
                      fontWeight: 700,
                      backgroundColor: c.exportFormat === 'PDF' ? 'rgba(239, 68, 68, 0.2)' : 'rgba(16, 185, 129, 0.2)',
                      color: c.exportFormat === 'PDF' ? '#FCA5A5' : '#6EE7B7',
                    }}>
                      {c.exportFormat}
                    </span>
                  </td>
                  <td style={{ padding: '8px 12px', textAlign: 'right', color: '#CBD5E1' }}>{c.pageCount}</td>
                  <td style={{ padding: '8px 12px', textAlign: 'right', color: '#00D4FF' }}>{c.astComplexityScore.toFixed(1)}</td>
                  <td style={{ padding: '8px 12px', textAlign: 'right', color: '#94A3B8' }}>${c.queryCostUsd.toFixed(5)}</td>
                  <td style={{ padding: '8px 12px', textAlign: 'right', color: '#94A3B8' }}>${c.renderCostUsd.toFixed(5)}</td>
                  <td style={{ padding: '8px 12px', textAlign: 'right', color: '#94A3B8' }}>${c.storageCostUsd.toFixed(6)}</td>
                  <td style={{ padding: '8px 12px', textAlign: 'right', color: '#34D399', fontWeight: 700 }}>
                    ${c.totalCostUsd.toFixed(5)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </Box>
      </Box>

    </Box>
  );
};

export default ReportChargebackDashboard;
