import React, { useState, useEffect } from 'react';
import { Box, Typography, Paper, Button, CircularProgress } from '@mui/material';
import MonacoCodeEditor from '../../components/UnifiedSemanticBuilder/MonacoCodeEditor.lazy';
import ForecastPanel from './ForecastPanel';
import RiskMitigationToolbar from './RiskMitigationToolbar';
import { useMutation } from '@tanstack/react-query';
import { apiFetch } from '../../../../lib/apiClient';

interface WhatIfEditorProps {
  initialSql?: string;
}

const useDebounce = (value: string, delay: number) => {
  const [debouncedValue, setDebouncedValue] = useState(value);
  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);
    return () => {
      clearTimeout(handler);
    };
  }, [value, delay]);
  return debouncedValue;
};

const WhatIfEditor: React.FC<WhatIfEditorProps> = ({ initialSql = '' }) => {
  const [sql, setSql] = useState(initialSql);
  const debouncedSql = useDebounce(sql, 750);

  const runForecastMutation = useMutation({
    mutationFn: async ({ fromDs, toDs, migrationSql }: { fromDs: string; toDs: string; migrationSql: string }) => {
      const res = await apiFetch('/api/rest/forecast', {
        method: 'POST',
        body: JSON.stringify({ from_ds: fromDs, to_ds: toDs, migration_sql: migrationSql }),
      });
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json();
    },
  });

  useEffect(() => {
    if (debouncedSql.trim()) {
      runForecastMutation.mutate({
        fromDs: 'dummy',
        toDs: 'dummy',
        migrationSql: debouncedSql,
      });
    }
  }, [debouncedSql]);

  return (
    <Box sx={{ mt: 4 }}>
      <Typography variant="h5" gutterBottom>
        What-If Migration Editor
      </Typography>
      <Typography paragraph color="text.secondary">
        Edit the migration SQL below to see how changes affect the predicted policy impact in real-time.
      </Typography>

      <RiskMitigationToolbar sql={sql} setSql={setSql} />

      <Paper sx={{ height: '250px', border: '1px solid', borderColor: 'divider', mb: 2 }}>
  <div className="editor-wrapper-full editor-h-400">
          <MonacoCodeEditor value={sql} language="json" onChange={(val: string) => setSql(val)} />
        </div>
      </Paper>
      <Button onClick={() => runForecastMutation.mutate({ fromDs: 'dummy', toDs: 'dummy', migrationSql: sql })} disabled={runForecastMutation.isPending}>
        {runForecastMutation.isPending ? <CircularProgress size={24} /> : 'Re-Forecast Now'}
      </Button>

      <ForecastPanel data={runForecastMutation.data?.forecast_policy_run} loading={runForecastMutation.isPending} error={runForecastMutation.error} />
    </Box>
  );
};

export default WhatIfEditor;
