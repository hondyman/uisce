import React, { useEffect, useRef, useState } from 'react';
import {
  Box,
  Paper,
  Stack,
  TextField,
  IconButton,
  Typography,
  Tooltip,
  CircularProgress,
  Button,
  Alert,
  Chip,
} from '@mui/material';
import {
  Send as SendIcon,
  AutoAwesome as SparklesIcon,
  ContentCopy as CopyIcon,
  Check as CheckIcon,
} from '@mui/icons-material';
import type { Conversation, ConversationMessage } from '../types/conversationTypes';
import { useExplorerTheme } from '../hooks/useExplorerTheme';

interface ConversationThreadProps {
  conversation: Conversation | null;
  isProcessing: boolean;
  error?: string | null;
  onSend: (content: string) => void;
  onApplyQueryState?: (state: Conversation['queryState']) => void;
}

function formatTime(iso?: string): string {
  if (!iso) return '';
  return new Date(iso).toLocaleTimeString(undefined, {
    hour: '2-digit',
    minute: '2-digit',
  });
}

function MessageBubble({
  message,
  theme,
}: {
  message: ConversationMessage;
  theme: ReturnType<typeof useExplorerTheme>;
}) {
  const isUser = message.role === 'user';
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      if (navigator?.clipboard?.writeText) {
        await navigator.clipboard.writeText(message.content);
      }
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // ignore
    }
  };

  return (
    <Stack
      direction="row"
      justifyContent={isUser ? 'flex-end' : 'flex-start'}
      sx={{ mb: 1.5 }}
    >
      <Box sx={{ maxWidth: '85%' }}>
        <Paper
          elevation={0}
          sx={{
            p: 1.5,
            borderRadius: 2,
            bgcolor: isUser ? theme.accent : theme.backgroundElevated,
            color: theme.text,
            border: isUser ? 'none' : `1px solid ${theme.border}`,
          }}
        >
          <Typography
            variant="body2"
            sx={{ whiteSpace: 'pre-wrap', lineHeight: 1.5 }}
          >
            {message.content}
          </Typography>
          {message.generatedSql && (
            <Box
              sx={{
                mt: 1,
                p: 1.25,
                borderRadius: 1.5,
                bgcolor: theme.background,
                color: theme.info,
                fontFamily: 'monospace',
                fontSize: 12,
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
              }}
            >
              {message.generatedSql}
            </Box>
          )}
          <Stack direction="row" spacing={1} alignItems="center" sx={{ mt: 0.75 }}>
            <Typography variant="caption" sx={{ color: isUser ? theme.text : theme.textMuted, opacity: isUser ? 0.7 : 1 }}>
              {formatTime(message.timestamp)}
            </Typography>
            {typeof message.confidence === 'number' && (
              <Chip
                size="small"
                label={`confidence ${(message.confidence * 100).toFixed(0)}%`}
                sx={{
                  height: 18,
                  fontSize: 10,
                  fontWeight: 700,
                  bgcolor: isUser ? theme.text : theme.background,
                  color: theme.text,
                }}
              />
            )}
          </Stack>
        </Paper>
        <Stack direction="row" spacing={0.5} justifyContent={isUser ? 'flex-end' : 'flex-start'} sx={{ mt: 0.5 }}>
          <Tooltip title={copied ? 'Copied' : 'Copy'}>
            <IconButton size="small" onClick={handleCopy} sx={{ color: theme.textMuted }}>
              {copied ? <CheckIcon sx={{ fontSize: 14, color: theme.success }} /> : <CopyIcon sx={{ fontSize: 14 }} />}
            </IconButton>
          </Tooltip>
        </Stack>
      </Box>
    </Stack>
  );
}

