import React, { Suspense } from 'react';
import {
  Box,
  Paper,
  Typography,
  Alert,
  Tabs,
  Tab,
  TableContainer,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
} from '@mui/material';
import ReactMarkdown from 'react-markdown';
import LazySyntaxHighlighter from '../../../components/LazySyntaxHighlighter';
import SeverityBadge from './SeverityBadge';

interface Violation {
  rule_id: string;
  severity: 'breaking' | 'medium' | 'low';
  message: string;
  qualified_path: string;
}

interface SimulationResult {
  policy_id: string;
  summary: {
    breaking: number;
    medium: number;
    low: number;
  };
  violations: Violation[];
  changelog_md: string;
}

interface SimulationResultDetailProps {
  result: SimulationResult;
}

const SimulationResultDetail: React.FC<SimulationResultDetailProps> = ({ result }) => {
  const [activeTab, setActiveTab] = React.useState(0);
  const totalViolations = result.summary.breaking + result.summary.medium + result.summary.low;
  const decision = result.summary.breaking > 0 ? 'BLOCK' : 'ALLOW';

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  return (
    <Paper sx={{ p: 3, mt: 4, border: '1px solid', borderColor: 'divider' }}>
      <Typography variant="h5" gutterBottom>
        Simulation Result
      </Typography>

      <Alert
        severity={decision === 'BLOCK' ? 'error' : 'success'}
        sx={{ mb: 3, '& .MuiAlert-message': { fontSize: '1.2rem', fontWeight: 'bold' } }}
      >
        Decision: {decision}
      </Alert>

      <Box sx={{ mb: 3 }}>
        <Typography variant="subtitle1" gutterBottom>
          Summary for Policy: <strong>{result.policy_id}</strong>
        </Typography>
        <SeverityBadge severity="breaking" count={result.summary.breaking} />
        <SeverityBadge severity="medium" count={result.summary.medium} />
        <SeverityBadge severity="low" count={result.summary.low} />
      </Box>

      <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Tabs value={activeTab} onChange={handleTabChange} aria-label="simulation result tabs">
          <Tab label={`Violations (${totalViolations})`} />
          <Tab label="Changelog" />
          <Tab label="Raw JSON" />
        </Tabs>
      </Box>

      <Box sx={{ pt: 3 }}>
        {activeTab === 0 && (
          <TableContainer component={Paper}>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Severity</TableCell>
                  <TableCell>Rule ID</TableCell>
                  <TableCell>Message</TableCell>
                  <TableCell>Path</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {result.violations.map((v, i) => (
                  <TableRow key={`${v.rule_id}-${i}`}>
                    <TableCell>
                      <SeverityBadge severity={v.severity} count={1} />
                    </TableCell>
                    <TableCell sx={{ fontFamily: 'monospace' }}>{v.rule_id}</TableCell>
                    <TableCell>{v.message}</TableCell>
                    <TableCell sx={{ fontFamily: 'monospace' }}>{v.qualified_path}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        )}
        {activeTab === 1 && (
          <Box className="markdown-body">
            {result.changelog_md ? (
              <ReactMarkdown>{result.changelog_md}</ReactMarkdown>
            ) : (
              <Typography color="text.secondary">No changelog available.</Typography>
            )}
          </Box>
        )}
        {activeTab === 2 && (
          <Paper variant="outlined" sx={{ maxHeight: '60vh', overflow: 'auto' }}>
            <Suspense fallback={<div>Loading code...</div>}>
              <LazySyntaxHighlighter language="json" showLineNumbers>
                {JSON.stringify(result, null, 2)}
              </LazySyntaxHighlighter>
            </Suspense>
          </Paper>
        )}
      </Box>
    </Paper>
  );
};

export default SimulationResultDetail;