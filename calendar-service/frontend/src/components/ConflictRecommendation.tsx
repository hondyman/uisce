import React, { useState, useEffect } from 'react';
import {
  Card,
  CardContent,
  Typography,
  Box,
  Button,
  Chip,
  Alert,
  AlertTitle,
  LinearProgress,
  Stack,
  Divider,
  Tooltip,
} from '@mui/material';
import {
  AutoFixHigh as AIFeatureIcon,
  Psychology as AIIcon,
  CheckCircle as CheckIcon,
  Cancel as CancelIcon,
  Help as HelpIcon,
} from '@mui/icons-material';

interface ConflictRecommendationProps {
  conflictId: string;
  onResolve: (strategy: string) => Promise<void>;
  loading?: boolean;
}

interface MLRecommendation {
  strategy: string;
  confidence: number;
  reasoning: string;
  alternative_strategies: string[];
  model_version: string;
}

const ConflictRecommendation: React.FC<ConflictRecommendationProps> = ({
  conflictId,
  onResolve,
  loading = false,
}) => {
  const [recommendation, setRecommendation] = useState<MLRecommendation | null>(null);
  const [loadingML, setLoadingML] = useState(true);
  const [selectedStrategy, setSelectedStrategy] = useState<string | null>(null);
  const [resolving, setResolving] = useState(false);

  useEffect(() => {
    loadRecommendation();
  }, [conflictId]);

  const loadRecommendation = async () => {
    try {
      setLoadingML(true);
      const response = await fetch(`/api/v1/ml/conflicts/${conflictId}/recommend`, {
        headers: {
          'X-Hasura-Tenant-Id': localStorage.getItem('tenant_id') || '',
        },
      });
      
      if (response.ok) {
        const data = await response.json();
        setRecommendation(data.recommendation);
        setSelectedStrategy(data.recommendation.strategy);
      }
    } catch (err) {
      console.error('Failed to load ML recommendation:', err);
    } finally {
      setLoadingML(false);
    }
  };

  const handleResolve = async () => {
    if (!selectedStrategy) return;
    
    setResolving(true);
    try {
      await onResolve(selectedStrategy);
    } catch (err) {
      console.error('Failed to resolve conflict:', err);
    } finally {
      setResolving(false);
    }
  };

  const getConfidenceColor = (confidence: number) => {
    if (confidence >= 0.8) return 'success';
    if (confidence >= 0.6) return 'warning';
    return 'error';
  };

  const getStrategyLabel = (strategy: string) => {
    const labels: Record<string, string> = {
      'keep_google': 'Keep Google Version',
      'keep_internal': 'Keep Internal Version',
      'merge': 'Merge Both',
      'skip': 'Skip (No Action)',
    };
    return labels[strategy] || strategy;
  };

  if (loadingML) {
    return (
      <Card variant="outlined">
        <CardContent>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <AIIcon color="primary" />
            <Typography variant="subtitle1">
              AI is analyzing this conflict...
            </Typography>
          </Box>
          <LinearProgress sx={{ mt: 2 }} />
        </CardContent>
      </Card>
    );
  }

  if (!recommendation) {
    return (
      <Card variant="outlined">
        <CardContent>
          <Alert severity="info">
            <AlertTitle>AI Recommendation Unavailable</AlertTitle>
            Please resolve this conflict manually.
          </Alert>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card variant="outlined" sx={{ borderColor: 'primary.light' }}>
      <CardContent>
        {/* AI Recommendation Header */}
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 2 }}>
          <Box
            sx={{
              p: 1,
              borderRadius: 1,
              backgroundColor: 'primary.light',
              color: 'primary.contrastText',
            }}
          >
            <AIIcon />
          </Box>
          <Box>
            <Typography variant="subtitle1" fontWeight="bold">
              AI Recommendation
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Model version: {recommendation.model_version}
            </Typography>
          </Box>
        </Box>

        {/* Recommended Strategy */}
        <Alert 
          severity={getConfidenceColor(recommendation.confidence)} 
          sx={{ mb: 2 }}
          icon={<AIFeatureIcon />}
        >
          <AlertTitle>Recommended: {getStrategyLabel(recommendation.strategy)}</AlertTitle>
          {recommendation.reasoning}
        </Alert>

        {/* Confidence Score */}
        <Box sx={{ mb: 2 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
            <Typography variant="body2" color="text.secondary">
              Confidence Score
            </Typography>
            <Typography variant="body2" fontWeight="bold">
              {(recommendation.confidence * 100).toFixed(0)}%
            </Typography>
          </Box>
          <LinearProgress
            variant="determinate"
            value={recommendation.confidence * 100}
            color={getConfidenceColor(recommendation.confidence)}
          />
        </Box>

        {/* Alternative Strategies */}
        {recommendation.alternative_strategies && recommendation.alternative_strategies.length > 0 && (
          <Box sx={{ mb: 2 }}>
            <Typography variant="subtitle2" gutterBottom>
              Alternative Strategies:
            </Typography>
            <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
              {recommendation.alternative_strategies.map((strategy) => (
                <Chip
                  key={strategy}
                  label={getStrategyLabel(strategy)}
                  size="small"
                  variant="outlined"
                  clickable
                  onClick={() => setSelectedStrategy(strategy)}
                  color={selectedStrategy === strategy ? 'primary' : 'default'}
                />
              ))}
            </Stack>
          </Box>
        )}

        <Divider sx={{ my: 2 }} />

        {/* Action Buttons */}
        <Stack direction="row" spacing={2} justifyContent="flex-end">
          <Tooltip title="AI will resolve this conflict automatically">
            <Button
              variant="contained"
              color="primary"
              onClick={handleResolve}
              disabled={resolving || !selectedStrategy}
              startIcon={resolving ? <AIFeatureIcon /> : <CheckIcon />}
            >
              {resolving ? 'Resolving...' : 'Accept Recommendation'}
            </Button>
          </Tooltip>
          
          <Button
            variant="outlined"
            color="inherit"
            onClick={() => setSelectedStrategy('skip')}
            disabled={resolving}
            startIcon={<CancelIcon />}
          >
            Skip for Now
          </Button>
        </Stack>

        {/* Auto-Resolve Info */}
        {recommendation.confidence >= 0.8 && (
          <Alert severity="success" sx={{ mt: 2 }} icon={<AIFeatureIcon />}>
            <Typography variant="caption">
              ✨ High confidence! This conflict can be auto-resolved. 
              Enable auto-resolution in settings to handle similar conflicts automatically.
            </Typography>
          </Alert>
        )}
      </CardContent>
    </Card>
  );
};

export default ConflictRecommendation;
