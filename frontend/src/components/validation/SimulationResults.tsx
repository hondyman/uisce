import React from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Chip,
  Alert,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Table,
  TableBody,
  TableRow,
  TableCell,
} from '@mui/material';
import {
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
  ExpandMore as ExpandMoreIcon,
} from '@mui/icons-material';

interface SimulationResult {
  rule_id: string;
  rule_name: string;
  rule_type: string;
  severity: string;
  status: 'pass' | 'fail';
  message: string;
  timestamp: string;
  instance_id: string;
  data_used: Record<string, any>;
}

interface SimulationResultsProps {
  result: SimulationResult | null;
  loading?: boolean;
  error?: string | null;
}

export const SimulationResults: React.FC<SimulationResultsProps> = ({
  result,
  loading,
  error,
}) => {
  if (loading) {
    return (
      <Card>
        <CardContent>
          <Typography>Running simulation...</Typography>
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ mb: 2 }}>
        <Typography variant="body2">{error}</Typography>
      </Alert>
    );
  }

  if (!result) {
    return null;
  }

  const isPassed = result.status === 'pass';

  return (
    <Box sx={{ mt: 3 }}>
      <Card variant="outlined">
        <CardContent>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 2 }}>
            {isPassed ? (
              <CheckCircleIcon color="success" sx={{ fontSize: 40 }} />
            ) : (
              <ErrorIcon color="error" sx={{ fontSize: 40 }} />
            )}
            <Box sx={{ flex: 1 }}>
              <Typography variant="h6" gutterBottom>
                {result.rule_name}
              </Typography>
              <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
                <Chip
                  label={result.status.toUpperCase()}
                  color={isPassed ? 'success' : 'error'}
                  size="small"
                />
                <Chip label={result.severity} size="small" variant="outlined" />
                <Chip label={result.rule_type} size="small" variant="outlined" />
              </Box>
            </Box>
          </Box>

          {result.message && (
            <Alert severity={isPassed ? 'success' : 'error'} sx={{ mb: 2 }}>
              {result.message}
            </Alert>
          )}

          <Typography variant="caption" color="text.secondary" gutterBottom display="block">
            Tested at: {new Date(result.timestamp).toLocaleString()}
          </Typography>

          <Accordion sx={{ mt: 2 }}>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography variant="subtitle2">View Data Used in Validation</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Table size="small">
                <TableBody>
                  {Object.entries(result.data_used)
                    .filter(([key]) => !key.startsWith('_'))
                    .map(([key, value]) => (
                      <TableRow key={key}>
                        <TableCell component="th" scope="row" sx={{ fontWeight: 'bold', width: '30%' }}>
                          {key}
                        </TableCell>
                        <TableCell>
                          <Typography variant="body2" component="code">
                            {typeof value === 'object' ? JSON.stringify(value, null, 2) : String(value)}
                          </Typography>
                        </TableCell>
                      </TableRow>
                    ))}
                </TableBody>
              </Table>

              <Accordion sx={{ mt: 2 }} disableGutters>
                <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                  <Typography variant="caption">Metadata</Typography>
                </AccordionSummary>
                <AccordionDetails>
                  <Table size="small">
                    <TableBody>
                      {Object.entries(result.data_used)
                        .filter(([key]) => key.startsWith('_'))
                        .map(([key, value]) => (
                          <TableRow key={key}>
                            <TableCell component="th" scope="row" sx={{ width: '40%' }}>
                              <Typography variant="caption">{key}</Typography>
                            </TableCell>
                            <TableCell>
                              <Typography variant="caption" component="code">
                                {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                              </Typography>
                            </TableCell>
                          </TableRow>
                        ))}
                    </TableBody>
                  </Table>
                </AccordionDetails>
              </Accordion>
            </AccordionDetails>
          </Accordion>
        </CardContent>
      </Card>
    </Box>
  );
};

export default SimulationResults;
