// React default import removed — using automatic JSX runtime
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Chip,
  Tooltip,
  Typography,
  Box,
} from '@mui/material';
import { format } from 'date-fns';
import SeverityBadge from './SeverityBadge';

interface DriftReportSummary {
  id: string;
  generated_at: string;
  schema_hash: string;
  severity_summary: {
    breaking?: number;
    medium?: number;
    low?: number;
  };
  has_severity_change: boolean;
}

interface DriftReportTableProps {
  reports: DriftReportSummary[];
  onSelectReport: (id: string) => void;
}

const DriftReportTable: React.FC<DriftReportTableProps> = ({ reports, onSelectReport }) => {
  return (
    <Table stickyHeader aria-label="drift reports table">
      <TableHead>
        <TableRow>
          <TableCell>Date</TableCell>
          <TableCell>Schema Hash</TableCell>
          <TableCell>Summary</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {reports.length === 0 ? (
          <TableRow>
            <TableCell colSpan={3} align="center">
              <Typography color="text.secondary" sx={{ p: 4 }}>
                No drift reports found.
              </Typography>
            </TableCell>
          </TableRow>
        ) : (
          reports.map((report) => (
            <TableRow
              key={report.id}
              hover
              onClick={() => onSelectReport(report.id)}
              sx={{
                cursor: 'pointer',
                ...(report.has_severity_change && { borderLeft: '4px solid', borderColor: 'warning.main' }),
              }}
            >
              <TableCell>
                <Tooltip title={report.generated_at}>
                  <Typography variant="body2">
                    {format(new Date(report.generated_at), 'yyyy-MM-dd HH:mm:ss')}
                  </Typography>
                </Tooltip>
              </TableCell>
              <TableCell>
                <Chip label={report.schema_hash} variant="outlined" size="small" sx={{ fontFamily: 'monospace' }} />
              </TableCell>
              <TableCell>
                <Box sx={{ display: 'flex', alignItems: 'center' }}>
                  {report.has_severity_change && (
                    <Tooltip title="Severity changed from previous report">
                      <Typography sx={{ mr: 1.5, fontSize: '1.2em' }}>🔥</Typography>
                    </Tooltip>
                  )}
                  <SeverityBadge severity="breaking" count={report.severity_summary?.breaking || 0} />
                  <SeverityBadge severity="medium" count={report.severity_summary?.medium || 0} />
                  <SeverityBadge severity="low" count={report.severity_summary?.low || 0} />
                  {(report.severity_summary?.breaking || 0) + (report.severity_summary?.medium || 0) + (report.severity_summary?.low || 0) === 0 && (
                    <Typography variant="caption" color="text.secondary">No changes</Typography>
                  )}
                </Box>
              </TableCell>
            </TableRow>
          ))
        )}
      </TableBody>
    </Table>
  );
};

export default DriftReportTable;