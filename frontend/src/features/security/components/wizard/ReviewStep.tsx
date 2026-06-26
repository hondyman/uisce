import React from 'react';
import {
  Box,
  Typography,
  Paper,
  Stack,
  Chip,
  Divider,
  Alert,
  Grid,
  Card,
  CardContent,
} from '@mui/material';
import {
  Group as GroupIcon,
  Business as BusinessIcon,
  Security as SecurityIcon,
  FilterList as FilterIcon,
  Visibility as VisibilityIcon,
  CheckCircle as CheckIcon,
} from '@mui/icons-material';
import { AccessRuleInput } from '../../../../api/accessRules';

interface ReviewStepProps {
  ruleData: AccessRuleInput;
}

export const ReviewStep: React.FC<ReviewStepProps> = ({ ruleData }) => {
  return (
    <Box>
      <Typography variant="h5" sx={{ fontWeight: 700, mb: 1 }}>
        Review and confirm
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 4 }}>
        Please review the access rule before saving
      </Typography>

      <Stack spacing={3}>
        <Alert severity="success" icon={<CheckIcon />}>
          Everything looks good! Click "Create Rule" to save this access rule.
        </Alert>

        <Grid container spacing={2}>
          <Grid item xs={12} md={6}>
            <Card elevation={2}>
              <CardContent>
                <Stack spacing={2}>
                  <Stack direction="row" alignItems="center" spacing={1}>
                    <GroupIcon color="primary" />
                    <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                      Who
                    </Typography>
                  </Stack>
                  <Typography variant="body2" color="text.secondary">
                    {ruleData.groupDn || 'Not specified'}
                  </Typography>
                </Stack>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12} md={6}>
            <Card elevation={2}>
              <CardContent>
                <Stack spacing={2}>
                  <Stack direction="row" alignItems="center" spacing={1}>
                    <BusinessIcon color="primary" />
                    <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                      What
                    </Typography>
                  </Stack>
                  <Typography variant="body2" color="text.secondary">
                    {ruleData.businessObjectId || 'Not specified'}
                  </Typography>
                </Stack>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12} md={6}>
            <Card elevation={2}>
              <CardContent>
                <Stack spacing={2}>
                  <Stack direction="row" alignItems="center" spacing={1}>
                    <SecurityIcon color="primary" />
                    <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                      Access Level
                    </Typography>
                  </Stack>
                  <Chip
                    label={ruleData.accessLevel}
                    color={ruleData.accessLevel === 'WRITE' ? 'primary' : 'default'}
                    size="small"
                  />
                </Stack>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12} md={6}>
            <Card elevation={2}>
              <CardContent>
                <Stack spacing={2}>
                  <Stack direction="row" alignItems="center" spacing={1}>
                    <FilterIcon color="primary" />
                    <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                      Row Filters
                    </Typography>
                  </Stack>
                  <Typography variant="body2" color="text.secondary">
                    {ruleData.rowFilterDsl || 'None (all rows visible)'}
                  </Typography>
                </Stack>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12}>
            <Card elevation={2}>
              <CardContent>
                <Stack spacing={2}>
                  <Stack direction="row" alignItems="center" spacing={1}>
                    <VisibilityIcon color="primary" />
                    <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                      Field Masks
                    </Typography>
                  </Stack>
                  {ruleData.columnMasks && ruleData.columnMasks.length > 0 ? (
                    <Stack spacing={1}>
                      {ruleData.columnMasks.map((mask, index) => (
                        <Stack key={index} direction="row" spacing={1} alignItems="center">
                          <Chip label={mask.semanticTermId} size="small" variant="outlined" />
                          <Typography variant="caption" color="text.secondary">
                            →
                          </Typography>
                          <Chip label={mask.maskType} size="small" color="primary" />
                        </Stack>
                      ))}
                    </Stack>
                  ) : (
                    <Typography variant="body2" color="text.secondary">
                      None (all fields visible)
                    </Typography>
                  )}
                </Stack>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12}>
            <Card elevation={2}>
              <CardContent>
                <Stack spacing={2}>
                  <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                    Applies To
                  </Typography>
                  <Stack direction="row" spacing={1}>
                    {ruleData.scope?.appliesToApis && <Chip label="APIs" size="small" color="primary" />}
                    {ruleData.scope?.appliesToBi && <Chip label="BI" size="small" color="primary" />}
                    {ruleData.scope?.appliesToAi && <Chip label="AI" size="small" color="primary" />}
                  </Stack>
                </Stack>
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      </Stack>
    </Box>
  );
};
