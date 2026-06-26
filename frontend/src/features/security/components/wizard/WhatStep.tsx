import React from 'react';
import {
  Box,
  Typography,
  Paper,
  Stack,
  Chip,
} from '@mui/material';
import { Business as BusinessIcon } from '@mui/icons-material';
import { AccessRuleInput } from '../../../../api/accessRules';
import { BusinessObjectSelector } from '../BusinessObjectSelector';

interface WhatStepProps {
  ruleData: AccessRuleInput;
  updateRuleData: (updates: Partial<AccessRuleInput>) => void;
}

export const WhatStep: React.FC<WhatStepProps> = ({ ruleData, updateRuleData }) => {
  return (
    <Box>
      <Typography variant="h5" sx={{ fontWeight: 700, mb: 1 }}>
        What data should they access?
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 4 }}>
        Select the type of data this rule controls
      </Typography>

      <Paper elevation={1} sx={{ p: 3, bgcolor: 'grey.50' }}>
        <Stack spacing={3}>
          <BusinessObjectSelector
            value={ruleData.businessObjectId}
            onChange={(id) => updateRuleData({ businessObjectId: id || '' })}
            required
            error={!ruleData.businessObjectId}
            helperText={!ruleData.businessObjectId ? 'Please select a data type' : undefined}
          />

          {ruleData.businessObjectId && (
            <Paper elevation={0} sx={{ p: 2, bgcolor: 'background.paper', border: '1px solid', borderColor: 'divider' }}>
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
                Selected Data Type
              </Typography>
              <Stack direction="row" spacing={1} alignItems="center">
                <Chip
                  icon={<BusinessIcon />}
                  label={ruleData.businessObjectId}
                  color="primary"
                  variant="outlined"
                />
              </Stack>
            </Paper>
          )}
        </Stack>
      </Paper>
    </Box>
  );
};
