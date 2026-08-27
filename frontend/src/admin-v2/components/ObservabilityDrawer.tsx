import React, { useEffect, useState } from 'react';
import { useTheme } from '@mui/material/styles';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Paper from '@mui/material/Paper';
import IconButton from '@mui/material/IconButton';
import ReactECharts from 'echarts-for-react';

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  traceId?: string;
}

export function ObservabilityDrawer({ open, onOpenChange, traceId }: Props) {
  const theme = useTheme();
  const [status, setStatus] = useState<'idle' | 'loading' | 'ready' | 'error' | 'no-backend'>('idle');
  const [steps, setSteps] = useState<Array<{ name: string; duration_ms: number; inputs?: unknown; outputs?: unknown }>>(
    []
  );
  const [summary, setSummary] = useState<{ total_ms?: number; tools?: number } | null>(null);

  useEffect(() => {
    if (!open || !traceId) return;

    setStatus('loading');
    let cancelled = false;
    (async () => {
      try {
        const mod = await import('../api/chatHistoryApi');
        const resp = await mod.chatHistoryApi.getTrace(traceId);
        if (cancelled) return;
        const spans = Array.isArray(resp?.spans) ? resp.spans : [];
        const flattened = flattenSpans(spans);
        setSteps(flattened);
        setSummary({
          total_ms: flattened.reduce((acc, s) => acc + s.duration_ms, 0),
          tools: flattened.length,
        });
        setStatus('ready');
      } catch (err) {
        if (cancelled) return;
        setStatus('error');
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [open, traceId]);

  if (!open) return null;

  const isDark = theme.palette.mode === 'dark';

  return (
    <Box
      sx={{
        position: 'fixed',
        inset: 0,
        zIndex: 50,
        bgcolor: 'rgba(0, 0, 0, 0.4)',
        display: 'flex',
        justifyContent: 'flex-end',
      }}
    >
      <Paper
        sx={{
          width: '100%',
          maxWidth: '42rem',
          height: '100%',
          overflowY: 'auto',
          boxShadow: theme.shadows[24],
          p: 3,
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 600, color: isDark ? '#fff' : '#111' }}>
              Observability
            </Typography>
            {traceId && (
              <Typography variant="caption" sx={{ fontFamily: 'monospace', mt: 0.5, display: 'block', color: isDark ? '#9ca3af' : '#6b7280' }}>
                trace_id: {traceId}
              </Typography>
            )}
          </Box>
          <IconButton
            onClick={() => onOpenChange(false)}
            sx={{ color: isDark ? '#9ca3af' : '#9ca3af', '&:hover': { color: isDark ? '#fff' : '#374151' } }}
            aria-label="Close"
          >
            <Typography sx={{ fontSize: '1.5rem', lineHeight: 1 }}>&times;</Typography>
          </IconButton>
        </Box>

        {status === 'loading' && (
          <Typography variant="body2" sx={{ color: isDark ? '#9ca3af' : '#6b7280' }}>
            Loading trace from Tempo…
          </Typography>
        )}
        {status === 'no-backend' && (
          <Typography variant="body2" sx={{ color: isDark ? '#9ca3af' : '#6b7280' }}>
            Trace backend not configured.
          </Typography>
        )}
        {status === 'error' && (
          <Typography variant="body2" sx={{ color: '#dc2626' }}>
            Failed to fetch trace. Observability backend may be offline; falling back to cached trace steps when
            available.
          </Typography>
        )}

        {status === 'ready' && (
          <>
            <Box sx={{ mb: 3, display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 2 }}>
              <Paper variant="outlined" sx={{ p: 1.5, borderRadius: 1 }}>
                <Typography variant="caption" sx={{ textTransform: 'uppercase', letterSpacing: '0.05em', color: isDark ? '#9ca3af' : '#6b7280' }}>
                  Total Duration
                </Typography>
                <Typography variant="h5" sx={{ fontWeight: 600, color: isDark ? '#fff' : '#111' }}>
                  {summary?.total_ms ?? 0} ms
                </Typography>
              </Paper>
              <Paper variant="outlined" sx={{ p: 1.5, borderRadius: 1 }}>
                <Typography variant="caption" sx={{ textTransform: 'uppercase', letterSpacing: '0.05em', color: isDark ? '#9ca3af' : '#6b7280' }}>
                  Spans
                </Typography>
                <Typography variant="h5" sx={{ fontWeight: 600, color: isDark ? '#fff' : '#111' }}>
                  {summary?.tools ?? 0}
                </Typography>
              </Paper>
            </Box>

            <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1, color: isDark ? '#d1d5db' : '#374151' }}>
              Execution Plan
            </Typography>
            <Box component="ol" sx={{ listStyle: 'none', pl: 0, '& li': { mb: 1 } }}>
              {steps.map((step, i) => (
                <Box
                  key={`${step.name}-${i}`}
                  component="li"
                  sx={{
                    border: 1,
                    borderColor: isDark ? '#374151' : '#e5e7eb',
                    borderRadius: 1,
                    p: 1.5,
                    bgcolor: isDark ? '#1f2937' : '#f9fafb',
                  }}
                >
                  <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <Typography variant="body2" sx={{ fontWeight: 500, color: isDark ? '#fff' : '#111' }}>
                      {step.name}
                    </Typography>
                    <Typography variant="caption" sx={{ fontFamily: 'monospace', color: isDark ? '#9ca3af' : '#6b7280' }}>
                      {step.duration_ms} ms
                    </Typography>
                  </Box>
                  <details sx={{ mt: 1 }}>
                    <Typography
                      component="summary"
                      variant="caption"
                      sx={{ color: '#2563eb', cursor: 'pointer', '&:hover': { textDecoration: 'underline' } }}
                    >
                      Show I/O
                    </Typography>
                    <Box
                      component="pre"
                      sx={{
                        mt: 1,
                        p: 1,
                        fontSize: '0.75rem',
                        bgcolor: isDark ? '#fff' : '#fff',
                        border: 1,
                        borderColor: isDark ? '#374151' : '#e5e7eb',
                        borderRadius: 0.5,
                        overflowX: 'auto',
                      }}
                    >
                      {JSON.stringify({ inputs: step.inputs, outputs: step.outputs }, null, 2)}
                    </Box>
                  </details>
                </Box>
              ))}
              {steps.length === 0 && <Typography variant="body2" sx={{ color: isDark ? '#9ca3af' : '#6b7280' }}>No spans recorded.</Typography>}
            </Box>

            {steps.length > 0 && (
              <Box sx={{ mt: 3 }}>
                <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1, color: isDark ? '#d1d5db' : '#374151' }}>
                  Step Timeline
                </Typography>
                <ReactECharts
                  option={{
                    tooltip: { trigger: 'axis' },
                    xAxis: { type: 'category', data: steps.map((s) => s.name) },
                    yAxis: { type: 'value', name: 'ms' },
                    series: [{ type: 'bar', data: steps.map((s) => s.duration_ms), itemStyle: { color: '#0284c7' } }],
                  }}
                  style={{ height: 240, width: '100%' }}
                  notMerge
                />
              </Box>
            )}
          </>
        )}
      </Paper>
    </Box>
  );
}

function flattenSpans(spans: unknown[]): Array<{ name: string; duration_ms: number; inputs?: unknown; outputs?: unknown }> {
  const out: Array<{ name: string; duration_ms: number; inputs?: unknown; outputs?: unknown }> = [];
  for (const raw of spans) {
    const s = raw as { operationName?: string; name?: string; durationMS?: number; duration_ms?: number; tags?: Record<string, unknown>; processID?: string };
    const name = s.operationName || s.name || 'span';
    const duration_ms = s.durationMS ?? s.duration_ms ?? 0;
    const inputs = s.tags?.inputs ?? s.tags?.sql ?? s.tags?.prompt;
    const outputs = s.tags?.outputs ?? s.tags?.rows ?? s.tags?.chart_spec;
    out.push({ name, duration_ms, inputs, outputs });
  }
  return out;
}
