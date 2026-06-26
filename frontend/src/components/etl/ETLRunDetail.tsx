import { Card, CardContent, Typography, Stack, Box } from '@mui/material';
import { useETLRun } from '../../api/etlRuns';
import { StatusBadge } from '../design/StatusBadge';

export function ETLRunDetail({ id }: { id: string }) {
  const { data, isLoading } = useETLRun(id);

  if (isLoading) return <Typography>Loading…</Typography>;
  if (!data) return <Typography>Not found</Typography>;

  const duration =
    data.completed_at ?
      `${((new Date(data.completed_at).getTime() - new Date(data.started_at).getTime()) / 1000).toFixed(1)}s`
      : 'In progress…';

  return (
    <Card>
      <CardContent>
        <Stack spacing={2}>
          <Typography variant="h5">ETL Run {data.etl_run_id}</Typography>

          <Stack direction="row" spacing={2} alignItems="center">
            <Typography>Status:</Typography>
            <StatusBadge status={data.status} />
          </Stack>

          <Box
            sx={{
              display: 'grid',
              gridTemplateColumns: 'repeat(2, 1fr)',
              gap: 2,
            }}
          >
            <Stack>
              <Typography variant="caption" color="textSecondary">
                Valuation Date
              </Typography>
              <Typography>{data.valuation_date}</Typography>
            </Stack>

            <Stack>
              <Typography variant="caption" color="textSecondary">
                Duration
              </Typography>
              <Typography>{duration}</Typography>
            </Stack>

            <Stack>
              <Typography variant="caption" color="textSecondary">
                Rules Evaluated
              </Typography>
              <Typography>{data.rules_evaluated}</Typography>
            </Stack>

            <Stack>
              <Typography variant="caption" color="textSecondary">
                Scenarios Evaluated
              </Typography>
              <Typography>{data.scenarios_evaluated}</Typography>
            </Stack>

            <Stack>
              <Typography variant="caption" color="textSecondary">
                WASM Version
              </Typography>
              <Typography>{data.wasm_version}</Typography>
            </Stack>

            <Stack>
              <Typography variant="caption" color="textSecondary">
                Orchestrator Version
              </Typography>
              <Typography>{data.orchestrator_version}</Typography>
            </Stack>
          </Box>

          {data.error_summary && (
            <Box>
              <Typography variant="h6" sx={{ mt: 2 }}>
                Errors
              </Typography>
              <pre
                style={{
                  backgroundColor: '#f5f5f5',
                  padding: '12px',
                  borderRadius: '4px',
                  overflow: 'auto',
                  fontSize: '0.875rem',
                }}
              >
                {data.error_summary}
              </pre>
            </Box>
          )}
        </Stack>
      </CardContent>
    </Card>
  );
}
