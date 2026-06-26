import React from 'react';
import {
  Box,
  Typography,
  Paper,
  Stack,
  Chip,
} from '@mui/material';
import { Group as GroupIcon } from '@mui/icons-material';
import { AccessRuleInput } from '../../../../api/accessRules';
import { GroupSelector } from '../GroupSelector';

interface WhoStepProps {
  ruleData: AccessRuleInput;
  updateRuleData: (updates: Partial<AccessRuleInput>) => void;
}

export const WhoStep: React.FC<WhoStepProps> = ({ ruleData, updateRuleData }) => {
  return (
    <Box>
      <Typography variant="h5" sx={{ fontWeight: 700, mb: 1 }}>
        Who should have access?
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 4 }}>
        Select the team or user group that this rule applies to
      </Typography>

      <Paper elevation={1} sx={{ p: 3, bgcolor: 'grey.50' }}>
        <Stack spacing={3}>
          <GroupSelector
            value={ruleData.groupDn}
            onChange={(dn) => updateRuleData({ groupDn: dn || '' })}
            required
            error={!ruleData.groupDn}
            helperText={!ruleData.groupDn ? 'Please select a group' : undefined}
          />

          {ruleData.groupDn && (
            <Paper elevation={0} sx={{ p: 2, bgcolor: 'background.paper', border: '1px solid', borderColor: 'divider' }}>
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
                Selected Group
              </Typography>
              <Stack direction="row" spacing={1} alignItems="center">
                <Chip
                  icon={<GroupIcon />}
                  label={ruleData.groupDn.split(',')[0].replace('cn=', '')}
                  color="primary"
                  variant="outlined"
                />
                <Typography variant="caption" color="text.secondary" noWrap>
                  {ruleData.groupDn}
                </Typography>
              </Stack>
            </Paper>
          )}
        </Stack>
      </Paper>
    </Box>
  );
};
