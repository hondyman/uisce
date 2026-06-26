import React from 'react';
import {
  Paper,
  Box,
  Typography,
  Stack,
  Chip,
  Divider,
  CircularProgress,
  Alert,
  Card,
  CardContent,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
} from '@mui/material';
import {
  Lightbulb as LightbulbIcon,
  TrendingUp as TrendingUpIcon,
  Warning as WarningIcon,
  CheckCircle as CheckCircleIcon,
} from '@mui/icons-material';

interface Explanation {
  rootCause?: string;
  timeline?: string;
  affectedSystems?: string[];
  recommendations?: string[];
  riskAssessment?: {
    level: string;
    description: string;
  };
  relatedEvents?: Array<{
    id: string;
    description: string;
  }>;
}

interface AIPanelProps {
  selectedEvent: any | null;
  explanation: Explanation | null;
  loading: boolean;
  userRole: string;
}

/**
 * AIPanel: Side panel showing AI-generated explanations
 * 
 * Displays:
 * - Root cause analysis
 * - Timeline and sequence of events
 * - Affected systems
 * - Recommendations
 * - Risk assessment
 * - Related events correlation
 */
export function AIPanel({
  selectedEvent,
  explanation,
  loading,
  userRole,
}: AIPanelProps) {
  const showAdvanced = ['global_admin', 'global_ops', 'tenant_admin'].includes(userRole);

  if (!selectedEvent) {
    return (
      <Paper sx={{ p: 3, height: '100%' }}>
        <Stack spacing={2} alignItems="center" justifyContent="center" sx={{ height: '100%' }}>
          <LightbulbIcon sx={{ color: 'text.secondary', fontSize: 48 }} />
          <Typography variant="body2" sx={{ color: 'text.secondary', textAlign: 'center' }}>
            Select an event to view AI-generated insights
          </Typography>
        </Stack>
      </Paper>
    );
  }

  return (
    <Paper sx={{ p: 2, height: 'auto', overflow: 'auto', maxHeight: '80vh' }}>
      <Stack spacing={2}>
        {/* Header */}
        <Box>
          <Typography variant="h6" sx={{ mb: 1 }}>
            AI Insights
          </Typography>
          {selectedEvent?.type && (
            <Chip label={selectedEvent.type} size="small" variant="outlined" />
          )}
        </Box>

        <Divider />

        {/* Loading State */}
        {loading && (
          <Box sx={{ display: 'flex', justifyContent: 'center', p: 3 }}>
            <CircularProgress size={40} />
          </Box>
        )}

        {/* Error State */}
        {!loading && !explanation && selectedEvent && (
          <Alert severity="warning">
            Unable to generate AI explanation. Please try again.
          </Alert>
        )}

        {/* Explanation Content */}
        {!loading && explanation && (
          <Stack spacing={2}>
            {/* Root Cause */}
            {explanation.rootCause && (
              <Card variant="outlined">
                <CardContent sx={{ pb: 2, '&:last-child': { pb: 2 } }}>
                  <Stack direction="row" spacing={1} alignItems="flex-start" sx={{ mb: 1 }}>
                    <LightbulbIcon sx={{ color: 'info.main', mt: 0.5 }} />
                    <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                      Root Cause
                    </Typography>
                  </Stack>
                  <Typography variant="body2" sx={{ ml: 4 }}>
                    {explanation.rootCause}
                  </Typography>
                </CardContent>
              </Card>
            )}

            {/* Risk Assessment */}
            {explanation.riskAssessment && (
              <Card variant="outlined">
                <CardContent sx={{ pb: 2, '&:last-child': { pb: 2 } }}>
                  <Stack direction="row" spacing={1} alignItems="flex-start" sx={{ mb: 1 }}>
                    <WarningIcon
                      sx={{
                        color:
                          explanation.riskAssessment.level === 'high'
                            ? 'error.main'
                            : explanation.riskAssessment.level === 'medium'
                            ? 'warning.main'
                            : 'success.main',
                        mt: 0.5,
                      }}
                    />
                    <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                      Risk Assessment
                    </Typography>
                  </Stack>
                  <Typography variant="body2" sx={{ ml: 4 }}>
                    <strong>{explanation.riskAssessment.level.toUpperCase()}</strong>
                    {' '}
                    {explanation.riskAssessment.description}
                  </Typography>
                </CardContent>
              </Card>
            )}

            {/* Timeline */}
            {explanation.timeline && (
              <Card variant="outlined">
                <CardContent sx={{ pb: 2, '&:last-child': { pb: 2 } }}>
                  <Stack direction="row" spacing={1} alignItems="flex-start" sx={{ mb: 1 }}>
                    <TrendingUpIcon sx={{ color: 'action.active', mt: 0.5 }} />
                    <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                      Timeline
                    </Typography>
                  </Stack>
                  <Typography variant="body2" sx={{ ml: 4, whiteSpace: 'pre-wrap' }}>
                    {explanation.timeline}
                  </Typography>
                </CardContent>
              </Card>
            )}

            {/* Affected Systems */}
            {explanation.affectedSystems && explanation.affectedSystems.length > 0 && (
              <Box>
                <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>
                  Affected Systems
                </Typography>
                <Stack direction="row" spacing={1} flexWrap="wrap">
                  {explanation.affectedSystems.map((system, idx) => (
                    <Chip key={idx} label={system} size="small" />
                  ))}
                </Stack>
              </Box>
            )}

            {/* Recommendations */}
            {explanation.recommendations && explanation.recommendations.length > 0 && (
              <Box>
                <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>
                  Recommendations
                </Typography>
                <List dense>
                  {explanation.recommendations.map((rec, idx) => (
                    <ListItem key={idx} disableGutters>
                      <ListItemIcon sx={{ minWidth: 32 }}>
                        <CheckCircleIcon
                          fontSize="small"
                          sx={{ color: 'success.main' }}
                        />
                      </ListItemIcon>
                      <ListItemText
                        primary={rec}
                        primaryTypographyProps={{ variant: 'body2' }}
                      />
                    </ListItem>
                  ))}
                </List>
              </Box>
            )}

            {/* Related Events (Advanced Users) */}
            {showAdvanced &&
              explanation.relatedEvents &&
              explanation.relatedEvents.length > 0 && (
                <Box>
                  <Divider sx={{ my: 2 }} />
                  <Typography variant="caption" sx={{ fontWeight: 600, mb: 1 }}>
                    Related Events
                  </Typography>
                  <List dense>
                    {explanation.relatedEvents.map((event, idx) => (
                      <ListItem key={idx} disableGutters>
                        <ListItemText
                          primary={event.description}
                          primaryTypographyProps={{ variant: 'caption' }}
                        />
                      </ListItem>
                    ))}
                  </List>
                </Box>
              )}
          </Stack>
        )}

        {/* Empty State with Advanced Info */}
        {!loading &&
          !explanation &&
          !selectedEvent && (
            <Box sx={{ p: 2, textAlign: 'center' }}>
              {showAdvanced && (
                <Alert severity="info" sx={{ mt: 2 }}>
                  Advanced: The AI explanation panel uses semantic and compliance context to
                  provide cross-tenant insights while respecting your tenant scope.
                </Alert>
              )}
            </Box>
          )}
      </Stack>
    </Paper>
  );
}

export default AIPanel;
