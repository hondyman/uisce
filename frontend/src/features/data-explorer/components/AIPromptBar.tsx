import React, { useState } from 'react';
import {
  Box,
  Paper,
  TextField,
  Button,
  Chip,
  Typography,
  CircularProgress,
  Alert,
} from '@mui/material';
import { Sparkles, ArrowRight, CornerDownRight, RotateCcw } from 'lucide-react';
import { useExplorerTheme } from '../hooks/useExplorerTheme';

interface AIPromptBarProps {
  onSendPrompt: (prompt: string) => void;
  isLoading: boolean;
  suggestedFollowUps: string[];
  error?: string | null;
  onResetChat?: () => void;
}

export const AIPromptBar: React.FC<AIPromptBarProps> = ({
  onSendPrompt,
  isLoading,
  suggestedFollowUps,
  error,
  onResetChat,
}) => {
  const theme = useExplorerTheme();
  const [inputVal, setInputVal] = useState('');

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      if (inputVal.trim() && !isLoading) {
        onSendPrompt(inputVal);
        setInputVal('');
      }
    }
  };

  const handleFollowUpClick = (chipText: string) => {
    if (!isLoading) {
      onSendPrompt(chipText);
    }
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, p: 2, pb: 1, bgcolor: theme.background, borderBottom: `1px solid ${theme.border}` }}>
      <Paper
        elevation={0}
        sx={{
          p: '4px 12px',
          display: 'flex',
          alignItems: 'center',
          gap: 1.5,
          bgcolor: theme.backgroundElevated,
          border: `1px solid ${theme.border}`,
          borderRadius: 2,
        }}
      >
        <Sparkles size={18} color={theme.accent} />
        <TextField
          fullWidth
          variant="standard"
          placeholder="Ask anything or request changes (e.g. 'Show total valuation by client for active accounts in 2026')..."
          value={inputVal}
          onChange={(e) => setInputVal(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={isLoading}
          InputProps={{
            disableUnderline: true,
            sx: {
              fontSize: '0.85rem',
              color: theme.text,
            },
          }}
        />
        <Button
          variant="contained"
          size="small"
          onClick={() => {
            if (inputVal.trim() && !isLoading) {
              onSendPrompt(inputVal);
              setInputVal('');
            }
          }}
          disabled={isLoading || !inputVal.trim()}
          endIcon={isLoading ? <CircularProgress size={13} color="inherit" /> : <ArrowRight size={14} />}
          sx={{
            bgcolor: theme.accent,
            color: theme.isDark ? theme.background : '#FFFFFF',
            textTransform: 'none',
            fontWeight: 700,
            fontSize: '0.75rem',
            px: 2,
            borderRadius: 1.5,
            whiteSpace: 'nowrap',
            '&:hover': { bgcolor: theme.accentDark },
          }}
        >
          {isLoading ? 'Synthesizing...' : 'Ask AI'}
        </Button>
      </Paper>

      {suggestedFollowUps.length > 0 && (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.8, flexWrap: 'wrap', px: 0.5 }}>
          <Typography
            variant="caption"
            sx={{
              color: theme.textMuted,
              fontWeight: 700,
              display: 'flex',
              alignItems: 'center',
              gap: 0.4,
              fontSize: '0.7rem',
            }}
          >
            <CornerDownRight size={12} /> Suggested:
          </Typography>
          {suggestedFollowUps.map((suggestion, idx) => (
            <Chip
              key={idx}
              size="small"
              label={suggestion}
              onClick={() => handleFollowUpClick(suggestion)}
              disabled={isLoading}
              sx={{
                fontSize: '0.68rem',
                fontWeight: 600,
                cursor: 'pointer',
                bgcolor: theme.accentMuted,
                color: theme.accent,
                border: `1px solid ${theme.border}`,
                '&:hover': {
                  bgcolor: theme.accent,
                  color: theme.isDark ? theme.background : '#FFFFFF',
                },
              }}
            />
          ))}
          {onResetChat && (
            <Button
              size="small"
              variant="text"
              onClick={onResetChat}
              startIcon={<RotateCcw size={11} />}
              sx={{
                ml: 'auto',
                fontSize: '0.68rem',
                textTransform: 'none',
                color: theme.textMuted,
                '&:hover': { color: theme.text },
              }}
            >
              Reset Context
            </Button>
          )}
        </Box>
      )}

      {error && (
        <Alert severity="error" sx={{ py: 0.5, fontSize: '0.75rem', borderRadius: 1.5 }}>
          {error}
        </Alert>
      )}
    </Box>
  );
};

export default AIPromptBar;
