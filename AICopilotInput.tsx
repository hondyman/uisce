import React, { useState } from 'react';
import { TextField, Button, CircularProgress, Box, Typography } from '@mui/material';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';

interface CopilotInputProps {
  tenantId: string;
  onDraftGenerated: (draft: any) => void;
}

export const AICopilotInput: React.FC<CopilotInputProps> = ({ tenantId, onDraftGenerated }) => {
  const [prompt, setPrompt] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleDraft = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch('/api/v1/ai/copilot/draft-bo', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tenant_id: tenantId, prompt: prompt }),
      });
      
      if (!response.ok) {
        const text = await response.text();
        throw new Error(text || "Failed to draft BO");
      }
      
      const draft = await response.json();
      onDraftGenerated(draft); // Hydrates the wizard state
    } catch (err: any) {
      setError(err.message || "An unexpected error occurred");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box sx={{ p: 2, border: '1px solid #e0e0e0', borderRadius: 2, bgcolor: '#f9f9f9', mb: 3 }}>
      <Typography variant="subtitle2" sx={{ mb: 1, display: 'flex', alignItems: 'center', fontWeight: 'bold' }}>
        <AutoAwesomeIcon sx={{ mr: 1, color: 'primary.main' }} /> AI Model Copilot
      </Typography>
      <Typography variant="body2" sx={{ mb: 2, color: 'text.secondary' }}>
        Type in a natural language prompt to describe the Business Object you want to create (e.g. including the table name).
      </Typography>
      <TextField
        fullWidth
        multiline
        rows={2}
        placeholder="e.g., Create a BO for Monthly Revenue by Region using the sales_ledger table"
        value={prompt}
        onChange={(e) => setPrompt(e.target.value)}
        sx={{ mb: 2 }}
      />
      {error && (
        <Typography variant="caption" sx={{ color: 'error.main', display: 'block', mb: 2 }}>
          {error}
        </Typography>
      )}
      <Button 
        variant="contained" 
        onClick={handleDraft} 
        disabled={loading || !prompt.trim()}
        startIcon={loading ? <CircularProgress size={20} color="inherit" /> : <AutoAwesomeIcon />}
      >
        Draft Business Object
      </Button>
    </Box>
  );
};
