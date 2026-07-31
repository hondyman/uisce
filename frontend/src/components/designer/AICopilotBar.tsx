import React, { useState } from 'react';
import {
  Paper,
  InputBase,
  IconButton,
  CircularProgress,
  Box,
  Alert,
  Tooltip
} from '@mui/material';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import SendIcon from '@mui/icons-material/Send';
import { PageLayoutSpec } from '../../types/pageDesigner';

interface AICopilotBarProps {
  onLayoutGenerated: (spec: PageLayoutSpec) => void;
  domain?: 'MDM' | 'ACCOUNTING' | 'ORDER_MGMT' | 'PORTFOLIO';
}

export const AICopilotBar: React.FC<AICopilotBarProps> = ({
  onLayoutGenerated,
  domain = 'PORTFOLIO',
}) => {
  const [prompt, setPrompt] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    if (!prompt.trim()) return;

    setLoading(true);
    setError(null);

    try {
      const res = await fetch('/api/ai/generate-page', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ prompt, domain }),
      });

      if (!res.ok) {
        throw new Error('Failed to generate AI page layout');
      }

      const generatedSpec: PageLayoutSpec = await res.json();
      onLayoutGenerated(generatedSpec);
      setPrompt('');
    } catch (err: any) {
      setError(err.message || 'AI Copilot request failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box sx={{ mb: 3 }}>
      <Paper
        component="form"
        onSubmit={handleSubmit}
        sx={{
          p: '4px 12px',
          display: 'flex',
          alignItems: 'center',
          bgcolor: '#1e293b',
          border: '1px solid #38bdf8',
          borderRadius: 2,
          boxShadow: '0 4px 20px rgba(56, 189, 248, 0.15)',
        }}
      >
        <AutoAwesomeIcon sx={{ color: '#38bdf8', mr: 1 }} />
        <InputBase
          sx={{ ml: 1, flex: 1, color: '#f8fafc', fontSize: '14px' }}
          placeholder="AI Page Copilot: Type e.g., 'Build an executive dashboard showing YTD sales by region and active customer accounts'..."
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          disabled={loading}
        />
        {loading ? (
            <CircularProgress sx={{ color: '#38bdf8', mx: 1, fontSize: 24 }}/>
        ) : (
          <Tooltip title="Generate Page via Gemini AI">
            <IconButton type="submit" sx={{ color: '#38bdf8' }} disabled={!prompt.trim()}>
              <SendIcon />
            </IconButton>
          </Tooltip>
        )}
      </Paper>

      {error && (
        <Alert severity="error" sx={{ mt: 1 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}
    </Box>
  );
};
