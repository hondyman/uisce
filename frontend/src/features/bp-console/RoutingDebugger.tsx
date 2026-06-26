import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  TextField,
  InputAdornment,
  Stack,
  Card,
  CardContent,
  Chip,
  IconButton,
  Tooltip,
  Divider,
  LinearProgress,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import RefreshIcon from '@mui/icons-material/Refresh';
import AltRouteIcon from '@mui/icons-material/AltRoute';
import GroupIcon from '@mui/icons-material/Group';
import PersonIcon from '@mui/icons-material/Person';
import PsychologyIcon from '@mui/icons-material/Psychology';
import CodeIcon from '@mui/icons-material/Code';
import WarningIcon from '@mui/icons-material/Warning';

// Types
interface RoutingDecision {
  id: string;
  workflowId: string;
  stepId: string;
  stepName: string;
  timestamp: string;
  routingType: 'StaticGroup' | 'DynamicRole' | 'Expression' | 'LLMAssisted';
  ruleId: string;
  ruleName: string;
  inputContext: Record<string, any>;
  resolvedAssignees: Array<{
    type: 'user' | 'group';
    id: string;
    name: string;
    priority: number;
  }>;
  fallbackUsed: boolean;
  llmReasoning?: string;
  expression?: string;
  executionTimeMs: number;
}

// Routing Type Icons
const RoutingTypeIcon: React.FC<{ type: string }> = ({ type }) => {
  switch (type) {
    case 'StaticGroup':
      return <GroupIcon />;
    case 'DynamicRole':
      return <PersonIcon />;
    case 'Expression':
      return <CodeIcon />;
    case 'LLMAssisted':
      return <PsychologyIcon />;
    default:
      return <AltRouteIcon />;
  }
};

// Routing Type Colors
const routingTypeColors: Record<string, 'primary' | 'secondary' | 'success' | 'warning'> = {
  StaticGroup: 'primary',
  DynamicRole: 'secondary',
  Expression: 'success',
  LLMAssisted: 'warning',
};

