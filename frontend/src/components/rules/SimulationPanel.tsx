import { useState } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  Slider,
  Tabs,
  Tab,
  Typography,
  Alert,
  LinearProgress,
  Stack,
  Paper,
  TabPanel,
} from '@mui/material';
import {
  PlayArrow as PlayIcon,
  Download as DownloadIcon,
  Share as ShareIcon,
  Bolt as ZapIcon,
} from '@mui/icons-material';

interface SimulationPanelProps {
  rule: any;
  businessObject: string;
  testData: any;
  onTestDataChange: (data: any) => void;
  simulationResults: any;
}

/**
 * SimulationPanel Component (Material-UI)
 * Right-panel for real-time rule testing
 */
export const SimulationPanel = ({
  rule,
  businessObject,
  testData,
  onTestDataChange,
  simulationResults,
}: SimulationPanelProps) => {
  const [activeTab, setActiveTab] = useState<'test' | 'trace' | 'impact'>('test');
  const [whatIfConfidence, setWhatIfConfidence] = useState<number>(70);
  const [showShareLink, setShowShareLink] = useState(false);

  const scenarios = [
    {
      id: 'default',
      name: 'Default (Full Year)',
      description: 'US holidays for 2026',
      region: 'US',
      year: 2026,
    },
    {
      id: 'conflict',
      name: 'Conflict Scenario',
      description: 'Christmas (known conflicts)',
      region: 'US',
      dates: ['2026-12-25'],
    },
    {
      id: 'gb',
      name: 'Multiple Regions',
      description: 'US & GB holidays',
      regions: ['US', 'GB'],
      year: 2026,
    },
  ];

  const handleScenarioSelect = (scenarioId: string) => {
    const scenario = scenarios.find((s) => s.id === scenarioId);
    if (scenario) {
      onTestDataChange(scenario);
    }
  };

  const generateSimulationLink = () => {
    const link = `${window.location.origin}/rules/${rule?.id}/simulation?scenario=${testData?.id}`;
    navigator.clipboard.writeText(link);
    setShowShareLink(true);
    setTimeout(() => setShowShareLink(false), 2000);
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Header */}
      <Paper sx={{ p: 2, borderRadius: 0 }} elevation={0}>
        <Typography variant="subtitle2" fontWeight="600">
          Simulation & Impact
        </Typography>
        <Typography variant="caption" color="textSecondary" sx={{ mt: 0.5, display: 'block' }}>
          Test and validate your rules
        </Typography>
      </Paper>

      {/* Tabs */}
      <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Stack direction="row" spacing={0} sx={{ display: 'flex' }}>
          <Button
            onClick={() => setActiveTab('test')}
            sx={{
              flex: 1,
              borderRadius: 0,
              fontWeight: activeTab === 'test' ? 600 : 400,
              borderBottom: activeTab === 'test' ? 3 : 0,
              borderBottomColor: 'primary.main',
              color: activeTab === 'test' ? 'primary.main' : 'text.secondary',
            }}
          >
            Test Data
          </Button>
          <Button
            onClick={() => setActiveTab('trace')}
            disabled={!simulationResults}
            sx={{
              flex: 1,
              borderRadius: 0,
              fontWeight: activeTab === 'trace' ? 600 : 400,
              borderBottom: activeTab === 'trace' ? 3 : 0,
              borderBottomColor: 'primary.main',
              color: activeTab === 'trace' ? 'primary.main' : 'text.secondary',
            }}
          >
            Execution Trace
          </Button>
          <Button
            onClick={() => setActiveTab('impact')}
            disabled={!simulationResults}
            sx={{
              flex: 1,
              borderRadius: 0,
              fontWeight: activeTab === 'impact' ? 600 : 400,
              borderBottom: activeTab === 'impact' ? 3 : 0,
              borderBottomColor: 'primary.main',
              color: activeTab === 'impact' ? 'primary.main' : 'text.secondary',
            }}
          >
            Impact
          </Button>
        </Stack>
      </Box>

      {/* Content */}
      <Box sx={{ flex: 1, overflowY: 'auto', p: 2 }}>
        {/* Test Data Tab */}
        {activeTab === 'test' && (
          <Stack spacing={2}>
            <Box>
              <Typography variant="caption" fontWeight="600" sx={{ mb: 1, display: 'block' }}>
                Test Scenario
              </Typography>
              <Stack spacing={1}>
                {scenarios.map((scenario) => (
                  <Card
                    key={scenario.id}
                    onClick={() => handleScenarioSelect(scenario.id)}
                    sx={{
                      cursor: 'pointer',
                      borderWidth: 2,
                      borderStyle: 'solid',
                      borderColor: testData?.id === scenario.id ? 'primary.main' : 'divider',
                      backgroundColor:
                        testData?.id === scenario.id ? 'primary.lighter' : 'background.paper',
                      '&:hover': {
                        borderColor: testData?.id === scenario.id ? 'primary.main' : 'action.hover',
                      },
                    }}
                  >
                    <CardContent sx={{ py: 1.5 }}>
                      <Typography variant="body2" fontWeight="500">
                        {scenario.name}
                      </Typography>
                      <Typography variant="caption" color="textSecondary" sx={{ mt: 0.5, display: 'block' }}>
                        {scenario.description}
                      </Typography>
                    </CardContent>
                  </Card>
                ))}
              </Stack>
            </Box>

            {testData && (
              <Button
                onClick={() => console.log('Run simulation')}
                variant="contained"
                color="primary"
                startIcon={<PlayIcon />}
                fullWidth
              >
                Run Simulation
              </Button>
            )}

            {testData && (
              <Paper sx={{ p: 1.5, backgroundColor: 'action.hover' }}>
                <Typography variant="caption" fontWeight="600" sx={{ mb: 1, display: 'block' }}>
                  Scenario Details
                </Typography>
                <Stack spacing={0.5}>
                  {testData.region && <Typography variant="caption">Region: {testData.region}</Typography>}
                  {testData.regions && <Typography variant="caption">Regions: {testData.regions.join(', ')}</Typography>}
                  {testData.year && <Typography variant="caption">Year: {testData.year}</Typography>}
                  {testData.dates && <Typography variant="caption">Dates: {testData.dates.join(', ')}</Typography>}
                </Stack>
              </Paper>
            )}
          </Stack>
        )}

        {/* Execution Trace Tab */}
        {activeTab === 'trace' && simulationResults && (
          <Stack spacing={1.5}>
            {simulationResults.executionTrace?.length > 0 ? (
              simulationResults.executionTrace.slice(0, 5).map((trace: any, idx: number) => (
                <Card key={idx}>
                  <CardContent sx={{ pb: 1 }}>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 1.5 }}>
                      <Box>
                        <Typography variant="body2" fontWeight="500">
                          {trace.date}
                        </Typography>
                        <Typography variant="caption" color="textSecondary" sx={{ mt: 0.5, display: 'block' }}>
                          {trace.region}
                        </Typography>
                      </Box>
                      <Box sx={{ textAlign: 'right' }}>
                        <Typography variant="body2" fontWeight="bold" color="primary">
                          {trace.winningRule}
                        </Typography>
                        <Typography variant="caption" color="textSecondary">
                          Confidence: {trace.confidence}%
                        </Typography>
                      </Box>
                    </Box>

                    <Stack spacing={0.5}>
                      {trace.evaluatedRules?.map((evaluated: any, ruleIdx: number) => (
                        <Paper
                          key={ruleIdx}
                          sx={{
                            p: 1,
                            backgroundColor: evaluated.matched ? 'success.lighter' : 'action.hover',
                            color: evaluated.matched ? 'success.dark' : 'text.secondary',
                          }}
                        >
                          <Typography variant="caption">
                            {evaluated.matched ? '✓' : '✗'} Priority {evaluated.priority}: {evaluated.condition}
                          </Typography>
                        </Paper>
                      ))}
                    </Stack>
                  </CardContent>
                </Card>
              ))
            ) : (
              <Box sx={{ p: 3, textAlign: 'center' }}>
                <Typography variant="body2" color="textSecondary">
                  No execution trace available
                </Typography>
              </Box>
            )}
          </Stack>
        )}

        {/* Impact Analysis Tab */}
        {activeTab === 'impact' && simulationResults && (
          <Stack spacing={2}>
            <Alert severity="info">
              <Box>
                <Typography variant="subtitle2" fontWeight="600" sx={{ mb: 1 }}>
                  Impact Analysis
                </Typography>
                <Stack spacing={1}>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                    <Typography variant="caption">Total dates affected</Typography>
                    <Typography variant="body2" fontWeight="bold">
                      {simulationResults.impactedDates || 0}
                    </Typography>
                  </Box>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                    <Typography variant="caption">Dates changed</Typography>
                    <Typography
                      variant="caption"
                      fontWeight="bold"
                      color={simulationResults.changedDates > 0 ? 'warning.main' : 'success.main'}
                    >
                      {simulationResults.changedDates || 0}
                    </Typography>
                  </Box>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                    <Typography variant="caption">Avg. confidence</Typography>
                    <Typography variant="caption" fontWeight="bold" color="primary">
                      {simulationResults.avgConfidence || 0}%
                    </Typography>
                  </Box>
                </Stack>
              </Box>
            </Alert>

            <Alert severity="warning">
              <Box>
                <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                  <ZapIcon sx={{ mr: 1, fontSize: '1.2rem' }} />
                  <Typography variant="subtitle2" fontWeight="600">
                    What If Analysis
                  </Typography>
                </Box>
                <Stack spacing={2}>
                  <Typography variant="caption" fontWeight="600">
                    Adjust confidence threshold:
                  </Typography>
                  <Slider
                    value={whatIfConfidence}
                    onChange={(e, newValue) => setWhatIfConfidence(newValue as number)}
                    step={5}
                    min={0}
                    max={100}
                    marks
                  />
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Typography variant="caption">Threshold: {whatIfConfidence}%</Typography>
                    <Button
                      onClick={() => console.log('Re-run with:', whatIfConfidence)}
                      variant="contained"
                      color="warning"
                      size="small"
                    >
                      Test
                    </Button>
                  </Box>
                </Stack>
              </Box>
            </Alert>

            <Paper sx={{ p: 1.5 }}>
              <Typography variant="subtitle2" fontWeight="600" sx={{ mb: 2 }}>
                Confidence Distribution
              </Typography>
              <Stack spacing={1.5}>
                {[
                  { range: '90-100%', count: 120, color: 'success.main' },
                  { range: '75-89%', count: 45, color: 'primary.main' },
                  { range: '60-74%', count: 20, color: 'warning.main' },
                  { range: '0-59%', count: 5, color: 'error.main' },
                ].map((bucket) => (
                  <Box key={bucket.range} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <Typography variant="caption" sx={{ width: 45 }}>
                      {bucket.range}
                    </Typography>
                    <Box sx={{ flex: 1 }}>
                      <LinearProgress
                        variant="determinate"
                        value={(bucket.count / 190) * 100}
                        sx={{ height: 16, borderRadius: 1 }}
                      />
                    </Box>
                    <Typography variant="caption" sx={{ width: 40, textAlign: 'right' }}>
                      {bucket.count}
                    </Typography>
                  </Box>
                ))}
              </Stack>
            </Paper>
          </Stack>
        )}
      </Box>

      {/* Footer */}
      <Box sx={{ borderTop: 1, borderColor: 'divider', p: 2, backgroundColor: 'action.hover' }}>
        <Stack spacing={1}>
          <Button
            onClick={generateSimulationLink}
            variant="outlined"
            startIcon={<ShareIcon />}
            fullWidth
          >
            {showShareLink ? '✓ Link Copied' : 'Share Simulation'}
          </Button>
          <Button variant="outlined" startIcon={<DownloadIcon />} fullWidth>
            Export Results
          </Button>
        </Stack>
      </Box>
    </Box>
  );
};

export default SimulationPanel;
