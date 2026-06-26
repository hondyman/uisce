import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  Tabs,
  Tab,
  Box,
  Typography,
  Chip,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
  Button,
  IconButton,
  Paper,
  Tooltip,
  Alert,
  AlertTitle,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import RefreshIcon from '@mui/icons-material/Refresh';
import AccountTreeIcon from '@mui/icons-material/AccountTree';
import WarningIcon from '@mui/icons-material/Warning';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';

interface ExplainStep {
  step: number;
  action: string;
  description: string;
  details?: Record<string, unknown>;
}

interface ExplainRow {
  key: Record<string, unknown>;
  fields: Record<string, unknown>;
  included: boolean;
}

interface Anomaly {
  type: string;
  severity: 'error' | 'warning' | 'info';
  message: string;
  suggestedAction?: string;
}

interface HoldingExplainData {
  header: {
    termId: string;
    entityType: string;
    entityId: string;
    value: number | string;
    evaluatedAt: string;
    evaluatorVersion: string;
  };
  summary: {
    humanReadable: string;
    tieBreakerApplied?: string;
    precedenceOrder?: string[];
    selectedType?: string;
  };
  lineage: {
    semanticTerm: string;
    dependencies: string[];
    physicalColumns: string[];
  };
  rows: ExplainRow[];
  evaluationPath: ExplainStep[];
  sql: {
    text: string;
    mode: 'row' | 'preagg';
  };
  anomalies: Anomaly[];
}

interface HoldingExplainModalProps {
  open: boolean;
  onClose: () => void;
  data: HoldingExplainData | null;
  onRecompute?: () => void;
  onOpenTerm?: (termId: string) => void;
  onStartBP?: (anomalyType: string) => void;
}

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel({ children, value, index }: TabPanelProps) {
  return (
    <div hidden={value !== index} style={{ padding: '16px 0' }}>
      {value === index && children}
    </div>
  );
}

