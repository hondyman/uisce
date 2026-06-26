import React from 'react';
import {
  Box,
  Typography,
  Paper,
  Stack,
  ToggleButton,
  ToggleButtonGroup,
  Alert,
  Checkbox,
  FormControlLabel,
  FormGroup,
} from '@mui/material';
import {
  Block as BlockIcon,
  Visibility as ReadIcon,
  Edit as WriteIcon,
} from '@mui/icons-material';
import { AccessRuleInput, AccessLevel } from '../../../../api/accessRules';

interface AccessLevelStepProps {
  ruleData: AccessRuleInput;
  updateRuleData: (updates: Partial<AccessRuleInput>) => void;
}

export const AccessLevelStep: React.FC<AccessLevelStepProps> = ({ ruleData, updateRuleData }) => {
  const handleAccessLevelChange = (_: any, value: AccessLevel | null) => {
    if (value) {
      updateRuleData({ accessLevel: value });
    }
  };

  const handleScopeChange = (field: 'appliesToApis' | 'appliesToBi' | 'appliesToAi') => (
    event: React.ChangeEvent<HTMLInputElement>
  ) => {
    updateRuleData({
      scope: {
        ...ruleData.scope,
        [field]: event.target.checked,
      },
    });
  };

  return (
    <Box>
      <Typography variant="h5" sx={{ fontWeight: 700, mb: 1 }}>
        What level of access?
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 4 }}>
        Choose the permission level for this group
      </Typography>

      <Stack spacing={3}>
        <Paper elevation={1} sx={{ p: 3, bgcolor: 'grey.50' }}>
          <Typography variant="subtitle2" sx={{ mb: 2, fontWeight: 600 }}>
            Access Level
          </Typography>
          <ToggleButtonGroup
            value={ruleData.accessLevel}
            exclusive
            onChange={handleAccessLevelChange}
            fullWidth
            size="large"
          >
            <ToggleButton value="NONE" sx={{ py: 2 }}>
              <Stack alignItems="center" spacing={1}>
                <BlockIcon />
                <Box>
                  <Typography variant="body2" sx={{ fontWeight: 600 }}>
                    No Access
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    Explicitly deny access
                  </Typography>
                </Box>
              </Stack>
            </ToggleButton>
            <ToggleButton value="READ" sx={{ py: 2 }}>
              <Stack alignItems="center" spacing={1}>
                <ReadIcon />
                <Box>
                  <Typography variant="body2" sx={{ fontWeight: 600 }}>
                    Read Only
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    View data only
                  </Typography>
                </Box>
              </Stack>
            </ToggleButton>
            <ToggleButton value="WRITE" sx={{ py: 2 }}>
              <Stack alignItems="center" spacing={1}>
                <WriteIcon />
                <Box>
                  <Typography variant="body2" sx={{ fontWeight: 600 }}>
                    Read & Write
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    View and modify data
                  </Typography>
                </Box>
              </Stack>
            </ToggleButton>
          </ToggleButtonGroup>
        </Paper>

        <Paper elevation={1} sx={{ p: 3, bgcolor: 'grey.50' }}>
          <Typography variant="subtitle2" sx={{ mb: 2, fontWeight: 600 }}>
            Where does this apply?
          </Typography>
          <FormGroup>
            <FormControlLabel
              control={
                <Checkbox
                  checked={ruleData.scope?.appliesToApis ?? true}
                  onChange={handleScopeChange('appliesToApis')}
                />
              }
              label={
                <Box>
                  <Typography variant="body2">APIs</Typography>
                  <Typography variant="caption" color="text.secondary">
                    REST and GraphQL endpoints
                  </Typography>
                </Box>
              }
            />
            <FormControlLabel
              control={
                <Checkbox
                  checked={ruleData.scope?.appliesToBi ?? true}
                  onChange={handleScopeChange('appliesToBi')}
                />
              }
              label={
                <Box>
                  <Typography variant="body2">Business Intelligence</Typography>
                  <Typography variant="caption" color="text.secondary">
                    Reports and dashboards
                  </Typography>
                </Box>
              }
            />
            <FormControlLabel
              control={
                <Checkbox
                  checked={ruleData.scope?.appliesToAi ?? true}
                  onChange={handleScopeChange('appliesToAi')}
                />
              }
              label={
                <Box>
                  <Typography variant="body2">AI/Analytics</Typography>
                  <Typography variant="caption" color="text.secondary">
                    AI queries and analysis
                  </Typography>
                </Box>
              }
            />
          </FormGroup>
        </Paper>

        {ruleData.accessLevel && (
          <Alert severity="info">
            <Typography variant="body2">
              <strong>{ruleData.accessLevel}</strong> access will be granted to the selected group for the chosen data type.
            </Typography>
          </Alert>
        )}
      </Stack>
    </Box>
  );
};
