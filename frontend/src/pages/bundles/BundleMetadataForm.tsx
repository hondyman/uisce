/**
 * Bundle Metadata Form Component
 *
 * Handles the basic bundle information (name, description)
 */

import type React from 'react';
import {
  TextField,
  Grid,
  Box,
  Typography,
} from '@mui/material';

interface BundleMetadataFormProps {
  name: string;
  description: string;
  onNameChange: (name: string) => void;
  onDescriptionChange: (description: string) => void;
  fieldErrors: Record<string, string[]>;
  fieldHelperText: (field: string) => string | undefined;
}

export const BundleMetadataForm: React.FC<BundleMetadataFormProps> = ({
  name,
  description,
  onNameChange,
  onDescriptionChange,
  fieldErrors,
  fieldHelperText,
}) => {
  return (
    <Box sx={{ mb: 4 }}>
      <Typography variant="h6" gutterBottom>
        Bundle Information
      </Typography>
      <Grid container spacing={3}>
        <Grid item xs={12} md={6}>
          <TextField
            fullWidth
            label="Bundle Name"
            value={name}
            onChange={(e) => onNameChange(e.target.value)}
            error={(fieldErrors.name?.length ?? 0) > 0}
            helperText={fieldHelperText('name') || ''}
            required
          />
        </Grid>
        <Grid item xs={12} md={6}>
          <TextField
            fullWidth
            label="Description"
            value={description}
            onChange={(e) => onDescriptionChange(e.target.value)}
            error={(fieldErrors.description?.length ?? 0) > 0}
            helperText={fieldHelperText('description') || ''}
            multiline
            rows={2}
          />
        </Grid>
      </Grid>
    </Box>
  );
};