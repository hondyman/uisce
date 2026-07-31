import React, { useState } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Divider,
  Paper,
  Stack,
  Typography,
  Chip,
  Alert,
} from '@mui/material';
import CodeIcon from '@mui/icons-material/Code';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';

interface BOValidationRuleBindingPreviewProps {
  businessObjectId?: string;
  tenantId?: string;
  ruleType: string;
  conditionJson: Record<string, unknown>;
}

export const BOValidationRuleBindingPreview: React.FC<BOValidationRuleBindingPreviewProps> = ({
  businessObjectId,
  tenantId,
  ruleType,
  conditionJson,
}) => {
  const [loading, setLoading] = useState(false);
  const [compiledResult, setCompiledResult] = useState<{
    status: string;
    compiled_sql: string;
    args: unknown[];
    physical_column: string;
    execution_time: string;
  } | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleCompileBinding = async () => {
    if (!businessObjectId) {
      setError('A Business Object must be selected to compile binding SQL.');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const resp = await fetch('/api/v1/validation-rules/execute-binding', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(tenantId ? { 'X-Tenant-ID': tenantId } : {}),
        },
        body: JSON.stringify({
          businessObjectId,
          tenantId,
          ruleType,
          conditionJson,
        }),
      });

      if (!resp.ok) {
        const errData = await resp.json().catch(() => ({ message: resp.statusText }));
        throw new Error(errData.message || 'Failed to compile binding SQL');
      }

      const data = await resp.json();
      setCompiledResult(data);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Compilation failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card variant="outlined" sx={{ mt: 2, borderColor: 'divider' }}>
      <CardContent>
        <Stack direction="row" justifyContent="space-between" alignItems="center" mb={1}>
          <Typography variant="subtitle1" fontWeight="600" display="flex" alignItems="center" gap={1}>
            <CodeIcon fontSize="small" color="primary" /> Binding SQL Execution Preview
          </Typography>
          <Button
            size="small"
            variant="contained"
            startIcon={loading ? <CircularProgress color="inherit" sx={{ fontSize: 16 }}/> : <PlayArrowIcon />}
            onClick={handleCompileBinding}
            disabled={loading || !businessObjectId}
          >
            {loading ? 'Compiling...' : 'Test Binding Translation'}
          </Button>
        </Stack>

        <Typography variant="caption" color="text.secondary" paragraph>
          Semantic validation conditions are compiled into physical table target predicates using active binding metadata.
        </Typography>

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {compiledResult && (
          <Box sx={{ mt: 2 }}>
            <Stack direction="row" spacing={1} mb={1}>
              <Chip
                icon={<CheckCircleOutlineIcon />}
                label={`Physical Target: ${compiledResult.physical_column}`}
                size="small"
                color="primary"
                variant="outlined"
              />
              <Chip
                label={`Compilation Time: ${compiledResult.execution_time}`}
                size="small"
                variant="outlined"
              />
            </Stack>

            <Divider sx={{ my: 1 }} />

            <Paper
              elevation={0}
              sx={{
                p: 1.5,
                bgcolor: 'grey.900',
                color: 'grey.100',
                fontFamily: 'monospace',
                fontSize: '0.825rem',
                borderRadius: 1,
                overflowX: 'auto',
              }}
            >
              <Typography variant="caption" sx={{ color: 'grey.500', display: 'block', mb: 0.5 }}>
                // Physical Binding SQL Query
              </Typography>
              <code>{compiledResult.compiled_sql}</code>
            </Paper>

            {compiledResult.args && compiledResult.args.length > 0 && (
              <Box sx={{ mt: 1 }}>
                <Typography variant="caption" color="text.secondary">
                  SQL Parameters: {JSON.stringify(compiledResult.args)}
                </Typography>
              </Box>
            )}
          </Box>
        )}
      </CardContent>
    </Card>
  );
};
