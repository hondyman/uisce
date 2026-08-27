import React, { useState } from 'react';
import {
  Box,
  Typography,
  Select,
  MenuItem,
  FormControl,
  Paper,
} from '@mui/material';
import CurrencyExchangeIcon from '@mui/icons-material/CurrencyExchange';

export interface AttributionRow {
  positionId: string;
  securityName: string;
  localCurrency: string;
  startValReport: number;
  endValReport: number;
  totalReturnPct: number;
  assetReturnPct: number;
  currencyReturnPct: number;
  interactionPct: number;
  startFxRate: number;
  endFxRate: number;
}

interface MultiCurrencyAttributionProps {
  baseCurrency?: string;
  reportCurrency?: string;
  onCurrencyChange?: (newCcy: string) => void;
  summary?: {
    portfolioTotalReturn: number;
    portfolioAssetReturn: number;
    portfolioFXReturn: number;
    portfolioInteraction: number;
    totalStartValReport: number;
    totalEndValReport: number;
  };
  holdings?: AttributionRow[];
}

export const MultiCurrencyAttributionView: React.FC<MultiCurrencyAttributionProps> = ({
  baseCurrency = 'USD',
  reportCurrency = 'EUR',
  onCurrencyChange,
  summary = {
    portfolioTotalReturn: 0.3571,
    portfolioAssetReturn: 0.2000,
    portfolioFXReturn: 0.1310,
    portfolioInteraction: 0.0262,
    totalStartValReport: 90000.0,
    totalEndValReport: 122142.86,
  },
  holdings = [
    {
      positionId: 'pos_tyo_01',
      securityName: 'Tokyo Robotics Corp',
      localCurrency: 'JPY',
      startValReport: 90000.0,
      endValReport: 122142.86,
      totalReturnPct: 0.3571,
      assetReturnPct: 0.2000,
      currencyReturnPct: 0.1310,
      interactionPct: 0.0262,
      startFxRate: 0.0060,
      endFxRate: 0.006786,
    },
  ],
}) => {
  const [selectedCurrency, setSelectedCurrency] = useState(reportCurrency);

  const handleCcyChange = (ccy: string) => {
    setSelectedCurrency(ccy);
    if (onCurrencyChange) {
      onCurrencyChange(ccy);
    }
  };

  return (
    <Box sx={{ width: '100%', bgcolor: '#050D1A', color: '#fff', borderRadius: 2, border: '1px solid #1E293B', overflow: 'hidden', fontFamily: 'sans-serif' }}>
      
      {/* Header & Triangulation Controls */}
      <Box sx={{ p: 2, bgcolor: '#071526', borderBottom: '1px solid #1E293B', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <CurrencyExchangeIcon sx={{ color: '#00D4FF', fontSize: 22 }} />
          <Typography variant="subtitle2" fontWeight="700" sx={{ letterSpacing: 0.5 }}>
            Multi-Currency Triangulation & FX Attribution Breakdown
          </Typography>
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <Typography variant="caption" sx={{ color: '#94A3B8' }}>
            Base CCY: <strong style={{ color: '#fff' }}>{baseCurrency}</strong>
          </Typography>
          
          <FormControl size="small">
            <Select
              value={selectedCurrency}
              onChange={(e) => handleCcyChange(e.target.value)}
              sx={{
                bgcolor: '#050D1A',
                color: '#00D4FF',
                fontSize: '12px',
                fontFamily: 'monospace',
                fontWeight: 700,
                height: 28,
                border: '1px solid rgba(0, 212, 255, 0.3)',
                '& .MuiSvgIcon-root': { color: '#00D4FF' },
              }}
            >
              <MenuItem value="USD">Presentation: USD ($)</MenuItem>
              <MenuItem value="EUR">Presentation: EUR (€)</MenuItem>
              <MenuItem value="GBP">Presentation: GBP (£)</MenuItem>
              <MenuItem value="CHF">Presentation: CHF (Fr)</MenuItem>
              <MenuItem value="JPY">Presentation: JPY (¥)</MenuItem>
            </Select>
          </FormControl>
        </Box>
      </Box>

      {/* Attribution Summary KPI Banner */}
      <Box sx={{ p: 2, bgcolor: '#0B1E36', display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 2, borderBottom: '1px solid #1E293B' }}>
        <Paper sx={{ p: 1.5, bgcolor: '#071526', border: '1px solid rgba(255,255,255,0.05)', borderRadius: 1.5 }}>
          <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block' }}>Portfolio Total Return (RT)</Typography>
          <Typography variant="h6" fontWeight="700" sx={{ color: summary.portfolioTotalReturn >= 0 ? '#10B981' : '#EF4444' }}>
            {(summary.portfolioTotalReturn * 100).toFixed(2)}%
          </Typography>
        </Paper>

        <Paper sx={{ p: 1.5, bgcolor: '#071526', border: '1px solid rgba(255,255,255,0.05)', borderRadius: 1.5 }}>
          <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block' }}>Pure Asset Return (RL)</Typography>
          <Typography variant="h6" fontWeight="700" sx={{ color: '#00D4FF' }}>
            {(summary.portfolioAssetReturn * 100).toFixed(2)}%
          </Typography>
        </Paper>

        <Paper sx={{ p: 1.5, bgcolor: '#071526', border: '1px solid rgba(255,255,255,0.05)', borderRadius: 1.5 }}>
          <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block' }}>FX Currency Impact (RC)</Typography>
          <Typography variant="h6" fontWeight="700" sx={{ color: summary.portfolioFXReturn >= 0 ? '#F59E0B' : '#EF4444' }}>
            {(summary.portfolioFXReturn * 100).toFixed(2)}%
          </Typography>
        </Paper>

        <Paper sx={{ p: 1.5, bgcolor: '#071526', border: '1px solid rgba(255,255,255,0.05)', borderRadius: 1.5 }}>
          <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block' }}>Interaction Effect (RL &times; RC)</Typography>
          <Typography variant="h6" fontWeight="700" sx={{ color: '#A855F7' }}>
            {(summary.portfolioInteraction * 100).toFixed(2)}%
          </Typography>
        </Paper>
      </Box>

      {/* Position Breakdown Table */}
      <Box sx={{ overflowX: 'auto' }}>
        <table style={{ width: '100%', textAlign: 'left', borderCollapse: 'collapse', fontSize: '11px', fontFamily: 'monospace' }}>
          <thead style={{ backgroundColor: '#071526', color: '#94A3B8', textTransform: 'uppercase', fontSize: '10px', borderBottom: '1px solid #1E293B' }}>
            <tr>
              <th style={{ padding: '10px 12px' }}>Position</th>
              <th style={{ padding: '10px 12px' }}>LCY</th>
              <th style={{ padding: '10px 12px', textAlign: 'right' }}>Start Val ({selectedCurrency})</th>
              <th style={{ padding: '10px 12px', textAlign: 'right' }}>End Val ({selectedCurrency})</th>
              <th style={{ padding: '10px 12px', textAlign: 'right' }}>Asset Return (RL)</th>
              <th style={{ padding: '10px 12px', textAlign: 'right' }}>FX Return (RC)</th>
              <th style={{ padding: '10px 12px', textAlign: 'right' }}>Interaction</th>
              <th style={{ padding: '10px 12px', textAlign: 'right', color: '#34D399' }}>Total (RT)</th>
            </tr>
          </thead>
          <tbody>
            {holdings.map((h) => (
              <tr key={h.positionId} style={{ borderBottom: '1px solid rgba(30, 41, 59, 0.6)' }}>
                <td style={{ padding: '8px 12px', color: '#E2E8F0', fontFamily: 'sans-serif', fontWeight: 500 }}>{h.securityName}</td>
                <td style={{ padding: '8px 12px' }}>
                  <span style={{ padding: '2px 6px', borderRadius: 4, backgroundColor: '#1E293B', color: '#67E8F9', fontSize: '10px', fontWeight: 700 }}>
                    {h.localCurrency}
                  </span>
                </td>
                <td style={{ padding: '8px 12px', textAlign: 'right', color: '#94A3B8' }}>${h.startValReport.toLocaleString(undefined, { minimumFractionDigits: 2 })}</td>
                <td style={{ padding: '8px 12px', textAlign: 'right', color: '#E2E8F0' }}>${h.endValReport.toLocaleString(undefined, { minimumFractionDigits: 2 })}</td>
                <td style={{ padding: '8px 12px', textAlign: 'right', color: '#00D4FF' }}>{(h.assetReturnPct * 100).toFixed(2)}%</td>
                <td style={{ padding: '8px 12px', textAlign: 'right', color: h.currencyReturnPct >= 0 ? '#FBBF24' : '#FB7185' }}>
                  {(h.currencyReturnPct * 100).toFixed(2)}%
                </td>
                <td style={{ padding: '8px 12px', textAlign: 'right', color: '#C084FC' }}>{(h.interactionPct * 100).toFixed(2)}%</td>
                <td style={{ padding: '8px 12px', textAlign: 'right', fontWeight: 700, color: h.totalReturnPct >= 0 ? '#34D399' : '#FB7185' }}>
                  {(h.totalReturnPct * 100).toFixed(2)}%
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </Box>
    </Box>
  );
};

export default MultiCurrencyAttributionView;
