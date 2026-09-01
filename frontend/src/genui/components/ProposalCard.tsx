import React from "react";
import { useTheme, Box, Typography, Table, TableBody, TableCell, TableContainer, TableHead, TableRow } from "@mui/material";
import { RebalanceProposal } from "../schema";
import ActionButton from "../../components/ui/ActionButton";

type Props = {
  data: RebalanceProposal;
  onApprove: () => void;
  onReject: () => void;
  onClarify: () => void;
};

export function ProposalCard({ data, onApprove, onReject, onClarify }: Props) {
  const theme = useTheme();
  const { advisor_view, orders, citations } = data;

  const isDark = theme.palette.mode === 'dark';

  return (
    <Box sx={{
      bgcolor: isDark ? 'grey.800' : 'background.paper',
      borderRadius: 2,
      boxShadow: 2,
      p: 3,
      border: 1,
      borderColor: isDark ? 'grey.700' : 'grey.200',
    }}>
      <header sx={{ mb: 3 }}>
        <Typography variant="h6" sx={{ mb: 1, fontWeight: 700, color: isDark ? 'grey.100' : 'grey.900' }}>
          {advisor_view.title}
        </Typography>
        <Typography sx={{ color: isDark ? 'grey.300' : 'grey.600' }}>
          {advisor_view.summary}
        </Typography>
      </header>

      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 2, mb: 2 }}>
        <Box sx={{ bgcolor: isDark ? 'grey.900' : 'grey.50', p: 1.5, borderRadius: 1 }}>
          <Typography variant="caption" sx={{ display: 'block', color: 'grey.500' }}>Tracking Error</Typography>
          <Typography sx={{ fontFamily: 'monospace', fontWeight: 500 }}>
            {advisor_view.tracking_error_before.toFixed(2)}% → {advisor_view.tracking_error_after.toFixed(2)}%
          </Typography>
        </Box>
        <Box sx={{ bgcolor: isDark ? 'grey.900' : 'grey.50', p: 1.5, borderRadius: 1 }}>
          <Typography variant="caption" sx={{ display: 'block', color: 'grey.500' }}>Est. Tax Impact</Typography>
          <Typography 
            sx={{ 
              fontFamily: 'monospace', 
              fontWeight: 500,
              color: advisor_view.tax_impact_usd < 0 ? 'success.main' : 'error.main'
            }}
          >
            {advisor_view.tax_impact_usd < 0 ? '+' : ''}{(-advisor_view.tax_impact_usd).toLocaleString('en-US', { style: 'currency', currency: 'USD' })}
          </Typography>
        </Box>
      </Box>

      {advisor_view.monte_carlo && (
        <Box sx={{ mb: 3, bgcolor: isDark ? 'rgba(33, 150, 243, 0.2)' : 'rgba(33, 150, 243, 0.1)', p: 2, borderRadius: 1, border: 1, borderColor: isDark ? 'info.dark' : 'info.light' }}>
          <Typography variant="subtitle2" sx={{ mb: 1, color: isDark ? 'info.light' : 'info.dark', fontWeight: 600 }}>
            Tax Impact Confidence (Monte Carlo)
          </Typography>
          <Box sx={{ fontSize: '0.875rem', color: isDark ? 'info.main' : 'grey.800' }}>
            <Typography variant="body2">
              Median Benefit: <Typography component="span" sx={{ fontFamily: 'monospace', fontWeight: 700 }}>
                {(-advisor_view.monte_carlo.median).toLocaleString('en-US', { style: 'currency', currency: 'USD' })}
              </Typography>
            </Typography>
            <Typography variant="caption" sx={{ opacity: 0.8, mt: 0.5, display: 'block' }}>
              80% Confidence Range: {(-advisor_view.monte_carlo.confidence80_min).toLocaleString('en-US', { style: 'currency', currency: 'USD' })} – {(-advisor_view.monte_carlo.confidence80_max).toLocaleString('en-US', { style: 'currency', currency: 'USD' })}
            </Typography>
          </Box>
        </Box>
      )}

      <Box sx={{ mb: 3 }}>
        <Typography variant="subtitle2" sx={{ mb: 1, color: 'grey.500', textTransform: 'uppercase', fontSize: '0.75rem', letterSpacing: '0.05em', fontWeight: 600 }}>
          Proposed Orders
        </Typography>
        <TableContainer>
          <Table size="small">
            <TableHead>
              <TableRow sx={{ bgcolor: isDark ? 'grey.800' : 'grey.50' }}>
                <TableCell sx={{ fontWeight: 600, fontSize: '0.75rem', color: 'grey.500', textTransform: 'uppercase' }}>Side</TableCell>
                <TableCell sx={{ fontWeight: 600, fontSize: '0.75rem', color: 'grey.500', textTransform: 'uppercase' }}>Symbol</TableCell>
                <TableCell align="right" sx={{ fontWeight: 600, fontSize: '0.75rem', color: 'grey.500', textTransform: 'uppercase' }}>Qty</TableCell>
                <TableCell align="right" sx={{ fontWeight: 600, fontSize: '0.75rem', color: 'grey.500', textTransform: 'uppercase' }}>Est. Value</TableCell>
                <TableCell sx={{ fontWeight: 600, fontSize: '0.75rem', color: 'grey.500', textTransform: 'uppercase' }}>Reason</TableCell>
              </TableRow>
            </TableHead>
            <TableBody sx={{ bgcolor: isDark ? 'grey.800' : 'background.paper' }}>
              {orders.map((o, i) => (
                <TableRow key={i}>
                  <TableCell sx={{ fontWeight: 500, color: o.side === 'BUY' ? 'success.main' : 'error.main', whiteSpace: 'nowrap', fontSize: '0.875rem' }}>
                    {o.side}
                  </TableCell>
                  <TableCell sx={{ whiteSpace: 'nowrap', fontSize: '0.875rem', color: isDark ? 'grey.100' : 'grey.900' }}>{o.symbol}</TableCell>
                  <TableCell align="right" sx={{ color: 'grey.500', fontSize: '0.875rem' }}>{o.qty}</TableCell>
                  <TableCell align="right" sx={{ color: 'grey.500', fontSize: '0.875rem' }}>{formatUSD(o.est_value_usd)}</TableCell>
                  <TableCell sx={{ color: 'grey.500', fontSize: '0.875rem' }}>{o.reason}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </Box>

      {orders.some(o => o.lots?.length > 0) && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="subtitle2" sx={{ mb: 1, color: 'grey.500', textTransform: 'uppercase', fontSize: '0.75rem', letterSpacing: '0.05em', fontWeight: 600 }}>
            Tax Lots Utilized
          </Typography>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            {orders.filter(o => o.lots?.length > 0).map((o, i) => (
              <Box key={i} sx={{ fontSize: '0.875rem' }}>
                <Typography component="span" sx={{ fontWeight: 500, color: isDark ? 'grey.300' : 'grey.700' }}>{o.symbol}:</Typography>
                <ul style={{ margin: 0, paddingLeft: '1rem' }}>
                  {o.lots!.map(l => (
                    <li key={l.lot_id} style={{ color: isDark ? 'grey.400' : 'grey.600', fontSize: '0.875rem' }}>
                      ID: {l.lot_id} • {l.term} term • UPNL: <span style={{ color: l.unrealized_pnl >= 0 ? theme.palette.success.main : theme.palette.error.main }}>{formatUSD(l.unrealized_pnl)}</span>
                    </li>
                  ))}
                </ul>
              </Box>
            ))}
          </Box>
        </Box>
      )}

      <Box sx={{ mb: 3, bgcolor: isDark ? 'rgba(33, 150, 243, 0.2)' : 'rgba(33, 150, 243, 0.1)', p: 2, borderRadius: 1, border: 1, borderColor: isDark ? 'info.dark' : 'info.light' }}>
        <Typography variant="subtitle2" sx={{ mb: 1, color: isDark ? 'info.light' : 'info.dark', textTransform: 'uppercase', fontSize: '0.75rem', letterSpacing: '0.05em', fontWeight: 600 }}>
          Evidence & Citations
        </Typography>
        <Box component="ul" sx={{ listStyle: 'none', m: 0, p: 0, display: 'flex', flexDirection: 'column', gap: 0.5 }}>
          {citations.map(c => (
            <Box component="li" key={c.id} sx={{ fontSize: '0.875rem', color: isDark ? 'info.light' : 'info.dark' }}>
              <Typography 
                component="span" 
                sx={{ 
                  fontFamily: 'monospace', 
                  fontSize: '0.75rem', 
                  bgcolor: isDark ? 'info.dark' : 'info.light', 
                  px: 0.5, 
                  borderRadius: 0.5, 
                  mr: 1 
                }}
              >
                {c.id}
              </Typography>
              <Typography component="span" sx={{ fontWeight: 500 }}>{c.source}</Typography>
              <Typography component="span" sx={{ mx: 0.5, color: isDark ? 'info.main' : 'info.dark' }}>•</Typography>
              <Typography component="span" sx={{ fontSize: '0.75rem', color: 'grey.500' }}>snap {c.snapshot_id}</Typography>
              <Typography sx={{ pl: 4, color: isDark ? 'grey.300' : 'grey.600', fontStyle: 'italic' }}>"{c.excerpt}"</Typography>
            </Box>
          ))}
        </Box>
      </Box>

      {advisor_view.disclosures && advisor_view.disclosures.length > 0 && (
        <Box sx={{ mb: 4 }}>
          <Typography variant="caption" sx={{ display: 'block', color: 'grey.400', textTransform: 'uppercase', letterSpacing: '0.05em', fontWeight: 600, mb: 0.5 }}>
            Disclosures
          </Typography>
          <Box component="ul" sx={{ listStyle: 'disc', pl: 2, fontSize: '0.75rem', color: 'grey.400', m: 0 }}>
            {advisor_view.disclosures.map((d, i) => <li key={i}>{d}</li>)}
          </Box>
        </Box>
      )}

      <Box sx={{ display: 'flex', gap: 1.5, pt: 2, borderTop: 1, borderColor: isDark ? 'grey.700' : 'grey.200' }}>
        <ActionButton variant="success" onClick={onApprove}>{data.actions.approve.label}</ActionButton>
        <ActionButton variant="danger" onClick={onReject}>{data.actions.reject.label}</ActionButton>
        <ActionButton variant="secondary" onClick={onClarify}>{data.actions.clarify.label}</ActionButton>
      </Box>
    </Box>
  );
}

function formatUSD(n: number) {
  const sign = n < 0 ? "-" : "";
  return `${sign}$${Math.abs(n).toLocaleString(undefined, { maximumFractionDigits: 0 })}`;
}
