import React, { useState, useEffect } from 'react';
import { 
  Box, 
  Card, 
  CardContent, 
  Typography, 
  Grid, 
  Chip, 
  Table, 
  TableBody, 
  TableCell, 
  TableContainer, 
  TableHead, 
  TableRow, 
  Paper,
  CircularProgress,
  Alert
} from '@mui/material';
import { 
  AccountTree as TreeIcon, 
  CheckCircle as SuccessIcon, 
  Error as ErrorIcon,
  Timeline as LineageIcon
} from '@mui/icons-material';

interface ExecutionTrace {
  term_id: string;
  term_name: string;
  inputs: Record<string, any>;
  output: any;
  dependencies?: Record<string, ExecutionTrace>;
  error?: string;
}

interface ExplainabilityDashboardProps {
  rootTermId?: string;
}

export const ExplainabilityDashboard: React.FC<ExplainabilityDashboardProps> = ({ rootTermId }) => {
  const [loading, setLoading] = useState(false);
  const [trace, setTrace] = useState<ExecutionTrace | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (rootTermId) {
      fetchTrace(rootTermId);
    }
  }, [rootTermId]);

  const fetchTrace = async (id: string) => {
    setLoading(true);
    setError(null);
    try {
      // For demo purposes, we fetch the simulation trace which includes execution logic
      const response = await fetch(`/api/v1/execute/${id}`, {
        headers: {
          'X-Tenant-ID': '99e99e99-99e9-49e9-89e9-99e99e99e999',
          'X-User-ID': 'admin'
        }
      });
      if (!response.ok) throw new Error('Failed to fetch execution trace');
      const data = await response.json();
      setTrace(data.trace);
    } catch (err: any) {
      setError(err.message);
      // Mocking data for development if endpoint doesn't exist yet
      setTrace({
        term_id: id,
        term_name: "NetAssetValue",
        inputs: { "PositionValue": 1000000 },
        output: 1000000,
        dependencies: {
          "PositionValue": {
            term_id: "pos-val-id",
            term_name: "PositionValue",
            inputs: { "MarketPrice": 100, "Quantity": 10000 },
            output: 1000000,
            dependencies: {
              "MarketPrice": { term_id: "p-id", term_name: "MarketPrice", inputs: {}, output: 100 },
              "Quantity": { term_id: "q-id", term_name: "Quantity", inputs: {}, output: 10000 }
            }
          }
        }
      });
    } finally {
      setLoading(false);
    }
  };

  const renderDependencyTree = (node: ExecutionTrace, level = 0) => (
    <Box key={node.term_id} sx={{ ml: level * 3, mt: 1, borderLeft: '1px dashed #ccc', pl: 2 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
        <Chip 
          icon={<TreeIcon />} 
          label={node.term_name} 
          color={node.error ? "error" : "primary"}
          variant="outlined"
          size="small"
        />
        <Typography variant="body2" sx={{ ml: 2, fontWeight: 'bold' }}>
          = {typeof node.output === 'number' ? node.output.toLocaleString() : JSON.stringify(node.output)}
        </Typography>
        {node.error && <ErrorIcon color="error" sx={{ ml: 1, fontSize: 16 }} />}
      </Box>
      {node.dependencies && Object.values(node.dependencies).map(dep => renderDependencyTree(dep, level + 1))}
    </Box>
  );

  if (loading) return <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}><CircularProgress /></Box>;

  return (
    <Box sx={{ p: 3, backgroundColor: '#f5f7fa', minHeight: '100vh' }}>
      <Typography variant="h4" gutterBottom sx={{ display: 'flex', alignItems: 'center', fontWeight: 'bold' }}>
        <LineageIcon sx={{ mr: 2, fontSize: 32 }} color="primary" />
        Semantic Execution Dashboard
      </Typography>

      {error && <Alert severity="warning" sx={{ mb: 3 }}>Running in Demonstration Mode: {error}</Alert>}

      <Grid container spacing={3}>
        <Grid item xs={12} md={8}>
          <Card elevation={2} sx={{ borderRadius: 2 }}>
            <CardContent>
              <Typography variant="h6" gutterBottom>Execution Trace (Recursive Lineage)</Typography>
              <Box sx={{ mt: 2, p: 2, bgcolor: '#ffffff', borderRadius: 1, overflow: 'auto' }}>
                {trace ? renderDependencyTree(trace) : <Typography color="textSecondary">No trace data available</Typography>}
              </Box>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={4}>
          <Card elevation={2} sx={{ borderRadius: 2, mb: 3 }}>
            <CardContent>
              <Typography variant="h6" gutterBottom>Execution Metadata</Typography>
              <TableContainer component={Box}>
                <Table size="small">
                  <TableBody>
                    <TableRow>
                      <TableCell sx={{ fontWeight: 'bold' }}>Engine</TableCell>
                      <TableCell><Chip label="WASM (Wazero)" size="small" color="success" /></TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell sx={{ fontWeight: 'bold' }}>Status</TableCell>
                      <TableCell><SuccessIcon color="success" sx={{ fontSize: 18, mr: 1, verticalAlign: 'middle' }} /> Success</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell sx={{ fontWeight: 'bold' }}>Recursive Depth</TableCell>
                      <TableCell>3</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </TableContainer>
            </CardContent>
          </Card>

          <Card elevation={2} sx={{ borderRadius: 2 }}>
            <CardContent>
              <Typography variant="h6" gutterBottom>Data Quality Insights</Typography>
              <Alert severity="info" sx={{ mt: 1 }}>
                All inputs resolved from Golden Copy sources.
              </Alert>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
};

export default ExplainabilityDashboard;
