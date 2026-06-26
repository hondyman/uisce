import { useState, Fragment, Suspense } from 'react';
import {
  Alert,
  Box,
  Paper,
  Tabs,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
  Collapse,
  CircularProgress,
  IconButton,
  Tooltip,
} from '@mui/material';
import { format } from 'date-fns';
import ReactMarkdown from 'react-markdown';
import LazySyntaxHighlighter from '../../../components/LazySyntaxHighlighter';
import { gql, useLazyQuery } from '@apollo/client';
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import KeyboardArrowUpIcon from '@mui/icons-material/KeyboardArrowUp';
import ReplayIcon from '@mui/icons-material/Replay';
import SeverityBadge from './SeverityBadge';
import SchemaHashTag from './SchemaHashTag';

const RE_EVALUATE_VIOLATION_QUERY = gql`
  query ReEvaluateViolation($violationId: uuid!) {
    ReEvaluateViolation(violation_id: $violationId) {
      rule_id
      severity
      message
      explain {
        selector
        path
        value
      }
    }
  }
`;

// Based on the blueprint
export interface DriftLogEntry {
  id: string;
  report_id: string;
  severity: 'breaking' | 'medium' | 'low';
  qualified_path: string;
  explanation: string;
  explain?: {
    selector: string;
    path: string;
    value: any;
  }[];
}

export interface DriftReport {
  id: string;
  generated_at: string; // ISO timestamp
  schema_hash: string;
  severity_summary: {
    breaking?: number;
    medium?: number;
    low?: number;
  };
  changelog_md?: string | null;
  raw_report: Record<string, unknown>;
  drift_log_entries: DriftLogEntry[];
}

interface DriftReportDetailProps {
  report: DriftReport;
}

const DriftReportDetail: React.FC<DriftReportDetailProps> = ({ report }) => {
  const [activeTab, setActiveTab] = useState(0);

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  const EntryRow: React.FC<{ entry: DriftLogEntry }> = ({ entry }) => {
  const [open, setOpen] = useState(false);
    const hasExplain = entry.explain && entry.explain.length > 0;

    const [runExplain, { loading: explainLoading, data: explainData, error: explainError }] = useLazyQuery(
      RE_EVALUATE_VIOLATION_QUERY,
      { variables: { violationId: entry.id } }
    );

    const currentExplain = explainData?.ReEvaluateViolation?.explain || entry.explain;

    return (
      <Fragment>
        <TableRow sx={{ '& > *': { borderBottom: 'unset' } }}>
          <TableCell sx={{ width: '50px' }}>
            {hasExplain && (
              <IconButton aria-label="expand row" size="small" onClick={() => setOpen(!open)}>
                {open ? <KeyboardArrowUpIcon /> : <KeyboardArrowDownIcon />}
              </IconButton>
            )}
          </TableCell>
          <TableCell>
            <SeverityBadge severity={entry.severity} count={1} />
          </TableCell>
          <TableCell sx={{ fontFamily: 'monospace' }}>{entry.qualified_path}</TableCell>
          <TableCell>{entry.explanation}</TableCell>
        </TableRow>
        {hasExplain && (
          <TableRow>
            <TableCell className="no-vertical-padding-cell" colSpan={4}>
              <Collapse in={open} timeout="auto" unmountOnExit>
                <Box sx={{ m: 2, p: 2, backgroundColor: 'action.hover', borderRadius: 1, position: 'relative' }}>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
                    <Typography variant="subtitle2" gutterBottom component="div">
                      Match Details
                    </Typography>
                    <Tooltip title="Re-run explanation against the latest policy">
                      <IconButton onClick={() => runExplain()} size="small" disabled={explainLoading}>
                        {explainLoading ? <CircularProgress size={20} /> : <ReplayIcon fontSize="small" />}
                      </IconButton>
                    </Tooltip>
                  </Box>

                  {explainError && (
                    <Alert severity="error" sx={{ mb: 1, fontSize: '0.8rem' }}>
                      Failed to re-run: {explainError.message}
                    </Alert>
                  )}

                  <Box component="ul" sx={{ pl: 2, m: 0, listStyleType: 'disc' }}>
                    {currentExplain.map((d: any, i: number) => (
                      <Box component="li" key={i} sx={{ mb: 1, fontSize: '0.875rem' }}>
                        Matched selector <Box component="code" sx={{ backgroundColor: 'background.default', p: 0.5, borderRadius: 1 }}>{d.selector}</Box> at <Box component="code" sx={{ backgroundColor: 'background.default', p: 0.5, borderRadius: 1 }}>{d.path}</Box> = <Box component="code" sx={{ backgroundColor: 'background.default', p: 0.5, borderRadius: 1 }}>{d.value}</Box>
                      </Box>
                    ))}
                  </Box>
                </Box>
              </Collapse>
            </TableCell>
          </TableRow>
        )}
  </Fragment>
    );
  };

  return (
    <Paper sx={{ p: 3, mt: 2 }}>
      {/* ReportMeta */}
      <Box sx={{ mb: 3 }}>
        <Typography variant="h5" gutterBottom>
          Report {report.id.substring(0, 8)}...
        </Typography>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flexWrap: 'wrap' }}>
          <Typography variant="caption" color="text.secondary">
            Generated: {format(new Date(report.generated_at), 'yyyy-MM-dd HH:mm:ss')}
          </Typography>
          <SchemaHashTag hash={report.schema_hash} />
        </Box>
        <Box sx={{ mt: 2 }}>
          <SeverityBadge severity="breaking" count={report.severity_summary?.breaking} />
          <SeverityBadge severity="medium" count={report.severity_summary?.medium} />
          <SeverityBadge severity="low" count={report.severity_summary?.low} />
        </Box>
      </Box>

      {/* TabNav */}
      <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Tabs value={activeTab} onChange={handleTabChange} aria-label="report detail tabs">
          <Tab label="Changelog" />
          <Tab label="Entries" />
          <Tab label="Raw JSON" />
        </Tabs>
      </Box>

      {/* TabContent */}
      <Box sx={{ pt: 3 }}>
        {activeTab === 0 && (
          <Box className="markdown-body">
            {report.changelog_md ? (
              <ReactMarkdown>{report.changelog_md}</ReactMarkdown>
            ) : (
              <Typography color="text.secondary">No changelog available for this report.</Typography>
            )}
          </Box>
        )}
        {activeTab === 1 && (
          <TableContainer component={Paper}>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell />
                  <TableCell>Severity</TableCell>
                  <TableCell>Path</TableCell>
                  <TableCell>Explanation</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {report.drift_log_entries.map((entry) => (
                  <EntryRow key={entry.id} entry={entry} />
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        )}
        {activeTab === 2 && (
          <Paper variant="outlined" sx={{ maxHeight: '60vh', overflow: 'auto' }}>
            <Suspense fallback={<div>Loading code...</div>}>
              <LazySyntaxHighlighter language="json" showLineNumbers>
                {JSON.stringify(report.raw_report, null, 2)}
              </LazySyntaxHighlighter>
            </Suspense>
          </Paper>
        )}
      </Box>
    </Paper>
  );
};

export default DriftReportDetail;