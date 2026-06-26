import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  CardHeader,
  CircularProgress,
  Divider,
  Grid,
  Paper,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tabs,
  TextField,
  Typography,
  Alert,
  Chip,
} from '@mui/material';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorIcon from '@mui/icons-material/Error';
import WarningIcon from '@mui/icons-material/Warning';
import InfoIcon from '@mui/icons-material/Info';

interface TestSampleData {
  [key: string]: any;
}

interface TestResult {
  row_id: string | number;
  status: 'pass' | 'fail' | 'warning';
  message: string;
  timestamp: string;
}

interface LivePreviewProps {
  rule: {
    target_entity: string;
    field_name: string;
    rule_condition: string;
    severity: 'error' | 'warning' | 'info';
  };
  onTestResults?: (results: TestResult[]) => void;
}

/**
 * Live Preview Component
 * 
 * Allows users to:
 * - Paste sample data and see rule validation results in real-time
 * - Build confidence before deployment
 * - See exactly how the rule will behave
 */
const LivePreview: React.FC<LivePreviewProps> = ({ rule, onTestResults }) => {
  const [tabValue, setTabValue] = useState(0);
  const [sampleData, setSampleData] = useState<string>('');
  const [testResults, setTestResults] = useState<TestResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [previewMode, setPreviewMode] = useState<'json' | 'csv'>('json');

  // Mock data samples for different entity types
  const mockSamples: Record<string, string> = {
    Employee: JSON.stringify(
      [
        {
          employee_id: 'E001',
          first_name: 'John',
          last_name: 'Doe',
          email: 'john@example.com',
          salary: 50000,
        },
        {
          employee_id: 'E002',
          first_name: 'Jane',
          last_name: 'Smith',
          email: null, // Will fail null check
          salary: 55000,
        },
      ],
      null,
      2
    ),
    Transaction: JSON.stringify(
      [
        { transaction_id: 'T001', amount: 100, status: 'completed' },
        { transaction_id: 'T002', amount: -50, status: 'failed' }, // Negative amount
        { transaction_id: 'T003', amount: 200, status: 'pending' },
      ],
      null,
      2
    ),
    Customer: JSON.stringify(
      [
        { customer_id: 'C001', name: 'Acme Corp', email: 'acme@example.com' },
        { customer_id: '', name: 'Unknown', email: null }, // Missing ID
        { customer_id: 'C003', name: 'Tech Inc', email: 'tech@example.com' },
      ],
      null,
      2
    ),
  };

  // Load sample data when entity changes
  useEffect(() => {
    const sample = mockSamples[rule.target_entity];
    if (sample) {
      setSampleData(sample);
    }
  }, [rule.target_entity]);

  // Simulate rule testing
  const handleTestRule = async () => {
    setLoading(true);
    setError(null);

    try {
      let data: TestSampleData[] = [];

      if (previewMode === 'json') {
        data = JSON.parse(sampleData);
      } else {
        // Parse CSV (simple parser)
        const lines = sampleData.trim().split('\n');
        const headers = lines[0].split(',').map((h) => h.trim());
        data = lines.slice(1).map((line) => {
          const values = line.split(',').map((v) => v.trim());
          return Object.fromEntries(headers.map((h, i) => [h, values[i]]));
        });
      }

      if (!Array.isArray(data)) {
        throw new Error('Data must be an array of objects');
      }

      // Simulate testing each row
      const results: TestResult[] = data.map((row, index) => {
        // Get the field value
        const fieldValue = row[rule.field_name];

        // Simple rule evaluation logic (in real app, send to backend)
        let status: 'pass' | 'fail' | 'warning' = 'pass';
        let message = '✓ Rule passed';

        // Mock rule evaluation based on rule condition
        if (rule.rule_condition.includes('NOT NULL')) {
          if (fieldValue === null || fieldValue === '' || fieldValue === undefined) {
            status = rule.severity === 'error' ? 'fail' : 'warning';
            message = `Field "${rule.field_name}" is null/empty`;
          }
        } else if (rule.rule_condition.includes('BETWEEN')) {
          const match = rule.rule_condition.match(/BETWEEN\s+([\d.]+)\s+AND\s+([\d.]+)/i);
          if (match) {
            const min = parseFloat(match[1]);
            const max = parseFloat(match[2]);
            const val = parseFloat(fieldValue);
            if (isNaN(val) || val < min || val > max) {
              status = rule.severity === 'error' ? 'fail' : 'warning';
              message = `Value ${val} is not between ${min} and ${max}`;
            }
          }
        } else if (rule.rule_condition.includes('MATCHES')) {
          const match = rule.rule_condition.match(/MATCHES\s+'([^']+)'/);
          if (match) {
            const pattern = new RegExp(match[1]);
            if (!pattern.test(String(fieldValue))) {
              status = rule.severity === 'error' ? 'fail' : 'warning';
              message = `Value "${fieldValue}" does not match pattern ${match[1]}`;
            }
          }
        }

        return {
          row_id: row.id || index + 1,
          status,
          message,
          timestamp: new Date().toISOString(),
        };
      });

      setTestResults(results);
      if (onTestResults) {
        onTestResults(results);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to test rule');
    } finally {
      setLoading(false);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'pass':
        return <CheckCircleIcon sx={{ color: '#4caf50' }} />;
      case 'fail':
        return <ErrorIcon sx={{ color: '#f44336' }} />;
      case 'warning':
        return <WarningIcon sx={{ color: '#ff9800' }} />;
      default:
        return <InfoIcon sx={{ color: '#2196f3' }} />;
    }
  };

  const passCount = testResults.filter((r) => r.status === 'pass').length;
  const failCount = testResults.filter((r) => r.status === 'fail').length;
  const warningCount = testResults.filter((r) => r.status === 'warning').length;

  return (
    <Card>
      <CardHeader
        title="Live Preview & Testing"
        subheader="Test your rule with sample data before deployment"
      />
      <Divider />

      <CardContent>
        <Tabs value={tabValue} onChange={(_, val) => setTabValue(val)} sx={{ mb: 2 }}>
          <Tab label="Sample Data" />
          <Tab label={`Test Results (${testResults.length})`} />
        </Tabs>

        {tabValue === 0 && (
          <Box>
            <Box sx={{ mb: 2 }}>
              <Typography variant="subtitle2" sx={{ mb: 1 }}>
                Test Data Format
              </Typography>
              <Box sx={{ display: 'flex', gap: 1 }}>
                <Chip
                  label="JSON"
                  onClick={() => setPreviewMode('json')}
                  color={previewMode === 'json' ? 'primary' : 'default'}
                  variant={previewMode === 'json' ? 'filled' : 'outlined'}
                />
                <Chip
                  label="CSV"
                  onClick={() => setPreviewMode('csv')}
                  color={previewMode === 'csv' ? 'primary' : 'default'}
                  variant={previewMode === 'csv' ? 'filled' : 'outlined'}
                />
              </Box>
            </Box>

            <TextField
              fullWidth
              multiline
              rows={10}
              value={sampleData}
              onChange={(e) => setSampleData(e.target.value)}
              placeholder="Paste sample data here..."
              sx={{ fontFamily: 'monospace', mb: 2 }}
            />

            {error && (
              <Alert severity="error" sx={{ mb: 2 }}>
                {error}
              </Alert>
            )}

            <Button
              variant="contained"
              startIcon={<PlayArrowIcon />}
              onClick={handleTestRule}
              disabled={!sampleData || loading}
              size="large"
              fullWidth
            >
              {loading ? <CircularProgress size={20} /> : 'Run Test'}
            </Button>
          </Box>
        )}

        {tabValue === 1 && (
          <Box>
            {testResults.length === 0 ? (
              <Alert severity="info">Run a test to see results</Alert>
            ) : (
              <>
                {/* Results Summary */}
                <Grid container spacing={2} sx={{ mb: 3 }}>
                  <Grid item xs={6} sm={3}>
                    <Paper sx={{ p: 2, textAlign: 'center', bgcolor: '#e8f5e9' }}>
                      <CheckCircleIcon sx={{ color: '#4caf50', mb: 1, fontSize: 32 }} />
                      <Typography variant="h6">{passCount}</Typography>
                      <Typography variant="caption">Passed</Typography>
                    </Paper>
                  </Grid>
                  <Grid item xs={6} sm={3}>
                    <Paper sx={{ p: 2, textAlign: 'center', bgcolor: '#fff3e0' }}>
                      <WarningIcon sx={{ color: '#ff9800', mb: 1, fontSize: 32 }} />
                      <Typography variant="h6">{warningCount}</Typography>
                      <Typography variant="caption">Warnings</Typography>
                    </Paper>
                  </Grid>
                  <Grid item xs={6} sm={3}>
                    <Paper sx={{ p: 2, textAlign: 'center', bgcolor: '#ffebee' }}>
                      <ErrorIcon sx={{ color: '#f44336', mb: 1, fontSize: 32 }} />
                      <Typography variant="h6">{failCount}</Typography>
                      <Typography variant="caption">Failed</Typography>
                    </Paper>
                  </Grid>
                  <Grid item xs={6} sm={3}>
                    <Paper sx={{ p: 2, textAlign: 'center' }}>
                      <Typography variant="h6">{testResults.length}</Typography>
                      <Typography variant="caption">Total Records</Typography>
                    </Paper>
                  </Grid>
                </Grid>

                {/* Results Table */}
                <TableContainer component={Paper}>
                  <Table>
                    <TableHead sx={{ bgcolor: '#f5f5f5' }}>
                      <TableRow>
                        <TableCell width="10%">Status</TableCell>
                        <TableCell>Row ID</TableCell>
                        <TableCell>Message</TableCell>
                        <TableCell width="15%">Time</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {testResults.map((result, idx) => (
                        <TableRow
                          key={idx}
                          sx={{
                            bgcolor:
                              result.status === 'fail'
                                ? '#ffebee'
                                : result.status === 'warning'
                                ? '#fff3e0'
                                : '#f1f8e9',
                          }}
                        >
                          <TableCell>{getStatusIcon(result.status)}</TableCell>
                          <TableCell>{result.row_id}</TableCell>
                          <TableCell>{result.message}</TableCell>
                          <TableCell>
                            {new Date(result.timestamp).toLocaleTimeString()}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              </>
            )}
          </Box>
        )}
      </CardContent>
    </Card>
  );
};

export default LivePreview;
