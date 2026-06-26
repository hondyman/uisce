import type { FC } from 'react';
import {
  Paper,
  Box,
  Typography,
  Button,
  CircularProgress,
} from '@mui/material';
import {
  Save,
  CheckCircle,
  Error as ErrorIcon,
} from '@mui/icons-material';

interface ViewHeaderProps {
  viewData?: any;
  viewName?: string;
  validationResult?: any;
  isSaving: boolean;
  isValidating: boolean;
  onSave: () => void;
  onValidate: () => void;
}

export const ViewHeader: FC<ViewHeaderProps> = ({
  viewData,
  viewName,
  validationResult,
  isSaving,
  isValidating,
  onSave,
  onValidate,
}) => {
  const validationIcon = validationResult?.valid === false
    ? <ErrorIcon color="error" />
    : validationResult?.valid === true
    ? <CheckCircle color="success" />
    : null;

  return (
    <Paper elevation={1} sx={{ p: 2, borderRadius: 0 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <Typography variant="h5" fontWeight={600}>
            {viewData?.title || viewName}
          </Typography>
          {validationIcon}
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Button
            variant="outlined"
            onClick={onValidate}
            disabled={isValidating}
            startIcon={isValidating ? <CircularProgress size={16} /> : <CheckCircle />}
          >
            Validate
          </Button>
          <Button
            variant="contained"
            onClick={onSave}
            disabled={isSaving}
            startIcon={isSaving ? <CircularProgress size={16} /> : <Save />}
          >
            Save
          </Button>
        </Box>
      </Box>
    </Paper>
  );
};