// Main Component
const RoutingDebugger: React.FC = () => {
  const [decisions, setDecisions] = useState<RoutingDecision[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');

  // Fetch routing decisions
  const fetchDecisions = async () => {
    setLoading(true);
    try {
      // Mock data
      const mockDecisions: RoutingDecision[] = [
        {
          id: 'rte-001',
          workflowId: 'wf-model-change-001',
          stepId: 'step-3',
          stepName: 'Advisor Review',
          timestamp: new Date(Date.now() - 3300000).toISOString(),
          routingType: 'DynamicRole',
          ruleId: 'rule-advisor',
          ruleName: 'Primary Advisor Resolution',
          inputContext: {
            clientId: 'client-123',
            portfolioId: 'port-456',
          },
          resolvedAssignees: [
            { type: 'user', id: 'user-001', name: 'john.advisor@firm.com', priority: 0 },
          ],
          fallbackUsed: false,
          executionTimeMs: 45,
        },
        {
          id: 'rte-002',
          workflowId: 'wf-model-change-001',
          stepId: 'step-4',
          stepName: 'Compliance Approval',
          timestamp: new Date(Date.now() - 3200000).toISOString(),
          routingType: 'StaticGroup',
          ruleId: 'rule-compliance',
          ruleName: 'Compliance Team Assignment',
          inputContext: {},
          resolvedAssignees: [
            { type: 'group', id: 'grp-compliance', name: 'Compliance Team', priority: 0 },
          ],
          fallbackUsed: false,
          executionTimeMs: 12,
        },
        {
          id: 'rte-003',
          workflowId: 'wf-escalation-001',
          stepId: 'step-7',
          stepName: 'Manager Escalation',
          timestamp: new Date(Date.now() - 86400000).toISOString(),
          routingType: 'Expression',
          ruleId: 'rule-manager',
          ruleName: 'Manager by Department',
          inputContext: {
            department: 'Wealth Management',
            region: 'Northeast',
          },
          resolvedAssignees: [
            { type: 'user', id: 'user-005', name: 'sarah.manager@firm.com', priority: 0 },
          ],
          expression: '$.department_managers[$.input.department]',
          fallbackUsed: false,
          executionTimeMs: 78,
        },
        {
          id: 'rte-004',
          workflowId: 'wf-complex-001',
          stepId: 'step-10',
          stepName: 'Expert Assignment',
          timestamp: new Date(Date.now() - 172800000).toISOString(),
          routingType: 'LLMAssisted',
          ruleId: 'rule-expert',
          ruleName: 'AI Expert Selection',
          inputContext: {
            caseType: 'Tax Optimization',
            clientComplexity: 'High',
            urgency: 'Medium',
          },
          resolvedAssignees: [
            { type: 'user', id: 'user-010', name: 'mike.tax@firm.com', priority: 0 },
            { type: 'user', id: 'user-011', name: 'lisa.wealth@firm.com', priority: 1 },
          ],
          llmReasoning: 'Selected Mike as primary due to tax expertise and current workload. Lisa as backup given her wealth management background.',
          fallbackUsed: false,
          executionTimeMs: 1250,
        },
        {
          id: 'rte-005',
          workflowId: 'wf-fallback-001',
          stepId: 'step-2',
          stepName: 'Team Lead Review',
          timestamp: new Date(Date.now() - 259200000).toISOString(),
          routingType: 'DynamicRole',
          ruleId: 'rule-teamlead',
          ruleName: 'Team Lead by Region',
          inputContext: {
            region: 'Unknown',
          },
          resolvedAssignees: [
            { type: 'group', id: 'grp-default', name: 'Default Review Team', priority: 0 },
          ],
          fallbackUsed: true,
          executionTimeMs: 55,
        },
      ];

      setDecisions(mockDecisions);
    } catch (error) {
      console.error('Failed to fetch routing decisions:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchDecisions();
  }, []);

  const filteredDecisions = decisions.filter(d =>
    d.stepName.toLowerCase().includes(searchQuery.toLowerCase()) ||
    d.routingType.toLowerCase().includes(searchQuery.toLowerCase()) ||
    d.workflowId.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const formatTime = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  // Stats
  const typeStats = decisions.reduce((acc, d) => {
    acc[d.routingType] = (acc[d.routingType] || 0) + 1;
    return acc;
  }, {} as Record<string, number>);

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Typography variant="h5" fontWeight={600}>
          <AltRouteIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
          Routing Debugger
        </Typography>
        <Stack direction="row" spacing={2}>
          <TextField
            size="small"
            placeholder="Search decisions..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon />
                </InputAdornment>
              ),
            }}
            sx={{ width: 300 }}
          />
          <Tooltip title="Refresh">
            <IconButton onClick={fetchDecisions}>
              <RefreshIcon />
            </IconButton>
          </Tooltip>
        </Stack>
      </Stack>

      {loading && <LinearProgress sx={{ mb: 2 }} />}

      {/* Summary Cards */}
      <Stack direction="row" spacing={2} sx={{ mb: 3 }}>
        {Object.entries(typeStats).map(([type, count]) => (
          <Card key={type} sx={{ flex: 1 }}>
            <CardContent sx={{ textAlign: 'center' }}>
              <RoutingTypeIcon type={type} />
              <Typography variant="h4">{count}</Typography>
              <Typography variant="caption" color="text.secondary">{type}</Typography>
            </CardContent>
          </Card>
        ))}
        <Card sx={{ flex: 1 }}>
          <CardContent sx={{ textAlign: 'center' }}>
            <WarningIcon color="warning" />
            <Typography variant="h4">{decisions.filter(d => d.fallbackUsed).length}</Typography>
            <Typography variant="caption" color="text.secondary">Fallbacks Used</Typography>
          </CardContent>
        </Card>
      </Stack>

      {/* Decisions Table */}
      <TableContainer component={Paper} sx={{ borderRadius: 2 }}>
        <Table>
          <TableHead>
            <TableRow sx={{ bgcolor: 'grey.50' }}>
              <TableCell>Step</TableCell>
              <TableCell>Routing Type</TableCell>
              <TableCell>Rule</TableCell>
              <TableCell>Resolved To</TableCell>
              <TableCell>Time</TableCell>
              <TableCell>Latency</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredDecisions.map((decision) => (
              <TableRow key={decision.id} hover>
                <TableCell>
                  <Typography fontWeight={500}>{decision.stepName}</Typography>
                  <Typography variant="caption" color="text.secondary">
                    {decision.workflowId}
                  </Typography>
                </TableCell>
                <TableCell>
                  <Chip
                    icon={<RoutingTypeIcon type={decision.routingType} />}
                    label={decision.routingType}
                    size="small"
                    color={routingTypeColors[decision.routingType]}
                  />
                  {decision.fallbackUsed && (
                    <Chip label="Fallback" size="small" color="warning" sx={{ ml: 0.5 }} />
                  )}
                </TableCell>
                <TableCell>
                  <Typography variant="body2">{decision.ruleName}</Typography>
                  {decision.expression && (
                    <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                      {decision.expression}
                    </Typography>
                  )}
                  {decision.llmReasoning && (
                    <Tooltip title={decision.llmReasoning}>
                      <Chip icon={<PsychologyIcon />} label="View Reasoning" size="small" sx={{ mt: 0.5 }} />
                    </Tooltip>
                  )}
                </TableCell>
                <TableCell>
                  <Stack spacing={0.5}>
                    {decision.resolvedAssignees.map((a, i) => (
                      <Chip
                        key={i}
                        icon={a.type === 'user' ? <PersonIcon /> : <GroupIcon />}
                        label={a.name}
                        size="small"
                        variant="outlined"
                      />
                    ))}
                  </Stack>
                </TableCell>
                <TableCell>
                  <Typography variant="caption">{formatTime(decision.timestamp)}</Typography>
                </TableCell>
                <TableCell>
                  <Chip label={`${decision.executionTimeMs}ms`} size="small" variant="outlined" />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
};

export default RoutingDebugger;
