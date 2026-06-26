import React from 'react';
import {
  Box,
  Typography,
  Paper,
  Stack,
  TextField,
  Select,
  MenuItem,
  IconButton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Alert,
  Button,
  FormControl,
  InputLabel,
} from '@mui/material';
import {
  Add as AddIcon,
  Delete as DeleteIcon,
  Visibility as VisibilityIcon,
  VisibilityOff as VisibilityOffIcon,
} from '@mui/icons-material';
import { AccessRuleInput, MaskType } from '../../../../api/accessRules';

interface ColumnMaskStepProps {
  ruleData: AccessRuleInput;
  updateRuleData: (updates: Partial<AccessRuleInput>) => void;
}

const maskTypes: MaskType[] = ['NONE', 'MASK', 'HIDE'];

export const ColumnMaskStep: React.FC<ColumnMaskStepProps> = ({ ruleData, updateRuleData }) => {
  const masks = ruleData.columnMasks || [];

  const addMask = () => {
    updateRuleData({
      columnMasks: [...masks, { semanticTermId: '', maskType: 'MASK' }],
    });
  };

  const removeMask = (index: number) => {
    updateRuleData({
      columnMasks: masks.filter((_, i) => i !== index),
    });
  };

  const updateMask = (index: number, field: 'semanticTermId' | 'maskType', value: string) => {
    const updated = [...masks];
    updated[index] = { ...updated[index], [field]: value };
    updateRuleData({ columnMasks: updated });
  };

  return (
    <Box>
      <Typography variant="h5" sx={{ fontWeight: 700, mb: 1 }}>
        Hide or mask fields (Optional)
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 4 }}>
        Control which fields are visible or masked for this group
      </Typography>

      <Stack spacing={3}>
        <Alert severity="info">
          Field masking lets you hide sensitive data or show it in a masked format (e.g., SSN: ***-**-1234)
        </Alert>

        <Paper elevation={1} sx={{ p: 3, bgcolor: 'grey.50' }}>
          <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
            <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
              Field Masks
            </Typography>
            <Button startIcon={<AddIcon />} onClick={addMask} size="small">
              Add Field
            </Button>
          </Stack>

          <TableContainer>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell sx={{ fontWeight: 600 }}>Field Name</TableCell>
                  <TableCell sx={{ fontWeight: 600 }}>Mask Type</TableCell>
                  <TableCell align="right" sx={{ fontWeight: 600 }}>
                    Actions
                  </TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {masks.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={3} align="center" sx={{ py: 3 }}>
                      <Typography variant="body2" color="text.secondary">
                        No field masks defined. Click "Add Field" to get started.
                      </Typography>
                    </TableCell>
                  </TableRow>
                )}
                {masks.map((mask, index) => (
                  <TableRow key={index}>
                    <TableCell>
                      <TextField
                        fullWidth
                        size="small"
                        placeholder="e.g., SSN, Salary, Email"
                        value={mask.semanticTermId}
                        onChange={(e) => updateMask(index, 'semanticTermId', e.target.value)}
                      />
                    </TableCell>
                    <TableCell>
                      <FormControl fullWidth size="small">
                        <Select
                          value={mask.maskType}
                          onChange={(e) => updateMask(index, 'maskType', e.target.value as MaskType)}
                        >
                          <MenuItem value="NONE">
                            <Stack direction="row" spacing={1} alignItems="center">
                              <VisibilityIcon fontSize="small" />
                              <span>Show (No Mask)</span>
                            </Stack>
                          </MenuItem>
                          <MenuItem value="MASK">
                            <Stack direction="row" spacing={1} alignItems="center">
                              <VisibilityIcon fontSize="small" />
                              <span>Partially Mask</span>
                            </Stack>
                          </MenuItem>
                          <MenuItem value="HIDE">
                            <Stack direction="row" spacing={1} alignItems="center">
                              <VisibilityOffIcon fontSize="small" />
                              <span>Hide Completely</span>
                            </Stack>
                          </MenuItem>
                        </Select>
                      </FormControl>
                    </TableCell>
                    <TableCell align="right">
                      <IconButton size="small" color="error" onClick={() => removeMask(index)}>
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Paper>

        {masks.length === 0 && (
          <Alert severity="warning">
            No field masks set. The group will see <strong>all fields</strong> (subject to row filters).
          </Alert>
        )}
      </Stack>
    </Box>
  );
};
