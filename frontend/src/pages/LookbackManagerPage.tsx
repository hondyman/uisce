import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Grid,
  TextField,
  Button,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Chip,
  Alert,
  CircularProgress
} from '@mui/material';
import HistoryIcon from '@mui/icons-material/History';
import CompareArrowsIcon from '@mui/icons-material/CompareArrows';
import { AICopilotBar } from '../components/designer/AICopilotBar';

export const LookbackManagerPage: React.FC = () => {
  const [boKey, setBoKey] = useState('Account');
  const [timeA, setTimeA] = useState('2025-12-31T00:00');
  const [timeB, setTimeB] = useState('2026-06-30T00:00');
  const [diffResults, setDiffResults] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);

  const handleRunAudit = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/audit/lookback-diff', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          bo_key: boKey,
          timestamp_a: new Date(timeA).toISOString(),
          timestamp_b: new Date(timeB).toISOString(),
        }),
      });
      const data = await res.json();
      setDiffResults(data.differences || []);
    } catch (err) {
      console.error('Failed to run lookback audit comparison', err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box sx={{ p: 4, bgcolor: '#0f172a', minHeight: '100vh', color: '#f8fafc' }}>
      <Box display="flex" alignItems="center" gap={1.5} mb={1}>
        <HistoryIcon sx={{ color: '#38bdf8', fontSize: 32 }} />
        <Typography variant="h4" fontWeight="700">
          Compliance & Lookback Audit Manager
        </Typography>
      </Box>
      <Typography variant="body1" color="#94a3b8" mb={3}>
        Perform side-by-side point-in-time forensic audits across multi-datasource Business Objects.
      </Typography>

      {/* AI Layout Copilot with Lookback Intent Support */}
      <AICopilotBar
        domain="PORTFOLIO"
        onLayoutGenerated={() => handleRunAudit()}
      />

      <Paper sx={{ p: 3, bgcolor: '#1e293b', border: '1px solid #334155', mb: 4 }}>
        <Grid container spacing={3} alignItems="center">
          <Grid size={{ xs: 12, md: 3 }}>
            <TextField
              label="Business Object"
              fullWidth
              size="small"
              value={boKey}
              onChange={(e) => setBoKey(e.target.value)}
              sx={{ bgcolor: '#0f172a', input: { color: '#fff' } }}
            />
          </Grid>
          <Grid size={{ xs: 12, md: 3 }}>
            <TextField
              label="Baseline Timestamp (A)"
              type="datetime-local"
              fullWidth
              size="small"
              value={timeA}
              onChange={(e) => setTimeA(e.target.value)}
              InputLabelProps={{ shrink: true }}
              sx={{ bgcolor: '#0f172a', input: { color: '#fff' } }}
            />
          </Grid>
          <Grid size={{ xs: 12, md: 3 }}>
            <TextField
              label="Comparison Timestamp (B)"
              type="datetime-local"
              fullWidth
              size="small"
              value={timeB}
              onChange={(e) => setTimeB(e.target.value)}
              InputLabelProps={{ shrink: true }}
              sx={{ bgcolor: '#0f172a', input: { color: '#fff' } }}
            />
          </Grid>
          <Grid size={{ xs: 12, md: 3 }}>
            <Button
              variant="contained"
              fullWidth
              startIcon={loading ? <CircularProgress color="inherit" sx={{ fontSize: 20 }}/> : <CompareArrowsIcon />}
              onClick={handleRunAudit}
              disabled={loading}
              sx={{ bgcolor: '#0284c7', '&:hover': { bgcolor: '#0369a1' } }}
            >
              {loading ? 'Computing Diffs...' : 'Generate Audit Matrix'}
            </Button>
          </Grid>
        </Grid>
      </Paper>

      {diffResults.length > 0 && (
        <Paper sx={{ p: 3, bgcolor: '#1e293b', border: '1px solid #334155' }}>
          <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
            <Typography variant="h6" fontWeight="600">
              Audit Diff Matrix ({diffResults.length} changes detected)
            </Typography>
            <Chip label={`Comparing ${timeA.slice(0, 10)} vs ${timeB.slice(0, 10)}`} color="info" size="small" />
          </Box>
          <Table size="small">
            <TableHead sx={{ bgcolor: '#0f172a' }}>
              <TableRow>
                <TableCell sx={{ color: '#94a3b8', fontWeight: 600 }}>Record ID</TableCell>
                <TableCell sx={{ color: '#94a3b8', fontWeight: 600 }}>Field Modified</TableCell>
                <TableCell sx={{ color: '#94a3b8', fontWeight: 600 }}>Value at Timestamp A</TableCell>
                <TableCell sx={{ color: '#94a3b8', fontWeight: 600 }}>Value at Timestamp B</TableCell>
                <TableCell sx={{ color: '#94a3b8', fontWeight: 600 }}>Delta / Shift</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {diffResults.map((row, idx) => (
                <TableRow
                  key={idx}
                  sx={{
                    bgcolor: row.is_significant ? 'rgba(244, 63, 94, 0.1)' : 'transparent',
                  }}
                >
                  <TableCell sx={{ color: '#38bdf8', fontWeight: 600 }}>{row.record_id}</TableCell>
                  <TableCell sx={{ color: '#f8fafc' }}>{row.field_name}</TableCell>
                  <TableCell sx={{ color: '#cbd5e1' }}>{row.value_a}</TableCell>
                  <TableCell sx={{ color: '#4ade80', fontWeight: 600 }}>{row.value_b}</TableCell>
                  <TableCell sx={{ color: row.is_significant ? '#f43f5e' : '#facc15', fontWeight: 600 }}>
                    {row.delta}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Paper>
      )}
    </Box>
  );
};