export const HoldingExplainModal: React.FC<HoldingExplainModalProps> = ({
  open,
  onClose,
  data,
  onRecompute,
  onOpenTerm,
  onStartBP,
}) => {
  const [tabValue, setTabValue] = useState(0);

  const handleCopySQL = () => {
    if (data?.sql.text) {
      navigator.clipboard.writeText(data.sql.text);
    }
  };

  if (!data) return null;

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'error':
        return 'error';
      case 'warning':
        return 'warning';
      default:
        return 'info';
    }
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="lg" fullWidth>
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <Typography variant="h6">
            {data.header.termId}
          </Typography>
          <Chip
            label={`Entity: ${data.header.entityId}`}
            size="small"
            color="primary"
            variant="outlined"
          />
          <Chip
            label={typeof data.header.value === 'number' 
              ? `$${data.header.value.toLocaleString()}` 
              : data.header.value}
            size="small"
            color="success"
          />
        </Box>
        <IconButton onClick={onClose}>
          <CloseIcon />
        </IconButton>
      </DialogTitle>

      <DialogContent>
        {/* Summary Banner */}
        <Paper sx={{ p: 2, mb: 2, bgcolor: 'primary.50' }}>
          <Typography variant="body1" fontWeight="bold">
            {data.summary.humanReadable}
          </Typography>
          {data.summary.tieBreakerApplied && (
            <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
              Tie-breaker applied: <strong>{data.summary.tieBreakerApplied}</strong>
              {data.summary.precedenceOrder && (
                <> ({data.summary.precedenceOrder.join(' → ')})</>
              )}
              {data.summary.selectedType && (
                <> → Selected: <Chip label={data.summary.selectedType} size="small" /></>
              )}
            </Typography>
          )}
          <Typography variant="caption" color="text.secondary" display="block" sx={{ mt: 1 }}>
            Evaluated: {data.header.evaluatedAt} | Version: {data.header.evaluatorVersion}
          </Typography>
        </Paper>

        {/* Tabs */}
        <Tabs value={tabValue} onChange={(_, v) => setTabValue(v)}>
          <Tab label="Overview" />
          <Tab label="Lineage" />
          <Tab label={`Rows (${data.rows.length})`} />
          <Tab label="Evaluation" />
          <Tab label="SQL" />
          {data.anomalies.length > 0 && (
            <Tab 
              label={
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                  Anomalies
                  <Chip label={data.anomalies.length} size="small" color="error" />
                </Box>
              } 
            />
          )}
        </Tabs>

        {/* Overview Tab */}
        <TabPanel value={tabValue} index={0}>
          <Box sx={{ display: 'grid', gap: 2 }}>
            <Paper sx={{ p: 2 }}>
              <Typography variant="subtitle2" gutterBottom>Term Details</Typography>
              <Typography variant="body2">
                <strong>Term ID:</strong> {data.header.termId}
              </Typography>
              <Typography variant="body2">
                <strong>Entity Type:</strong> {data.header.entityType}
              </Typography>
              <Typography variant="body2">
                <strong>Entity ID:</strong> {data.header.entityId}
              </Typography>
              <Typography variant="body2">
                <strong>Resolved Value:</strong> {typeof data.header.value === 'number' 
                  ? `$${data.header.value.toLocaleString()}`
                  : data.header.value}
              </Typography>
            </Paper>
          </Box>
        </TabPanel>

        {/* Lineage Tab */}
        <TabPanel value={tabValue} index={1}>
          <Paper sx={{ p: 2 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
              <AccountTreeIcon />
              <Typography variant="subtitle1">Data Lineage</Typography>
            </Box>
            <Typography variant="body2" gutterBottom>
              <strong>Semantic Term:</strong> {data.lineage.semanticTerm}
            </Typography>
            <Typography variant="body2" gutterBottom>
              <strong>Dependencies:</strong>
            </Typography>
            <Box sx={{ pl: 2, mb: 2 }}>
              {data.lineage.dependencies.map((dep) => (
                <Chip key={dep} label={dep} size="small" sx={{ mr: 0.5, mb: 0.5 }} />
              ))}
            </Box>
            <Typography variant="body2" gutterBottom>
              <strong>Physical Columns:</strong>
            </Typography>
            <Box sx={{ pl: 2 }}>
              {data.lineage.physicalColumns.map((col) => (
                <Chip key={col} label={col} size="small" variant="outlined" sx={{ mr: 0.5, mb: 0.5 }} />
              ))}
            </Box>
          </Paper>
        </TabPanel>

        {/* Rows Tab */}
        <TabPanel value={tabValue} index={2}>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Included</TableCell>
                <TableCell>Key</TableCell>
                <TableCell>Fields</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {data.rows.map((row, idx) => (
                <TableRow 
                  key={idx}
                  sx={{ 
                    bgcolor: row.included ? 'success.50' : 'transparent',
                    opacity: row.included ? 1 : 0.6 
                  }}
                >
                  <TableCell>
                    {row.included 
                      ? <CheckCircleIcon color="success" fontSize="small" />
                      : <span>—</span>
                    }
                  </TableCell>
                  <TableCell>
                    {Object.entries(row.key).map(([k, v]) => (
                      <Chip key={k} label={`${k}: ${v}`} size="small" sx={{ mr: 0.5 }} />
                    ))}
                  </TableCell>
                  <TableCell>
                    {Object.entries(row.fields).map(([k, v]) => (
                      <Typography key={k} variant="body2" component="span" sx={{ mr: 1 }}>
                        {k}: <strong>{String(v)}</strong>
                      </Typography>
                    ))}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TabPanel>

        {/* Evaluation Tab */}
        <TabPanel value={tabValue} index={3}>
          {data.evaluationPath.map((step) => (
            <Paper key={step.step} sx={{ p: 2, mb: 1 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <Chip label={step.step} size="small" color="primary" />
                <Typography variant="subtitle2">{step.action}</Typography>
              </Box>
              <Typography variant="body2" sx={{ mt: 1 }}>
                {step.description}
              </Typography>
              {step.details && (
                <Box sx={{ mt: 1, p: 1, bgcolor: 'grey.100', borderRadius: 1 }}>
                  <Typography variant="caption" component="pre" sx={{ whiteSpace: 'pre-wrap' }}>
                    {JSON.stringify(step.details, null, 2)}
                  </Typography>
                </Box>
              )}
            </Paper>
          ))}
        </TabPanel>

        {/* SQL Tab */}
        <TabPanel value={tabValue} index={4}>
          <Paper sx={{ p: 2 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <Typography variant="subtitle2">Executed SQL</Typography>
                <Chip label={data.sql.mode === 'row' ? 'Row Mode' : 'Pre-Aggregation Mode'} size="small" />
              </Box>
              <Tooltip title="Copy SQL">
                <IconButton size="small" onClick={handleCopySQL}>
                  <ContentCopyIcon fontSize="small" />
                </IconButton>
              </Tooltip>
            </Box>
            <Box sx={{ p: 2, bgcolor: 'grey.900', borderRadius: 1 }}>
              <Typography
                component="pre"
                sx={{ 
                  color: 'grey.100', 
                  fontFamily: 'monospace', 
                  fontSize: '0.85rem',
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                  m: 0
                }}
              >
                {data.sql.text}
              </Typography>
            </Box>
          </Paper>
        </TabPanel>

        {/* Anomalies Tab */}
        {data.anomalies.length > 0 && (
          <TabPanel value={tabValue} index={5}>
            {data.anomalies.map((anomaly, idx) => (
              <Alert 
                key={idx} 
                severity={getSeverityColor(anomaly.severity) as 'error' | 'warning' | 'info'}
                icon={<WarningIcon />}
                sx={{ mb: 1 }}
                action={
                  anomaly.suggestedAction && onStartBP && (
                    <Button 
                      color="inherit" 
                      size="small"
                      onClick={() => onStartBP(anomaly.type)}
                    >
                      {anomaly.suggestedAction}
                    </Button>
                  )
                }
              >
                <AlertTitle>{anomaly.type}</AlertTitle>
                {anomaly.message}
              </Alert>
            ))}
          </TabPanel>
        )}

        {/* Actions */}
        <Box sx={{ display: 'flex', gap: 1, mt: 2 }}>
          {onRecompute && (
            <Button startIcon={<RefreshIcon />} variant="outlined" onClick={onRecompute}>
              Recompute
            </Button>
          )}
          {onOpenTerm && (
            <Button 
              startIcon={<AccountTreeIcon />} 
              variant="outlined"
              onClick={() => onOpenTerm(data.header.termId)}
            >
              Open Term
            </Button>
          )}
        </Box>
      </DialogContent>
    </Dialog>
  );
};

export default HoldingExplainModal;
