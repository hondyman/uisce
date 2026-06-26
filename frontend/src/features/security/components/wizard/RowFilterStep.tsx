import React from 'react';
import {
  Box,
  Typography,
  Stack,
  Alert,
  CircularProgress,
  Chip,
} from '@mui/material';
import { CheckCircle as CheckIcon, Error as ErrorIcon, Info as InfoIcon } from '@mui/icons-material';
import { AccessRuleInput } from '../../../../api/accessRules';
import { RowFilterBuilder } from '../RowFilterBuilder';
import { useRuleValidation } from '../../hooks/useRuleValidation';

interface RowFilterStepProps {
  ruleData: AccessRuleInput;
  updateRuleData: (updates: Partial<AccessRuleInput>) => void;
}

export const RowFilterStep: React.FC<RowFilterStepProps> = ({ ruleData, updateRuleData }) => {
  const { validation, validating } = useRuleValidation(
    ruleData.rowFilterDsl || '',
    ruleData.businessObjectId
  );

  return (
    <Box>
      <Typography variant="h5" sx={{ fontWeight: 700, mb: 1 }}>
        Filter data rows (Optional)
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 4 }}>
        Restrict which data rows this group can see. Leave empty to allow all rows.
      </Typography>

      <Stack spacing={3}>
        <Alert severity="info" icon={<InfoIcon />}>
          Row filters let you limit data based on conditions. Use the visual builder below to create your filter.
        </Alert>

        <RowFilterBuilder
          value={ruleData.rowFilterDsl || ''}
          onChange={(dsl) => updateRuleData({ rowFilterDsl: dsl })}
        />

        {/* Real-time Validation Feedback */}
        {validating && (
          <Stack direction="row" spacing={1} alignItems="center">
            <CircularProgress size={16} />
            <Typography variant="caption" color="text.secondary">
              Validating filter...
            </Typography>
          </Stack>
        )}

        {validation && !validating && (
          <>
            {validation.valid ? (
              <Alert severity="success" icon={<CheckIcon />}>
                <Stack spacing={1}>
                  <Typography variant="body2">
                    Filter is valid and will be applied correctly
                  </Typography>
                  {validation.sql && (
                    <Box>
                      <Typography variant="caption" color="text.secondary">
                        Generated SQL:
                      </Typography>
                      <Typography
                        variant="caption"
                        sx={{
                          display: 'block',
                          fontFamily: 'monospace',
                          bgcolor: 'success.light',
                          p: 1,
                          borderRadius: 1,
                          mt: 0.5,
                        }}
                      >
                        {validation.sql}
                      </Typography>
                    </Box>
                  )}
                </Stack>
              </Alert>
            ) : (
              <Alert severity="error" icon={<ErrorIcon />}>
                <Typography variant="body2">
                  {validation.error || 'Invalid filter expression'}
                </Typography>
              </Alert>
            )}
          </>
        )}

        {!ruleData.rowFilterDsl && !validating && (
          <Alert severity="warning">
            No filter set. The group will have access to <strong>all rows</strong> of this data type (subject to column masks).
          </Alert>
        )}
      </Stack>
    </Box>
  );
};
