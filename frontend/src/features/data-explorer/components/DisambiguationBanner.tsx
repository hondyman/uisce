import React from 'react';
import { Box, Typography, Button } from '@mui/material';
import { HelpCircle } from 'lucide-react';

interface DisambiguationBannerProps {
  question: string;
  options: { label: string; action: () => void }[];
  onDismiss: () => void;
}

export const DisambiguationBanner: React.FC<DisambiguationBannerProps> = ({
  question,
  options,
  onDismiss,
}) => {
  return (
    <Box
      sx={{
        p: 1.2,
        px: 2,
        mb: 1.5,
        borderRadius: 2,
        bgcolor: 'rgba(2, 132, 199, 0.08)',
        border: '1px solid rgba(2, 132, 199, 0.25)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <HelpCircle size={16} color="#0284C7" />
        <Typography variant="caption" sx={{ color: '#0284C7', fontWeight: 700 }}>
          {question}
        </Typography>
      </Box>

      <Box sx={{ display: 'flex', gap: 1 }}>
        {options.map((opt, i) => (
          <Button
            key={i}
            size="small"
            onClick={opt.action}
            sx={{
              textTransform: 'none',
              fontSize: '0.72rem',
              py: 0.3,
              px: 1.2,
              bgcolor: 'rgba(2, 132, 199, 0.15)',
              color: '#0284C7',
              fontWeight: 600,
              '&:hover': { bgcolor: 'rgba(2, 132, 199, 0.25)' },
            }}
          >
            {opt.label}
          </Button>
        ))}
        <Button
          size="small"
          onClick={onDismiss}
          sx={{ color: '#64748B', fontSize: '0.72rem', textTransform: 'none' }}
        >
          Skip
        </Button>
      </Box>
    </Box>
  );
};

export default DisambiguationBanner;
