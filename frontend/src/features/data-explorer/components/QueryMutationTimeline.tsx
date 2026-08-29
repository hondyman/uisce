import React from 'react';
import { Box, Typography, Paper } from '@mui/material';
import { useTheme } from '@mui/material/styles';
import {
  FilterList as FilterIcon,
  ArrowDownward as ArrowDownCircleIcon,
  Sync as RepeatIcon,
  AddCircle as PlusCircleIcon,
  AutoAwesome as SparklesIcon,
  Chat as MessageSquareIcon,
  RemoveCircle as MinusCircleIcon,
  Storage as DatabaseIcon,
} from '@mui/icons-material';
import { ChatMessage, MutationIntent } from '../types/explorerTypes';

interface QueryMutationTimelineProps {
  messages: ChatMessage[];
}

export const QueryMutationTimeline: React.FC<QueryMutationTimelineProps> = ({ messages }) => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';

  if (messages.length === 0) {
    return (
      <Box sx={{ p: 3, textAlign: 'center' }}>
        <Typography variant="caption" sx={{ color: isDark ? '#64748B' : '#94A3B8' }}>
          No conversational mutations yet. Ask a question or drill into the chart to see events.
        </Typography>
      </Box>
    );
  }

  const getIntentConfig = (intent?: MutationIntent) => {
    switch (intent) {
      case 'new_query':
        return { icon: <DatabaseIcon sx={{ width: 14, height: 14 }} />, color: '#38BDF8', label: 'Generated New Query' };
      case 'drill_down':
        return { icon: <ArrowDownCircleIcon sx={{ width: 14, height: 14 }} />, color: '#A78BFA', label: 'Drilled Down' };
      case 'drill_across':
        return { icon: <RepeatIcon sx={{ width: 14, height: 14 }} />, color: '#F472B6', label: 'Swapped Dimension' };
      case 'add_filter':
        return { icon: <FilterIcon sx={{ width: 14, height: 14 }} />, color: '#2DD4BF', label: 'Applied Filter' };
      case 'add_measure':
        return { icon: <PlusCircleIcon sx={{ width: 14, height: 14 }} />, color: '#10B981', label: 'Added Metric' };
      case 'remove_element':
        return { icon: <MinusCircleIcon sx={{ width: 14, height: 14 }} />, color: '#FB923C', label: 'Removed Element' };
      default:
        return { icon: <SparklesIcon sx={{ width: 14, height: 14 }} />, color: '#94A3B8', label: 'Updated View' };
    }
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5, p: 2 }}>
      {messages.map((msg) => {
        const isUser = msg.role === 'user';
        const config = !isUser ? getIntentConfig(msg.mutationIntent) : null;

        return (
          <Box
            key={msg.id}
            sx={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: isUser ? 'flex-end' : 'flex-start',
              width: '100%',
            }}
          >
            {/* User Prompt Bubble */}
            {isUser && (
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, maxWidth: '85%' }}>
                <Paper
                  elevation={0}
                  sx={{
                    p: 1.2,
                    bgcolor: isDark ? 'rgba(56, 189, 248, 0.1)' : '#E0F2FE',
                    color: isDark ? '#E2E8F0' : '#0F172A',
                    border: `1px solid ${isDark ? 'rgba(56, 189, 248, 0.2)' : 'transparent'}`,
                    borderRadius: '14px 14px 2px 14px',
                  }}
                >
                  <Typography variant="body2" sx={{ fontSize: '0.78rem', lineHeight: 1.35 }}>
                    "{msg.content}"
                  </Typography>
                </Paper>
                <MessageSquareIcon sx={{ width: 15, height: 15, color: isDark ? '#64748B' : '#94A3B8' }} />
              </Box>
            )}

            {/* AI Action Indicator */}
            {!isUser && config && (
              <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1.2, maxWidth: '90%', mt: 0.5 }}>
                <Box
                  sx={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    minWidth: 26,
                    height: 26,
                    borderRadius: '50%',
                    bgcolor: `${config.color}20`,
                    color: config.color,
                  }}
                >
                  {config.icon}
                </Box>
                <Box>
                  <Typography
                    variant="caption"
                    sx={{
                      fontWeight: 800,
                      color: config.color,
                      display: 'block',
                      fontSize: '0.7rem',
                      letterSpacing: 0.3,
                      textTransform: 'uppercase',
                    }}
                  >
                    {config.label}
                  </Typography>
                  <Typography
                    variant="body2"
                    sx={{ color: isDark ? '#94A3B8' : '#64748B', fontSize: '0.75rem', lineHeight: 1.35 }}
                  >
                    {msg.content}
                  </Typography>
                </Box>
              </Box>
            )}
          </Box>
        );
      })}
    </Box>
  );
};

export default QueryMutationTimeline;
