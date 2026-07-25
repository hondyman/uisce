import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  CardHeader,
  Button,
  Grid,
  CircularProgress,
  Typography,
  Stack,
  Paper,
  Chip,
  Container
} from '@mui/material';

// Local dev logger for this package
const devLog = (...args: any[]) => { if (process.env.NODE_ENV !== 'production') console.log(...args); };
import { BusinessProcess, BPStep, ProcessStepType } from '../types/business-process';

interface BusinessProcessDesignerProps {
  tenantId: string;
  datasourceId: string;
}

export const BusinessProcessDesigner: React.FC<BusinessProcessDesignerProps> = ({
  tenantId,
  datasourceId
}) => {
  const [processes, setProcesses] = useState<BusinessProcess[]>([]);
  const [stepTypes, setStepTypes] = useState<ProcessStepType[]>([]);
  const [selectedProcess, setSelectedProcess] = useState<BusinessProcess | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadDesignerData();
  }, [tenantId, datasourceId]);

  const loadDesignerData = async () => {
    try {
      // TODO: Replace with actual API calls
      const processesResponse = await fetch(`/api/business-process/?tenant_id=${tenantId}&datasource_id=${datasourceId}`);
      const stepTypesResponse = await fetch('/api/business-process/step-types');

      const processesData = await processesResponse.json();
      const stepTypesData = await stepTypesResponse.json();

      setProcesses(processesData || []);
      setStepTypes(stepTypesData || []);
    } catch (error) {
      console.error('Failed to load designer data:', error);
    } finally {
      setLoading(false);
    }
  };

  const executeProcess = async (processId: string) => {
    try {
      const response = await fetch(`/api/business-process/${processId}/execute`, {
        method: 'POST',
      });
      const result = await response.json();
      devLog('Process execution started:', result);
    } catch (error) {
      console.error('Failed to execute process:', error);
    }
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', py: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Container maxWidth="lg" sx={{ py: 3 }}>
      <Typography variant="h5" sx={{ fontWeight: 'bold', mb: 4 }}>
        Business Process Designer
      </Typography>

      <Grid container spacing={3} sx={{ mb: 4 }}>
        {/* Process List */}
        <Grid item xs={12} lg={4}>
          <Card sx={{ boxShadow: 2 }}>
            <CardHeader title="Processes" />
            <CardContent>
              <Stack spacing={2} sx={{ mb: 2 }}>
                {processes.map((process) => (
                  <Paper
                    key={process.id}
                    onClick={() => setSelectedProcess(process)}
                    sx={{
                      p: 2,
                      border: '1px solid',
                      borderColor: selectedProcess?.id === process.id ? 'primary.main' : 'divider',
                      bgcolor: selectedProcess?.id === process.id ? 'action.selected' : 'background.paper',
                      cursor: 'pointer',
                      '&:hover': {
                        bgcolor: 'action.hover'
                      }
                    }}
                  >
                    <Typography variant="subtitle2" sx={{ fontWeight: 'medium' }}>
                      {process.processName}
                    </Typography>
                    <Typography variant="body2" color="textSecondary" sx={{ mb: 1 }}>
                      {process.description}
                    </Typography>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <Chip
                        label={process.isActive ? 'Active' : 'Inactive'}
                        size="small"
                        color={process.isActive ? 'success' : 'default'}
                      />
                      <Button
                        size="small"
                        variant="contained"
                        color="primary"
                        onClick={(e) => {
                          e.stopPropagation();
                          executeProcess(process.id);
                        }}
                      >
                        Execute
                      </Button>
                    </Box>
                  </Paper>
                ))}
              </Stack>
              <Button fullWidth variant="contained" color="primary">
                Create Process
              </Button>
            </CardContent>
          </Card>
        </Grid>

        {/* Process Canvas */}
        <Grid item xs={12} lg={8}>
          <Card sx={{ boxShadow: 2, minHeight: 400 }}>
            <CardContent>
              {selectedProcess ? (
                <Box>
                  <Typography variant="h6" sx={{ fontWeight: 'bold', mb: 3 }}>
                    {selectedProcess.processName}
                  </Typography>
                  <Paper
                    sx={{
                      border: '2px dashed',
                      borderColor: 'divider',
                      borderRadius: 2,
                      p: 4,
                      textAlign: 'center'
                    }}
                  >
                    <Typography color="textSecondary" sx={{ mb: 1 }}>
                      Process Canvas
                    </Typography>
                    <Typography variant="caption" color="textSecondary">
                      Steps: {selectedProcess.steps?.length || 0}
                    </Typography>
                    {/* TODO: Implement drag-and-drop canvas */}
                  </Paper>
                </Box>
              ) : (
                <Box
                  sx={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    height: 300,
                    color: 'text.secondary'
                  }}
                >
                  <Typography>Select a process to view its design</Typography>
                </Box>
              )}
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Step Types Palette */}
      <Card sx={{ boxShadow: 2 }}>
        <CardHeader title="Step Types" />
        <CardContent>
          <Grid container spacing={2}>
            {stepTypes.map((stepType) => (
              <Grid item xs={6} sm={4} md={3} key={stepType.id}>
                <Paper
                  sx={{
                    p: 2,
                    textAlign: 'center',
                    border: '1px solid',
                    borderColor: 'divider',
                    '&:hover': {
                      bgcolor: 'action.hover'
                    }
                  }}
                >
                  <Typography variant="h5" sx={{ mb: 1 }}>
                    {stepType.icon_svg || '📋'}
                  </Typography>
                  <Typography variant="subtitle2" sx={{ fontWeight: 'medium' }}>
                    {stepType.label}
                  </Typography>
                  <Typography variant="caption" color="textSecondary">
                    {stepType.description}
                  </Typography>
                </Paper>
              </Grid>
            ))}
          </Grid>
        </CardContent>
      </Card>
    </Container>
  );
};