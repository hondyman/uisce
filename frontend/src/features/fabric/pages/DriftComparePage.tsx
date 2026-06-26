import React, { useState, useMemo } from 'react';
import { Box, Typography, Paper, Select, MenuItem, CircularProgress, Alert, FormControlLabel, Switch } from '@mui/material';
import { gql, useQuery } from '@apollo/client';
import { format } from 'date-fns';
import DriftCompare from '../components/DriftCompare';
import { DriftReport } from '../components/DriftReportDetail';
import { diffReports, DiffResult } from './diff';

// This query is from the blueprint
const GET_ALL_REPORTS_FOR_SELECT = gql`
  query GetAllReportsForSelect {
    drift_reports(order_by: { generated_at: desc }) {
      id
      generated_at
      schema_hash
    }
  }
`;

const COMPARE_DRIFT_REPORTS = gql`
  query CompareDriftReports($idA: uuid!, $idB: uuid!) {
    reportA: drift_reports_by_pk(id: $idA) {
      id
      generated_at
      schema_hash
      severity_summary
      drift_log_entries {
        id
        severity
        qualified_path
        explanation
      }
    }
    reportB: drift_reports_by_pk(id: $idB) {
      id
      generated_at
      schema_hash
      severity_summary
      drift_log_entries {
        id
        severity
        qualified_path
        explanation
      }
    }
  }
`;

interface ReportOption {
  id: string;
  generated_at: string;
  schema_hash: string;
}

interface CompareData {
  reportA: DriftReport;
  reportB: DriftReport;
}

const DriftComparePage: React.FC = () => {
  const [reportIdA, setReportIdA] = useState<string>('');
  const [reportIdB, setReportIdB] = useState<string>('');
  const [showOnlySeverityChanges, setShowOnlySeverityChanges] = useState(false);

  const { data: optionsData, loading: optionsLoading, error: optionsError } = useQuery<{ drift_reports: ReportOption[] }>(
    GET_ALL_REPORTS_FOR_SELECT
  );

  const { data: compareData, loading: compareLoading, error: compareError } = useQuery<CompareData>(
    COMPARE_DRIFT_REPORTS,
    {
      variables: { idA: reportIdA, idB: reportIdB },
      skip: !reportIdA || !reportIdB,
    }
  );

  const diff: DiffResult | null = useMemo(() => {
    if (compareData?.reportA && compareData?.reportB) {
      const fullDiff = diffReports(
        { id: compareData.reportA.id, schema_hash: compareData.reportA.schema_hash, drift_log_entries: compareData.reportA.drift_log_entries },
        { id: compareData.reportB.id, schema_hash: compareData.reportB.schema_hash, drift_log_entries: compareData.reportB.drift_log_entries }
      );

      if (showOnlySeverityChanges) {
        return { ...fullDiff, changed: fullDiff.changed.filter(c => c.severityChanged) };
      }
      return fullDiff;
    }
    return null;
  }, [compareData, showOnlySeverityChanges]);

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Compare Drift Reports
      </Typography>
      <Paper sx={{ p: 2, mb: 3, display: 'flex', alignItems: 'center', gap: 2 }}>
        <Select value={reportIdA} onChange={(e) => setReportIdA(e.target.value)} displayEmpty size="small" sx={{ minWidth: 250 }}>
          <MenuItem value="" disabled>Select Report A (Old)</MenuItem>
          {optionsLoading && <MenuItem disabled>Loading...</MenuItem>}
          {optionsData?.drift_reports.map((r) => (
            <MenuItem key={r.id} value={r.id}>
              {format(new Date(r.generated_at), 'yy-MM-dd HH:mm')} - {r.schema_hash}
            </MenuItem>
          ))}
        </Select>
        <Typography>vs.</Typography>
        <Select value={reportIdB} onChange={(e) => setReportIdB(e.target.value)} displayEmpty size="small" sx={{ minWidth: 250 }}>
          <MenuItem value="" disabled>Select Report B (New)</MenuItem>
          {optionsLoading && <MenuItem disabled>Loading...</MenuItem>}
          {optionsData?.drift_reports.map((r) => (
            <MenuItem key={r.id} value={r.id}>
              {format(new Date(r.generated_at), 'yy-MM-dd HH:mm')} - {r.schema_hash}
            </MenuItem>
          ))}
        </Select>
      </Paper>
      
      <Paper sx={{ p: 2, mb: 3 }}>
        <FormControlLabel
          control={
            <Switch checked={showOnlySeverityChanges} onChange={(e) => setShowOnlySeverityChanges(e.target.checked)} />
          }
          label="Show only severity changes"
        />
      </Paper>

      {optionsError && <Alert severity="error">Failed to load report options: {optionsError.message}</Alert>}

      {compareLoading && <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}><CircularProgress /></Box>}
      {compareError && <Alert severity="error">Failed to compare reports: {compareError.message}</Alert>}
      {diff && (
        <DriftCompare diff={diff} />
      )}
    </Box>
  );
};

export default DriftComparePage;