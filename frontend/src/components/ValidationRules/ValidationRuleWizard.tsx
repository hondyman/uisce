import React, { useState } from 'react';
import { Button, TextField, CircularProgress, Alert, Paper, Typography, Box } from '@mui/material';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import { devLog } from '../../utils/devLogger';

interface ValidationRuleWizardProps {
  onScriptGenerated: (script: string) => void;
  targetEntity: string;
}

export const ValidationRuleWizard: React.FC<ValidationRuleWizardProps> = ({
  onScriptGenerated,
  targetEntity
}) => {
  const [prompt, setPrompt] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [lastExplanation, setLastExplanation] = useState<string | null>(null);

  const handleGenerate = async () => {
    if (!prompt.trim()) return;

    setLoading(true);
    setError(null);
    setLastExplanation(null);

    try {
      const response = await fetch('/api/ai/generate-validation', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          prompt: prompt,
          entity: targetEntity
        }),
      });

      if (!response.ok) {
        throw new Error(`Generation failed: ${response.statusText}`);
      }

      const data = await response.json();
      devLog('Generated script:', data);
      
      onScriptGenerated(data.script);
      setLastExplanation(data.explanation);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Paper elevation={0} sx={{ p: 2, mb: 2, background: 'rgba(30, 30, 30, 0.5)', border: '1px solid rgba(255, 255, 255, 0.1)' }}>
      <Box display="flex" alignItems="center" gap={1} mb={2}>
        <AutoAwesomeIcon color="primary" />
        <Typography variant="h6" color="primary">
          AI Validation Assistant
        </Typography>
      </Box>

      <Typography variant="body2" color="text.secondary" paragraph>
        Describe your validation logic in plain English, and I'll write the Starlark script for you.
      </Typography>

      <Box display="flex" gap={1}>
        <TextField
          fullWidth
          size="small"
          placeholder="e.g., Validate that the age is over 18 or verify email format"
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          disabled={loading}
          onKeyPress={(e) => e.key === 'Enter' && handleGenerate()}
        />
        <Button
          variant="contained"
          onClick={handleGenerate}
          disabled={loading || !prompt.trim()}
          startIcon={loading ? <CircularProgress size={20} color="inherit" /> : <AutoAwesomeIcon />}
        >
          {loading ? 'Generating...' : 'Generate'}
        </Button>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mt: 2 }}>
          {error}
        </Alert>
      )}

      {lastExplanation && (
        <Alert severity="success" sx={{ mt: 2 }}>
          {lastExplanation}
        </Alert>
      )}
    </Paper>
  );
};