export const ConversationThread: React.FC<ConversationThreadProps> = ({
  conversation,
  isProcessing,
  error,
  onSend,
}) => {
  const theme = useExplorerTheme();
  const [draft, setDraft] = useState('');
  const scrollRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [conversation?.messages.length, isProcessing]);

  const handleSend = () => {
    const trimmed = draft.trim();
    if (!trimmed || isProcessing) return;
    onSend(trimmed);
    setDraft('');
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  if (!conversation) {
    return (
      <Paper
        elevation={0}
        sx={{
          flex: 1,
          m: 2,
          p: 4,
          borderRadius: 3,
          border: `1px dashed ${theme.border}`,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          bgcolor: theme.background,
        }}
      >
        <SparklesIcon sx={{ fontSize: 32, color: theme.textMuted, mb: 1 }} />
        <Typography variant="subtitle1" sx={{ color: theme.text, fontWeight: 700 }}>
          Start a conversation
        </Typography>
        <Typography variant="body2" sx={{ color: theme.textMuted, textAlign: 'center', maxWidth: 360 }}>
          Pick one from the rail or hit "New conversation". Ask in natural language — the explorer will translate it
          into dimensions and measures for the active Business Object.
        </Typography>
      </Paper>
    );
  }

  return (
    <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', bgcolor: theme.background }}>
      <Box sx={{ px: 2, py: 1.25, borderBottom: `1px solid ${theme.border}`, bgcolor: theme.backgroundElevated }}>
        <Stack direction="row" justifyContent="space-between" alignItems="center">
          <Stack direction="row" spacing={1} alignItems="center">
            <Box
              sx={{
                width: 28,
                height: 28,
                borderRadius: 1.5,
                bgcolor: theme.background,
                color: theme.text,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <SparklesIcon sx={{ fontSize: 16 }} />
            </Box>
            <Box>
              <Typography variant="subtitle2" sx={{ fontWeight: 700, color: theme.text }}>
                {conversation.title}
              </Typography>
              <Typography variant="caption" sx={{ color: theme.textMuted }}>
                {conversation.messages.length} messages · {isProcessing ? 'thinking…' : 'idle'}
              </Typography>
            </Box>
          </Stack>
        </Stack>
      </Box>

      <Box ref={scrollRef} sx={{ flex: 1, overflow: 'auto', px: 2, py: 2 }}>
        {conversation.messages.length === 0 && (
          <Box sx={{ textAlign: 'center', py: 6, color: theme.textMuted }}>
            <Typography variant="body2">
              No messages yet. Ask something like "show total revenue by region for last quarter".
            </Typography>
          </Box>
        )}
        {conversation.messages.map((message) => (
          <MessageBubble key={message.id} message={message} theme={theme} />
        ))}
        {isProcessing && (
          <Stack direction="row" spacing={1} alignItems="center" sx={{ color: theme.textMuted, px: 1 }}>
            <CircularProgress size={14} sx={{ color: theme.accent }} />
            <Typography variant="caption">Generating query…</Typography>
          </Stack>
        )}
        {error && (
          <Alert severity="error" sx={{ mt: 2 }}>
            {error}
          </Alert>
        )}
      </Box>

      <Box sx={{ p: 2, borderTop: `1px solid ${theme.border}`, bgcolor: theme.backgroundElevated }}>
        <Stack direction="row" spacing={1} alignItems="flex-end">
          <TextField
            fullWidth
            multiline
            maxRows={4}
            placeholder="Ask the explorer…"
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={handleKeyDown}
            sx={{
              '& .MuiOutlinedInput-root': {
                borderRadius: 3,
                bgcolor: theme.background,
                color: theme.text,
                '& fieldset': { borderColor: theme.border },
                '&:hover fieldset': { borderColor: theme.textMuted },
                '&.Mui-focused fieldset': { borderColor: theme.accent },
              },
              '& .MuiInputBase-input::placeholder': {
                color: theme.textMuted,
                opacity: 1,
              },
            }}
          />
          <Button
            variant="contained"
            onClick={handleSend}
            disabled={!draft.trim() || isProcessing}
            startIcon={<SendIcon />}
            sx={{
              bgcolor: theme.accent,
              color: theme.isDark ? theme.background : '#FFFFFF',
              borderRadius: 999,
              textTransform: 'none',
              fontWeight: 700,
              px: 3,
              '&:hover': { bgcolor: theme.accentDark },
            }}
          >
            Send
          </Button>
        </Stack>
      </Box>
    </Box>
  );
};

export default ConversationThread;
