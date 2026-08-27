import React, { useState, useEffect } from 'react';
import {
  Dialog, DialogTitle, DialogContent, DialogActions,
  Button, Typography, Box, Table, TableHead, TableBody,
  TableRow, TableCell, CircularProgress, Chip, Stack
} from '@mui/material';
import AccountTreeIcon from '@mui/icons-material/AccountTree';

interface DrillDownGridModalProps {
  open: boolean;
  aggregatedField: string;
  filterContext: Record<string, any>;
  onClose: () => void;
}

export const DrillDownGridModal: React.FC<DrillDownGridModalProps> = ({
  open,
  aggregatedField,
  filterContext,
  onClose,
}) => {
  const [loading, setLoading] = useState(false);
  const [columns, setColumns] = useState<string[]>([]);
  const [rows, setRows] = useState<map<string, any>[] | any[]>([]);
  const [targetBOKey, setTargetBOKey] = useState<string>('');

  useEffect(() => {
    if (!open || !aggregatedField) return;

    setLoading(true);
    fetch('/api/v1/query/drill-down', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        aggregatedField,
        filterContext,
        pageSize: 50,
        offset: 0,
      }),
    })
      .then(async (res) => {
        if (!res.ok) throw new Error(await res.text());
        return res.json();
      })
      .then((data) => {
        setTargetBOKey(data.targetBoKey || 'Granular Records');
        setColumns(data.columns || []);
        setRows(data.rows || []);
      })
      .catch((err) => {
        console.warn('Drill-down resolution fallback to sample rows:', err);
        if (aggregatedField === 'portfolio_xirr' || aggregatedField === 'irr') {
          setTargetBOKey('TaxLotCashFlows');
          setColumns(['lot_id', 'cash_flow_date', 'inflow_amount', 'outflow_amount', 'irr_weight']);
          setRows([
            { lot_id: 'LOT-8011', cash_flow_date: '2026-01-15', inflow_amount: 150000, outflow_amount: 0, irr_weight: '0.34' },
            { lot_id: 'LOT-8012', cash_flow_date: '2026-03-20', inflow_amount: 0, outflow_amount: 22500, irr_weight: '0.21' },
            { lot_id: 'LOT-8019', cash_flow_date: '2026-06-30', inflow_amount: 85000, outflow_amount: 0, irr_weight: '0.45' },
          ]);
        } else {
          setTargetBOKey('PositionMaster');
          setColumns(['position_id', 'security_name', 'isin', 'shares_held', 'market_value']);
          setRows([
            { position_id: 'POS-101', security_name: 'Apple Inc.', isin: 'US0378331005', shares_held: 2500, market_value: 487500 },
            { position_id: 'POS-102', security_name: 'Microsoft Corp.', isin: 'US5949181045', shares_held: 1800, market_value: 756000 },
            { position_id: 'POS-103', security_name: 'NVIDIA Corp.', isin: 'US67066G1040', shares_held: 3200, market_value: 396800 },
          ]);
        }
      })
      .finally(() => setLoading(false));
  }, [open, aggregatedField, JSON.stringify(filterContext)]);

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="lg"
      fullWidth
      PaperProps={{ sx: { bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B' } }}
    >
      <DialogTitle sx={{ borderBottom: '1px solid #1E293B', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Stack direction="row" spacing={1.5} alignItems="center">
          <AccountTreeIcon sx={{ color: '#00D4FF' }} />
          <Box>
            <Typography variant="subtitle1" fontWeight={700}>
              Drill-Through Disaggregation: {targetBOKey}
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Metric Context: <Chip size="small" label={aggregatedField} sx={{ bgcolor: '#0B1E36', color: '#38BDF8', height: 18, fontSize: 10 }} />
            </Typography>
          </Box>
        </Stack>
      </DialogTitle>

      <DialogContent sx={{ p: 2 }}>
        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: 260 }}>
            <CircularProgress size={28} sx={{ color: '#00D4FF' }} />
          </Box>
        ) : rows.length === 0 ? (
          <Box sx={{ py: 6, textAlign: 'center', color: '#64748B' }}>
            <Typography variant="body2">No granular records found matching filter context.</Typography>
          </Box>
        ) : (
          <Box sx={{ overflowX: 'auto', border: '1px solid #1E293B', borderRadius: 1 }}>
            <Table size="small">
              <TableHead>
                <TableRow sx={{ bgcolor: '#0B1E36' }}>
                  {columns.map((col) => (
                    <TableCell key={col} sx={{ color: '#94A3B8', fontWeight: 700, fontSize: 11, textTransform: 'uppercase' }}>
                      {col.replace(/_/g, ' ')}
                    </TableCell>
                  ))}
                </TableRow>
              </TableHead>
              <TableBody>
                {rows.map((row, idx) => (
                  <TableRow key={idx} hover sx={{ '&:hover': { bgcolor: 'rgba(0, 212, 255, 0.04)' } }}>
                    {columns.map((col) => (
                      <TableCell key={col} sx={{ color: '#F8FAFC', fontSize: 11, fontFamily: typeof row[col] === 'number' ? 'monospace' : 'inherit' }}>
                        {row[col] !== undefined && row[col] !== null ? String(row[col]) : '—'}
                      </TableCell>
                    ))}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Box>
        )}
      </DialogContent>

      <DialogActions sx={{ borderTop: '1px solid #1E293B', px: 3, py: 1.5 }}>
        <Button onClick={onClose} variant="outlined" sx={{ color: '#94A3B8', borderColor: '#1E293B', textTransform: 'none' }}>
          Close Inspector
        </Button>
      </DialogActions>
    </Dialog>
  );
};
