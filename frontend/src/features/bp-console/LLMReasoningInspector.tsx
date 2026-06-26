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
  Alert,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  LinearProgress,
  Tab,
  Tabs,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import RefreshIcon from '@mui/icons-material/Refresh';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import PsychologyIcon from '@mui/icons-material/Psychology';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import WarningIcon from '@mui/icons-material/Warning';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';

// Types
interface LLMInvocation {
  id: string;
  workflowId: string;
  stepId: string;
  stepName: string;
  stepType: string;
  timestamp: string;
  profileId: string;
  modelName: string;
  promptTemplate: string;
  promptFilled: string;
  inputSnapshot: Record<string, any>;
  outputRaw: string;
  outputProcessed: any;
  tokensUsed: number;
  latencyMs: number;
  safetyFlags: string[];
  policyViolations: string[];
}

// Code Block Component
const CodeBlock: React.FC<{ code: string; title?: string }> = ({ code, title }) => {
  const handleCopy = () => {
    navigator.clipboard.writeText(code);
  };

  return (
    <Box sx={{ position: 'relative' }}>
      {title && (
        <Typography variant="caption" color="text.secondary" sx={{ mb: 0.5, display: 'block' }}>
          {title}
        </Typography>
      )}
      <Paper
        sx={{
          p: 2,
          bgcolor: '#1e1e1e',
          color: '#d4d4d4',
          fontFamily: 'monospace',
          fontSize: '0.85rem',
          borderRadius: 1,
          overflow: 'auto',
          maxHeight: 300,
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-word',
        }}
      >
        {code}
        <IconButton
          size="small"
          onClick={handleCopy}
          sx={{ position: 'absolute', top: 8, right: 8, color: 'grey.500' }}
        >
          <ContentCopyIcon fontSize="small" />
        </IconButton>
      </Paper>
    </Box>
  );
};

