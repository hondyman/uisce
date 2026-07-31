import React, { useState } from 'react';
import { Box, Typography, Button, Stack } from '@mui/material';
import ThumbUpOutlined from '@mui/icons-material/ThumbUpOutlined';
import ThumbDownOutlined from '@mui/icons-material/ThumbDownOutlined';
import apiClient from '../../utils/apiClient';

const ERROR_CATEGORIES = [
  { id: 'WRONG_TABLE', label: 'Wrong Table / Business Object' },
  { id: 'INCORRECT_FORMULA', label: 'Incorrect Calculation / Formula' },
  { id: 'MISSING_DATA', label: 'Missing or Incorrectly Filtered Data' },
  { id: 'HALLUCINATED_SCHEMA', label: 'Hallucinated Schema or Columns' },
];

interface FeedbackActionBarProps {
  interactionId?: string;
  recommendationLabel: string;
  targetBoKey?: string;
}

export const FeedbackActionBar: React.FC<FeedbackActionBarProps> = ({
  interactionId,
  recommendationLabel,
  targetBoKey = 'general',
}) => {
  const [voted, setVoted] = useState<'up' | 'down' | null>(null);
  const [showTags, setShowTags] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const submit = async (
    ratingType: 'THUMBS_UP' | 'THUMBS_DOWN',
    errorCategory = 'NONE'
  ) => {
    const prev = voted;
    setVoted(ratingType === 'THUMBS_UP' ? 'up' : 'down');

    if (ratingType === 'THUMBS_DOWN' && errorCategory === 'NONE') {
      setShowTags(true);
      return;
    }

    setShowTags(false);
    setSubmitError(null);

    try {
      await apiClient('/api/v1/ai/feedback/explicit', {
        method: 'POST',
        body: JSON.stringify({
          interaction_id: interactionId ?? '',
          target_bo_key: targetBoKey,
          rating_type: ratingType,
          error_category: errorCategory,
          recommendation_label: recommendationLabel,
        }),
      });
    } catch {
      setVoted(prev);
      setSubmitError('Failed to submit feedback');
    }
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, mt: 1 }}>
      {submitError && (
        <Typography variant="caption" sx={{ color: 'error.main', fontSize: '0.7rem' }}>
          {submitError}
        </Typography>
      )}
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <Typography variant="caption" sx={{ color: 'text.disabled', fontSize: '0.65rem' }}>
          Rate accuracy:
        </Typography>
        <Button
          size="small"
          onClick={() => submit('THUMBS_UP')}
          sx={{
            minWidth: 0,
            p: 0.5,
            borderRadius: 1,
            color: voted === 'up' ? 'success.main' : 'text.secondary',
            bgcolor: voted === 'up' ? 'success.dark' : 'transparent',
            '&:hover': { bgcolor: voted === 'up' ? 'success.dark' : 'action.hover' },
          }}
          title="Helpful"
        >
          <ThumbUpOutlined sx={{ fontSize: 14 }} />
        </Button>
        <Button
          size="small"
          onClick={() => submit('THUMBS_DOWN')}
          sx={{
            minWidth: 0,
            p: 0.5,
            borderRadius: 1,
            color: voted === 'down' ? 'error.main' : 'text.secondary',
            bgcolor: voted === 'down' ? 'error.dark' : 'transparent',
            '&:hover': { bgcolor: voted === 'down' ? 'error.dark' : 'action.hover' },
          }}
          title="Not helpful"
        >
          <ThumbDownOutlined sx={{ fontSize: 14 }} />
        </Button>
      </Box>

      {showTags && (
        <Box
          sx={{
            display: 'flex',
            flexDirection: 'column',
            gap: 0.5,
            p: 1,
            bgcolor: 'background.default',
            border: 1,
            borderColor: 'divider',
            borderRadius: 1.5,
          }}
        >
          <Typography variant="caption" sx={{ color: 'text.disabled', fontSize: '0.6rem', mb: 0.5 }}>
            Help us improve — what went wrong?
          </Typography>
          <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 0.5 }}>
            {ERROR_CATEGORIES.map((cat) => (
              <Button
                key={cat.id}
                size="small"
                onClick={() => submit('THUMBS_DOWN', cat.id)}
                sx={{
                  textAlign: 'left',
                  px: 1,
                  py: 0.5,
                  borderRadius: 1,
                  bgcolor: 'background.paper',
                  border: 1,
                  borderColor: 'divider',
                  color: 'text.primary',
                  '&:hover': { bgcolor: 'error.dark', borderColor: 'error.main' },
                  textTransform: 'none',
                  fontSize: '0.65rem',
                }}
              >
                {cat.label}
              </Button>
            ))}
          </Box>
        </Box>
      )}
    </Box>
  );
};
