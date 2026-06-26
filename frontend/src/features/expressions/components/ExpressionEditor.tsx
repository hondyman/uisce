import React, { useState } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Grid,
  Chip,
  Alert,
  Paper,
  Tabs,
  Tab,
  IconButton,
  Tooltip,
} from '@mui/material';
import {
  PlayArrow as RunIcon,
  Save as SaveIcon,
  Code as CodeIcon,
  BugReport as TestIcon,
} from '@mui/icons-material';
import Editor from '@monaco-editor/react';

// ============================================================================
// EXPRESSION EDITOR
// Starlark expression editor with syntax highlighting and testing
// ============================================================================

interface ExpressionEditorProps {
  initialExpression?: {
    id?: string;
    name: string;
    rule_type: 'validation' | 'calculation' | 'condition';
    script: string;
    description?: string;
  };
  onSave?: (expression: any) => void;
}

export const ExpressionEditor: React.FC<ExpressionEditorProps> = ({
  initialExpression,
  onSave,
}) => {
  const [name, setName] = useState(initialExpression?.name || '');
  const [ruleType, setRuleType] = useState<'validation' | 'calculation' | 'condition'>(
    initialExpression?.rule_type || 'calculation'
  );
  const [script, setScript] = useState(initialExpression?.script || '');
  const [description, setDescription] = useState(initialExpression?.description || '');
  const [testData, setTestData] = useState('{\n  "amount": 1000,\n  "limit": 5000\n}');
  const [testResult, setTestResult] = useState<any>(null);
  const [activeTab, setActiveTab] = useState(0);

  const handleTest = async () => {
    try {
      const data = JSON.parse(testData);
      // TODO: Call API to test expression
      // const response = await fetch('/api/expressions/test', {
      //   method: 'POST',
      //   body: JSON.stringify({ script, data, rule_type: ruleType }),
      // });
      // setTestResult(await response.json());
      
      // Mock result
      setTestResult({
        success: true,
        result: ruleType === 'validation' 
          ? { is_valid: true, message: 'Validation passed' }
          : ruleType === 'calculation'
          ? 42.5
          : 'auto_approve',
      });
    } catch (error) {
      setTestResult({ success: false, error: String(error) });
    }
  };

  const handleSave = () => {
    if (onSave) {
      onSave({
        id: initialExpression?.id,
        name,
        rule_type: ruleType,
        script,
        description,
      });
    }
  };

  const getExampleScript = () => {
    switch (ruleType) {
      case 'validation':
        return `# Validation Rule
# Returns (is_valid, message) tuple

if amount > limit:
    result = (False, "Amount exceeds limit of $" + str(limit))
else:
    result = (True, None)`;

      case 'calculation':
        return `# Calculation
# Returns computed value

result = (amount - cost_basis) / cost_basis * 100`;

      case 'condition':
        return `# Condition Rule
# Returns action string

if amount > 1000000:
    result = "committee_review"
elif amount > 100000:
    result = "manager_approval"
else:
    result = "auto_approve"`;

      default:
        return '';
    }
  };

  return (
    <Box>
      <Card>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Expression Editor
          </Typography>

          <Grid container spacing={2} sx={{ mb: 2 }}>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Expression Name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g., Transaction Amount Limit"
              />
            </Grid>
            <Grid item xs={12} md={6}>
              <FormControl fullWidth>
                <InputLabel>Expression Type</InputLabel>
                <Select
                  value={ruleType}
                  onChange={(e) => setRuleType(e.target.value as any)}
                  label="Expression Type"
                >
                  <MenuItem value="validation">Validation</MenuItem>
                  <MenuItem value="calculation">Calculation</MenuItem>
                  <MenuItem value="condition">Condition</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                multiline
                rows={2}
              />
            </Grid>
          </Grid>

          <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}>
            <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)}>
              <Tab label="Script" icon={<CodeIcon />} iconPosition="start" />
              <Tab label="Test" icon={<TestIcon />} iconPosition="start" />
            </Tabs>
          </Box>

          {activeTab === 0 && (
            <Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                <Typography variant="subtitle2">Starlark Script</Typography>
                <Button
                  size="small"
                  onClick={() => setScript(getExampleScript())}
                >
                  Load Example
                </Button>
              </Box>
              <Paper variant="outlined" sx={{ mb: 2 }}>
                <Editor
                  height="300px"
                  defaultLanguage="python"
                  value={script}
                  onChange={(value) => setScript(value || '')}
                  theme="vs-dark"
                  options={{
                    minimap: { enabled: false },
                    fontSize: 13,
                    lineNumbers: 'on',
                    scrollBeyondLastLine: false,
                  }}
                />
              </Paper>

              <Alert severity="info" sx={{ mb: 2 }}>
                <Typography variant="subtitle2" gutterBottom>
                  Available Context Variables:
                </Typography>
                <Typography variant="body2" component="div">
                  • Access data via <code>context.field_name</code> or directly as <code>field_name</code>
                  <br />
                  • Built-in functions: <code>abs, min, max, round, len, str, int, float, bool</code>
                  <br />
                  • Set <code>result</code> variable with your return value
                </Typography>
              </Alert>
            </Box>
          )}

          {activeTab === 1 && (
            <Box>
              <Typography variant="subtitle2" gutterBottom>
                Test Data (JSON)
              </Typography>
              <Paper variant="outlined" sx={{ mb: 2 }}>
                <Editor
                  height="200px"
                  defaultLanguage="json"
                  value={testData}
                  onChange={(value) => setTestData(value || '')}
                  theme="vs-dark"
                  options={{
                    minimap: { enabled: false },
                    fontSize: 13,
                    lineNumbers: 'on',
                  }}
                />
              </Paper>

              <Button
                variant="contained"
                startIcon={<RunIcon />}
                onClick={handleTest}
                sx={{ mb: 2 }}
              >
                Run Test
              </Button>

              {testResult && (
                <Alert
                  severity={testResult.success ? 'success' : 'error'}
                  sx={{ mb: 2 }}
                >
                  <Typography variant="subtitle2" gutterBottom>
                    Test Result:
                  </Typography>
                  <Paper
                    variant="outlined"
                    sx={{
                      p: 1,
                      bgcolor: 'background.default',
                      fontFamily: 'monospace',
                      fontSize: '0.875rem',
                    }}
                  >
                    <pre style={{ margin: 0 }}>
                      {JSON.stringify(testResult.result || testResult.error, null, 2)}
                    </pre>
                  </Paper>
                </Alert>
              )}
            </Box>
          )}

          <Box sx={{ display: 'flex', gap: 1, justifyContent: 'flex-end', mt: 2 }}>
            <Button variant="outlined" onClick={() => setActiveTab(1)}>
              Test
            </Button>
            <Button
              variant="contained"
              startIcon={<SaveIcon />}
              onClick={handleSave}
              disabled={!name || !script}
            >
              Save Expression
            </Button>
          </Box>
        </CardContent>
      </Card>

      {/* Expression Examples */}
      <Card sx={{ mt: 2 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Expression Examples
          </Typography>
          <Grid container spacing={2}>
            <Grid item xs={12} md={4}>
              <Paper variant="outlined" sx={{ p: 2 }}>
                <Chip label="Validation" size="small" color="primary" sx={{ mb: 1 }} />
                <Typography variant="subtitle2" gutterBottom>
                  Amount Limit Check
                </Typography>
                <Typography
                  variant="body2"
                  sx={{ fontFamily: 'monospace', fontSize: '0.75rem', color: 'text.secondary' }}
                >
                  {`if amount > limit:\n  result = (False, "Exceeds")\nelse:\n  result = (True, None)`}
                </Typography>
              </Paper>
            </Grid>
            <Grid item xs={12} md={4}>
              <Paper variant="outlined" sx={{ p: 2 }}>
                <Chip label="Calculation" size="small" color="success" sx={{ mb: 1 }} />
                <Typography variant="subtitle2" gutterBottom>
                  Portfolio Return %
                </Typography>
                <Typography
                  variant="body2"
                  sx={{ fontFamily: 'monospace', fontSize: '0.75rem', color: 'text.secondary' }}
                >
                  {`result = ((current - cost) / cost * 100) if cost > 0 else 0`}
                </Typography>
              </Paper>
            </Grid>
            <Grid item xs={12} md={4}>
              <Paper variant="outlined" sx={{ p: 2 }}>
                <Chip label="Condition" size="small" color="warning" sx={{ mb: 1 }} />
                <Typography variant="subtitle2" gutterBottom>
                  Approval Routing
                </Typography>
                <Typography
                  variant="body2"
                  sx={{ fontFamily: 'monospace', fontSize: '0.75rem', color: 'text.secondary' }}
                >
                  {`if amount > 1000000:\n  result = "committee"\nelse:\n  result = "auto"`}
                </Typography>
              </Paper>
            </Grid>
          </Grid>
        </CardContent>
      </Card>
    </Box>
  );
};

export default ExpressionEditor;
