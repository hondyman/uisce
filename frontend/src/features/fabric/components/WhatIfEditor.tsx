import React, { useState, useEffect } from 'react';
import { useMutation } from '@apollo/client';
import { Box, Typography, Paper, Button, CircularProgress } from '@mui/material';
import MonacoCodeEditor from '../../components/UnifiedSemanticBuilder/MonacoCodeEditor.lazy';
import ForecastPanel from './ForecastPanel';
import { FORECAST_POLICY_RUN } from '../pages/PolicySimulationPage'; // Re-using the mutation
import RiskMitigationToolbar from './RiskMitigationToolbar';

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
  const debouncedSql = useDebounce(sql, 750); // 750ms debounce

  const [runForecast, { data, loading, error }] = useMutation(FORECAST_POLICY_RUN);

  useEffect(() => {
    if (debouncedSql.trim()) {
      // The forecast action needs `from_ds` and `to_ds`, but they aren't relevant
      // when providing SQL. We pass dummy values.
      runForecast({
        variables: {
          fromDs: 'dummy',
          toDs: 'dummy',
          migrationSql: debouncedSql,
        },
      });
    }
  }, [debouncedSql, runForecast]);

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
      <Button onClick={() => runForecast({ variables: { fromDs: 'dummy', toDs: 'dummy', migrationSql: sql } })} disabled={loading}>
        {loading ? <CircularProgress size={24} /> : 'Re-Forecast Now'}
      </Button>

      <ForecastPanel data={data?.forecast_policy_run} loading={loading} error={error} />
    </Box>
  );
};

export default WhatIfEditor;