// Main Component
const LLMReasoningInspector: React.FC = () => {
  const [invocations, setInvocations] = useState<LLMInvocation[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedTab, setSelectedTab] = useState(0);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  // Fetch LLM invocations
  const fetchInvocations = async () => {
    setLoading(true);
    try {
      // TODO: Replace with actual API call
      // const response = await fetch('/api/bp/llm-invocations');
      // const data = await response.json();

      // Mock data
      const mockInvocations: LLMInvocation[] = [
        {
          id: 'llm-001',
          workflowId: 'wf-model-change-001',
          stepId: 'step-1',
          stepName: 'Parse Request',
          stepType: 'Interpretation',
          timestamp: new Date(Date.now() - 3500000).toISOString(),
          profileId: 'interpretation_default',
          modelName: 'gemini-2.0-flash-exp',
          promptTemplate: `Extract the following fields from the input:
{{fields_to_extract}}

Input:
{{input_text}}

Return as JSON:`,
          promptFilled: `Extract the following fields from the input:
name, account_number, change_type, reason

Input:
"Hi, I'd like to change my investment model from Aggressive Growth to Balanced for account 12345. 
The reason is that I'm approaching retirement and want to reduce risk exposure."

Return as JSON:`,
          inputSnapshot: {
            fields_to_extract: 'name, account_number, change_type, reason',
            input_text: 'Hi, I\'d like to change my investment model...',
          },
          outputRaw: `{
  "account_number": "12345",
  "change_type": "model_change",
  "from_model": "Aggressive Growth",
  "to_model": "Balanced",
  "reason": "Approaching retirement, reduce risk exposure"
}`,
          outputProcessed: {
            account_number: '12345',
            change_type: 'model_change',
            from_model: 'Aggressive Growth',
            to_model: 'Balanced',
            reason: 'Approaching retirement, reduce risk exposure',
          },
          tokensUsed: 342,
          latencyMs: 1250,
          safetyFlags: [],
          policyViolations: [],
        },
        {
          id: 'llm-002',
          workflowId: 'wf-model-change-001',
          stepId: 'step-2',
          stepName: 'Risk Classification',
          stepType: 'Classification',
          timestamp: new Date(Date.now() - 3400000).toISOString(),
          profileId: 'classification_default',
          modelName: 'gemini-2.0-flash-exp',
          promptTemplate: `Categories:
{{categories}}

Input to classify:
{{input_text}}

Classification:`,
          promptFilled: `Categories:
low_risk, medium_risk, high_risk, requires_review

Input to classify:
Model change from Aggressive Growth to Balanced for account approaching retirement.
Account value: $1,500,000

Classification:`,
          inputSnapshot: {
            categories: 'low_risk, medium_risk, high_risk, requires_review',
            input_text: 'Model change from Aggressive Growth to Balanced...',
          },
          outputRaw: 'medium_risk',
          outputProcessed: 'medium_risk',
          tokensUsed: 128,
          latencyMs: 890,
          safetyFlags: [],
          policyViolations: [],
        },
        {
          id: 'llm-003',
          workflowId: 'wf-recommendation-001',
          stepId: 'step-5',
          stepName: 'Generate Recommendation',
          stepType: 'Recommendation',
          timestamp: new Date(Date.now() - 86400000).toISOString(),
          profileId: 'recommendation_default',
          modelName: 'gemini-2.0-flash-exp',
          promptTemplate: `Context:
{{context}}

Constraints:
{{constraints}}

Generate recommendation:`,
          promptFilled: `Context:
Client approaching retirement (age 63), current allocation 80% equity.
Risk tolerance: Medium. Time horizon: 5 years.

Constraints:
- Maximum equity allocation: 60%
- Must maintain dividend income
- No high-risk alternatives

Generate recommendation:`,
          inputSnapshot: { context: 'Client approaching retirement...', constraints: 'Max equity 60%...' },
          outputRaw: `{
  "recommendation": "Transition to Balanced Growth model",
  "allocation": {"equity": 55, "fixed_income": 35, "alternatives": 10},
  "rationale": "Reduces risk while maintaining growth potential for 5-year horizon",
  "risk_score": 5.2
}`,
          outputProcessed: {
            recommendation: 'Transition to Balanced Growth model',
            allocation: { equity: 55, fixed_income: 35, alternatives: 10 },
          },
          tokensUsed: 512,
          latencyMs: 2100,
          safetyFlags: ['policy_compliance'],
          policyViolations: [],
        },
      ];

      setInvocations(mockInvocations);
    } catch (error) {
      console.error('Failed to fetch LLM invocations:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchInvocations();
  }, []);

  const filteredInvocations = invocations.filter(inv =>
    inv.stepName.toLowerCase().includes(searchQuery.toLowerCase()) ||
    inv.stepType.toLowerCase().includes(searchQuery.toLowerCase()) ||
    inv.workflowId.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const formatTime = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Typography variant="h5" fontWeight={600}>
          <PsychologyIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
          LLM Reasoning Inspector
        </Typography>
        <Stack direction="row" spacing={2}>
          <TextField
            size="small"
            placeholder="Search invocations..."
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
            <IconButton onClick={fetchInvocations}>
              <RefreshIcon />
            </IconButton>
          </Tooltip>
        </Stack>
      </Stack>

      {loading && <LinearProgress sx={{ mb: 2 }} />}

      {/* Summary Cards */}
      <Stack direction="row" spacing={2} sx={{ mb: 3 }}>
        <Card sx={{ flex: 1 }}>
          <CardContent>
            <Typography variant="subtitle2" color="text.secondary">Total Invocations</Typography>
            <Typography variant="h4">{invocations.length}</Typography>
          </CardContent>
        </Card>
        <Card sx={{ flex: 1 }}>
          <CardContent>
            <Typography variant="subtitle2" color="text.secondary">Avg Latency</Typography>
            <Typography variant="h4">
              {Math.round(invocations.reduce((a, b) => a + b.latencyMs, 0) / (invocations.length || 1))}ms
            </Typography>
          </CardContent>
        </Card>
        <Card sx={{ flex: 1 }}>
          <CardContent>
            <Typography variant="subtitle2" color="text.secondary">Total Tokens</Typography>
            <Typography variant="h4">
              {invocations.reduce((a, b) => a + b.tokensUsed, 0).toLocaleString()}
            </Typography>
          </CardContent>
        </Card>
        <Card sx={{ flex: 1 }}>
          <CardContent>
            <Typography variant="subtitle2" color="text.secondary">Policy Violations</Typography>
            <Typography variant="h4" color={invocations.some(i => i.policyViolations.length > 0) ? 'error.main' : 'success.main'}>
              {invocations.reduce((a, b) => a + b.policyViolations.length, 0)}
            </Typography>
          </CardContent>
        </Card>
      </Stack>

      {/* Invocation List */}
      <Stack spacing={2}>
        {filteredInvocations.map((inv) => (
          <Accordion
            key={inv.id}
            expanded={expandedId === inv.id}
            onChange={() => setExpandedId(expandedId === inv.id ? null : inv.id)}
            sx={{ borderRadius: 2, '&:before': { display: 'none' } }}
          >
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Stack direction="row" alignItems="center" spacing={2} sx={{ width: '100%' }}>
                <PsychologyIcon color="primary" />
                <Box sx={{ flex: 1 }}>
                  <Typography fontWeight={500}>{inv.stepName}</Typography>
                  <Typography variant="caption" color="text.secondary">
                    {inv.workflowId} • {formatTime(inv.timestamp)}
                  </Typography>
                </Box>
                <Chip label={inv.stepType} size="small" color="secondary" />
                <Chip label={`${inv.latencyMs}ms`} size="small" variant="outlined" />
                <Chip label={`${inv.tokensUsed} tokens`} size="small" variant="outlined" />
                {inv.policyViolations.length > 0 && (
                  <Chip icon={<WarningIcon />} label="Violations" size="small" color="error" />
                )}
                {inv.safetyFlags.length > 0 && inv.policyViolations.length === 0 && (
                  <Chip icon={<CheckCircleIcon />} label="Compliant" size="small" color="success" />
                )}
              </Stack>
            </AccordionSummary>
            <AccordionDetails>
              <Tabs value={selectedTab} onChange={(_, v) => setSelectedTab(v)} sx={{ mb: 2 }}>
                <Tab label="Prompt" />
                <Tab label="Input" />
                <Tab label="Output" />
                <Tab label="Metadata" />
              </Tabs>

              {selectedTab === 0 && (
                <Stack spacing={2}>
                  <CodeBlock code={inv.promptTemplate} title="Prompt Template" />
                  <CodeBlock code={inv.promptFilled} title="Filled Prompt" />
                </Stack>
              )}

              {selectedTab === 1 && (
                <CodeBlock
                  code={JSON.stringify(inv.inputSnapshot, null, 2)}
                  title="Input Snapshot"
                />
              )}

              {selectedTab === 2 && (
                <Stack spacing={2}>
                  <CodeBlock code={inv.outputRaw} title="Raw Output" />
                  <CodeBlock
                    code={JSON.stringify(inv.outputProcessed, null, 2)}
                    title="Processed Output"
                  />
                </Stack>
              )}

              {selectedTab === 3 && (
                <Stack spacing={2}>
                  <Stack direction="row" spacing={4}>
                    <Box>
                      <Typography variant="caption" color="text.secondary">Model</Typography>
                      <Typography>{inv.modelName}</Typography>
                    </Box>
                    <Box>
                      <Typography variant="caption" color="text.secondary">Profile</Typography>
                      <Typography>{inv.profileId}</Typography>
                    </Box>
                    <Box>
                      <Typography variant="caption" color="text.secondary">Latency</Typography>
                      <Typography>{inv.latencyMs}ms</Typography>
                    </Box>
                    <Box>
                      <Typography variant="caption" color="text.secondary">Tokens</Typography>
                      <Typography>{inv.tokensUsed}</Typography>
                    </Box>
                  </Stack>
                  <Divider />
                  <Box>
                    <Typography variant="caption" color="text.secondary">Safety Checks</Typography>
                    <Stack direction="row" spacing={1} sx={{ mt: 0.5 }}>
                      {inv.safetyFlags.length > 0 ? (
                        inv.safetyFlags.map((flag, i) => (
                          <Chip key={i} label={flag} size="small" color="success" />
                        ))
                      ) : (
                        <Typography variant="body2" color="text.secondary">None configured</Typography>
                      )}
                    </Stack>
                  </Box>
                  {inv.policyViolations.length > 0 && (
                    <Alert severity="error">
                      <Typography fontWeight={500}>Policy Violations</Typography>
                      {inv.policyViolations.map((v, i) => (
                        <Typography key={i}>• {v}</Typography>
                      ))}
                    </Alert>
                  )}
                </Stack>
              )}
            </AccordionDetails>
          </Accordion>
        ))}
      </Stack>

      {filteredInvocations.length === 0 && !loading && (
        <Alert severity="info" sx={{ mt: 2 }}>
          No LLM invocations found matching your search.
        </Alert>
      )}
    </Box>
  );
};

export default LLMReasoningInspector;
