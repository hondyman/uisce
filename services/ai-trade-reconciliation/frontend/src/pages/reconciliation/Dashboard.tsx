import React, { useEffect, useState } from 'react';
import { LineChart, Line, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import {
  Container,
  Grid,
  Card,
  CardContent,
  CardHeader,
  Typography,
  Box,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Button,
  Stack,
  CircularProgress,
  Alert
} from '@mui/material';
import { useTheme } from '@mui/material/styles';

interface ReconciliationResult {
  id: string;
  run_date: string;
  match_rate: number;
  matched_count: number;
  unmatched_count: number;
  discrepancies: any[];
  status: string;
}

interface Task {
  id: string;
  status: string;
  priority: string;
  created_at: string;
}

const AIReconciliationDashboard: React.FC = () => {
  const theme = useTheme();
  const [result, setResult] = useState<ReconciliationResult | null>(null);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        // Fetch latest result
        const resultRes = await fetch('/api/reconciliation/results/latest');
        const resultData = await resultRes.json();
        setResult(resultData);

        // Fetch open tasks
        const tasksRes = await fetch('/api/reconciliation/tasks');
        const tasksData = await tasksRes.json();
        setTasks(tasksData || []);

        setLoading(false);
      } catch (err) {
        console.error('Failed to fetch data', err);
        setLoading(false);
      }
    };

    fetchData();
    const interval = setInterval(fetchData, 60000); // Refresh every minute

    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (!result) {
    return (
      <Container maxWidth="lg" sx={{ py: 4 }}>
        <Alert severity="error">No reconciliation data available</Alert>
      </Container>
    );
  }

  const matchData = [
    { name: 'Matched', value: result.matched_count },
    { name: 'Unmatched', value: result.unmatched_count },
  ];

  const COLORS = ['#10b981', '#ef4444'];

  const getPriorityColor = (priority: string): 'error' | 'warning' | 'success' => {
    switch (priority) {
      case 'high':
        return 'error';
      case 'medium':
        return 'warning';
      default:
        return 'success';
    }
  };

  const getSeverityColor = (severity: string): 'error' | 'warning' | 'success' => {
    switch (severity) {
      case 'high':
        return 'error';
      case 'medium':
        return 'warning';
      default:
        return 'success';
    }
  };

  return (
    <Box sx={{ bgcolor: 'background.default', minHeight: '100vh', py: 4 }}>
      <Container maxWidth="lg">
        <Typography variant="h3" sx={{ fontWeight: 'bold', mb: 4 }}>
          AI Trade Reconciliation Dashboard
        </Typography>

        {/* KPI Cards */}
        <Grid container spacing={3} sx={{ mb: 4 }}>
          <Grid item xs={12} sm={6} md={4}>
            <Card sx={{ boxShadow: 2 }}>
              <CardContent>
                <Typography color="textSecondary" gutterBottom variant="overline">
                  Match Rate
                </Typography>
                <Typography variant="h4" sx={{ fontWeight: 'bold', color: 'success.main', mb: 1 }}>
                  {(result.match_rate * 100).toFixed(1)}%
                </Typography>
                <Typography variant="caption" color="textSecondary">
                  {result.matched_count} of {result.matched_count + result.unmatched_count} trades
                </Typography>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12} sm={6} md={4}>
            <Card sx={{ boxShadow: 2 }}>
              <CardContent>
                <Typography color="textSecondary" gutterBottom variant="overline">
                  Run Date
                </Typography>
                <Typography variant="h6" sx={{ fontWeight: 'bold', mb: 1 }}>
                  {new Date(result.run_date).toLocaleDateString()}
                </Typography>
                <Typography variant="caption" color="textSecondary">
                  {new Date(result.run_date).toLocaleTimeString()}
                </Typography>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12} sm={6} md={4}>
            <Card sx={{ boxShadow: 2 }}>
              <CardContent>
                <Typography color="textSecondary" gutterBottom variant="overline">
                  Open Tasks
                </Typography>
                <Typography variant="h4" sx={{ fontWeight: 'bold', color: 'warning.main', mb: 1 }}>
                  {tasks.filter(t => t.status === 'open').length}
                </Typography>
                <Typography variant="caption" color="textSecondary">
                  Requiring attention
                </Typography>
              </CardContent>
            </Card>
          </Grid>
        </Grid>

        {/* Charts */}
        <Grid container spacing={3} sx={{ mb: 4 }}>
          <Grid item xs={12} md={6}>
            <Card sx={{ boxShadow: 2 }}>
              <CardHeader title="Match Distribution" />
              <CardContent>
                <ResponsiveContainer width="100%" height={300}>
                  <PieChart>
                    <Pie
                      data={matchData}
                      cx="50%"
                      cy="50%"
                      labelLine={false}
                      label={({ name, value }) => `${name}: ${value}`}
                      outerRadius={80}
                      fill="#8884d8"
                      dataKey="value"
                    >
                      {COLORS.map((color, index) => (
                        <Cell key={`cell-${index}`} fill={color} />
                      ))}
                    </Pie>
                    <Tooltip />
                  </PieChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12} md={6}>
            <Card sx={{ boxShadow: 2 }}>
              <CardHeader title="Discrepancies by Severity" />
              <CardContent>
                <Stack spacing={2}>
                  {result.discrepancies.reduce((acc, d) => {
                    const severity = d.severity || 'unknown';
                    acc[severity] = (acc[severity] || 0) + 1;
                    return acc;
                  }, {} as Record<string, number>) &&
                  Object.keys(
                    result.discrepancies.reduce((acc, d) => {
                      const severity = d.severity || 'unknown';
                      acc[severity] = (acc[severity] || 0) + 1;
                      return acc;
                    }, {} as Record<string, number>)
                  ).length > 0 ? (
                    Object.entries(
                      result.discrepancies.reduce((acc, d) => {
                        const severity = d.severity || 'unknown';
                        acc[severity] = (acc[severity] || 0) + 1;
                        return acc;
                      }, {} as Record<string, number>)
                    ).map(([severity, count]) => (
                      <Box
                        key={severity}
                        sx={{
                          display: 'flex',
                          justifyContent: 'space-between',
                          alignItems: 'center'
                        }}
                      >
                        <Typography variant="body2" sx={{ textTransform: 'capitalize' }}>
                          {severity}
                        </Typography>
                        <Chip
                          label={count}
                          color={getSeverityColor(severity)}
                          size="small"
                        />
                      </Box>
                    ))
                  ) : (
                    <Typography variant="body2" color="textSecondary">
                      No discrepancies found
                    </Typography>
                  )}
                </Stack>
              </CardContent>
            </Card>
          </Grid>
        </Grid>

        {/* Tasks Table */}
        <Card sx={{ boxShadow: 2, mb: 4 }}>
          <CardHeader title="Open Tasks" />
          <CardContent>
            {tasks.length > 0 ? (
              <TableContainer>
                <Table>
                  <TableHead>
                    <TableRow sx={{ bgcolor: 'grey.100' }}>
                      <TableCell>Task ID</TableCell>
                      <TableCell>Priority</TableCell>
                      <TableCell>Status</TableCell>
                      <TableCell>Created</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {tasks.slice(0, 5).map((task) => (
                      <TableRow key={task.id} hover>
                        <TableCell variant="body" sx={{ fontSize: '0.875rem' }}>
                          {task.id.substring(0, 8)}
                        </TableCell>
                        <TableCell>
                          <Chip
                            label={task.priority.toUpperCase()}
                            size="small"
                            color={getPriorityColor(task.priority)}
                          />
                        </TableCell>
                        <TableCell sx={{ textTransform: 'capitalize' }}>{task.status}</TableCell>
                        <TableCell sx={{ fontSize: '0.875rem' }}>
                          {new Date(task.created_at).toLocaleDateString()}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            ) : (
              <Typography variant="body2" color="textSecondary">
                No open tasks
              </Typography>
            )}
          </CardContent>
        </Card>

        {/* Action Buttons */}
        <Stack direction="row" gap={2}>
          <Button variant="contained" color="primary" size="large">
            Download Report
          </Button>
          <Button variant="outlined" color="primary" size="large">
            Run Reconciliation
          </Button>
        </Stack>
      </Container>
    </Box>
  );
};

export default AIReconciliationDashboard;
